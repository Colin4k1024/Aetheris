# Example: Multi-Region Inventory Optimization — Horizontal Scaling

> **⚠️ This is a technical example/demonstration, not a real company case study.**
>
> For real-world case studies, see [Discussions › Show & Tell](https://github.com/Colin4k1024/Aetheris/discussions/category/show-and-tell).

**Type:** Example / Technical Demo
**Industry:** Logistics
**Use Case:** Autonomous inventory replenishment across 50 warehouses
**Aetheris Features:** Regional Scheduling, Lease Fencing, SLA Quotas, Durable Execution, Multi-Worker

---

## Problem Statement

A large-scale inventory optimization system must:

- Handle long-running optimization tasks (hours) that survive worker crashes
- Prevent multiple workers from processing the same warehouse (race conditions)
- Route jobs to workers in the same region (data residency)
- Handle high volume (50K+ jobs/day) with per-region rate limiting

```
❌ Long-running job (2 hours) → worker crashes → all progress lost
❌ Two workers pick up same warehouse → duplicate processing
❌ EU warehouse job routed to US worker → GDPR violation
❌ Burst of 10K jobs → overwhelm EU region capacity
```

---

## Solution

### 1. Regional Scheduling

```yaml
# Worker in US-East
# configs/worker.yaml
scheduler:
  region: "us-east-1"
  allowed_regions: ["us-east-1", "us-east-2"]

# Worker in EU-West
scheduler:
  region: "eu-west-1"
  allowed_regions: ["eu-west-1", "eu-central-1"]
```

Jobs automatically route to workers in the matching region. EU data never leaves EU.

### 2. Lease Fencing

```go
// Scheduler prevents duplicate execution
lease, err := scheduler.AcquireLease(ctx, jobID, workerID, 5*time.Minute)
if err == ErrLeaseHeldByOther {
    log.Printf("Job %s already being processed by another worker, skipping", jobID)
    return nil // Another worker has it, don't touch
}
// We hold the lease — we own this job for the next 5 minutes
```

```go
// Renew lease while processing
for {
    select {
    case <-ctx.Done():
        // Job cancelled — release lease
        scheduler.ReleaseLease(ctx, jobID, workerID)
        return ctx.Err()
    case <-time.After(4 * time.Minute):
        // Renew before expiry
        scheduler.RenewLease(ctx, jobID, workerID, 5*time.Minute)
    }
}
```

Each job has a 5-minute lease. Only the lease holder can execute. Crashed worker loses lease → job reassigned.

### 3. SLA Quotas (Per-Region Rate Limiting)

```bash
# Per-region rate limiting
curl -X POST http://localhost:8080/api/sla/quotas \
  -H "Content-Type: application/json" \
  -d '{
    "agent_id": "inventory-agent",
    "region": "eu-west-1",
    "max_rpm": 1000,
    "max_daily": 50000
  }'
```

Burst of EU jobs → rate limiter queues them, doesn't overwhelm.

### 4. Durable Execution

```go
// Long-running job (2+ hours) survives worker crashes
job, err := runtime.SubmitJob(ctx, &Job{
    Name:   "optimize-warehouse-eu-17",
    Agent:  inventoryAgent,
    Input:  warehouseConfig,
    // Automatically checkpointed every step
})

// Worker crashes at hour 1?
// New worker picks up at hour 1 checkpoint — not from scratch
```

---

## Architecture

```
┌─────────────────────────────────────────────────────────┐
│                   Regional Workers                       │
├─────────────┬─────────────┬─────────────┬───────────────┤
│  us-east-1  │  us-west-2  │  eu-west-1  │   apac-east   │
│  (12 j/sec) │  (10 j/sec) │  (8 j/sec)  │   (15 j/sec)  │
│  [100% US]  │  [100% US]  │  [100% EU]  │  [100% APAC]  │
└─────────────┴─────────────┴─────────────┴───────────────┘
         │              │             │              │
         └──────────────┴─────────────┴──────────────┘
                           │
                    PostgreSQL
                 (Primary + Replica per region)
                    Lease Fencing
                   (prevents duplicates)
```

---

## Key Aetheris Features Used

| Feature | Benefit |
|---------|---------|
| **Lease Fencing** | Prevents duplicate processing across workers |
| **Region Scheduling** | Data residency compliance |
| **SLA Quotas** | Per-region rate limiting |
| **Durable Execution** | Long-running jobs survive crashes |
| **Multi-Worker** | Horizontal scaling |
| **Event Sourcing** | Full replay capability for debugging |

---

## Run This Example

```bash
# Start 3 workers in different regions
REGION=us-east-1 go run ./cmd/worker &
REGION=eu-west-1 go run ./cmd/worker &
REGION=apac-east go run ./cmd/worker &

# Submit 1000 inventory jobs
for i in $(seq 1 1000); do
  curl -X POST http://localhost:8080/api/jobs \
    -d "{\"agent\":\"inventory-agent\",\"region\":\"eu-west-1\",\"input\":{\"warehouse\":\"eu-$i\"}}"
done

# Observe: EU jobs only processed by EU worker, leases prevent duplicates
aetheris jobs list --region eu-west-1
```

---

## Learn More

- [Regional Scheduling](../guides/deployment.md)
- [Lease Fencing](../guides/runtime-guarantees.md)
- [SLA Quota Manager](../guides/sla-management.md)
- [Multi-Region Deployment](../guides/multi-region-deployment.md)
