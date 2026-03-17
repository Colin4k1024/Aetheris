# Enterprise IAM 架构设计

本文档描述 Aetheris Enterprise Edition 中身份与访问管理 (IAM) 模块的架构设计。

## 概述

IAM 模块为企业提供：
- **组织管理**：多租户架构下的组织、部门结构
- **身份认证**：本地用户 + 外部身份提供商 (IdP)
- **会话管理**：JWT Token 颁发、刷新、吊销

## 核心概念

### Entity Relationship

```
Organization (组织)
    │
    ├── Department (部门)
    │       │
    │       └── User (用户)
    │
    └── User (用户)
            │
            ├── Identity (身份凭证)
            │       ├── Local (用户名/密码)
            │       └── External (OIDC/LDAP/SAML)
            │
            └── Session (会话)
                    │
                    └── Token (JWT)
```

## 模块设计

### 1. 组织管理 (Organization Service)

```go
type Organization struct {
    ID          string    `json:"id"`
    Name        string    `json:"name"`
    Domain      string    `json:"domain"`       // 企业域名
    ParentID    *string   `json:"parent_id"`   // 父组织 (多租户)
    Settings    Settings  `json:"settings"`
    CreatedAt   time.Time `json:"created_at"`
    UpdatedAt   time.Time `json:"updated_at"`
}

type Settings struct {
    DefaultRole string `json:"default_role"`
    MFAEnabled  bool   `json:"mfa_enabled"`
    IPWhitelist []string `json:"ip_whitelist"`
}
```

### 2. 用户管理 (User Service)

```go
type User struct {
    ID             string    `json:"id"`
    OrganizationID string    `json:"org_id"`
    DepartmentID   *string   `json:"dept_id"`
    Email          string    `json:"email"`
    DisplayName    string    `json:"display_name"`
    Status         UserStatus `json:"status"` // active, suspended, deleted
    CreatedAt      time.Time `json:"created_at"`
    UpdatedAt      time.Time `json:"updated_at"`
}

type UserStatus string

const (
    UserStatusActive    UserStatus = "active"
    UserStatusSuspended UserStatus = "suspended"
    UserStatusDeleted   UserStatus = "deleted"
)
```

### 3. 身份认证 (Identity Service)

```go
type Identity struct {
    ID        string       `json:"id"`
    UserID    string       `json:"user_id"`
    Type      IdentityType `json:"type"`
    Provider  string       `json:"provider,omitempty"`  // "local", "oidc", "ldap", "saml"
    Claims    Claims      `json:"claims"`
    VerifiedAt time.Time  `json:"verified_at"`
}

type IdentityType string

const (
    IdentityTypeLocal   IdentityType = "local"
    IdentityTypeOIDC    IdentityType = "oidc"
    IdentityTypeLDAP    IdentityType = "ldap"
    IdentityTypeSAML    IdentityType = "saml"
)
```

### 4. 会话管理 (Session Service)

```go
type Session struct {
    ID            string     `json:"id"`
    UserID        string     `json:"user_id"`
    OrganizationID string    `json:"org_id"`
    TokenID       string     `json:"token_id"`      // JWT ID
    IPAddress     string     `json:"ip_address"`
    UserAgent     string     `json:"user_agent"`
    ExpiresAt     time.Time  `json:"expires_at"`
    CreatedAt     time.Time  `json:"created_at"`
    RevokedAt     *time.Time `json:"revoked_at,omitempty"`
}
```

## 认证流程

### 本地认证流程

```
┌──────────┐     ┌──────────────┐     ┌─────────────────┐     ┌─────────────┐
│  Client  │────▶│   API GW     │────▶│  Identity Svc   │────▶│  User Store │
└──────────┘     └──────────────┘     └─────────────────┘     └─────────────┘
                       │                       │                       │
                       │  1. POST /auth/login  │                       │
                       │─────────────────────▶│                       │
                       │                       │  2. Verify credential│
                       │                       │──────────────────────▶│
                       │                       │                       │
                       │  3. Return JWT        │                       │
                       │◀──────────────────────│                       │
                       │─────────────────────▶│                       │
                       │  4. Store session     │                       │
                       │──────────────────────▶│                       │
```

