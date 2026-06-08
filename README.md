# Aetheris

**The reliability layer your AI agents are missing.**

Your agent is processing 1,000 customer records. It reaches record 847 — and the process dies.

**Without Aetheris:** start over from record 1. Re-run 847 LLM calls. Pay twice. Pray nothing was written twice.

**With Aetheris:** restart. It resumes from recorded progress. Runtime Tool side effects are not repeated when the production Ledger and Effect Store are configured.

---

## The problem with AI agents in production

Every production AI agent eventually hits the same three walls:

| Failure mode | What happens today |
|---|---|
| Process crash mid-task | Restart from the beginning; re-run all LLM calls |
| Retry after tool failure | Email sent twice, order created twice, payment charged twice |
| "Why did the AI do that?" | No visibility, no audit trail, no replay |

Aetheris is an open-source runtime that solves all three — without requiring you to rewrite your agent.

---

## Quickstart — no Docker required

**Requirements:** Go 1.26.1+, Git

```bash
git clone https://github.com/Colin4k1024/Aetheris.git
cd Aetheris
make run-embedded        # starts with embedded SQLite, no external services
```

```bash
curl http://localhost:8080/api/health   # {"status":"ok", ...}
```

**From Python** (`pip install aetheris`):

```python
from aetheris import AetherisClient

client = AetherisClient("http://localhost:8080")
job = client.run("my-agent", "Summarize the Q3 earnings report")
result = job.wait()
print(result.output)
```

**From any language** — Aetheris exposes a REST API. Wrap your existing agent with two config lines:

```yaml
# configs/api.embedded.yaml
agents:
  agents:
    my_python_agent:
      type: "external_http"
      external:
        url: "http://localhost:9000/invoke"
        timeout: "120s"
```

Then submit a job:

```bash
curl -X POST http://localhost:8080/api/agents/my_python_agent/message \
  -H "Idempotency-Key: task-001" \
  -H "Content-Type: application/json" \
  -d '{"message": "Process customer batch #42"}'
```

→ [Full quickstart guide](docs/guides/quickstart.md)

---

## Core guarantees

### 1. Crash recovery
Every job step is checkpointed. If the worker dies, the next worker picks up from the last checkpoint — not the beginning.

```
Job progress:  ████████████████████░░░░░░░░░░  (step 16/25)
Worker crash!  💀
Restart:       ████████████████████            (resumes at step 16)
```

### 2. At-most-once Runtime Tool boundary
External API calls implemented as Aetheris Runtime Tools (payments, emails, order creation) are wrapped in an invocation ledger and Effect Store. If a step is retried after a recorded commit, the runtime injects the recorded result instead of repeating the side effect.

```python
# Without Aetheris:  retry → email sent twice
# With Aetheris Runtime Tool: retry → ledger returns recorded result, email send is not repeated
```

### 3. Full decision audit trail
Every LLM call, tool invocation, and checkpoint is appended to an immutable event log. You can replay any job from any point — without re-calling LLMs or external APIs.

```bash
aetheris trace <job-id>    # view the full decision timeline
aetheris replay <job-id>   # replay without side effects
```

Guarantees are configuration-dependent. See the [guarantee matrix](docs/guides/guarantee-matrix.md) for the exact boundary between embedded mode, `external_http`, native Runtime Tools, and production Postgres deployments.

---

## Connect your existing agent

Aetheris works with **any agent, in any language**. You don't need to change your agent code.

For split API/Worker deployments, load the same `external_http` agent definition into both processes so the API can accept `/api/agents/:id/message` and the Worker can execute the job.

### Python (LangChain / any agent)

```python
# Your existing LangChain agent — unchanged
from langchain_openai import ChatOpenAI
from langchain.agents import create_react_agent

agent = create_react_agent(ChatOpenAI(), tools, prompt)

# Expose it as an HTTP endpoint (one function)
from aetheris.integrations.langchain import serve
serve(agent, port=9000)   # Aetheris will call this endpoint durably
```

```yaml
agents:
  agents:
    my_langchain_agent:
      type: "langchain"
      external:
        url: "http://localhost:9000/invoke"
```

When you need Aetheris to own LangChain/LangGraph internal steps, switch to embedded mode and expose an explicit manifest from the Python adapter:

```yaml
agents:
  agents:
    my_langchain_agent:
      type: "langchain"
      external:
        mode: "embedded"
        url: "http://localhost:9000"
        manifest_path: "./configs/framework-agents/my_langchain_agent.manifest.json"
```

→ [Full LangChain integration guide](docs/adapters/langchain.md)

### Any HTTP service

```yaml
# Add to configs/api.embedded.yaml
agents:
  agents:
    my_agent:
      type: "external_http"
      external:
        url: "http://your-agent:9000/invoke"
```

Your agent receives a job envelope with `message`, `job_id`, and `idempotency_key`. It returns `{"answer": "...", "final": true}`.

→ [External HTTP adapter docs](docs/adapters/external-http-agent.md)

### Go (Eino / native)

```go
// Built-in via AgentFactory — config-driven
// configs/agents.yaml
agents:
  my_eino_agent:
    type: "react"
    llm: "default"
    tools: ["web_search", "calculator"]
```

