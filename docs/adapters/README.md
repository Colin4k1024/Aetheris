# Adapter Index

This page compares available agent adapters and helps you choose one.

## Built-in Adapters

| Adapter | Best for | Effort | Checkpoint granularity | Status |
| ------- | -------- | ------ | ---------------------- | ------ |
| [External HTTP Agent](external-http-agent.md) | Existing HTTP-service agents in any language (JSON or SSE-streaming) | Low | Job/tool-call level | ✅ Stable |
| [LangChain/LangGraph Python](langchain.md) | Existing Python LangChain agents and LangGraph compiled graphs | Low | Job/tool-call level or manifest node-level | ✅ Stable |
| [Custom Agent Adapter](custom-agent.md) | Existing imperative/custom agents | Low-Medium | Step-level (TaskGraph-based) | ✅ Stable |
| [Custom Node Registration](custom-nodes.md) | Extending TaskGraph with built-in/custom node types | Low | Step-level (adapter-based) | ✅ Stable |
| [Eino Examples](eino-examples.md) | cloudwego/eino-examples patterns (ReAct, DEER, etc.) | Low | Step-level | ✅ Stable |

## Go Open-Source Framework Adapters

Aetheris supports integrating Go-based open-source agent frameworks directly as TaskGraph nodes:

| Framework | Node Type | Description | Status |
| --------- | --------- | ----------- | ------ |
| LangChainGo | `langchaingo` | LangChain for Go implementation | ✅ Stable |
| LangGraphGo | `langgraphgo` | LangGraph for Go implementation | ✅ Stable |
| Google ADK | `adk` | Google Agent Development Kit | ✅ Stable |
| Firebase Genkit | `genkit` | Firebase Genkit (Go) | ✅ Stable |
| Protocol-Lattice | `protocol_lattice` | Graph-aware memory agent framework | ✅ Stable |
| LinGoose | `lingoose` | AI/LLM application framework | ✅ Stable |
| Anyi | `anyi` | Autonomous AI agent framework | ✅ Stable |
| Agent SDK Go | `agent_sdk` | Minimal agent SDK | ✅ Stable |

## Selection guide

- Pick **Eino Examples Adapter** for cloudwego/eino patterns (ReAct, DEER-Go, Manus, Chain, Graph, Workflow) - especially useful when you want to use local LLMs via Ollama.
- Pick **External HTTP Agent** when the user already has a running agent service and needs the lowest-cost migration path. Use `protocol: "sse_legacy"` for SSE-streaming services such as [superagent-base](https://github.com/Colin4k1024/superagent-base).
- Pick **LangChain/LangGraph Python** when the user already has a Python framework agent and wants to register it as `type: "langchain"` or `type: "langgraph"` in AgentRuntime. Use default black-box mode for low-cost migration; use `external.mode: "embedded"` plus an explicit manifest when Aetheris should own the internal execution graph.
- Pick **LangChainGo** if you want to use LangChain patterns in Go.
- Pick **LangGraphGo** if you want to use LangGraph state management patterns.
- Pick **Google ADK** if you want to use Google's agent development patterns.
- Pick **Firebase Genkit** if you want to use Firebase's AI development patterns.
- Pick **Protocol-Lattice** for production-ready graph-aware memory and multi-agent orchestration.
- Pick **LinGoose** for simple and lightweight AI/LLM applications.
- Pick **Anyi** for autonomous AI agent workflows.
- Pick **Agent SDK Go** for minimal agent SDK patterns.

## Common requirements

Regardless of adapter:

1. External side effects must go through Aetheris Tool path.
2. Wait/signal must use Aetheris wait contract (`correlation_key`).
3. Replay determinism must be validated in staging before production rollout.

## Framework examples

See [examples/](../examples/) for complete working examples:

| Example | Description |
| ------- | ----------- |
| [Human Approval](../examples/human_approval_agent/) | Approval workflows with human-in-the-loop |
| [Multi-Agent Collaboration](../examples/multi_agent_collaboration/) | Complex multi-agent systems |
| [Eino Agent with Tools](../examples/eino_agent_with_tools/) | ReAct agent with tools using Ollama |
| [Eino Chain](../examples/eino_chain/) | Chain composition pattern |
| [Eino Stateful](../examples/eino_stateful/) | Stateful agent example |
| [superagent-base](external-http-agent.md#superagent-base-integration) | SSE-streaming open-source agent platform via `sse_legacy` protocol |
