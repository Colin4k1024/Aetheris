# Checkpoint 与状态恢复：Worker 崩溃后发生了什么

> 理解 Aetheris 如何让 Agent 「死而复生」。

## 0. 场景：Worker 崩溃了

```
时间线：
14:00  Agent 开始处理退款申请 #12345
14:01  ├─ 读取申请详情 ✓
14:02  ├─ 调用风控 API ✓
14:03  ├─ 执行退款 ✓
       │         [就在这时候！]
14:03  └─ 发送确认邮件 [崩溃！]
       
14:05  服务器重启，Worker 恢复
14:06  Agent 重新开始处理...

用户视角：
- 收到了退款
- 没收到确认邮件
- 又收到了一封确认邮件（因为重新执行了！）
```

这显然是错的。**正确的行为应该是**：Agent 恢复后从「发送邮件」这一步继续，而不是从头再来。

这就是 **Checkpoint（检查点）** 要解决的问题。

## 1. Checkpoint 的本质

### 1.1 什么是 Checkpoint？

**Checkpoint = 状态快照**

在某个时刻，把 Agent 的完整状态保存下来：
- 执行到哪一步了？
- 当前的变量/内存是什么？
- 之前的工具调用结果是什么？

崩溃恢复时：
1. 加载最近的 Checkpoint
2. 恢复状态
3. 从中断点继续执行

### 1.2 Checkpoint 存储什么？

```go
type Checkpoint struct {
    // 任务标识
    JobID      string `json:"job_id"`
    
    // 执行位置
    StepID     string `json:"step_id"`       // 当前执行到的步骤
    Cursor     int64  `json:"cursor"`        // 事件序列号
    
    // 执行状态（核心！）
    State      map[string]interface{} `json:"state"`
    
    // 节点状态
    NodeStates map[string]NodeState `json:"node_states"`
    
    // 工具结果缓存（用于 At-Most-Once）
    ToolResults map[string]ToolResult `json:"tool_results"`
    
    // 元数据
    CreatedAt  int64 `json:"created_at"`
}
```

### 1.3 何时写入 Checkpoint？

Aetheris 的策略是：**每步执行完成后立即写入**

```
Step 1 ──▶ Step 2 ──▶ Step 3 ──▶ Step 4
              │           │           │
              │           │       [Checkpoint]
              │           │
              │       [Checkpoint]
              │
          [Checkpoint]
```

这意味着：
- 最多丢失一个步骤的执行
- 恢复时最多重做一步

## 2. 完整的恢复流程

### 2.1 整体流程图

```
┌─────────────────────────────────────────────────────────────┐
│                      Worker A 执行任务                       │
│  Job #123: Step 1 → Step 2 → Step 3                        │
│                                                             │
│  T1: 执行 Step 3（发送邮件）                               │
│  T2: 邮件 API 调用成功                                      │
│  T3: 准备更新数据库...                                       │
│  T4: [崩溃！]                                               │
└─────────────────────────────┬───────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────┐
│                    Scheduler 检测到故障                      │
│                                                             │
│  - Worker A 失联（Heartbeat 超时）                          │
│  - Lease 过期                                                │
│  - Job #123 重新入队                                        │
└─────────────────────────────┬───────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────┐
│                    Worker B 接收任务                         │
│                                                             │
│  Job #123 状态：Step 3 执行中                               │
└─────────────────────────────┬───────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────┐
│                     Runner 加载 Checkpoint                   │
│                                                             │
│  1. 读取 Job 的 Checkpoint                                  │
│  2. 恢复 State（变量、内存）                                │
│  3. 恢复 Tool Results（Step 1-2 的工具调用结果）           │
│  4. 确定恢复点：Step 3                                      │
└─────────────────────────────┬───────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────┐
│                    从中断点继续执行                          │
│                                                             │
│  Step 3: 发送邮件                                           │
│  ├─ 检查 Tool Ledger：idempotency_key 已存在                │
│  │  → 直接返回缓存结果，不重复调用                         │
│  └─ 继续执行后续步骤                                        │
│                                                             │
│  ✅ 用户只收到一封邮件！                                     │
└─────────────────────────────────────────────────────────────┘
```

### 2.2 恢复时的关键判断

```go
func (r *Runner) resumeFromCheckpoint(ctx context.Context, job *Job) error {
    checkpoint := job.Checkpoint
    
    // 1. 恢复执行状态
    r.state = checkpoint.State
    
    // 2. 恢复工具结果缓存
    for key, result := range checkpoint.ToolResults {
        toolLedger.Put(key, result)
    }
    
    // 3. 确定从哪个步骤继续
    currentStepID := checkpoint.StepID
    
    // 4. 查找下一个要执行的步骤
    nextStep := r.findNextStep(currentStepID)
    
    // 5. 执行
    return r.executeStep(nextStep)
}
```

## 3. Checkpoint 的存储

### 3.1 存储位置

```sql
-- Job 表中的 JSONB 字段
CREATE TABLE jobs (
    id UUID PRIMARY KEY,
    agent_id UUID NOT NULL,
    status VARCHAR(20) NOT NULL,
    checkpoint JSONB,  -- Checkpoint 存储在这里
    created_at TIMESTAMP,
    updated_at TIMESTAMP
);
```

### 3.2 Checkpoint 写入时机

