---
artifact: api-cli-contract
task: compliance-surface-definition
date: 2026-05-26
role: architect
status: draft-for-implementation
---

# API and CLI Contract

## Contract Decision

The stable product contract for the next slice should be CLI-first and evidence-first. HTTP compliance endpoints remain experimental until they have the same offline verification semantics and compatibility tests as the CLI.

## HTTP Surface

All compliance HTTP endpoints remain behind `api.forensics.experimental=true`.

### `GET /api/compliance/templates`

Purpose: list available built-in templates.

Response:

```json
{
  "templates": [
    {
      "name": "GDPR",
      "standard": "GDPR",
      "version": "2026.05",
      "retention_days": 365,
      "export_format": "json"
    }
  ]
}
```

Compatibility:

- Optional fields may be added.
- Existing field names and primitive types must not change while the endpoint is documented.

### `POST /api/compliance/report`

Purpose: generate an auditor-facing report bound to a previously verified signed evidence package.

Request:

```json
{
  "tenant_id": "tenant-a",
  "standard": "SOX",
  "start": "2026-05-01",
  "end": "2026-05-26",
  "evidence_package_id": "job-123-evidence.zip",
  "evidence_verification": {
    "package_id": "job-123-evidence.zip",
    "job_id": "job-123",
    "root_hash": "abc123",
    "verified": true,
    "signed": true,
    "signature_valid": true,
    "signer_key_id": "release-key-2026-05",
    "verified_at": "2026-05-26T10:00:00Z"
  }
}
```

Required validation:

- `tenant_id` and `standard` are required.
- `evidence_package_id` is required.
- `evidence_verification.package_id`, when supplied, must match `evidence_package_id`.
- `verified`, `signed`, and `signature_valid` must be true.
- `root_hash` must be non-empty.
- unsupported `standard` values return `400`.

Response:

```json
{
  "tenant_id": "tenant-a",
  "standard": "SOX",
  "template_name": "SOX",
  "template_version": "2026.05",
  "compliance_rate": 75,
  "controls": [],
  "summary": {
    "compliant": 3,
    "unsupported": 1
  },
  "unsupported_controls": [
    {
      "control_id": "SOX.X",
      "reason": "Control is outside automated Aetheris evidence coverage"
    }
  ],
  "evidence_package_id": "job-123-evidence.zip",
  "evidence_verification": {
    "package_id": "job-123-evidence.zip",
    "root_hash": "abc123",
    "verified": true,
    "signed": true,
    "signature_valid": true
  },
  "compliance_notice": "Aetheris reports summarize runtime evidence only; they are not legal compliance certifications."
}
```

### `POST /api/compliance/apply`

Current behavior: performs an in-memory framework check and returns a summary.

Decision: do not promote this endpoint as a stable or integrated API. The verb `apply` implies a mutating policy action, but the current implementation does not mutate runtime config, tenant policy, storage, or enforcement state.

Allowed next steps:

- Keep it experimental and document it as legacy/prototype.
- Replace it with `POST /api/compliance/check` or `POST /api/compliance/preview`.
- Add a real policy-application runtime path before using the `apply` name.

## CLI Surface

### `aetheris compliance templates`

Purpose: list local compliance templates.

Output:

```json
{
  "templates": [
    {
      "name": "SOX",
      "standard": "SOX",
      "version": "2026.05"
    }
  ]
}
```

### `aetheris compliance report`

Purpose: generate a compliance report offline from a signed evidence ZIP.

Command:

```bash
aetheris compliance report \
  --tenant tenant-a \
  --standard SOX \
  --evidence ./job-123-evidence.zip \
  --public-key "$AETHERIS_EVIDENCE_PUBLIC_KEY" \
  --start 2026-05-01 \
  --end 2026-05-26 \
  --output ./compliance-report.json
```

Required flags:

- `--tenant`
- `--standard`
- `--evidence`
- `--public-key`

Optional flags:

- `--start`
- `--end`
- `--output`; stdout when omitted
- `--format json`; only `json` is accepted for the first implementation

Exit codes:

- `0`: report generated
- `1`: invalid flags, unsupported standard, invalid time range, output write failure
- `2`: evidence verification failed

Compatibility:

- New optional flags may be added.
- Existing required flags must not change names in a `2.x` line once implemented.
- The command must be deterministic for the same evidence package, template version, and time range except `generated_at`.

## Auth and Safety

- HTTP endpoints must remain tenant-scoped via `X-Tenant-ID` or explicit `tenant_id`.
- CLI offline mode must not call the API unless an explicit future `--api-url` mode is added.
- Reports must always include the compliance notice.
- Reports must never hide unsupported controls by folding them into compliant controls.
