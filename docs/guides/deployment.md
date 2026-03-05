# Deployment

This document summarizes the deployment options and prerequisites for Aetheris v2.3.0+.

## Prerequisites

- **Go**: 1.25.7+ (aligned with [go.mod](../go.mod) and CI).
- **Postgres** (for jobstore): If using `jobstore.type=postgres`, prepare the database and apply the schema. Schema: [internal/runtime/jobstore/schema.sql](../internal/runtime/jobstore/schema.sql); Compose can mount it for init.
- **Docker** (for containerized deployment): Docker 20.10+

## Quick Start (Docker Compose)

**Recommended for local development and testing.**

### Full Stack (API + 2 Workers + Postgres + Monitoring)

```bash
# Start complete stack with monitoring (Jaeger, Grafana)
make docker-run

# Or use the script directly
./scripts/local-2.0-stack.sh start
```

**Services**:
| Service | Port | Description |
|---------|------|-------------|
| API | 8080 | HTTP API server |
| Worker1 | - | Background job processor |
| Worker2 | - | Background job processor |
| PostgreSQL | 5432 | Job store and event persistence |
| Redis | 6379 | Cache and RAG |
| Jaeger | 16686 | Distributed tracing |
| Grafana | 3000 | Metrics dashboard |

### Basic Stack (API + Worker + Postgres)

```bash
docker compose -f deployments/compose/docker-compose.yml up -d --build
```

### Verify Deployment

```bash
# Health check
curl http://localhost:8080/api/health

# Create agent
curl -X POST http://localhost:8080/api/agents \
  -H "Content-Type: application/json" \
  -d '{"name":"test-agent"}'

# View workers
curl http://localhost:8080/api/system/workers
```

### Stop Services

```bash
./scripts/local-2.0-stack.sh stop
```

## Production Deployment

### Production Requirements

For production environments, ensure:

1. **PostgreSQL** - Required for durable job storage
2. **Authentication** - Enable JWT and configure secrets
3. **TLS/SSL** - Enable HTTPS for API endpoints
4. **CORS** - Configure specific allowed origins (not `*`)
5. **Monitoring** - Enable OpenTelemetry and metrics

### Production Config Example

```yaml
# configs/api.yaml
app:
  env: "production"

auth:
  jwt:
    secret: "${JWT_SECRET}"  # Use environment variable
    enabled: true
    jwt_key: "${JWT_KEY}"

jobstore:
  type: "postgres"
  postgres:
    dsn: "${POSTGRES_DSN}?sslmode=require"

cors:
  enabled: true
  allowed_origins:
    - "https://your-domain.com"

monitoring:
  prometheus:
    enable: true
    port: 9092
  tracing:
    enable: true
    export_endpoint: "localhost:4317"
```

### Scaling Workers

Scale workers horizontally for higher throughput:

```bash
# Scale workers
docker compose -f deployments/compose/docker-compose.yml up -d --scale worker=4
```

## Database Schema

### Initial Setup

```bash
# Run schema on startup (automatic with Compose)
# Or manually apply:
psql -h localhost -U aetheris -d aetheris -f internal/runtime/jobstore/schema.sql
```

### Schema Updates

If upgrading from an older version:

```sql
-- Add missing columns
ALTER TABLE jobs ADD COLUMN IF NOT EXISTS cancel_requested_at TIMESTAMPTZ;
```

## Kubernetes

For production Kubernetes deployment, see [deployments/k8s/README.md](../deployments/k8s/README.md).

## Multi-Environment Deployment

Use the same runtime contract across `dev`, `staging`, and `prod`, with different scale and safety gates.

| Environment | Suggested topology | Main purpose |
|-------------|--------------------|--------------|
| `dev` | Compose (single node) | Feature development, local debugging |
| `staging` | Compose or K8s with Postgres | Integration validation, release rehearsal |
| `prod` | K8s + managed Postgres + monitoring | Production traffic and SLOs |

### Recommended promotion flow

1. `dev`: run `./scripts/release-2.0.sh` and local stack smoke checks.
2. `staging`: deploy candidate image/tag, run end-to-end scenarios (agent run, replay, export/verify).
3. `prod`: rollout with canary/rolling strategy and monitor error rate, stuck jobs, and queue backlog.

### Operational gates before promotion

- CI green (`.github/workflows/ci.yml`)
- Postgres integration tests green
- Runtime forensics checks pass (`export` + `verify`, consistency API)
- Rollback plan verified (previous image/tag ready)
- Security baseline checklist completed

---

For config (api.yaml, worker.yaml, model.yaml) and env vars see [config.md](../reference/config.md); for API and CLI usage see [usage.md](usage.md), [cli.md](cli.md), and [troubleshooting.md](troubleshooting.md).
