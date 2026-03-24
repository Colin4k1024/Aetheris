# Aetheris v2.5.0 Release Notes

**Release Date:** 2026-03-24

---

## Highlights

Aetheris v2.5.0 introduces the **At-Most-Once Atomic Commit Protocol Phase 1** — closing the crash window between tool execution success and Ledger commit to prevent double execution in production multi-worker deployments.

---

## What's New

### At-Most-Once Atomic Commit Protocol (Phase 1)

- **LedgerAcquired event**: Written to the event store before tool execution begins, providing a durable marker that a tool execution is in progress
- **LedgerCommitted event**: Written after successful Ledger commit, completing the atomic sequence
- **Orphaned acquisition detection**: On replay, `Acquire()` detects orphaned `ledger_acquired` events (no corresponding `ledger_committed`) and returns `WaitOtherWorker` — preventing double execution after a crash
- **LedgerEventSink interface**: New pluggable interface for writing ledger events to the event store
- **AtomicCommit protocol**: `DefaultAtomicCommit()` function that sequences: `ledger_acquired` → tool execution → `Ledger.Commit()` → `ledger_committed`

### New Event Types

| Event Type | Description |
|------------|-------------|
| `ledger_acquired` | Execution permit granted, tool not yet executed |
| `ledger_committed` | Result committed, tool execution complete |

### Crash Recovery Protocol

```
Crash window: ledger_acquired written → tool executed → ???

Recovery:
  1. Acquire() checks for orphaned ledger_acquired (no ledger_committed)
  2. If orphaned found → WaitOtherWorker (in-progress, don't execute)
  3. If committed=true in store → ReturnRecordedResult (already done)
  4. If no record → AllowExecute (safe to proceed)
```

### Validation

- `make fmt-check` ✓
- `make vet` ✓
- `make test` (full suite with race detector) ✓
- `make build` ✓
- `TestAtomicCommit_CrashRecovery` ✓
- `TestAtomicCommit_OrphanedAcquired` ✓
- `TestAtomicCommit_StoreStartedBeforeAcquired` ✓

---

## Installation

### Binary

Download from [GitHub Releases](https://github.com/Colin4k1024/Aetheris/releases)

### From Source

```bash
git clone https://github.com/Colin4k1024/Aetheris
cd Aetheris
make build
```

### Docker

```bash
docker pull aetheris/runtime:v2.5.0
```
