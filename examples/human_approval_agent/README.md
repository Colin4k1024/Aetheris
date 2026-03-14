# Human-in-the-Loop Approval Example

This example demonstrates how to build human-in-the-loop workflows with Aetheris.

## Prerequisites

- Go 1.25.7+
- **Cloud LLM**: Set `DASHSCOPE_API_KEY` (Qwen) or `OPENAI_API_KEY` (OpenAI)

## Overview

Aetheris supports pausing agent execution and waiting for human input before continuing. This is essential for:
- Approval workflows (refunds, payments, contracts)
- Quality gates (document review, code review)
- Exception handling (fallback to human decision)

## Architecture

```
┌─────────────┐    ┌──────────────┐    ┌─────────────┐
│   Analyze  │───▶│  Wait/Human  │───▶│   Execute   │
│   Request  │    │   Approval   │    │   Action    │
└─────────────┘    └──────────────┘    └─────────────┘
                       ↑                      │
                       └──────────────────────┘
                         signal / resume
```

## Quick Start

### Create Agent with Approval Workflow

```bash
# Create agent with human approval workflow
curl -X POST http://localhost:8080/api/agents \
  -H "Content-Type: application/json" \
  -d '{
    "name": "refund-approval",
    "workflow": {
      "nodes": [
        {"id": "analyze", "type": "llm"},
        {"id": "wait_approval", "type": "wait", "config": {
          "wait_kind": "signal",
          "correlation_key": "refund-approval"
        }},
        {"id": "execute", "type": "tool", "config": {
          "tool_name": "process_refund"
        }}
      ],
      "edges": [
        {"from": "analyze", "to": "wait_approval"},
        {"from": "wait_approval", "to": "execute"}
      ]
    }
  }'
```

### Submit Request (Waits at Approval)

```bash
# Submit refund request - job will pause at approval node
curl -X POST http://localhost:8080/api/agents/{agent_id}/message \
  -d '{"message": "客户申请退款 $99.99"}'
```

### Check Job Status

```bash
# Job will be in "waiting" status
curl http://localhost:8080/api/jobs/{job_id}
# Response: {"status": "waiting", "waiting_for": "refund-approval"}
```

### Human Approval

```bash
# Approve
curl -X POST http://localhost:8080/api/jobs/{job_id}/signal \
  -H "Content-Type: application/json" \
  -d '{
    "correlation_key": "refund-approval",
    "signal": "approved",
    "comment": "已核实，同意退款"
  }'

# Or Reject
curl -X POST http://localhost:8080/api/jobs/{job_id}/signal \
  -H "Content-Type: application/json" \
  -d '{
    "correlation_key": "refund-approval",
    "signal": "rejected",
    "comment": "退款原因不充分"
  }'
```

## Workflow Patterns

### Pattern 1: Simple Approval

```
LLM Analysis → Wait Approval → Execute
```

### Pattern 2: Multi-Level Approval

```
Validate → Finance Approval → CEO Approval (> $10k) → Execute
```

### Pattern 3: Conditional Approval

```
LLM Analysis →
  If amount < $1000: Auto Approve → Execute
  If amount >= $1000: Manager Approval → Execute
```

### Pattern 4: Review Cycle

```
Generate Draft → Legal Review → (revisions) → Manager Review → Publish
```

## Wait Types

### Signal Wait

```go
{
    "type": "wait",
    "config": {
        "wait_kind": "signal",
        "correlation_key": "approval-123"
    }
}
```

Resume via `POST /api/jobs/{id}/signal`

### Message Wait (Queue)

```go
{
    "type": "wait",
    "config": {
        "wait_kind": "message",
        "correlation_key": "queue-123"
    }
}
```

Resume by publishing to message queue

### Timeout

```go
{
    "type": "wait",
    "config": {
        "wait_kind": "signal",
        "correlation_key": "approval-123",
        "timeout": "24h"
    }
}
```

Job auto-fails after timeout if not signaled

## Use Cases

### 1. Financial Approvals

- Refund requests
- Invoice approval
- Purchase orders
- Budget requests

### 2. Content Moderation

- User-generated content review
- Document approval
- Marketing materials

### 3. Exception Handling

- Failed transactions
- Anomaly detection
- Manual intervention

### 4. Compliance

- Regulatory approvals
- Audit sign-offs
- Legal review

## Best Practices

1. **Clear correlation keys**: Use descriptive IDs like `refund-{order_id}`
2. **Timeout设置**: Set reasonable timeouts to avoid stuck jobs
3. **Rich context**: Include all relevant info in signal payload
4. **Audit trail**: All approvals are recorded in event stream

## Monitoring

```bash
# List waiting jobs
curl "http://localhost:8080/api/jobs?status=waiting"

# Get job events
curl http://localhost:8080/api/jobs/{job_id}/events

# Replay for audit
curl http://localhost:8080/api/jobs/{job_id}/replay
```

## See Also

- [Getting Started with Agents](../../docs/getting-started-agents.md)
- [Runtime Guarantees](../../docs/runtime-guarantees.md)
- [API Reference](../../docs/api.md)