```go
func (r *Runner) executeStep(step *Step) error {
    // 执行步骤
    result, err := r.runNode(step.Node)
    if err != nil {
        return err
    }
    
    // 更新状态
    r.state[step.ID] = result
    
    // 写入事件
    jobstore.AppendEvent(&Event{
        Type:    "StepCompleted",
        StepID:  step.ID,
        Result:  result,
    })
    
    // [关键！] 写入 Checkpoint
    checkpoint := &Checkpoint{
        JobID:       r.job.ID,
        StepID:      step.ID,
        Cursor:      r.eventCursor,
        State:       r.state,
        ToolResults: r.toolResults,
        CreatedAt:   time.Now().Unix(),
    }
    jobStore.SaveCheckpoint(r.job.ID, checkpoint)
    
    return nil
}
```

## 4. 边缘情况处理

### 4.1 Checkpoint 写入一半崩溃

```
场景：
1. 执行 Step 3 成功
2. 开始写入 Checkpoint
3. [崩溃]
   - Checkpoint 可能部分写入
   - 状态不一致
```

**解决方案**：
- 使用数据库事务：先写事件，再写 Checkpoint
- 或者：写 Checkpoint 到临时位置，完成后原子替换

```go
func (r *Runner) saveCheckpointAtomic(checkpoint *Checkpoint) error {
    // 1. 写入临时位置
    tempKey := fmt.Sprintf("checkpoint/%s.tmp", r.job.ID)
    redis.Set(tempKey, checkpoint)
    
    // 2. 写入正式位置（原子操作）
    finalKey := fmt.Sprintf("checkpoint/%s", r.job.ID)
    redis.Rename(tempKey, finalKey)
    
    return nil
}
```

### 4.2 没有 Checkpoint（新任务）

```
场景：任务刚开始就崩溃了
- Job Created 事件已写入
- Checkpoint 不存在
```

**解决方案**：从头开始执行

```go
func (r *Runner) runJob(job *Job) error {
    checkpoint := jobStore.GetCheckpoint(job.ID)
    
    if checkpoint == nil {
        // 新任务：从头开始
        return r.planAndExecute(job.Goal)
    } else {
        // 有 Checkpoint：从恢复点继续
        return r.resumeFromCheckpoint(job)
    }
}
```

### 4.3 Checkpoint 太旧

```
场景：任务执行了 100 步，中途崩溃
- Checkpoint 在 Step 50
- 需要从 Step 51 恢复
- 但事件流有 100 条记录
```

**解决方案**：从 Checkpoint 恢复后，跳过已执行的步骤

```go
func (r *Runner) resumeFromCheckpoint(checkpoint *Checkpoint) error {
    // 从 Checkpoint 恢复状态
    r.state = checkpoint.State
    
    // 获取 Checkpoint 之后的事件
    events := jobstore.GetEventsAfter(checkpoint.Cursor)
    
    // 重放事件以重建执行上下文
    for _, event := range events {
        r.replayEvent(event)
    }
    
    // 从中断点继续
    return r.executeFromStep(checkpoint.StepID)
}
```

## 5. Session 与 Checkpoint

### 5.1 什么是 Session？

**Session = 一次完整的 Agent 执行会话**

```go
type Session struct {
    SessionID   string                 // 会话 ID
    JobID       string                 // 关联的 Job
    AgentID     string                 // 使用的 Agent
    State       map[string]interface{} // 会话状态
    Checkpoints []Checkpoint           // 历史检查点
}
```

### 5.2 Session 的作用

- **多轮对话**：用户可以多次发送消息，共享同一个 Session
- **长期记忆**：Session 可以跨多次执行持久化
- **调试入口**：可以通过 Session ID 查看完整执行历史

## 6. 与其他系统的对比

| 系统 | Checkpoint 策略 | 恢复粒度 |
|------|-----------------|----------|
| Kubernetes | Pod 级别 | 整个 Pod |
| Temporal | Activity 级别 | 整个 Activity |
| **Aetheris** | **Step 级别** | **单个 Step** |

Aetheris 的优势：**恢复粒度更细，丢失的工作更少**。

## 7. 性能考量

### 7.1 Checkpoint 开销

每次执行完一步都要写 Checkpoint，有一定开销：

- 写入 PostgreSQL（同步）
- 序列化 JSON
- 网络 round-trip

### 7.2 优化策略

1. **异步写入**：Checkpoint 写入后台异步执行
2. **增量 Checkpoint**：只记录变更的部分
3. **批量写入**：多个 Job 的 Checkpoint 批量提交

```go
func (r *Runner) saveCheckpointAsync(checkpoint *Checkpoint) {
    go func() {
        // 异步写入，不阻塞执行
        r.checkpointChan <- checkpoint
    }()
}

func (r *CheckpointWriter) run() {
    for {
        batch := <-r.checkpointChan
        
        // 批量写入
        for _, cp := range batch.Collect(100) {
            db.SaveCheckpoint(cp)
        }
    }
}
```

## 8. 小结

Checkpoint 是 Aetheris **持久化执行**的关键：

1. **每步执行后写入** — 最多丢失一步
2. **保存完整状态** — State、NodeStates、ToolResults
3. **恢复时跳过已执行步骤** — 结合 Tool Ledger 保证不重复
4. **支持异步写入** — 平衡可靠性与性能

有了 Checkpoint，Worker 崩溃不再是噩梦——**Agent 可以优雅地「死而复生」**。

---

*下篇预告：Human-in-the-Loop 实战——审批流与长时间等待*
