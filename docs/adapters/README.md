# Adapter Index

This page compares available migration adapters and helps you choose one.

| Adapter                                             | Best for                                            | Effort     | Checkpoint granularity              | Status    |
| --------------------------------------------------- | --------------------------------------------------- | ---------- | ----------------------------------- | --------- |
| [Custom Agent Adapter](custom-agent.md)             | Existing imperative/custom agents                   | Low-Medium | Step-level (TaskGraph-based)        | ✅ Stable |
| [LangGraph Adapter](langgraph.md)                   | Existing LangGraph flows                            | Medium     | Bridge-level first, then step-level | ✅ Stable |
| [Custom Node Registration](custom-nodes.md)         | Extending TaskGraph with built-in/custom node types | Low        | Step-level (adapter-based)          | ✅ Stable |
| [LlamaIndex Adapter](../examples/llamaindex_agent/) | LlamaIndex agents                                   | Low        | Step-level                          | ✅ Stable |
| [Vertex AI Adapter](../examples/vertex_agent/)      | Google Vertex AI Agent Engine                       | Medium     | Step-level                          | ✅ Stable |
| [AWS Bedrock Adapter](../examples/bedrock_agent/)   | AWS Bedrock Agents                                  | Medium     | Step-level                          | ✅ Stable |
| [AgentScope Adapter](../examples/agentscope_agent/) | AgentScope multi-agent framework                    | Medium     | Step-level                          | ✅ Stable |

## Selection guide

- Pick **Custom Agent Adapter** when your current agent logic is framework-neutral and you can extract tools/planner directly.
- Pick **LangGraph Adapter** when you already rely on LangGraph state transitions and want staged migration to Aetheris runtime guarantees.
- Pick **LlamaIndex Adapter** for agents built with LlamaIndex framework.
- Pick **Vertex AI Adapter** for Google Cloud Vertex AI Agent Engine integration.
- Pick **AWS Bedrock Adapter** for AWS Bedrock Agents integration.
- Pick **AgentScope Adapter** for AgentScope multi-agent framework.

## Common requirements

Regardless of adapter:

1. External side effects must go through Aetheris Tool path.
2. Wait/signal must use Aetheris wait contract (`correlation_key`).
3. Replay determinism must be validated in staging before production rollout.

## Framework examples

See [examples/](../examples/) for complete working examples of each adapter:

| Example                                                             | Description                               |
| ------------------------------------------------------------------- | ----------------------------------------- |
| [LangGraph](../examples/langgraph-agent/)                           | Run LangGraph flows on Aetheris           |
| [LangGraph Complete](../examples/langgraph-complete/)               | Full LangGraph workflow example           |
| [AutoGen](../examples/autogen_agent/)                               | Microsoft AutoGen multi-agent support     |
| [CrewAI](../examples/crewai_agent/)                                 | CrewAI crew orchestration                 |
| [LlamaIndex](../examples/llamaindex_agent/)                         | LlamaIndex agent integration              |
| [Vertex AI](../examples/vertex_agent/)                              | Google Vertex AI Agent Engine             |
| [AWS Bedrock](../examples/bedrock_agent/)                           | AWS Bedrock Agents                        |
| [AgentScope](../examples/agentscope_agent/)                         | AgentScope multi-agent framework          |
| [Human Approval](../examples/human_approval_agent/)                 | Approval workflows with human-in-the-loop |
| [Multi-Agent Collaboration](../examples/multi_agent_collaboration/) | Complex multi-agent systems               |
