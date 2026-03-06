# At-Most-Once 执行保证：Invocation Ledger 原理

> 本文深入介绍 Aetheris 如何通过 Invocation Ledger（工具调用账本）保证同一逻辑步的 Tool/LLM 调用最多执行一次，以及 Replay 时为何不会重复调用外部 API。

## 问题：为什么需要 At-Most-Once

在生产环境中，以下情况会导致**同一工具或 LLM 调用被执行多次**：

| 情况          | 后果                                                                            |
| ------------- | ------------------------------------------------------------------------------- |
| **重试**      | 某步超时或失败，Scheduler 重新入队，若直接重跑整步会再次调用 Tool               |
| **崩溃**      | Worker 在 Tool 执行后、写回结果前崩溃，新 Worker 从 Checkpoint 恢复时会重跑该步 |
| **多 Worker** | 两个 Worker 同时认领同一 Job（租约未及时生效时），可能都去执行同一 Step         |

一旦涉及**外部副作用**（扣款、发短信、发邮件、写库），重复执行会带来资损或合规风险。因此 Aetheris 将 **At-Most-Once** 作为运行时契约：在给定条件下，同一逻辑步的副作用**至多执行一次**。

## Ledger 的角色：谁有权执行

Aetheris 的 **Invocation Ledger**（工具调用账本）不保存「是否执行」的布尔值，而是做**执行权裁决**：

- **Runner 不直接执行 Tool**：执行前先向 Ledger 请求许可。
- **Ledger 返回**：
  - **AllowExecute**：尚无该步的已提交记录，允许执行；执行成功后调用 `Commit` 写入结果。
  - **ReturnRecordedResult**：该步已有 committed 记录（或 Replay 注入的结果），**禁止**再执行，只把已记录结果注入上下文并返回。

因此，**唯一会真正调用 Tool 的代码路径**是：Ledger 返回 `AllowExecute` → 执行 Tool → `Commit`。Replay 或重试时，Ledger 发现已有记录即返回 `ReturnRecordedResult`，Runner 只做结果注入，不调用 Tool。

## Replay 语义：为何不会再次调用 LLM/Tool

- **Replay** 指崩溃或换 Worker 后，从事件流重建执行状态并继续。
- 重建时会把事件流里已存在的 `tool_invocation_finished`、`command_committed` 等整理成 **CompletedToolInvocations / CompletedCommandIDs**，注入到 Runner 的上下文中。
- 对每个 Tool/LLM 节点，Adapter 调用 `InvocationLedger.Acquire(..., replayResult)`；若 `replayResult` 非空或 Ledger/Store 中已有该 idempotency_key 的提交记录，则返回 **ReturnRecordedResult**，Runner **禁止**调用 Tool/LLM，只注入结果。

所以：**Replay = 查账本/事件恢复结果，有记录则恢复、无记录才执行；已提交的 Tool/LLM 步永不再次执行。**

## 成立条件（生产必配）

| 配置                                            | 作用                                                                 |
| ----------------------------------------------- | -------------------------------------------------------------------- |
| **JobStore（如 PostgreSQL）**                   | 事件流持久化，多 Worker 共享                                         |
| **InvocationLedger + 共享 ToolInvocationStore** | 裁决「是否允许执行」、存储已提交的 Tool 结果                         |
| **Effect Store（可选但推荐）**                  | LLM 调用与两步提交：先 PutEffect 再 Append，崩溃后 catch-up 不重执行 |

单进程或未配置 Ledger 时，仅适合**开发/单 Worker**；跨进程或多 Worker 场景必须配置上述组件，否则无法保证 At-Most-Once。

## 与「生产级 Agent」文章的关系

- [使用 Aetheris 构建生产级 AI Agent](./02-production-agents.md) 侧重**怎么用**、与传统方案对比、部署建议。
- 本文侧重**原理与契约**：Ledger 裁决、Replay 不重执行、以及生产必须满足的配置条件。

## 小结

- **Invocation Ledger** 裁决每一步是否允许执行；只有 `AllowExecute` 时才会真正调用 Tool/LLM，并随后 `Commit`。
- **Replay** 时从事件流/Ledger 恢复已提交结果，禁止对已提交步再次执行，从而保证 At-Most-Once。
- 生产环境需配置 Postgres JobStore、InvocationLedger、共享 ToolInvocationStore；可选 Effect Store 以强化崩溃后的 catch-up 语义。

## 延伸阅读

- [design/execution-guarantees.md](../../design/execution-guarantees.md) — 正式保证一览与条件
- [使用 Aetheris 构建生产级 AI Agent](./02-production-agents.md)
