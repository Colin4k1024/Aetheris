# Security Best Practices for Aetheris

This document outlines security best practices for deploying and running Aetheris in production environments.

## Overview

Aetheris handles AI agent execution with access to external tools, APIs, and data. Security is critical for production deployments.

---

## 1. Sandboxing

### Tool Isolation

Tools executed by Aetheris should be isolated from sensitive system resources.

#### Recommendations

1. **Use resource limits** — Configure memory, CPU, and execution time limits for tool processes
2. **Network isolation** — Run tools in containers with restricted network access
3. **File system restrictions** — Limit file system access to specific directories

#### Implementation Example

```go
// Configure tool execution with limits
toolCfg := &ToolConfig{
    MaxMemoryMB:    512,
    MaxCPUPercent:  50,
    TimeoutSeconds: 30,
    AllowedPaths:   []string{"/data/agent-workspace/"},
    NetworkPolicy:  "restricted",
}
```

### Process Sandboxing

- Use gVisor or Firecracker for enhanced isolation
- Consider running untrusted tools in ephemeral containers
- Implement tool timeout enforcement

---

## 2. Permission Management

### Principle of Least Privilege

Every component should have only the permissions it needs to function.

#### Role-Based Access Control (RBAC)

Aetheris supports fine-grained RBAC:

| Role | Permissions |
|------|-------------|
| Viewer | Read-only access to jobs and traces |
| Operator | Create jobs, view traces, manage own agents |
| Admin | Full access, including configuration and users |
| Security | View audit logs, manage security policies |

#### Tool Permissions

Configure per-tool permissions:

```go
// Tool permission configuration
permissions := &ToolPermissions{
    ToolName:       "http.request",
    AllowedHosts:   []string{"api.example.com", "*.trusted.cdn.com"},
    BlockedHosts:   []string{"localhost", "169.254.169.254"},
    RateLimit:      100, // requests per minute
    RequireApproval: false,
}
```

### API Security

1. **Authentication** — Use OAuth 2.0 or API keys
2. **TLS** — Enforce TLS 1.3 for all connections
3. **Input validation** — Validate all API inputs
4. **Rate limiting** — Prevent abuse

---

## 3. Logging & Forensics

### Audit Logging

All security-relevant events must be logged:

```go
// Security audit event types
const (
    EventJobCreated      = "job:created"
    EventJobApproved     = "job:approved"
    EventToolExecuted    = "tool:executed"
    EventToolBlocked     = "tool:blocked"
    EventAuthFailed      = "auth:failed"
    EventConfigChanged   = "config:changed"
)
```

#### Log Fields

Each audit log entry should include:

| Field | Description |
|-------|-------------|
| `timestamp` | ISO 8601 timestamp |
| `event_type` | Event category |
| `actor` | User or agent ID |
| `resource` | Affected resource |
| `action` | Action performed |
| `result` | Success/failure |
| `metadata` | Additional context |

### Log Retention

- **Security logs:** Minimum 1 year (compliance requirements may vary)
- **Application logs:** 90 days recommended
- **Debug logs:** 7 days (disable in production)

### Forensic Capabilities

1. **Job Replay** — Reconstruct any historical job execution
2. **Trace Export** — Export full traces for analysis
3. **State Snapshots** — Access checkpoints for debugging

---

## 4. Network Security

### Network Policies

```yaml
# Example Kubernetes NetworkPolicy
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: aetheris-agent-network
spec:
  podSelector:
    matchLabels:
      app: aetheris-worker
  policyTypes:
    - Ingress
    - Egress
  egress:
    - to:
        - podSelector:
            matchLabels:
              app: aetheris-api
      ports:
        - protocol: TCP
          port: 8080
    - to:
        - namespaceSelector: {}
      ports:
        - protocol: TCP
          port: 5432
```

### Secrets Management

1. **Never commit secrets** — Use environment variables or secret stores
2. **Rotate secrets** — Implement automatic rotation
3. **Audit access** — Log all secret access attempts

---

## 5. Input Validation & Sanitization

### Gatekeeper

Aetheris includes a Gatekeeper component for input validation:

```go
// Configure gatekeeper
gk := gatekeeper.New(
    gatekeeper.WithAllowedHosts([]string{"api.example.com"}),
    gatekeeper.WithBlockedPatterns([]string{"*.*.evil.com", "localhost"}),
    gatekeeper.WithNetworkValidation(true),
    gatekeeper.WithTypeValidation(true),
)
```

### Validation Rules

1. **URL validation** — Validate and sanitize all URLs
2. **Path traversal prevention** — Block `../` in file paths
3. **SQL injection prevention** — Use parameterized queries
4. **Command injection** — Never pass unsanitized input to shell commands

---

## 6. Security Checklist

Before deploying to production:

- [ ] Enable authentication on all APIs
- [ ] Configure TLS with valid certificates
- [ ] Set up RBAC with least-privilege roles
- [ ] Configure tool whitelist/blacklist
- [ ] Enable audit logging
- [ ] Set up log aggregation and retention
- [ ] Implement network policies
- [ ] Configure resource limits for tools
- [ ] Test security configurations
- [ ] Review access controls regularly

---

## 7. Incident Response

### Security Events to Monitor

1. **Authentication failures** — Repeated failed logins
2. **Authorization violations** — Access denied events
3. **Tool blocks** — Blocked network requests or file access
4. **Rate limit exceeded** — Unusual API usage patterns

### Response Steps

1. **Identify** — Determine scope and nature of incident
2. **Contain** — Isolate affected components
3. **Eradicate** — Remove threat
4. **Recover** — Restore normal operation
5. **Document** — Record findings and lessons learned

---

## Related Documentation

- [Security Policy](../../SECURITY.md)
- [Gatekeeper Configuration](../../reference/gatekeeper.md)
- [API Authentication](../guides/authentication.md)
- [Audit Logging](../guides/audit-logging.md)

---

*Last updated: 2026-03-17*
