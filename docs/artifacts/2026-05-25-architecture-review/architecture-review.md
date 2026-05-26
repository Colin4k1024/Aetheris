---
artifact: architecture-review
task: architecture-review
date: 2026-05-25
role: architect
status: draft-for-review
scope: repository-wide architecture assessment
---

# Aetheris Architecture Review

## 1. Review Conclusion

Aetheris 的整体架构方向是合理的。它不应被定位为又一个 Agent 框架，而应定位为 **Agent-native durable execution runtime**：在 LangChain、LangGraph、Eino、OpenAI Agents SDK、外部 HTTP Agent 等 Agent 构建层之外，提供可恢复、可重放、可审计、可治理的执行底座。

这个方向与业界 durable execution 的主线一致。Temporal、DBOS、Restate、LangGraph 都在解决“长任务如何跨崩溃恢复、如何避免重复执行已完成步骤、如何记录执行历史”的问题；Aetheris 的差异化在于将这些思想显式落到 LLM Agent、工具调用、副作用账本、外部黑盒 Agent 接入和合规取证上。

当前架构最有价值的核心是：

- Event Store / JobStore 作为执行事实源
- PlanGenerated 锁定执行路径，避免恢复时重新规划
- Invocation Ledger + Effect Store 管理 Tool/LLM 副作用
- Scheduler / Runner / Checkpoint 支撑长任务恢复
- external_http 提供低迁移成本的黑盒 Agent 接入路径

当前最需要收敛的风险是：

- 公开保证边界需要更精确，特别是 `external_http` 与 native Runtime Tool 的差异
- 执行主线存在概念重叠，`Eino AgentFactory`、`Durable Runner`、legacy Agent runtime 的权威边界需要进一步明确
- 事件模型承担过多职责，Replay-critical、Trace narrative、Audit/Evidence 事件应分层治理
- “exactly-once”类表述要改为条件化表达，避免超过无全局事务架构实际能证明的范围

总体判断：**架构方向成立，核心技术路线有业界依据，但需要一次边界收敛和保证语义降噪，优先级高于继续扩功能。**

## 2. Architecture Positioning

### 2.1 Recommended Positioning

Aetheris 的推荐定位：

> Aetheris is a reliability runtime for AI agents: it wraps existing and native agents with durable execution, replay, at-most-once side-effect boundaries, traceability, and governance.

换成中文：

> Aetheris 是智能体的可靠执行层，而不是智能体写法本身。

这个定位能解释为什么项目同时支持：

- native Go/Eino agent
- external HTTP black-box agent
- Tool / MCP tool plane
- RAG / ingest pipeline
- Job / trace / replay / evidence API

它们不是并列产品线，而是同一个运行时目标下的不同入口。

### 2.2 What Aetheris Should Not Claim

Aetheris 不应宣称自己在所有模式下都自动获得完整 exactly-once 或零重复副作用。更准确的表达是：

- 对 native Runtime Tool：在配置共享 JobStore、Invocation Ledger、Effect Store 后，可提供强 at-most-once side-effect boundary。
- 对 external_http：Aetheris 只控制外层 `external_agent_call`；外部 Agent 内部的支付、发信、写库等副作用仍需自行幂等，或迁移成 Runtime Tool。
- 对 embedded / memory dev mode：适合本地体验，不应承诺跨进程生产级恢复语义。

## 3. Current Architecture Evidence

### 3.1 Event-Sourced Runtime

证据：

- `internal/runtime/jobstore/event.go` 定义了 `JobCreated`、`PlanGenerated`、`NodeStarted`、`NodeFinished`、`CommandCommitted`、`ToolInvocationStarted`、`ToolInvocationFinished`、`StepCommitted`、`JobWaiting`、`WaitCompleted` 等事件。
- `JobEvent` 包含 `PrevHash` 与 `Hash`，说明事件链已承担审计与篡改检测基础。
- `design/execution-guarantees.md` 明确将事件流作为 Replay 和审计依据。

