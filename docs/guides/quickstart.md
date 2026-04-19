# Quickstart — 5 分钟入门 Aetheris

> 本指南帮助你快速启动 Aetheris Runtime 并运行第一个 Agent DAG

## 目标

- 启动本地 Aetheris API
- 运行一个简单的代码审查 Agent
- 观察事件溯源和任务执行

## 前置条件

- **Go 1.25.7+**
- **Docker** (可选，用于 Postgres)

---

## 步骤 1: 克隆并构建项目

```bash
# 克隆项目
cd ~/Desktop/poc/CoRag

# 构建所有二进制
make build

# 或者只构建 CLI
go build -o bin/aetheris ./cmd/cli
```

## 步骤 2: 选择运行模式

### 模式 A: 快速体验 (内存模式，无需 Docker)

```bash
# 修改配置使用内存模式
# 编辑 configs/api.yaml，找到 jobstore.type，改为 memory

# 或者直接用命令行覆盖:
go run ./cmd/api --config.jobstore.type=memory
```

### 模式 B: 完整功能 (PostgreSQL + 事件溯源)

```bash
# 启动 PostgreSQL
docker run -d --name aetheris-pg -p 5432:5432 \
  -e POSTGRES_USER=aetheris -e POSTGRES_PASSWORD=aetheris \
  -e POSTGRES_DB=aetheris postgres:15-alpine

# 应用 schema
psql "postgres://aetheris:aetheris@localhost:5432/aetheris?sslmode=disable" \
  -f internal/runtime/jobstore/schema.sql

# 启动 API (默认使用 postgres)
go run ./cmd/api
```

## 步骤 3: 启动服务

```bash
# 终端 1: 启动 API
go run ./cmd/api

# 终端 2: 健康检查
curl http://localhost:8080/api/health
```

预期输出:
```json
{"status":"ok","version":"2.3.0"}
```

## 步骤 4: 创建并运行第一个 Agent

使用 CLI 创建一个简单的代码审查 Agent:

```bash
# 使用 CLI 创建 Agent 并提交任务
./bin/aetheris chat --system "你是一个代码审查助手" \
  --message "请审查以下 Python 代码:\n\ndef add(a, b):\n    return a + b"
```

或者使用 HTTP API:

```bash
# 创建 Agent Job
curl -X POST http://localhost:8080/api/v1/agents \
  -H "Content-Type: application/json" \
  -d '{
    "name": "code-reviewer",
    "model": "qwen-plus",
    "system_prompt": "你是一个专业的代码审查助手。检查代码的潜在问题、性能和最佳实践。"
  }'
```

```bash
# 提交任务
curl -X POST http://localhost:8080/api/v1/agents/code-reviewer/run \
  -H "Content-Type: application/json" \
  -d '{
    "user_message": "请审查这段 Python 代码:\n\ndef fibonacci(n):\n    if n <= 1:\n        return n\n    return fibonacci(n-1) + fibonacci(n-2)"
  }'
```

## 步骤 5: 查看执行结果

```bash
# 查看 Job 状态
./bin/aetheris jobs list

# 查看 Job 详情和事件流
./bin/aetheris trace <job_id>

# 查看实时日志
./bin/aetheris logs <job_id>
```

## 步骤 6: 体验崩溃恢复 (可选)

```bash
# 1. 启动 Worker
go run ./cmd/worker

# 2. 提交一个长时间运行的任务
# 3. 在执行过程中手动停止 Worker (Ctrl+C)
# 4. 重新启动 Worker
go run ./cmd/worker

# 5. 观察任务自动恢复执行
./bin/aetheris trace <job_id>
```

## 代码示例: 运行 DAG

创建一个自定义 DAG:

```go
// main.go
package main

import (
    "context"
    "fmt"
    
    "github.com/Colin4k1024/Aetheris/v2/internal/agent"
    "github.com/Colin4k1024/Aetheris/v2/internal/agent/runtime"
)

func main() {
    // 创建 DAG Agent
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
    
    // 运行 DAG
    result, err := dag.Run(context.Background(), nil)
    fmt.Printf("✅ 结果: %v\n", result)
}
```

运行:

```bash
go run main.go
```

预期输出:
```
📥 Fetching data...
📝 Processing 3 items...
💾 Saving 3 results...
✅ 结果: 3
```

## 事件流可视化

提交任务后，查看事件流:

```bash
# 使用 trace 命令
./bin/aetheris trace job_abc123
```

输出示例:
```
┌─────────────────────────────────────────────────────────────┐
│ Job: job_abc123 (code-reviewer)                            │
├─────────────────────────────────────────────────────────────┤
│ Created: 2025-03-17 10:00:00                               │
│ Status: COMPLETED (duration: 2.3s)                         │
├─────────────────────────────────────────────────────────────┤
│ Events:                                                     │
│ ┌─────────────────────────────────────────────────────────┐ │
│ │ [10:00:00] JobCreated                                   │ │
│ │ [10:00:01] StepStarted: analyze_code                   │ │
│ │ [10:00:02] ToolCalled: llm.analyze                     │ │
│ │ [10:00:03] StepCompleted: analyze_code                 │ │
│ │ [10:00:04] JobCompleted                                 │ │
│ └─────────────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────────┘
```

## 下一步

- [事件溯源概念](./concepts/event-sourcing.md) — 深入理解 Aetheris 的核心设计
- [Code Review Agent 教程](./tutorials/code-review-agent.md) — 构建企业级代码审查 Agent
- [Audit Agent 教程](./tutorials/audit-agent.md) — 构建合规审计 Agent
- [长程任务教程](./tutorials/long-running-tasks.md) — 处理小时级任务
