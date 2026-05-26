---
artifact: compliance-surface-definition
date: 2026-05-26
role: architect
status: draft-for-implementation
---

# Compliance Surface Definition

This artifact defines the next vertical slice for the broader `pkg/compliance` surface.

## Decision

Keep `pkg/compliance` as `prototype` except for the already integrated signed-evidence-bound report generator.

Promote only one bounded slice next: offline compliance report generation from a verified evidence package. The slice must have a real runtime path, API/CLI contract, tests, and operations evidence before any status change.

## Artifacts

- [runtime-path](./runtime-path.md)
- [api-cli-contract](./api-cli-contract.md)
- [test-and-ops-evidence](./test-and-ops-evidence.md)

## Boundaries

- Aetheris can prove runtime evidence integrity, tenant-scoped runtime events, RBAC/redaction/retention behavior, and explicit unsupported controls.
- Aetheris cannot certify legal compliance, customer policy completeness, external GRC evidence, workforce processes, vendor controls, or legal interpretations.
- `POST /api/compliance/apply` is not promotable as-is because it sounds mutating but currently performs an in-memory check. It must either remain experimental or be replaced by an explicit `check`/`preview` contract.
