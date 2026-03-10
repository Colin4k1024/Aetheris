# 多 Worker 部署与调度正确性

> 当一个 Agent 崩溃时，另一个 Worker 如何接管？防止两个 Worker 同时执行同一任务？

## 0. 分布式系统的经典问题

```
场景：两个 Worker 同时处理同一个 Job

时间线：

Worker A                          Worker B
─────────────────────────────────────────────────────
1. 获取 Job #123
2. 开始执行 Step 1
3. 调用 Stripe API 退款 $100
4. [网络抖动]
   ↓
   [失去心跳]
5.                            5. Scheduler 认为 A 死了
                                 获取 Job #123
 6. 开始6.                           执行 Step 1
7.                            7. 调用 Stripe API 退款 $100

结果：用户被扣款 $200！
```

这就是经典的**脑裂问题**。Aetheris 必须解决这个问题。

## 1. Scheduler 架构

### 1.1 核心职责

Scheduler 是 Aetheris 的大脑：

```
┌─────────────────────────────────────────────────────────────┐
│                      Scheduler                               │
├─────────────────────────────────────────────────────────────┤
│  1. 任务分配   → 把 Job 分发给 Worker                        │
│  2. 负载均衡   → 合理分配任务                                │
│  3. 故障检测   → 检测 Worker 崩溃                            │
│  4. 故障恢复   → 把失败的 Job 重新入队                       │
│  5. 限流       → 防止 Worker 过载                           │
└─────────────────────────────────────────────────────────────┘
```

### 1.2 组件结构

```
┌─────────────────────────────────────────────────────────────┐
│                    Scheduler 组件                             │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│  ┌─────────────┐    ┌─────────────┐    ┌─────────────┐    │
│  │  Job Queue  │───▶│  Assigner   │───▶│   Lease     │    │
│  │  (待执行任务) │    │  (分配器)   │    │  (租约管理) │    │
│  └─────────────┘    └─────────────┘    └─────────────┘    │
│         │                                    │              │
│         │                ┌───────────────────┘              │
│         ▼                ▼                                  │
│  ┌─────────────┐    ┌─────────────┐                        │
│  │  Selector   │    │  Fencing    │                        │
│  │  (选择策略)  │    │  (防脑裂)   │                        │
│  └─────────────┘    └─────────────┘                        │
│                                                             │
└─────────────────────────────────────────────────────────────┘
```

## 2. Lease 机制：任务的「锁」

### 2.1 什么是 Lease？

**Lease = 临时的任务所有权**

```go
type Lease struct {
    JobID          string    // 任务 ID
    WorkerID       string    // Worker ID
    FencingToken   int64     // 栅栏令牌（防脑裂关键）
    IssuedAt       int64     // 发放时间
    ExpiresAt      int64     // 过期时间
    HeartbeatAt    int64     // 最后心跳时间
}
```

### 2.2 Lease 工作流程

```
时间线：

T1: Worker A 请求 Job #123
    │
    │ Scheduler 检查队列
    │
T2: Scheduler 发放 Lease
    │ Lease { job_id: 123, worker_id: "A", token: 1, expires: +30s }
    │
T3: Worker A 开始执行
    │ - 定期发送 Heartbeat（每 10s）
    │ - Lease 续期
    │
T4: [Worker A 崩溃]
    │
T5: 15s 没有收到 Heartbeat
    │
T6: Lease 过期
    │ Scheduler 标记 Lease 失效
    │ Job #123 重新入队
    │
T7: Worker B 请求 Job #123
    │
T8: Scheduler 发放新 Lease
    │ Lease { job_id: 123, worker_id: "B", token: 2, expires: +30s }
    │
T9: Worker B 开始执行
```

### 2.3 Heartbeat 机制

