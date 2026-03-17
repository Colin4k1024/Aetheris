# Enterprise RBAC 架构设计

本文档描述 Aetheris Enterprise Edition 中基于角色的访问控制 (RBAC) 模块的架构设计。

## 概述

RBAC 模块提供：
- **角色 (Role)**：预定义和自定义角色
- **权限 (Permission)**：细粒度操作权限
- **策略 (Policy)**：角色-权限绑定
- **资源控制**：基于 Namespace 的资源隔离

## 核心概念

### Permission (权限)

权限是对系统资源的操作许可：

```go
type Permission struct {
    ID          string    `json:"id"`
    Name        string    `json:"name"`         // e.g., "job:read", "job:create"
    Resource    string    `json:"resource"`     // e.g., "job", "agent", "user"
    Action      string    `json:"action"`       // e.g., "read", "write", "delete", "execute"
    Description string    `json:"description"`
}
```

### Role (角色)

角色是权限的集合：

```go
type Role struct {
    ID          string    `json:"id"`
    Name        string    `json:"name"`         // e.g., "admin", "developer", "viewer"
    OrgID       string    `json:"org_id"`
    Type        RoleType  `json:"type"`         // "system" | "custom"
    Permissions []string  `json:"permissions"`  // Permission IDs
    IsDefault   bool      `json:"is_default"`   // 新用户默认角色
    Description string    `json:"description"`
    CreatedAt   time.Time `json:"created_at"`
    UpdatedAt   time.Time `json:"updated_at"`
}

type RoleType string

const (
    RoleTypeSystem RoleType = "system"  // 系统内置，不可删除
    RoleTypeCustom RoleType = "custom"  // 用户自定义
)
```

### Policy (策略)

策略将角色绑定到用户或组：

```go
type Policy struct {
    ID        string    `json:"id"`
    OrgID     string    `json:"org_id"`
    Subject   Subject   `json:"subject"`     // 谁
    RoleIDs   []string  `json:"role_ids"`    // 赋予什么角色
    Resource  Resource  `json:"resource"`     // 作用于什么资源
    Condition *Condition `json:"condition,omitempty"` // 条件
    CreatedBy string    `json:"created_by"`
    CreatedAt time.Time `json:"created_at"`
}

type Subject struct {
    Type string `json:"type"` // "user" | "department" | "service_account"
    ID   string `json:"id"`
}

type Resource struct {
    Type string   `json:"type"` // "namespace" | "job" | "agent"
    IDs  []string `json:"ids"`  // 资源 ID 列表，空=全部
}

type Condition struct {
    TimeRange    *TimeRange `json:"time_range,omitempty"`
    IPRange      []string   `json:"ip_range,omitempty"`
    MFARequired  bool       `json:"mfa_required,omitempty"`
}
```

## 内置角色

| 角色 | 权限 | 描述 |
|------|------|------|
| **Owner** | 全部权限 | 组织所有者，可管理组织设置、计费 |
| **Admin** | 管理全部资源 | 管理员，可管理用户、角色、策略 |
| **Developer** | job:*, agent:* | 开发人员，可创建和管理 Job/Agent |
| **Operator** | job:read, job:execute, job:cancel | 运维人员，可执行 Job |
| **Viewer** | job:read, agent:read | 只读访问 |

## 权限矩阵

| 资源 | 操作 | Owner | Admin | Developer | Operator | Viewer |
|------|------|-------|-------|-----------|-----------|--------|
| **User** | read | ✅ | ✅ | ❌ | ❌ | ❌ |
| | create | ✅ | ✅ | ❌ | ❌ | ❌ |
| | update | ✅ | ✅ | ❌ | ❌ | ❌ |
| | delete | ✅ | ✅ | ❌ | ❌ | ❌ |
| **Role** | read | ✅ | ✅ | ❌ | ❌ | ❌ |
| | assign | ✅ | ✅ | ❌ | ❌ | ❌ |
| **Job** | read | ✅ | ✅ | ✅ | ✅ | ✅ |
| | create | ✅ | ✅ | ✅ | ❌ | ❌ |
| | execute | ✅ | ✅ | ✅ | ✅ | ❌ |
| | cancel | ✅ | ✅ | ✅ | ✅ | ❌ |
| | delete | ✅ | ✅ | ✅ | ❌ | ❌ |
| **Agent** | read | ✅ | ✅ | ✅ | ✅ | ✅ |
| | create | ✅ | ✅ | ✅ | ❌ | ❌ |
| | update | ✅ | ✅ | ✅ | ❌ | ❌ |
| | delete | ✅ | ✅ | ✅ | ❌ | ❌ |
| **Namespace** | read | ✅ | ✅ | ✅ | ✅ | ✅ |
| | create | ✅ | ✅ | ❌ | ❌ | ❌ |
| | manage | ✅ | ✅ | ❌ | ❌ | ❌ |

