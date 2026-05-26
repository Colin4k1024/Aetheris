# RoutingAdvisor Contract

`RoutingAdvisor` is an experimental evidence-first boundary for capability
routing.

It is designed for integrations such as WisePick, where an external or local
advisor recommends which capability, tool, agent, model, adapter, or workflow
path should execute a planned step.

## Boundary

The advisor may be called during original execution only.

Replay must reuse the recorded route decision evidence and must not call the
advisor again.

## Canonical Design

The contract is defined in:

- [RoutingAdvisor architecture](../artifacts/2026-05-26-routing-advisor-contract/arch-design.md)
- [RoutingAdvisor API contract](../artifacts/2026-05-26-routing-advisor-contract/api-contract.md)
- [RoutingAdvisor test plan](../artifacts/2026-05-26-routing-advisor-contract/test-plan.md)

## Status

Current status: design draft / experimental.

No stable HTTP API, CLI command, or default runtime integration exists yet.

## Core Rule

If a route decision cannot be recorded as durable evidence, it cannot influence
execution.