```go
// Worker 定期发送心跳
func (w *Worker) heartbeat() {
    for {
        time.Sleep(10 * time.Second)
        
        err := scheduler.RenewLease(w.lease)
        if err != nil {
            // Lease 已过期，停止执行
            w.stop()
            return
        }
    }
}

// Scheduler 续约
func (s *Scheduler) RenewLease(lease *Lease) error {
    // 1. 验证 Lease 仍然有效
    stored, err := s.leaseStore.Get(lease.JobID)
    if err != nil || stored.FencingToken != lease.FencingToken {
        return ErrLeaseInvalid  // 已被其他人接管
    }
    
    // 2. 续期
    lease.ExpiresAt = time.Now().Add(30 * time.Second)
    lease.HeartbeatAt = time.Now()
    s.leaseStore.Put(lease)
    
    return nil
}
```

## 3. Fencing Token：防脑裂的核心

### 3.1 什么是 Fencing Token？

**Fencing Token = 递增的许可证**

每次发放新 Lease 时，Token 递增：

```
Lease #1: job=123, token=1, worker=A
Lease #2: job=123, token=2, worker=B
Lease #3: job=123, token=3, worker=C
```

### 3.2 关键验证逻辑

```go
// Runner 执行任何操作前，必须验证 Token
func (r *Runner) executeStep(step *Step) error {
    // 1. 获取当前的 Lease
    lease, err := r.scheduler.GetLease(r.job.ID)
    if err != nil {
        return ErrNoLease
    }
    
    // 2. 验证 Fencing Token
encingToken != r    if lease.F.currentToken {
        // Token 不匹配！说明有其他人正在执行
        return ErrFencedOff
    }
    
    // 3. 验证 Lease 未过期
    if time.Now().After(lease.ExpiresAt) {
        return ErrLeaseExpired
    }
    
    // 4. 执行步骤
    return r.runStep(step)
}
```

### 3.3 脑裂场景处理

```
场景：Worker A 崩溃后复活

时间线：

T1: Worker A (token=1) 执行 Step 1
T2: Worker A 崩溃
T3: Worker B (token=2) 接管，执行 Step 2
T4: Worker A 复活，还以为自己在执行
T5: Worker A 尝试执行 Step 3
    │
    │ 验证 Token
    │ - 当前 Lease: token=2
    │ - Worker A 持有: token=1
    │ - 不匹配！拒绝执行
    │
    ✓ 成功防止脑裂！
```

## 4. 多 Worker 部署模式

### 4.1 基础部署

```yaml
# docker-compose.yaml
version: '3.8'

services:
  postgres:
    image: postgres:15
    environment:
      POSTGRES_PASSWORD: aetheris
    
  redis:
    image: redis:7
    
  api:
    image: aetheris/api
    ports:
      - "8080:8080"
    depends_on:
      - postgres
      - redis
      
  worker1:
    image: aetheris/worker
    environment:
      WORKER_ID: worker1
      LEASE_DURATION: 30s
    depends_on:
      - postgres
      - redis
      
  worker2:
    image: aetheris/worker
    environment:
      WORKER_ID: worker2
      LEASE_DURATION: 30s
    depends_on:
      - postgres
      - redis
```

### 4.2 Kubernetes 部署

```yaml
# deployment.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: aetheris-worker
spec:
  replicas: 3
  selector:
    matchLabels:
      app: aetheris-worker
  template:
    metadata:
      labels:
        app: aetheris-worker
    spec:
      containers:
      - name: worker
        image: aetheris/worker:latest
        env:
        - name: WORKER_ID
          valueFrom:
            fieldRef:
              fieldPath: metadata.name
        - name: LEASE_DURATION
          value: "30s"
        resources:
          limits:
            memory: "512Mi"
            cpu: "500m"
```

### 4.3 部署拓扑

