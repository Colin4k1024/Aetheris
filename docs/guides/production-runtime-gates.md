# Production Runtime Gates

This page documents the startup gates required before Aetheris can make production-grade durable execution claims.

The gates are enforced when either of these is set:

```yaml
runtime:
  profile: "prod"
  strict: true
```

`profile: "prod"` or `strict: true` is enough to enable the checks.

## Enforced Gates

| Gate | API process | Worker process | Why |
|---|---:|---:|---|
| `jobstore.type=postgres` with DSN | Yes | Yes | Shared event history, leases, and job metadata |
| `effect_store.type=postgres` with DSN | Yes | Yes | Shared Effect Store for strong Replay catch-up |
| `checkpoint_store.type=postgres` with DSN | Yes | Yes | Durable resume cursor/state |
| Default database password rejected | Yes | Yes | Prevent accidental release with demo credentials |
| `sslmode=disable` rejected | Yes | Yes | Require encrypted DB transport in production |
| Specific CORS origins | Yes | N/A | API must not expose wildcard origins |
| API authentication enabled | Yes | N/A | API must not run unauthenticated in production |
| JWT key configured | Yes | N/A | Auth must not use an empty signing secret |

Implementation references:

- API gate: `internal/app/api/app.go` `validateProductionRuntimeConfig`
- Worker gate: `internal/app/worker/app.go` `validateProductionRuntimeConfig`
- Tests: `internal/app/api/app_utils_test.go`, `internal/app/worker/app_utils_test.go`

## Ledger Note

There is no separate public `invocation_ledger` config block today. The Invocation Ledger is built from the ToolInvocationStore when the DAG compiler is assembled.

In Postgres production mode, the ToolInvocationStore uses the same Postgres-backed runtime storage path as the Effect Store wiring. This means production claims require the Postgres Effect Store configuration and the shared Job/Event Store configuration together.

## Example Production Shape

```yaml
runtime:
  profile: "prod"
  strict: true

jobstore:
  type: "postgres"
  dsn: "${JOBSTORE_DSN}" # must not use default password; must not use sslmode=disable

effect_store:
  type: "postgres"
  dsn: "${EFFECT_STORE_DSN}"

checkpoint_store:
  type: "postgres"
  dsn: "${CHECKPOINT_STORE_DSN}"

api:
  cors:
    allow_origins:
      - "https://app.example.com"
  middleware:
    auth: true
    jwt_key: "${JWT_SECRET}"
```

## What This Gate Does Not Prove

The startup gate proves the process is not running in a known-unsafe production configuration. It does not prove:

- the database backup/restore plan works
- downstream services honor idempotency keys
- an `external_http` black-box agent has safe internal side effects
- the deployment has enough capacity for the configured workload

Use the [runtime cost model](runtime-cost-model.md), [guarantee matrix](guarantee-matrix.md), and release drills for those checks.
