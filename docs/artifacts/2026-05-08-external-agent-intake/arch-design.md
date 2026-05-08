# Architecture: External HTTP Agent Intake

## Flow

`/api/agents/:id/message` creates a Job for a configured `external_http` agent. During job creation, Aetheris writes a `PlanGenerated` event containing a single `tool` node named `external_agent_call`. The existing Runner executes that node through `ToolNodeAdapter`, so Ledger, EffectStore, command events, and trace events remain on the normal path.

## HTTP Contract

The tool sends `message`, `session_id`, and metadata containing `agent_id`, `job_id`, and `idempotency_key`. It forwards the same idempotency key in the `Idempotency-Key` header, plus `X-Aetheris-Job-ID` and `X-Aetheris-Agent-ID`.

## Reliability Boundary

The Runtime controls the outer `external_agent_call` invocation. Internal side effects performed by the external service are opaque. Strong at-most-once semantics require migrating those side effects into Aetheris Runtime Tools.