## 模块架构

```
┌─────────────────────────────────────────────────────────────────┐
│                        API Layer                                 │
│  POST /api/v1/rbac/roles    GET /api/v1/rbac/roles/:id         │
│  POST /api/v1/rbac/policies GET /api/v1/rbac/policies           │
└────────────────────────────┬────────────────────────────────────┘
                             │
┌────────────────────────────▼────────────────────────────────────┐
│                     RBAC Service                                │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐            │
│  │ RoleManager │  │PolicyManager │  │PermChecker  │            │
│  └─────────────┘  └─────────────┘  └─────────────┘            │
└────────────────────────────┬────────────────────────────────────┘
                             │
┌────────────────────────────▼────────────────────────────────────┐
│                     Storage Layer                               │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐            │
│  │  RoleStore  │  │ PolicyStore │  │AuditStore   │            │
│  └─────────────┘  └─────────────┘  └─────────────┘            │
└─────────────────────────────────────────────────────────────────┘
```

### 核心组件

#### 1. RoleManager

```go
type RoleManager interface {
    // 系统角色操作
    GetSystemRoles(ctx context.Context) ([]Role, error)
    GetRole(ctx context.Context, id string) (*Role, error)
    ListRoles(ctx context.Context, orgID string) ([]Role, error)
    
    // 自定义角色操作
    CreateRole(ctx context.Context, role *Role) error
    UpdateRole(ctx context.Context, role *Role) error
    DeleteRole(ctx context.Context, id string) error
}
```

#### 2. PolicyManager

```go
type PolicyManager interface {
    CreatePolicy(ctx context.Context, policy *Policy) error
    UpdatePolicy(ctx context.Context, policy *Policy) error
    DeletePolicy(ctx context.Context, id string) error
    ListPolicies(ctx context.Context, orgID string, filter PolicyFilter) ([]Policy, error)
    
    // 批量操作
    AssignRoles(ctx context.Context, subject Subject, roleIDs []string) error
    RevokeRoles(ctx context.Context, subject Subject, roleIDs []string) error
}
```

#### 3. PermChecker (权限检查器)

```go
type PermChecker interface {
    // 检查用户是否具有某权限
    Check(ctx context.Context, userID, permission string) (bool, error)
    
    // 检查用户是否具有某资源的访问权限
    CheckResource(ctx context.Context, userID, resourceType, resourceID, action string) (bool, error)
    
    // 获取用户所有权限
    GetUserPermissions(ctx context.Context, userID string) ([]string, error)
    
    // 获取用户在特定资源上的权限
    GetResourcePermissions(ctx context.Context, userID, resourceType, resourceID string) ([]string, error)
}
```

## 数据库 Schema

