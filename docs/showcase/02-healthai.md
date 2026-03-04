# HealthAI - Patient Triage Assistant

**Industry:** Healthcare
**Use Case:** Multi-step patient triage with human-in-the-loop approval
**Results:** 40% faster triage, 100% decision traceability

---

## Company Overview

HealthAI provides AI-powered healthcare decision support systems for clinics and hospitals. Their triage assistant helps nurses prioritize patient care based on symptoms and medical history.

---

## Challenge

HealthAI's triage assistant needed to:

| Requirement | Previous Solution Issues |
|------------|-------------------------|
| Collect symptoms across multiple steps | State lost when worker restarted |
| Wait for human doctor approval | No native support, built custom queue |
| HIPAA-compliant audit logs | Fragmented, incomplete |

> "Our previous solution couldn't handle the 'waiting for approval' scenario well. Nurses would lose their place if the system restarted." — Product Manager, HealthAI

---

## Solution

Leveraged Aetheris v2.0+ features:

### 1. Human-in-the-Loop with StatusParked

```bash
# Agent pauses and waits for doctor approval
curl -X POST http://localhost:8080/api/agents/agent-xxx/message \
  -d '{"message":"Patient presents with chest pain"}'

# Job pauses automatically when needing human input
# GET /api/jobs/job-xxx returns status: "parked"

# Doctor reviews and approves
curl -X POST http://localhost:8080/api/jobs/job-xxx/resume \
  -d '{"decision":"approved","notes":"Proceed with ECG"}'
```

### 2. State Checkpoints

```go
// Automatic state persistence at each step
type PatientState struct {
    Name         string    `json:"name"`
    Symptoms     []string  `json:"symptoms"`
    Vitals       Vitals    `json:"vitals"`
    Assessment   string    `json:"assessment,omitempty"`
    DoctorNote   string    `json:"doctor_note,omitempty"`
}
```

Every step checkpointed to PostgreSQL — no data loss during pause.

### 3. Evidence Export for HIPAA

```bash
# Export complete audit trail
curl -X POST http://localhost:8080/api/jobs/job-xxx/export \
  -o patient-123-evidence.zip

# Contains:
# - All events with timestamps
# - State snapshots
# - LLM prompts/responses
# - Doctor decisions
```

---

## Results

| Metric | Before | After |
|--------|--------|-------|
| Average triage time | 45 min | **27 min** |
| Data loss incidents | 12/month | **0** |
| Decision traceability | 60% | **100%** |
| HIPAA audit ready | No | **Yes** |

---

## Architecture

```
┌──────────────┐     ┌─────────────────┐     ┌──────────────┐
│   Patient   │────▶│   Aetheris     │────▶│   Doctor    │
└──────────────┘     │   Runtime      │     │   Review    │
                    │                 │     └──────────────┘
                    │  1. Collect    │
                    │     Symptoms   │
                    │                 │
                    │  2. Park &     │
                    │     Wait       │◀────
                    │                 │
                    │  3. Resume     │
                    │     with       │
                    │     Approval   │
                    └─────────────────┘
```

---

## Key Aetheris Features Used

| Feature | Benefit |
|---------|---------|
| **StatusParked** | Native wait-for-human support |
| **State Checkpoints** | Preserve patient data across pauses |
| **Evidence Chain** | Complete audit trail for HIPAA |
| **RBAC** | Doctor-only approval permissions |

---

## Learn More

- [Human-in-the-Loop](../guides/getting-started-agents.md)
- [Evidence Package Export](../guides/m3-evidence-graph-guide.md)
- [RBAC Configuration](../guides/m2-rbac-guide.md)
