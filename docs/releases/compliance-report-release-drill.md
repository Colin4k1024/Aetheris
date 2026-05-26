# Compliance Report Release Drill

This drill verifies the experimental compliance report boundary.

## Command

```bash
./scripts/release-compliance-report-drill.sh
```

The script writes a report under `artifacts/release/`.

## What It Covers

- template version metadata
- signed evidence verification binding
- unsupported control visibility
- HTTP report request validation
- SOX/GDPR/HIPAA report generation tests

## Pass Criteria

All targeted `pkg/compliance` and `internal/api/http` tests must pass.

The drill promotes compliance reports only as evidence-bound report generators. It does not make `/api/compliance/*` a stable API surface and does not claim legal compliance certification.
