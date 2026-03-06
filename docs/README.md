# Documentation

This directory is the entry point for architecture, usage, and API documentation.

## Quick start

Install **Go 1.25.7+**, clone the repo, then run:

```bash
go run ./cmd/api
```

Health check: `curl http://localhost:8080/api/health`. For full startup, environment variables, and typical flows see [guides/usage.md](guides/usage.md); for upload → retrieve E2E steps see [guides/test-e2e.md](guides/test-e2e.md).

## Project names

- **Aetheris** — Product/project name and CLI command (`aetheris`)
- **rag-platform** — go.mod module name (internal use only, not user-facing)

Environment variables use `AETHERIS_*` prefix (e.g., `AETHERIS_API_URL`).

## Version and changes

Recommended **Go 1.25.7+**, aligned with go.mod and CI.

- [CHANGELOG.md](../CHANGELOG.md) — Version history and notable changes (v0.8 persistent runtime, event JobStore, Job/Scheduler/Checkpoint/Steppable, v1 Agent API, TaskGraph execution layer, RulePlanner, planner selection, etc.)
- [STATUS.md](STATUS.md) — Single source of truth for current release status and post-2.0 evolution policy

## Documentation Structure

```
docs/
├── README.md          # This file
├── STATUS.md          # Current release status (authoritative)
├── guides/            # User guides and tutorials
├── reference/         # API and configuration reference
├── releases/          # Release notes and checklists
├── milestones/        # Milestone implementation summaries
├── roadmaps/         # Roadmap and planning documents
├── concepts/         # Concept and design discussions
├── blog/             # Technical blog articles
└── adapters/         # Integration adapters
```

## Blog

Technical blog articles covering Aetheris core concepts, use cases, and deep dives.

| Article                                                                        | Description                   |
| ------------------------------------------------------------------------------ | ----------------------------- |
| [blog/01-quick-start.md](blog/01-quick-start.md)                               | 5 分钟快速开始                |
| [blog/02-production-agents.md](blog/02-production-agents.md)                   | 生产级 AI Agent 构建指南      |
| [blog/03-observability.md](blog/03-observability.md)                           | 可观测性实战                  |
| [blog/04-enterprise-features.md](blog/04-enterprise-features.md)               | 企业级功能指南                |
| [blog/05-human-in-the-loop.md](blog/05-human-in-the-loop.md)                   | 审批流与 Wait/Signal          |
| [blog/06-at-most-once-ledger.md](blog/06-at-most-once-ledger.md)               | Invocation Ledger 原理        |
| [blog/07-event-sourcing-replay.md](blog/07-event-sourcing-replay.md)           | 事件溯源与 Replay 恢复        |
| [blog/08-multi-worker-lease-fencing.md](blog/08-multi-worker-lease-fencing.md) | 多 Worker 与 Lease Fencing    |
| [blog/09-compliance-evidence-chain.md](blog/09-compliance-evidence-chain.md)   | 合规审计与 Evidence Chain     |
| [blog/10-long-running-checkpoint.md](blog/10-long-running-checkpoint.md)       | 长任务与 Checkpoint 恢复      |
| [blog/11-when-to-choose-aetheris.md](blog/11-when-to-choose-aetheris.md)       | 选型对比：LangGraph、Temporal |

## Recommended reading order

- **Getting started**: This README → [blog/01-quick-start.md](blog/01-quick-start.md) → [guides/usage.md](guides/usage.md) → [design/core.md](../design/core.md)
- **Core concepts**: [blog/11-when-to-choose-aetheris.md](blog/11-when-to-choose-aetheris.md) (选型) → [blog/05-human-in-the-loop.md](blog/05-human-in-the-loop.md) (人机协同) → [blog/06-at-most-once-ledger.md](blog/06-at-most-once-ledger.md) (执行保证)
- **Recovery & Fault Tolerance**: [blog/07-event-sourcing-replay.md](blog/07-event-sourcing-replay.md) → [blog/10-long-running-checkpoint.md](blog/10-long-running-checkpoint.md) → [blog/08-multi-worker-lease-fencing.md](blog/08-multi-worker-lease-fencing.md)
- **Advanced**: [design/services.md](../design/services.md), [design/execution-guarantees.md](../design/execution-guarantees.md), [blog/09-compliance-evidence-chain.md](blog/09-compliance-evidence-chain.md)
- **Operations**: [guides/tracing.md](guides/tracing.md), [reference/config.md](reference/config.md), [guides/deployment.md](guides/deployment.md)

## Guides

User guides and tutorials for getting started and daily operations.

