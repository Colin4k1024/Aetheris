# Adapter Index

This page compares available migration adapters and helps you choose one.

| Adapter                                             | Best for                                            | Effort     | Checkpoint granularity              | Status    |
| --------------------------------------------------- | --------------------------------------------------- | ---------- | ----------------------------------- | --------- |
| [Custom Agent Adapter](custom-agent.md)             | Existing imperative/custom agents                   | Low-Medium | Step-level (TaskGraph-based)        | ✅ Stable |
| [Custom Node Registration](custom-nodes.md)         | Extending TaskGraph with built-in/custom node types | Low        | Step-level (adapter-based)          | ✅ Stable |
| [Eino Examples](eino-examples.md)                    | cloudwego/eino-examples patterns (ReAct, DEER, etc.) | Low     | Step-level                          | ✅ Stable |

## Selection guide

- Pick **Custom Agent Adapter** when your current agent logic is framework-neutral and you can extract tools/planner directly.
- Pick **Eino Examples Adapter** for cloudwego/eino patterns (ReAct, DEER-Go, Manus, Chain, Graph, Workflow) - especially useful when you want to use local LLMs via Ollama.

## Common requirements

Regardless of adapter:

1. External side effects must go through Aetheris Tool path.
2. Wait/signal must use Aetheris wait contract (`correlation_key`).
3. Replay determinism must be validated in staging before production rollout.

## Framework examples

See [examples/](../examples/) for complete working examples:

| Example                                                             | Description                               |
| ------------------------------------------------------------------- | ----------------------------------------- |
| [Human Approval](../examples/human_approval_agent/)                 | Approval workflows with human-in-the-loop |
| [Multi-Agent Collaboration](../examples/multi_agent_collaboration/) | Complex multi-agent systems               |
| [Eino Agent with Tools](../examples/eino_agent_with_tools/)          | ReAct agent with tools using Ollama       |
| [Eino Chain](../examples/eino_chain/)                              | Chain composition pattern                 |
| [Eino Stateful](../examples/eino_stateful/)                       | Stateful agent example                    |
