---
artifact: runtime-path
task: compliance-surface-definition
date: 2026-05-26
role: architect
status: draft-for-implementation
---

# Runtime Path

## First Principles

Compliance output is useful only when every claim can point back to verifiable runtime evidence. If a control cannot be proven from Aetheris events, evidence packages, or release drills, the system must mark it unsupported instead of treating it as compliant.

## Current Runtime Facts

| Capability | Runtime path today | Status |
|---|---|---|
| Evidence export | `POST /api/jobs/:id/export`, `aetheris export <job_id>` | production-ready for evidence ZIP export |
| Evidence signature verification | `aetheris verify <evidence.zip> [--public-key base64]` | production-ready for signed evidence packages |
| Compliance report generation | `POST /api/compliance/report` with `evidence_verification` | integrated, experimental API |
| Compliance template listing | `GET /api/compliance/templates` | integrated, experimental API |
| Compliance apply/check | `POST /api/compliance/apply` | prototype/experimental; not a real apply path |
| Broader `pkg/compliance` validators | in-package helpers such as HIPAA checks | prototype; not wired into runtime, API, CLI, or release gates |

## Target Slice

Promote an offline compliance report path:

1. Export a job evidence ZIP from a completed runtime execution.
2. Verify the ZIP locally with the trusted public key.
3. Convert the verification result into `compliance.EvidenceVerification`.
4. Generate a compliance report from a versioned template.
5. Emit explicit unsupported controls for anything outside Aetheris runtime evidence coverage.
6. Store the generated report as an operator/auditor artifact, not as a legal certification.

## Runtime Flow

```text
job runtime events
  -> evidence package export
  -> signed ZIP verification
  -> evidence verification summary
  -> compliance reporter
  -> report JSON / ops artifact
```

The report generator must not query mutable runtime state after the evidence package has been verified unless the source is explicitly declared as a non-authoritative metric supplement. The evidence ZIP is the authority for audit claims.

## Required Data Boundary

The report input must contain:

- `tenant_id`
- `standard` (`GDPR`, `HIPAA`, or `SOX`)
- `time_range`
- `evidence_package_id`
- `evidence_verification.package_id`
- `evidence_verification.root_hash`
- `evidence_verification.verified=true`
- `evidence_verification.signed=true`
- `evidence_verification.signature_valid=true`
- optional `evidence_verification.signer_key_id`

The output must preserve:

- template name and version
- evidence package ID
- evidence verification result
- controls and summary
- unsupported controls
- compliance notice

## Non-Promotable Paths

These stay prototype until they receive their own vertical slice:

- mutating "apply compliance" behavior
- legal compliance certification
- external GRC system synchronization
- customer policy evidence ingestion
- cross-job or cross-tenant compliance aggregation
- automatic enforcement or blocking based on compliance score

## Promotion Gate

This slice may move from `prototype` to `integrated` only after:

- CLI command exists and can generate a report offline from a verified evidence ZIP.
- API contract and CLI contract are both documented.
- Tests cover success, missing signature, invalid root hash, unsupported controls, bad standard, and invalid time range.
- Release drill records the offline CLI path and writes an artifact under `artifacts/release/`.
