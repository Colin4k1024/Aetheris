# Aetheris Open Core Boundary

This document defines what ships in OSS core versus Enterprise add-ons.

## OSS Core

- Durable runtime execution loop (`internal/agent/runtime`)
- Event sourcing and replay (`internal/runtime/jobstore`, `internal/agent/replay`)
- Local-first embedded mode (`jobstore.type=embedded`)
- Core API/CLI/SDK (`internal/api/http`, `cmd/cli`, `pkg/agent/sdk`)
- Trace and proof export primitives (`pkg/proof`, `/api/jobs/:id/trace`, `/api/jobs/:id/verify`)
- MCP host bridge and tool adapter abstractions (`internal/agent/tools/mcp_host.go`)

## Agent Construction Boundary

- Default authoring path is **Eino-first** (agent construction outside Aetheris runtime core).
- Aetheris OSS core focuses on execution guarantees: durability, replay, scheduling, auditability, and MCP tool hosting.
- Legacy `/api/agents/*` and non-Eino adapter paths are compatibility facades for migration, not the primary product narrative.

## Enterprise Add-ons

- Persistent RBAC administration and policy packs (Postgres-backed role store)
- Compliance dashboards and report workflows (SOX/PCI/GDPR templates)
- Tamper-evident operational audit analytics and retention controls
- Enterprise SSO federation and org-level governance controls
- Multi-cluster scheduling and advanced control-plane management

## Minimum Enterprise Security Baseline

- Authentication enabled in production
- Persistent RBAC role storage
- API access audit logging enabled
- Evidence export and verification available for regulated workflows