推理：

Event sourcing 是 durable execution 的正确底层模型。它能将“执行中状态”转化为“可重放历史”，使 Worker 崩溃、租约过期、恢复执行、审计取证都可以落到同一份事实源。

影响：

这个方向正确，但事件类型已经混合了执行语义、Trace 叙事、Memory、Compliance、Evidence 和业务 marker。后续需要分层，否则事件流会越来越难维护。

### 3.2 PlanGenerated as Decision Lock

证据：

- `design/internal/runtime-contract.md` 要求没有 `PlanGenerated` 不得执行或恢复。
- `internal/agent/runtime/executor/runner_test.go` 已覆盖无 `PlanGenerated` 时失败并置 Job Failed 的行为。
- `docs/artifacts/2026-05-08-external-agent-intake/arch-design.md` 说明 external_http 在 Job 创建时写入单节点 `PlanGenerated`。

推理：

LLM Planner 是非确定性来源。恢复或 Replay 时重新调用 Planner，会导致执行路径漂移。因此 “LLM proposes, runtime records, replay follows recorded plan” 是正确边界。

影响：

这是 Aetheris 区别于普通 Agent 框架的重要架构资产。后续所有新执行入口都必须先写入等价的 decision record，再进入 Runner。

### 3.3 Invocation Ledger and Effect Store

证据：

- `internal/agent/runtime/executor/ledger.go` 定义了 `Acquire`、`Commit`、`Recover` 三阶段 Invocation Ledger。
- `internal/agent/runtime/executor/effect_store.go` 定义了 Tool、LLM、HTTP、Time、Random、Human effect 记录。
- `internal/agent/runtime/executor/node_adapter.go` 中 LLM adapter 在配置 Effect Store 时写入 LLM effect，并在 Replay 防御路径下直接注入，不重新调用 LLM。
- `design/execution-guarantees.md` 将 Invocation Ledger、ToolInvocationStore、Effect Store 列为强保证条件。

推理：

普通 checkpoint 只能避免重复计算，不能自动避免重复外部副作用。Invocation Ledger 解决“能不能执行”，Effect Store 解决“已执行但事件未提交时如何恢复”。这两个组件是 Agent runtime 生产化的关键。

影响：

这是架构中最值得保留和加强的部分。需要把配置矩阵写清楚，并在生产启动时阻止“声明生产级保证但未启用 Ledger/Effect Store”的配置。

### 3.4 API / Worker Separation

证据：

- `design/services.md` 将 API 定义为控制面，Worker/agent-service 定义为数据面。
- 文档要求 Postgres 模式下 API 不执行 Job，Worker 通过 Claim / Heartbeat 执行。
- `internal/agent/job/scheduler.go` 体现了 Pending/Parked claim、WakeupQueue、并发限制、RetryMax、Backoff 等调度职责。

推理：

控制面与执行面分离是合理的。它避免 API 重启影响执行租约，也让 Worker 横向扩展、失败回收和运行时隔离成为可能。

影响：

需要继续检查实际 `app.go` 装配路径，确保生产模式不会在 API 进程里意外启动内存 Scheduler 或执行器。

### 3.5 External HTTP Agent Intake

证据：

- `docs/adapters/external-http-agent.md` 明确 external_http 是 MVP 迁移路径。
- 文档写明 Aetheris 只控制外层调用，外部 Agent 内部副作用不自动获得 at-most-once 保证。
- README 也提醒黑盒 demo 只展示单次外部 HTTP 调用的 durable submission 和 trace visibility，内部 per-step checkpoint 需要迁移到 native tools/workflows。

推理：

黑盒 Agent 接入是正确的商业和工程入口。它降低迁移门槛，让用户不用先重写 Agent 就能获得 Job、Trace、Timeout、Audit。但它不能承接 Aetheris 最强的副作用保证。

影响：

