# 事件溯源与 Replay 恢复：从崩溃中继续执行

> 本文介绍 Aetheris 如何用事件流（Event Sourcing）存储 Job 执行历史，以及崩溃后如何通过 Replay 重建状态并从 Checkpoint 继续执行。

## Job 以事件流存储

Aetheris 不只为 Job 存「当前状态」，而是存**完整事件流**，例如：

| 事件类型                                               | 含义                                          |
| ------------------------------------------------------ | --------------------------------------------- |
| `job_created`                                          | Job 被创建                                    |
| `plan_generated`                                       | Planner 产出 TaskGraph                        |
| `node_started` / `node_finished`                       | 某节点开始/结束                               |
| `tool_invocation_started` / `tool_invocation_finished` | 工具调用开始/完成                             |
| `command_committed`                                    | LLM/命令类步骤已提交（Effect Store 两步提交） |
| `state_changed`                                        | 状态变更（如 state_before → state_after）     |
| `job_completed` / `job_failed`                         | Job 结束                                      |

事件按 **job_id + version** 顺序追加，支持乐观并发（Append 时校验 expectedVersion）。这样无论何时崩溃，**已发生的事实**都在事件流里，可被后续 Replay 使用。

## Checkpoint 与恢复点

- 每步（节点）执行完成后，Runner 会写入 **Checkpoint**，并更新 **Job.Cursor** 指向该恢复点。
- 崩溃或重启后，新 Worker 认领该 Job 时：
  1. 从 JobStore **ListEvents(job_id)** 拉取完整事件流；
  2. 用 **ReplayContextBuilder** 从事件构建出 ReplayContext（TaskGraphState、CompletedNodeIDs、CompletedToolInvocations、StateChangesByStep 等）；
  3. 根据 **Job.Cursor** 从对应节点之后继续执行，未完成的节点才真正跑，已完成的从事件/Ledger 注入结果。

因此恢复是**从最近 Checkpoint 继续**，而不是从头重跑。

## Confirmation Replay：已提交步不再执行

对已产生副作用的步骤（Tool 调用、LLM 调用），Replay 时**禁止再次执行**：

- 事件流或 Ledger 中若已有该步的 `tool_invocation_finished` / `command_committed`，Runner 只做 **Confirmation**（可选：校验外部世界与记录一致），然后**注入已记录结果**，不调用 Tool/LLM。
- 这样既保证 **At-Most-Once**，又保证恢复后行为与「从未崩溃」一致。

## 流程图（概念）

```
[Worker A 执行 Step1 → 写 Checkpoint → 写事件]
         ↓
[Worker A 崩溃]
         ↓
[Worker B Claim Job → ListEvents → 构建 ReplayContext]
         ↓
[从 Cursor 恢复：Step1 从事件注入结果，不重执行]
         ↓
[执行 Step2, Step3, ... → 继续写 Checkpoint 与事件]
```

更详细的 Runner ↔ Ledger ↔ JobStore 序列见 [design/runtime-core-diagrams.md](../../design/runtime-core-diagrams.md)。

## 实战：验证崩溃恢复

1. **启动 Postgres + API + Worker**（见 [02-production-agents](./02-production-agents.md) 部署建议）。

2. **发起一个多步任务**：

   ```bash
   curl -X POST http://localhost:8080/api/agents/<agent_id>/message \
     -H "Content-Type: application/json" \
     -d '{"message":"执行一个需要多步的任务"}'
   ```

   记录返回的 `job_id`。

3. **在执行过程中终止 Worker**：

   ```bash
   pkill -f "go run ./cmd/worker"
   ```

4. **重新启动 Worker**：

   ```bash
   go run ./cmd/worker &
   ```

5. **观察 Job 状态**：新 Worker 会通过 Reclaim 或下一轮 Claim 认领该 Job，从事件流 Replay 并从 Cursor 继续，最终应完成或进入下一等待状态：
   ```bash
   curl http://localhost:8080/api/jobs/<job_id>
   # 或查看 Trace 页面
   open http://localhost:8080/api/jobs/<job_id>/trace/page
   ```

## 小结

- **事件流**记录 Job 的完整历史；**Checkpoint + Cursor** 标记恢复点。
- **Replay** 从事件流重建状态，已提交的 Tool/LLM 步只注入结果不重执行，从 Cursor 继续执行未完成步骤。
- 崩溃后新 Worker 认领 Job、Replay、继续执行，形成可恢复的闭环。

## 延伸阅读

- [design/runtime-core-diagrams.md](../../design/runtime-core-diagrams.md) — Runner–Ledger–JobStore 序列与 StepOutcome
- [At-Most-Once 执行保证：Invocation Ledger 原理](./06-at-most-once-ledger.md)
- [使用 Aetheris 构建生产级 AI Agent](./02-production-agents.md)
