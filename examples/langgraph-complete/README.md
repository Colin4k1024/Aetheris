# LangGraph Complete Example

This example demonstrates a complete integration of LangGraph with Aetheris for building a production-ready research agent.

## Overview

The example implements a **Research Agent** that:

- Uses LangGraph for reasoning and workflow orchestration
- Leverages Aetheris for durability, replay, and audit
- Supports human-in-the-loop approval workflows
- Provides checkpoint-based crash recovery

## Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                        Aetheris Runtime                         │
│  ┌─────────────────────────────────────────────────────────┐   │
│  │  LangGraph Adapter                                      │   │
│  │  ┌─────────────────────────────────────────────────┐   │   │
│  │  │  ResearchGraphClient (LangGraph)               │   │   │
│  │  │  - Search → Analyze → Generate Answer          │   │   │
│  │  └─────────────────────────────────────────────────┘   │   │
│  └─────────────────────────────────────────────────────────┘   │
│                                                                  │
│  ┌─────────────────────────────────────────────────────────┐   │
│  │  Aetheris Guarantees                                   │   │
│  │  - Event sourcing (JobStore)                          │   │
│  │  - Idempotency (Tool Ledger)                         │   │
│  │  - Crash recovery (Checkpoint Store)                  │   │
│  │  - Human approval (Signal/Wait)                      │   │
│  └─────────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────────┘
```

## Quick Start

### Prerequisites

- Go 1.25+
- Aetheris API running (see [docs](https://github.com/Colin4k1024/Aetheris))

### Run the Demo

```bash
cd examples/langgraph-complete
go run main.go
```

### Expected Output

```
=== LangGraph + Aetheris Research Agent Demo ===

=== TaskGraph Definition ===
{
  "nodes": [...],
  "edges": [...]
}

=== Running Demo ===

--- Demo 1: Simple Research Query ---
Query: What is machine learning?
Status: completed
Answer: Based on the research...

--- Demo 2: Query Requiring Approval ---
Query: I want to purchase an expensive AI model
Status: waiting_approval
Pending Answer: Based on the research...
Action: Waiting for human approval via Aetheris signal API
```

## Integration Steps

### Step 1: Implement LangGraph Client

```go
type ResearchGraphClient struct {
    APIEndpoint string
    APIKey      string
}

func (c *ResearchGraphClient) Invoke(ctx context.Context, input map[string]any) (map[string]any, error) {
    // Your LangGraph logic here
    return map[string]any{"status": "completed"}, nil
}
```

### Step 2: Create Aetheris Adapter

```go
type LangGraphAdapter struct {
    Client      agentexec.LangGraphClient
    EffectStore agentexec.EffectStore
}
```

### Step 3: Define TaskGraph

```go
taskGraph := &planner.TaskGraph{
    Nodes: []planner.TaskNode{
        {ID: "research", Type: planner.NodeLangGraph},
        {ID: "approval", Type: planner.NodeWait},
    },
    Edges: []planner.TaskEdge{
        {From: "research", To: "approval"},
    },
}
```

### Step 4: Submit to Aetheris

```bash
# Create agent
curl -X POST http://localhost:8080/api/agents \
  -H "Content-Type: application/json" \
  -d '{
    "name": "research-agent",
    "workflow": <taskgraph_json>
  }'

# Submit job
curl -X POST http://localhost:8080/api/agents/{agent_id}/message \
  -d '{"message": "Research machine learning"}'
```

## API Usage

### Create Agent

```bash
curl -X POST http://localhost:8080/api/agents \
  -H "Content-Type: application/json" \
  -d '{
    "name": "langgraph-researcher",
    "description": "Research agent with LangGraph"
  }'
```

### Submit Research Request

```bash
curl -X POST http://localhost:8080/api/agents/{agent_id}/message \
  -d '{"message": "Research the latest developments in quantum computing"}'
```

### Check Job Status

```bash
curl http://localhost:8080/api/jobs/{job_id}
```

### Approve (if required)

```bash
curl -X POST http://localhost:8080/api/jobs/{job_id}/signal \
  -H "Content-Type: application/json" \
  -d '{
    "correlation_key": "research-approval",
    "signal": "approved"
  }'
```

### View Events

```bash
curl http://localhost:8080/api/jobs/{job_id}/events
```

### Replay for Audit

```bash
curl http://localhost:8080/api/jobs/{job_id}/replay
```

## Key Features

### 1. Event Sourcing

Every action is recorded in the JobStore:

```bash
# View event stream
curl http://localhost:8080/api/jobs/{job_id}/events | jq
```

### 2. Crash Recovery

If the worker crashes, Aetheris automatically recovers:

```bash
# Job resumes from last checkpoint
curl http://localhost:8080/api/jobs/{job_id}
# Status: "running" (recovered)
```

### 3. Human-in-the-Loop

```go
// In your LangGraph client, return wait error for approval
return nil, &LangGraphError{
    Code:           "wait",
    CorrelationKey: "research-approval",
    Message:       "Requires human approval",
}
```

Then approve via API:

```bash
curl -X POST http://localhost:8080/api/jobs/{job_id}/signal \
  -d '{"correlation_key": "research-approval", "signal": "approved"}'
```

### 4. Idempotency

The Tool Ledger ensures side effects are executed at most once:

- First execution: Calls LangGraph → stores result
- Replay: Restores from Ledger → skips LangGraph call

## Production Considerations

### 1. Use PostgreSQL for Durability

```go
effectStore := agentexec.NewEffectStorePg(db)
```

### 2. Configure Rate Limiting

```yaml
rate_limits:
  tools:
    langgraph_api:
      qps: 10
      max_concurrent: 5
```

### 3. Enable Monitoring

```yaml
opentelemetry:
  endpoint: "localhost:4318"
  service_name: "aetheris-langgraph"
```

## See Also

- [LangGraph Adapter Doc](../../docs/adapters/langgraph.md)
- [Aetheris Getting Started](../../docs/getting-started-agents.md)
- [Runtime Guarantees](../../docs/runtime-guarantees.md)
- [API Reference](../../docs/api.md)
