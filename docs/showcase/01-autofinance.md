# Example: Financial Trading Agent — Autonomous Rebalancing

> **⚠️ This is a technical example/demonstration, not a real company case study.**
>
> For real-world case studies, see [Discussions › Show & Tell](https://github.com/Colin4k1024/Aetheris/discussions/category/show-and-tell).

**Type:** Example / Technical Demo
**Industry:** Fintech
**Use Case:** Autonomous investment portfolio rebalancing agent
**Aetheris Features:** Durable Execution, At-Most-Once, Evidence Chain

---

## Problem Statement

A portfolio rebalancing agent must:

- Execute reliably even during worker restarts (market volatility = high restart risk)
- **Never** execute duplicate trades (network timeouts cause retries → duplicate executions)
- Provide complete audit trail for regulatory compliance

```
❌ Worker crash → lost execution state → restart from scratch
❌ Retry on timeout → same order sent twice → $50K duplicate trade
❌ No audit trail → regulatory violation
```

---

## Solution

### 1. Durable Execution with PostgreSQL

Job state persists to PostgreSQL. Any process restart preserves execution context.

```yaml
# configs/api.yaml
jobstore:
  type: postgres
  dsn: "postgres://user:password@db:5432/aetheris"
```

```go
// Job automatically checkpoints after each step
job, err := runtime.SubmitJob(ctx, &Job{
    Name:   "rebalance-portfolio",
    Agent:  portfolioAgent,
    Input:  map[string]int{"AAPL": 100, "GOOGL": 50, "MSFT": 75},
})
// Worker crash? Job resumes from last checkpoint — no restart from scratch.
```

### 2. At-Most-Once via Effects Ledger

```go
// Tool execution with idempotency key
toolCallID, err := ledger.Commit(ctx, "execute_trade", map[string]any{
    "symbol":    "AAPL",
    "quantity":  100,
    "action":    "BUY",
    "idempotency_key": "order-AAPL-2026-01-15-001",
})
if err == ErrAlreadyCommitted {
    log.Printf("Trade already executed, skipping")
    return nil
}
```

- Commit **before** execution, rollback on failure
- Deduplication via stored idempotency keys
- **0 duplicate trades** even after worker crashes

### 3. Evidence Chain for Compliance

```bash
# Export audit package
curl -X POST http://localhost:8080/api/jobs/{job_id}/export \
  -o evidence-2026-01-15.zip

# Evidence package contains:
# - All events with timestamps
# - State snapshots at each checkpoint
# - LLM prompts and responses
# - Tool calls with effects
# - Cryptographic integrity hash
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
│  Worker Pool (N nodes)                                  │
│  ├── Scheduler (Lease Fencing)                          │
│  ├── Planner (TaskGraph Generation)                      │
│  └── Runner (Execution + Effects Ledger)                │
├─────────────────────────────────────────────────────────┤
│  PostgreSQL                                             │
│  ├── Job Events (Event Sourcing)                        │
│  ├── Checkpoints (State Recovery)                      │
│  └── Effects Ledger (At-Most-Once)                     │
└─────────────────────────────────────────────────────────┘
```

---

## Key Aetheris Features Used

| Feature | Why It Matters for Trading |
|---------|---------------------------|
| **At-Most-Once** | Never duplicate a trade — critical for financial compliance |
| **Durable Execution** | Survive worker crashes mid-session without losing progress |
| **Evidence Chain** | Regulatory audit trail with cryptographic integrity |
| **Lease Fencing** | Multiple workers can't process the same job |

---

## Run This Example

```bash
# Start PostgreSQL
docker run -d --name aetheris-pg -p 5432:5432 \
  -e POSTGRES_USER=aetheris -e POSTGRES_PASSWORD=secret \
  -e POSTGRES_DB=aetheris postgres:15-alpine

# Apply schema
psql "postgres://aetheris:secret@localhost:5432/aetheris?sslmode=disable" \
  -f internal/runtime/jobstore/schema.sql

# Run the example
cd examples/financial-trading-agent
go run ./...

# Observe: kill the worker mid-execution, restart it — job resumes
```

---

## Learn More

- [Runtime Guarantees](../guides/runtime-guarantees.md)
- [Effects Ledger (At-Most-Once)](../docs/concepts/event-sourcing.md)
- [Evidence Package Export](../guides/m3-evidence-graph-guide.md)
- [Human-in-the-Loop](../guides/getting-started-agents.md)