| Document                                                                         | Description                                                                                  |
| -------------------------------------------------------------------------------- | -------------------------------------------------------------------------------------------- |
| [guides/get-started.md](guides/get-started.md)                                   | Quick start guide                                                                            |
| [guides/usage.md](guides/usage.md)                                               | Startup, environment variables, typical flows, API endpoint summary, FAQ                     |
| [guides/getting-started-agents.md](guides/getting-started-agents.md)             | Agent development guide                                                                      |
| [guides/cli.md](guides/cli.md)                                                   | aetheris subcommands, install and run, REST API mapping                                      |
| [guides/sdk.md](guides/sdk.md)                                                   | High-level Agent API (NewAgent, RegisterTool, Run), comparison with Job/Runner               |
| [guides/examples.md](guides/examples.md)                                         | basic_agent, simple_chat_agent, streaming, tool, workflow purpose and run instructions       |
| [guides/test-e2e.md](guides/test-e2e.md)                                         | Upload → parse → split → index → retrieve (PDF / AGENTS.md)                                  |
| [guides/e2e-business-scenario-refund.md](guides/e2e-business-scenario-refund.md) | E2E business scenario example                                                                |
| [guides/deployment.md](guides/deployment.md)                                     | Compose / Docker / K8s overview and prerequisites                                            |
| [guides/observability.md](guides/observability.md)                               | Execution Trace UI (Job timeline, step latency, retry reasons), GET /api/jobs/:id/trace/page |
| [guides/tracing.md](guides/tracing.md)                                           | OpenTelemetry config, OTEL_EXPORTER_OTLP_ENDPOINT, local Jaeger                              |
| [guides/security.md](guides/security.md)                                         | Security baseline and release checklist                                                      |
| [guides/capacity-planning.md](guides/capacity-planning.md)                       | Capacity planning guide                                                                      |
| [guides/runtime-guarantees.md](guides/runtime-guarantees.md)                     | Runtime guarantees and semantics                                                             |
| [guides/troubleshooting.md](guides/troubleshooting.md)                           | Troubleshooting guide and FAQ                                                                |

### Feature Guides

M1-M4 milestone feature guides.

| Document                                                               | Description                                |
| ---------------------------------------------------------------------- | ------------------------------------------ |
| [guides/m2-rbac-guide.md](guides/m2-rbac-guide.md)                     | RBAC implementation guide                  |
| [guides/m2-redaction-guide.md](guides/m2-redaction-guide.md)           | Data redaction guide                       |
| [guides/m2-retention-guide.md](guides/m2-retention-guide.md)           | Data retention policy guide                |
| [guides/m3-evidence-graph-guide.md](guides/m3-evidence-graph-guide.md) | Evidence graph guide                       |
| [guides/m3-forensics-api-guide.md](guides/m3-forensics-api-guide.md)   | Forensics API guide                        |
| [guides/m3-ui-guide.md](guides/m3-ui-guide.md)                         | UI/UX implementation guide                 |
| [guides/m4-signature-guide.md](guides/m4-signature-guide.md)           | Digital signature guide                    |
| [guides/multi-region-deployment.md](guides/multi-region-deployment.md) | Multi-region deployment guide              |
| [guides/enterprise-integrations.md](guides/enterprise-integrations.md) | LDAP, queue, storage integrations          |
| [guides/sla-management.md](guides/sla-management.md)                   | SLA management and enforcement             |
| [guides/security-advanced.md](guides/security-advanced.md)             | Advanced security (mTLS, signing, secrets) |

## Reference

API and configuration reference documentation.

| Document                                               | Description                                                                     |
| ------------------------------------------------------ | ------------------------------------------------------------------------------- |
| [reference/config.md](reference/config.md)             | api.yaml, model.yaml, worker.yaml field reference and env vars                  |
| [reference/api-contract.md](reference/api-contract.md) | v2.2 stable/experimental API boundaries, compatibility and deprecation strategy |

## Releases

Release notes, checklists, and upgrade guides.

| Document                                                                       | Description                                                                         |
| ------------------------------------------------------------------------------ | ----------------------------------------------------------------------------------- |
| [releases/2.0-RELEASE-NOTES.md](releases/2.0-RELEASE-NOTES.md)                 | 2.0 release notes                                                                   |
| [releases/2.1-RELEASE-READY.md](releases/2.1-RELEASE-READY.md)                 | 2.1 release readiness                                                               |
| [releases/AETHERIS-2.1-RELEASE.md](releases/AETHERIS-2.1-RELEASE.md)           | Aetheris 2.1 release announcement                                                   |
| [releases/release-acceptance-v0.9.md](releases/release-acceptance-v0.9.md)     | v0.9 runtime correctness (Worker crash recovery, API restart, multi-Worker, Replay) |
| [releases/release-certification-1.0.md](releases/release-certification-1.0.md) | 1.0 release gate checklist                                                          |
| [releases/release-checklist-v1.0.md](releases/release-checklist-v1.0.md)       | Post-release checklist (core features, distributed, CLI/API, logging and docs)      |
| [releases/release-checklist-2.0.md](releases/release-checklist-2.0.md)         | 2.0 release checklist                                                               |
| [releases/upgrade-1.x-to-2.0.md](releases/upgrade-1.x-to-2.0.md)               | Upgrade and rollback guide                                                          |
| [releases/performance-baseline-2.0.md](releases/performance-baseline-2.0.md)   | Release performance baseline                                                        |
| [releases/runbook-failure-drills.md](releases/runbook-failure-drills.md)       | Failure drill runbook                                                               |

