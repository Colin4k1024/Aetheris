# Show HN: Aetheris – Temporal-style durable execution runtime for AI agents (Go)

> **Status**: Ready to post
> **Prepared**: 2026-07-04
> **Target time**: Monday-Thursday 8:00-10:00 AM ET

---

## HN Post Body

I've been building AI agents for the past year and kept hitting the same wall: agents work great in demos but break in production.

The core problem: **when an agent crashes mid-run, you have no idea what side effects it already caused.** An agent calls Stripe to process a refund, the call succeeds, the process dies before recording the result. On restart, it calls Stripe again. The user gets double-charged. This isn't a Stripe bug -- it's an agent runtime problem.

LangGraph, CrewAI, and most agent frameworks are stateless orchestrators. They retry failed steps but don't track *which side effects already happened*. Fine for demos, not for anything touching payments, emails, or external APIs.

**Aetheris** approaches this differently:

1. **At-most-once tool execution** -- Every tool call gets an idempotency key derived from `{job_id}:{step_id}:{attempt}:{tool_name}:{input_hash}`. Before calling the tool, we check an InvocationLedger. If the key exists, we return the cached result. No double charges.

2. **Event-sourced job history** -- Every step transition is appended to a PostgreSQL event log. On crash recovery, the Worker replays the log to reconstruct state. Steps that already committed their effects are injected from the log, not re-executed.

3. **Lease fencing** -- Workers hold leases on jobs. If a Worker dies, its leases expire and others reclaim the jobs. A fencing key prevents "zombie" Workers from committing stale results.

4. **Deterministic replay** -- Agent behavior is fully reconstructable from the persisted event stream. External I/O in replay paths is either banned or injected from the log.

We also handle human-in-the-loop: agents can park (`StatusParked`) while waiting for approval, then resume with the response -- across arbitrary worker restarts.

**Production proof** (not just claims):

- Crash recovery demo with step-by-step verification: `examples/crash_recovery/`
- Grafana dashboard for live observability: `deployments/compose/grafana/`
- k6 load test at 100 concurrent jobs: `benchmarks/k6/`
- Go benchmarks for JobStore throughput: `benchmarks/reports/`
- Formal runtime guarantees matrix: `docs/guides/runtime-guarantees.md`

**Tech stack:** Go 1.26, cloudwego/eino, Hertz, PostgreSQL. Embedded mode (SQLite) for zero-dependency local dev.

**Quick start (no Docker needed):**

```bash
git clone https://github.com/Colin4k1024/Aetheris.git
cd Aetheris && make run-embedded
curl http://localhost:8080/api/health
```

Python SDK available: `pip install aetheris`.

Repo has examples for ReAct agents, multi-agent collaboration, and human approval flows.

Still early. Known limitations: Python/JS SDKs are minimal, the UI is CLI-only, and the eino dependency means partial coupling to ByteDance's agent primitives (working on making this pluggable).

Feedback wanted: (1) does the at-most-once guarantee matter to you in practice? (2) what integrations first -- LangChain, Temporal? (3) is the embedded-mode experience actually zero-friction?

GitHub: https://github.com/Colin4k1024/Aetheris

---

## Comment Prep

### "How is this different from Temporal?"

Temporal is a general-purpose workflow engine requiring a separate server and SDK. Aetheris is purpose-built for AI agents:
- Built-in LLM/tool effect capture (Temporal doesn't know what an LLM call is)
- Built-in human-in-the-loop state machine (StatusParked -> resume)
- YAML-driven agent definitions (no workflow code needed)
- Embedded mode with zero external services

### "How is this different from LangGraph?"

LangGraph is a great graph orchestration framework but is stateless by default:
- No InvocationLedger (the same tool call gets re-executed on retry)
- No cross-process persistence (LangGraph Cloud has it, but it's paid)
- No Worker lease system (can't scale to multiple workers)

### "Why Go instead of Python?"

The agent runtime is infrastructure, and Go fits better here:
- Concurrent worker scheduling: goroutines are more controllable than asyncio
- Predictable memory usage (LLM calls are already heavy enough)
- Simple deployment: single binary, no venv/dependency hell
- Type safety catches silent errors in replay paths

Python/JS SDKs (wrapping the REST API) are available -- you don't write Go to use Aetheris.

### "How big is the eino lock-in risk?"

Fair concern. Current state:
- eino handles the agent reasoning loop and tool call orchestration
- The persistence layer (JobStore/EventStore/Ledger) is fully independent of eino
- We're abstracting the Agent interface so LangChain4Go and other frameworks can plug into the runtime and only use Aetheris for persistence + scheduling

Short-term: not fully decoupled. But eino is Apache 2.0, used at scale inside ByteDance -- should have enough maintenance momentum.

### "What's your test coverage?"

The core runtime has unit tests and integration tests. Run `make test` to verify. The test suite covers crash recovery paths, lease fencing correctness, event-sourced replay, and at-most-once ledger behavior. We also have Go benchmarks for the JobStore and a k6 load test suite for concurrent job execution.

### "Who's using this in production?"

Currently in internal use and POC stage. We're actively looking for early adopters who have agents touching external APIs (payments, email, databases) and need crash recovery guarantees. If that's you, we'd love to hear about your use case.
