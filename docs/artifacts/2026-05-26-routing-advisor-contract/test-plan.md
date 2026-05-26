---
artifact: test-plan
task: routing-advisor-contract
date: 2026-05-26
role: qa-engineer
status: draft-for-review
---

# RoutingAdvisor Test Plan

## Unit Tests

| Area | Required assertions |
|---|---|
| Schema validation | Required fields are enforced for request, decision, and outcome. |
| Candidate validation | Selected candidate must exist in the candidate set. |
| Hashing | `decision_hash` is deterministic and excludes itself from canonical input. |
| Fallback policy | Only `fail_open`, `fail_closed`, and `cached_decision` are accepted. |
| Adapter normalization | WisePick-like payload maps into Aetheris evidence without losing candidate scores or reason codes. |

## Runtime Integration Tests

1. Original execution calls the advisor once.
2. Runtime records `route_decision_recorded` before selected capability execution.
3. Replay reads `route_decision_recorded`.
4. Replay does not call the advisor.
5. Missing route evidence during replay fails with `routing_decision_missing`.
6. Hash mismatch fails with `routing_decision_hash_mismatch`.
7. Tenant A cannot read or reuse Tenant B routing evidence.

## Failure Tests

| Failure | Expected behavior |
|---|---|
| Advisor timeout + `fail_open` | Runtime uses deterministic local fallback and records fallback reason. |
| Advisor timeout + `fail_closed` | Job fails before selected capability execution. |
| Invalid advisor response | Runtime rejects decision and records no selected route. |
| Advisor selects unknown candidate | Runtime rejects decision. |
| Advisor selects disallowed tenant capability | Runtime rejects decision. |

## Evidence Export Tests

- Evidence ZIP includes `route_decision_recorded`.
- Hash chain validation covers route decision events.
- Evidence graph exposes route decision as an `llm_decision` or future
  `route_decision` evidence node.
- Forensics query can filter or summarize route decisions after the event type
  exists.

## Contract Fixtures

Create fixtures under a later implementation path:

```text
testdata/routing_advisor/request.valid.json
testdata/routing_advisor/decision.valid.json
testdata/routing_advisor/decision.hash-mismatch.json
testdata/routing_advisor/decision.unknown-candidate.json
testdata/routing_advisor/outcome.valid.json
```

## Release Evidence

Before promotion from design to integrated:

- `go test ./pkg/routing ./internal/agent/...`
- Integration test showing replay bypasses advisor calls.
- Release drill artifact recording original execution and replay paths.
- Docs updated in `docs/reference/api-contract.md` and `docs/guides/`.

## Non-Regression Rule

If advisor integration is disabled, existing runtime paths must behave exactly
as they do today.
