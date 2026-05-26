# Forensics Read Model Release Drill

This drill verifies the experimental forensics read model boundary.

## Command

```bash
./scripts/release-forensics-read-model-drill.sh
```

The script writes a report under `artifacts/release/`.

## What It Covers

- tenant isolation for `POST /api/forensics/query`
- required `agent_filter`
- pagination limit cap at 200
- large event stream `event_count`
- experimental route gating
- batch export status flow

## Pass Criteria

All targeted `internal/api/http` tests must pass.

The drill does not promote `/api/forensics/*` to stable API status by itself. The endpoints remain experimental until API contract stabilization and read-model indexing decisions are complete.
