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

## Usage

Add the framework as a node in your TaskGraph:

```json
{
  "id": "my-agent",
  "type": "langchaingo",
  "config": {
    "model": "gpt-4",
    "tools": ["tool1", "tool2"],
    "agent_type": "zeroShotReactDescription"
  }
}
```

## Client Interface

Each framework adapter requires a client implementation that creates the agent:

```go
type LangChainGoClient interface {
    CreateAgent(ctx context.Context, config map[string]any) (agents.Agent, []tools.Tool, error)
    GetLLM(ctx context.Context, config map[string]any) (llms.LLM, error)
}
```

## Configuration

The `config` field in TaskGraph node supports framework-specific options:

- `model`: LLM model to use
- `temperature`: Model temperature
- `max_tokens`: Maximum tokens
- Framework-specific options

## Error Handling

Adapters map framework errors to Aetheris error types:

- `StepResultRetryableFailure`: Transient errors that can be retried
- `StepResultPermanentFailure`: Permanent errors that should not be retried
- `SignalWaitRequired`: For async operations that need to wait

## Checkpoint & Replay

All framework adapters support:

- EffectStore for at-most-once execution guarantees
- Checkpoint-based replay
- Event sourcing for audit trails
