# Aetheris v2.4.0 Release Notes

**Release Date:** 2026-03-24

---

## Highlights

Aetheris v2.4.0 introduces the **Human-in-the-Loop (HITL) Approval Engine Phase 1** — a structured approval request API built on top of the existing Wait/Signal primitives. This release lays the foundation for enterprise compliance workflows where human sign-off is required before AI agents execute high-stakes actions.

---

## What's New

### HITL Approval Engine (Phase 1)

- **Structured ApprovalRequest API**: `POST /api/approvals`, `GET /api/approvals`, `GET /api/approvals/:id`
- **Approval Actions**: `POST /api/approvals/:id/approve`, `POST /api/approvals/:id/reject`
- **Approval Event Types**: `approval_requested` and `approval_completed` events written to the job event store
- **ApprovalStore Interface**: Pluggable storage backend (`NewMemStore()` for testing/single-instance; PostgreSQL-backed store for production)
- **Automatic Job Resume**: When an approval is completed, `wait_completed` is appended to the job event stream and the job is re-queued automatically
- **Idempotent Signal Handling**: Leverages existing signal inbox for at-least-once delivery

### New Packages

| Package | Description |
|---------|-------------|
| `internal/agent/approval` | ApprovalRequest model, ApprovalStore interface, in-memory implementation |

---

## API Endpoints

| Method | Path | Description |
|--------|------|-------------|
| `POST` | `/api/approvals` | Create an approval request |
| `GET` | `/api/approvals` | List pending approvals (filter by `job_id` or `approver_id`) |
| `GET` | `/api/approvals/:id` | Get a specific approval request |
| `POST` | `/api/approvals/:id/approve` | Approve an approval request |
| `POST` | `/api/approvals/:id/reject` | Reject an approval request |

---

## Approval Model

```
ApprovalRequest {
  ID, JobID, NodeID, CorrelationKey
  ApproverType: anyone | specific | role
  ApproverID, Role
  Title, Description, Payload
  Status: pending | approved | rejected | expired | delegated
  ApprovalResponse (after completion)
  ExpiresAt, CreatedAt, UpdatedAt
}
```

---

## Validation

- `make fmt-check` ✓
- `make vet` ✓
- `make test` (full suite with race detector) ✓
- `make build` ✓

---

## Installation

### Binary

Download from [GitHub Releases](https://github.com/Colin4k1024/Aetheris/releases)

### From Source

```bash
git clone https://github.com/Colin4k1024/Aetheris
cd Aetheris
make build
```

### Docker

```bash
docker pull aetheris/runtime:v2.4.0
```
