# PRD: External HTTP Agent Intake

## Goal

Enable teams with existing HTTP-based agents to run them through Aetheris with minimal migration, gaining durable submission, event traces, timeouts, retries, and audit visibility.

## Success Criteria

- A configured `external_http` agent can be addressed by `POST /api/agents/:id/message`.
- Job creation writes `PlanGenerated` with one `external_agent_call` tool node.
- The external agent receives message, session ID, job metadata, and idempotency headers.
- `job_completed` includes the external response answer when execution succeeds.
- Documentation clearly states that black-box mode does not guarantee at-most-once for side effects inside the external agent.

## Out Of Scope

- CLI script agents.
- Streaming partial responses from external agents.
- Automatic decomposition of external agent internals into TaskGraph nodes.
