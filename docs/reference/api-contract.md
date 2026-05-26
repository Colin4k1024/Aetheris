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
- `POST /api/jobs/:id/message`
- `GET /api/jobs/:id/events`
- `GET /api/jobs/:id/trace`
- `GET /api/jobs/:id/replay`
- `GET /api/jobs/:id/verify`
- `POST /api/jobs/:id/export` (evidence ZIP; includes signed proof when `security.evidence_signing.enabled=true`)

### Run APIs

- `POST /api/runs` (canonical submission)
- `GET /api/runs/:id`
- `GET /api/runs/:id/events`
- `POST /api/runs/:id/tool-calls`
- `POST /api/runs/:id/pause`
- `POST /api/runs/:id/resume`
- `POST /api/runs/:id/human-decisions`

### Agent APIs

- `GET /api/agents/:id/state`
- `POST /api/agents/:id/resume`
- `POST /api/agents/:id/stop`
- `GET /api/agents/:id/jobs`
- `GET /api/agents/:id/jobs/:job_id`

Agent definitions are loaded from configuration. Runtime creation/listing APIs are not part of the stable `2.x` surface.

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
- Endpoints exposed only when `api.forensics.experimental=true`:
  - `POST /api/forensics/query`
  - `POST /api/forensics/batch-export`
  - `GET /api/forensics/export-status/:task_id`
  - `GET /api/forensics/consistency/:job_id`
  - `POST /api/forensics/ai/detect-anomalies`
  - `GET /api/jobs/:id/evidence-graph`
  - `GET /api/jobs/:id/audit-log`
  - `GET /api/compliance/templates`
  - `POST /api/compliance/apply`
  - `POST /api/compliance/report`

Forensics query read-model shape and filter compatibility are documented in `docs/guides/forensics-read-model.md`, but the endpoints remain experimental until the API compatibility contract is explicitly promoted.

Compliance report shape and evidence-binding rules are documented in `docs/guides/compliance-reporting.md`. Reports must include signed evidence verification metadata, template versioning, and explicit unsupported controls; the endpoints remain experimental and do not certify legal compliance.

The next compliance contract slice is defined in `docs/artifacts/2026-05-26-compliance-surface-definition/api-cli-contract.md`. It is CLI-first and evidence-first: HTTP compliance endpoints stay experimental until the offline evidence verification and report-generation path has matching tests and release evidence.

AI-forensics detector behavior and false-positive budget are documented in `docs/guides/ai-forensics-eval.md`. The detector is eval-gated, but `/api/forensics/ai/detect-anomalies` remains experimental and must not be treated as an autonomous enforcement API.

RoutingAdvisor capability-routing behavior is defined as an experimental internal contract in `docs/guides/routing-advisor-contract.md`. There is no stable public HTTP or CLI surface for this capability. Replay must use recorded route decision evidence and must not call external routing advisors.

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

- Contract docs are updated (`docs/reference/api-contract.md`)
- Compatibility checks pass for stable endpoints (smoke + regression tests)
- Deprecations (if any) include migration guidance
- Release notes include API delta summary

## 8. References

- `docs/release-checklist-2.0.md`
- `docs/upgrade-1.x-to-2.0.md`
- `docs/guides/runtime-guarantees.md`
- `docs/guides/evidence-signing.md`
- `docs/guides/forensics-read-model.md`
- `docs/STATUS.md`
