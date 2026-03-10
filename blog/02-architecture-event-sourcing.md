# Aetheris 核心架构解析：事件溯源与可恢复执行

> 理解 Aetheris 如何用「事件流」实现「记忆」与「恢复」。

## 0. 架构概览

在深入细节之前，先看看 Aetheris 的整体架构：

```
┌─────────────────────────────────────────────────────────────────┐
│                        Clients (CLI / HTTP / SDK)               │
└─────────────────────────────┬───────────────────────────────────┘
                              │
┌─────────────────────────────▼───────────────────────────────────┐
│                      API Layer (Hertz)                          │
│  ├─ Agent API      (Agent 管理)                                  │
│  ├─ Job API        (任务管理)                                    │
│  ├─ Runs API       (执行历史)                                    │
│  └─ Trace API      (链路追踪)                                    │
└─────────────────────────────┬───────────────────────────────────┘
                              │
┌─────────────────────────────▼───────────────────────────────────┐
│                     Agent Runtime Core                          │
│  ┌──────────────┐    ┌──────────────┐    ┌──────────────┐     │
│  │ Job Manager  │───▶│  Scheduler   │───▶│   Runner     │     │
│  │  (任务创建)   │    │ (Lease 管理)  │    │ (执行+检查点) │     │
│  └──────────────┘    └──────────────┘    └──────────────┘     │
│         │                                       │              │
│         └───────────────────┬───────────────────┘              │
│                             ▼                                   │
│  ┌──────────────────────────────────────────────────────────┐  │
│  │              Execution Engine (eino)                     │  │
│  │  ┌─────────┐  ┌─────────┐  ┌─────────┐  ┌─────────────┐ │  │
│  │  │ LLM Node│  │Tool Node│  │Wait Node│  │Node Adapter │ │  │
│  │  │ (模型调用)│  │(工具执行)│  │(暂停等待)│  │(LangGraph等)│ │  │
│  │  └─────────┘  └─────────┘  └─────────┘  └─────────────┘ │  │
│  └──────────────────────────────────────────────────────────┘  │
└─────────────────────────────┬───────────────────────────────────┘
                              │
┌─────────────────────────────▼───────────────────────────────────┐
│                    Event & State Layer                         │
│  ┌────────────┐  ┌────────────┐  ┌────────────┐  ┌──────────┐  │
│  │ JobStore   │  │  Tool      │  │  Effect    │  │Checkpoint│  │
│  │ (事件源)    │  │  Ledger    │  │  Store     │  │  Store   │  │
│  │            │  │ (幂等性)    │  │ (副作用记录)│  │ (恢复点)  │  │
│  └────────────┘  └────────────┘  └────────────┘  └──────────┘  │
└─────────────────────────────┬───────────────────────────────────┘
                              │
┌─────────────────────────────▼───────────────────────────────────┐
│                      Storage Layer                             │
│              PostgreSQL (主存储)  │  Redis (缓存)              │
└─────────────────────────────────────────────────────────────────┘
```

## 1. 核心设计原则：事件溯源

### 1.1 什么是事件溯源？

**事件溯源（Event Sourcing）** 是一种架构模式：

> 不存储对象的当前状态，而是存储导致状态变化的所有事件。

**传统方式**（状态快照）：
```
当前状态：订单已支付，已发货

数据库记录：Order { status: "shipped" }
```

**事件溯源方式**：
```
事件流：
1. OrderCreated { order_id: 123, items: [...] }
2. PaymentReceived { order_id: 123, amount: 100 }
3. ShipmentInitiated { order_id: 123, tracking: "SF123" }

当前状态：通过重放事件流得到
```

### 1.2 为什么 Aetheris 使用事件溯源？

对于 Agent 来说，**事件溯源天然适合**：

1. **完整历史** — 每一步执行都被记录，可追溯、可审计
2. **可恢复** — 崩溃后重放事件流即可恢复状态
3. **可回放** — 支持"Replay"调试，重新运行历史执行
4. **可证明** — 事件链本身就是执行证明

## 2. JobStore：事件流存储

### 2.1 事件类型

Aetheris 定义了丰富的事件类型：

