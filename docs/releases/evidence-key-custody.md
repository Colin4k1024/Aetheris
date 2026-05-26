# Evidence Signing Key Custody

This runbook defines the minimum custody rules for `security.evidence_signing`.

## Scope

The current production-ready path signs evidence ZIP exports with an Ed25519 private key injected into the API process through configuration or environment variables.

This covers:

- `POST /api/jobs/:id/export`
- `proof.json` signature inside the exported ZIP
- Offline verification through `aetheris verify evidence.zip --public-key <base64-public-key>`

It does not claim HSM/KMS custody. KMS/Vault-backed signing is a future hardening option.

## Key Ownership

| Asset | Owner | Storage | Distribution |
|---|---|---|---|
| Ed25519 private key | Release/security owner | Secret manager or CI secret store | API process only |
| Ed25519 public key | Release/security owner | Release artifact, docs, or public key registry | Auditors, operators, and customers who verify evidence |
| Key ID | Release owner | Config and release notes | Same as public key |

## Injection Rules

- Store the private key as raw 64-byte Ed25519 private key bytes encoded with standard base64.
- Inject it through `AETHERIS_EVIDENCE_SIGNING_PRIVATE_KEY`.
- Inject the matching public key through `AETHERIS_EVIDENCE_SIGNING_PUBLIC_KEY` when startup validation should verify the pair.
- Never store private key material in Git, evidence ZIPs, logs, release notes, or support tickets.

Example:

```yaml
security:
  evidence_signing:
    enabled: true
    key_id: "release-2026-q2"
    private_key_base64: "${AETHERIS_EVIDENCE_SIGNING_PRIVATE_KEY}"
    public_key_base64: "${AETHERIS_EVIDENCE_SIGNING_PUBLIC_KEY}"
```

## Startup Expectations

Startup must fail when:

- signing is enabled but the private key is empty
- the private key is not valid base64
- the decoded private key is not 64 bytes
- the configured public key is invalid
- the configured public key does not match the private key

## Evidence Verification

Auditors should verify with:

```bash
aetheris verify evidence.zip --public-key <base64-public-key>
```

Verification must fail if the package is unsigned, tampered, or signed with a different private key.

## Incident Response

If the private key is exposed:

1. Disable the compromised key immediately.
2. Generate a new Ed25519 key pair.
3. Update `security.evidence_signing.key_id` and injected secrets.
4. Publish the retired public key status with the affected time window.
5. Re-run `scripts/release-evidence-signing-drill.sh`.
6. Keep the old public key available for historical verification unless it is explicitly revoked.