external_http 应被明确标为 Level 1 migration。高风险场景必须有 Level 2/3 迁移路径：外部副作用改造成 Runtime Tool，或外部服务强制接受并尊重 Aetheris idempotency key。

## 4. Industry Comparison

### 4.1 Temporal

Temporal 官方文档将 Workflow Execution 描述为 durable、reliable、scalable function execution；其 Event History 是 append-only 历史，用于 crash recovery、audit 和 replay。Temporal 也明确要求 Workflow code 确定性，Activity 承担外部副作用，并建议 Activity 幂等。

对 Aetheris 的启示：

- Aetheris 的 Event Store + Runner + Tool/LLM effect 边界是合理方向。
- Aetheris 应学习 Temporal 的强边界：Workflow/decision code 必须确定性，Activity/Tool 负责不可靠外部世界。
- Aetheris 不应把 Activity/Tool 的 exactly-once 说满；Temporal 也强调 Activity 默认 at-least-once，副作用依赖 idempotency 或配置约束。

参考：

- https://docs.temporal.io/workflow-execution
- https://docs.temporal.io/workflow-execution/event
- https://docs.temporal.io/develop/python/best-practices/error-handling

### 4.2 LangGraph

LangGraph 官方 durable execution 文档强调 checkpoint、resume、human-in-the-loop 和 long-running task。其 persistence 文档说明 checkpointer 会在 execution step 保存 graph state，并支持 time travel、fault-tolerant execution。

对 Aetheris 的启示：

- Aetheris 与 LangGraph 共享 Agent workflow checkpoint/resume 问题空间。
- Aetheris 的差异化应放在更强的 side-effect ledger、外部 Agent 包装、合规证据链，而不是单纯“也能 checkpoint”。
- LangGraph time travel 会重放 checkpoint 之后的 LLM/API 调用；Aetheris 如果坚持强 Replay 不重调 LLM/Tool，这是一个可被清晰讲出的差异点。

参考：

- https://docs.langchain.com/oss/python/langgraph/durable-execution
- https://docs.langchain.com/oss/python/langgraph/persistence
- https://docs.langchain.com/oss/python/langgraph/use-time-travel

### 4.3 DBOS

DBOS 官方架构文档说明其 durable workflows 基于 Postgres checkpoint workflows and steps；失败后从最后完成步骤恢复。DBOS 也要求 workflow function 确定性，非确定性操作和外部 API 应放在 step 中。

对 Aetheris 的启示：

- Aetheris 的 Postgres-first production lane 合理，尤其适合开源 self-host 和企业内网部署。
- DBOS 的 “one database write per step” 是一个很好的成本模型。Aetheris 需要建立类似的事件/step 写入成本说明。
- Aetheris 可以学习 DBOS 的简洁开发体验：用户只需要理解 Job、Step、Tool、Effect，而不是理解所有内部事件。

参考：

- https://docs.dbos.dev/architecture.md
- https://docs.dbos.dev/python/tutorials/workflow-tutorial

### 4.4 Restate

Restate 官方文档描述了 execution journal、durable handlers、workflow key deduplication、idempotency key、durable timers 和 durable promises。它通过 journal replay 跳过已记录动作并继续执行。

对 Aetheris 的启示：

- Aetheris 的 Job event stream 与 Restate journal 思路接近。
- Aetheris 的 signal / waiting / wakeup 模型与 Restate 的 durable promises / timers 是同一类需求。
- Restate 对 request lifecycle 的表达很清晰，Aetheris 应补一份同等清晰的 “Aetheris Job Lifecycle” 文档。

参考：

- https://docs.restate.dev/guides/request-lifecycle
- https://docs.restate.dev/tour/workflows
- https://docs.restate.dev/foundations/invocations
- https://docs.restate.dev/foundations/actions

### 4.5 OpenAI Agents SDK and AutoGen

OpenAI Agents SDK 强调 agents、tools、handoffs、guardrails、tracing；AutoGen Core 强调 agent runtime、message delivery、agent lifecycle、distributed runtime。