```go
// 任务生命周期事件
JobCreated      // 任务创建
JobScheduled    // 任务调度
JobParked       // 任务暂停（等待）
JobResumed      // 任务恢复
JobCompleted    // 任务完成
JobFailed       // 任务失败

// 执行步骤事件
StepStarted     // 步骤开始
StepCompleted   // 步骤完成
StepFailed      // 步骤失败
StepRetrying    // 步骤重试

// LLM 调用事件
LLMRequest      // LLM 请求
LLMResponse     // LLM 响应

// 工具调用事件
ToolInvocated   // 工具调用
ToolCompleted   // 工具完成
ToolFailed      // 工具失败

// 人工介入事件
SignalReceived  // 收到信号
MessageReceived // 收到消息
```

### 2.2 事件结构

```go
type Event struct {
    EventID    string    `json:"event_id"`     // 唯一 ID
    JobID      string    `json:"job_id"`       // 所属任务
    EventType  string    `json:"event_type"`   // 事件类型
    Timestamp  int64     `json:"timestamp"`    // 时间戳
    SequenceID int64     `json:"sequence_id"`  // 序列号（递增）
    
    // 事件内容（不同类型不同结构）
    Payload    json.RawMessage `json:"payload"`
    
    // 证据链
    Evidence   *Evidence `json:"evidence,omitempty"`
}
```

### 2.3 事件写入示例

一个典型的 Agent 执行过程会产生以下事件流：

```
[Event #1] JobCreated
  { goal: "处理退款申请 #12345", agent_type: "refund" }

[Event #2] StepStarted
  { step_id: "step_1", node_type: "llm", name: "analyze_request" }

[Event #3] LLMRequest
  { prompt: "分析退款原因...", model: "gpt-4" }

[Event #4] LLMResponse
  { response: "批准退款，原因合理", model: "gpt-4" }

[Event #5] StepCompleted
  { step_id: "step_1", output: { decision: "approve" } }

[Event #6] StepStarted
  { step_id: "step_2", node_type: "tool", name: "call_refund_api" }

[Event #7] ToolInvocated
  { tool: "stripe.refund", input: { amount: 100, ... } }

[Event #8] ToolCompleted
  { tool: "stripe.refund", output: { refund_id: "re_123" } }

[Event #9] StepCompleted
  { step_id: "step_2", output: { status: "success" } }

[Event #10] JobCompleted
  { final_state: { refund_status: "completed" } }
```

## 3. Runner：执行与检查点

### 3.1 Checkpoint 机制

事件流记录了**发生了什么**，但重放整个事件流效率很低。

**Checkpoint（检查点）** 是定期保存的「状态快照」：

```
时间线：
──────────────────────────────────────────────────────────────▶
│ Event 1 │ Event 2 │ Event 3 │ Event 4 │ Event 5 │ Event 6 │
│         │         │         │ Checkp. │         │ Checkp. │
                                              ▲           ▲
                                            保存状态     保存状态
```

崩溃恢复时：
1. 找到最近的 Checkpoint
2. 恢复状态
3. 从 Checkpoint 之后的事件继续重放

### 3.2 Checkpoint 内容

```go
type Checkpoint struct {
    JobID         string                 `json:"job_id"`
    StepID        string                 `json:"step_id"`        // 当前执行到的步骤
    Cursor        int                    `json:"cursor"`          // 事件序列号
    State         map[string]interface{} `json:"state"`          // 执行状态
    NodeStates    map[string]NodeState    `json:"node_states"`     // 各节点状态
    LastToolResult map[string]interface{} `json:"last_tool_result"` // 工具结果（用于幂等）
}
```

### 3.3 执行恢复流程

```
┌─────────────────────────────────────────────────────────────┐
│ Worker A 执行 Job #123                                      │
│                                                             │
│ Step 1 ──▶ Step 2 ──▶ Step 3 ──▶ [崩溃！]                  │
│                                              ↓              │
│                                      写入 Checkpoint #3     │
│                                      (记录：执行到了 Step 3) │
└─────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────┐
│ Worker B 接收 Job #123（Scheduler 重新分配）                 │
│                                                             │
│ 1. 读取 Checkpoint #3                                      │
│ 2. 恢复 State、NodeStates、ToolResults                     │
│ 3. 从 Step 4 继续执行                                       │
│    （Step 1-3 不再执行，结果从 Checkpoint 注入）             │
└─────────────────────────────────────────────────────────────┘
```

## 4. 多 Worker 协调：Scheduler

### 4.1 Lease 机制

多个 Worker 同时监听任务队列，需要避免**重复执行**：

