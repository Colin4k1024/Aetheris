---
artifact: execute-log
task: architecture-review
date: 2026-05-25
role: architect
status: completed
---

# Execute Log: Architecture Review Actions

## Completed Actions

| ID | Action | Result |
|---|---|---|
| P0-01 | Add guarantee matrix | Added [guarantee-matrix.md](../../guides/guarantee-matrix.md) and linked it from README/runtime docs |
| P0-02 | Clarify `external_http` boundary | Updated [external-http-agent.md](../../adapters/external-http-agent.md) with Level 1 migration semantics |
| P0-03 | Clarify Eino/Aetheris authority split | Updated [design/core.md](../../../design/core.md) with `Eino builds; Aetheris executes durably` chain |
| P0-04 | Resolve Effect Store ordering conflict | Added [effect-store-contract.md](../../../design/internal/effect-store-contract.md), marked old docs historical, and aligned Tool success path implementation |
| P1-01 | Classify event types | Added [event-taxonomy.md](../../../design/internal/event-taxonomy.md) |
| P1-02 | Add Job lifecycle doc | Added [job-lifecycle.md](../../guides/job-lifecycle.md) |
| P1-03 | Document production runtime gates | Added [production-runtime-gates.md](../../guides/production-runtime-gates.md) and linked existing API/Worker startup gates from config docs |
| P1-04 | Define external agent migration path | Added [external-http-migration.md](../../adapters/external-http-migration.md) with Level 1/2/3 capability boundaries |
| P1-05 | Add runtime cost model | Added [runtime-cost-model.md](../../guides/runtime-cost-model.md) for event/write amplification and sizing questions |
| P2-01 | Define prototype promotion slices | Added [prototype-promotion-backlog.md](prototype-promotion-backlog.md) and [ADR-0001](../../adr/ADR-0001-runtime-first-promotion-policy.md) |
| P2-02 | Align API stable/experimental boundary | Moved forensics, evidence graph, audit log, compliance, and AI-forensics endpoints to experimental API contract surface |
| P2-03 | Add experimental route regression guard | Extended router tests to keep 3.0 candidate routes disabled unless `api.forensics.experimental=true` |
| P3-01 | Implement signed evidence export slice | Added `security.evidence_signing`, signed `proof.json` export, CLI public-key verification, and [evidence-signing.md](../../guides/evidence-signing.md) |
| P3-02 | Productionize signed evidence release gate | Added key custody/rotation runbooks, evidence signing release drill, release checklist entries, and release script integration |
| P3-03 | Productionize forensics read model gate | Added [forensics-read-model.md](../../guides/forensics-read-model.md), tenant isolation/pagination/large-event tests, release drill, and release script integration |
| P3-04 | Productionize RBAC/redaction/retention hardening gate | Added [rbac-redaction-retention-hardening.md](../../guides/rbac-redaction-retention-hardening.md), redacted evidence export, retention replay invariant, release drill, and release script integration |
| P3-05 | Promote compliance reports as evidence-bound report generators | Added [compliance-reporting.md](../../guides/compliance-reporting.md), signed evidence binding, template versioning, explicit unsupported controls, HTTP export tests, and release drill integration |
| P3-06 | Promote AI forensics detection as an eval-gated detector | Added [ai-forensics-eval.md](../../guides/ai-forensics-eval.md), golden eval dataset, false-positive budget, severity mapping, retry/tamper signal extraction, HTTP tests, and release drill integration |
| P3-07 | Gate distributed verifier promotion with operational readiness evidence | Added [distributed-verifier-readiness.md](../../guides/distributed-verifier-readiness.md), root-hash consensus/divergence tests, readiness assessment, and release drill integration |
| P3-08 | Define monitoring quality scorer SRE semantics | Added [monitoring-quality-scorer.md](../../guides/monitoring-quality-scorer.md), healthy/degraded/critical/noisy alert semantics, tests, and release drill integration |

## Code Change

`internal/agent/runtime/executor/node_adapter.go` now persists the completed Tool effect before appending `tool_invocation_finished` / `command_committed`, matching the current strong Replay contract.

## Verification

```bash
/usr/local/go/bin/gofmt -w internal/agent/runtime/executor/node_adapter.go
/usr/local/go/bin/go test ./internal/agent/runtime/executor
/usr/local/go/bin/go test ./internal/app/api
/usr/local/go/bin/go test ./internal/app/api ./internal/app/worker ./internal/agent/runtime/executor
/usr/local/go/bin/gofmt -w internal/api/http/router_test.go
/usr/local/go/bin/go test ./internal/api/http ./internal/app/api ./internal/app/worker ./internal/agent/runtime/executor
/usr/local/go/bin/gofmt -w pkg/proof/types.go pkg/proof/export.go pkg/proof/verify.go pkg/config/config.go internal/api/http/handler.go internal/api/http/forensics.go internal/app/api/app.go cmd/cli/main.go cmd/cli/main_test.go internal/api/http/forensics_test.go internal/app/api/app_utils_test.go
/usr/local/go/bin/go test ./pkg/proof ./pkg/config ./internal/api/http ./internal/app/api ./cmd/cli
/usr/local/go/bin/go test ./...
./scripts/release-evidence-signing-drill.sh
./scripts/release-forensics-read-model-drill.sh
./scripts/release-rbac-redaction-retention-drill.sh
./scripts/release-compliance-report-drill.sh
./scripts/release-ai-forensics-eval-drill.sh
./scripts/release-distributed-verifier-drill.sh
./scripts/release-monitoring-quality-scorer-drill.sh
```

Result: executor, app/api, and app/worker package tests passed.
Final batched verification: api/http, app/api, app/worker, and executor package tests passed.
Signed evidence verification: proof, config, api/http, app/api, and cli package tests passed.
Full repository verification: `go test ./...` passed.
Evidence signing release drill: passed and wrote an artifact under `artifacts/release/`.
Forensics read model drill: passed and wrote an artifact under `artifacts/release/`.
RBAC/redaction/retention drill: passed and wrote an artifact under `artifacts/release/`.
Compliance report drill: passed and wrote an artifact under `artifacts/release/`.
AI forensics eval drill: passed and wrote an artifact under `artifacts/release/`.
Distributed verifier drill: passed and wrote an artifact under `artifacts/release/`.
Monitoring quality scorer drill: passed and wrote an artifact under `artifacts/release/`.
