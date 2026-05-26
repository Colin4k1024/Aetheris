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

## Non-Goals

- Do not use advisor output to mutate a job after replay starts.
- Do not hide candidate scores or reason codes.
- Do not make WisePick a required runtime dependency.
- Do not promote this contract beyond experimental without tests, config,
  API/CLI docs, release drill, and ops evidence.
