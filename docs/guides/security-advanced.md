# Advanced Security Features

> **Version**: v2.3.0+

Aetheris provides enterprise-grade security features including mTLS, API signing, and secrets management.

## mTLS (Mutual TLS)

### Overview

mTLS provides bidirectional authentication between services, ensuring both client and server verify each other's certificates.

### Configuration

```yaml
security:
  mtls:
    enabled: true
    cert_file: /path/to/server.crt
    key_file: /path/to/server.key
    ca_file: /path/to/ca.crt
    client_cert_file: /path/to/client.crt
    client_key_file: /path/to/client.key
    insecure_skip_verify: false  # Only for testing
```

### HTTP Server

```go
import "rag-platform/pkg/security/mtls"

server, err := mtls.NewHTTPServer(mtls.HTTPServerConfig{
    CertFile: "server.crt",
    KeyFile:  "server.key",
    CAFile:   "ca.crt",
})
```

### gRPC Server

```go
import "rag-platform/pkg/security/mtls"

server, err := mtls.NewGRPCServer(mtls.GRPCServerConfig{
    CertFile: "server.crt",
    KeyFile:  "server.key",
    CAFile:   "ca.crt",
})
```

## API Request Signing

### Overview

API signing validates that requests have not been tampered with in transit.

### Configuration

```yaml
security:
  api_signing:
    enabled: true
    algorithm: hmac-sha256
    clock_skew: 5m
    required_paths:
      - /api/admin/*
      - /api/jobs/*
```

### Middleware Usage

```go
import "rag-platform/internal/api/http/middleware"

signer, err := middleware.NewSignerMiddleware("your-secret-key", []string{
    "/api/admin/*",
    "/api/jobs/*",
})

// Add to Hertz
h.Use(signer.Middleware())
```

### Client Signing

```go
import "rag-platform/pkg/security/signer"

s := signer.NewSigner("your-secret-key", signer.WithAlgorithm("hmac-sha256"))

// Sign request
req, _ := http.NewRequest("POST", "/api/jobs", nil)
s.SignRequest(req, time.Now())
```

## Secrets Management

### Overview

Aetheris supports multiple secrets providers:

- **AWS Secrets Manager**
- **HashiCorp Vault** (coming soon)
- **Kubernetes Secrets** (coming soon)
- **Environment Variables**

### AWS Secrets Manager

```yaml
security:
  secrets:
    provider: aws
    config:
      region: us-east-1
      secret_prefix: aetheris/
```

### Usage

```go
import "rag-platform/pkg/secrets"

store, err := secrets.NewStore(secrets.Config{
    Provider: "aws",
    Config: map[string]string{
        "region": "us-east-1",
    },
})

// Get secret
value, err := store.Get(ctx, "my-secret-key")

// Set secret
err = store.Set(ctx, "my-secret-key", "secret-value")

// Delete secret
err = store.Delete(ctx, "my-secret-key")
```

## IP Allowlist

### Configuration

```yaml
security:
  ip_allowlist:
    enabled: true
    allow_ips:
      - 10.0.0.0/8
      - 192.168.1.0/24
    block_ips:
      - 10.0.0.100
    trusted_proxies:
      - 10.0.0.1
```

### Usage

```go
import "rag-platform/internal/api/http/middleware"

allowlist, err := middleware.NewIPAllowList(middleware.IPAllowListConfig{
    Enabled:        true,
    AllowIPs:       []string{"10.0.0.0/8"},
    TrustedProxies: []string{"10.0.0.1"},
})

h.Use(allowlist.Middleware())
```

## SSO/OIDC

### Configuration

```yaml
security:
  sso:
    issuer_url: https://accounts.google.com
    client_id: your-client-id
    client_secret: ${OIDC_CLIENT_SECRET}
    redirect_url: https://your-app.com/auth/callback
```

### Usage

```go
import "rag-platform/pkg/security/sso"

oidc, err := sso.NewOIDC(sso.OIDCConfig{
    IssuerURL:    "https://accounts.google.com",
    ClientID:     "your-client-id",
    ClientSecret: "your-client-secret",
})

// Get authorization URL
authURL := oidc.AuthCodeURL("state", "http://localhost:8080/callback")

// Exchange code for token
token, err := oidc.Exchange(ctx, code)

// Get user info
user, err := oidc.UserInfo(ctx, token)
```

## Security Best Practices

1. **Always use mTLS** in production
2. **Rotate secrets** regularly
3. **Enable API signing** for sensitive endpoints
4. **Use IP allowlists** to restrict access
5. **Enable audit logging** for compliance
