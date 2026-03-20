# Aetheris API Contract (v2.2)

## 1. Scope

This document defines the external API compatibility boundary for Aetheris `2.x`.

- Stable APIs: backward-compatible across `2.x` minors and patches
- Experimental APIs: may change in minor releases
- Internal packages (`internal/`) are out of compatibility scope

## 2. Versioning and Compatibility

- Major (`x.0.0`): may include breaking changes
- Minor (`0.x.0`): backward-compatible feature additions
- Patch (`0.0.x`): bug fixes and non-breaking behavior fixes

Compatibility window:

- Stable APIs are guaranteed across all `2.x`
- Deprecated stable APIs are supported for at least 2 minor versions before removal

## 3. Stable API Surface (v2.2)

### Job APIs

- `POST /api/agents/:id/message` (legacy facade)
- `GET /api/jobs/:id`
- `POST /api/jobs/:id/stop`
- `POST /api/jobs/:id/signal`
- `GET /api/jobs/:id/approval`
- `POST /api/jobs/:id/approval/approve`
- `POST /api/jobs/:id/approval/reject`
- `POST /api/jobs/:id/approval/delegate`
- `POST /api/jobs/:id/message`
- `GET /api/jobs/:id/events`
- `GET /api/jobs/:id/trace`
- `GET /api/jobs/:id/replay`
- `GET /api/jobs/:id/verify`
- `GET /api/jobs/:id/ledger`
- `POST /api/jobs/:id/export`

### Run APIs

- `POST /api/runs` (canonical submission)
- `GET /api/runs/:id`
- `GET /api/runs/:id/events`
- `POST /api/runs/:id/tool-calls`
- `POST /api/runs/:id/pause`
- `POST /api/runs/:id/resume`
- `POST /api/runs/:id/human-decisions`

### Agent APIs

- `POST /api/agents`
- `GET /api/agents`
- `GET /api/agents/:id/state`
- `GET /api/agents/:id/approvals`
- `POST /api/agents/:id/resume`
- `POST /api/agents/:id/stop`
- `GET /api/agents/:id/jobs`
- `GET /api/agents/:id/jobs/:job_id`

### Document APIs

- `POST /api/documents/upload`
- `POST /api/documents/upload/async`
- `GET /api/documents/upload/status/:task_id`
- `GET /api/documents/`
- `GET /api/documents/:id`
- `DELETE /api/documents/:id`

### Knowledge APIs

- `GET /api/knowledge/collections`
- `POST /api/knowledge/collections`
- `DELETE /api/knowledge/collections/:id`

### Observability APIs

- `GET /api/observability/summary`
- `GET /api/observability/stuck`
- `GET /api/jobs/:id/trace/page`
- `GET /api/jobs/:id/trace/cognition`
- `GET /api/jobs/:id/nodes/:node_id`
- `GET /api/trace/overview/page`

### Forensics APIs

- `POST /api/forensics/query`
- `POST /api/forensics/batch-export`
- `GET /api/forensics/export-status/:task_id`
- `GET /api/forensics/consistency/:job_id`

### Evidence & Audit APIs

- `GET /api/jobs/:id/evidence-graph`
- `GET /api/jobs/:id/audit-log`

### RBAC APIs

- `GET /api/rbac/role`
- `POST /api/rbac/role`
- `POST /api/rbac/check`

### Tool APIs

- `GET /api/tools`
- `GET /api/tools/:name`

### System APIs

- `GET /api/system/status`
- `GET /api/system/metrics`
- `GET /api/system/workers`

## 4. Experimental Surface

Experimental APIs may change without major bump, but should be noted in release notes:

- New endpoints not listed in Section 3
- Optional response fields marked experimental in docs/release notes
- Adapter-specific runtime internals

## 5. Request/Response Change Policy

For stable endpoints:

- Allowed:
  - Add optional request fields
  - Add optional response fields
  - Add new non-default query parameters
- Not allowed in `2.x`:
  - Remove required fields
  - Rename existing fields
  - Change field types incompatibly
  - Change endpoint semantics incompatibly

## 6. Deprecation Policy

When deprecating a stable API:

1. Mark as deprecated in docs
2. Add migration path in release notes
3. Keep API available for >= 2 minor versions
4. Remove only in next major, or after window with explicit notice

Example:

- Deprecated in `v2.2.0`
- Earliest removal target: `v2.4.0` (or `v3.0.0`)

### Current deprecation note (runtime-first migration)

- `POST /api/agents/:id/message` is retained as a compatibility facade.
- Canonical submission path for new integrations is `POST /api/runs` + `GET /api/jobs/:id`.
- During migration, `/api/agents/:id/message` may include optional response object `runtime_submission` with fields such as:
  - `legacy_facade` (bool)
  - `canonical_api` (string)
  - `job_id` (string)
  - `run_id` (string, optional)
  - `run_status` (`created` / `best_effort` / `disabled`)

## 7. Release Gates for Contract Safety (P0)

A `2.x` release should not be published unless:

- Contract docs are updated (`docs/api-contract.md`)
- Compatibility checks pass for stable endpoints (smoke + regression tests)
- Deprecations (if any) include migration guidance
- Release notes include API delta summary

## 8. References

- `docs/release-checklist-2.0.md`
- `docs/upgrade-1.x-to-2.0.md`
- `docs/runtime-guarantees.md`

## 9. Approval and Ledger Notes

The following endpoints are part of the stable operator-facing surface in `2.x`:

- Approval detail and actions are exposed under `/api/jobs/:id/approval*`
- Agent approval inbox listing is exposed under `/api/agents/:id/approvals`
- Tool invocation audit/troubleshooting is exposed under `/api/jobs/:id/ledger`

These endpoints are intended to support human-in-the-loop approval workflows and operator inspection without dropping down into raw event streams.

Approval responses may include optional `expires_at`, `expired`, and `expiry_action` fields for wait states configured with an expiration deadline. `expiry_action` tells operators whether timeout will default to `expired`, resume as `rejected`, or terminal-cancel the job.

When an approval wait has already expired, action endpoints such as approve, reject, and delegate return `409 Conflict` rather than re-opening the wait.

When a worker or embedded runtime is running, expired approval waits are also auto-settled in the background. The default behavior appends `wait_completed` with decision `expired` and moves the job back to `pending`, while nodes configured with `expiry_action=rejected` resume on the rejection path and `expiry_action=cancelled` move the job to terminal `job_cancelled`.
