# 性能优化：吞吐量与延迟调优

> 让 Aetheris 在生产环境中跑得更快、更稳。

## 0. 性能问题的根源

Agent Runtime 的性能挑战与传统后端不同：

```
传统 API：
请求 → 处理 → 响应
（毫秒级）

Agent Runtime：
请求 → 规划 → N × (LLM调用/工具调用) → 响应
（秒级甚至分钟级）
```

**瓶颈不在 Aetheris 本身，而在：**
1. LLM 调用延迟
2. 工具调用延迟
3. 数据库写入

## 1. 性能指标

### 1.1 核心指标

| 指标 | 定义 | 目标 |
|------|------|------|
| **吞吐量 (Throughput)** | 每秒处理的 Job 数 | > 100 jobs/s |
| **延迟 (Latency)** | Job 从创建到完成的时间 | P99 < 30s |
| **可用性 (Availability)** | 成功完成的 Job / 总 Job | > 99.9% |
| **恢复时间 (Recovery Time)** | 故障后恢复的时间 | < 30s |

### 1.2 瓶颈分布

```
Job 执行时间分解：

┌─────────────────────────────────────────────────────────────┐
│  示例：一个包含 5 个步骤的 Job，总耗时 60 秒                  │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│  LLM 调用 (3次)    ████████████████████████████  45s  75%   │
│  工具调用 (2次)    ████████████                10s  17%   │
│  DB 写入           ████                       3s   5%    │
│  其他开销           ██                         2s   3%    │
│                                                             │
└─────────────────────────────────────────────────────────────┘
```

## 2. LLM 调用优化

### 2.1 并行调用

**多个 LLM 调用可以并行执行：**

```go
// ❌ 串行：一步步执行
result1 := llm.Call(prompt1)  // 5s
result2 := llm.Call(prompt2)  // 5s
result3 := llm.Call(prompt3)  // 5s
// 总计：15s

// ✅ 并行：同时执行
results, _ := parallel.Invoke(
    func() return llm.Call(prompt1),  // 5s
    func() return llm.Call(prompt2),  // 5s
    func() return llm.Call(prompt3),  // 5s
)
// 总计：5s
```

### 2.2 缓存

**相同的 LLM 请求可以直接返回缓存：**

```go
// LLM 请求缓存
type LLMCache struct {
    redis *redis.Client
}

func (c *LLMCache) Get(prompt, model string) (string, bool) {
    key := hash(prompt + model)
    result, err := c.redis.Get(ctx, "llm:"+key).Result()
    return result, err == nil
}

func (c *LLMCache) Set(prompt, model, response string) {
    key := hash(prompt + model)
    c.redis.SetEX(ctx, "llm:"+key, response, 24*time.Hour)
}

// 使用缓存
func (r *Runner) callLLM(req *LLMRequest) (*LLMResponse, error) {
    // 1. 检查缓存
    if cached, ok := r.llmCache.Get(req.Prompt, req.Model); ok {
        return cached, nil
    }
    
    // 2. 调用 LLM
    response, err := r.llmClient.Call(req)
    if err != nil {
        return nil, err
    }
    
    // 3. 写入缓存
    r.llmCache.Set(req.Prompt, req.Model, response)
    
    return response, nil
}
```

### 2.3 模型选择

```
┌─────────────────────────────────────────────────────────────┐
│                     模型选择策略                              │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│  简单任务（如分类）→ 小模型（快、便宜）                       │
│  ┌─────────────────────────────────────┐                   │
│  │ GPT-3.5 Turbo                       │                   │
│  │ - 延迟: < 3s                        │                   │
│  │ - 成本: $0.5/1M tokens              │                   │
│  └─────────────────────────────────────┘                   │
│                                                             │
│  复杂任务（如推理）→ 大模型（慢、贵）                        │
│  ┌─────────────────────────────────────┐                   │
│  │ GPT-4                               │                   │
│  │ - 延迟: < 30s                       │                   │
│  │ - 成本: $30/1M tokens               │                   │
│  └─────────────────────────────────────┘                   │
│                                                             │
│  路由示例：                                                 │
│  if task.complexity < 0.3 → gpt-3.5                       │
│  else if task.complexity < 0.7 → gpt-4                   │
│  else → gpt-4 + CoT                                        │
│                                                             │
└─────────────────────────────────────────────────────────────┘
```

