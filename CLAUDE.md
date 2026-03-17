# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

**Aetheris** (also known as CoRag) is an execution runtime for intelligent agents — "Temporal for Agents." It provides a durable, replayable, and observable environment where AI agents can plan, execute, pause, resume, and recover long-running tasks.

- **Go Module:** `rag-platform`
- **CLI:** `aetheris`
- **Default API Port:** 8080

## Common Commands

```bash
# Build all binaries (api, worker, cli) to bin/
make build

# Build and start API + Worker in background
make run

# Run only API or Worker
make run-api
make run-worker

# Stop services
make stop

# Run tests (with race detector)
make test

# Run integration tests (runtime + http)
make test-integration

# Format and lint
make fmt         # gofmt -w
make fmt-check   # check formatting
make vet         # go vet
make tidy        # go mod tidy

# Docker
make docker-build  # Build runtime container
make docker-run    # Start local 2.0 stack (postgres + api + workers)
make docker-stop   # Stop local stack

# Health check
curl http://localhost:8080/api/health
```

## Architecture

Aetheris treats agents as **virtual processes** — workers schedule and host processes; processes can pause, wait for signals, receive messages, and resume across different workers.

### Core Components

| Component         | Path                                      | Purpose                                                                      |
| ----------------- | ----------------------------------------- | ---------------------------------------------------------------------------- |
| **API Server**    | `cmd/api/`                                | HTTP server (Hertz), creates/interacts with agents                           |
| **Worker**        | `cmd/worker/`                             | Background execution worker, schedules and executes jobs                     |
| **CLI**           | `cmd/cli/`                                | Command-line tool (`aetheris init`, `chat`, `jobs`, `trace`, `replay`, etc.) |
| **AgentFactory**  | `internal/runtime/eino/agent_factory.go`  | Config-driven Eino ADK agent creation (recommended entry point)              |
| **Tool Bridge**   | `internal/runtime/eino/tool_bridge.go`    | Converts Aetheris RuntimeTool → Eino InvokableTool (interface abstraction)   |
| **Eino Engine**   | `internal/runtime/eino/engine.go`         | Workflow compilation, runner management, integrates AgentFactory              |
| **Agent Runtime** | `internal/agent/runtime/`                 | Core execution engine (DAG compiler + runner)                                |
| **Job Store**     | `internal/agent/runtime/job/`             | Event-sourced durable execution history (PostgreSQL)                         |
| **Scheduler**     | `internal/agent/runtime/job/scheduler.go` | Leases and retries tasks with lease fencing                                  |
| **Runner**        | `internal/agent/runtime/runner/`          | Step-level execution with checkpointing                                      |
| **Planner**       | `internal/agent/planner/`                 | Produces TaskGraph from goals                                                |
| **Executor**      | `internal/agent/runtime/executor/`        | Executes DAG nodes using eino framework                                      |
| **Effects**       | `internal/agent/effects/`                 | At-most-once tool execution guarantee via Ledger                             |

> **Note:** `internal/agent/agent.go` (legacy Agent struct) is deprecated. Use `AgentFactory` for all new agent construction.

### Execution Flow

```
User → Agent API → AgentFactory (config-driven) → Eino ADK Runner
                  → Job → Scheduler → Runner → Planner → TaskGraph → Tool/Workflow Nodes

Tool Flow:
  Tool Registry → RegistryToolBridge → Eino InvokableTool → ADK Agent ToolsConfig
```

### Key Design Documents

- `design/core.md` — Overall architecture
- `design/runtime-core-diagrams.md` — Runtime flow and StepOutcome semantics
- `design/execution-guarantees.md` — Formal guarantees table
- `design/internal/1.0-runtime-semantics.md` — Three mechanisms and Execution Proof Chain
- `design/internal/scheduler-correctness.md` — Lease fencing, step timeout guarantees
- `design/internal/step-contract.md` — Contract for writing correct steps (deterministic, side effects through Tools)

### Storage

- **PostgreSQL** — Job events, job state, checkpoints (primary)
- **Redis** — Optional for RAG/indexer

### Three Core Use Cases

1. **Human-in-the-Loop Operations** — Approval flows, StatusParked for long waits
2. **Long-Running API Orchestration** — At-most-once tool execution, crash recovery
3. **Auditable Decision Agents** — Evidence graph, execution proof chain, replay

### Key Technologies

- **Agent Framework:** cloudwego/eino
- **Web Framework:** cloudwego/hertz
- **Database:** jackc/pgx/v5 (PostgreSQL)
- **Cache:** redis/go-redis/v9
- **Auth:** hertz-contrib/jwt
- **Observability:** OpenTelemetry, Prometheus, slog

### Package Structure (internal/)

| Package | Lines | Purpose |
|---------|-------|---------|
| **agent/** | 18,716 | Core agent runtime engine |
| agent/runtime/ | 9,333 | DAG compiler, runner, state management |
| agent/job/ | 2,222 | Event-sourced job management, scheduling |
| agent/tools/ | 1,496 | Tool registry, MCP integration |
| agent/planner/ | 527 | TaskGraph generation from goals |
| agent/memory/ | 940 | Agent memory (episodic, longterm, short-term) |
| **api/** | 9,610 | HTTP (Hertz) and gRPC APIs |
| **runtime/** | 4,778 | Eino integration, job store, sessions |
| **app/** | 4,336 | Application layer orchestration |
| **pipeline/** | 3,165 | RAG pipelines (ingest, query) |
| **model/** | 1,970 | LLM, embedding, vision model clients |
| **storage/** | 1,532 | Vector, metadata, object, cache storage |
| **tool/** | 1,045 | Built-in tools (LLM, RAG, HTTP, workflow) |

> **Note:** The `agent/` package is well-organized into focused subpackages. Do not split further unless there's a clear architectural need.