→ [Eino integration guide](docs/adapters/eino-examples.md)

---

## How it works

```
Your Agent (Python/JS/Go/any)
        │
        ▼
  Aetheris API ──── idempotency key ──▶ Invocation Ledger
        │                                    (Runtime Tool boundary)
        ▼
  Durable Worker ──── checkpoint ──────▶ Event Store
        │                                    (crash recovery)
        ▼
  Trace & Replay API ───────────────────────────────▶ Audit
```

The runtime is event-sourced: every state transition is an append-only event. With the production Ledger and Effect Store configured, replay injects recorded LLM and Runtime Tool results instead of re-calling them.

---

## vs. LangGraph Platform / Temporal / vanilla frameworks

| | Aetheris | LangGraph Platform | Temporal |
|---|---|---|---|
| Open source + self-hosted | ✅ | ❌ (cloud only) | ✅ |
| No infrastructure for local dev | ✅ (embedded SQLite) | ❌ | ❌ (requires server) |
| At-most-once Runtime Tool boundary | ✅ with Ledger + Effect Store | ⚠️ manual | ⚠️ activity/idempotency dependent |
| Works with any agent framework | ✅ | ❌ LangGraph only | ❌ requires SDK |
| LLM decision audit trail | ✅ | ✅ | ❌ |
| Replay without re-calling recorded LLM/Tool effects | ✅ with Effect Store | ⚠️ checkpoint-dependent | ✅ for recorded workflow history |

---

## Explore the external_http batch demo

See the current black-box adapter boundary in 2 minutes:

```bash
cd examples/crash_recovery
pip install aetheris
python demo.py
# Starts a local external_http demo agent and submits one durable batch job
```

The example shows durable submission and trace visibility around one external HTTP call. For true per-step checkpoint resume inside the work itself, use native Aetheris tools/workflows instead of a single `external_http` call.

→ [External HTTP batch demo](examples/crash_recovery/)

---

## Repository map

| Path | Purpose |
|---|---|
| [cmd/api](cmd/api) | HTTP API service |
| [cmd/worker](cmd/worker) | Background job worker |
| [cmd/cli](cmd/cli) | CLI: `aetheris trace/replay/jobs/chat` |
| [configs](configs) | Runtime configs (embedded, Docker, production) |
| [examples](examples) | Working examples for each integration pattern |
| [sdk/python](sdk/python) | Python SDK (`pip install aetheris`) |
| [docs](docs) | Guides, API reference, design notes |
| [internal/agent](internal/agent) | Core runtime engine |

---

## Documentation

| Goal | Link |
|---|---|
| Get started in 5 minutes | [docs/guides/quickstart.md](docs/guides/quickstart.md) |
| Connect an existing HTTP agent | [docs/adapters/external-http-agent.md](docs/adapters/external-http-agent.md) |
| Migrate external side effects safely | [docs/adapters/external-http-migration.md](docs/adapters/external-http-migration.md) |
| Connect a LangChain agent | [docs/adapters/langchain.md](docs/adapters/langchain.md) |
| Understand crash recovery | [docs/guides/runtime-guarantees.md](docs/guides/runtime-guarantees.md) |
| Compare guarantee boundaries | [docs/guides/guarantee-matrix.md](docs/guides/guarantee-matrix.md) |
| Follow the Job lifecycle | [docs/guides/job-lifecycle.md](docs/guides/job-lifecycle.md) |
| Check production gates | [docs/guides/production-runtime-gates.md](docs/guides/production-runtime-gates.md) |
| Estimate runtime cost | [docs/guides/runtime-cost-model.md](docs/guides/runtime-cost-model.md) |
| Sign evidence packages | [docs/guides/evidence-signing.md](docs/guides/evidence-signing.md) |
| Understand forensics query read model | [docs/guides/forensics-read-model.md](docs/guides/forensics-read-model.md) |
| Harden RBAC, redaction, and retention | [docs/guides/rbac-redaction-retention-hardening.md](docs/guides/rbac-redaction-retention-hardening.md) |
| Generate evidence-bound compliance reports | [docs/guides/compliance-reporting.md](docs/guides/compliance-reporting.md) |
| Evaluate AI forensics detection | [docs/guides/ai-forensics-eval.md](docs/guides/ai-forensics-eval.md) |
| Assess distributed verifier readiness | [docs/guides/distributed-verifier-readiness.md](docs/guides/distributed-verifier-readiness.md) |
| Interpret monitoring quality scores | [docs/guides/monitoring-quality-scorer.md](docs/guides/monitoring-quality-scorer.md) |
| Promote prototype capabilities | [docs/artifacts/2026-05-25-architecture-review/prototype-promotion-backlog.md](docs/artifacts/2026-05-25-architecture-review/prototype-promotion-backlog.md) |
| Deploy to production (Docker) | [docs/guides/deployment.md](docs/guides/deployment.md) |
| API reference | [docs/reference/api.md](docs/reference/api.md) |

---

## License

Apache 2.0 — free to use, self-host, and modify.
