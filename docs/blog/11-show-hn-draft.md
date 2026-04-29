# Show HN: Aetheris – Temporal-style durable execution runtime for AI agents (Go)

> **草稿用途**：Hacker News Show HN 帖子。正式发布前需最终审阅。  
> 保存路径：`docs/blog/11-show-hn-draft.md`

---

## HN 帖子正文

**标题候选（选一个）**

- `Show HN: Aetheris – Temporal-style durable execution runtime for AI agents (Go)`
- `Show HN: Aetheris – Agent execution with at-most-once tool calls and crash recovery (Go)`

---

### 正文

I've been building AI agents for the past year and kept hitting the same wall: agents work great in demos but break in production.

The core problem: **when an agent crashes mid-run, you have no idea what side effects it already caused.**

Example: An agent calls Stripe to process a refund, the Stripe call succeeds, then the process crashes before recording the result. On restart, the agent calls Stripe again. The user gets double-charged. This isn't a Stripe bug — it's an agent runtime problem.

LangGraph, CrewAI, and most agent frameworks are stateless orchestrators. They'll retry failed steps, but they don't track *which side effects already happened*. That's fine for demos, not fine for anything touching payments, emails, or external APIs.

**Aetheris** approaches this differently:

1. **At-most-once tool execution** — Every tool call gets an idempotency key derived from `{job_id}:{step_id}:{attempt}:{tool_name}:{input_hash}`. Before calling the tool, we check an InvocationLedger. If the key exists, we return the cached result without calling the tool again. No double Stripe charges.

2. **Event-sourced job history** — Every step transition is appended to a PostgreSQL event log. On crash recovery, the Worker replays the event log to reconstruct state. Steps that already committed their effects are injected from the log, not re-executed. LLM calls during replay don't actually call the LLM.

3. **Lease fencing** — Workers hold leases on jobs. If a Worker dies, its leases expire and other Workers reclaim the jobs. The fencing key prevents a "zombie" Worker from committing stale results.

4. **Deterministic replay** — Agent behavior is reconstructed entirely from the persisted event stream. `time.Now()`, random values, and external I/O in replay paths are either banned (via linter rules) or injected from the event log.

We also handle human-in-the-loop flows: an agent can park (`StatusParked`) while waiting for human approval, then resume with the response — across arbitrary worker restarts.

**Tech stack:** Go 1.26, cloudwego/eino (ByteDance's agent framework), Hertz, PostgreSQL for the job store. There's also an embedded mode (SQLite-backed) for zero-dependency local development.

**Quick start (zero Docker required):**

```bash
git clone https://github.com/Colin4k1024/Aetheris
cd Aetheris && go mod download && make build
CONFIG_PATH=configs/api.embedded.yaml go run ./cmd/api
# health check
curl http://localhost:8080/api/health
```

The repo has examples for ReAct agents, multi-agent collaboration, stateful agents, and human approval flows.

Still early. Known limitations: the Python/JS SDKs are minimal, the UI is a CLI, and the eino dependency means you're partially tied to ByteDance's agent primitives (we're working on making this pluggable).

Would love feedback on: (1) whether the at-most-once guarantee matters to you in practice, (2) what integrations you'd want first (LangChain? Temporal?), (3) whether the embedded-mode experience is actually zero-friction.

GitHub: https://github.com/Colin4k1024/Aetheris

---

## 关键论点清单（答复评论备用）

### "你们和 Temporal 有什么区别？"

Temporal 是通用工作流引擎，需要独立的 Temporal 服务器和 SDK。Aetheris 专门为 AI Agent 设计：
- 内置 LLM/Tool 效应捕获（Temporal 不知道什么是 LLM 调用）
- 内置 Human-in-the-Loop 状态机（StatusParked → resume）
- 开箱即用的 Agent 定义 YAML（不需要写工作流代码）
- 嵌入式模式无需任何服务器

### "你们和 LangGraph 有什么区别？"

LangGraph 是优秀的图编排框架，但默认无状态：
- 没有 InvocationLedger（同一工具调用在重试时会被再次执行）
- 没有跨进程持久化（LangGraph Cloud 有，但需付费）
- 没有 Worker 租约系统（无法扩展到多 Worker）

### "为什么用 Go 不用 Python？"

Agent Runtime 是基础设施层，Go 在这里比 Python 更合适：
- 并发 Worker 调度：goroutine 比 asyncio 更可控
- 内存占用可预测（LLM 调用本身已经很重了）
- 部署简单：单二进制，无 venv/依赖地狱
- 类型安全减少了 replay 路径中的静默错误

Python/JS SDK（封装 REST API）已有最小实现，不需要用 Go。

### "eino 依赖有多大的锁定风险？"

这是一个合理的担忧。当前状态：
- eino 处理 Agent 的推理循环和工具调用编排
- 持久化层（JobStore/EventStore/Ledger）完全独立于 eino
- 我们正在把 Agent 接口抽象化，目标是让 LangChain4Go、其他框架可以接入 runtime，只用 Aetheris 的持久化和调度层

短期内不会完全解耦，但 eino 是 Apache 2.0 的开源项目，ByteDance 内部大规模使用，应该有足够的维护动力。

---

## 发帖时间建议

HN 黄金时间（ET）：
- 周一 - 周四 8:00-10:00 AM
- 避免周五下午和周末

标签：`go`, `ai`, `agents`, `distributed-systems`
