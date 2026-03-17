# 事件溯源 (Event Sourcing) 概念指南

> 面向 Python 开发者的 Aetheris 事件溯源入门指南

## 什么是事件溯源？

**事件溯源 (Event Sourcing)** 是一种架构模式，在这种模式中，应用程序的状态变化不是直接存储为"当前状态"，而是通过记录一系列不可变的**事件 (Events)** 来表示。

### 核心概念对比

| 传统持久化模式 | 事件溯源模式 |
|--------------|-------------|
| 存储当前状态 (State) | 存储状态变更历史 (Events) |
| UPDATE / DELETE | 仅 APPEND (追加) |
| 当前状态是真实来源 | 事件序列是真实来源 |
| 难以追溯历史 | 完整可追溯 (Audit Trail) |

### Python 开发者熟悉的类比

如果你使用过 Python 的 **`dataclasses`** 或 **Pydantic**，可以将事件溯源类比为：

```python
# 传统模式：直接存储最终状态
class User:
    def __init__(self, name: str, email: str, status: str):
        self.name = name
        self.email = email
        self.status = status  # 当前状态

# 事件溯源：存储所有状态变更
events = [
    UserCreated(name="张三", email="zhangsan@example.com"),
    EmailChanged(old="zhangsan@old.com", new="zhangsan@example.com"),
    StatusChanged(old="pending", new="active"),
]
# 当前状态 = 从事件序列重建
```

## Aetheris 中的事件溯源

Aetheris 是一个为 AI Agent 设计的**持久化运行时**，它的核心就是基于事件溯源。

### 核心组件

```
┌─────────────────────────────────────────────────────────────────┐
│                        Aetheris Runtime                         │
├─────────────────────────────────────────────────────────────────┤
│  ┌─────────────┐    ┌─────────────┐    ┌─────────────────────┐ │
│  │   Job       │───▶│  Scheduler │───▶│  Runner (执行器)    │ │
│  │  (任务)     │    │  (调度器)   │    │                     │ │
│  └─────────────┘    └─────────────┘    └─────────────────────┘ │
│         │                                       │               │
│         ▼                                       ▼               │
│  ┌──────────────────────────────────────────────────────────┐  │
│  │              Job Store (PostgreSQL)                      │  │
│  │  ┌────────────┬────────────┬────────────┬──────────────┐  │  │
│  │  │ JobCreated │StepStarted│StepCompleted│StepFailed   │  │  │
│  │  │ JobPaused  │JobResumed │ JobCanceled │ JobCompleted │  │  │
│  │  └────────────┴────────────┴────────────┴──────────────┘  │  │
│  │                   不可变事件序列                          │  │
│  └──────────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────────┘
```

### 事件类型

Aetheris 的 Job Store 记录以下核心事件：

| 事件 | 说明 |
|------|------|
| `JobCreated` | 任务创建 |
| `JobScheduled` | 任务已调度 |
| `StepStarted` | 步骤开始执行 |
| `StepCompleted` | 步骤成功完成 |
| `StepFailed` | 步骤执行失败 |
| `ToolCalled` | 工具被调用 |
| `ToolResult` | 工具返回结果 |
| `JobPaused` | 任务暂停 (等待人类审批等) |
| `JobResumed` | 任务恢复 |
| `JobCanceled` | 任务取消 |
| `JobCompleted` | 任务完成 |

### 事件示例 (JSON)

```json
{
  "event_id": "evt_abc123",
  "job_id": "job_xyz789",
  "event_type": "StepCompleted",
  "timestamp": "2025-03-17T10:30:00Z",
  "payload": {
    "step_id": "step_001",
    "step_name": "analyze_code",
    "output": "发现 3 个潜在问题",
    "duration_ms": 1250
  }
}
```

## 为什么选择事件溯源？

### 1. 完全可追溯 (Full Audit Trail)

```python
# Python 中实现类似的审计日志
import logging
from dataclasses import dataclass, field
from typing import List
from datetime import datetime

@dataclass
class Event:
    event_type: str
    timestamp: datetime = field(default_factory=datetime.now)
    payload: dict = field(default_factory=dict)

class EventStore:
    def __init__(self):
        self._events: List[Event] = []
    
    def append(self, event: Event):
        self._events.append(event)  # 只追加，不修改
    
    def get_history(self, entity_id: str) -> List[Event]:
        return [e for e in self._events if e.payload.get("entity_id") == entity_id]
```

### 2. 崩溃恢复 (Crash Recovery)

```
传统模式:                    事件溯源模式:
┌──────────────┐            ┌──────────────┐
│ Worker A     │            │ Worker A     │
│ 处理 step_3 │ 崩溃        │ 处理 step_3 │ 崩溃
│ 保存状态...  │            │ 记录事件     │
└──────────────┘            └──────────────┘
                               │
                               ▼
┌──────────────┐            ┌──────────────┐
│ Worker B     │            │ Worker B     │
│ 不知道 step_3│            │ 读取事件历史 │
│ 状态！       │            │ 重放 step_1-2│
│              │            │ 继续 step_3  │
└──────────────┘            └──────────────┘
```