对 Aetheris 的启示：

- 这些框架证明 Agent runtime、handoff、tool surface、trace 是业界主线。
- 它们更偏 Agent 编排与可观测，不等同于 durable execution runtime。
- Aetheris 应与它们互补：作为执行可靠性层，而不是替代它们的 Agent 编写体验。

参考：

- https://openai.github.io/openai-agents-python/tracing/
- https://developers.openai.com/api/docs/guides/agents/define-agents
- https://microsoft.github.io/autogen/dev/user-guide/core-user-guide/framework/agent-and-agent-runtime.html

## 5. Architecture Fitness Assessment

| Dimension | Assessment | Reason |
|---|---|---|
| Problem validity | Strong | 生产 Agent 确实会遇到 crash recovery、duplicate side effects、audit/replay 问题 |
| Core architecture | Strong | Event sourcing + durable scheduler + ledger/effect store 是正确底座 |
| Industry alignment | Strong | Temporal、DBOS、Restate、LangGraph 都验证了 durable execution 方向 |
| Differentiation | Medium-Strong | Agent-specific side-effect ledger、LLM replay、external_http migration 是差异点 |
| Conceptual clarity | Medium | Eino-first、Aetheris Runner、legacy Agent runtime 的边界还需收敛 |
| Guarantee precision | Medium | 文档中强保证和条件保证有时混在一起 |
| Operational readiness | Medium | Status 文档已标记部分能力为 integrated/prototype，生产门禁还需继续收紧 |
| Maintainability | Medium | 事件类型和文档数量较多，需要分层和 single source of truth |

## 6. Key Risks and Recommendations

### R1. Guarantee Boundary Drift

风险：

用户可能从 README 理解为所有 Agent 自动获得 zero duplicates，但 external_http 黑盒内部副作用不受 Aetheris 控制。

建议：

- 在 README、quickstart、runtime guarantees 中统一增加 guarantee matrix。
- 每个入口标注语义等级：embedded dev、external_http、native Runtime Tool、production Postgres。
- 将 “Zero duplicates” 改成 “No duplicate Runtime Tool side effects under configured Ledger/Effect Store”。

### R2. Runtime Authority Split

风险：

文档同时强调 `AgentFactory` 是 Agent 构建统一入口，又强调 `Runner.RunForJob` 是 durable 执行核心。两者并不冲突，但容易被读成两个运行时。

建议：

- 明确分层语言：Eino builds agents/workflows; Aetheris executes jobs durably。
- 在 `design/core.md` 中补一张权威链路图：API creates Job and PlanGenerated -> Worker claims Job -> Durable Runner executes recorded plan -> Eino/ADK only used inside node/tool construction where applicable。
- 标记 legacy Agent runtime 的去留计划。

### R3. Event Taxonomy Overload

风险：

所有事件进入同一 `EventType` 后，Replay 关键事件、Trace 叙事事件、审计事件、业务 marker 的演进节奏会互相影响。

建议：

- 定义三层事件分类：
  - Execution History：影响恢复和 Replay，必须强兼容。
  - Trace Stream：用于 UI/调试，可演进。
  - Audit/Evidence Stream：用于合规和导出，需 schema version。
- 在代码中至少增加注释和分类文档，后续再考虑物理表拆分。

### R4. Effect Store Ordering Contract Needs One Truth

风险：

旧 artifact 中曾写过“事件流优先，EffectStore 可选加速层”，当前 `effect_store.go` 与 `execution-guarantees.md` 采用“PutEffect 后 Append，用 catch-up 防重复”的强 Replay 设计。两个说法不一致。

建议：

- 以当前 `design/execution-guarantees.md` / `effect_store.go` 为准。
- 将旧文档标记为 historical 或修正文案。
- 生成一份 `effect-store-contract.md`，明确 crash window、catch-up、ledger precedence、人工介入路径。

### R5. Product Surface Too Broad

