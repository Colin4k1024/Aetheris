# Aetheris 文档 / Aetheris Documentation

[English](#overview) | [中文](#概述)

---

# Overview

**Aetheris** (also known as **CoRag** — Chunk-Omni Retrieval-Augmented Generation) is a durable, replayable execution runtime for intelligent AI agents — the "Temporal for Agents" that production AI systems desperately need.

## What Problem Does Aetheris Solve?

```
❌ Worker crashed → Restart from beginning
❌ Tool called twice → Duplicate payments  
❌ Need to audit AI decisions → No trace
❌ Agent waiting for approval → Wastes resources
❌ Need to replay failed run → Impossible
```

Aetheris provides **production-grade reliability** for AI agents built with LangChainGo, LangGraphGo, Google ADK, or any other agent framework.

---

## Core Concepts / 核心概念

| English | 中文 | Description |
|---------|------|-------------|
| **Event Sourcing** | 事件溯源 | Records all state changes as immutable events instead of storing current state |
| **Checkpoint** | 检查点 | State snapshot after step completion, enables resume from interruption |
| **At-Most-Once** | 最多执行一次 | Tool calls never repeat, even after crashes — guaranteed by Invocation Ledger |
| **Human-in-the-Loop** | 人机交互 | Agent pauses for approval, resumes without waste — uses Wait/Signal mechanism |
| **Lease Fencing** | 租约隔离 | Prevents duplicate execution when workers crash and restart |
| **DAG / TaskGraph** | 有向无环图/任务图 | Directed acyclic graph of step dependencies |
| **StepOutcome** | 步骤结果语义 | Each step produces exactly one outcome: Pure, SideEffectCommitted, Retryable, PermanentFailure, Compensated |

### Event Sourcing / 事件溯源

**Event Sourcing** is an architectural pattern where application state changes are recorded as an append-only sequence of immutable **Events**, rather than storing the current state directly.

```
Traditional: UPDATE user SET status='active' WHERE id=1
Event Sourcing: APPEND UserStatusChanged(user_id=1, old="pending", new="active", timestamp=...)
```

In Aetheris, the Job Store records events like `JobCreated`, `StepStarted`, `StepCompleted`, `ToolCalled`, `JobPaused`, `JobResumed`, `JobCompleted` — providing a complete audit trail and enabling deterministic replay.

### Checkpoint Recovery / 检查点恢复

When a Worker crashes mid-execution, Aetheris recovers by:
1. Reading the event history from Job Store
2. Reconstructing state by replaying events up to the last Checkpoint
3. Resuming execution from the next uncompleted step

```
Worker A: process step_3 → crash
Worker B: reads event history → replays step_1-2 → continues step_3
```

### At-Most-Once Execution / 最多执行一次

Aetheris guarantees tool calls never repeat through the **Invocation Ledger**:

1. Before executing a tool, Runner requests permission from Ledger
2. Ledger returns `AllowExecute` if no prior record exists; Runner executes and calls `Commit`
3. On replay/crash recovery, Ledger returns `ReturnRecordedResult` if already committed — tool is NOT re-executed

### Human-in-the-Loop / 人机交互

For scenarios requiring manual approval (legal contracts, payment authorization):

```
Plan → Generate contract → Wait(correlation_key="approval-123") → [Human approval 1-3 days]
                                                                    ↓
                                            Signal(approval-123, approved=true)
                                                                    ↓
                                            Send contract / Continue → Complete
```

- **Wait Node**: Job enters `StatusParked`, not占用执行槽
- **Signal API**: Call `POST /api/jobs/:id/signal` with `correlation_key` and `payload` to resume

---

## Architecture / 架构

```
┌─────────────────────────────────────────────────────────────────┐
│                     Aetheris Runtime                             │
├─────────────────────────────────────────────────────────────────┤
│  Authoring Layer          │  Control Plane                      │
│  (Eino, LangChainGo, ...) │  (API / CLI / SDK)                 │
├───────────────────────────┼─────────────────────────────────────┤
│  Data Plane (Runtime Core)│  Durable Stores                     │
│  ┌─────────────────────┐  │  ┌─────────────────────────────┐   │
│  │ Scheduler           │  │  │ Event Store (PostgreSQL)    │   │
│  │ Runner              │  │  │ Checkpoint Store            │   │
│  │ Tool Plane          │  │  │ Effect + Invocation Store   │   │
│  │ Replay/Verify       │  │  │ Job Metadata Store          │   │
│  └─────────────────────┘  │  └─────────────────────────────┘   │
└─────────────────────────────────────────────────────────────────┘
```

### Core Components / 核心组件

| Component | 组件 | Responsibility |
|-----------|------|----------------|
| **API Server** | API 服务器 | HTTP server (Hertz), creates and interacts with agents |
| **Worker** | 工作器 | Background execution worker, schedules and executes jobs |
| **CLI** | 命令行工具 | `init`, `chat`, `jobs`, `trace`, `replay` commands |
| **AgentFactory** | Agent 工厂 | Config-driven Eino ADK agent creation (recommended entry point) |
| **Job Store** | 任务存储 | Event-sourced durable execution history (PostgreSQL) |
| **Scheduler** | 调度器 | Leases and retries tasks with lease fencing |
| **Runner** | 执行器 | Step-level execution with checkpointing |
| **Invocation Ledger** | 调用账本 | At-most-once tool execution guarantee |

---

## Quick Start / 快速开始

### Prerequisites / 前置条件

- **Go 1.25.7+**
- **Docker** (optional, for PostgreSQL)

### Installation / 安装

```bash
# Install CLI
go install github.com/Colin4k1024/Aetheris/cmd/cli@latest

# Or use Docker
./scripts/local-2.0-stack.sh start

# Build from source
cd ~/Desktop/poc/CoRag
make build
```

### Start Services / 启动服务

```bash
# Quick mode (memory, no Docker needed)
go run ./cmd/api --config.jobstore.type=memory

# Full mode (PostgreSQL + event sourcing)
docker run -d --name aetheris-pg -p 5432:5432 \
  -e POSTGRES_USER=aetheris -e POSTGRES_PASSWORD=secret \
  -e POSTGRES_DB=aetheris postgres:15-alpine

psql "postgres://aetheris:secret@localhost:5432/aetheris?sslmode=disable" \
  -f internal/runtime/jobstore/schema.sql

go run ./cmd/api
```

### Run Your First Agent / 运行第一个 Agent

```bash
# Using CLI
./bin/aetheris chat --system "你是一个代码审查助手" \
  --message "请审查以下 Python 代码:\n\ndef add(a, b):\n    return a + b"

# Using HTTP API
curl -X POST http://localhost:8080/api/v1/agents/code-reviewer/run \
  -H "Content-Type: application/json" \
  -d '{"user_message": "请审查这段 Python 代码..."}'
```

### Monitor Execution / 监控执行

```bash
# List jobs
./bin/aetheris jobs list

# View event trace
./bin/aetheris trace <job_id>

# View logs
./bin/aetheris logs <job_id>
```

---

## Key API Reference / 关键 API 说明

### Create and Run Agent / 创建并运行 Agent

```bash
# Create agent
curl -X POST http://localhost:8080/api/v1/agents \
  -H "Content-Type: application/json" \
  -d '{
    "name": "code-reviewer",
    "model": "qwen-plus",
    "system_prompt": "你是一个专业的代码审查助手"
  }'
```

### Signal (Human Approval) / 信号通知（人工审批）

```bash
# Resume a parked job with approval result
curl -X POST http://localhost:8080/api/jobs/<job_id>/signal \
  -H "Content-Type: application/json" \
  -d '{
    "correlation_key": "approval-123",
    "payload": {"approved": true, "comment": "已阅，同意"}
  }'
```

### Query Job Status / 查询任务状态

```bash
curl http://localhost:8080/api/jobs/<job_id>
# Returns: status (running/completed/failed/waiting), events, checkpoints
```

### Trace Execution / 追踪执行

```bash
# View full event stream and execution timeline
./bin/aetheris trace <job_id>
```

Output example:
```
┌─────────────────────────────────────────────────────────────┐
│ Job: job_abc123 (code-reviewer)                            │
├─────────────────────────────────────────────────────────────┤
│ Created: 2025-03-17 10:00:00                               │
│ Status: COMPLETED (duration: 2.3s)                         │
├─────────────────────────────────────────────────────────────┤
│ Events:                                                     │
│ [10:00:00] JobCreated                                      │
│ [10:00:01] StepStarted: analyze_code                        │
│ [10:00:02] ToolCalled: llm.analyze                         │
│ [10:00:03] StepCompleted: analyze_code                     │
│ [10:00:04] JobCompleted                                    │
└─────────────────────────────────────────────────────────────┘
```

---

## Go SDK Example / Go SDK 示例

```go
package main

import (
    "context"
    "fmt"
    
    "rag-platform/internal/agent"
    "rag-platform/internal/agent/runtime"
)

func main() {
    // Create DAG Agent
    dag := agent.DAG{
        Nodes: []agent.Node{
            {
                Name: "fetch",
                Run: func(ctx context.Context, input any) (any, error) {
                    fmt.Println("📥 Fetching data...")
                    return []string{"item1", "item2", "item3"}, nil
                },
            },
            {
                Name: "process",
                Run: func(ctx context.Context, input any) (any, error) {
                    items := input.([]string)
                    fmt.Printf("📝 Processing %d items...\n", len(items))
                    results := make([]string, len(items))
                    for i, item := range items {
                        results[i] = item + " (processed)"
                    }
                    return results, nil
                },
            },
            {
                Name: "save",
                Run: func(ctx context.Context, input any) (any, error) {
                    results := input.([]string)
                    fmt.Printf("💾 Saving %d results...\n", len(results))
                    return len(results), nil
                },
            },
        },
        Edges: []agent.Edge{
            {From: "fetch", To: "process"},
            {From: "process", To: "save"},
        },
    }
    
    // Run DAG
    result, err := dag.Run(context.Background(), nil)
    fmt.Printf("✅ Result: %v\n", result)
}
```

---

## StepOutcome Semantics / 步骤结果语义

Each step produces exactly one outcome:

| Outcome | 语义 | Meaning |
|---------|------|---------|
| **Pure** | 纯函数 | No side effects; safe to replay |
| **SideEffectCommitted** | 已提交副作用 | World changed; must not re-execute |
| **Retryable** | 可重试 | Failure, world unchanged; retry allowed |
| **PermanentFailure** | 永久失败 | Failure; job cannot continue |
| **Compensated** | 已回滚 | Rollback applied; terminal state |

---

## Execution Guarantees / 执行保证

| Guarantee | 保证 | Description |
|-----------|------|-------------|
| **At-Most-Once** | 最多执行一次 | Tool calls never repeat, even after crashes |
| **Crash Recovery** | 崩溃恢复 | Agents resume from checkpoints, not from scratch |
| **Deterministic Replay** | 确定性重放 | Reproduce any run for debugging or auditing |
| **Event Sourcing** | 事件溯源 | Full execution history as append-only event stream |

---

## Comparison / 对比

| Problem | Without Aetheris | With Aetheris |
|---------|------------------|---------------|
| Worker crash | Restart from beginning | Resume from checkpoint |
| Duplicate calls | Possible ($$$ loss) | Guaranteed at-most-once |
| Debug | Guess what happened | Deterministic replay |
| Audit | Impossible | Full evidence chain |
| Human approval | Wastes resources | StatusParked |

---

## Project Structure / 项目结构

```
CoRag/
├── cmd/                  # Entry points (api, worker, cli)
├── internal/              # Private application code
│   ├── agent/            # Agent runtime (execution, scheduling, recovery)
│   ├── api/              # HTTP API
│   ├── runtime/          # Runtime core (eino workflow orchestration)
│   └── storage/          # Data storage
├── pkg/                  # Public libraries
├── configs/              # Configuration files
├── docs/                 # Documentation
│   ├── guides/           # User guides
│   ├── concepts/         # Core concepts
│   └── blog/             # Technical articles
└── examples/             # Example code
```

---

## Documentation Links / 文档链接

| Document | 文档 | Description |
|----------|------|-------------|
| [guides/quickstart.md](./guides/quickstart.md) | 快速入门 | 5-minute quickstart tutorial |
| [guides/getting-started-agents.md](./guides/getting-started-agents.md) | Agent 开发指南 | Agent development guide |
| [concepts/event-sourcing.md](./concepts/event-sourcing.md) | 事件溯源 | Event sourcing concept (Chinese) |
| [blog/05-human-in-the-loop.md](./blog/05-human-in-the-loop.md) | 人机交互 | Human-in-the-loop approval flow |
| [blog/06-at-most-once-ledger.md](./blog/06-at-most-once-ledger.md) | 调用账本 | At-Most-Once Ledger principle |
| [guides/runtime-guarantees.md](./guides/runtime-guarantees.md) | 执行保证 | Runtime guarantees and semantics |
| [reference/config.md](./reference/config.md) | 配置参考 | Configuration reference |

---

## License / 许可证

Apache License 2.0 — free for commercial use.

---

<div align="center">

**Built with [eino](https://github.com/cloudwego/eino), [hertz](https://github.com/cloudwego/hertz), [pgx](https://github.com/jackc/pgx)**

**Aetheris — The Missing Layer for Production-Ready AI Agents**

</div>

---

# 概述

**Aetheris**（又称 **CoRag** — Chunk-Omni Retrieval-Augmented Generation）是一个持久化、可重放的人工智能 Agent 执行运行时 —— 堪称"Agent 领域的 Temporal"，专为生产级 AI 系统打造。

## Aetheris 解决什么问题？

```
❌ Worker 崩溃 → 从头开始
❌ 工具被调用两次 → 重复付款
❌ 需要审计 AI 决策 → 无迹可循
❌ Agent 等待审批 → 浪费资源
❌ 需要重放失败的任务 → 不可能
```

Aetheris 为使用 LangChainGo、LangGraphGo、Google ADK 或任何其他 Agent 框架构建的 AI Agent 提供**生产级可靠性**。

---

## 核心概念

| 英文 | 中文 | 说明 |
|------|------|------|
| **Event Sourcing** | 事件溯源 | 将所有状态变更记录为不可变事件序列，而非直接存储当前状态 |
| **Checkpoint** | 检查点 | 步骤完成后的状态快照，支持从中断处恢复 |
| **At-Most-Once** | 最多执行一次 | 工具调用永不重复，即使崩溃也不受影响 —— 通过调用账本实现 |
| **Human-in-the-Loop** | 人机交互 | Agent 暂停等待审批，恢复后无缝继续执行 |
| **Lease Fencing** | 租约隔离 | 防止 Worker 崩溃重启后的重复执行 |
| **DAG / TaskGraph** | 有向无环图/任务图 | 步骤依赖的有向无环图 |
| **StepOutcome** | 步骤结果语义 | 每个步骤产生唯一结果：Pure、SideEffectCommitted、Retryable、PermanentFailure、Compensated |

### 事件溯源

**事件溯源**是一种架构模式，应用程序的状态变更通过记录一系列不可变的**事件**来表示，而非直接存储当前状态。

```
传统模式：UPDATE user SET status='active' WHERE id=1
事件溯源：APPEND UserStatusChanged(user_id=1, old="pending", new="active", timestamp=...)
```

在 Aetheris 中，Job Store 记录 `JobCreated`、`StepStarted`、`StepCompleted`、`ToolCalled`、`JobPaused`、`JobResumed`、`JobCompleted` 等事件 —— 提供完整的审计追踪并支持确定性重放。

### 检查点恢复

当 Worker 在执行过程中崩溃时，Aetheris 通过以下方式恢复：
1. 从 Job Store 读取事件历史
2. 通过重放事件重建到上一个检查点的状态
3. 从下一个未完成的步骤继续执行

### 最多执行一次

Aetheris 通过**调用账本（Invocation Ledger）**保证工具调用永不重复：

1. 执行工具前，Runner 向 Ledger 请求许可
2. 若无先前记录，Ledger 返回 `AllowExecute`；Runner 执行并调用 `Commit`
3. 重放/崩溃恢复时，若已提交，Ledger 返回 `ReturnRecordedResult` —— 工具**不会**重新执行

### 人机交互

对于需要人工审批的场景（法律合同、付款授权）：

```
计划 → 生成合同 → Wait(correlation_key="approval-123") → [人工审批 1-3 天]
                                                                    ↓
                                            Signal(approval-123, approved=true)
                                                                    ↓
                                            发送合同 / 继续执行 → 完成
```

---

## 快速开始

### 前置条件

- **Go 1.25.7+**
- **Docker**（可选，用于 PostgreSQL）

### 安装

```bash
# 安装 CLI
go install github.com/Colin4k1024/Aetheris/cmd/cli@latest

# 或使用 Docker
./scripts/local-2.0-stack.sh start

# 从源码构建
cd ~/Desktop/poc/CoRag
make build
```

### 启动服务

```bash
# 快速模式（内存模式，无需 Docker）
go run ./cmd/api --config.jobstore.type=memory

# 完整模式（PostgreSQL + 事件溯源）
docker run -d --name aetheris-pg -p 5432:5432 \
  -e POSTGRES_USER=aetheris -e POSTGRES_PASSWORD=secret \
  -e POSTGRES_DB=aetheris postgres:15-alpine

psql "postgres://aetheris:secret@localhost:5432/aetheris?sslmode=disable" \
  -f internal/runtime/jobstore/schema.sql

go run ./cmd/api
```

### 运行第一个 Agent

```bash
# 使用 CLI
./bin/aetheris chat --system "你是一个代码审查助手" \
  --message "请审查以下 Python 代码:\n\ndef add(a, b):\n    return a + b"

# 使用 HTTP API
curl -X POST http://localhost:8080/api/v1/agents/code-reviewer/run \
  -H "Content-Type: application/json" \
  -d '{"user_message": "请审查这段 Python 代码..."}'
```

### 监控执行

```bash
# 列出任务
./bin/aetheris jobs list

# 查看事件追踪
./bin/aetheris trace <job_id>

# 查看日志
./bin/aetheris logs <job_id>
```

---

## 关键 API 说明

### 创建并运行 Agent

```bash
curl -X POST http://localhost:8080/api/v1/agents \
  -H "Content-Type: application/json" \
  -d '{
    "name": "code-reviewer",
    "model": "qwen-plus",
    "system_prompt": "你是一个专业的代码审查助手"
  }'
```

### 信号通知（人工审批）

```bash
curl -X POST http://localhost:8080/api/jobs/<job_id>/signal \
  -H "Content-Type: application/json" \
  -d '{
    "correlation_key": "approval-123",
    "payload": {"approved": true, "comment": "已阅，同意"}
  }'
```

### 查询任务状态

```bash
curl http://localhost:8080/api/jobs/<job_id>
# 返回: status (running/completed/failed/waiting), events, checkpoints
```

---

## 项目链接

- [GitHub 仓库](https://github.com/Colin4k1024/Aetheris)
- [快速入门](./guides/quickstart.md)
- [Agent 开发指南](./guides/getting-started-agents.md)
- [事件溯源概念](./concepts/event-sourcing.md)
- [运行时保证](./guides/runtime-guarantees.md)
- [配置参考](./reference/config.md)