```
┌─────────────────────────────────────────────────────────────┐
│                     部署拓扑                                  │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│    ┌──────────────┐      ┌──────────────┐                  │
│    │   API Pod    │      │   API Pod    │                  │
│    │   (副本 2)   │      │   (副本 2)   │                  │
│    └──────┬───────┘      └──────┬───────┘                  │
│           │                      │                          │
│           └──────────┬───────────┘                          │
│                      │                                       │
│    ┌─────────────────┼─────────────────┐                    │
│    │                 │                 │                    │
│    ▼                 ▼                 ▼                    │
│ ┌──────┐       ┌──────┐       ┌──────┐                    │
│ │ W1   │       │ W2   │       │ W3   │  ← Worker Pods      │
│ │      │       │      │       │      │    (可伸缩)          │
│ └──────┘       └──────┘       └──────┘                    │
│    │                 │                 │                    │
│    └─────────────────┼─────────────────┘                    │
│                      ▼                                       │
│           ┌─────────────────────┐                            │
│           │  PostgreSQL + Redis │                            │
│           │    (高可用部署)      │                            │
│           └─────────────────────┘                            │
│                                                             │
└─────────────────────────────────────────────────────────────┘
```

## 5. 调度策略

### 5.1 任务选择策略

```go
// 调度策略接口
type ScheduleStrategy interface {
    // 选择下一个要执行的 Job
    SelectJob(pendingJobs []Job, workers []Worker) *Job
    
    // 是否应该把 Job 分发给这个 Worker
    ShouldAssign(job *Job, worker *Worker) bool
}

// 1. 简单 FIFO
type FIFOStrategy struct{}

func (s *FIFOStrategy) SelectJob(pendingJobs []Job, workers []Worker) *Job {
    // 选择最早创建的 Job
    sort.Slice(pendingJobs, func(i, j int) bool {
        return pendingJobs[i].CreatedAt.Before(pendingJobs[j].CreatedAt)
    })
    return &pendingJobs[0]
}

// 2. 优先级
type PriorityStrategy struct{}

func (s *PriorityStrategy) SelectJob(pendingJobs []Job, workers []Worker) *Job {
    // 选择优先级最高的 Job
    sort.Slice(pendingJobs, func(i, j int) bool {
        return pendingJobs[i].Priority > pendingJobs[j].Priority
    })
    return &pendingJobs[0]
}

// 3. 公平调度（避免某个 Agent 饿死）
type FairShareStrategy struct{}
```

### 5.2 Worker 选择策略

```go
// Worker 选择：选择负载最低的
func (s *Scheduler) selectWorker(job *Job) *Worker {
    workers := s.getAvailableWorkers()
    
    // 过滤掉负载最高的
    minLoad := workers[0].CurrentLoad()
    bestWorker := workers[0]
    
    for _, w := range workers {
        if w.CurrentLoad() < minLoad {
            minLoad = w.CurrentLoad()
            bestWorker = w
        }
    }
    
    return bestWorker
}
```

## 6. 故障处理

### 6.1 Worker 故障检测

```go
// 检测 Worker 故障
func (s *Scheduler) monitorWorkers() {
    for {
        time.Sleep(5 * time.Second)
        
        for _, worker := range s.workers {
            sinceLastHeartbeat := time.Since(worker.LastHeartbeat())
            
            if sinceLastHeartbeat > worker.LeaseDuration() {
                // Worker 失联
                s.handleWorkerFailure(worker)
            }
        }
    }
}

// 处理 Worker 故障
func (s *Scheduler) handleWorkerFailure(worker *Worker) {
    // 1. 使其持有的所有 Lease 过期
    for _, lease := range s.getLeasesByWorker(worker.ID) {
        s.expireLease(lease)
    }
    
    // 2. 把失败的 Job 重新入队
    for _, jobID := range s.getJobsByWorker(worker.ID) {
        job := s.jobStore.Get(jobID)
        job.Status = "pending"
        s.jobQueue.Enqueue(job)
    }
    
    // 3. 记录日志
    log.Printf("Worker %s failed, %d jobs requeued", worker.ID, len(jobs))
}
```