### 2.4 Streaming

**使用流式响应减少感知延迟：**

```go
// 启用 LLM Streaming
stream, err := llmClient.CreateCompletionStream(ctx, &CompletionRequest{
    Model:       "gpt-4",
    Prompt:      prompt,
    Stream:      true,
})

// 边生成边返回
for {
    chunk, err := stream.Recv()
    if err == io.EOF {
        break
    }
    
    // 实时发送给孩子客户端
    sendToClient(chunk)
}
```

## 3. 数据库优化

### 3.1 写入优化

```go
// ❌ 同步写入：每条事件都同步写入
func (j *JobStore) AppendEvent(event *Event) error {
    return j.db.Insert(event)  // 每次都 fsync
}

// ✅ 批量写入：累积后批量提交
func (j *JobStore) AppendEventAsync(event *Event) {
    j.eventChan <- event  // 写入 channel
}

func (j *JobStore) flushLoop() {
    batch := collectEvents(100, 5*time.Second)  // 收集批量或定时
    
    // 事务批量写入
    j.db.Transaction(func(tx *DB) error {
        for _, event := range batch {
            tx.Insert(event)
        }
        return nil
    })
}
```

### 3.2 读写分离

```go
// 读操作走只读副本
func (j *JobStore) GetJob(id string) (*Job, error) {
    return j.readDB.Query("SELECT * FROM jobs WHERE id = ?", id)
}

// 写操作走主库
func (j *JobStore) CreateJob(job *Job) error {
    return j.writeDB.Insert(job)
}

// 热点数据走 Redis
func (j *JobStore) GetJobStatus(id string) (string, error) {
    // 先查 Redis
    status, err := j.redis.Get("job:status:" + id).Result()
    if err == nil {
        return status, nil
    }
    
    // 查数据库
    job, err := j.readDB.GetJob(id)
    if err != nil {
        return "", err
    }
    
    // 回填 Redis
    j.redis.Set("job:status:"+id, job.Status, 30*time.Second)
    
    return job.Status, nil
}
```

### 3.3 索引优化

```sql
-- 常用查询的索引
CREATE INDEX idx_events_job_id ON events(job_id);
CREATE INDEX idx_events_sequence ON events(job_id, sequence_id);
CREATE INDEX idx_jobs_status ON jobs(status);
CREATE INDEX idx_jobs_agent_id ON jobs(agent_id);
CREATE INDEX idx_jobs_created_at ON jobs(created_at);

-- 复合索引
CREATE INDEX idx_jobs_status_created ON jobs(status, created_at);
```

## 4. Checkpoint 优化

### 4.1 增量 Checkpoint

```go
// ❌ 全量 Checkpoint：每次保存全部状态
func (r *Runner) saveFullCheckpoint() {
    checkpoint := &Checkpoint{
        JobID:  r.job.ID,
        State:  r.state,  // 全部状态
    }
    r.jobStore.Save(checkpoint)
}

// ✅ 增量 Checkpoint：只保存变更
func (r *Runner) saveIncrementalCheckpoint(prev *Checkpoint) {
    changes := diff(prev.State, r.state)
    
    checkpoint := &Checkpoint{
        JobID:      r.job.ID,
        StateDelta: changes,  // 只有变更
    }
    r.jobStore.Save(checkpoint)
}
```

### 4.2 异步 Checkpoint

```go
// 异步写入 Checkpoint
func (r *Runner) saveCheckpointAsync() {
    select {
    case r.checkpointChan <- r.currentCheckpoint:
        // 成功放入队列
    default:
        // 队列满，跳过一次 Checkpoint
        // （最多丢失一个步骤）
    }
}
```

## 5. 连接池与资源管理

### 5.1 HTTP 连接池

```go
// LLM 客户端连接池
httpClient := &http.Client{
    Transport: &http.Transport{
        MaxIdleConns:        100,
        MaxIdleConnsPerHost: 100,
        IdleConnTimeout:     90 * time.Second,
    },
    Timeout: 60 * time.Second,
}

// 数据库连接池
db.SetMaxOpenConns(25)
db.SetMaxIdleConns(10)
db.SetConnMaxLifetime(5 * time.Minute)
```

