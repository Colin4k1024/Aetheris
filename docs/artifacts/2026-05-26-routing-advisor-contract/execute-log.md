---
artifact: execute-log
task: routing-advisor-contract
date: 2026-05-26
role: tech-lead
status: completed-design-draft
---

# Execute Log

## Inputs

- GitHub issue #197: WisePick deterministic routing proposal.
- Maintainer triage: P2 exploration, needs design, experimental scope.
- WisePick response: adapter maps route decisions into Aetheris-like evidence
  and agrees replay should reuse serialized route evidence.
- Local code review: existing `pkg/routing` is model-tier routing, not the same
  boundary as capability routing.

## Decisions

| Decision | Rationale |
|---|---|
| Keep `RoutingAdvisor` experimental | The contract lacks runtime tests, config, and release evidence. |
| Define evidence schema before adapter code | Replay and audit boundaries need a canonical payload first. |
| Do not bind to WisePick | WisePick can be the first adapter, but Aetheris should own the contract. |
| Replay must bypass advisor calls | Fresh advisor calls during replay would break determinism. |
| No stable HTTP/CLI surface yet | The first surface should be internal plus event evidence. |

## Implementation Follow-Up

1. Add Go structs for request, candidate, decision, outcome, and fallback policy.
2. Add canonical hashing helper.
3. Add `route_decision_recorded` event type.
4. Add a no-op/local deterministic advisor.
5. Add WisePick adapter behind config.
6. Add replay tests proving no outbound advisor call.

## Verification

Design-only changes require:

```bash
git diff --check -- . ':!AGENTS.md'
```

Code implementation follow-up must additionally run:

```bash
go test ./pkg/routing ./internal/agent/... ./internal/api/http
```
