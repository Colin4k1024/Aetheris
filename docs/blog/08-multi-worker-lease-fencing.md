# 多 Worker 与 Lease Fencing：分布式调度如何不重复执行

> 本文介绍在部署多个 Worker 时，Aetheris 如何通过 Job 级租约、attempt_id 校验与 Lease Fencing 避免同一 Job 被多个 Worker 同时执行或重复写事件。

## 问题：多 Worker 可能带来的冲突

若多个 Worker 同时从 JobStore 拉取任务：

- 可能**同时认领同一 Job**，导致同一批步骤被两个进程执行；
- 或 Worker A 失去租约后仍向事件流/Ledger 写入，与 Worker B 的写入冲突或重复。

因此需要**执行权归属**清晰：同一时刻只有一个 Worker 有权推进该 Job，且失去租约的 Worker **不能再写**。

## 机制概览

| 机制                                | 作用                                                                                                  |
| ----------------------------------- | ----------------------------------------------------------------------------------------------------- |
| **Job 级租约（Claim / Heartbeat）** | 只有成功 Claim 某 Job 的 Worker 才能执行；Heartbeat 续租，超时未续则视为放弃                          |
| **attempt_id**                      | Claim 时生成，写入事件与 Ledger 时校验；非当前 attempt 的写入被拒绝（ErrStaleAttempt）                |
| **Reclaim**                         | 仅依据「租约过期」回收 Job，不回收 Blocked（如 job_waiting）的 Job；回收后其他 Worker 可重新 Claim    |
| **Lease Fencing**                   | 事件 Append、Ledger Commit、Cursor 更新等写操作均在「当前租约持有者」下进行，否则拒绝或由约定保证不写 |

## Lease Fencing 范围

以下写操作都必须在「当前持有该 Job 租约的 Worker」下进行，否则会被拒绝或通过 attempt_id 拒绝：

| 写操作            | 强制方式                                                                                         |
| ----------------- | ------------------------------------------------------------------------------------------------ |
| **事件 Append**   | Event store 按 attempt_id 校验；非当前 attempt 返回 ErrStaleAttempt                              |
| **Ledger Commit** | InvocationLedger 可配置 AttemptValidator；Commit 前校验 context 中 job 的 attempt 仍为当前持有者 |
| **Cursor 更新**   | 仅由持有租约的 Worker 在 RunForJob 中调用；失去租约的 Worker 必须停止执行并不再调用 Runner       |

因此失去租约的 Worker 即使尚未进程退出，其后续写入也会被存储层拒绝，不会与新城主冲突。

## 配置要点

- **JobStore**：生产使用 `jobstore.type: postgres`，多 Worker 共享同一 Postgres。
- **ToolInvocationStore**：Invocation Ledger 的持久化需共享（如同一 Postgres），以便多 Worker 看到同一套「是否已提交」记录。
- **Worker 侧**：启动多个 Worker 进程（或副本），每个都会参与 Claim；Heartbeat 在 `executeJob` 内通过 ticker 定期调用，超时或进程退出后由 Reclaim 回收 Job，其他 Worker 可认领并继续。

## 与 At-Most-Once 文章的分工

- [At-Most-Once 执行保证：Invocation Ledger 原理](./06-at-most-once-ledger.md) 侧重**单次执行的裁决**：Ledger 决定某步是执行还是注入已记录结果。
- 本文侧重**谁在执行**：租约、attempt_id、多 Worker 下的 Reclaim 与 Lease Fencing，保证「同一 Job 同一时刻只由一个 Worker 推进」。

## 小结

- **Job 级租约 + attempt_id** 保证执行权唯一；**Reclaim** 只回收过期租约的 Job，不回收 Blocked。
- **Lease Fencing** 保证事件 Append、Ledger Commit、Cursor 更新等写操作仅由当前租约持有者完成，失去租约的 Worker 写入被拒绝。
- 多 Worker 部署时需共享 Postgres（JobStore + 可选 ToolInvocationStore），并依赖 Heartbeat 与 Reclaim 形成闭环。

## 延伸阅读

- [design/internal/scheduler-correctness.md](../../design/internal/scheduler-correctness.md) — 租约、两步提交、Step 状态
- [design/execution-guarantees.md](../../design/execution-guarantees.md) — 保证一览与条件
- [At-Most-Once 执行保证：Invocation Ledger 原理](./06-at-most-once-ledger.md)