### 5.2 Worker 并发控制

```go
// Worker 并发限制
type Worker struct {
    maxConcurrent int
    semaphore    chan struct{}
}

func NewWorker(maxConcurrent int) *Worker {
    return &Worker{
        maxConcurrent: maxConcurrent,
        semaphore:    make(chan struct{}, maxConcurrent),
    }
}

func (w *Worker) ExecuteJob(job *Job) error {
    w.semaphore <- struct{}{}  // 获取令牌
    defer func() { <-w.semaphore }()
    
    // 执行 Job
    return w.runJob(job)
}
```

## 6. 监控与调优

### 6.1 关键指标

```yaml
# Prometheus 指标
- aetheris_jobs_completed_total
- aetheris_jobs_failed_total
- aetheris_job_duration_seconds
- aetheris_step_duration_seconds
- aetheris_llm_call_duration_seconds
- aetheris_tool_call_duration_seconds
- aetheris_db_write_duration_seconds
- aetheris_active_jobs
- aetheris_worker_load
```

### 6.2 性能面板

```
┌─────────────────────────────────────────────────────────────┐
│                   Aetheris Performance                       │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│  Throughput           Latency (P99)        Errors           │
│  ─────────           ───────────────      ──────           │
│  ████████████ 150/s   ████████ 12s        █ 0.1%          │
│                                                             │
│  Job Duration Breakdown   Worker Utilization               │
│  ────────────────────   ─────────────────                  │
│  LLM:  ████████████     ██████████████ 80%                │
│  Tool: ██████             ░░░░░░░░░░░░░ 20% idle          │
│  DB:   ██                                                │
│                                                             │
│  [ Jobs/Sec ]    [ P50 Latency ]    [ P99 Latency ]      │
│       150              8s                  25s              │
│                                                             │
└─────────────────────────────────────────────────────────────┘
```

### 6.3 调优参数

```yaml
# 配置文件
runtime:
  # Worker 配置
  worker:
    max_concurrent_jobs: 10      # 单 Worker 最大并发
    heartbeat_interval: 10s     # 心跳间隔
    lease_duration: 30s         # Lease 过期时间
    
  # LLM 配置
  llm:
    timeout: 60s                # LLM 调用超时
    max_retries: 3             # 最大重试
    retry_delay: 1s            # 重试延迟
    cache_enabled: true        # 启用缓存
    
  # 数据库配置
  database:
    batch_size: 100            # 批量写入大小
    flush_interval: 1s         # 刷新间隔
    connection_pool: 25       # 连接池大小
    
  # Checkpoint 配置
  checkpoint:
    enabled: true
    async: true                # 异步写入
    interval: 1s               # 写入间隔
```

## 7. 常见问题与解决方案

### 7.1 LLM 延迟高

| 问题 | 解决方案 |
|------|----------|
| 模型太慢 | 切换到更快的模型 |
| 网络延迟 | 使用本地模型或专线 |
| 并发太高 | 限流或排队 |
| Token 太多 | 优化 Prompt 或使用摘要 |

### 7.2 数据库成为瓶颈

| 问题 | 解决方案 |
|------|----------|
| 写入太慢 | 批量写入、异步写入 |
| 查询太慢 | 添加索引、使用缓存 |
| 连接不够 | 增大连接池 |

### 7.3 Worker 负载不均

| 问题 | 解决方案 |
|------|----------|
| 某些 Job 太重 | 拆分为小 Job |
| 调度不均 | 使用公平调度策略 |
| 热点数据 | 分散到不同 Worker |

## 8. 小结

性能优化是一个持续的过程：

1. **LLM 优化** — 并行调用、缓存、模型选择、流式响应
2. **数据库优化** — 批量写入、读写分离、索引优化
3. **Checkpoint 优化** — 增量写入、异步写入
4. **资源管理** — 连接池、并发控制
5. **监控调优** — 持续监控、动态调整

**记住**：Agent Runtime 的性能瓶颈 80% 在 LLM，20% 在基础设施。先优化 LLM 调用，再优化数据库。

---

*下篇预告：从原型到生产——部署踩坑与最佳实践*