### 3. 时间旅行 (Time Travel)

```go
// Aetheris 中可以重放任意历史状态
// 类似 Python: for event in history: state = apply(state, event)

// 1. 获取 Job 的完整事件历史
events, err := jobStore.GetEvents(ctx, jobID)

// 2. 重放事件重建任意时间点的状态
for _, event := range events {
    if event.Timestamp > targetTime {
        break
    }
    state = replay(state, event)
}

// 3. 甚至可以"回溯"重新执行
```

### 4. 多租户与并发

事件溯源天然支持**乐观锁**和**Lease Fencing**：

```go
// Lease Fencing - 防止 Worker 崩溃后的双跑问题
type Lease struct {
    JobID      string
    WorkerID   string
    LeaseToken int64  // 每次获取 Lease 递增
    ExpiresAt  time.Time
}

// 只有持有有效 Lease 的 Worker 才能执行
if job.Lease.WorkerID == myWorkerID && job.Lease.LeaseToken == currentToken {
    // 安全执行
    executeStep()
}
```

## Python 开发者如何理解？

### 类比 Django/Flask 的 ORM

```python
# 传统 ORM: 保存当前状态
user = User.objects.get(id=1)
user.status = "active"
user.save()  # UPDATE users SET status='active' WHERE id=1

# 事件溯源: 记录状态变更
# UserStatusChanged(user_id=1, old="pending", new="active", timestamp=...)
# 每次查询时: SELECT * FROM events WHERE user_id=1 
#            -> 按 timestamp 排序 -> 重放重建当前状态
```

### 类比 Redux (前端状态管理)

```javascript
// Redux: 基于 Action 的状态管理
const reducer = (state, action) => {
  switch (action.type) {
    case 'INCREMENT':
      return { count: state.count + 1 };
    case 'DECREMENT':
      return { count: state.count - 1 };
  }
};

// Aetheris: 类似但针对分布式 AI Agent
// Action = Event
// Reducer = Event Handler / Replay Logic
```

## Aetheris 事件溯源实战

### 启动带事件溯源的运行时

```bash
# 1. 启动 PostgreSQL (事件存储)
docker run -d --name aetheris-pg -p 5432:5432 \
  -e POSTGRES_USER=aetheris -e POSTGRES_PASSWORD=aetheris \
  -e POSTGRES_DB=aetheris postgres:15-alpine

# 2. 应用 schema
psql "postgres://aetheris:aetheris@localhost:5432/aetheris?sslmode=disable" \
  -f internal/runtime/jobstore/schema.sql

# 3. 启动 API (默认使用 postgres jobstore)
go run ./cmd/api

# 4. 启动 Worker (可选，支持多 Worker 分布式执行)
go run ./cmd/worker
```

### 查看事件历史

```bash
# 使用 CLI 查看 Job 的事件流
./bin/aetheris trace <job_id>

# 或直接查询数据库
psql "postgres://aetheris:aetheris@localhost:5432/aetheris?sslmode=disable" \
  -c "SELECT event_type, payload, created_at FROM job_events ORDER BY created_at;"
```

### 实现一个可恢复的 Agent

```go
// Aetheris 中的 Agent 本质上就是一个事件处理器
agent := &Agent{
    Name: "code_reviewer",
    Steps: []Step{
        {
            Name: "fetch_code",
            Run: func(ctx context.Context, input any) (any, error) {
                // 从 GitHub 获取代码
                return fetchCodeFromGitHub(input.(string))
            },
        },
        {
            Name: "analyze",
            Run: func(ctx context.Context, input any) (any, error) {
                // 使用 LLM 分析代码
                return analyzeWithLLM(input.(string))
            },
        },
        {
            Name: "report",
            Run: func(ctx context.Context, input any) (any, error) {
                // 生成报告
                return generateReport(input.(*AnalysisResult))
            },
        },
    },
}

// 运行时自动处理:
// 1. 记录 JobCreated 事件
// 2. 每个 Step 执行前记录 StepStarted
// 3. 每个 Step 执行后记录 StepCompleted/StepFailed
// 4. 崩溃恢复: 重放事件历史，从断点继续
```

## 总结

| 概念 | 说明 |
|------|------|
| **Event (事件)** | 不可变的状态变更记录 |
| **Event Store** | 事件持久化存储 (PostgreSQL) |
| **Job** | Agent 的一次执行任务 |
| **Step** | Job 中的单个执行步骤 |
| **Replay** | 从事件历史重放重建状态 |
| **Lease Fencing** | 防止分布式环境下的双跑 |

事件溯源让 Aetheris 成为 **"Temporal for Agents"** — 一个真正可靠、可追溯、可恢复的 AI Agent 运行时。

## 下一步

- [Quickstart 快速入门](./quickstart.md) — 5 分钟启动第一个 Agent
- [教程: Code Review Agent](./tutorials/code-review-agent.md) — 实战代码审查 Agent
- [教程: Audit Agent](./tutorials/audit-agent.md) — 实战审计 Agent
- [教程: 长程任务](./tutorials/long-running-tasks.md) — 处理小时级任务
