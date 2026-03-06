# 何时选择 Aetheris：与 LangGraph、Temporal 的对比

> 本文帮助技术选型决策者与架构师快速判断 Aetheris 是否适合当前场景，并了解其与 LangGraph、Temporal 等方案的差异。

## Aetheris 的定位

Aetheris 是 **Agent Workflow Runtime**（可类比「Temporal 之于工作流」）：

- 核心是**任务编排、事件溯源、恢复与可观测**，而非单一 AI 应用或一次性对话。
- 强调**可信执行**：At-Most-Once、崩溃恢复、Replay、审计与证据链。
- RAG/检索/生成以 **Pipeline 或工具** 形式接入，是默认可选能力之一，并非 Runtime 唯一内置场景。

因此更适合「需要持久化、可恢复、可审计的 Agent 执行」的场景，而不是「无状态对话」或「短时原型」。

## 适用场景（推荐使用 Aetheris）

| 场景                   | 简要说明                                                             |
| ---------------------- | -------------------------------------------------------------------- |
| **Human-in-the-Loop**  | 审批流、人工确认后再继续；Wait/Signal、长时间等待不丢状态            |
| **长任务**             | 数据处理、报告生成、批量导入；崩溃后从 Checkpoint 恢复、进度可查     |
| **外部集成**           | 扣款、发短信、发邮件、写 CRM；Tool 调用需 At-Most-Once，不能重复执行 |
| **合规审计**           | 金融、医疗、政府；需要「谁、何时、为什么做了什么决策」的证据链       |
| **多步推理且需可恢复** | 研究助手、销售流程等多步 DAG；中间失败可重试，已完成步不重执行       |

详见 [Human-in-the-Loop](./05-human-in-the-loop.md)、[长任务与 Checkpoint 恢复实战](./10-long-running-checkpoint.md)、[合规审计与 Evidence Chain](./09-compliance-evidence-chain.md) 等。

## 反例：不适合 Aetheris 的场景

| 场景                           | 为什么不适合                        | 更合适的方案              |
| ------------------------------ | ----------------------------------- | ------------------------- |
| **无状态聊天机器人**           | 单次请求/响应，无需持久化、多步恢复 | LangChain + stateless API |
| **原型 / Demo Agent**          | 崩溃可接受，无审计需求              | LangGraph + 内存存储      |
| **纯内存短任务（&lt;1 分钟）** | 执行时间短，崩溃风险低              | 直接调用 LLM API          |
| **无外部副作用**               | 不调用 API/数据库，Replay 无意义    | 纯函数 + 缓存             |

若业务仅需「一问一答」或快速试跑想法，用 Aetheris 会引入不必要的复杂度（Postgres、Worker、事件流等）；可先用轻量方案，待有持久化与审计需求再迁到 Aetheris。

## 与 LangGraph、Temporal 的对比（概念）

| 维度              | Aetheris                         | LangGraph               | Temporal                   |
| ----------------- | -------------------------------- | ----------------------- | -------------------------- |
| **定位**          | Agent 可信执行运行时             | 状态图/Agent 编排库     | 通用工作流引擎             |
| **持久化**        | 事件流 + JobStore（Postgres）    | 多为内存或自建          | 自带持久化与事件历史       |
| **At-Most-Once**  | 内置（Ledger + Replay）          | 需自行实现              | 通过 Activity 等语义保证   |
| **多 Worker**     | Scheduler + 租约 + Lease Fencing | 通常单进程              | 原生多 Worker、任务分发    |
| **人机协同**      | Wait/Signal、StatusParked        | 需自行建模              | 支持 Signal/Query 等       |
| **审计 / 证据链** | Evidence Graph、Trace、事件流    | 需自行集成              | 有历史与查询，偏工作流审计 |
| **适用**          | Agent 长任务、合规、人机协同     | 快速搭建 Agent 图、原型 | 通用业务流程、长时工作流   |

- **LangGraph**：适合快速把 Agent 图跑起来、状态在内存或自管存储即可的场景；要 At-Most-Once、多 Worker、完整审计需自己补。
- **Temporal**：适合通用工作流（不限于 Agent）；若你主要做「AI Agent 的可信执行与证据链」，Aetheris 在 Agent 侧语义更贴合并开箱可用。
- **Aetheris**：专注 Agent 的持久化执行、恢复、At-Most-Once 与审计，适合上述「适用场景」中的需求。

## 小结

- **选 Aetheris**：Human-in-the-Loop、长任务、有外部副作用（支付/短信/邮件等）、合规审计、多步可恢复推理。
- **不选 Aetheris**：无状态对话、原型 Demo、纯内存短任务、无外部副作用。
- 与 LangGraph、Temporal 的差异：Aetheris 面向 **Agent 的可信执行与证据链**，在持久化、At-Most-Once、多 Worker、人机协同与审计上开箱可用；LangGraph 偏灵活编排与原型，Temporal 偏通用工作流。

## 延伸阅读

- [design/core.md](../../design/core.md) §13 使用场景与 §13.6 反例
- [Aetheris 入门 - 5 分钟快速开始](./01-quick-start.md)
- [使用 Aetheris 构建生产级 AI Agent](./02-production-agents.md)
