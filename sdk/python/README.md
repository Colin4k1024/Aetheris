# Aetheris Python SDK

Give your AI agents **crash recovery**, **at-most-once tool execution**, and **human-in-the-loop** support — without changing a line of your agent code.

```bash
pip install aetheris
```

---

## The problem this solves

Your LangChain / AutoGen / any Python agent fails halfway through a task. What happens?

- **Without Aetheris:** Start over. Re-run every LLM call. Risk duplicate API calls.  
- **With Aetheris:** Resume from the last checkpoint. Zero duplicates. Zero wasted tokens.

---

## Quickstart

**Step 1:** Start Aetheris (no Docker needed):

```bash
git clone https://github.com/Colin4k1024/Aetheris
cd Aetheris && make run-embedded
```

**Step 2:** Submit a job from Python:

```python
from aetheris import AetherisClient

client = AetherisClient("http://localhost:8080")
job = client.run("my-agent", "Summarize the Q3 earnings report")
result = job.wait(timeout=120)
print(result.output)
```

---

## LangChain integration

Make any LangChain agent durable in minutes:

```python
from langchain_openai import ChatOpenAI
from langchain.agents import create_react_agent, AgentExecutor
from langchain import hub
from aetheris.integrations.langchain import serve

# Build your agent as usual
llm = ChatOpenAI(model="gpt-4o-mini")
prompt = hub.pull("hwchase17/react")
agent = create_react_agent(llm, tools=[], prompt=prompt)
executor = AgentExecutor(agent=agent, tools=[])

# Expose it as a durable Aetheris endpoint — one line
serve(executor)  # listens on :9000, blocks until Ctrl+C
```

Then add it to your Aetheris config:

```yaml
# configs/api.embedded.yaml
agents:
  my_langchain_agent:
    type: "external_http"
    external:
      url: "http://localhost:9000"
      timeout: "120s"
```

Submit and monitor from Python:

```python
from aetheris import AetherisClient

client = AetherisClient()
job = client.run("my_langchain_agent", "Explain quantum entanglement simply")
print(job.wait().output)
```

---

## Human-in-the-loop

```python
from aetheris import AetherisClient

client = AetherisClient()
job = client.run("refund-agent", "Process refund for order #12345")

# The agent parks itself waiting for approval
while not job.is_terminal:
    job = client.get_job(job.id)
    if job.is_waiting:
        print("Waiting for human approval…")
        job.signal({"approved": True, "reviewer": "alice@example.com"})
        break
    import time; time.sleep(2)

result = job.wait()
print(result.output)
```

---

## Idempotent submission

Safe to call multiple times — returns the existing job, never creates a duplicate:

```python
job = client.run(
    "invoice-agent",
    "Generate invoice for customer C-999",
    idempotency_key="invoice-C-999-2024-Q4",   # stable key
)
```

---

## API reference

### `AetherisClient(base_url="http://localhost:8080", *, token=None, timeout=30.0)`

| Method | Description |
|--------|-------------|
| `client.run(agent_id, message, *, idempotency_key=None) → Job` | Submit a message; returns immediately |
| `client.get_job(job_id) → Job` | Fetch current job state |
| `client.list_jobs(agent_id, *, status=None, limit=20) → list[Job]` | List jobs for an agent |
| `client.signal_job(job_id, payload, *, correlation_key="")` | Resume a waiting job |
| `client.health() → bool` | Server reachability check |

### `Job`

| Attribute | Description |
|-----------|-------------|
| `job.id` | Job ID |
| `job.status` | `JobStatus` enum: `pending / running / completed / failed / waiting` |
| `job.output` | Output when completed |
| `job.is_terminal` | `True` when status is completed/failed/cancelled |
| `job.is_waiting` | `True` when parked for human input |
| `job.wait(timeout=300, poll_interval=2)` | Block until terminal; raises on failure |
| `job.signal(payload, *, correlation_key="")` | Resume from waiting state |

### Exceptions

| Exception | When raised |
|-----------|-------------|
| `AetherisError` | Base SDK exception |
| `JobFailedError` | Job ended in failed/cancelled state |
| `TimeoutError` | `job.wait()` exceeded the timeout |

---

## Installation options

```bash
pip install aetheris              # requests included (recommended)
pip install aetheris[httpx]       # use httpx instead
pip install aetheris[langchain]   # include langchain for integrations
```

---

## Resources

- [Aetheris GitHub](https://github.com/Colin4k1024/Aetheris)
- [Quickstart guide](https://github.com/Colin4k1024/Aetheris/blob/main/docs/guides/quickstart.md)
- [LangChain integration](https://github.com/Colin4k1024/Aetheris/blob/main/docs/adapters/langchain.md)
- [Crash recovery demo](https://github.com/Colin4k1024/Aetheris/blob/main/examples/crash_recovery/)
- [API reference](https://github.com/Colin4k1024/Aetheris/blob/main/docs/reference/api.md)
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
