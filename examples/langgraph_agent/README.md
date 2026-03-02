# LangGraph Adapter Example

This example demonstrates how to run a LangGraph-based agent on Aetheris while keeping runtime guarantees (durability, replay, at-most-once side effects).

## Overview

The LangGraph adapter allows you to:
- Run LangGraph flows inside Aetheris nodes
- Keep Aetheris guarantees for job lifecycle, wait/signal, replay, and audit
- Map LangGraph nodes to Aetheris steps for finer-grained control

## Architecture

```
┌─────────────────────────────────────────────────────────┐
│                     Aetheris                            │
│  ┌─────────────────────────────────────────────────┐   │
│  │  LangGraph Node Adapter                         │   │
│  │  ┌───────────────────────────────────────────┐   │   │
│  │  │  Your LangGraph Client (Black-box)       │   │   │
│  │  │  - invoke() / stream() / state()        │   │   │
│  │  └───────────────────────────────────────────┘   │   │
│  └─────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────┘
```

## Quick Start

### 1. Implement LangGraphClient

```go
type MyLangGraphClient struct {
    APIEndpoint string
    APIKey      string
}

func (c *MyLangGraphClient) Invoke(ctx context.Context, input map[string]any) (map[string]any, error) {
    // Call your LangGraph API
    return map[string]any{"result": "ok"}, nil
}

func (c *MyLangGraphClient) Stream(ctx context.Context, input map[string]any, onChunk func(chunk map[string]any) error) error {
    return nil
}

func (c *MyLangGraphClient) State(ctx context.Context, threadID string) (map[string]any, error) {
    return map[string]any{"thread_id": threadID}, nil
}
```

### 2. Create Adapter

```go
adapter := &agentexec.LangGraphNodeAdapter{
    Client:      &MyLangGraphClient{},
    EffectStore: agentexec.NewEffectStorePg(db), // Production: use PostgreSQL
}
```

### 3. Build TaskGraph

```go
taskGraph := &planner.TaskGraph{
    Nodes: []planner.TaskNode{
        {ID: "lg_invoke", Type: planner.NodeLangGraph},
        {ID: "wait_approval", Type: planner.NodeWait, Config: map[string]any{
            "wait_kind":       "signal",
            "correlation_key": "approval-123",
        }},
    },
    Edges: []planner.TaskEdge{
        {From: "lg_invoke", To: "wait_approval"},
    },
}
```

## Key Features

### Error Mapping

The adapter maps LangGraph errors to Aetheris semantics:

| LangGraph Error | Aetheris Result |
|-----------------|-----------------|
| `retryable` | `StepResultRetryableFailure` |
| `permanent` | `StepResultPermanentFailure` |
| `wait` + correlation_key | Signal wait (`job_waiting`) |

### Side Effects

All external side effects should go through Aetheris Tool path:

```go
// Instead of direct external calls in LangGraph
// Use Aetheris Tool adapter
{ID: "send_email", Type: planner.NodeTool, Config: map[string]any{
    "tool_name": "email_tool",
}}
```

### Human-in-the-Loop

Replace ad-hoc human approval in LangGraph with Aetheris wait:

```go
// Return error with wait code
return nil, &LangGraphError{
    Code:           LangGraphErrorWait,
    CorrelationKey: "approval-123",
    Message:        "需要人工审批",
}

// Resume via API
// curl -X POST http://localhost:8080/api/jobs/{job_id}/signal \
//   -d '{"correlation_key": "approval-123"}'
```

## Replay and Recovery

Aetheris provides full replay guarantees:

1. **Effect Store**: Records LangGraph outputs to prevent re-execution
2. **Event Stream**: Full audit trail of all operations
3. **Replay**: Deterministic replay from recorded events

```bash
# View events
curl http://localhost:8080/api/jobs/{job_id}/events

# Replay job
curl http://localhost:8080/api/jobs/{job_id}/replay

# Verify
aetheris verify --job-id {job_id}
```

## Production Considerations

1. **Effect Store**: Use PostgreSQL effect store for durability
2. **Secrets**: Use Vault or K8s secrets for API keys
3. **Monitoring**: Enable OpenTelemetry for tracing
4. **Rate Limiting**: Configure per-tool rate limits

## See Also

- [LangGraph Adapter Documentation](../../docs/adapters/langgraph.md)
- [Aetheris Getting Started](../../docs/getting-started-agents.md)
- [Runtime Guarantees](../../docs/runtime-guarantees.md)
