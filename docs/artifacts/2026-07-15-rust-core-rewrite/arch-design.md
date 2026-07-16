# Rust Core Rewrite — Architecture

> **Status**: Phase 5 complete, Phase 6 in progress
> **Branch**: feat/rust-core-rewrite

## Overview

Aetheris's core runtime has been rewritten in Rust for correctness-critical paths (event sourcing, two-phase commit, deterministic replay). The Go layer retains the HTTP API, RAG pipeline, and Eino-compatible agent framework.

```
┌─────────────────────────────────────────────────────────┐
│                    Go Layer (retained)                    │
│  cmd/api (Hertz HTTP) · cmd/worker · cmd/cli             │
│  internal/api/http (67 routes)                           │
│  internal/pipeline/* (RAG)                               │
│  internal/model/* (LLM registry)                         │
│  internal/storage/* (Vector/Cache)                       │
│  internal/tool/* (MCP, Gatekeeper)                       │
│  pkg/routing · pkg/compliance · pkg/redaction            │
└───────────────────────┬─────────────────────────────────┘
                        │ CGo FFI (C ABI)
                        ▼
┌─────────────────────────────────────────────────────────┐
│                 Rust Core (aetheris-core/)                │
│  aetheris-jobstore    — Event store (oris kernel-pg)     │
│  aetheris-executor    — DAG runner, replay, compensation │
│  aetheris-effects     — Two-phase commit ledger          │
│  aetheris-scheduler   — Lease + heartbeat                │
│  aetheris-planner     — DAG compiler                     │
│  aetheris-memory      — 4-tier memory system             │
│  aetheris-config      — YAML config loader               │
│  aetheris-auth        — RBAC                             │
│  aetheris-metrics     — Prometheus + OTel                 │
│  aetheris-types       — Protobuf types + proof chain     │
│  aetheris-ffi         — C ABI export (17 functions)      │
└─────────────────────────────────────────────────────────┘
```

## Feature Flag

Controlled by environment variable:

| Variable | Values | Default |
|----------|--------|---------|
| `AETHERIS_USE_RUST_CORE` | `true` / `false` | `false` |

When `true`, the Go layer uses Rust FFI for JobStore, EffectStore, and Executor.
When `false` (default), the Go layer uses the native Go implementations.

## Module Mapping

| Go Module | Rust Crate | FFI Function |
|-----------|-----------|-------------|
| `internal/runtime/jobstore` | `aetheris-jobstore` | `aetheris_jobstore_*` |
| `internal/agent/runtime/effects` | `aetheris-effects` | `aetheris_effectstore_*` |
| `internal/agent/runtime/executor` | `aetheris-executor` | `aetheris_executor_*` |
| `internal/agent/planner` | `aetheris-planner` | (via executor) |
| `internal/agent/memory` | `aetheris-memory` | (via executor) |
| `pkg/config` | `aetheris-config` | (loaded at init) |
| `pkg/auth` | `aetheris-auth` | (via executor) |
| `pkg/metrics` | `aetheris-metrics` | (via executor) |
| `pkg/proof` | `aetheris-types::proof` | (via executor) |

## Protobuf Schema

Shared types defined in `aetheris-core/proto/domain/`:
- `job_event.proto` — JobEvent, JobStatus, StepResultType, StepResult
- `agent_state.proto` — AgentState, AgentInstance, AgentConfig
- `tool_call.proto` — ToolInvocation, ToolResult, ToolCapability
- `checkpoint.proto` — Checkpoint, TaskGraph, TaskNode
- `memory.proto` — MemoryEntry, MemoryNamespace, MemoryQuery
- `job_commands.proto` — AppendRequest/Response, ClaimRequest/Response

Codegen:
- Rust: prost (`build.rs`)
- Go: `protoc --go_out`

## Build

```bash
# Build Rust core only
make build-rust

# Build everything (Rust + Go)
make build-all

# Run Rust tests
cd aetheris-core && PROTOC=~/.local/bin/protoc cargo test

# Run Go tests (with Rust FFI)
go test ./internal/coreffi/ ./internal/corebridge/
```

## Database Schema

Rust core uses the same PostgreSQL schema as Go (`internal/runtime/jobstore/schema.sql`):
- `job_events` — Event stream
- `job_claims` — Distributed lease
- `job_snapshots` — Compaction snapshots
- 21 tables total

## Test Coverage

| Component | Tests | Status |
|-----------|-------|--------|
| Rust: aetheris-jobstore | 6 | Pass |
| Rust: aetheris-effects | 4 | Pass |
| Rust: aetheris-planner | 5 | Pass |
| Rust: aetheris-executor | 5 | Pass |
| Rust: aetheris-memory | 4 | Pass |
| Rust: aetheris-metrics | 3 | Pass |
| Rust: aetheris-types | 2 | Pass |
| Rust: aetheris-config | 1 | Pass |
| Rust: aetheris-auth | 4 | Pass |
| Go: coreffi | 4 | Pass |
| Go: corebridge | 6 | Pass |
| **Total** | **44** | **All Pass** |

## Rollout Strategy

1. **Phase 6a**: Deploy with `AETHERIS_USE_RUST_CORE=false` (default, no change)
2. **Phase 6b**: Enable in staging with `AETHERIS_USE_RUST_CORE=true`
3. **Phase 6c**: Gradual production rollout (10% → 50% → 100%)
4. **Phase 6d**: Remove Go native fallback code (next major version)

## Risk Mitigation

- Feature flag allows instant rollback to Go native
- Both implementations share the same PostgreSQL schema
- Rust core has comprehensive unit tests (34 tests)
- Go FFI integration tested (4 tests)
- Corebridge fallback tested (6 tests)
