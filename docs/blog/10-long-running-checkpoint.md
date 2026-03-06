# 长任务与 Checkpoint 恢复实战

> 本文介绍 Aetheris 如何支持长时间运行的任务（如数据处理、报告生成、批量导入），并在 Worker 崩溃后从最近 Checkpoint 恢复、不重做已完成步骤，同时提供进度可查（如已完成 3/5 步）。

## 场景与需求

典型长任务场景包括：

- 数据处理（约 1 小时）：拉取 → 清洗 → 聚合 → 写回
- 报告生成（约 30 分钟）：多数据源 → 计算 → 生成 PDF/Excel
- 批量导入（约 2 小时）：读取文件 → 解析 → 校验 → 入库

需求可以归纳为：

- 任务运行时间远大于 1 分钟，**Worker 可能中途崩溃**；
- 崩溃后应**从最近完成步恢复**，不重新执行已完成步骤；
- **进度可追踪、可审计**（例如已完成 3/5 步、当前卡在哪一步）。

## Aetheris 能力概览

| 能力             | 说明                                                                 |
| ---------------- | -------------------------------------------------------------------- |
| **Checkpoint**   | 每步完成后写入 Checkpoint，更新 Job.Cursor；恢复时从 Cursor 之后继续 |
| **Event Stream** | 完整事件流可推导「已完成哪些节点」，用于展示进度（如 3/5 步）        |
| **At-Most-Once** | 已完成的步骤在 Replay 时只注入结果不重执行，避免重复拉数据、重复写库 |

典型流程（概念）：

```
Plan → 拉取数据(10 min) → Checkpoint → 清洗数据(20 min) → Checkpoint
  → 生成报告(30 min) → Checkpoint → 发送报告 → 完成
```

任一步完成后都有 Checkpoint；若在「生成报告」中途崩溃，新 Worker 会从事件流 Replay，前两步只注入结果，从「生成报告」继续（或重试该步）。

## 实战：构造长任务并验证恢复

### 1. 准备环境

使用 Postgres + API + Worker（见 [02-production-agents](./02-production-agents.md) 或 [07-event-sourcing-replay](./07-event-sourcing-replay.md)）。

### 2. 发起多步任务

通过 Agent 消息触发一个会执行多步的 Job（具体步骤取决于你的 Planner 与 TaskGraph 配置）：

```bash
curl -X POST http://localhost:8080/api/agents/<agent_id>/message \
  -H "Content-Type: application/json" \
  -d '{"message":"执行一个多步数据处理任务"}'
```

记录返回的 `job_id`。

### 3. 查询进度

在任务执行过程中或恢复后，可查 Job 状态与事件流，推断进度：

```bash
# Job 状态（含 status、cursor 等）
curl http://localhost:8080/api/jobs/<job_id>

# 事件流（可解析已完成节点数）
curl http://localhost:8080/api/jobs/<job_id>/events

# Trace 页面（时间线、节点列表、已完成步骤）
open http://localhost:8080/api/jobs/<job_id>/trace/page
```

Trace 页面上可看到已完成的节点与当前执行到哪一步，便于展示「已完成 3/5 步」等进度。

### 4. 模拟崩溃与恢复

- 在任务执行中途终止 Worker：`pkill -f "go run ./cmd/worker"`；
- 再启动 Worker：`go run ./cmd/worker &`；
- 新 Worker 会通过 Reclaim/Claim 认领该 Job，从事件流 Replay 并从 Cursor 继续；
- 再次查询 `GET /api/jobs/<job_id>` 或 Trace 页面，确认任务从断点继续并最终完成。

## 小结

- **长任务**依赖每步后的 **Checkpoint + Cursor**，崩溃后从 Cursor 继续，已完成的步通过事件流/Ledger 注入结果不重执行。
- **进度**可通过 Event Stream 与 Trace 推导（已完成节点、当前节点），用于前端或运维展示。
- 与 [事件溯源与 Replay 恢复](./07-event-sourcing-replay.md) 机制一致，本文侧重长任务场景与实战验证。

## 延伸阅读

- [design/core.md](../../design/core.md) §13.2 使用场景（长任务）
- [事件溯源与 Replay 恢复：从崩溃中继续执行](./07-event-sourcing-replay.md)
- [使用 Aetheris 构建生产级 AI Agent](./02-production-agents.md)
