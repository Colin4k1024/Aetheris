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
| signal \<job_id\>                                                 | Send signal to a job                                                                                                    |
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
| signal \<job_id\>         | POST /api/jobs/:id/signal                                                           |
| evidence-graph \<job_id\> | GET /api/jobs/:id/evidence-graph                                                    |
| export \<job_id\>         | POST /api/jobs/:id/export                                                           |
| verify \<job_id\>         | GET /api/jobs/:id/verify                                                            |
| tool list                 | GET /api/tools                                                                      |
| tool get \<name\>         | GET /api/tools/:name                                                                |

For more endpoints and flows see [usage.md](usage.md) "API endpoint summary" and "Typical flows".
