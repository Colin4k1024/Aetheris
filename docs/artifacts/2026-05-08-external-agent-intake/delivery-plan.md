# Delivery Plan: External HTTP Agent Intake

## Implementation

- Extend agent config with `external.url`, `external.timeout`, and `external.token_env`.
- Register configured agents under stable IDs matching `configs/agents.yaml`.
- Register `external_agent_call` as a native Runtime Tool when external HTTP agents exist.
- Generate a static single-node TaskGraph for `external_http` agents at job creation.
- Extract the answer from committed tool output into the final `job_completed` payload.

## Rollout

- Start with HTTP black-box service agents only.
- Ask users to support `Idempotency-Key` for Level 1 deduplication.
- Migrate high-risk external actions into Runtime Tools for strong execution guarantees.