## Milestones

Milestone implementation summaries (M1-M4, 2.0).

| Document                                                                             | Description                |
| ------------------------------------------------------------------------------------ | -------------------------- |
| [milestones/m1-implementation-summary.md](milestones/m1-implementation-summary.md)   | M1 implementation summary  |
| [milestones/m2-implementation-summary.md](milestones/m2-implementation-summary.md)   | M2 implementation summary  |
| [milestones/m3-implementation-summary.md](milestones/m3-implementation-summary.md)   | M3 implementation summary  |
| [milestones/m4-implementation-summary.md](milestones/m4-implementation-summary.md)   | M4 implementation summary  |
| [milestones/2.0-implementation-summary.md](milestones/2.0-implementation-summary.md) | 2.0 implementation summary |
| [milestones/2.0-milestones-overview.md](milestones/2.0-milestones-overview.md)       | 2.0 milestones overview    |

## Roadmaps

Current roadmap and planning documents. Historical documents are archived in [roadmaps/archive/](roadmaps/archive/).

| Document                                                                               | Description                                |
| -------------------------------------------------------------------------------------- | ------------------------------------------ |
| [roadmaps/2.1-execution-plan.md](roadmaps/2.1-execution-plan.md)                       | 8-week execution plan (2026-02 to 2026-04) |
| [roadmaps/2026-Q1-ACTION-PLAN.md](roadmaps/2026-Q1-ACTION-PLAN.md)                     | 2026 Q1 action plan                        |
| [roadmaps/CURRENT-STATUS-AND-FOCUS.md](roadmaps/CURRENT-STATUS-AND-FOCUS.md)           | Current status and focus                   |
| [roadmaps/DEPLOYMENT-PRODUCTION.md](roadmaps/DEPLOYMENT-PRODUCTION.md)                 | Production deployment guide                |
| [roadmaps/EVIDENCE-PACKAGE-FOR-AUDITORS.md](roadmaps/EVIDENCE-PACKAGE-FOR-AUDITORS.md) | Evidence package for auditors              |

## Concepts

Concept and design discussion documents.

| Document                                                                                     | Description                                             |
| -------------------------------------------------------------------------------------------- | ------------------------------------------------------- |
| [concepts/devops.md](concepts/devops.md)                                                     | Eino Dev IDE plugin, visual orchestration and debugging |
| [concepts/adk.md](concepts/adk.md)                                                           | ADK integration                                         |
| [concepts/evidence-package.md](concepts/evidence-package.md)                                 | Evidence package documentation                          |
| [concepts/migration-to-m1.md](concepts/migration-to-m1.md)                                   | Migration guide to M1                                   |
| [concepts/next_plan.md](concepts/next_plan.md)                                               | Next plan discussion                                    |
| [concepts/improvement-checklist-1.0-to-2.0.md](concepts/improvement-checklist-1.0-to-2.0.md) | Improvement checklist from 1.0 to 2.0                   |

## Design Docs

Architecture and design documents in the `design/` directory. These are the **public** design docs intended for external readers. Internal implementation details and design notes live in [design/internal/](../design/internal/).

| Document                                                              | Description                                                                                     |
| --------------------------------------------------------------------- | ----------------------------------------------------------------------------------------------- |
| [design/core.md](../design/core.md)                                   | Overall architecture, layers, Agent Runtime and task execution, Pipeline and eino orchestration |
| [design/services.md](../design/services.md)                           | Multi-service architecture (api / agent / index)                                                |
| [design/execution-guarantees.md](../design/execution-guarantees.md)   | Trusted execution runtime guarantees and semantics                                              |
| [design/runtime-core-diagrams.md](../design/runtime-core-diagrams.md) | Runtime core: Runner–Ledger–JobStore sequence and StepOutcome state diagram                     |
| [design/aetheris-2.0-overview.md](../design/aetheris-2.0-overview.md) | Aetheris 2.0 feature overview and roadmap                                                       |
| [design/v2.md](../design/v2.md)                                       | 2.0 development roadmap and completed modules                                                   |
| [design/milestone.md](../design/milestone.md)                         | 2.0 compliance and audit milestone goals                                                        |

## Other Resources

| Resource                                 | Description                         |
| ---------------------------------------- | ----------------------------------- |
| [examples/](../examples/)                | Example code                        |
| [adapters/README.md](adapters/README.md) | Custom vs LangGraph migration paths |
| [deployments/](../deployments/)          | Docker, Compose, K8s directories    |
