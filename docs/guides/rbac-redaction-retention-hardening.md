# RBAC, Redaction, and Retention Hardening

This guide defines the production hardening boundary for the 2.x multi-tenant safety slice.

## Status

The slice is `production-ready` for the following bounded claims:

- RBAC role/permission checks are tenant-scoped.
- Evidence exports can be redacted while preserving package verification.
- Retention GC can archive/delete expired tool invocation lifecycle records without mutating replay event history.

This does not claim full enterprise policy management, external KMS-backed redaction encryption, or legal compliance certification.

## RBAC Boundary

RBAC is role based and tenant scoped.

| Role | Key permissions |
|---|---|
| `admin` | all runtime and management permissions |
| `operator` | view, stop, export, trace, tool execution |
| `auditor` | view, export, trace, audit view |
| `user` | view, create, trace |

The release drill verifies:

- admin has privileged permissions
- user cannot export
- auditor can export but cannot create/stop
- same user can have different roles in different tenants
- HTTP tenant/RBAC matrix remains green through `release-tenant-regression.sh`

## Redacted Evidence Export

`proof.ExportEvidenceZip` honors `ExportOptions.RedactionEnabled`.

When enabled, event payloads are redacted before serialization and the exported package hash chain is rebuilt over the redacted evidence.

Default evidence redaction covers:

- `email`: hashed with `RedactionSalt`
- `phone`: hashed with `RedactionSalt`
- `ssn`: redacted
- `credit_card`: redacted
- `api_key`, `token`, `password`, `secret`: removed

Safe fields remain available for audit context.

## Retention Boundary

Retention GC acts on lifecycle records such as tool invocation rows. It must not delete or rewrite the event history required for replay and audit.

The release drill verifies that expired lifecycle refs can be archived/deleted while the event stream version and events remain intact.

## Release Gate

Run:

```bash
./scripts/release-rbac-redaction-retention-drill.sh
```

The drill covers RBAC role matrix, HTTP tenant/RBAC regression, redacted evidence export, and retention replay invariants.
