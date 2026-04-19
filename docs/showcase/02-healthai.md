# Example: Healthcare Triage Assistant — Human-in-the-Loop Approval

> **⚠️ This is a technical example/demonstration, not a real company case study.**
>
> For real-world case studies, see [Discussions › Show & Tell](https://github.com/Colin4k1024/Aetheris/discussions/category/show-and-tell).

**Type:** Example / Technical Demo
**Industry:** Healthcare
**Use Case:** Multi-step patient triage with human-in-the-loop approval
**Aetheris Features:** StatusParked (Human-in-the-Loop), State Checkpoints, Evidence Export

---

## Problem Statement

A patient triage agent must:

- Collect symptoms across multiple steps (cannot lose state if worker restarts)
- Pause and wait for human doctor approval before taking action
- Maintain HIPAA-compliant audit logs

```
❌ Worker restart → patient data lost → start triage from scratch
❌ No native pause/resume → custom queue spaghetti code
❌ Audit logs incomplete → HIPAA violation
```

---

## Solution

### 1. Human-in-the-Loop with StatusParked

```go
// Agent pauses automatically when human approval is needed
result, err := agent.Run(ctx, &agent.Input{
    Message:    "Patient presents with chest pain",
    ParkOn:     []string{"prescribe_medication", "order_test"},
    ParkReason: "requires doctor approval",
})

// Job status becomes "parked" — no worker resources consumed
// Doctor reviews and approves via API
```

```bash
# Check job status
curl http://localhost:8080/api/jobs/{job_id}
# {"status": "parked", "park_reason": "requires doctor approval"}

# Doctor reviews and resumes
curl -X POST http://localhost:8080/api/jobs/{job_id}/resume \
  -H "Content-Type: application/json" \
  -d '{"decision": "approved", "notes": "Proceed with ECG"}'

# Agent resumes from checkpoint with doctor's decision in context
```

### 2. State Checkpoints

Every step automatically checkpointed to PostgreSQL:

```go
type PatientState struct {
    Name       string    `json:"name"`
    Symptoms   []string  `json:"symptoms"`
    Vitals     Vitals    `json:"vitals"`
    Assessment string    `json:"assessment,omitempty"`
    DoctorNote string    `json:"doctor_note,omitempty"`
    // Checkpointed automatically after each step
}

// After pause + resume, agent has full context:
// "Patient John: symptoms=[chest_pain, shortness_of_breath],
//  vitals={bp:140/90}, doctor_note: Proceed with ECG"
```

### 3. Evidence Export for Compliance

```bash
# Export complete audit trail
curl -X POST http://localhost:8080/api/jobs/{job_id}/export \
  -o patient-123-evidence.zip

# Evidence package contains:
# - All events with timestamps
# - State snapshots at each checkpoint
# - LLM prompts/responses
# - Doctor approval decisions
# - Cryptographic integrity hash
```

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
                    │     Wait       │◀────  (auto-pauses,
                    │                 │       no resources)
                    │  3. Resume     │
                    │     with       │
                    │     Approval   │
                    └─────────────────┘
```

---

## Key Aetheris Features Used

| Feature | Benefit |
|---------|---------|
| **StatusParked** | Native wait-for-human, no custom queue |
| **State Checkpoints** | Preserve patient data across pauses/restarts |
| **Evidence Chain** | Complete audit trail for HIPAA compliance |
| **RBAC** | Only authorized doctors can approve |

---

## Run This Example

```bash
cd examples/human-approval-agent
go run . --patient-symptoms "chest_pain,shortness_of_breath"

# Job pauses at triage decision step
# Check status: curl http://localhost:8080/api/jobs/{job_id}

# Doctor approves via API or CLI
aetheris jobs approve {job_id} --notes "Proceed with ECG"

# Agent resumes with approval in context
```

---

## Learn More

- [Human-in-the-Loop](../guides/getting-started-agents.md)
- [Evidence Package Export](../guides/m3-evidence-graph-guide.md)
- [RBAC Configuration](../guides/m2-rbac-guide.md)
