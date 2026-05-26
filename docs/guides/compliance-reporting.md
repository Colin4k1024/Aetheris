# Compliance Reporting

Compliance reports are experimental auditor-facing summaries generated from Aetheris runtime evidence. They are report generators, not legal compliance certifications.

## Boundary

`POST /api/compliance/report` is available only when `api.forensics.experimental=true`.

Aetheris' broader compliance surface is defined in
[`docs/artifacts/2026-05-26-compliance-surface-definition`](../artifacts/2026-05-26-compliance-surface-definition/README.md).
Only the signed-evidence-bound report generator is currently integrated; the
broader `pkg/compliance` package remains prototype until the offline CLI path,
tests, and release drill are implemented.

A report must be bound to a signed evidence package verification result:

- `evidence_package_id` identifies the ZIP or artifact being referenced.
- `evidence_verification.verified=true` confirms the package was verified before report generation.
- `evidence_verification.signed=true` and `signature_valid=true` confirm the report is based on signed evidence.
- `evidence_verification.root_hash` pins the report to the evidence chain root.

The API rejects report generation when these fields are missing or false.

## Output Contract

Reports include:

- `template_name` and `template_version`
- `evidence_package_id`
- `evidence_verification`
- `controls`
- `unsupported_controls`
- `summary`
- `compliance_notice`

`unsupported_controls` is intentional. It identifies controls that Aetheris cannot certify from runtime evidence alone and that require customer policy evidence, external system evidence, or manual auditor review.

## Current Templates

| Template | Version | Scope |
|---|---:|---|
| GDPR | 2026.05 | Runtime audit, data protection, breach notification evidence summary |
| HIPAA | 2026.05 | Runtime audit and security-management evidence summary |
| SOX | 2026.05 | Financial workflow audit trail, RBAC, retention/replay evidence summary |

## Non-Goals

- Aetheris does not certify legal compliance.
- Aetheris does not replace legal counsel, GRC tooling, policy evidence, or external auditor judgment.
- Unsupported controls must remain visible; do not silently count them as compliant.
- `POST /api/compliance/apply` is not a promoted runtime apply path. Treat it as experimental/prototype until it is renamed to a check/preview API or backed by real policy-enforcement semantics.
