# Aetheris Strategy and User Stories

This document aligns Aetheris product narrative with the B2D transformation goal:
from a cloud-hosted SaaS experience to an enterprise AI agent runtime foundation.

## Strategic Positioning

- Aetheris is positioned as **B2D infrastructure** for enterprise AI agents.
- Primary users are enterprise developers and architects suffering from runtime fragility and audit pressure in production.
- Product focus is durable execution, deterministic recovery, compliance-grade traceability, and local-first deployment.

## Three Core User Stories

### 1) Long Lifecycle and Human-in-the-Loop

As an enterprise application developer, I need to build workflows such as travel planning or large refund approvals.  
I need the agent to pause at critical nodes, release compute resources, and wait for human approval.  
After approval, execution must resume losslessly from the breakpoint without re-spending previous LLM tokens.

### 2) Financial and Legal Compliance

As a financial compliance/risk engineer, I need an AI agent for diff analysis and anomaly detection.  
Every API call and reasoning decision must be recorded in an immutable log so we can fully reconstruct and explain why the AI acted in a certain way during audits.

### 3) Data Privacy and Local Deployment

As an enterprise architect focused on data sovereignty, I need a lightweight agent engine that runs in private networks or air-gapped environments.  
Core business context must stay local, avoiding continuous upload to external hosted platforms.

## User Story to Capability Mapping

| User Story                         | Runtime Capability                                                                  | Key Locations                                                                                                                   |
| ---------------------------------- | ----------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------- |
| Long lifecycle + human-in-the-loop | Wait/Signal semantics, parked status, checkpoint recovery, replay injection         | `internal/agent/runtime/executor/wait_signal.go`, `internal/api/http/handler.go`, `docs/guides/runtime-guarantees.md`           |
| Financial/legal compliance         | Event sourcing, trace/proof export, auditable runtime events                        | `internal/runtime/jobstore`, `pkg/proof`, `docs/guides/runtime-guarantees.md`, `docs/roadmaps/EVIDENCE-PACKAGE-FOR-AUDITORS.md` |
| Data privacy + local deployment    | Embedded local stores, local-first mode, no external DB dependency in embedded mode | `internal/agent/job/embedded_store.go`, `docs/embedded-mode.md`                                                                 |

## Four-Phase Strategic Refactor

| Phase   | Goal                                               | Status                              | Evidence                                                                                                                                  |
| ------- | -------------------------------------------------- | ----------------------------------- | ----------------------------------------------------------------------------------------------------------------------------------------- |
| Phase 1 | Lightweight local event/state storage              | Implemented                         | Embedded JobStore + local persistence path in `internal/agent/job/embedded_store.go`; `jobstore.type=embedded` in `docs/embedded-mode.md` |
| Phase 2 | Durable execution API (interrupt/resume semantics) | Implemented                         | Pause/resume/signal behavior and waitlike execution in `internal/api/http/handler.go` and `internal/agent/runtime/executor`               |
| Phase 3 | MCP standard integration                           | Implemented                         | MCP Host and tool adapter bridge in `internal/agent/tools/mcp_host.go`                                                                    |
| Phase 4 | Open Core business model                           | Implemented (boundary and baseline) | OSS vs Enterprise split in `docs/open-core-boundary.md`; RBAC/audit baseline in docs and runtime middleware                               |

## Alignment Principles

- Keep **runtime-first** APIs (`/api/runs`, `/api/jobs`) as the canonical execution path.
- Keep legacy `/api/agents/*` only as migration facade, not as strategic center.
- Prefer local-first deployment for privacy-sensitive enterprise environments.
- Preserve replay and at-most-once guarantees as non-negotiable product identity.

## Related Documents

- `README.md`
- `design/core.md`
- `docs/open-core-boundary.md`
- `docs/embedded-mode.md`
- `docs/guides/runtime-guarantees.md`
- `docs/roadmaps/CURRENT-STATUS-AND-FOCUS.md`