```sql
-- 权限定义表 (系统内置)
CREATE TABLE permissions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(100) NOT NULL UNIQUE,
    resource VARCHAR(100) NOT NULL,
    action VARCHAR(50) NOT NULL,
    description TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- 角色表
CREATE TABLE roles (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id UUID NOT NULL REFERENCES organizations(id),
    name VARCHAR(100) NOT NULL,
    type VARCHAR(50) NOT NULL DEFAULT 'custom',
    permissions JSONB NOT NULL DEFAULT '[]',
    is_default BOOLEAN NOT NULL DEFAULT FALSE,
    description TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(org_id, name)
);

-- 策略表
CREATE TABLE policies (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id UUID NOT NULL REFERENCES organizations(id),
    subject_type VARCHAR(50) NOT NULL,
    subject_id UUID NOT NULL,
    role_ids JSONB NOT NULL DEFAULT '[]',
    resource_type VARCHAR(100),
    resource_ids JSONB DEFAULT '[]',
    condition JSONB,
    created_by UUID NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- 角色-权限关联视图 (物化)
CREATE MATERIALIZED VIEW role_permissions_view AS
SELECT 
    r.id AS role_id,
    r.org_id,
    p.id AS permission_id,
    p.name AS permission_name,
    p.resource,
    p.action
FROM roles r,
     jsonb_array_elements_text(r.permissions) perm_id
JOIN permissions p ON p.id = perm_id;

CREATE UNIQUE INDEX idx_role_permissions ON role_permissions_view(role_id, permission_id);
CREATE INDEX idx_role_permissions_org ON role_permissions_view(org_id);

-- 用户-角色关联表
CREATE TABLE user_roles (
    user_id UUID NOT NULL REFERENCES users(id),
    role_id UUID NOT NULL REFERENCES roles(id),
    granted_by UUID NOT NULL,
    granted_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (user_id, role_id)
);

-- 索引
CREATE INDEX idx_policies_org ON policies(org_id);
CREATE INDEX idx_policies_subject ON policies(subject_type, subject_id);
CREATE INDEX idx_roles_org ON roles(org_id);
CREATE INDEX idx_user_roles_user ON user_roles(user_id);
CREATE INDEX idx_user_roles_role ON user_roles(role_id);
```

## 权限检查流程

```
┌──────────┐     ┌──────────────┐     ┌─────────────────┐
│  Request │────▶│   API GW     │────▶│  Middleware     │
└──────────┘     └──────────────┘     └────────┬────────┘
                                                 │
                                                 ▼
                                        ┌────────────────┐
                                        │ PermChecker    │
                                        │                │
                                        │ 1. Get User    │
                                        │ 2. Get Roles   │
                                        │ 3. Get Perms   │
                                        │ 4. Check       │
                                        └────────┬───────┘
                                                 │
                              ┌──────────────────┼──────────────────┐
                              ▼                  ▼                  ▼
                         ┌─────────┐        ┌─────────┐       ┌─────────┐
                         │ Allow   │        │ Deny    │       │ Deny    │
                         │ 200 OK  │        │ 403     │       │ 401     │
                         └─────────┘        └─────────┘       └─────────┘
```

### Middleware 实现示例

```go
func RBACMiddleware(permChecker PermChecker) middleware.Middleware {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            // 从 context 获取用户信息
            userID := r.Context().Value("user_id")
            if userID == nil {
                http.Error(w, "Unauthorized", http.StatusUnauthorized)
                return
            }
            
            // 从路由获取所需权限
            requiredPerm := getRequiredPermission(r)
            if requiredPerm == "" {
                next.ServeHTTP(w, r)
                return
            }
            
            // 检查权限
            allowed, err := permChecker.Check(r.Context(), userID.(string), requiredPerm)
            if err != nil {
                http.Error(w, "Internal Error", http.StatusInternalServerError)
                return
            }
            
            if !allowed {
                http.Error(w, "Forbidden", http.StatusForbidden)
                return
            }
            
            next.ServeHTTP(w, r)
        })
    }
}
```

## API 端点

| 方法 | 路径 | 描述 |
|------|------|------|
| GET | /api/v1/rbac/permissions | 列出所有权限 |
| GET | /api/v1/rbac/roles | 列出角色 |
| POST | /api/v1/rbac/roles | 创建角色 |
| GET | /api/v1/rbac/roles/:id | 获取角色详情 |
| PUT | /api/v1/rbac/roles/:id | 更新角色 |
| DELETE | /api/v1/rbac/roles/:id | 删除角色 |
| GET | /api/v1/rbac/policies | 列出策略 |
| POST | /api/v1/rbac/policies | 创建策略 |
| PUT | /api/v1/rbac/policies/:id | 更新策略 |
| DELETE | /api/v1/rbac/policies/:id | 删除策略 |
| POST | /api/v1/rbac/users/:id/roles | 分配角色 |
| DELETE | /api/v1/rbac/users/:id/roles/:role_id | 撤销角色 |

## 审计与合规

所有 RBAC 操作都会被记录到审计日志：

- 角色创建/修改/删除
- 策略创建/修改/删除
- 角色分配/撤销
- 权限检查失败

## 相关文档

- [IAM 架构设计](iam.md)
- [审计日志架构](audit.md)
- [多租户架构](multi-tenant.md)
