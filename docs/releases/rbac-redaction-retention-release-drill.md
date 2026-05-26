# RBAC, Redaction, and Retention Release Drill

This drill is the release gate for the 2.x multi-tenant safety hardening slice.

## Command

```bash
./scripts/release-rbac-redaction-retention-drill.sh
```

The script writes a report under `artifacts/release/`.

## What It Covers

- RBAC role/permission matrix
- tenant-scoped RBAC behavior
- HTTP tenant/RBAC regression suite
- redacted evidence export fixture
- retention GC invariant: lifecycle cleanup must not mutate replay event history

## Pass Criteria

All targeted suites must pass:

- `pkg/auth`
- `internal/api/http`
- `pkg/proof`
- `internal/runtime/jobstore`

The drill does not certify legal compliance. It only gates the bounded runtime safety claims documented in [rbac-redaction-retention-hardening.md](../guides/rbac-redaction-retention-hardening.md).
