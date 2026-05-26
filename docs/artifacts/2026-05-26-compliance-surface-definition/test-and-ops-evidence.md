---
artifact: test-and-ops-evidence
task: compliance-surface-definition
date: 2026-05-26
role: qa-engineer
status: draft-for-implementation
---

# Test and Ops Evidence

## Test Matrix

| Layer | Required coverage | Current status |
|---|---|---|
| `pkg/compliance` unit tests | template versioning, unsupported controls, signed evidence binding, metrics source, invalid time range, unsupported standard | partially present |
| HTTP contract tests | missing evidence rejected, unsigned evidence rejected, signature-invalid evidence rejected, package ID mismatch rejected, valid report accepted | present for current report path |
| CLI tests | command parsing, local evidence verification, report generation, output write, exit codes | missing |
| Runtime integration | export evidence ZIP -> verify signature -> generate report | missing as a single CLI path |
| Release drill | package + HTTP report gate writes an artifact | present for current report path |
| Ops runbook evidence | key custody/rotation references, generated report artifact, unsupported controls visible | partially present |

## Required New Tests

### CLI Unit Tests

- `aetheris compliance templates` prints built-in template metadata.
- `aetheris compliance report` rejects missing `--tenant`.
- `aetheris compliance report` rejects missing `--standard`.
- `aetheris compliance report` rejects missing `--evidence`.
- `aetheris compliance report` rejects missing `--public-key`.
- `aetheris compliance report` exits with code `2` when evidence verification fails.
- `aetheris compliance report` writes valid JSON when evidence verification succeeds.

### Package Tests

- Report generation preserves the evidence package root hash.
- Report generation preserves signer key ID when supplied.
- Unsupported controls are included in both `unsupported_controls` and `summary`.
- Compliance notice is always present.

### Integration Test

Create one test or drill path that performs:

```text
fixture job events
  -> evidence ZIP export
  -> signed verification
  -> compliance report generation
  -> JSON schema/field assertions
```

Minimum assertions:

- evidence package is signed
- signature verifies against the supplied public key
- report root hash matches verified root hash
- report template version is non-empty
- unsupported controls are visible
- report does not claim legal certification

## Verification Commands

Current gate:

```bash
go test ./pkg/compliance ./internal/api/http
./scripts/release-compliance-report-drill.sh
```

Target gate after CLI implementation:

```bash
go test ./pkg/compliance ./internal/api/http ./cmd/cli
./scripts/release-compliance-report-drill.sh
```

The release drill must include the offline CLI command once implemented.

## Ops Evidence

Each release drill artifact must record:

- timestamp
- git commit
- Go version
- tested package list
- HTTP contract test result
- CLI contract test result after CLI implementation
- evidence package signature status
- template standard and version
- sample unsupported-control count
- generated report path

## Runbook Updates Required

Before promotion, update:

- `docs/guides/compliance-reporting.md`
- `docs/guides/cli.md`
- `docs/reference/api-contract.md`
- `docs/releases/compliance-report-release-drill.md`
- `scripts/release-compliance-report-drill.sh`
- `Makefile`

## Release Decision Rule

The broader `pkg/compliance` surface remains `prototype` until the offline CLI path and release drill are implemented. Only the evidence-bound report generator may remain `integrated`.

Do not mark compliance as `production-ready` unless release evidence proves the full operational path, including key verification, artifact generation, failure handling, and visible unsupported controls.
