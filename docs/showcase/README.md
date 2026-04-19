# User Case Studies

Real-world examples of how organizations use Aetheris to power their AI agents.

> **Want to share your story?** Open a PR or start a [Discussion](https://github.com/Colin4k1024/Aetheris/discussions/category/show-and-tell).

---

## Case Study Template

Want to submit a case study? Use this template:

```markdown
## [Company/Project Name]

**Industry:** <e.g., Fintech, Healthcare, E-commerce>
**Use Case:** <Brief description>
**Results:** <Key metrics>

### Challenge
<What problem did they solve?>

### Solution
<How did they use Aetheris?>

### Results
<What were the outcomes?>
```

---

## Technical Examples (Not Real Companies)

> **⚠️ The following are technical demonstration examples, not real company case studies.**
> These illustrate how Aetheris features solve specific production problems.
> To submit a real case study, see [below](#submit-your-case-study).

### 1. Financial Trading Agent — At-Most-Once Execution

**Industry:** Fintech
**Use Case:** Autonomous portfolio rebalancing with zero duplicate trades
**Key Features:** Effects Ledger, Durable Execution, Evidence Chain

#### Challenge
AutoFinance needed an AI agent to handle portfolio rebalancing decisions that must execute reliably even during market volatility. Previous solutions using LangGraph experienced:
- Lost execution state during worker restarts
- Duplicate trades during retry scenarios
- No audit trail for regulatory compliance

#### Solution
Built on Aetheris with:
- **Durable Execution**: Job state persisted to PostgreSQL, survives any process restart
- **At-Most-Once Trades**: Effects Ledger ensures each trade executes exactly once
- **Evidence Chain**: Complete audit trail for regulatory compliance

#### Results
- **99.9%** execution reliability (vs. 85% previous)
- **0** duplicate trades in 6 months
- **100%** audit compliance with evidence exports

---

### 2. Healthcare Triage Agent — Human-in-the-Loop

**Industry:** Healthcare
**Use Case:** Multi-step patient triage with human-in-the-loop approval
**Key Features:** StatusParked, State Checkpoints, Evidence Export

#### Challenge
HealthAI's triage assistant needed to:
- Collect patient symptoms across multiple steps
- Wait for human doctor approval before final diagnosis
- Maintain HIPAA-compliant audit logs

#### Solution
Leveraged Aetheris features:
- **Human-in-the-Loop**: `StatusParked` for waiting approval
- **State Checkpoints**: Patient data preserved across pauses
- **Audit Export**: Evidence packages for compliance

#### Results
- **40%** faster average triage time
- **100%** of decisions have full trace
- **0** data loss incidents during approval waits

---

### 3. Multi-Region Inventory Optimization — Horizontal Scaling

**Industry:** Logistics
**Use Case:** Autonomous inventory replenishment across 50 warehouses
**Key Features:** Regional Scheduling, Lease Fencing, SLA Quotas, Multi-Worker

#### Challenge
LogiShip manages inventory across 50 warehouses with:
- Long-running optimization tasks (hours)
- Multiple workers processing different warehouses
- Need for regional data residency

#### Solution
Aetheris regional scheduling + multi-worker architecture:
- **Region-Aware Scheduling**: Jobs routed to local workers
- **Lease Fencing**: Prevents duplicate processing
- **SLA Quotas**: Per-region rate limiting

#### Results
- **25%** reduction in stockouts
- **$2M** annual savings from optimized inventory
- **99.99%** job completion rate

---

## Use Case Categories

| Category | Examples | Aetheris Features Used |
|----------|----------|----------------------|
| **Financial Trading** | Portfolio rebalancing, risk analysis | At-Most-Once, Audit |
| **Healthcare** | Patient triage, diagnosis assistance | Human-in-the-Loop, Evidence |
| **Customer Support** | Ticket escalation, auto-resolution | Durable Execution, Trace |
| **Data Processing** | ETL pipelines, report generation | Multi-worker, Checkpoints |
| **DevOps** | Infrastructure automation, incident response | Durable Execution, Recovery |

---

## Submit Your Case Study

1. Fork the repo
2. Add your case study to this page
3. Open a PR

Or start a [Discussion](https://github.com/Colin4k1024/Aetheris/discussions/category/show-and-tell) to share your experience!

---

*More case studies coming soon!*
