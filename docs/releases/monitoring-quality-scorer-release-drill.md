# Monitoring Quality Scorer Release Drill

This drill verifies the monitoring quality scorer's offline SRE semantics.

## Command

```bash
./scripts/release-monitoring-quality-scorer-drill.sh
```

The script writes a report under `artifacts/release/`.

## What It Covers

- deterministic scoring stays within 0-100
- low-quality decisions produce recommendations
- healthy, degraded, critical, and noisy alert semantics

## Pass Criteria

All targeted `pkg/monitoring` tests must pass.

Passing this drill does not wire the scorer into `/api/observability/*`. It records the current decision to keep it as an offline report utility until SRE aggregation and alert ownership are defined.
