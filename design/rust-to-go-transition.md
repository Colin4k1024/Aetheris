# Rust → Go Transition Plan (Aetheris)

## Scope / selected components
This plan targets the remaining Aetheris components currently implemented in Rust or coupled to Rust tooling/FFI, focusing on areas that represent the highest maintenance cost and operational risk:

- **Durable storage layer**
  - event log & job/event persistence
  - checkpoint/snapshot persistence
  - effect/ledger persistence
- **Execution/scheduling core**
  - session runtime state machine
  - step executor, idempotency, and replay semantics
- **Tool/runtime bridging layer**
  - MCP transports (SSE/STDIO) and tool bridge
  - embedded runtime adapters

## Rationale
- **Operational ownership:** Go aligns with the majority of Aetheris runtime code and existing production ops tooling; reducing a second ecosystem lowers the barrier to contribution and incident response.
- **FFI risk reduction:** Eliminating cgo/FFI and Rust runtime dependencies removes a major class of hard-to-debug crashes, memory ownership issues, and deployment portability problems.
- **Consistency:** One language for core runtime + tooling improves API coherence, testing strategy, and documentation.

## Target interfaces / Go architecture
Use explicit Go interfaces to keep the migration incremental and to allow dual implementations during the transition.

### Storage
Create/standardize small interfaces in Go (or reuse existing ones where they already exist) that express only the durable contract, not usage patterns:

- `JobStore`: append-only event persistence, state snapshotting, optimistic concurrency
- `CheckpointStore`: save/load checkpoints, TTL cleanup
- `EffectStore` / `LedgerStore`: at-most-once effect tracking, replay safety

### Execution core
- `Runtime` (high-level orchestration): submit job/session, poll status, signal/pause/resume
- `Executor`: step execution with idempotency, deterministic replay

### Tool bridge / transport
- `Transport`: stream events/frames, backpressure-aware, with unified cancellation semantics
- `ToolBridge`: translate runtime tool calls to host tool protocols (MCP/Eino), with explicit timeouts and error modeling

## Suggested timeline
Time estimates assume one core engineer + reviewer bandwidth and a safety-first rollout.

**Week 1: Planning & spike**
- inventory Rust components and exact cross-language boundaries
- define the Go interface contracts above and acceptance tests (behavioral)

**Weeks 2–4: Storage layer migration**
- implement Go `JobStore` + `CheckpointStore` adapters with parity metrics
- run dual-write (Rust primary / Go shadow) and compare event streams

**Weeks 5–6: Execution core**
- port replay/state machine logic into Go using the storage interfaces
- stress-test with fault injection (network partitions, restart during checkpoint, etc.)

**Weeks 7–8: Tool bridge / transport**
- port MCP transports and tool bridge into Go
- ensure backpressure, cancellation, and unblocked shutdown

**Weeks 9–10: Cutover & cleanup**
- flip feature flags to Go implementations
- remove Rust dependencies
- finalize documentation, upgrade guides, and TODO retirement

## Risk mitigation
- **Dual-run with shadow writes:** dual persistence (or mirrored read validation) until parity confidence is >99.9% on core metrics.
- **Strict compatibility gates:** keep external API + file formats stable; version internal schemas with migrations.
- **Load testing + SLOs:** compare throughput/latency and error rates before cutover; add alerts for divergence.
- **Rollback plan:** cutover uses feature flags and supports immediate revert within one deploy cycle; keep Rust build working until Go path is proven.
- **Security & compliance:** ensure cryptographic operations, signing, and audit trails are ported with vetted libraries and re-reviewed threat models.

## Definition of done
- Go implementations replace Rust in production, with performance within the agreed budget and passing replay/consistency tests.
- Rust code and build steps removed; CI green with only Go toolchain requirements.
- Documentation updated: upgrade guide, interface reference, and cutover checklist.
