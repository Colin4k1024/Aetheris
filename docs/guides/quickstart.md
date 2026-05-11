# Quickstart

Get Aetheris running locally in under 5 minutes. No Docker, no PostgreSQL, no Redis — embedded mode uses local stores.

## Requirements

- Go 1.26.1+
- Git
- Python 3.8+ (for the SDK and demo examples)

---

## Step 1 — Clone and build

```bash
git clone https://github.com/Colin4k1024/Aetheris.git
cd Aetheris
make build
```

## Step 2 — Start a mock agent

In **terminal 1**, run a one-liner mock agent (pure Python stdlib):

```bash
python3 -c '
import json, sys
from http.server import BaseHTTPRequestHandler, HTTPServer

class Handler(BaseHTTPRequestHandler):
    def log_message(self, *a): pass  # silence logs
    def do_POST(self):
        n = int(self.headers.get("content-length", 0))
        payload = json.loads(self.rfile.read(n) or b"{}")
        body = json.dumps({
            "answer": "Echo: " + payload.get("message", ""),
            "final": True,
            "metadata": {},
        }).encode()
        self.send_response(200)
        self.send_header("content-type", "application/json")
        self.send_header("content-length", str(len(body)))
        self.end_headers()
        self.wfile.write(body)

print("Mock agent on :9001", flush=True)
HTTPServer(("127.0.0.1", 9001), Handler).serve_forever()
'
```

## Step 3 — Start Aetheris

In **terminal 2**:

```bash
make run-embedded
```

This starts API (`:8080`) + Worker in background using embedded SQLite. Verify:

```bash
curl http://localhost:8080/api/health
# → {"status":"ok"}
```

The embedded config (`configs/api.embedded.yaml`) already registers a `crash_demo_batch_processor` agent. To add your own agent, append to that file or create a copy.

Expected shape:

```json
{
  "status": "ok"
}
```

## 4. Submit A Job

Submit a message to the `quickstart_http` agent:

## Step 4 — Submit a job via REST

In **terminal 2** (or a new one):

```bash
curl -X POST http://localhost:8080/api/agents/crash_demo_batch_processor/message \
  -H "Content-Type: application/json" \
  -H "Idempotency-Key: quickstart-1" \
  -d '{"message":"Process my first durable job"}'
```

Response:

```json
{
  "status": "accepted",
  "agent_id": "crash_demo_batch_processor",
  "job_id": "job_abc123"
}
```

## Step 5 — Inspect the job

```bash
JOB_ID="job_abc123"   # replace with your actual job_id

# Status
curl http://localhost:8080/api/jobs/$JOB_ID

# Full event trace (audit log)
curl http://localhost:8080/api/jobs/$JOB_ID/trace
```

---

## Step 6 — Use the Python SDK (optional)

```bash
pip install aetheris
```

```python
from aetheris import AetherisClient

client = AetherisClient("http://localhost:8080")

# Submit — returns immediately
job = client.run("crash_demo_batch_processor", "Process my second durable job",
                 idempotency_key="sdk-quickstart-1")
print(f"Job: {job.id} | Status: {job.status.value}")

# Block until done (polls every 2s, timeout 60s)
result = job.wait(timeout=60)
print("Output:", result.output)
```

---

## Step 7 — Connect your LangChain agent (optional)

```bash
pip install aetheris[langchain] langchain-openai
```

```python
from langchain_openai import ChatOpenAI
from langchain.agents import create_react_agent, AgentExecutor
from langchain import hub
from aetheris.integrations.langchain import serve

llm = ChatOpenAI(model="gpt-4o-mini")
agent = create_react_agent(llm, tools=[], prompt=hub.pull("hwchase17/react"))
executor = AgentExecutor(agent=agent, tools=[])

serve(executor, port=9000)   # makes your agent durable; blocks until Ctrl+C
```

Register in `configs/api.embedded.yaml`:

```yaml
agents:
  agents:
    my_langchain_agent:
      type: "external_http"
      external:
        url: "http://localhost:9000"
        timeout: "120s"
```

Full guide: [../adapters/langchain.md](../adapters/langchain.md)

---

## Step 8 — Stop Aetheris

```bash
make stop-embedded
```

---

## Next steps

| Goal | Resource |
|------|----------|
| Crash recovery demo | [examples/crash_recovery/](../../examples/crash_recovery/) |
| LangChain integration | [../adapters/langchain.md](../adapters/langchain.md) |
| Connect any HTTP agent | [../adapters/external-http-agent.md](../adapters/external-http-agent.md) |
| Runtime guarantees | [runtime-guarantees.md](runtime-guarantees.md) |
| Human-in-the-loop | [../concepts/human-in-the-loop.md](../concepts/human-in-the-loop.md) |
| Docker Compose deployment | [deployment.md](deployment.md) |

