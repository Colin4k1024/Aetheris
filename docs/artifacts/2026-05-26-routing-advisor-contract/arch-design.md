---
artifact: arch-design
task: routing-advisor-contract
date: 2026-05-26
role: architect
status: draft-for-review
---

# RoutingAdvisor Architecture

## First Principles

Aetheris can accept external routing advice only if the advice becomes durable
runtime evidence. A recovered or replayed job must not make a fresh outbound
routing call, because that would change the execution path and break replay
determinism.

## Current Code Boundary

`pkg/routing` currently routes model choices by tier, cost, latency, quality,
and failover. It is useful, but it is not the same boundary as issue #197.

`RoutingAdvisor` is a higher-level capability selector:

- model router: chooses an LLM/model
- routing advisor: chooses a capability, tool, agent, or adapter path

The two may later compose, but this design keeps them separate.

## Runtime Placement

```text
JobCreated
  -> PlanGenerated
  -> RoutingAdvisor.Decide
  -> route_decision_recorded evidence event
  -> Tool / Agent / Adapter execution
  -> routing feedback recorded after terminal outcome
```

During replay:

```text
Job replay
  -> read route_decision_recorded
  -> select recorded capability
  -> never call external advisor
```

## Proposed Internal Interface

```go
type RoutingAdvisor interface {
    Decide(ctx context.Context, req RouteDecisionRequest) (RouteDecision, error)
    RecordOutcome(ctx context.Context, outcome RouteOutcome) error
}
```

The interface is intentionally small. It does not own tool execution, job
scheduling, retries, ledger acquisition, or evidence export.

## Integration Points

| Integration point | Contract |
|---|---|
| Planner | May provide candidate capabilities and constraints. |
| Job event store | Must persist the route decision before executing the selected capability. |
| Tool bridge / external agent bridge | Receives only the selected target. |
| Replay | Reads recorded decision and bypasses advisor calls. |
| Evidence export | Includes routing evidence in the job event chain. |
| Monitoring | Counts advisor decisions, fallback policy, and advisor errors. |

## Event Model

Add one experimental event type in a later implementation:

```text
route_decision_recorded
```

Payload must use the canonical schema in [api-contract](./api-contract.md).

This event is replay-relevant. It must be included in evidence exports and hash
chain validation.

## Failure Policy

The advisor must declare one of:

- `fail_open`: use local deterministic fallback when advisor fails.
- `fail_closed`: fail the job before executing an unadvised route.
- `cached_decision`: reuse a previously recorded decision with matching
  decision key.

Default policy for an external adapter should be `fail_open` only when the local
fallback is deterministic and evidence-recorded. Otherwise use `fail_closed`.

## WisePick Adapter Boundary

WisePick can be the first external adapter if it maps into the Aetheris schema:

```text
WisePick response
  -> adapter normalization
  -> Aetheris RouteDecision evidence
  -> persisted route_decision_recorded event
```

The adapter must not require changes to Aetheris replay semantics.

## Replay Invariants and Deterministic Serialization Rules

These invariants are required for correct durable execution. Violating any
of them breaks replay determinism and audit trail integrity.

### Invariant 1 — Advisor.Decide is called exactly once per decision point

During original execution, `RoutingAdvisor.Decide` must be called exactly
once for a given `decision_key`. The runtime must not retry or fan-out to
multiple advisors for the same decision.

### Invariant 2 — Evidence persisted before capability executes

The `route_decision_recorded` event must be written to the JobStore
**before** the selected capability (tool, agent, adapter) begins execution.
This ensures that a crash between routing and execution produces a recoverable
state: the runtime retries execution using the persisted decision.

### Invariant 3 — Replay never calls the advisor

During replay, the runtime must read the persisted `route_decision_recorded`
event and **must not** call `Decide` on any advisor, local or remote. This
guarantees that replay is deterministic regardless of advisor availability.

```text
Replay invariant:
  for all (job, node, step) where route_decision_recorded exists:
    advisor.Decide must NOT be called
    execution must use the recorded RouteDecision
```

### Invariant 4 — Hash verification is mandatory on replay

Before using a recovered `RouteDecision`, the runtime must call
`VerifyDecisionHash`. If verification fails, the runtime must not proceed
with that decision. The job must surface `routing_decision_hash_mismatch`.

This detects storage corruption and prevents tampered evidence from
influencing execution paths.

### Invariant 5 — Missing evidence is a hard failure on replay

If the runtime enters replay for a step that should have a
`route_decision_recorded` event but none is found in the event log, this
is a hard failure. The job must surface `routing_decision_missing`.

Re-running the advisor to recover is not permitted: it would produce a
different decision with a different `decision_id` and `decision_hash`,
breaking audit trail continuity.

### Deterministic Serialization Rules

The canonical hash (`decision_hash`) is computed over the route decision
using these deterministic serialization rules:

1. **Exclude `decision_hash` itself** — the hash covers all other fields.
2. **Sort candidates by `capability_id`** alphabetically (ascending).
3. **Sort `reason_codes` within each candidate** alphabetically (ascending).
4. **Marshal using `encoding/json`** with no custom encoder.
5. **Timestamps** must be stored and serialized as `time.Time` (Go); the
   canonical form is RFC3339 with nanosecond precision when non-zero,
   otherwise `"Z"` suffix for UTC.
6. **Nil `metadata` maps** serialize as `null`, not `{}`.
7. **Empty `reason_codes`** serialize as `[]`, not `null`.
8. **All string fields** are byte-for-byte: no normalization is applied.

Changing any of these rules constitutes a breaking change to the hash
contract. Any such change requires a schema version bump and a migration
path for persisted decisions.

### Evidence Lifecycle

```text
Original execution:
  advisor.Decide(req) → RouteDecision
  HashRouteDecision(d) → d.DecisionHash
  jobstore.Append(route_decision_recorded, d)     ← MUST precede capability execution
  capability.Execute(d.Selected.CapabilityID)

Replay:
  jobstore.FindByDecisionKey(req.DecisionKey) → d
  VerifyDecisionHash(d)                           ← MUST pass; fail = hard error
  capability.Execute(d.Selected.CapabilityID)     ← same capability as original
```

### Recovery Validation Suite

The recovery invariants above are validated by the test suite in
`pkg/routing/recovery_validation_test.go`. The nine-step proof covers:

| Step | Assertion |
|------|-----------|
| 1    | `advisor.Decide` called exactly once during original execution |
| 2    | `route_decision_recorded` event persisted with valid hash |
| 3–4  | Evidence survives simulated worker crash (event-sourced durability) |
| 5    | Replay performs zero advisor calls |
| 6    | Replay returns the same `decision_id` as original |
| 7    | `decision_hash` is immutable across replay |
| 8    | Route evidence JSON is byte-identical between original and replay |
| 9    | Same capability is executed on replay (execution trace unchanged) |

Fixture-driven tests in `pkg/routing/fixture_test.go` validate the canonical
JSON schemas in `pkg/routing/testdata/routing_advisor/`.

## Non-Goals

- Do not use advisor output to mutate a job after replay starts.
- Do not hide candidate scores or reason codes.
- Do not make WisePick a required runtime dependency.
- Do not promote this contract beyond experimental without tests, config,
  API/CLI docs, release drill, and ops evidence.