```
Scheduler 分配策略：
1. Job 进入 Pending 队列
2. Scheduler 选一个 Worker，授予 Lease（租约）
3. Worker 在 Lease 有效期内执行
4. Lease 过期前 Worker 需要续租（Heartbeat）
5. 如果 Worker 崩溃，Lease 自动过期，Job 重新入队
```

### 4.2 Fencing Token

为了防止「脑裂」（两个 Worker 同时认为自己在执行同一 Job），Aetheris 使用 **Fencing Token**：

```go
type Lease struct {
    JobID        string
    WorkerID     string
    FencingToken int64    // 每次授权递增
    ExpiresAt    int64
}
```

执行任何操作前，Runner 必须验证 Fencing Token：

```
Worker A (Token=1)  ──▶ 执行 Step 1
        │
        │ [崩溃]
        ▼
Worker B (Token=2)  ──▶ 尝试执行 Step 2
        │             │
        │             │ 检查 Token：当前是 2，我的也是 2
        │             │ ✅ 允许执行
        ▼
Worker A (复活)     ──▶ 尝试执行 Step 2
                      │ 检查 Token：当前是 2，我是 1
                      │ ❌ 拒绝执行（Token 不匹配）
```

## 5. 存储层设计

### 5.1 PostgreSQL：主存储

```sql
-- 任务表
CREATE TABLE jobs (
    id UUID PRIMARY KEY,
    agent_id UUID NOT NULL,
    status VARCHAR(20) NOT NULL,
    created_at TIMESTAMP,
    updated_at TIMESTAMP,
    checkpoint JSONB
);

-- 事件表（核心）
CREATE TABLE events (
    id BIGSERIAL PRIMARY KEY,
    job_id UUID NOT NULL,
    sequence_id BIGINT NOT NULL,
    event_type VARCHAR(30) NOT NULL,
    payload JSONB NOT NULL,
    created_at TIMESTAMP DEFAULT NOW(),
    
    UNIQUE(job_id, sequence_id)  -- 保证顺序
);

-- 索引
CREATE INDEX idx_events_job_id ON events(job_id);
CREATE INDEX idx_events_sequence ON events(job_id, sequence_id);
```

### 5.2 Redis：缓存与加速

```go
// Redis 用于：
// 1. Lease 缓存（快速过期检查）
// 2. 热点 Job 状态缓存
// 3. RAG 向量索引（可选）
// 4. 分布式锁
```

## 6. 核心执行流程

```
用户请求
    │
    ▼
┌─────────────────┐
│  API Layer      │  创建 Job，生成 JobCreated 事件
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│  Job Manager    │  写入 JobStore，返回 Job ID
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│  Scheduler      │  分配 Worker，获取 Lease
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│  Runner         │
│  ├─ 加载 Job    │  从 Checkpoint 恢复或全新开始
│  ├─ PlanGoal   │  生成 TaskGraph
│  ├─ RunDAG     │  逐节点执行
│  │   ├─ LLM   │  调用模型
│  │   ├─ Tool  │  调用工具（检查 Ledger）
│  │   └─ Wait  │  暂停等待信号
│  ├─ Checkpoint│  每步完成后写入
│  └─ WriteEvent│  每步记录事件
└────────┬────────┘
         │
         ▼
      Job Complete / Failed
```

## 7. 关键设计权衡

### 7.1 事件流 vs 状态快照

| 方案 | 优点 | 缺点 |
|------|------|------|
| 纯事件流 | 完整历史、可审计 | 重放慢、存储大 |
| 纯快照 | 恢复快 | 丢失历史、难调试 |
| **Aetheris 方案** | 两者兼顾 | 需要维护一致性 |

### 7.2 Checkpoint 频率

- **太频繁**：写入压力大
- **太稀疏**：恢复时间长

Aetheris 默认**每步执行后**写入 Checkpoint，兼顾恢复速度和写入开销。

## 8. 小结

Aetheris 的核心架构围绕「**事件溯源**」和「**检查点恢复**」设计：

1. **JobStore** — 事件流是执行历史的唯一真相来源
2. **Checkpoint** — 定期保存状态快照，加速恢复
3. **Scheduler** — Lease + Fencing Token 防止重复执行
4. **Runner** — 负责任务执行、状态管理、事件写入

这套架构保证了 Agent 的**可靠性**—— 崩溃可恢复、执行可审计、状态可追溯。

---

*下篇预告：At-Most-Once 语义——如何保证工具调用不重复*
