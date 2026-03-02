# AgentScope Adapter Example

This example demonstrates how to integrate AgentScope multi-agent system with Aetheris for production-ready agent execution.

## Overview

AgentScope is a multi-agent platform designed for building distributed agent applications. This example shows how to:
- Connect AgentScope agents to Aetheris runtime
- Leverage Aetheris for durability and audit
- Add human-in-the-loop workflows
- Achieve crash recovery

## Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                        Aetheris Runtime                         │
│  ┌─────────────────────────────────────────────────────────┐   │
│  │  AgentScope Adapter                                       │   │
│  │  ┌─────────────────────────────────────────────────┐   │   │
│  │  │  AgentScope Multi-Agent System                    │   │   │
│  │  │  - Researcher Agent                             │   │   │
│  │  │  - Analyst Agent                                │   │   │
│  │  │  - Summarizer Agent                            │   │   │
│  │  └─────────────────────────────────────────────────┘   │   │
│  └─────────────────────────────────────────────────────────┘   │
│                                                                  │
│  ┌─────────────────────────────────────────────────────────┐   │
│  │  Aetheris Guarantees                                   │   │
│  │  - Event sourcing, Idempotency, Recovery               │   │
│  └─────────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────────┘
```

## Quick Start

### Run the Demo

```bash
cd examples/agentscope_agent
go run main.go
```

### Expected Output

```
=== AgentScope + Aetheris Multi-Agent Demo ===

=== AgentScope TaskGraph ===
{
  "nodes": [...],
  "edges": [...]
}

=== Running Demo ===

--- Demo 1: Simple Research Task ---
Query: Analyze the impact of AI on software development
Status: completed
Summary: Multi-Agent Analysis Complete:
[researcher]: ...
[analyst]: ...
[summarizer]: ...
```

## Integration Steps

### Step 1: Implement AgentScope Client

```go
type AgentScopeClient struct {
    Endpoint string
    APIKey   string
}

func (c *AgentScopeClient) Invoke(ctx context.Context, input map[string]any) (map[string]any, error) {
    // Your AgentScope logic here
    return map[string]any{"status": "completed"}, nil
}

func (c *AgentScopeClient) Stream(ctx context.Context, input map[string]any, onChunk func(chunk map[string]any) error) error {
    return nil
}

func (c *AgentScopeClient) State(ctx context.Context, sessionID string) (map[string]any, error) {
    return map[string]any{"session_id": sessionID}, nil
}
```

### Step 2: Define TaskGraph

```go
taskGraph := &planner.TaskGraph{
    Nodes: []planner.TaskNode{
        {ID: "multi_agent", Type: planner.NodeLangGraph},  // Reuse or create NodeAgentScope
        {ID: "approval", Type: planner.NodeWait},
    },
    Edges: []planner.TaskEdge{
        {From: "multi_agent", To: "approval"},
    },
}
```

### Step 3: Submit to Aetheris

```bash
# Create agent
curl -X POST http://localhost:8080/api/agents \
  -H "Content-Type: application/json" \
  -d '{
    "name": "agentscope-team",
    "workflow": <taskgraph_json>
  }'

# Submit task
curl -X POST http://localhost:8080/api/agents/{agent_id}/message \
  -d '{"message": "Research and analyze AI trends"}'
```

## API Usage

### Create Agent

```bash
curl -X POST http://localhost:8080/api/agents \
  -H "Content-Type: application/json" \
  -d '{
    "name": "agentscope-multi-agent",
    "description": "Multi-agent research team"
  }'
```

### Submit Task

```bash
curl -X POST http://localhost:8080/api/agents/{agent_id}/message \
  -d '{"message": "Analyze market trends for AI products"}'
```

### Check Status

```bash
curl http://localhost:8080/api/jobs/{job_id}
```

### View Events

```bash
curl http://localhost:8080/api/jobs/{job_id}/events
```

## Key Features

### 1. Multi-Agent Coordination

AgentScope excels at coordinating multiple specialized agents:

```go
Agents: []AgentConfig{
    {Name: "researcher", Role: "Research", Tools: []string{"search", "scrape"}},
    {Name: "analyst", Role: "Analysis", Tools: []string{"analyze", "compare"}},
    {Name: "summarizer", Role: "Summary", Tools: []string{"summarize"}},
}
```

### 2. Event Sourcing

```bash
curl http://localhost:8080/api/jobs/{job_id}/events
```

### 3. Crash Recovery

Aetheris automatically recovers from failures:

```bash
# Job resumes from last checkpoint
curl http://localhost:8080/api/jobs/{job_id}
```

### 4. Human Approval

```go
// Return wait error for approval
return map[string]any{
    "status": "waiting_approval",
    "requires_approval": true,
}, nil

// Approve via API
curl -X POST http://localhost:8080/api/jobs/{job_id}/signal \
  -d '{"correlation_key": "agentscope-review", "signal": "approved"}'
```

## Production Considerations

### 1. Use PostgreSQL Effect Store

```go
effectStore := agentexec.NewEffectStorePg(db)
```

### 2. Configure Rate Limiting

```yaml
rate_limits:
  tools:
    agentscope_api:
      qps: 20
      max_concurrent: 10
```

### 3. Enable Monitoring

```yaml
opentelemetry:
  endpoint: "localhost:4318"
  service_name: "aetheris-agentscope"
```

## Error Mapping

| AgentScope Error | Aetheris Result |
|-----------------|-----------------|
| `wait` | Signal wait for approval |
| `retryable` | Retryable failure |
| `permanent` | Permanent failure |

## Use Cases

- **Research Teams**: Multiple agents researching different aspects
- **Analysis Pipelines**: Sequential/parallel data analysis
- **Customer Service**: Agent teams handling different query types
- **Content Generation**: Collaborative writing/editing agents

## See Also

- [AgentScope GitHub](https://github.com/agentscope-ai/agentscope)
- [LangGraph Adapter](../langgraph_agent/)
- [AutoGen Adapter](../autogen_agent/)
- [CrewAI Adapter](../crewai_agent/)
- [Aetheris Getting Started](../../docs/getting-started-agents.md)
