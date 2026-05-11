# External HTTP Batch Demo

This example shows the **reliability boundary** of the `external_http` adapter.

It runs a slow batch processor behind Aetheris, submits the work as a durable job, and then polls the job/event APIs while the external service finishes the batch. Because this path is a single black-box HTTP call, Aetheris tracks the **outer job** and its event trail, but it does **not** checkpoint or resume individual records inside the external service.

If you need true per-step checkpoint/replay semantics for each record, move that work into Aetheris Runtime Tools or a native workflow instead of a single `external_http` call.

## Requirements

- Aetheris running in embedded mode (`make run-embedded` from repo root)
- Python 3.9+
- `pip install aetheris` (or `pip install requests` if you only want the no-SDK demo path)

## Run it

**Terminal 1 — start Aetheris:**
```bash
# from repo root
make run-embedded
```

**Terminal 2 — run the demo:**
```bash
cd examples/crash_recovery
pip install aetheris
python demo.py
```

## How it works

The demo uses the `crash_demo_batch_processor` entry in `configs/api.embedded.yaml`.
The local Python process starts a tiny HTTP server on `:9001`, Aetheris sends one `external_http` invocation to that server, and the demo prints:

- local batch progress from inside the external service
- the Aetheris job status
- the number of events recorded for the job

This demonstrates what Aetheris sees for black-box agents today: durable submission, job state, retries, and event/trace visibility around the outer call.