### 6.2 Job 故障恢复

```go
// Job 恢复流程
func (s *Scheduler) recoverJob(jobID string) error {
    job := s.jobStore.Get(jobID)
    
    // 1. 检查 Job 当前状态
    switch job.Status {
    case "running":
        // Job 正在执行时 Worker 崩溃
        // 检查是否有 Checkpoint
        if job.Checkpoint != nil {
            // 从 Checkpoint 恢复
            job.Status = "pending"
            s.jobQueue.Enqueue(job)
            log.Printf("Job %s recovered from checkpoint at step %s", 
                jobID, job.Checkpoint.StepID)
        } else {
            // 没有 Checkpoint，只能从头开始
            job.Status = "pending"
            job.Checkpoint = nil
            s.jobQueue.Enqueue(job)
            log.Printf("Job %s recovered from start (no checkpoint)")
        }
        
    case "parked":
        // Job 在等待 Signal，状态已持久化
        // 只需要重新入队
        job.Status = "running"
        s.jobQueue.Enqueue(job)
        
    case "failed":
        // Job 彻底失败
        // 可以选择重试或标记为失败
        if job.RetryCount < 3 {
            job.RetryCount++
            job.Status = "pending"
            s.jobQueue.Enqueue(job)
        }
    }
    
    return nil
}
```

## 7. 正确性保证

### 7.1 调度正确性定理

Aetheris 保证：

> **在任何时刻，每个 Job 最多被一个 Worker 执行。**

### 7.2 证明要点

1. **Lease 唯一性**：同一时间只有一个有效的 Lease
2. **Fencing 验证**：每次执行前验证 Token
3. **过期检测**：定期检测失效的 Lease
4. **原子操作**：Lease 发放和过期都是原子的

### 7.3 测试：Four Fatal Tests

Aetheris 有 4 个核心测试用例，验证调度正确性：

| 测试 | 场景 | 验证 |
|------|------|------|
| TestCrashBeforeTool | Worker 崩溃在工具调用前 | 工具不被执行 |
| TestCrashAfterTool | Worker 崩溃在工具调用后、提交前 | 工具不重复执行 |
| TestTwoWorkersSameStep | 两个 Worker 同时执行同一步骤 | 第二个被拒绝 |
| TestReplayRestore | Replay 时恢复输出 | 外部状态一致性 |

## 8. 运维建议

### 8.1 Worker 数量

```
公式：Worker 数量 = 峰值并发任务数 / 单 Worker 容量

示例：
- 峰值：100 个并发任务
- 单 Worker 容量：10 个任务
- 需要的 Worker：10-15 个（留 50% 余量）
```

### 8.2 Lease 时长

```
原则：
- 任务执行时间短 → Lease 时长短（如 30s）
- 任务执行时间长 → Lease 时长长（如 5min）
- 人工等待的任务 → Lease 时长应大于 Wait Timeout

配置：
LEASE_DURATION=30s
HEARTBEAT_INTERVAL=10s  # Lease 时长的 1/3
```

### 8.3 监控指标

```bash
# 关键指标
- scheduler.lease_renewals_total      # Lease 续约次数
- scheduler.lease_expirations_total   # Lease 过期次数
- scheduler.jobs_requeued_total       # Job 重新入队次数
- worker.heartbeat_misses_total      # 心跳丢失次数
- worker.tasks_completed_total       # 完成任务数
```

## 9. 小结

多 Worker 部署的核心是**调度正确性**：

1. **Lease 机制** — 临时的任务所有权
2. **Fencing Token** — 防止脑裂
3. **Heartbeat** — 检测 Worker 故障
4. **故障恢复** — Job 自动重新入队
5. **调度策略** — FIFO、优先级、公平调度

有了这套机制，**Aetheris 可以安全地扩展到多个 Worker**，支撑高并发任务处理。

---

*下篇预告：性能优化——吞吐量与延迟调优*
