# CrewAI Adapter Example

This example demonstrates how to run CrewAI crews on Aetheris runtime.

## Overview

CrewAI is a framework for building AI agent crews. This adapter allows you to:
- Run CrewAI crews inside Aetheris
- Keep Aetheris guarantees for durability, replay, and audit
- Add hierarchical approval workflows

## Architecture

```
┌─────────────────────────────────────────────────────────┐
│                     Aetheris                            │
│  ┌─────────────────────────────────────────────────┐   │
│  │  CrewAI Adapter                                 │   │
│  │  ┌───────────────────────────────────────────┐   │   │
│  │  │  CrewAI Crew                              │   │   │
│  │  │  - Agents (Researcher, Writer, etc.)     │   │   │
│  │  │  - Tasks (sequential/hierarchical)      │   │   │
│  │  │  - Process (sequential/hierarchical)    │   │   │
│  │  └───────────────────────────────────────────┘   │   │
│  └─────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────┘
```

## Quick Start

### 1. Implement CrewAIClient

```go
type CrewAIClient struct {
    Endpoint string
    APIKey   string
}

func (c *CrewAIClient) Invoke(ctx context.Context, input map[string]any) (map[string]any, error) {
    // Call your CrewAI service
    return map[string]any{
        "status":       "completed",
        "final_output": "Processed by CrewAI",
    }, nil
}

func (c *CrewAIClient) Stream(ctx context.Context, input map[string]any, onChunk func(chunk map[string]any) error) error {
    return nil
}
```

### 2. Configure Crew

```go
config := CrewConfig{
    Agents: []AgentConfig{
        {Role: "Researcher", Goal: "Research the topic", Tools: []string{"search"}},
        {Role: "Writer", Goal: "Write a report", Tools: []string{"write"}},
    },
    Tasks: []TaskConfig{
        {Description: "Research task", Agent: "Researcher"},
        {Description: "Write task", Agent: "Writer"},
    },
    Process: ProcessSequential,
}
```

### 3. Convert to TaskGraph

```go
// Sequential mode
taskGraph := &planner.TaskGraph{
    Nodes: []planner.TaskNode{
        {ID: "task_0", Type: planner.NodeTool, Config: map[string]any{"agent": "Researcher"}},
        {ID: "task_1", Type: planner.NodeTool, Config: map[string]any{"agent": "Writer"}},
    },
    Edges: []planner.TaskEdge{
        {From: "task_0", To: "task_1"},
    },
}

// Hierarchical mode (with manager approval)
taskGraph := &planner.TaskGraph{
    Nodes: []planner.TaskNode{
        {ID: "task_0", Type: planner.NodeTool},
        {ID: "task_1", Type: planner.NodeTool},
        {ID: "approval", Type: planner.NodeWait, Config: map[string]any{
            "wait_kind":       "signal",
            "correlation_key": "manager-approval",
        }},
    },
    Edges: []planner.TaskEdge{
        {From: "task_0", To: "task_1"},
        {From: "task_1", To: "approval"},
    },
}
```

## Key Features

### Sequential Process

Agents complete tasks one by one:

```
Researcher → Writer → Output
```

### Hierarchical Process

Manager agent oversees the crew:

```
Researcher ──┐
Writer ──────┼──→ Manager (approval) → Output
Analyzer ────┘
```

### Human-in-the-Loop

```go
// 需要审批时返回错误
return nil, &CrewAIError{
    Code:          CrewAIErrorAwaitApproval,
    Message:       "需要项目经理审批",
    ApprovalLevel: "manager",
}

// 恢复执行
// curl -X POST http://localhost:8080/api/jobs/{job_id}/signal \
//   -d '{"correlation_key": "manager-approval", "input": "approved"}'
```

### Error Mapping

| CrewAI Error | Aetheris Result |
|--------------|-----------------|
| `await_approval` | Signal wait for approval |
| `retryable` | Retryable failure |
| `permanent` | Permanent failure |

## Migration Patterns

### Pattern 1: Black-box Crew

Run entire crew as one node:

```go
{ID: "crew", Type: planner.NodeLangGraph}  // 复用
```

### Pattern 2: Task-by-Task

Map each CrewAI task to Aetheris step:

```go
Nodes: []planner.TaskNode{
    {ID: "research", Type: planner.NodeTool, Config: map[string]any{"agent": "Researcher"}},
    {ID: "analyze", Type: planner.NodeTool, Config: map[string]any{"agent": "Analyzer"}},
    {ID: "write", Type: planner.NodeTool, Config: map[string]any{"agent": "Writer"}},
}
```

### Pattern 3: Hybrid

```go
Nodes: []planner.TaskNode{
    {ID: "quick_tasks", Type: planner.NodeLangGraph},  // 简单任务合并
    {ID: "complex_analysis", Type: planner.NodeTool},  // 复杂任务单独
}
```

## Example Use Cases

### 1. Market Research

```
Researcher (收集数据) → Analyzer (分析趋势) → Writer (生成报告) → Manager (审批) → 发布
```

### 2. Code Development

```
Architect (设计) → Coder (实现) → Reviewer (审查) → Manager (审批) → 合并
```

### 3. Content Creation

```
Planner (策划) → Writer (撰写) → Editor (编辑) → Publisher (发布)
```

## Production Considerations

1. **CrewAI Service**: Deploy as separate microservice
2. **Effect Store**: Use PostgreSQL for durability
3. **Rate Limiting**: Configure per-agent limits
4. **Monitoring**: Enable OpenTelemetry tracing

## See Also

- [CrewAI Documentation](https://docs.crewai.com/)
- [LangGraph Adapter](../langgraph_agent/)
- [AutoGen Adapter](../autogen_agent/)
- [Aetheris Getting Started](../../docs/getting-started-agents.md)
