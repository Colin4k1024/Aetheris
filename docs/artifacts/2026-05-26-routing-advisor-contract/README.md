---
artifact: routing-advisor-contract
date: 2026-05-26
role: architect
status: draft-for-review
source_issue: https://github.com/Colin4k1024/Aetheris/issues/197
---

# RoutingAdvisor Contract

This artifact defines the experimental `RoutingAdvisor` contract requested by
issue #197. The goal is to support deterministic capability routing without
weakening Aetheris replay, evidence, tenant isolation, and at-most-once
guarantees.

## Decision

Define `RoutingAdvisor` as an evidence-first experimental boundary.

The advisor may select a capability, tool, agent, or adapter during original
execution, but replay must reuse recorded route evidence and must not call an
external advisor again.

## Artifacts

- [arch-design](./arch-design.md)
- [api-contract](./api-contract.md)
- [test-plan](./test-plan.md)
- [execute-log](./execute-log.md)

## Scope

In scope:

- Experimental Go interface contract.
- Canonical routing evidence schema.
- Replay rule: recorded route evidence is immutable.
- Failure policy: `fail_open`, `fail_closed`, or `cached_decision`.
- WisePick can be mapped as one adapter, but is not a hard dependency.

Out of scope:

- Stable public HTTP API.
- Stable CLI surface.
- Default runtime enablement.
- Autonomous enforcement based only on advisor output.
- Replacing existing `pkg/routing` model-tier router.
