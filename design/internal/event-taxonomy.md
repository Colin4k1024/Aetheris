# Event Taxonomy

This document classifies Aetheris events by architectural responsibility. `internal/runtime/jobstore/event.go` still defines the concrete event names, but not every event has the same compatibility or replay weight.

## Event Classes

| Class | Purpose | Replay impact | Compatibility rule | Examples |
|---|---|---|---|---|
| Execution History | Reconstruct and advance durable execution | Critical | Backward-compatible forever, schema version required for breaking payload changes | `job_created`, `plan_generated`, `node_started`, `node_finished`, `command_emitted`, `command_committed`, `tool_invocation_started`, `tool_invocation_finished`, `step_committed`, `job_completed`, `job_failed` |
| Coordination | Lease, queue, wait, signal, resume | Critical for scheduling | Backward-compatible; must preserve idempotency and correlation semantics | `job_queued`, `job_leased`, `job_running`, `job_waiting`, `wait_completed`, `job_parked`, `job_resumed`, `agent_message` |
| Recorded Effects | Capture non-deterministic values used by replay | Critical when used | Payload must be stable and injectible | `timer_fired`, `random_recorded`, `uuid_recorded`, `http_recorded` |
| Trace Narrative | Explain behavior to users and operators | Non-critical | Evolvable with additive fields | `state_checkpointed`, `agent_thought_recorded`, `decision_made`, `tool_selected`, `tool_result_summarized`, `reasoning_snapshot`, `memory_read`, `memory_write`, `plan_evolution` |
| Audit / Evidence | Compliance, export, access history, proof | Non-critical to execution, critical to audit | Schema version recommended; retention policy required | `access_audited`, `evidence_export_requested`, `evidence_export_completed`, `critical_decision_made`, `human_approval_given`, `payment_executed`, `email_sent` |
| Retention / Lifecycle | Archive/delete lifecycle | Non-critical to replay after retention boundary | Must not erase active execution requirements | `job_archived`, `job_deleted` |
| Ledger Markers | Side-effect arbitration and proof | Critical when ledger events are enabled | Must align with Invocation Ledger state | `ledger_acquired`, `ledger_committed` |

## Rules

1. Replay code must depend only on Execution History, Coordination, Recorded Effects, and Ledger Markers.
2. Trace Narrative events must never be required for correctness.
3. Audit / Evidence events may be required for compliance export, but must not change execution behavior.
4. New event types must declare their class in code comments and design docs.
5. Breaking payload changes to Execution History require a schema version and a migration or compatibility reader.

## Current Concern

`EventType` currently hosts all classes in one enum. That is acceptable for now, but the class distinction must be preserved in documentation, tests, and API contracts. A future storage split can move Trace and Audit events to separate streams without changing the execution semantics.
