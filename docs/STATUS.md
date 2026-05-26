# Aetheris Status (Single Source of Truth)

> Last updated: 2026-05-08
> Scope: repository status, release lane, and post-2.0 evolution policy.
> Current Version: **v2.5.3**

## 1. Purpose

This file is the authoritative status entry for `main`.

- If other roadmap/release docs conflict with this file, use this file as final truth.
- Historical docs remain valuable for context, but not for current release decisions.

## 2. Status Model

Every capability must be labeled with one of these states:

- `prototype`: package/design exists, not integrated into main API/CLI flow.
- `integrated`: connected to runtime/API/CLI, but not yet release-gated.
- `production-ready`: integrated and covered by release gates + drills.

## 3. Current Snapshot

### 3.1 Runtime lane (2.x)

- Durable execution / event sourcing: `production-ready`
- Deterministic replay + at-most-once tool execution: `production-ready`
- Signal + human-in-the-loop (at-least-once delivery): `production-ready`
- Observability summary/stuck endpoints: `integrated` (UI/SRE workflows still strengthening)
- Multi-Adapter Support: `production-ready` (LangChainGo, LangGraphGo, Google ADK, Genkit, Protocol-Lattice, LinGoose)

### 3.2 Forensics and compliance lane (M1-M3)

- Evidence export/verify: `production-ready`, including signed evidence ZIP export with env-injected Ed25519 keys and offline public-key verification
- Forensics query/evidence graph/audit-log APIs: `integrated` with read-model release drill; exposed only when `api.forensics.experimental=true`
- RBAC/redaction/retention: `production-ready` for tenant-scoped RBAC checks, redacted evidence export, and retention GC replay invariants
- Compliance reports: `integrated` as signed-evidence-bound report generators with template versioning and explicit unsupported controls; exposed only when `api.forensics.experimental=true`
- AI-forensics detection: `integrated` as an eval-gated detector with a golden dataset, false-positive budget, severity mapping, event-signal extraction, and release drill; API remains experimental
- Distributed verifier: `prototype` with root-hash comparison and readiness drill; promotion is blocked until saturation, lease, and recovery evidence exists
- Monitoring quality scorer: `prototype` offline SRE report utility with alert semantics and release drill; not wired into `/api/observability/*`

### 3.3 Enterprise lane (M4 / 3.0 candidates)

The following are currently treated as `prototype` unless explicitly promoted by a vertical slice:

- `pkg/compliance`

`pkg/signature` is promoted only for evidence ZIP signing. Broader signing/key-management surfaces remain outside GA scope.

`prototype` means “technical reserve”, not “GA”.

## 4. Active Release Strategy

Current release lane: **Operational Runtime first (2.x)**.

Priority order:

1. Runtime correctness and recoverability
2. Multi-tenant safety and operational readiness
3. Release gates and runbooks
4. Selective 3.0 productization by vertical slices

## 5. 2.x Exit Gates (Must Pass)

A 2.x production release requires all of:

- CI green (`build`, `vet`, `test`, Postgres integration)
- Release checklist completed (`docs/release-checklist-2.0.md`)
- P0 performance gate report available
- P0 failure drill report available (including DB outage drill in release rehearsal)
- Upgrade/rollback runbook validated
- Security baseline checklist completed

If any gate is missing, release status stays `integrated`, not `production-ready`.

## 6. 3.0 Promotion Policy

A 3.0 capability may be promoted from `prototype` to `integrated` only when all are present:

- config: documented and wired in runtime config
- schema/storage: migration and backward compatibility notes
- API: routed and contract-documented
- CLI: command surface implemented
- tests: unit + integration + failure-path coverage
- ops: observability and runbook updates

No “docs-only complete” claims without these artifacts.

Promotion order and required vertical slices are tracked in [prototype-promotion-backlog.md](artifacts/2026-05-25-architecture-review/prototype-promotion-backlog.md).

## 7. Doc Governance Rules

To avoid roadmap confusion:

- `docs/STATUS.md` is the single current-state source.
- Roadmap docs should focus on plan and history, not final state authority.
- “Complete / Ready” wording is allowed only when status is `production-ready`.

## 8. Next Focus (Recommended)

1. Harden 2.x gates in CI/release pipeline (make P0 gates non-optional for release jobs).
2. Close multi-tenant operational gaps (isolation tests, authz drills, runbook evidence).
3. Next 3.0 slice candidate: implement the offline compliance report CLI slice defined in [compliance-surface-definition](artifacts/2026-05-26-compliance-surface-definition/README.md).
