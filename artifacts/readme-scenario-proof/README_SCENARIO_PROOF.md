# README Scenario Proof

Generated on 2026-06-17.

## What Was Run

1. README quickstart health path
   - Built `bin/api`, `bin/worker`, and `bin/aetheris`.
   - Started API and Worker with embedded configs using a clean local data dir.
   - Verified `GET /api/health`.

2. README `external_http` batch demo
   - Ran `examples/crash_recovery/demo.py`.
   - Used `AGENT_PORT=19001` because local port `9001` was occupied by a system service on this machine.
   - Submitted one durable job to `crash_demo_batch_processor`.

3. Audit/trace proof
   - Fetched `/api/jobs/<job-id>`.
   - Fetched `/api/jobs/<job-id>/events`.
   - Fetched `/api/jobs/<job-id>/trace`.

## Result

- Job ID: `job-20094b04-a0b3-4f39-9ea7-d367fa3b0094`
- Job status: `completed`
- Local records processed by the external agent: `20`
- Recorded event count: `16`
- Trace node duration: `4076 ms`
- Embedded evidence files created:
  - `data/embedded/jobs.json`
  - `data/embedded/job_events.json`
  - `data/embedded/checkpoints.json`
  - `data/embedded/tool_invocations.json`
  - `data/embedded/effects.json`
  - `data/embedded/agent_state.json`

## GIF Deliverables

- `readme-quickstart-health.gif`
- `readme-external-http-batch-demo.gif`
- `readme-event-trace-proof.gif`

## Raw Evidence

- `health-demo.json`
- `job.json`
- `events.json`
- `trace.json`
- `crash_recovery_demo_success.log`
- `api.demo.log`
- `worker.demo.log`

## Boundary Note

The README itself states that the `external_http` example demonstrates the outer-call durability boundary. This proof confirms durable submission, job state transition, event capture, tool invocation evidence, state checkpoint, and trace visibility around the black-box HTTP call. It does not claim per-record checkpoint/resume inside the external Python service.
