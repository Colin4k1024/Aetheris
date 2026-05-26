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
| AI forensics detection | Golden eval dataset, false-positive budget, severity mapping, event-signal extraction, HTTP tests, and release drill exist | Keep `integrated` as an eval-gated detector; API remains experimental | Continue running eval drill; do not use for autonomous blocking without policy/human layer | Done |
| Distributed verifier | Root-hash consensus/divergence tests, readiness assessment, runbook, and release drill exist | Keep prototype | Promotion remains blocked until single-node saturation, lease, and recovery evidence exists | Done |
| Monitoring quality scorer | Alert semantics, SRE runbook, healthy/degraded/critical/noisy tests, and release drill exist | Keep prototype as offline report utility; do not fold into `/api/observability/*` yet | Define aggregation windows, Prometheus labels, and alert ownership before API integration | Done |

## Non-Goals

- Do not declare all forensics/compliance/distributed packages `production-ready` in one batch.
- Do not expose gated 3.0 APIs by default.
- Do not add new top-level product lanes until the runtime guarantee boundary is already clear.

## Next Engineering Ticket

The next active implementation ticket is the compliance surface slice defined in
[compliance-surface-definition](../2026-05-26-compliance-surface-definition/README.md).

Implementation should start with the offline CLI path: verify a signed evidence
ZIP, generate a versioned compliance report, preserve unsupported controls, and
extend the compliance report release drill.

The next active design ticket is the issue #197 routing-advisor contract defined
in [routing-advisor-contract](../2026-05-26-routing-advisor-contract/README.md).
It should stay design-only until replay, event evidence, config, adapter, and
release-drill tests are implemented.
