# LogiShip - Supply Chain Optimization

**Industry:** Logistics
**Use Case:** Autonomous inventory replenishment across 50 warehouses
**Results:** 25% reduction in stockouts, $2M annual savings

---

## Company Overview

LogiShip is a logistics company managing inventory across 50 warehouses in 5 regions (US, EU, APAC). Their AI agent automates inventory replenishment decisions based on demand forecasting.

---

## Challenge

| Challenge | Impact |
|-----------|--------|
| Long-running optimization (hours) | Worker crashes lost all progress |
| Multiple workers = race conditions | Same warehouse processed twice |
| Regional data residency | EU data must stay in EU |
| High volume (50K jobs/day) | Need rate limiting per region |

> "We tried Kubernetes Jobs but couldn't handle the complexity of agent state. Workers would fight over jobs, or crash and lose everything." — VP Engineering, LogiShip

---

## Solution

### 1. Regional Scheduling (v2.2.0)

```yaml
# Worker in US-East
configs/worker.yaml
scheduler:
  region: "us-east-1"
  allowed_regions: ["us-east-1", "us-east-2"]

# Worker in EU-West
scheduler:
  region: "eu-west-1"
  allowed_regions: ["eu-west-1", "eu-central-1"]
```

Jobs automatically route to workers in the same region.

### 2. Lease Fencing

```go
// Scheduler prevents duplicate execution
lease, err := scheduler.AcquireLease(ctx, jobID, workerID, 5*time.Minute)
if err == ErrLeaseHeldByOther {
    // Another worker has it, skip
    return
}
```

Each job has a 5-minute lease. Only the lease holder can execute.

### 3. SLA Quotas

```bash
# Per-region rate limiting
curl -X POST http://localhost:8080/api/sla/quotas \
  -d '{
    "agent_id": "inventory-agent",
    "region": "eu-west-1",
    "max_rpm": 1000,
    "max_daily": 50000
  }'
```

---

## Architecture

```
┌─────────────────────────────────────────────────────────┐
│                   Regional Workers                       │
├─────────────┬─────────────┬─────────────┬───────────────┤
│  us-east-1  │  us-west-2  │  eu-west-1  │   apac-east   │
│  (12 jobs/s)│  (10 jobs/s)│  (8 jobs/s)│  (15 jobs/s) │
└─────────────┴─────────────┴─────────────┴───────────────┘
         │              │             │              │
         └──────────────┴─────────────┴──────────────┘
                           │
                    PostgreSQL
                    (Primary + Replica per region)
```

---

## Results

| Metric | Before | After |
|--------|--------|-------|
| Stockout rate | 8% | **6%** |
| Annual savings | — | **$2M** |
| Job completion | 97% | **99.99%** |
| Processing cost/job | $0.12 | **$0.03** |

> "Aetheris reduced our infrastructure costs by 75% while improving reliability. The regional scheduling alone saved us $500K in data transfer costs." — CTO, LogiShip

---

## Key Aetheris Features Used

| Feature | Benefit |
|---------|---------|
| **Lease Fencing** | Prevents duplicate processing |
| **Region Scheduling** | Data residency compliance |
| **SLA Quotas** | Per-region rate limiting |
| **Durable Execution** | Long-running jobs survive crashes |
| **Multi-Worker** | Horizontal scaling |

---

## Learn More

- [Regional Scheduling](../guides/deployment.md)
- [SLA Quota Manager](../guides/m2-rbac-guide.md)
- [Scheduler Design](../design/scheduler-correctness.md)
