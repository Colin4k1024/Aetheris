# Test Plan: External HTTP Agent Intake

## Automated Tests

- `pkg/config`: external HTTP config parses and validates required URL, duration, and token env.
- `internal/app/api`: external tool sends the expected body and headers to an `httptest` server.
- `internal/app/api`: external tool maps non-2xx upstream responses to errors.
- `internal/app/api`: configured external agents are registered under stable IDs and planned as one `external_agent_call` tool node.

## Manual Smoke

1. Configure an `external_http` agent in `configs/agents.yaml`.
2. Start a local HTTP server implementing `/invoke`.
3. Submit `POST /api/agents/:id/message` with an `Idempotency-Key`.
4. Verify `/api/jobs/:job_id/events` contains `plan_generated`, `command_committed`, and `job_completed.answer`.
