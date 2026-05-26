---
artifact: prototype-promotion-backlog
task: architecture-review
date: 2026-05-25
role: architect
status: active
---

# Prototype Promotion Backlog

This backlog turns the architecture review's remaining P2 action into executable vertical slices. It is intentionally strict: a package or endpoint is not promoted because code exists; it is promoted only when the runtime path, API contract, tests, and operations evidence all exist.

## Promotion Rule

Use the policy in [docs/STATUS.md](../../STATUS.md): `prototype` -> `integrated` requires config, storage/schema notes, API/CLI surface, tests, and ops/runbook updates. `integrated` -> `production-ready` additionally requires release gates and failure drills.

## Recommended Order

| Slice | Current evidence | Target | Required actions before promotion | Decision |
|---|---|---|---|---|
| Signed evidence package | `security.evidence_signing` config, signed ZIP export, CLI public-key verification, key custody/rotation runbooks, and release drill exist | Keep `production-ready` for evidence ZIP signing only | Continue running release drill; future hardening may add KMS/Vault-backed signing | Done |
| Forensics query read model | Read-model doc, tenant isolation test, pagination cap test, large-event-stream test, and release drill exist | Keep `integrated`; endpoints remain experimental until API contract promotion | Continue running release drill; add indexed/materialized read model before removing experimental gate | Done |
| RBAC/redaction/retention hardening | Release drill covers role matrix, tenant/RBAC HTTP matrix, redacted evidence export, and retention replay invariants | Keep `production-ready` for bounded runtime safety claims | Continue running release drill; external policy/KMS/legal certification remain out of scope | Done |
| Compliance reports | Signed evidence binding, template versions, unsupported controls, HTTP export tests, and release drill exist | Keep `integrated` as a report generator, not a compliance guarantee | Continue running release drill; legal certification, GRC integration, and external policy evidence remain out of scope | Done |
| AI forensics detection | `pkg/ai_forensics` and `/api/forensics/ai/detect-anomalies` exist as 3.0-M4 candidates | Keep prototype | Define eval dataset and false-positive budget before API contract stabilization | Next |
| Distributed verifier | `pkg/distributed` exists as technical reserve | Keep prototype | Prove single-node runtime saturation, lease limits, and recovery bottlenecks before distributed promotion | Hold |
| Monitoring quality scorer | `pkg/monitoring` exists as standalone logic | Keep prototype or fold into observability | Connect to `/api/observability/*`; define alert semantics; add SRE runbook | Later |

## Non-Goals

- Do not declare all forensics/compliance/distributed packages `production-ready` in one batch.
- Do not expose gated 3.0 APIs by default.
- Do not add new top-level product lanes until the runtime guarantee boundary is already clear.

## Next Engineering Ticket

Start with **AI forensics detection**:

1. Define a small golden eval dataset for missing evidence, suspicious retry loops, and tampered reasoning snapshots.
2. Set a false-positive budget and expected severity mapping before changing API behavior.
3. Add release drill coverage that runs detector tests against the golden dataset.
4. Keep `/api/forensics/ai/detect-anomalies` experimental until eval quality is stable.
