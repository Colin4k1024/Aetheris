# AutoGen Adapter Example

This example demonstrates how to run Microsoft AutoGen agents on Aetheris runtime.

## Overview

Microsoft AutoGen is a framework for building multi-agent applications. This adapter allows you to:
- Run AutoGen conversations inside Aetheris
- Keep Aetheris guarantees for durability, replay, and audit
- Add human-in-the-loop capabilities to AutoGen workflows

## Architecture

```
┌─────────────────────────────────────────────────────────┐
│                     Aetheris                            │
│  ┌─────────────────────────────────────────────────┐   │
│  │  AutoGen Adapter                                │   │
│  │  ┌───────────────────────────────────────────┐   │   │
│  │  │  AutoGen Agents                           │   │   │
│  │  │  - GroupChat                              │   │   │
│  │  │  - ConversableAgent                       │   │   │
│  │  │  - AssistantAgent                         │   │   │
│  │  └───────────────────────────────────────────┘   │   │
│  └─────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────┘
```

## Quick Start

### 1. Implement AutoGenClient

```go
type AutoGenClient struct {
    Endpoint string
    APIKey   string
}

func (c *AutoGenClient) Invoke(ctx context.Context, input map[string]any) (map[string]any, error) {
    // Call your AutoGen service
    return map[string]any{
        "status":      "completed",
        "last_message": "Response from AutoGen",
    }, nil
}

func (c *AutoGenClient) Stream(ctx context.Context, input map[string]any, onChunk func(chunk map[string]any) error) error {
    return nil
}
```

### 2. Create TaskGraph with AutoGen

```go
taskGraph := &planner.TaskGraph{
    Nodes: []planner.TaskNode{
        {
            ID:   "autogen_chat",
            Type: planner.NodeAutoGen, // 需要创建新的节点类型
            Config: map[string]any{
                "agents":    []string{"assistant", "critic"},
                "max_turns": 10,
            },
        },
        {
            ID:   "human_review",
            Type: planner.NodeWait,
            Config: map[string]any{
                "wait_kind":       "signal",
                "correlation_key": "approval",
            },
        },
    },
    Edges: []planner.TaskEdge{
        {From: "autogen_chat", To: "human_review"},
    },
}
```

## Key Features

### Multi-Agent Collaboration

AutoGen excels at multi-agent scenarios:

```go
// 配置多个 agents 协作
config := AutoGenConfig{
    Agents:   []string{"planner", "coder", "critic"},
    MaxTurns: 15,
}
```

### Human-in-the-Loop

```go
// 需要人类输入时返回错误
return nil, &AutoGenError{
    Code:          AutoGenErrorNeedsInput,
    Message:       "需要人类审批",
    CorrelationID: "approval-123",
}

// 恢复执行
// curl -X POST http://localhost:8080/api/jobs/{job_id}/signal \
//   -d '{"correlation_key": "approval-123", "input": "approved"}'
```

### Error Mapping

| AutoGen Error | Aetheris Result |
|---------------|-----------------|
| `needs_input` | Signal wait for human input |
| `retryable` | Retryable failure |
| `permanent` | Permanent failure |

## Migration Patterns

### Pattern 1: Black-box (Recommended for Start)

Run entire AutoGen conversation as one Aetheris node:

```go
{ID: "autogen", Type: planner.NodeLangGraph}  // 复用
```

### Pattern 2: Node-by-node

Split AutoGen agents into individual Aetheris steps:

```go
Nodes: []planner.TaskNode{
    {ID: "planner", Type: planner.NodeTool},   // Agent 1
    {ID: "coder", Type: planner.NodeTool},     // Agent 2
    {ID: "critic", Type: planner.NodeTool},    // Agent 3
}
```

### Pattern 3: Hybrid

Combine both approaches:

```go
Nodes: []planner.TaskNode{
    {ID: "quick_reply", Type: planner.NodeLangGraph},  // 简单场景
    {ID: "complex_reasoning", Type: planner.NodeTool},  // 复杂场景单独处理
}
```

## Production Considerations

1. **AutoGen Service**: Deploy AutoGen as separate service
2. **Effect Store**: Use PostgreSQL for durability
3. **Rate Limiting**: Configure per-agent rate limits
4. **Monitoring**: Enable OpenTelemetry

## Example Use Cases

### 1. Code Review System

```
User → AutoGen(code review) → Human approval → Execute changes
```

### 2. Customer Support

```
User → AutoGen(triage) → AutoGen(respond) → Human review (optional) → Send response
```

### 3. Research Assistant

```
User → AutoGen(search) → AutoGen(summarize) → Human approval → Generate report
```

## See Also

- [AutoGen Documentation](https://microsoft.github.io/autogen/)
- [LangGraph Adapter](../langgraph_agent/)
- [Aetheris Getting Started](../../docs/getting-started-agents.md)
- [Runtime Guarantees](../../docs/runtime-guarantees.md)
