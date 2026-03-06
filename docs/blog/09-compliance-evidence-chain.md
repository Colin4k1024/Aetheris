# 合规审计与 Evidence Chain：为什么 AI 做出了这个决策

> 本文介绍 Aetheris 如何记录「谁、何时、为什么做了什么决策」，以及如何通过 Evidence Graph、Decision Snapshot、Reasoning Snapshot 与 Trace 回答「为什么 AI 发送了这封邮件」等合规与审计问题。

## 需求：可审计、可归因

在金融、医疗、政府等场景，不仅需要「能跑」，还需要：

- 记录**谁**（哪个 Agent/用户）、**何时**、**为什么**做了某个决策；
- 决策依据可追溯：用了哪些数据（如 RAG 文档）、调用了哪些 API、LLM 的输入输出与参数是什么；
- **Execution Proof Chain**：执行证明链，便于审计与事后复盘。

Aetheris 将这类能力统称为**证据链（Evidence Chain）**与**可审计执行**。

## Aetheris 能力概览

| 能力                     | 说明                                                                          |
| ------------------------ | ----------------------------------------------------------------------------- |
| **Evidence Graph**       | 记录 RAG doc IDs、tool invocation IDs、LLM model/temperature 等，关联到每一步 |
| **Decision Snapshot**    | Planner 级：为什么生成这个 Plan（goal、task_graph_summary、plan_hash）        |
| **Reasoning Snapshot**   | Step 级：goal、state_before、state_after、evidence（本步依据）                |
| **Event Stream + Trace** | 完整时间线事件流，Trace UI 与 API 可展示「为什么 AI 做出了这个决策」          |

产品表述：不仅是「可追踪」，而且是**可审计、可归因、可回答为什么**。

## 典型流程（概念）

```
PlanGenerated(evidence: goal, memory_keys)
  → Step1(evidence: rag_doc_ids, llm_decision)
  → Step2(evidence: tool_invocation_id)
  → Trace 展示完整证据链
```

- **Plan 层**：决策快照说明「为什么选这个 TaskGraph」。
- **Step 层**：每步的 reasoning_snapshot 与 evidence 说明「用了哪些输入、产生了什么输出、调用了哪些 Tool」。
- **Trace**：聚合为时间线，支持按节点查看 State Diff、证据与推理快照。

## 如何使用 Trace 与 API

### Trace 页面

每个 Job 执行完成后，可打开 Trace 页面查看时间线与证据：

```bash
# 获取 job_id 后访问
open http://localhost:8080/api/jobs/<job_id>/trace/page
```

页面上可看到：

- 时间线条：plan、node、tool、recovery 等片段；
- 节点列表与详情：点击某节点查看 state_before/state_after、evidence、tool invocation 等；
- 决策与推理快照（若已接入 Decision/Reasoning Snapshot 事件）。

### 结构化 API

```bash
# 获取 Trace 数据（含 execution_tree、reasoning_snapshot 等）
curl http://localhost:8080/api/jobs/<job_id>/trace

# 获取原始事件流（审计用）
curl http://localhost:8080/api/jobs/<job_id>/events
```

审计或合规导出时，可基于事件流与 Trace 数据生成「决策依据报告」。

## 与企业级功能文章的区别

- [Aetheris 企业级功能指南](./04-enterprise-features.md) 侧重 **RBAC、审计日志、数据脱敏、留存策略** 等权限与运维策略。
- 本文侧重 **决策归因与证据链**：为什么 AI 做出了某个动作、依据是什么、如何从 Trace/Event 中还原。

两者互补：04 偏「谁能在系统里做什么、数据如何保留与脱敏」，本文偏「单次执行的决策依据与证据」。

## 小结

- **Evidence Graph + Decision/Reasoning Snapshot** 记录 Plan 与每步的决策依据；**Event Stream + Trace** 提供完整时间线与查询接口。
- 通过 Trace 页面与 `/api/jobs/:id/trace`、`/api/jobs/:id/events` 可回答「为什么 AI 做出了这个决策」，满足合规与审计需求。
- 适合金融、医疗、政府等需要可审计、可归因的场景。

## 延伸阅读

- [design/core.md](../../design/core.md) §13.4 使用场景（合规审计）
- [Aetheris 可观测性实战](./03-observability.md)
- [Aetheris 企业级功能指南](./04-enterprise-features.md)
