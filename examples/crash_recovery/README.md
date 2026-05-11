# Crash Recovery Demo

This demo shows Aetheris crash recovery in action.

A simulated agent processes 1,000 records one by one. You can kill the process at any point, restart it, and it will **resume from the last completed record** — not from the beginning.

## What you'll see

```
[Aetheris] Submitting batch job... (job_id: abc-123)
Processing record   1/1000  ✓
Processing record   2/1000  ✓
...
Processing record 847/1000  ✓
^C   <── kill here

[restart]
[Aetheris] Job abc-123 is RUNNING, last checkpoint: record 847
Resuming from record 848...
Processing record 848/1000  ✓
...
Processing record 1000/1000 ✓
[Aetheris] Done. 1000 records processed. 0 duplicates.
```

## Requirements

- Aetheris running in embedded mode (`make run-embedded` from repo root)
- Python 3.9+
- `pip install requests` (or `pip install aetheris[requests]` for the SDK)

## Run it

**Terminal 1 — start Aetheris:**
```bash
# from repo root
make run-embedded
```

**Terminal 2 — run the demo:**
```bash
cd examples/crash_recovery
pip install requests
python demo.py
```

Interrupt with `Ctrl+C` at any point, then restart:
```bash
python demo.py
```

The demo automatically detects and resumes in-progress jobs.

## How it works

The demo registers a lightweight HTTP agent with Aetheris (`batch_processor`).
Each record is processed as a separate tool invocation tracked by the Invocation Ledger.

When the process is killed:
- Aetheris detects the worker heartbeat has stopped
- The lease is released after the timeout
- The next worker (or restart) picks up the job from the last checkpoint
- Already-completed steps return cached results from the Ledger — no re-execution

This is the `at-most-once` guarantee in practice.