### OIDC 联邦认证流程

```
┌──────────┐     ┌──────────────┐     ┌─────────────────┐     ┌─────────────┐
│  Client  │────▶│   API GW     │────▶│  Identity Svc   │────▶│  OIDC IdP   │
└──────────┘     └──────────────┘     └─────────────────┘     └─────────────┘
                       │                       │                       │
                       │  1. GET /auth/oidc/authorize                      │
                       │─────────────────────▶│                       │
                       │  2. Redirect to IdP  │                       │
                       │◀──────────────────────│                       │
                       │                       │                       │
                       │  3. IdP Login        │─────────────────────▶│
                       │                       │                       │
                       │  4. Callback with code                        │
                       │─────────────────────▶│                       │
                       │                       │  5. Exchange token    │
                       │                       │─────────────────────▶│
                       │                       │                       │
                       │  6. Create/Update User & Return JWT            │
                       │◀──────────────────────│                       │
```

## Token 设计

### Access Token (JWT)

```go
type AccessTokenClaims struct {
    // Standard claims
    jwt.RegisteredClaims
    
    // Custom claims
    UserID         string   `json:"sub"`
    OrganizationID string   `json:"org_id"`
    DepartmentID   string   `json:"dept_id"`
    Roles          []string `json:"roles"`
    Permissions    []string `json:"permissions"`
}
```

### Refresh Token

- 存储在数据库中，与 Session 关联
- 支持轮换 (Rotation)：每次刷新时生成新的 Refresh Token
- 支持批量吊销 (按用户、按组织)

## 数据库 Schema

### PostgreSQL Tables

```sql
-- 组织表
CREATE TABLE organizations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    domain VARCHAR(255) UNIQUE,
    parent_id UUID REFERENCES organizations(id),
    settings JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- 用户表
CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id UUID NOT NULL REFERENCES organizations(id),
    dept_id UUID REFERENCES departments(id),
    email VARCHAR(255) NOT NULL,
    display_name VARCHAR(255),
    status VARCHAR(50) NOT NULL DEFAULT 'active',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(org_id, email)
);

-- 身份凭证表
CREATE TABLE identities (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id),
    type VARCHAR(50) NOT NULL,
    provider VARCHAR(50),
    claims JSONB NOT NULL,
    verified_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- 会话表
CREATE TABLE sessions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id),
    org_id UUID NOT NULL REFERENCES organizations(id),
    token_id VARCHAR(255) NOT NULL,
    ip_address INET,
    user_agent TEXT,
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    revoked_at TIMESTAMPTZ
);

-- 索引
CREATE INDEX idx_sessions_user_id ON sessions(user_id);
CREATE INDEX idx_sessions_token_id ON sessions(token_id);
CREATE INDEX idx_sessions_org_id ON sessions(org_id);
CREATE INDEX idx_users_org_id ON users(org_id);
```

## API 端点

| 方法 | 路径 | 描述 |
|------|------|------|
| POST | /api/v1/auth/login | 本地登录 |
| POST | /api/v1/auth/logout | 登出 |
| POST | /api/v1/auth/refresh | 刷新 Token |
| GET | /api/v1/auth/oidc/authorize | OIDC 授权起始 |
| GET | /api/v1/auth/oidc/callback | OIDC 回调 |
| GET | /api/v1/users | 列出用户 |
| POST | /api/v1/users | 创建用户 |
| GET | /api/v1/users/:id | 获取用户详情 |
| PUT | /api/v1/users/:id | 更新用户 |
| DELETE | /api/v1/users/:id | 删除用户 |
| GET | /api/v1/orgs | 列出组织 |
| POST | /api/v1/orgs | 创建组织 |

## 安全考虑

1. **密码策略**：最小长度、复杂度要求、定期过期
2. **MFA 支持**：TOTP / WebAuthn
3. **登录保护**：失败次数限制、IP 黑名单
4. **Token 安全**：短期 Access Token (15min)、长期 Refresh Token
5. **审计日志**：所有认证事件记录审计日志

## 相关文档

- [RBAC 设计](rbac.md)
- [审计日志架构](audit.md)
- [SSO 集成方案](sso-integration.md)
