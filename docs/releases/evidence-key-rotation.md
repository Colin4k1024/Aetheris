# Evidence Signing Key Rotation

This runbook defines the supported key rotation process for signed evidence ZIPs.

## Rotation Model

Aetheris writes `signer_key_id` into signed `proof.json`. Verifiers use the matching public key out of band.

Rotation therefore uses an overlap period:

1. Publish the new public key before switching exporters.
2. Switch API config to the new private key and key ID.
3. Keep the old public key available for historical evidence verification.
4. Mark the old key as `retired` after the overlap period.

## Standard Rotation

1. Generate a new Ed25519 key pair out of band.
2. Store the new private key in the deployment secret store.
3. Publish the new public key and key ID to the release channel used by auditors.
4. Update API config:

```yaml
security:
  evidence_signing:
    enabled: true
    key_id: "release-2026-q3"
    private_key_base64: "${AETHERIS_EVIDENCE_SIGNING_PRIVATE_KEY}"
    public_key_base64: "${AETHERIS_EVIDENCE_SIGNING_PUBLIC_KEY}"
```

5. Restart or roll the API service.
6. Export a new evidence ZIP.
7. Verify with the new public key:

```bash
aetheris verify evidence.zip --public-key <new-base64-public-key>
```

8. Verify one historical package with the old public key.
9. Record the key IDs and verification results in the release notes.

## Emergency Rotation

Use emergency rotation when a private key is suspected to be exposed.

1. Stop exports or set `security.evidence_signing.enabled=false` until a new key is ready.
2. Generate and inject a new key pair.
3. Change `key_id`.
4. Publish a revocation notice for the compromised key and affected time range.
5. Re-run the evidence signing release drill.
6. Require manual approval before publishing new signed evidence packages.

## Public Key Distribution Policy

Every signed release or compliance export must identify:

- key ID
- public key
- activation time
- retirement time, when applicable
- revocation status, when applicable

Do not distribute private keys or seed material.
