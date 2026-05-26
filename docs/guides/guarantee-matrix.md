# Aetheris Guarantee Matrix

This page is the single short-form reference for what Aetheris guarantees in each runtime mode. Use it before making reliability, audit, or production-readiness claims.

For formal failure semantics, see [runtime guarantees](runtime-guarantees.md) and [execution guarantees](../../design/execution-guarantees.md).
For production startup requirements, see [production runtime gates](production-runtime-gates.md).

## Mode Summary

| Runtime mode | What Aetheris controls | Durable resume | LLM replay without re-call | Runtime Tool side-effect boundary | External agent internal side effects |
|---|---|---:|---:|---:|---:|
| Embedded / memory dev | In-process Job, checkpoint, and trace | No, process-local only | No production guarantee | No cross-process guarantee | Not controlled |
| `external_http` black-box agent | Job, outer `external_agent_call`, timeout, trace, idempotency key forwarding | Yes, for the outer Aetheris Job when durable stores are configured | Only for Aetheris-managed nodes | Only for the outer Runtime Tool call | Not controlled unless the external service deduplicates with the forwarded key |
| Native Runtime Tool | Job, plan, step, tool invocation, ledger, effect record, trace | Yes, with shared durable stores | Yes, with Effect Store | Strong at-most-once boundary with Invocation Ledger + Effect Store | Not applicable |
| Production Postgres multi-worker | API control plane, Worker claim/lease, event stream, checkpoints, ledger/effects | Yes | Yes, when Effect Store is enabled | Strongest supported mode | Still not controlled inside black-box external agents |

## Required Configuration by Claim

| Claim | Minimum required configuration | Notes |
|---|---|---|
| Job survives process crash | Shared durable JobStore/Event Store, normally Postgres | Memory mode is for local development only. |
| Worker failover is safe | Postgres JobStore/Event Store with lease and `attempt_id` validation | Stale workers must be unable to append execution events. |
| Runtime Tool is not re-executed after committed | Invocation Ledger + shared ToolInvocationStore | Tools must use stable idempotency keys. |
| Crash after effect but before event does not duplicate side effect | Effect Store + catch-up path | Effect Store must be shared in multi-worker deployments. |
| Replay does not call LLM | Effect Store for LLM effects + event stream `command_committed` records | Without Effect Store this is not a production guarantee. |
| Signals eventually wake jobs | Durable wait events + WakeupQueue for multi-worker production | Without WakeupQueue, polling may still work but has delay/scale limits. |
| External HTTP agent internal payment/email/write is not duplicated | External agent must deduplicate using `Idempotency-Key`, or move the side effect into a Runtime Tool | Aetheris only controls the outer `external_agent_call`. |

## Public Wording Rules

Use these terms precisely:

- Good: "Aetheris resumes durable jobs from recorded progress when configured with shared durable stores."
- Good: "Runtime Tools get an at-most-once side-effect boundary with Invocation Ledger and Effect Store."
- Good: "`external_http` is a Level 1 migration path; internal side effects remain the external service's responsibility."
- Avoid: "All agents have zero duplicates."
- Avoid: "Exactly-once execution" without naming the required store, ledger, effect, and idempotency conditions.
- Avoid: "Replay never calls APIs" for black-box external agents or configurations without Effect Store.

## Migration Levels

| Level | Pattern | Reliability boundary |
|---|---|---|
| Level 0 | Embedded local demo | Fast onboarding, no production durability claim |
| Level 1 | `external_http` black-box agent | Durable outer job and trace; external internals opaque |
| Level 2 | External agent forwards Aetheris idempotency keys to its own side effects | Duplicate risk reduced if downstream systems honor keys |
| Level 3 | High-risk side effects migrated into Aetheris Runtime Tools | Full Aetheris Ledger/Effect Store boundary |

See [external HTTP migration](../adapters/external-http-migration.md) for the step-by-step path.
