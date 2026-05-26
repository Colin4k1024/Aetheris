# Distributed Verifier Release Drill

This drill verifies the distributed verifier prototype boundary.

## Command

```bash
./scripts/release-distributed-verifier-drill.sh
```

The script writes a report under `artifacts/release/`.

## What It Covers

- accepted root-hash consensus across organizations
- divergent root-hash detection
- pull failure, empty stream, and missing hash behavior
- promotion readiness assessment

## Pass Criteria

All targeted `pkg/distributed` tests must pass.

Passing this drill does not promote the distributed verifier to production-ready. The drill records that root-hash comparison works and that the package remains prototype until saturation, lease, and recovery evidence exists.
