# AI Forensics Eval Release Drill

This drill verifies the experimental AI forensics detector against a deterministic golden eval dataset and false-positive budget.

## Command

```bash
./scripts/release-ai-forensics-eval-drill.sh
```

The script writes a report under `artifacts/release/`.

## What It Covers

- golden eval dataset pass/fail
- zero false positives on clean execution
- missing evidence severity
- suspicious retry loop severity
- tampered reasoning snapshot severity
- HTTP detector signal extraction from event streams
- experimental route gating

## Pass Criteria

All targeted `pkg/ai_forensics` and `internal/api/http` tests must pass.

The drill keeps `/api/forensics/ai/detect-anomalies` experimental. Passing the drill means the detector is eval-gated, not production-certified for autonomous enforcement.
