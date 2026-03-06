# Experimental Packages

This directory contains packages that were implemented as part of 3.0 roadmap planning but are not currently integrated into the main execution path. They represent future capabilities that may be needed in later releases.

## Moved Packages

| Package        | Original Purpose                       | Status                                                   |
| -------------- | -------------------------------------- | -------------------------------------------------------- |
| `region`       | Regional-aware scheduling, GeoDNS      | Not integrated; depends on internal (architecture issue) |
| `replication`  | Data replication/sync                  | Not integrated                                           |
| `sla`          | Deadline, quotas, reporting            | Not integrated                                           |
| `audit`        | Audit signing                          | Not integrated (auth used instead)                       |
| `distributed`  | Distributed ledger protocol            | 3.0 planning                                             |
| `signature`    | Ed25519 keystore                       | 3.0 planning                                             |
| `integration`  | LDAP, storage, queue adapters          | Not integrated                                           |
| `secrets`      | Vault, AWS, K8s, env secret management | Not integrated                                           |
| `retention`    | Data retention policy engine           | 3.0 planning                                             |
| `monitoring`   | Quality scoring                        | 3.0 planning                                             |
| `ai_forensics` | AI anomaly detection                   | 3.0 planning                                             |
| `compliance`   | GDPR/SOX/HIPAA templates               | Not integrated                                           |
| `redaction`    | Data redaction engine                  | Not integrated                                           |
| `errors`       | Common error utilities                 | Not adopted; no imports in codebase                      |

## Restoration

If any of these packages become needed:

1. Move them back from `pkg/experimental/` to `pkg/`
2. Update imports in the relevant code
3. Ensure architecture issues (like `region` depending on `internal`) are resolved

## Alternative: Complete Removal

If a package is confirmed not needed for the 3.0 roadmap, it can be permanently deleted:

```bash
rm -rf pkg/experimental/<package>
```
