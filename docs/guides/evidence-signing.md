# Evidence Signing

Aetheris can sign exported evidence ZIPs with Ed25519. Signing is optional by default and is configured on the API process that serves `POST /api/jobs/:id/export`.

## What Is Signed

When `security.evidence_signing.enabled=true`, the exported ZIP includes a signed `proof.json`.

The signature covers:

- `manifest.json`
- `events.ndjson`
- `ledger.ndjson`

Offline verification still checks the manifest file hashes, event hash chain, and ledger consistency. If a public key is supplied, it also verifies the signature.

## Configuration

```yaml
security:
  evidence_signing:
    enabled: true
    key_id: "release-2026-q2"
    private_key_base64: "${AETHERIS_EVIDENCE_SIGNING_PRIVATE_KEY}"
    public_key_base64: "${AETHERIS_EVIDENCE_SIGNING_PUBLIC_KEY}"
```

Key material is raw Ed25519 key bytes encoded with standard base64:

- private key: 64 bytes before base64 encoding
- public key: 32 bytes before base64 encoding

`public_key_base64` is optional at runtime, but when present startup validates that it matches the configured private key.

## Export

```bash
aetheris export <job_id> --output evidence.zip
```

If signing is enabled on the API server, the ZIP is signed automatically.

## Verify

```bash
aetheris verify evidence.zip --public-key <base64-public-key>
```

Verification fails if a public key is provided and the package is unsigned or the signature does not match.

## Status

Signed evidence export is `integrated`: the runtime export path, config, CLI verification, and regression tests exist. It is not `production-ready` until release gates cover key rotation, key custody, and failed-signing drills.
