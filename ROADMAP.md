# Aetheris Roadmap

## Vision

Aetheris aims to be the standard runtime for production-grade AI agents.

## Core Principles

1. **Reliability First** - Agents must be recoverable and auditable
2. **Framework Agnostic** - Support all popular agent frameworks
3. **Enterprise Ready** - RBAC, compliance, observability built-in

## Version History

- **v2.3.0** (Current) - Performance & Scale
  - PostgreSQL connection pool optimization (PoolManager)
  - Redis caching layer for job metadata
  - gRPC API-Worker communication
  - Per-tenant rate limiting
  - Redis-based leader election
  - Local development mode (--dev flag)

- **v2.2.0** - Enhanced Observability & Multi-Adapter Release
  - OpenTelemetry integration with Jaeger
  - Grafana metrics dashboard
  - LlamaIndex, Vertex AI, Bedrock, AgentScope adapters
  - RBAC with Postgres RoleStore
  - Region-aware scheduling
  - SLA Quota manager

## Upcoming Features

### v3.0.0 - Enterprise Scale (Q3-Q4 2026)

#### Multi-Region Deployment
- [ ] Cross-region job replication
- [ ] Global load balancing with GeoDNS
- [ ] Region failover automation
- [ ] Async event write batching
- [ ] Index optimization for job queries

#### Scalability Features
- [ ] Horizontal worker scaling with Kubernetes HPA
- [ ] Leader election for scheduler (Redis-based)
- [ ] Distributed job locking with Redis
- [ ] Worker affinity and topology awareness
- [ ] Auto-scaling based on queue depth

#### Multi-Tenant Improvements
- [ ] Tenant-specific connection pools
- [ ] Per-tenant rate limiting
- [ ] Tenant isolation verification tests
- [ ] Cross-tenant query prevention

#### Developer Experience
- [ ] Enhanced CLI debugging tools
- [ ] Local development mode with hot reload
- [ ] Step-by-step execution trace viewer

---

### v3.0.0 - Enterprise Scale (Q3-Q4 2026)

#### Multi-Region Deployment
- [ ] Cross-region job replication
- [ ] Global负载均衡 with GeoDNS
- [ ] Region failover automation
- [ ] Latency-based routing

#### Enterprise Compliance
- [ ] **Audit Log Signature** - Cryptographic signing of audit logs
- [ ] Data residency controls (GDPR compliance)
- [ ] SOC 2 Type II readiness
- [ ] HIPAA compliance framework (optional)
- [ ] PII masking in logs/traces

#### Enhanced SLA Guarantees
- [ ] Job deadline enforcement
- [ ] Step-level SLA tracking
- [ ] Automated failover on SLA breach
- [ ] SLA reporting dashboard

#### Advanced Security
- [ ] mTLS for all internal communication
- [ ] Secrets management integration (Vault, AWS Secrets Manager)
- [ ] API request signing
- [ ] IP allowlist support

#### Enterprise Integrations
- [ ] SAML/OIDC SSO
- [ ] LDAP/Active Directory integration
- [ ] Enterprise message queues (RabbitMQ, Amazon SQS)
- [ ] Cloud storage backends (S3, GCS, Azure Blob)

---

### Future Considerations (Post-3.0)

#### AI-Native Features
- [ ] LLM-native task decomposition
- [ ] Self-healing agent workflows
- [ ] Predictive scaling based on workload patterns

#### Developer Platform
- [ ] Visual workflow builder (Web UI)
- [ ] One-click deployment to cloud
- [ ] Agent marketplace/community

---

## Community Requests

Vote on features: [Discussions](https://github.com/Colin4k1024/Aetheris/discussions)

## Contribution

See [CONTRIBUTING.md](CONTRIBUTING.md) to help build Aetheris.
