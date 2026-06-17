# Customer Support Agent Example

This example is a real `external_http` agent for Aetheris. It is not an echo
server: it accepts customer support messages, extracts an order id, looks up a
small local order index, applies refund/replacement policy, creates a support
ticket, and stores an idempotency ledger so retries do not create duplicate
cases.

It demonstrates the Level 1 migration path for an existing business agent:
Aetheris owns the durable outer Job, event log, trace, and `external_agent_call`
record. The customer support service still owns its internal business logic and
its own idempotency ledger.

## What It Proves

| Check | Result |
| --- | --- |
| Agent policy unit tests | `PASS` |
| Direct HTTP idempotency | Same `Idempotency-Key` reuses the same support ticket |
| Aetheris job execution | Job reaches `completed` through `external_agent_call` |
| Event and trace visibility | Events and trace JSON are captured in `artifacts/` |

## Run

From the repository root:

```bash
python3 -m unittest discover -s examples/customer_support_agent
python3 examples/customer_support_agent/demo.py
python3 examples/customer_support_agent/render_gifs.py
```

The demo starts:

- `customer_support_agent` on `127.0.0.1:19002`
- Aetheris API on `127.0.0.1:18080`
- Aetheris Worker with metrics on `127.0.0.1:19093`

All runtime config is generated under `.tmp/` and all proof material is written
under `artifacts/`.

## GIF Test Results

### 1. Policy And Idempotency Unit Tests

![Customer support agent unit tests](artifacts/customer-support-agent-unit-tests.gif)

### 2. Direct Agent Idempotency

![Customer support agent direct idempotency](artifacts/customer-support-agent-direct-idempotency.gif)

### 3. Aetheris Job, Events, And Trace

![Customer support agent Aetheris trace](artifacts/customer-support-agent-aetheris-trace.gif)

## Evidence Files

The generated evidence files are intentionally checked in with the example so
reviewers can inspect the exact run behind the GIFs.

| File | Purpose |
| --- | --- |
| `artifacts/unit-tests.log` | Unit test output used by the first GIF |
| `artifacts/direct-idempotency.log` | Direct `/invoke` duplicate-call proof |
| `artifacts/aetheris-job.log` | Aetheris job polling and final summary |
| `artifacts/job.json` | Final Aetheris Job API response |
| `artifacts/events.json` | Full event stream |
| `artifacts/trace.json` | Trace API response |
| `artifacts/agent-ledger.json` | External agent idempotency ledger |

## Agent Contract

The agent implements the standard JSON `external_http` protocol:

```json
{
  "message": "Customer says order A-1042 arrived damaged...",
  "session_id": "optional",
  "metadata": {
    "agent_id": "customer_support_agent",
    "job_id": "job-...",
    "idempotency_key": "..."
  }
}
```

It returns:

```json
{
  "answer": "Ticket CS-...: ...",
  "final": true,
  "metadata": {
    "ticket_id": "CS-...",
    "action": "approve_replacement_and_refund",
    "tools_used": ["extract_order", "lookup_order", "evaluate_policy", "record_case", "draft_customer_reply"]
  }
}
```

## Boundary

This example proves the black-box `external_http` boundary: Aetheris records the
outer call and trace, while the external service owns its internal ledger. For
strong at-most-once guarantees on high-risk side effects such as refunds,
payments, or email sends, move those actions into native Aetheris Runtime Tools.
