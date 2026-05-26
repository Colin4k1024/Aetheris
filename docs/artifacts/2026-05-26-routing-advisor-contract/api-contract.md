---
artifact: api-contract
task: routing-advisor-contract
date: 2026-05-26
role: architect
status: draft-for-review
---

# RoutingAdvisor API Contract

## Compatibility Boundary

There is no stable public HTTP API for `RoutingAdvisor` in this design.

The first contract is internal and evidence-bound:

- Go interface for runtime integration.
- JSON payload schema for `route_decision_recorded`.
- Experimental config surface.

Any future HTTP or CLI surface must remain experimental until replay, tenant,
and evidence tests exist.

## Route Decision Request

```json
{
  "schema_version": "routing.advisor.request.v1alpha1",
  "job_id": "job_123",
  "run_id": "run_123",
  "tenant_id": "tenant_a",
  "plan_id": "plan_abc",
  "node_id": "node_retrieve",
  "step_id": "step_1",
  "decision_key": "job_123/node_retrieve/step_1",
  "goal": "summarized user goal",
  "constraints": {
    "max_latency_ms": 5000,
    "max_cost_usd": 0.02,
    "required_capabilities": ["retrieval", "summarization"],
    "risk_level": "low"
  },
  "candidates": [
    {
      "capability_id": "tool:web_search",
      "kind": "tool",
      "provider": "builtin",
      "metadata": {
        "tool_name": "web_search"
      }
    }
  ]
}
```

Required fields:

- `schema_version`
- `job_id`
- `tenant_id`
- `decision_key`
- `candidates`

## Route Candidate

```json
{
  "capability_id": "agent:conversation",
  "kind": "agent",
  "provider": "aetheris",
  "score": 0.91,
  "rank": 1,
  "reason_codes": ["lowest_expected_latency", "historical_success"],
  "metadata": {
    "agent_id": "conversation"
  }
}
```

Allowed `kind` values:

- `tool`
- `agent`
- `adapter`
- `model`
- `workflow`

## Route Decision Evidence

This is the canonical payload for `route_decision_recorded`.

```json
{
  "schema_version": "routing.advisor.decision.v1alpha1",
  "decision_id": "route_decision_01HZ",
  "decision_key": "job_123/node_retrieve/step_1",
  "decision_hash": "sha256:...",
  "job_id": "job_123",
  "run_id": "run_123",
  "tenant_id": "tenant_a",
  "plan_id": "plan_abc",
  "node_id": "node_retrieve",
  "step_id": "step_1",
  "advisor": {
    "name": "wisepick",
    "version": "0.1.9",
    "adapter": "aetheris-wisepick",
    "adapter_version": "v1alpha1"
  },
  "selected": {
    "capability_id": "tool:web_search",
    "kind": "tool",
    "provider": "builtin",
    "score": 0.91,
    "rank": 1,
    "reason_codes": ["lowest_expected_latency", "historical_success"],
    "metadata": {
      "tool_name": "web_search"
    }
  },
  "candidates": [
    {
      "capability_id": "tool:web_search",
      "kind": "tool",
      "provider": "builtin",
      "score": 0.91,
      "rank": 1,
      "reason_codes": ["lowest_expected_latency", "historical_success"]
    }
  ],
  "fallback_policy": "fail_open",
  "fallback_reason_codes": [],
  "created_at": "2026-05-26T12:00:00Z"
}
```

Required fields:

- `schema_version`
- `decision_id`
- `decision_key`
- `decision_hash`
- `job_id`
- `tenant_id`
- `advisor.name`
- `selected.capability_id`
- `selected.kind`
- `fallback_policy`
- `created_at`

Allowed `fallback_policy` values:

- `fail_open`
- `fail_closed`
- `cached_decision`

## Canonical Hash Rule

`decision_hash` must be computed over canonical JSON after removing
`decision_hash` itself.

Minimum algorithm:

```text
sha256(canonical_json(route_decision_without_decision_hash))
```

The canonicalization implementation must be deterministic across Go versions.

## Route Outcome

```json
{
  "schema_version": "routing.advisor.outcome.v1alpha1",
  "decision_id": "route_decision_01HZ",
  "job_id": "job_123",
  "tenant_id": "tenant_a",
  "selected_capability_id": "tool:web_search",
  "success": true,
  "latency_ms": 1234,
  "cost_usd": 0.0012,
  "error_class": "",
  "completed_at": "2026-05-26T12:00:05Z"
}
```

Outcome feedback is not replay-authoritative. It can improve future routing, but
cannot change the recorded route for an existing replay.

## Experimental Config

```yaml
routing_advisor:
  enabled: false
  provider: "wisepick"
  endpoint: "http://localhost:8787"
  timeout: "2s"
  fallback_policy: "fail_open"
  record_outcome: true
```

Rules:

- Default is disabled.
- Enabling requires explicit config.
- External endpoints are never called during replay.
- Config must support tenant-scoped disablement before production promotion.

## Error Codes

Future API or CLI surfaces should use these classes:

| Code | Meaning |
|---|---|
| `routing_advisor_unavailable` | Advisor could not be reached. |
| `routing_advisor_invalid_response` | Adapter could not normalize response. |
| `routing_decision_hash_mismatch` | Recorded evidence hash is invalid. |
| `routing_decision_missing` | Replay needs a recorded route decision but none exists. |
| `routing_candidate_not_allowed` | Advisor selected a capability outside the allowed candidate set. |
