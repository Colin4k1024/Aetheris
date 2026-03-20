# CLI

The CLI is used for debugging and admin: start services, submit/inspect runtime workloads, and troubleshoot jobs and traces. The binary is **aetheris**.

> Runtime-first note: canonical APIs are `/api/runs/*` and `/api/jobs/*`. Agent-centric commands (`agent create`, `chat`) are compatibility facades for migration.

## Install and run

From the repo root:

```bash
go build -o bin/aetheris ./cmd/cli
```

`bin/aetheris` (and any root-level `./cli` executable) is a build artifact and should not be committed to git.

Put `bin/aetheris` in your PATH to run it directly. Or run without building:

```bash
go run ./cmd/cli <command> [args]
```

## Release build (example)

Build reproducible release artifacts to `artifacts/`:

```bash
mkdir -p artifacts
GOOS=darwin GOARCH=arm64 go build -trimpath -ldflags "-s -w" -o artifacts/aetheris-darwin-arm64 ./cmd/cli
GOOS=linux  GOARCH=amd64 go build -trimpath -ldflags "-s -w" -o artifacts/aetheris-linux-amd64 ./cmd/cli
shasum -a 256 artifacts/aetheris-* > artifacts/aetheris-checksums.txt
```

This repo stores source and build scripts only. Publish binaries via release assets, not as tracked files in the repository.

## API base URL

The CLI uses the **AETHERIS_API_URL** environment variable for the API base URL; default is `http://localhost:8080`. Set it for remote or custom deployment.

## Subcommands

| Command                                                           | Description                                                                                                             |
| ----------------------------------------------------------------- | ----------------------------------------------------------------------------------------------------------------------- |
| version                                                           | Print version (e.g. aetheris cli 2.2.0)                                                                                 |
| health                                                            | Health check (prints ok)                                                                                                |
| config                                                            | Show config summary (e.g. api.port, api.host)                                                                           |
| server start                                                      | Start API (runs go run ./cmd/api)                                                                                       |
| worker start                                                      | Start Worker (runs go run ./cmd/worker)                                                                                 |
| agent create [name]                                               | [legacy facade] Create agent, print agent_id; default name "default" if omitted                                         |
| agent list                                                        | List all agents                                                                                                         |
| agent state \<agent_id\>                                          | Get agent state                                                                                                         |
| chat [agent_id]                                                   | [legacy facade] Interactive chat: send messages, get job_id, poll status; uses AETHERIS_AGENT_ID if agent_id not passed |
| jobs \<agent_id\>                                                 | List jobs for this agent                                                                                                |
| job \<job_id\>                                                    | Get job details                                                                                                         |
| trace \<job_id\>                                                  | Print job execution timeline (trace JSON) and Trace page URL                                                            |
| workers                                                           | List active workers (Postgres mode)                                                                                     |
| replay \<job_id\>                                                 | Print job event stream (for replay) and Trace page URL                                                                  |
| monitor [--watch] [--interval N]                                  | Print observability summary; optional watch mode                                                                        |
| stuck                                                             | Show stuck jobs                                                                                                         |
| migrate m1-sql                                                    | Print M1 incremental migration SQL (job_events hash fields)                                                             |
| migrate backfill-hashes --input events.ndjson --output out.ndjson | Backfill `prev_hash/hash` for NDJSON event exports                                                                      |
| cancel \<job_id\>                                                 | Request cancel of a running job                                                                                         |
| signal \<job_id\> \<correlation_key\>                           | Send signal to a waiting or parked job                                                                                  |
| approvals list \<agent_id\>                                      | List current pending approvals for an agent                                                                            |
| approvals get \<job_id\>                                         | Get approval details for a waiting job                                                                                  |
| approvals approve \<job_id\> [reason]                            | Approve a waiting job and move it back to pending                                                                       |
| approvals reject \<job_id\> [reason]                             | Reject a waiting job and move it back to pending                                                                        |
| approvals delegate \<job_id\> \<delegate_to\> [reason]          | Record a delegation action and keep the job waiting                                                                     |
| ledger inspect \<job_id\>                                        | Inspect tool invocation ledger state for a job                                                                          |
| debug \<job_id\> [--compare-replay]                               | Agent debugger: timeline + evidence + replay verification                                                               |
| verify \<job_id\>                                                 | Execution verification: execution_hash, event_chain_root_hash, ledger proof, replay proof                               |
| verify \<evidence.zip\>                                           | Offline evidence package verification                                                                                   |
| evidence-graph \<job_id\>                                         | Get job evidence graph                                                                                                  |
| export \<job_id\>                                                 | Export job forensics data                                                                                               |
| tool list                                                         | List available tools                                                                                                    |
| tool get \<name\>                                                 | Get tool definition                                                                                                     |

