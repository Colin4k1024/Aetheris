# Multi-Agent Collaboration Example

This example demonstrates how to build multi-agent workflows with Aetheris.

## Prerequisites

- Go 1.25.7+
- **Cloud LLM**: Set `DASHSCOPE_API_KEY` (Qwen) or `OPENAI_API_KEY` (OpenAI)

## Overview

Multi-agent systems enable complex workflows where different agents specialize in specific tasks:
- **Parallel execution** for speed
- **Sequential processing** for dependencies
- **Conditional routing** based on context

## Architecture Patterns

### Sequential Agents

```
Agent A → Agent B → Agent C
```

Each agent completes its task before the next starts.

### Parallel Agents

```
   ┌→ Agent A →┐
→ Root ──→ Agent B → Aggregate → Next
   └→ Agent C →┘
```

Multiple agents run simultaneously, results aggregated.

### Hierarchical Agents

```
Manager Agent
  ├──→ Agent A
  ├──→ Agent B
  └──→ Agent C
```

Manager coordinates sub-agents.

## Examples

### 1. Research & Report

```
Researcher → Analyzer → Writer → Editor → Publisher
```

Each agent adds value sequentially.

### 2. Parallel Data Collection

```
  Web Search ─┐
  DB Query   ─┼─→ Aggregate → Analyze
  Doc Search ─┘
```

Collect from multiple sources simultaneously.

### 3. Customer Support Triage

```
Triage → Technical / Billing / Escalate → Respond
```

Route to appropriate specialist.

### 4. Code Review

```
Security Scan ─┐
Style Check   ─┼─→ Aggregate → Human Review
Coverage      ─┘
```

Parallel checks with human oversight.

## Quick Start

### Create Multi-Agent Workflow

```bash
curl -X POST http://localhost:8080/api/agents \
  -H "Content-Type: application/json" \
  -d '{
    "name": "research-team",
    "workflow": {
      "nodes": [
        {"id": "search", "type": "tool", "config": {"tool_name": "web_search"}},
        {"id": "analyze", "type": "llm", "config": {"prompt": "分析搜索结果"}},
        {"id": "write", "type": "llm", "config": {"prompt": "撰写报告"}},
        {"id": "review", "type": "wait", "config": {
          "wait_kind": "signal",
          "correlation_key": "review"
        }}
      ],
      "edges": [
        {"from": "search", "to": "analyze"},
        {"from": "analyze", "to": "write"},
        {"from": "write", "to": "review"}
      ]
    }
  }'
```

### Execute Workflow

```bash
curl -X POST http://localhost:8080/api/agents/{agent_id}/message \
  -d '{"message": "研究 AI Agent 的最新发展"}'
```

### Monitor Progress

```bash
# Check job status
curl http://localhost:8080/api/jobs/{job_id}

# View events
curl http://localhost:8080/api/jobs/{job_id}/events

# View trace
curl http://localhost:8080/api/jobs/{job_id}/trace
```

## Advanced Patterns

### Conditional Routing

Use LLM to decide next step:

```json
{
  "nodes": [
    {"id": "triage", "type": "llm"},
    {"id": "route_a", "type": "tool"},
    {"id": "route_b", "type": "tool"}
  ],
  "edges": [
    {"from": "triage", "to": "route_a", "condition": "{{triage.action}} == 'A'"},
    {"from": "triage", "to": "route_b", "condition": "{{triage.action}} == 'B'"}
  ]
}
```

### Parallel with Barrier

All parallel tasks must complete before proceeding:

```json
{
  "edges": [
    {"from": "task_a", "to": "aggregate"},
    {"from": "task_b", "to": "aggregate"},
    {"from": "task_c", "to": "aggregate"}
  ]
}
```

### Loop Pattern

For iterative refinement:

```
Generate → Review → (needs revision?) → Generate
           ↓ yes
         Finish
```

## Best Practices

1. **Clear agent roles**: Each agent should have a specific purpose
2. **Data passing**: Use `{{node_id.output}}` to pass data between agents
3. **Error handling**: Add retry policies for unreliable agents
4. **Human checkpoints**: Add wait nodes for critical decisions

## Use Cases

- **Research automation**: Gather, analyze, summarize
- **Content generation**: Draft, review, edit, publish
- **Customer service**: Triage, route, resolve
- **Code review**: Scan, check, aggregate, approve
- **Data processing**: Extract, transform, validate, store

## See Also

- [Human Approval Example](../human_approval_agent/)
- [Getting Started](../../docs/guides/getting-started-agents.md)