风险：

项目同时覆盖 RAG、MCP、Eino、external_http、forensics、compliance、distributed、redaction、RBAC、dashboard 等，很容易让核心价值被稀释。

建议：

- 2.x 主线只保留一个北极星：可靠执行 Runtime。
- RAG/MCP/Compliance 作为 proof scenarios，不作为同级产品主线。
- 3.0 能力必须按 `docs/STATUS.md` 的 promotion policy 逐个垂直切片推进。

## 7. Recommended Architecture North Star

建议用以下分层作为未来所有文档、代码和官网叙事的统一模型：

```text
Agent Authoring Layer
  Eino / LangGraph / OpenAI Agents / AutoGen / Custom HTTP Agent

Aetheris Control Plane
  API / CLI / SDK / Auth / Tenant / Config / Job Submission

Aetheris Durable Execution Plane
  Scheduler / Lease / Runner / Step Executor / Wait-Signal / Retry

Aetheris Side-Effect Plane
  Runtime Tool / Invocation Ledger / Effect Store / Idempotency Key

Aetheris Evidence Plane
  Event History / Replay / Trace / Verify / Export / Audit

Durable Stores
  Postgres JobStore / EventStore / CheckpointStore / InvocationStore / EffectStore
```

核心原则：

- Agent framework 是上层，不是事实源。
- PlanGenerated / Event History 是执行事实源。
- Tool/LLM/HTTP/Human 都是 effect，必须被记录或显式标记为不受控。
- external_http 是迁移入口，不是完整保证边界。
- 生产级保证必须和存储配置绑定。

## 8. Decision Log

| Decision | Conclusion | Evidence | Implication |
|---|---|---|---|
| D1: 是否继续押注 durable execution runtime | Yes | 代码已有 EventStore、Runner、Ledger、Effect Store；业界 Temporal/DBOS/Restate/LangGraph 均验证方向 | 继续建设，但聚焦可靠执行 |
| D2: 是否把 Aetheris 定位为 Agent framework | No | 项目价值在执行可靠性，不在 prompt/agent 编写体验 | 官网和 README 应避免与 LangGraph/OpenAI Agents 正面替代叙事 |
| D3: 是否保留 external_http | Yes | 它是最低迁移成本入口 | 必须显式标注黑盒内部副作用边界 |
| D4: 是否继续扩展 enterprise/compliance | Selectively | `docs/STATUS.md` 已将多项能力标为 prototype | 每次只推进一个端到端 slice |
| D5: 是否需要重构事件模型 | Not immediately | 现有事件流可工作，但概念负载高 | 先文档分类，再代码治理 |

## 9. Next Actions

优先级 P0：

- 增加 guarantee matrix，统一 README、runtime guarantees、external_http docs。
- 明确 `Eino builds / Aetheris executes` 的权威链路。
- 修正旧 artifact 中与当前 Effect Store 顺序相冲突的描述。

优先级 P1：

- 为 EventType 建立分类文档：Execution / Trace / Audit。
- 为 production config 增加 Ledger/Effect Store 强校验。
- 补充 Job Lifecycle 官方架构页，对齐 Restate/Temporal 级别的表达清晰度。

优先级 P2：

- 将 forensics/compliance/distributed 等 prototype 逐个按 vertical slice 晋级。
- 为 external_http 设计 Level 2 migration guide：从黑盒 Agent 内部副作用迁移到 Runtime Tool。
- 建立成本模型：每个 Job/Step/Tool 会产生多少事件、多少 DB 写入。

## 10. Final Assessment

Aetheris 的架构不是空想。它解决的是一个真实且正在变热的问题：**Agent 不是一次性请求，而是长生命周期、有副作用、需要恢复和审计的执行过程。**

当前方向值得继续。但接下来最重要的不是再堆能力，而是把“我到底保证什么、在什么条件下保证、哪些模式不保证”讲清楚，并让代码、文档、配置门禁共同证明这些保证。
