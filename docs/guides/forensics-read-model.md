# Forensics Read Model

The forensics query surface is an experimental API backed by a defined read model. This page documents the ownership, schema, filter compatibility, and current performance boundary.

## Scope

Applies to:

- `POST /api/forensics/query`
- `POST /api/forensics/batch-export`
- `GET /api/forensics/export-status/:task_id`
- `GET /api/forensics/consistency/:job_id`
- `GET /api/jobs/:id/evidence-graph`
- `GET /api/jobs/:id/audit-log`

These endpoints are exposed only when `api.forensics.experimental=true`.

## Read Model Ownership

| Field | Owner | Source |
|---|---|---|
| `job_id` | Job store | `job.Job.ID` |
| `agent_id` | Job store | `job.Job.AgentID` |
| `tenant_id` | Job store | `job.Job.TenantID` |
| `created_at` | Job store | `job.Job.CreatedAt` |
| `status` | Job store | `job.Job.Status` |
| `event_count` | Event store | `jobstore.ListEvents(job_id)` length |
| `tool_calls` | Event store | `tool_invocation_finished.payload.tool_name` |
| `key_events` | Event store | selected evidence event types |

The job store owns tenant scoping. The event store owns evidence details. The query handler combines them only after narrowing candidates by `agent_filter` and tenant.

## Required Query Shape

Current implementation requires `agent_filter`. This keeps the first read model bounded and avoids an unindexed cross-tenant scan.

```json
{
  "agent_filter": ["agent_123"],
  "tenant_id": "tenant_a",
  "limit": 20,
  "offset": 0
}
```

If `tenant_id` is omitted, the authenticated tenant from request context is used. If neither is present, `default` is used.

## Filter Compatibility

| Filter | Compatibility rule |
|---|---|
| `tenant_id` | Exact tenant scope. Must never return jobs from another tenant. |
| `agent_filter` | Required. Exact match against `agent_id`. |
| `status_filter` | Case-insensitive match against job status string. |
| `tool_filter` | Exact match or trailing `*` prefix match, for example `stripe*`. |
| `event_filter` | Exact event type match. |
| `time_range.start` | Includes jobs created at or after the start time. |
| `time_range.end` | Includes jobs created at or before the end time. |

New filters must be additive and optional while the surface remains experimental.

## Pagination Compatibility

- `limit <= 0` defaults to `20`.
- `limit > 200` is capped to `200`.
- `offset < 0` is treated as `0`.
- `page` is `offset / limit` after limit normalization.
- Results are sorted by `created_at` descending.

## Performance Boundary

Current implementation is bounded by:

- candidate jobs returned by `agent_filter` + tenant
- one event-store read per candidate job
- max response page size of 200

This is acceptable for experimental query workflows and release drills. Before removing the experimental gate, add an indexed read model or materialized summary for high-cardinality tenants and define a latency SLO.

## Release Gate

Run:

```bash
./scripts/release-forensics-read-model-drill.sh
```

The drill covers tenant isolation, pagination cap, large event stream handling, experimental route gating, and batch export status flow.
