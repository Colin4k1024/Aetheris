# 使用 Aetheris 构建生产级 AI Agent

> 本文介绍 Aetheris 作为 Agent Hosting Runtime 与传统方案的区别，以及如何在生产环境中利用其崩溃恢复、At-Most-Once 执行等特性。

## 传统方案的痛点

在生产环境中运行 AI Agent 时，开发者经常遇到：

| 痛点 | 影响 |
|------|------|
| **服务重启丢失状态** | Job 执行到一半被中断，用户需重新发起 |
| **重复执行** | 网络超时导致重试，工具被调用多次（如扣款两次） |
| **不可观测** | 出了问题无法追溯，只能看日志猜原因 |
| **无法审计** | 合规场景无法证明 Agent 的决策过程 |

## Aetheris 的解决思路

### 1. 虚拟进程模型

Aetheris 将 Agent 视为**虚拟进程**，具有：
- **持久化状态**：Job 事件存储在 PostgreSQL，重启不丢
- **Lease Fencing**：Scheduler 通过租约防止多 Worker 抢执行
- **Checkpoint**：执行过程中的状态可恢复

```
User → API → Job → Scheduler → Runner → Planner → TaskGraph → Tools
                     ↑                              |
                     └────── 事件存储 (Postgres) ────┘
```

### 2. At-Most-Once 执行保证

Aetheris 通过 **Effects Ledger** 确保工具调用只执行一次：

```go
// 伪代码示例
toolCallID := ledger.Commit(tool.Name, tool.Input)
defer func() {
    if err != nil {
        ledger.Rollback(toolCallID) // 补偿
    }
}()
```

关键点：
- **工具调用前**：先记录到 Ledger，返回 `tool_call_id`
- **执行后**：标记为 `committed`
- **失败时**：自动触发补偿（Rollback）

### 3. 崩溃恢复

```bash
# 启动完整运行时（需要 Postgres）
docker compose -f deployments/compose/docker-compose.yml up -d

# 启动 API + Worker
go run ./cmd/api &
go run ./cmd/worker &
```

**测试崩溃恢复**：

```bash
# 1. 发送一个长时间任务
curl -X POST http://localhost:8080/api/agents/<id>/message \
  -d '{"message":"执行一个复杂任务"}'

# 2. 在执行过程中 kill Worker
pkill -f "go run ./cmd/worker"

# 3. 重新启动 Worker
go run ./cmd/worker &

# 4. 检查 Job 状态 - 应该继续执行并完成
curl http://localhost:8080/api/jobs/<job_id>
```

### 与 LangGraph/AutoGen 的对比

| 特性 | Aetheris | LangGraph | AutoGen |
|------|----------|-----------|---------|
| 持久化执行 | ✅ 事件溯源 | ❌ 内存状态 | ❌ 内存状态 |
| At-Most-Once | ✅ Ledger | ❌ 需要自行实现 | ❌ 需要自行实现 |
| 多 Worker | ✅ Scheduler | ❌ 单进程 | ❌ 需要额外编排 |
| 证据包导出 | ✅ 完整取证 | ❌ | ❌ |
| Trace UI | ✅ 内置 | 需额外集成 | 需额外集成 |

## 生产部署建议

### 1. 配置高可用

```yaml
# configs/api.yaml
jobstore:
  type: postgres
  dsn: "postgres://user:pass@host:5432/aetheris?sslmode=require"

# 启动多个 Worker
go run ./cmd/worker &
go run ./cmd/worker &
```

### 2. 监控与告警

Aetheris v2.2.0+ 提供：
- **Prometheus 指标**：`aetheris_queue_backlog`、`aetheris_stuck_job_count`
- **OpenTelemetry**：自动追踪 HTTP 请求
- **Jaeger**：分布式追踪

```bash
# 启动完整栈（包含监控组件）
make docker-run
# 访问 Grafana: http://localhost:3000
# 访问 Jaeger: http://localhost:16686
```

### 3. SLA/Quota 管理

v2.2.0 引入了 SLA Quota Manager：

```bash
# 配置 SLA
curl -X POST http://localhost:8080/api/sla/quotas \
  -H "Content-Type: application/json" \
  -d '{"agent_id":"agent-1","max_rpm":100,"max_daily":10000}'
```

## 总结

Aetheris 为生产级 AI Agent 提供了一站式解决方案：
- **虚拟进程模型** = 持久化 + 状态恢复
- **Effects Ledger** = At-Most-Once 执行保证
- **证据链** = 可审计、可追溯
- **完整监控栈** = 可观测性开箱即用

如果你的项目需要：
- 长时间运行的 Agent 任务
- 关键业务场景（如支付、审批）
- 合规审计要求

Aetheris 是值得考虑的选择。

## 延伸阅读

- [执行保证详解](./guides/runtime-guarantees.md)
- [可观测性配置](./guides/observability.md)
- [故障排查](./guides/get-started.md#故障排查)
