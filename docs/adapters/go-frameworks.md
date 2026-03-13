# Go Open-Source Framework Adapters

Aetheris provides adapters for integrating Go-based open-source agent frameworks as TaskGraph nodes. This allows you to leverage existing Go agent ecosystems while benefiting from Aetheris's execution guarantees.

## Supported Frameworks

| Framework | Node Type | Repository |
| --------- | --------- | ---------- |
| LangChainGo | `langchaingo` | [tmc/langchaingo](https://github.com/tmc/langchaingo) |
| LangGraphGo | `langgraphgo` | [smallnest/langgraphgo](https://github.com/smallnest/langgraphgo) |
| Google ADK | `adk` | [google/adk-go](https://github.com/google/adk-go) |
| Firebase Genkit | `genkit` | [firebase/genkit](https://github.com/firebase/genkit) |
| Protocol-Lattice | `protocol_lattice` | [Protocol-Lattice/go-agent](https://github.com/Protocol-Lattice/go-agent) |
| LinGoose | `lingoose` | [henomis/lingoose](https://github.com/henomis/lingoose) |
| Anyi | `anyi` | [jieliu2000/anyi](https://github.com/jieliu2000/anyi) |
| Agent SDK Go | `agent_sdk` | [timwhitez/agent-sdk-golang](https://github.com/timwhitez/agent-sdk-golang) |

## Architecture

Each adapter follows the same pattern:

```
TaskGraph Node → Adapter → Framework Agent → Result
                ↓
         EffectStore (for replay)
                ↓
         CommandEventSink (for events)
```

## Usage in TaskGraph

Add the framework as a node in your TaskGraph:

### LangChainGo

```json
{
  "id": "my-agent",
  "type": "langchaingo",
  "config": {
    "model": "gpt-4",
    "temperature": 0.7,
    "agent_type": "zeroShotReactDescription",
    "tools": ["search", "calculator"]
  }
}
```

### LangGraphGo

```json
{
  "id": "my-agent",
  "type": "langgraphgo",
  "config": {
    "model": "gpt-4",
    "graph_definition": "your-graph-json"
  }
}
```

### Google ADK

```json
{
  "id": "my-agent",
  "type": "adk",
  "config": {
    "model": "gemini-pro",
    "agent_name": "my-adk-agent"
  }
}
```

### Firebase Genkit

```json
{
  "id": "my-agent",
  "type": "genkit",
  "config": {
    "model": "vertex-ai",
    "flow_definition": "your-flow"
  }
}
```

### Protocol-Lattice

```json
{
  "id": "my-agent",
  "type": "protocol_lattice",
  "config": {
    "model": "gpt-4",
    "enable_memory": true
  }
}
```

### LinGoose

```json
{
  "id": "my-agent",
  "type": "lingoose",
  "config": {
    "model": "gpt-4",
    "agent_type": "react"
  }
}
```

### Anyi

```json
{
  "id": "my-agent",
  "type": "anyi",
  "config": {
    "model": "gpt-4",
    "autonomous": true
  }
}
```

### Agent SDK Go

```json
{
  "id": "my-agent",
  "type": "agent_sdk",
  "config": {
    "model": "gpt-4",
    "tools": ["browser", "file"]
  }
}
```

## Client Interface

Each framework adapter requires a client implementation. Here are the interfaces:

### LangChainGoClient

```go
type LangChainGoClient interface {
    CreateAgent(ctx context.Context, config map[string]any) (agents.Agent, []tools.Tool, error)
    GetLLM(ctx context.Context, config map[string]any) (llms.LLM, error)
}
```

### LangGraphGoClient

```go
type LangGraphGoClient interface {
    CreateAgent(ctx context.Context, config map[string]any) (interface {
        Invoke(ctx context.Context, input interface{}) (interface{}, error)
    }, error)
}
```

### ADKClient

```go
type ADKClient interface {
    CreateAgent(ctx context.Context, config map[string]any) (interface {
        Invoke(ctx context.Context, input string) (string, error)
    }, error)
}
```

## Configuration Options

All adapters support these common config options:

| Option | Type | Description |
|--------|------|-------------|
| `model` | string | LLM model to use |
| `temperature` | float | Model temperature (0.0-1.0) |
| `max_tokens` | int | Maximum tokens to generate |
| `timeout` | int | Request timeout in seconds |

Framework-specific options can be found in each framework's documentation.

## Error Handling

Adapters map framework errors to Aetheris error types:

- `StepResultRetryableFailure`: Transient errors that can be retried (network issues, rate limits)
- `StepResultPermanentFailure`: Permanent errors that should not be retried (invalid config, auth failures)
- `SignalWaitRequired`: For async operations that need to wait

## Checkpoint & Replay

All framework adapters support:

- **EffectStore** for at-most-once execution guarantees
- **Checkpoint-based replay** for crash recovery
- **Event sourcing** for audit trails
- **Step-level granularity** for fine-grained control

## Benefits of Using Adapters

1. **Full Execution Control**: All agent executions are controlled by Aetheris
2. **Replay Support**: Failed executions can be replayed from checkpoints
3. **Audit Trail**: Complete event history for compliance
4. **Multi-Framework**: Use the best framework for each use case
5. **Vendor Neutral**: Not locked into any single framework

## Choosing a Framework

- **LangChainGo**: Best for migrating from Python LangChain
- **LangGraphGo**: Best for graph-based state management
- **Google ADK**: Best for Google Cloud integration
- **Firebase Genkit**: Best for Firebase ecosystem
- **Protocol-Lattice**: Best for production multi-agent systems
- **LinGoose**: Best for lightweight applications
- **Anyi**: Best for autonomous agents
- **Agent SDK Go**: Best for minimal implementations
