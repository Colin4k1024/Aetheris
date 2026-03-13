# Building Flexible AI Agents with Aetheris Multi-Framework Support

*Date: 2026-03-13*

## Introduction

Aetheris now supports integrating 8 Go-based open-source agent frameworks directly into your agent runtime. This blog post explores how to leverage this multi-framework support to build more flexible and powerful AI agents.

## The Challenge of Framework Lock-in

Traditional agent platforms often force you to choose a single framework:
- LangChain for its extensive tooling
- LangGraph for state management
- Custom implementations for specific needs

Aetheris breaks this constraint by providing adapters for multiple frameworks, allowing you to:
- Choose the best tool for each specific use case
- Migrate between frameworks without rewriting logic
- Combine multiple frameworks in a single agent workflow

## Supported Frameworks

Aetheris now supports:

1. **LangChainGo** - The Go implementation of LangChain
2. **LangGraphGo** - Graph-based state management
3. **Google ADK** - Google's Agent Development Kit
4. **Firebase Genkit** - Firebase's AI development framework
5. **Protocol-Lattice** - Production-ready graph-aware memory
6. **LinGoose** - Lightweight AI/LLM framework
7. **Anyi** - Autonomous AI agent framework
8. **Agent SDK Go** - Minimal agent SDK

## Architecture

Each framework adapter follows a consistent pattern:

```
TaskGraph Node → Adapter → Framework Agent
                      ↓
              EffectStore (replay)
                      ↓
              CommandEventSink (events)
```

This architecture ensures:
- **At-most-once execution**: No duplicate tool calls
- **Crash recovery**: Replay from checkpoints
- **Full audit trail**: Complete event history
- **Step-level control**: Fine-grained execution management

## Code Example

Using LangChainGo in your TaskGraph:

```json
{
  "id": "research-agent",
  "type": "langchaingo",
  "config": {
    "model": "gpt-4",
    "tools": ["search", "web_scraper", "calculator"]
  }
}
```

The adapter automatically handles:
- Tool execution through Aetheris
- Checkpoint creation
- Event emission
- Error mapping

## Real-World Use Cases

### Multi-Framework Workflow

Combine different frameworks in one agent:

```json
{
  "nodes": [
    {
      "id": "planner",
      "type": "langgraphgo",
      "config": {"model": "gpt-4"}
    },
    {
      "id": "executor",
      "type": "langchaingo",
      "config": {"model": "gpt-4", "tools": [...]}
    },
    {
      "id": "validator",
      "type": "protocol_lattice",
      "config": {"enable_memory": true}
    }
  ],
  "edges": [
    {"from": "planner", "to": "executor"},
    {"from": "executor", "to": "validator"}
  ]
}
```

### Framework Migration

Start with one framework, migrate gradually:

1. Deploy agent using LangChainGo
2. Add Protocol-Lattice nodes for memory
3. Migrate core logic to native eino
4. Maintain LangChainGo as fallback

## Performance Considerations

Each framework has different characteristics:

| Framework | Cold Start | Memory | Best For |
|-----------|-----------|--------|----------|
| LangChainGo | Medium | Medium | Tool-rich agents |
| LangGraphGo | Medium | High | Complex state |
| Google ADK | High | High | Enterprise |
| Protocol-Lattice | Medium | High | Production |

## Conclusion

Aetheris's multi-framework support provides unprecedented flexibility for building AI agents. By decoupling the execution platform from the agent framework, you can:

- Choose the best tool for each job
- Avoid vendor lock-in
- Leverage the Go ecosystem
- Maintain execution guarantees

Try it out and let us know which framework works best for your use cases!
