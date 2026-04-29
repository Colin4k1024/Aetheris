# Aetheris Python SDK

Python client for the [Aetheris](https://github.com/Colin4k1024/Aetheris) durable agent execution runtime.

Aetheris gives your AI agents **at-most-once tool execution**, **crash recovery**, and **human-in-the-loop** support — this SDK lets you submit and monitor jobs from any Python application.

## Installation

```bash
pip install aetheris[requests]   # use requests (recommended)
# or
pip install aetheris[httpx]      # use httpx (async-compatible)
```

## Quick start

First, start the Aetheris server (embedded mode, no Docker needed):

```bash
git clone https://github.com/Colin4k1024/Aetheris
cd Aetheris
make build
CONFIG_PATH=configs/api.embedded.yaml ./bin/api
```

Then, in Python:

```python
from aetheris import AetherisClient

client = AetherisClient("http://localhost:8080")

# Submit a goal to an agent defined in configs/agents.yaml
job = client.run("my-agent", "Summarise the Q3 earnings report")
print(f"Job submitted: {job.id}  (status={job.status.value})")

# Block until it completes (polls every 2 s, timeout 5 min)
result = job.wait(timeout=120)
print("Output:", result.output)
```

## API reference

### `AetherisClient(base_url, *, token=None, timeout=30.0)`

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `base_url` | `str` | `"http://localhost:8080"` | Aetheris API server URL |
| `token` | `str \| None` | `None` | JWT bearer token (required when server auth is enabled) |
| `timeout` | `float` | `30.0` | HTTP request timeout in seconds |

#### `client.run(agent_id, message, *, idempotency_key=None) → Job`

Submit a message to an agent. Creates a durable job and returns immediately.

- **`agent_id`** — must match a key in `configs/agents.yaml`
- **`idempotency_key`** — re-submitting the same key returns the existing job without creating a duplicate (safe to retry on network error)

#### `client.get_job(job_id) → Job`

Fetch the current state of a job by ID.

#### `client.list_jobs(agent_id, *, status=None, limit=20) → list[Job]`

List jobs for a given agent. Optional `status` filter: `"pending"`, `"running"`, `"completed"`, `"failed"`, `"waiting"`.

#### `client.signal_job(job_id, payload, *, correlation_key="") → None`

Send a signal to a **WAITING** (human-in-the-loop) job.

#### `client.health() → bool`

Returns `True` if the server is reachable and healthy.

### `Job`

| Attribute | Type | Description |
|-----------|------|-------------|
| `id` | `str` | Job ID |
| `agent_id` | `str` | Agent identifier |
| `status` | `JobStatus` | Current status |
| `goal` | `str` | The original message |
| `output` | `Any` | Output payload when `status == COMPLETED` |
| `raw` | `dict` | Full raw API response |

#### `job.wait(*, timeout=300.0, poll_interval=2.0) → Job`

Block until the job reaches a terminal state. Raises `JobFailedError` on failure, `TimeoutError` if the deadline is exceeded.

#### `job.signal(payload, *, correlation_key="") → None`

Shorthand for `client.signal_job(job.id, payload, ...)`.

### Exceptions

| Exception | When raised |
|-----------|-------------|
| `AetherisError` | Base exception for all SDK errors |
| `JobFailedError` | Job ended in `failed` or `cancelled` state |
| `TimeoutError` | `job.wait()` exceeded the timeout |

## Human-in-the-loop example

```python
from aetheris import AetherisClient, JobStatus

client = AetherisClient("http://localhost:8080")

job = client.run("refund-agent", "Process refund for order #12345")

# Poll manually to detect the waiting state
while not job.is_terminal:
    job = client.get_job(job.id)
    if job.is_waiting:
        print("Agent is waiting for human approval…")
        # Approve the action
        job.signal({"approved": True, "reviewer": "alice@example.com"})
    else:
        import time; time.sleep(2)

print("Final status:", job.status.value)
```

## Idempotent submission

```python
# Safe to call multiple times — second call returns the existing job
job = client.run(
    "invoice-agent",
    "Generate invoice for customer C-999",
    idempotency_key="invoice-C-999-2024-Q4",
)
```

## Integration with LangChain / AutoGen

Use Aetheris as a durable execution layer underneath your orchestration framework:

```python
# LangChain tool example
from langchain.tools import tool
from aetheris import AetherisClient

_client = AetherisClient("http://localhost:8080")

@tool
def run_durable_agent(agent_id: str, goal: str) -> str:
    """Run an Aetheris agent with at-most-once execution guarantees."""
    job = _client.run(agent_id, goal)
    result = job.wait(timeout=300)
    return str(result.output)
```

## Development

```bash
git clone https://github.com/Colin4k1024/Aetheris
cd Aetheris/sdk/python
pip install -e ".[dev]"
pytest
```

## License

Apache 2.0 — see [LICENSE](../../LICENSE).
