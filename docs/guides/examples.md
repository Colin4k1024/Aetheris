# Examples

This page helps you choose the right example after the main quickstart.

If this is your first time running Aetheris, start with [quickstart.md](quickstart.md), not the examples directory. The examples are for learning specific patterns after you have already submitted one job and inspected its trace.

## Recommended After Quickstart

| Example | Use it for | Run shape |
| ------- | ---------- | --------- |
| [workflow](../../examples/workflow/) | Pure Eino DAG workflow without an agent | `go run ./examples/workflow` |
| [basic_agent](../../examples/basic_agent/) | Minimal Eino ChatModelAgent | Requires `OPENAI_API_KEY` |
| [eino_agent_with_tools](../../examples/eino_agent_with_tools/) | Eino agent calling tools | Requires `OPENAI_API_KEY` |
| [human_approval_agent](../../examples/human_approval_agent/) | Human-in-the-loop approval flow | Requires model configuration |

## Compatibility / Legacy

These remain useful for understanding older or lower-level APIs, but they are not the preferred onboarding path.

| Example | Notes |
| ------- | ----- |
| [simple_chat_agent](../../examples/simple_chat_agent/) | Uses the older in-process Aetheris agent API. |
| [sdk_agent](../../examples/sdk_agent/) | Demonstrates SDK-style usage rather than the runtime-first HTTP path. |

## Advanced / Experimental Patterns

Use these when you are exploring patterns, not when trying to validate the core runtime.

| Example | Pattern |
| ------- | ------- |
| [streaming](../../examples/streaming/) | Streaming response handling |
| [tool](../../examples/tool/) | Standalone tool-calling agent |
| [skill_agent](../../examples/skill_agent/) | Skill discovery and execution |
| [supervisor_agent](../../examples/supervisor_agent/) | Supervisor and sub-agent coordination |
| [plan_execute_agent](../../examples/plan_execute_agent/) | Plan-then-execute agent loop |
| [multi_agent_collaboration](../../examples/multi_agent_collaboration/) | Multi-agent collaboration |
| [mcp-gateway](../../examples/mcp-gateway/) | MCP gateway integration |
| [ai-customer-bot](../../examples/ai-customer-bot/) | Larger customer-service scenario |
| [ollama](../../examples/ollama/) | Local model setup experiments |

## API-Based Flow

Most examples are standalone Go programs. They do not require `go run ./cmd/api`, and they do not create Aetheris jobs over HTTP.

For the runtime path that creates jobs and traces, use:

- [quickstart.md](quickstart.md)
- [../adapters/external-http-agent.md](../adapters/external-http-agent.md)
- [../reference/api.md](../reference/api.md)
