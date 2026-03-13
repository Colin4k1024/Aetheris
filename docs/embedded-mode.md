# Embedded Mode Quickstart

`embedded` mode runs Aetheris without external PostgreSQL.

## Local Run

- API:
  - `API_CONFIG_PATH=configs/api.embedded.yaml MODEL_CONFIG_PATH=configs/model.yaml go run ./cmd/api`
- Worker:
  - `WORKER_CONFIG_PATH=configs/worker.embedded.yaml MODEL_CONFIG_PATH=configs/model.yaml go run ./cmd/worker`

Persistent data is stored in `./data/embedded` (configured via `jobstore.dsn`).

## Docker Compose

- `docker compose -f deployments/compose/docker-compose.embedded.yml up -d`

This starts:

- `api` on `:8080`
- `worker` consuming jobs
- shared embedded data volume `embedded_data`

## Notes

- `jobstore.type=embedded` enables local durable event/state stores.
- `effect_store.type=embedded` and `checkpoint_store.type=embedded` keep replay and resume data local.
- This mode is intended for local-first and private-network deployments.
