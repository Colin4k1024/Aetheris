# Rust Core Rewrite Plan

> **Status**: approved
> **Since**: 2026-07-15
> **Approach**: Option C — Core rewrite + Go shell
> **Timeline**: ~19 weeks (4.5 months), 2-3 engineers

## Executive Summary

Rewrite Aetheris's core runtime (event store, executor, scheduler, effects, memory) in Rust using `oris-runtime` as the agent framework, while keeping the HTTP API layer and RAG pipeline in Go. Communication via C ABI (FFI) with protobuf for shared domain types.

**Scope**: ~23,000 LOC → Rust, ~44,000 LOC stays in Go.

## Background

Aetheris (CoRag) is a ~67,700 LOC Go codebase implementing "Temporal for Agents" — a durable, replayable execution runtime for AI agents. The rewrite targets correctness-critical paths (event sourcing, two-phase commit, deterministic replay) where Rust's type system and ownership model provide stronger guarantees.

### Key Motivations

- **Correctness**: Event sourcing, optimistic concurrency, and proof chains benefit from Rust's compile-time guarantees
- **oris-runtime**: Colin's own AI agent framework (`oris-runtime` v0.61.0) provides stateful graph, agent, tool, and multi-step execution — direct replacement for `cloudwego/eino`
- **Performance**: Core execution path (scheduler → executor → jobstore) is latency-sensitive
- **Safety**: Two-phase commit and compensation logic are high-stakes — Rust eliminates data races at compile time

## Architecture

### Module Ownership

```
┌─────────────────────────────────────────────────────────┐
│                    Go Shell (保留)                        │
│  cmd/api · cmd/worker · cmd/cli                         │
│  internal/api/http (67 routes, Hertz)                   │
│  internal/api/grpc (3 proto files)                      │
│  internal/pipeline/* (RAG ingest/query)                 │
│  internal/model/* (LLM/Embedding/Vision registry)       │
│  internal/storage/* (Vector/Cache/Metadata/Object)      │
│  internal/tool/* (MCP, Gatekeeper, Builtin)             │
│  pkg/routing · pkg/compliance · pkg/redaction           │
│  pkg/ai_forensics · pkg/forensics · pkg/evidence        │
└───────────────────────┬─────────────────────────────────┘
                        │ FFI (C ABI / cbindgen)
                        ▼
┌─────────────────────────────────────────────────────────┐
│                 Rust Core (新建)                          │
│  aetheris-jobstore    — oris kernel-postgres 封装        │
│  aetheris-executor    — DAG runner, replay, compensation │
│  aetheris-effects     — two-phase commit ledger          │
│  aetheris-scheduler   — lease + heartbeat                │
│  aetheris-planner     — DAG compiler                     │
│  aetheris-memory      — 4-tier: short/working/epi/long  │
│  aetheris-config      — figment config loader            │
│  aetheris-auth        — RBAC                             │
│  aetheris-metrics     — Prometheus + OTel                │
│  aetheris-types       — protobuf generated types         │
│  aetheris-ffi         — C ABI export (cbindgen)          │
└─────────────────────────────────────────────────────────┘
```

### Decision Matrix

| Module | LOC | Owner | Rationale |
|--------|-----|-------|-----------|
| `internal/agent/runtime/executor` | 9,962 | Rust | Correctness-critical — replay, compensation, determinism |
| `internal/runtime/jobstore` | 1,904 | Rust | Event sourcing core — optimistic concurrency, distributed lease |
| `internal/runtime/eino` | 2,966 | Rust (oris-runtime) | oris-runtime provides equivalent agent/workflow capabilities |
| `pkg/effects` | 1,541 | Rust | Two-phase commit — no room for data races |
| `internal/agent/planner` | 531 | Rust | DAG compiler, pure logic |
| `internal/agent/scheduler` | 554 | Rust | Lease renewal + timeout reclamation |
| `internal/agent/memory` | 940 | Rust | 4-tier memory concurrent access |
| `internal/agent/domain` | 701 | Rust (protobuf) | Shared domain types |
| `internal/agent/tools` | 1,737 | Rust (partial) | MCP bridge → oris tool trait |
| `internal/model/*` | 1,970 | Rust (partial) | LLM call chain managed by oris |
| `pkg/config` | 936 | Rust | Config loading is entry point |
| `pkg/auth` | 434 | Rust | RBAC on core execution path |
| `pkg/metrics + tracing` | 1,407 | Rust | Observability on core path |
| `pkg/proof + signature` | 1,048 | Rust | Cryptographic operations |
| `internal/api/http` | 7,255 | Go | Hertz 67 routes, high migration cost |
| `internal/api/grpc` | 3,117 | Go | Protobuf service layer |
| `internal/pipeline/*` | 3,166 | Go | RAG pipeline, coupled to Eino |
| `internal/storage/*` | 1,532 | Go | Vector/cache adapters |
| `internal/tool/*` | 2,644 | Go (partial) | MCP server side stays Go |
| `pkg/routing` | 2,083 | Go | LLM routing decisions |
| `pkg/compliance/redaction` | 1,743 | Go | Non-core path |

