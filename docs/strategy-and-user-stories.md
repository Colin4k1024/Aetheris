# Aetheris Strategy and User Stories

This document aligns Aetheris product narrative with the B2D transformation goal:
from a cloud-hosted SaaS experience to an enterprise AI agent runtime foundation.

## Strategic Positioning

- Aetheris is positioned as **B2D infrastructure** for enterprise AI agents.
- Primary users are enterprise developers and architects suffering from runtime fragility and audit pressure in production.
- Product focus is durable execution, deterministic recovery, compliance-grade traceability, and local-first deployment.

## Three Core User Stories

### 1) Long Lifecycle and Human-in-the-Loop

As an enterprise application developer, I need to build workflows such as travel planning or large refund approvals.  
I need the agent to pause at critical nodes, release compute resources, and wait for human approval.  
After approval, execution must resume losslessly from the breakpoint without re-spending previous LLM tokens.

**Root problem**: Most agent frameworks treat execution as a single in-memory call stack. Pausing for days means losing all intermediate state. Re-running consumes LLM tokens, re-triggers upstream API calls, and may produce different results.

**Aetheris solution**: `StatusParked` persists the full execution graph to durable storage. The `wait_signal` node blocks on a named signal. When `approve` arrives via API or CLI, the scheduler re-leases the Job and the Runner resumes from the exact checkpoint — no data re-fetched, no LLM re-called, no state inconsistency.

**Acceptance criteria**:
- Agent can pause mid-DAG for arbitrarily long durations (hours, days)
- Resuming does not re-invoke completed steps
- All intermediate state (query results, LLM decisions) is preserved and injected on resume
- Evidence of approval decision is recorded in the event stream

---

### 2) Financial and Legal Compliance

As a financial compliance/risk engineer, I need an AI agent for diff analysis and anomaly detection.  
Every API call and reasoning decision must be recorded in an immutable log so we can fully reconstruct and explain why the AI acted in a certain way during audits.

**Root problem**: Agent "logs" are typically unstructured stdout lines that cannot be replayed, correlated, or cryptographically attested. When regulators ask "why did the AI recommend product X?", there is no structured answer.

**Aetheris solution**: Every step transition (`StepStarted`, `StepCompleted`, `ToolCalled`, `ToolResult`, `LLMGenerated`) is appended to an event-sourced Job Store. The Evidence Package API exports a signed, structured proof package with full causal chain from input to output. Deterministic replay re-runs the Agent against the recorded event stream without touching live APIs, producing an identical decision trace.

**Acceptance criteria**:
- Complete event log from job creation to completion, append-only, never modified
- Replay produces same outputs as original run (no LLM/Tool re-calls)
- Evidence package exportable in auditor-readable format
- Timestamps and attempt IDs on every event for forensic attribution

---

### 3) Data Privacy and Local Deployment

As an enterprise architect focused on data sovereignty, I need a lightweight agent engine that runs in private networks or air-gapped environments.  
Core business context must stay local, avoiding continuous upload to external hosted platforms.

**Root problem**: Cloud-first agent runtimes send execution traces, prompts, and intermediate results to vendor backends. For regulated industries (healthcare, finance, defense), this is a hard blocker.

**Aetheris solution**: Embedded mode runs with zero external dependencies — SQLite-backed event store, in-process scheduler, local effect store. No phone-home, no external telemetry. Can be deployed in K8s within a private VPC or even on an air-gapped machine.

**Acceptance criteria**:
- Fully functional with only local resources (no PostgreSQL/Redis required in embedded mode)
- No outbound network calls to vendor infrastructure
- Configuration schema supports private LLM endpoints (Ollama, private Azure endpoints)

---

### 4) Supply Chain and Multi-System Orchestration

As a procurement engineer, I need an Agent to send RFQs to 30+ suppliers, confirm responses, and create purchase orders. Any API call to a confirmed supplier must never be duplicated — duplicate POs cost real money.

**Root problem**: When orchestrating tens of external APIs, network failures and partial successes create ambiguity. Without per-call idempotency tracking, retry = duplicate order.

**Aetheris solution**: Each supplier API call uses a unique `idempotency_key` derived from `(jobID, stepID, supplierID)`. The Invocation Ledger records authorization per key. On retry, the Ledger returns "already committed" and injects the recorded result — the real supplier API is never called again.

**Acceptance criteria**:
- Each supplier API called at most once per job, regardless of retries or worker crashes
- Partial failure (some suppliers confirmed, some not) recovers from exact state without re-confirming already-confirmed suppliers
- Full audit log of which supplier was contacted when, what was requested, what was returned

