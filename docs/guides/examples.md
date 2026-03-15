# Examples

This page briefly describes each example under [examples/](../examples/) and how to run it.

## basic_agent

Uses **eino-ext** OpenAI ChatModel and **ChatModelAgent** to run one in-process conversation. Demonstrates eino ADK agent usage.

**Run**: Set `OPENAI_API_KEY`.

```bash
OPENAI_API_KEY=sk-xxx go run ./examples/basic_agent
```

Does not require a running API server.

---

## simple_chat_agent

Minimal Aetheris agent example: no HTTP, calls `pkg/agent` `Agent.Run` for one turn. Shows checkpointed execution in the Aetheris runtime.

**Run**:

```bash
OPENAI_API_KEY=sk-xxx go run ./examples/simple_chat_agent
```

Does not require a running API server.

---

## streaming

Uses eino ChatModel + ChatModelAgent with **streaming** output. Demonstrates streaming response handling.

**Run**: Set `OPENAI_API_KEY`.

```bash
OPENAI_API_KEY=sk-xxx go run ./examples/streaming
```

Does not require a running API server.

---

## tool

Registers **tools** in eino; the agent calls them during the conversation. Shows how to define tools and have the agent use them.

**Run**: Set `OPENAI_API_KEY`.

```bash
OPENAI_API_KEY=sk-xxx go run ./examples/tool
```

Does not require a running API server.

---

## workflow

Uses **eino compose** to build a DAG (Graph), define input/output types and nodes, and run it. Demonstrates a pure DAG workflow (no agent).

**Run**:

```bash
go run ./examples/workflow
```

Does not require the API or external model services.

---

## Using with the API

These examples are **standalone processes** using eino / pkg/agent and do not need `go run ./cmd/api`. To create agents, send messages, and query jobs over HTTP, use the flows in [usage.md](usage.md) or the CLI ([cli.md](cli.md)) and start the API first (default http://localhost:8080).

---

## skill_agent

Demonstrates **eino Skill capability** - a folder containing instructions, scripts, and resources that Agents can discover and use on-demand.

Features:
- Skill directory structure with SKILL.md
- Progressive loading mechanism (Discovery → Activation → Execution)
- Three context modes: inline, fork, isolate
- YAML frontmatter parsing for skill metadata

**Run**:

```bash
go run ./examples/skill_agent
```

See [examples/skill_agent/README.md](../../examples/skill_agent/README.md) for details.

---

## supervisor_agent

Demonstrates the **Supervisor Agent** pattern - a multi-agent architecture where a supervisory agent coordinates specialized sub-agents.

Architecture:
- Supervisor analyzes requests and delegates to appropriate sub-agents
- Sub-agents: Researcher, Writer, Coder (extensible)
- Task delegation workflow

**Run**:

```bash
go run ./examples/supervisor_agent
```

See [examples/supervisor_agent/README.md](../../examples/supervisor_agent/README.md) for details.

---

## plan_execute_agent

Demonstrates the **Plan-Execute Agent** pattern - a two-phase agent that plans before executing.

Features:
- Plan Phase: Analyze task and create execution plan
- Execute Phase: Execute steps according to plan
- Dynamic plan adjustment during execution
- Step-by-step progress tracking

**Run**:

```bash
go run ./examples/plan_execute_agent
```

See [examples/plan_execute_agent/README.md](../../examples/plan_execute_agent/README.md) for details.