## Technology Stack

| Capability | Go Current | Rust Replacement |
|-----------|------------|-----------------|
| Agent Framework | cloudwego/eino | **oris-runtime** (v0.61.0) |
| Async Runtime | goroutine | **tokio** (oris-runtime dependency) |
| PostgreSQL | pgx/v5 | **sqlx** (oris kernel-postgres feature) |
| Redis | go-redis/v9 | **redis-rs** |
| Serialization | JSON | **serde + prost** (protobuf) |
| FFI Export | — | **cbindgen** |
| Cryptography | crypto/ed25519 | **ed25519-dalek + sha2** |
| Observability | OTel Go SDK | **opentelemetry-rust** |
| Config | Viper | **figment** |
| Error Handling | error wrapping | **thiserror + anyhow** |
| Logging | slog | **tracing** (tokio ecosystem) |

## Protobuf Schema

Shared type definitions in `proto/domain/`:

| File | Types |
|------|-------|
| `job_event.proto` | JobEvent, JobStatus, StepResultType, StepResult |
| `agent_state.proto` | AgentState, AgentInstance, AgentConfig |
| `tool_call.proto` | ToolInvocation, ToolResult, ToolCapability |
| `checkpoint.proto` | Checkpoint, TaskGraph, CursorNode |
| `memory.proto` | MemoryEntry, MemoryNamespace, MemoryQuery |
| `job_commands.proto` | AppendRequest/Response, ClaimRequest/Response |

Codegen:
- **Rust**: prost (`build.rs`) → `src/generated/*.rs`
- **Go**: `protoc --go_out` → `pb/*.pb.go`
- **CI**: Breaking change detection on `.proto` files

## FFI Boundary

### Go → Rust (Primary Path)

```c
// JobStore
int32_t aetheris_jobstore_new(const char* dsn, uintptr_t* out_handle);
int32_t aetheris_jobstore_append(uintptr_t handle, const char* job_id,
                                  int32_t expected_version,
                                  const uint8_t* event_json, size_t len,
                                  int32_t* out_new_version);
int32_t aetheris_jobstore_claim(uintptr_t handle, const char* worker_id,
                                 uint8_t** out_json, size_t* out_len);
void    aetheris_free(uint8_t* ptr, size_t len);

// Executor
int32_t aetheris_executor_new(uintptr_t jobstore_handle, uintptr_t* out_handle);
int32_t aetheris_executor_run_step(uintptr_t handle, const char* job_id,
                                    const uint8_t* input_json, size_t len,
                                    uint8_t** out_result, size_t* out_result_len);
```

### Rust → Go (Callback, On-Demand)

```c
typedef int32_t (*go_http_call_fn)(const char* url, const uint8_t* body, size_t body_len,
                                    uint8_t** out_resp, size_t* out_resp_len);
typedef void    (*go_log_fn)(int32_t level, const char* message);
void aetheris_register_callbacks(go_http_call_fn http_fn, go_log_fn log_fn);
```

### Error Code Convention

| Code | Meaning |
|------|---------|
| 0 | Success |
| -1 | Invalid input |
| -2 | Version conflict (optimistic concurrency) |
| -3 | Database error |
| -4 | Timeout |
| -5 | Internal error |

## Rust Workspace Structure

```
aetheris-core/
├── Cargo.toml              (workspace)
├── crates/
│   ├── aetheris-types/     (protobuf generated + domain extensions)
│   ├── aetheris-jobstore/  (oris kernel-postgres wrapper)
│   ├── aetheris-executor/  (DAG runner, replay, compensation)
│   ├── aetheris-effects/   (two-phase commit)
│   ├── aetheris-scheduler/ (lease + heartbeat)
│   ├── aetheris-planner/   (DAG compiler)
│   ├── aetheris-memory/    (4-tier memory)
│   ├── aetheris-config/    (figment config loader)
│   ├── aetheris-auth/      (RBAC)
│   ├── aetheris-metrics/   (Prometheus + OTel)
│   └── aetheris-ffi/       (C ABI export, cbindgen)
├── proto/
│   └── domain/             (.proto files, synced from Aetheris repo)
└── build.rs
```

