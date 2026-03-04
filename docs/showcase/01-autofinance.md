# AutoFinance - AI Financial Advisor

**Industry:** Fintech
**Use Case:** Autonomous investment portfolio rebalancing agent
**Results:** 99.9% execution reliability, 60% reduction in manual oversight

---

## Company Overview

AutoFinance is a fintech startup providing AI-powered wealth management services to retail investors. Their core product is an autonomous agent that monitors market conditions and executes portfolio rebalancing decisions.

---

## Challenge

AutoFinance needed an AI agent to handle portfolio rebalancing decisions that must execute reliably even during market volatility. Their previous solution (built with LangGraph) experienced:

| Problem | Impact |
|---------|--------|
| Lost execution state | Jobs failed mid-execution during worker restarts |
| Duplicate trades | Network timeouts caused retries, leading to duplicate executions |
| No audit trail | Could not meet regulatory compliance requirements |

> "We lost $50K in a single incident due to a duplicate trade. We needed a solution that guarantees execution exactly once." — CTO, AutoFinance

---

## Solution

AutoFinance rebuilt their agent on Aetheris, leveraging:

### 1. Durable Execution

```yaml
jobstore:
  type: postgres
  dsn: "postgres://user:pass@db:5432/aetheris"
```

Job state persists to PostgreSQL. Any process restart preserves execution context.

### 2. At-Most-Once via Effects Ledger

```go
// Tool execution with idempotency
toolCallID, err := ledger.Commit(ctx, "execute_trade", map[string]interface{}{
    "symbol":   "AAPL",
    "quantity": 100,
    "action":   "BUY",
})
```

- Unique `tool_call_id` per trade
- Commit before execution, rollback on failure
- Deduplication via stored trade IDs

### 3. Evidence Chain for Compliance

```bash
# Export evidence package for audit
curl -X POST http://localhost:8080/api/jobs/job-xxx/export \
  -o evidence-2026-01-15.zip

# Verify evidence integrity
aetheris verify evidence-2026-01-15.zip
```

---

## Architecture

```
┌─────────────────────────────────────────────────────────┐
│                    Aetheris Runtime                      │
├─────────────────────────────────────────────────────────┤
│  API Server (Hertz)                                     │
│  ├── Job Management (Create/Query/Stop)                 │
│  └── Evidence Export                                    │
├─────────────────────────────────────────────────────────┤
│  Worker Pool (3 nodes)                                  │
│  ├── Scheduler (Lease Fencing)                          │
│  ├── Planner (TaskGraph Generation)                     │
│  └── Runner (Execution + Effects Ledger)                │
├─────────────────────────────────────────────────────────┤
│  PostgreSQL                                              │
│  ├── Job Events (Event Sourcing)                        │
│  ├── Checkpoints (State Recovery)                       │
│  └── Effects Ledger (At-Most-Once)                      │
└─────────────────────────────────────────────────────────┘
```

---

## Results

| Metric | Before | After |
|--------|--------|-------|
| Execution reliability | 85% | **99.9%** |
| Duplicate trades (6 mo) | 12 | **0** |
| Manual oversight hours/week | 40 | **16** |
| Audit compliance | Partial | **100%** |

> "Aetheris gave us the reliability and auditability we needed to operate in production. It's Temporal for Agents, exactly what we needed." — Lead Engineer, AutoFinance

---

## Future Plans

- Multi-region deployment for global customers
- Real-time risk monitoring with human-in-the-loop
- Integration with Bloomberg Terminal

---

## Learn More

- [Runtime Guarantees](../guides/runtime-guarantees.md)
- [Evidence Package Export](../guides/m3-evidence-graph-guide.md)
- [At-Most-Once Execution](https://docs.aetheris.ai/concepts/execution-guarantees)
