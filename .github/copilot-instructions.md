# Aetheris repository instructions for GitHub Copilot review

You are reviewing a Go repository that implements a durable execution runtime for intelligent agents.

Prioritize correctness, durability, replay safety, and operational reliability over style-only feedback.

## Project review priorities

Focus reviews on these invariants first:

1. At-most-once side effects
   - Flag any change that can cause duplicate external tool execution, duplicate writes, duplicate messages, or repeated effects after retries or crash recovery.
   - Check effect ledger, invocation identity, idempotency keys, and commit ordering.

2. Event-sourced correctness
   - Flag changes that can break append-only event semantics, event ordering, event version compatibility, or replay determinism.
   - Watch for hidden mutable state that is not reconstructed from events/checkpoints.

3. Checkpoint and recovery safety
   - Verify cursor, checkpoint, and persisted state transitions are consistent.
   - Flag cases where a crash between steps can leave the job in an unrecoverable or ambiguous state.
   - Prefer "persist before acknowledge" and explicit recovery semantics.

4. Scheduler and lease fencing
   - Review for race conditions, duplicate leasing, stale lease ownership, clock/timeout issues, and retry storms.
   - Flag changes that weaken fencing, retry backoff, task ownership checks, or recovery under worker restarts.

5. Deterministic replay
   - Flag use of time.Now, rand, map iteration order, goroutine ordering assumptions, hidden I/O, or nondeterministic behavior in replay-sensitive paths unless explicitly isolated and persisted.
   - Runtime decisions must be reproducible from persisted state.

6. Human-in-the-loop and parked states
   - Verify pause/resume behavior preserves state, input context, and authorization boundaries.
   - Flag transitions that may resume incorrectly or bypass required approval gates.

7. Auditability and observability
   - Prefer review comments when changes reduce traceability, remove important events/metadata, or make incident debugging harder.
   - Important runtime transitions should remain observable via logs, metrics, traces, or persisted events.

## Go-specific expectations

- Prefer context propagation for request-scoped and job-scoped operations.
- Flag missing timeout/cancel handling in network, DB, or long-running operations.
- Flag swallowed errors, ambiguous retries, or retries that can re-run committed side effects.
- Watch for data races, unsafe shared mutable state, goroutine leaks, and channel deadlocks.
- Prefer explicit error wrapping with actionable context.

## Testing expectations

Prioritize comments when PRs change runtime, scheduler, storage, checkpoint, replay, effect, or recovery logic without adequate tests.

Encourage tests for:
- crash recovery
- duplicate delivery / duplicate execution prevention
- replay determinism
- lease fencing and concurrent workers
- parked/resume flows
- backward compatibility of persisted events or state restoration

## Review style

- Be concise and specific.
- Prefer high-signal findings over style nitpicks.
- Explain why the issue matters operationally.
- Suggest a concrete fix or a validating test when possible.
- Avoid commenting on formatting unless it affects correctness or maintainability.

## Key directories

Focus extra attention on changes in these directories:
- `internal/agent/runtime/` - Core execution engine
- `internal/agent/runtime/job/` - Event-sourced job management
- `internal/agent/runtime/runner/` - Step-level execution with checkpointing
- `internal/agent/effects/` - At-most-once tool execution ledger
- `internal/runtime/eino/` - Eino workflow integration
- `cmd/api/` - API server
- `cmd/worker/` - Background worker