## Execution Phases

### Phase 0: Foundation (Week 1-2)

**Goal**: Rust project skeleton, protobuf definitions, FFI compilation pipeline, CI

| Task | Output | Owner |
|------|--------|-------|
| Create `aetheris-core/` Rust workspace | Cargo.toml workspace config, crate split | Rust |
| Define `proto/domain/*.proto` (6 files) | JobEvent, AgentState, ToolCall, Checkpoint, Memory, StepResult | Shared |
| prost codegen config | `build.rs`, `cargo build` auto-generates Rust structs | Rust |
| protoc Go codegen config | Makefile target, `protoc --go_out` generates Go structs | Go |
| cbindgen + FFI skeleton | Empty function export + Go `import "C"` compiles | Rust |
| CI pipeline | `cargo build` + `cargo test` + `go build` + cross-compile | DevOps |

**Exit Criteria**:
- `cargo build` passes
- Go calls Rust empty function and gets success
- Protobuf-generated Rust/Go types can serialize/deserialize each other

### Phase 1: Core Storage (Week 3-5)

**Goal**: JobStore + Effects implemented in Rust, callable from Go via FFI

| Task | Go Module | Rust Crate |
|------|-----------|------------|
| JobStore trait + oris kernel-postgres impl | `internal/runtime/jobstore` | `aetheris-jobstore` |
| Append (optimistic concurrency + proof chain) | `pgstore.go:Append` | oris kernel-postgres |
| Claim / Heartbeat / Watch | `pgstore.go:Claim/Heartbeat` | oris kernel-postgres |
| Snapshot Create/Get/Delete | `pgstore.go:Snapshot*` | `aetheris-jobstore` |
| MemoryStore impl (for testing) | `memory_store.go` | `aetheris-jobstore` |
| EffectStore trait + impl | `internal/agent/runtime/effects` | `aetheris-effects` |
| EffectLog (two-phase commit) | `pkg/effects` | `aetheris-effects` |
| FFI export: jobstore_* / effect_* | — | `aetheris-ffi` |
| Go wrapper: RustJobStore | `internal/runtime/jobstore/` | — |

**Exit Criteria**:
- Go existing JobStore test cases pass via Rust FFI
- Benchmark: Rust Append throughput >= Go pgx direct
- Two-phase commit correctness tests pass

### Phase 2: Execution Engine (Week 6-9)

**Goal**: Core execution engine in Rust — most complex phase

| Task | Go Module | Rust Crate |
|------|-----------|------------|
| StepResult type + classification logic | `runner.go:StepResultType` | `aetheris-executor` |
| Runner core loop | `runner.go` (1,644 LOC) | `aetheris-executor` |
| DAG compiler (TaskGraph → execution plan) | `internal/agent/planner` | `aetheris-planner` |
| Scheduler polling loop | `scheduler.go` | `aetheris-scheduler` |
| Checkpoint create/restore | `checkpoint.go` | `aetheris-executor` |
| Replay engine | `internal/agent/replay` (1,235 LOC) | `aetheris-executor` |
| Compensation register/execute | `compensation.go` | `aetheris-executor` |
| Determinism verification | `internal/agent/determinism` | `aetheris-executor` |
| FFI export: executor_* / scheduler_* | — | `aetheris-ffi` |

**Exit Criteria**:
- Go `internal/agent/runtime/executor` all test cases pass via FFI
- Replay consistency verification passes
- Crash recovery scenario passes (kill worker → restart → recover from checkpoint)

### Phase 3: Agent Runtime (Week 10-12)

**Goal**: oris-runtime integration, replacing Eino agent/workflow capabilities

| Task | Go Module | Rust Crate |
|------|-----------|------------|
| oris-runtime init + config | `internal/runtime/eino/engine.go` | `aetheris-executor` |
| AgentFactory → oris agent creation | `internal/runtime/eino/agent_factory.go` | `aetheris-executor` |
| Workflow compile → oris graph | `internal/runtime/eino/engine.go:Compile` | oris-runtime |
| Tool bridge (RuntimeTool ↔ oris Tool) | `internal/runtime/eino/tool_bridge.go` | `aetheris-ffi` |
| Memory 4-tier implementation | `internal/agent/memory` (940 LOC) | `aetheris-memory` |
| MCP client bridge (Rust side) | `internal/agent/tools/mcp*` | `aetheris-ffi` |

**Exit Criteria**:
- Go API creates Run → Rust oris-runtime orchestrates execution → returns result
- Tool call chain works (Go registered tool → Rust executor calls → Go handler executes)
- Memory read/write consistency

### Phase 4: Shared Libraries (Week 13-14)