---

### 5) Medical AI and High-Stakes Write Operations

As a healthcare application developer, I need an AI diagnostic assistant to analyze patient records and write structured summaries to the HIS system. Duplicate writes must be impossible — two conflicting diagnostic summaries are a patient safety risk.

**Root problem**: Idempotency at the application layer is hard when the write operation is expensive (LLM-generated) and the HIS API does not provide native deduplication. Retry-on-timeout creates duplicate records.

**Aetheris solution**: LLM generation is wrapped in the Effect Store: before the write, `Effect.Put(key, result)` persists the generated content. Then `EventStore.Append(command_committed)` finalizes. On crash between the two, the catch-up path reads from Effect Store and re-applies without re-generating. The HIS write receives a stable idempotency key, preventing duplicate records.

**Acceptance criteria**:
- LLM-generated content is persisted before being applied to HIS
- Crash between generation and write does not cause re-generation or duplicate write
- Every write attempt is traceable to a specific job, step, and idempotency key

---

### 6) Regulatory-Grade Audit for AI Decision Agents

As a risk officer in a bank, I need the AI agent's complete decision history — every data point queried, every model output, every action taken — to be available on demand for regulatory examination within 24 hours of any incident.

**Root problem**: Agent execution is ephemeral. Once the process exits, the reasoning chain is gone. Reconstruction from logs is incomplete and non-deterministic.

**Aetheris solution**: The append-only Event Store is the source of truth. Every `ToolCalled`, `ToolResult`, `LLMGenerated`, and `StepCompleted` event is immutable once written. The `aetheris trace <job_id>` CLI and `/api/jobs/:id/events` API expose the full chain. The Evidence Package can be exported as a tamper-evident audit bundle with cryptographic consistency checks.

**Acceptance criteria**:
- Full execution history retrievable for any job within seconds
- Replay produces identical reasoning trace (no drift)
- Evidence package includes input context, intermediate states, and final output with timestamps

## User Story to Capability Mapping

| User Story                         | Runtime Capability                                                                  | Key Locations                                                                                                                   |
| ---------------------------------- | ----------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------- |
| Long lifecycle + human-in-the-loop | Wait/Signal semantics, parked status, checkpoint recovery, replay injection         | `internal/agent/runtime/executor/wait_signal.go`, `internal/api/http/handler.go`, `docs/guides/runtime-guarantees.md`           |
| Financial/legal compliance         | Event sourcing, trace/proof export, auditable runtime events                        | `internal/runtime/jobstore`, `pkg/proof`, `docs/guides/runtime-guarantees.md`, `docs/roadmaps/EVIDENCE-PACKAGE-FOR-AUDITORS.md` |
| Data privacy + local deployment    | Embedded local stores, local-first mode, no external DB dependency in embedded mode | `internal/agent/job/embedded_store.go`, `docs/embedded-mode.md`                                                                 |

## Four-Phase Strategic Refactor

| Phase   | Goal                                               | Status                              | Evidence                                                                                                                                  |
| ------- | -------------------------------------------------- | ----------------------------------- | ----------------------------------------------------------------------------------------------------------------------------------------- |
| Phase 1 | Lightweight local event/state storage              | Implemented                         | Embedded JobStore + local persistence path in `internal/agent/job/embedded_store.go`; `jobstore.type=embedded` in `docs/embedded-mode.md` |
| Phase 2 | Durable execution API (interrupt/resume semantics) | Implemented                         | Pause/resume/signal behavior and waitlike execution in `internal/api/http/handler.go` and `internal/agent/runtime/executor`               |
| Phase 3 | MCP standard integration                           | Implemented                         | MCP Host and tool adapter bridge in `internal/agent/tools/mcp_host.go`                                                                    |
| Phase 4 | Open Core business model                           | Implemented (boundary and baseline) | OSS vs Enterprise split in `docs/open-core-boundary.md`; RBAC/audit baseline in docs and runtime middleware                               |

## Alignment Principles

- Keep **runtime-first** APIs (`/api/runs`, `/api/jobs`) as the canonical execution path.
- Keep legacy `/api/agents/*` only as migration facade, not as strategic center.
- Prefer local-first deployment for privacy-sensitive enterprise environments.
- Preserve replay and at-most-once guarantees as non-negotiable product identity.

## Related Documents

- `README.md`
- `design/core.md`
- `docs/open-core-boundary.md`
- `docs/embedded-mode.md`
- `docs/guides/runtime-guarantees.md`
- `docs/roadmaps/CURRENT-STATUS-AND-FOCUS.md`
