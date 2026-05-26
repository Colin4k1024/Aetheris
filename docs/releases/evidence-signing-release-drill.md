# Evidence Signing Release Drill

This drill is the release gate for the signed evidence export slice.

## Command

Run from repository root:

```bash
./scripts/release-evidence-signing-drill.sh
```

The script writes a report under `artifacts/release/`.

## What It Covers

- config parsing for valid Ed25519 signing keys
- startup-level rejection for invalid private keys
- signed `proof.json` generation during evidence export
- offline CLI verification with a supplied public key
- unsigned/tampered package failure coverage through existing proof tests

## Pass Criteria

The drill passes only when all targeted suites pass:

- `./internal/app/api`
- `./internal/api/http`
- `./cmd/cli`
- `./pkg/proof`

The release checklist should include the generated report path.

## Manual Follow-Up

Before a production release, confirm:

- the private key comes from the approved secret store
- the public key and key ID are published to the auditor/customer verification channel
- key rotation and emergency revoke contacts are documented
- historical public keys remain available