**Goal**: Supporting libraries in Rust

| Task | Rust Crate | Notes |
|------|------------|-------|
| Config loading (YAML + env) | `aetheris-config` | figment, reads same YAML as Go Viper |
| RBAC permission check | `aetheris-auth` | Deserialize role/permission from protobuf |
| Prometheus metrics | `aetheris-metrics` | prometheus-client crate |
| OTel tracing | `aetheris-metrics` | opentelemetry-rust |
| Proof chain verification | `aetheris-types` | SHA256 hash chain |
| Ed25519 signing | `aetheris-types` | ed25519-dalek |

**Exit Criteria**:
- Config loads and matches Go Viper output
- Metrics export matches existing Prometheus format
- Proof chain verification produces identical results

### Phase 5: Go Integration (Week 15-17)

**Goal**: Go layer fully switches to Rust FFI, regression test coverage

| Task | Notes |
|------|-------|
| Go wrapper layer complete | Every Rust FFI function has a Go method, error code mapping |
| Existing Go test suite passes | `go test ./...` all pass (calling Rust underneath) |
| Integration tests | API → Go handler → FFI → Rust → PG full chain |
| Performance comparison benchmark | Go native vs Rust FFI, confirm no significant regression |
| Docker build pipeline | `cargo build --release` + `go build` unified build |
| Callback mechanism implementation | Rust → Go callback (scenarios confirmed in Phase 3) |

**Exit Criteria**:
- `go test ./...` passes with Rust backend
- Integration test full chain passes
- Performance: Rust FFI within 10% of Go native (or better)

### Phase 6: Cut-over (Week 18-19)

**Goal**: Gradual rollout, clean up Go legacy code

| Task | Notes |
|------|-------|
| Feature flag control | `use_rust_core=true/false` env var switch |
| Gradual rollout | 10% → 50% → 100% traffic on Rust path |
| Mark Go legacy code deprecated | `internal/agent/runtime/executor`, `internal/runtime/jobstore`, etc. |
| Documentation update | Architecture diagram, dev guide, deployment notes |
| Legacy code cleanup | Next version removes deprecated Go modules |

**Exit Criteria**:
- 100% traffic on Rust path, no Go fallback needed
- Legacy Go modules removed
- Documentation updated

## Timeline

```
Week  1-2   ████████ Phase 0: Foundation
Week  3-5   ████████████ Phase 1: Core Storage
Week  6-9   ████████████████ Phase 2: Execution Engine
Week 10-12  ████████████ Phase 3: Agent Runtime
Week 13-14  ████████ Phase 4: Shared Libraries
Week 15-17  ████████████ Phase 5: Go Integration
Week 18-19  ████████ Phase 6: Cut-over
────────────────────────────────────────────
Total: ~19 weeks (~4.5 months), 2-3 engineers
```

## Risk Register

| Risk | Impact | Mitigation |
|------|--------|------------|
| oris-runtime event sourcing semantics mismatch | Phase 1 blocker | Phase 0 spike validation |
| CGo FFI performance overhead (serialization + cross-language call) | Hot path regression | Benchmark gate, shared memory for hot paths |
| Protobuf type evolution breaks Go/Rust compatibility | Runtime errors | Proto breaking change CI check |
| Rust team learning curve | Schedule delay | Phase 0-1 as ramp-up period |
| Dual-language maintenance cost | Long-term burden | Phase 6 cleanup is a hard constraint |

## Dependencies

| Dependency | Owner | Status |
|-----------|-------|--------|
| oris-runtime v0.61.0 | Colin | Available, kernel-postgres feature ready |
| PostgreSQL schema (21 tables) | Shared | Compatible — Rust reads existing schema |
| Proto definitions | To create in Phase 0 | — |
| CI/CD pipeline update | DevOps | To create in Phase 0 |

## Verification Strategy

### Per-Phase Verification

- **Phase 0**: Compilation + basic FFI roundtrip
- **Phase 1**: Existing Go JobStore test suite against Rust impl
- **Phase 2**: Existing Go executor test suite + replay/compensation tests
- **Phase 3**: End-to-end agent run via oris-runtime
- **Phase 4**: Config/metrics/proof output comparison
- **Phase 5**: Full `go test ./...` + integration tests + benchmarks
- **Phase 6**: Production traffic observation

### Cross-Cutting Verification

- **Data compatibility**: Rust reads/writes existing PostgreSQL data without migration
- **API compatibility**: Go HTTP API surface unchanged — zero breaking changes for clients
- **Performance**: Core path latency ≤ Go native (target: equal or better)
- **Correctness**: Proof chain hash output identical between Go and Rust implementations
