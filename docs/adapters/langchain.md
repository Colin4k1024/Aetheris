# LangChain + Aetheris — Durable Agent Execution

Make any LangChain agent crash-proof, idempotent, and auditable — without changing your agent code.

## The problem

LangChain agents are stateless by design. When something fails mid-run:

- The entire chain restarts from scratch
- Tool calls that already succeeded fire again (duplicate emails, duplicate charges)
- You lose the reasoning trace

Aetheris wraps your LangChain agent with a durable execution shell that handles all of this.

## How it works

```
                   ┌─────────────┐       HTTP
  Client SDK  ───▶  │   Aetheris  │ ─────────────▶  Your LangChain Agent
  (Python/REST)     │   Runtime   │ ◀─────────────  (any host, any process)
                   │             │
                   │ checkpoint  │
                   │ recovery    │
                   │ dedup       │
                   └─────────────┘
```

Aetheris calls your agent via HTTP. Your agent processes one message and returns an answer. Aetheris handles the durable shell around that outer call:

- Durable submission and job status tracking
- Retry/error handling for the outer `external_http` invocation
- At-most-once delivery for that invocation via the idempotency key
- Audit trail for the job, events, and trace APIs

For step-level checkpointing *inside* the work itself, move high-risk side effects or long-running substeps into Aetheris Runtime Tools or native workflows. Black-box `external_http` agents remain a single external call from the runtime's perspective.

## Installation

```bash
pip install aetheris[langchain]
# or if you already have langchain:
pip install aetheris
```

## Option 1: `serve()` — expose your agent over HTTP

The easiest integration. One line wraps your agent as an Aetheris-compatible HTTP server:

```python
# my_agent.py
from langchain_openai import ChatOpenAI
from langchain.agents import create_react_agent, AgentExecutor
from langchain import hub
from aetheris.integrations.langchain import serve

llm = ChatOpenAI(model="gpt-4o-mini")
prompt = hub.pull("hwchase17/react")

# Define tools
from langchain.tools import DuckDuckGoSearchRun
tools = [DuckDuckGoSearchRun()]

agent = create_react_agent(llm, tools, prompt)
executor = AgentExecutor(agent=agent, tools=tools, verbose=True)

# Make it durable — expose as Aetheris-compatible endpoint
serve(executor, port=9000)
```

Run your agent:

```bash
python my_agent.py
# [aetheris] LangChain agent listening on http://localhost:9000
```

Register it in Aetheris config:

```yaml
# configs/api.embedded.yaml (or api.yaml)
agents:
  agents:
    research_agent:
      type: "external_http"
      description: "LangChain ReAct research agent"
      external:
        url: "http://localhost:9000"
        timeout: "120s"
```

Submit jobs from Python:

```python
from aetheris import AetherisClient

client = AetherisClient("http://localhost:8080")
job = client.run("research_agent", "What are the latest developments in fusion energy?")
result = job.wait(timeout=300)
print(result.output)
```

## Option 2: `AetherisLangChainAdapter` — low-level control

For custom HTTP frameworks or when you need more control over request handling:

```python
from aetheris.integrations.langchain import AetherisLangChainAdapter

adapter = AetherisLangChainAdapter(
    executor,
    input_key="input",   # key used in runnable.invoke({...})
    output_key="output", # key to extract from the result dict
)

# In your HTTP handler:
result = adapter.invoke(request_body)  # request_body = Aetheris job envelope
# Returns: {"answer": "...", "final": True, "metadata": {...}}
```

## Option 3: LCEL chains

Works with any LangChain Runnable, not just AgentExecutor:

```python
from langchain_openai import ChatOpenAI
from langchain.prompts import ChatPromptTemplate
from aetheris.integrations.langchain import serve, AetherisLangChainAdapter

# LCEL chain
prompt = ChatPromptTemplate.from_messages([
    ("system", "You are a helpful assistant."),
    ("human", "{input}"),
])
chain = prompt | ChatOpenAI(model="gpt-4o-mini")

serve(chain, port=9000)
```

For chains that don't return dicts, the adapter converts the output via `str()`.

## Request/response format

Aetheris sends this envelope to your agent:

```json
{
  "message": "user goal or prompt",
  "session_id": "sess_abc123",
  "metadata": {
    "agent_id": "research_agent",
    "job_id": "job_xyz789",
    "idempotency_key": "key_..."
  }
}
```

Your agent should return:

```json
{
  "answer": "the agent's final response",
  "final": true,
  "metadata": {}
}
```

The `serve()` function handles all of this automatically. You only need the raw format if you're implementing a custom server.

## External HTTP boundary demo

Want to see the current `external_http` boundary in action? Run the demo:

```bash
# Terminal 1: start Aetheris
make run-embedded

# Terminal 2: start your LangChain agent
python my_agent.py

# Terminal 3: run the external_http boundary demo
python examples/crash_recovery/demo.py
```

## Audit trail

Every job execution is logged as an append-only event stream:

```bash
curl http://localhost:8080/api/jobs/{job_id}/events
curl http://localhost:8080/api/jobs/{job_id}/trace
```

`/events` returns the raw event stream, while `/trace` returns the higher-level narrative view with a `timeline` array and trace metadata.

## FAQ

**Q: Does my LangChain agent need to be idempotent?**  
A: For `external_http`, Aetheris handles idempotency for the outer invocation via the job-level ledger. Internal side effects inside your LangChain process are still your responsibility unless you move them into Aetheris Runtime Tools.

**Q: Can I use async LangChain chains?**  
A: The `serve()` integration uses a synchronous HTTP server. For async chains, call `asyncio.run(chain.ainvoke(...))` inside your handler, or implement a custom server using `AetherisLangChainAdapter` with an async framework like FastAPI.

**Q: What if my agent needs persistent memory?**  
A: `serve()` forwards the Aetheris `message` field to your runnable by default. If you also need `session_id` or envelope metadata for memory keys, use `AetherisLangChainAdapter` in a custom handler and pass those fields into your runnable explicitly.

**Q: Can I run multiple LangChain agents?**  
A: Yes. Run each on a different port and register each in the active runtime config loaded by the API/Worker processes (for example `configs/api.embedded.yaml` plus `configs/worker.embedded.yaml` in split mode):

```yaml
agents:
  agents:
    research_agent:
      type: "external_http"
      external:
        url: "http://localhost:9000"
    summarizer_agent:
      type: "external_http"
      external:
        url: "http://localhost:9001"
```

## What's next?

- [AutoGen adapter](./autogen.md) _(coming soon)_
- [CrewAI adapter](./crewai.md) _(coming soon)_
- [Full API reference](../reference/api.md)
- [Crash recovery deep dive](../concepts/crash-recovery.md)
