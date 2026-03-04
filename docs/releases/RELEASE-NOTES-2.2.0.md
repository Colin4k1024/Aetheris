# Aetheris v2.2.0 Release Notes

**Release Date:** 2026-03-04

---

## Highlights

Aetheris v2.2.0 brings major enhancements in **observability**, **multi-adapter support**, and **enterprise features**. This release makes it easier than ever to build production-ready AI agents.

---

## What's New

### Observability Enhancements

- **Jaeger Integration** - Distributed tracing with Jaeger
- **OpenTelemetry** - Enabled by default for all HTTP requests
- **Grafana Dashboard** - New panels for Plan/Compile duration, Node execution, Run control

### Multi-Adapter Support

- **LlamaIndex Adapter** - `NodeLlamaIndex` for LlamaIndex workflows
- **Vertex AI Agent Engine** - `NodeVertex` for Google Vertex AI
- **AWS Bedrock Agents** - `NodeBedrock` for AWS Bedrock

### Enterprise Features

- **RBAC** - Postgres RoleStore with API endpoints
- **Region Scheduling** - Region-aware scheduler for global deployments
- **SLA/Quota** - Quota manager and SLO monitor

---

## Installation

### Binary

Download from [GitHub Releases](https://github.com/Colin4k1024/Aetheris/releases)

### From Source

```bash
git clone https://github.com/Colin4k1024/Aetheris
cd Aetheris
make build
```

### Docker

```bash
docker compose -f deployments/compose/docker-compose.yml up -d
```

---

## Upgrading

### From v2.1.x

1. Pull latest code: `git pull`
2. Rebuild: `make build`
3. Restart services

### Configuration Changes

No breaking changes in v2.2.0. New optional configs:

```yaml
# Optional: Jaeger tracing
otel:
  endpoint: "jaeger:4317"
  service_name: "aetheris-api"

# Optional: Region scheduling
scheduler:
  region: "us-east-1"
  allowed_regions: ["us-east-1", "us-west-2"]
```

---

## Full Changelog

### Added
- Jaeger integration in docker-compose
- OpenTelemetry tracing enabled by default
- Grafana dashboard panels
- LlamaIndex adapter (NodeLlamaIndex)
- Vertex AI Agent Engine adapter (NodeVertex)
- AWS Bedrock Agents adapter (NodeBedrock)
- Postgres RoleStore for RBAC
- RBAC API endpoints
- Region configuration
- Region-aware scheduler
- SLA Quota manager
- SLO monitor

### Changed
- Docker-compose default configuration

### Documentation
- 3 new adapter examples

---

## Breaking Changes

**None** - v2.2.0 is fully backward compatible.

---

## Known Issues

- None reported

---

## Contributors

Thank you to all contributors who made this release possible!

---

## What's Next

- v2.3.0 - Performance optimizations
- v3.0.0 - Enterprise features GA

---

## Resources

- **Documentation**: https://docs.aetheris.ai
- **GitHub**: https://github.com/Colin4k1024/Aetheris
- **Discord**: https://discord.gg/PrrK2Mua
- **Discussions**: https://github.com/Colin4k1024/Aetheris/discussions

---

*Thank you for using Aetheris!*
