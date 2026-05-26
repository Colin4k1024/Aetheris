# Runtime Cost Model

This page gives an engineering estimate of the storage and write amplification created by Aetheris durable execution. Exact byte size depends on payload size, metadata, tracing detail, and database encoding.

## Baseline Event Counts

| Step type | Typical records | Notes |
|---|---:|---|
| Job submission | 2-3 events | `job_created`, `plan_generated`, optional `job_queued` |
| Pure/LLM node | 3-5 events | `node_started`, `command_emitted`, `command_committed`, `node_finished`, optional `step_committed` |
| Runtime Tool node | 5-8 events | node events plus `tool_invocation_started`, `tool_invocation_finished`, command commit, summaries |
| Wait / HITL node | 2-4 events before resume | `node_started`, `job_waiting` / `job_parked`; later `wait_completed` / `job_resumed` |
| Job completion | 1-2 events | `job_completed`, optional checkpoint/session updates |

## Write Amplification by Store

| Store | When written | Purpose |
|---|---|---|
| Job/Event Store | Every lifecycle transition and execution event | Replay authority, lease state, audit base |
| Checkpoint Store | Step or checkpoint boundary | Resume cursor/state without replaying long histories |
| ToolInvocationStore | Runtime Tool acquire/commit | Invocation Ledger arbitration |
| Effect Store | Completed Tool/LLM/HTTP effects | Strong Replay catch-up and effect metadata |
| Trace/Audit derived views | Optional or API-derived | UI, export, forensics |

## Example: 10-Step Agent Job

Assume:

- 3 LLM nodes
- 5 Runtime Tool nodes
- 1 wait/resume node
- 1 pure/workflow node

Approximate writes:

| Category | Estimate |
|---|---:|
| Initial job/plan events | 2-3 |
| LLM node events | 9-15 |
| Tool node events | 25-40 |
| Wait/resume events | 3-5 |
| Pure/workflow node events | 3-5 |
| Completion events | 1-2 |
| Checkpoints | up to 10 |
| Effect Store records | about 8 (LLM + Tool effects) |
| ToolInvocationStore records | about 5 |

Expected total: roughly 45-70 event-store writes plus checkpoint/effect/ledger writes. Large LLM outputs or tool payloads dominate storage size more than event count.

## Cost Controls

| Control | Effect |
|---|---|
| Snapshotting | Reduces replay time for long event histories |
| Runtime GC | Removes expired checkpoints/tool invocation records according to retention policy |
| Event taxonomy | Keeps replay-critical history separate from trace/audit expansion |
| Payload discipline | Store summaries in trace events and full payload only where replay/audit requires it |
| External HTTP migration | Start with one outer call, then extract only high-risk side effects into Runtime Tools |

## Sizing Questions

Before production rollout, answer:

- Average and P95 steps per Job?
- Ratio of LLM nodes to Tool nodes?
- Average LLM/tool payload size?
- Retention requirement for event history, effects, checkpoints, and audit exports?
- Expected concurrent waiting Jobs?
- Required replay latency for incident response?

## Related Docs

- [Job lifecycle](job-lifecycle.md)
- [Event taxonomy](../../design/internal/event-taxonomy.md)
- [Production runtime gates](production-runtime-gates.md)
- [Runtime guarantees](runtime-guarantees.md)