## Mapping to REST API

| CLI command               | REST API                                                                            |
| ------------------------- | ----------------------------------------------------------------------------------- |
| agent create [name]       | POST /api/agents (legacy facade)                                                    |
| agent list                | GET /api/agents                                                                     |
| agent state \<agent_id\>  | GET /api/agents/:id/state                                                           |
| chat                      | POST /api/agents/:id/message (legacy facade); poll GET /api/agents/:id/jobs/:job_id |
| jobs \<agent_id\>         | GET /api/agents/:id/jobs (legacy facade)                                            |
| job \<job_id\>            | GET /api/jobs/:id                                                                   |
| trace \<job_id\>          | GET /api/jobs/:id/trace                                                             |
| replay \<job_id\>         | GET /api/jobs/:id/events                                                            |
| workers                   | GET /api/system/workers                                                             |
| monitor                   | GET /api/observability/summary                                                      |
| stuck                     | GET /api/observability/stuck                                                        |
| cancel \<job_id\>         | POST /api/jobs/:id/stop                                                             |
| signal \<job_id\> \<correlation_key\> | POST /api/jobs/:id/signal                                             |
| approvals list \<agent_id\>      | GET /api/agents/:id/approvals                                                     |
| approvals get \<job_id\>         | GET /api/jobs/:id/approval                                                        |
| approvals approve \<job_id\>     | POST /api/jobs/:id/approval/approve                                               |
| approvals reject \<job_id\>      | POST /api/jobs/:id/approval/reject                                                |
| approvals delegate \<job_id\> \<delegate_to\> | POST /api/jobs/:id/approval/delegate                               |
| ledger inspect \<job_id\>        | GET /api/jobs/:id/ledger                                                          |
| evidence-graph \<job_id\> | GET /api/jobs/:id/evidence-graph                                                    |
| export \<job_id\>         | POST /api/jobs/:id/export                                                           |
| verify \<job_id\>         | GET /api/jobs/:id/verify                                                            |
| tool list                 | GET /api/tools                                                                      |
| tool get \<name\>         | GET /api/tools/:name                                                                |

## Approval workflow shortcuts

For human-in-the-loop jobs, a typical CLI flow is:

```bash
# Find pending approvals for an agent
aetheris approvals list refund-agent

# Inspect the approval request
aetheris approvals get <job_id>

# If the response shows expired=true, the approval can no longer be acted on
# The response also shows expiry_action so operators can see the configured timeout outcome.
# Running workers auto-settle expired approval waits using the node's expiry_action.
# The default is decision=expired; configured nodes may instead resume as rejected or terminal-cancel the job.

# Approve or reject it
aetheris approvals approve <job_id> "policy-ok"
aetheris approvals reject <job_id> "missing evidence"

# Or delegate while keeping the job in waiting state
aetheris approvals delegate <job_id> backup-reviewer "OOO"

# For at-most-once troubleshooting, inspect the tool ledger
aetheris ledger inspect <job_id>
```

For more endpoints and flows see [usage.md](usage.md) "API endpoint summary" and "Typical flows".
