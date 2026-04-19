# Aetheris Blog

Welcome to the Aetheris blog! Here you'll find tutorials, guides, and insights about building production-ready AI Agents.

## Latest Posts

### Getting Started

- [Aetheris 入门 - 5 分钟快速开始](./01-quick-start.md)
- [使用 Aetheris 构建生产级 AI Agent](./02-production-agents.md)

### Advanced Topics

- [Aetheris 可观测性实战](./03-observability.md)
- [Aetheris 企业级功能指南](./04-enterprise-features.md)

### Human-in-the-Loop & Workflow

- [Human-in-the-Loop：审批流与 Wait/Signal](./05-human-in-the-loop.md) — 人工审批后再继续、Signal API 唤醒任务

### Execution Guarantees & Recovery

- [At-Most-Once 执行保证：Invocation Ledger 原理](./06-at-most-once-ledger.md) — Ledger 裁决、Replay 不重执行、生产配置条件
- [事件溯源与 Replay 恢复：从崩溃中继续执行](./07-event-sourcing-replay.md) — 事件流、Checkpoint、Confirmation Replay
- [多 Worker 与 Lease Fencing：分布式调度如何不重复执行](./08-multi-worker-lease-fencing.md) — 租约、attempt_id、Lease Fencing 范围

### Compliance & Long-Running

- [合规审计与 Evidence Chain：为什么 AI 做出了这个决策](./09-compliance-evidence-chain.md) — 证据链、Decision/Reasoning Snapshot、Trace 取证
- [长任务与 Checkpoint 恢复实战](./10-long-running-checkpoint.md) — 长任务场景、进度可查、崩溃恢复验证

### Choosing Aetheris

- [何时选择 Aetheris：与 LangGraph、Temporal 的对比](./11-when-to-choose-aetheris.md) — 适用场景、反例、选型对比表
- [构建灵活的多框架 Agent：Aetheris 多框架支持](./12-multi-framework-agent-support.md) — LangChainGo、LangGraphGo、Google ADK 等框架集成

### New (v2.5.3)

- [再见了 LangChain——我的 AI Agent 终于不会在生产环境崩溃了](./13-goodbye-langchain-crash-recovery.md) — 实战对比：Worker 崩溃 5 次、Job 自动恢复、0 重复执行

### Runtime & Architecture

- [为什么 AI Agent 需要自己的 Runtime](./2026-04-14-why-agents-need-runtime.md) — Agent 与微服务本质差异、事件溯源、检查点、At-Most-Once 执行保证

## Quick Links

- [Documentation](../guides/get-started.md)
- [GitHub Repository](https://github.com/aetheris-ai/CoRag)
- [Discord Community](https://discord.gg/PrrK2Mua)
