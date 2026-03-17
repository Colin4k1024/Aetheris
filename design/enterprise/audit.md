# Enterprise 审计日志架构设计

本文档描述 Aetheris Enterprise Edition 中审计日志模块的架构设计。

## 概述

审计日志模块提供：
- **完整操作追踪**：所有用户和管理操作的不可变记录
- **合规报告**：SOX/PCI/GDPR 等合规框架支持
- **安全分析**：异常行为检测和告警
- **证据导出**：可验证的审计证据包

## 核心概念

### Audit Event (审计事件)

```go
type AuditEvent struct {
    ID            string    `json:"id"`
    OrgID         string    `json:"org_id"`
    Timestamp     time.Time `json:"timestamp"`
    
    // 事件主体 (谁)
    Actor         Actor     `json:"actor"`
    
    // 事件内容 (做了什么)
    Action        string    `json:"action"`        // e.g., "user.create", "job.execute"
    ResourceType  string    `json:"resource_type"` // e.g., "user", "job", "agent"
    ResourceID    string    `json:"resource_id"`
    
    // 事件详情
    Details       map[string]interface{} `json:"details"`
    Changes       []Change              `json:"changes,omitempty"` // 用于 update 操作
    
    // 上下文
    ClientIP      string    `json:"client_ip"`
    UserAgent     string    `json:"user_agent"`
    RequestID     string    `json:"request_id"`
    
    // 溯源
    TraceID       string    `json:"trace_id,omitempty"`
    SessionID     string    `json:"session_id,omitempty"`
}

type Actor struct {
    Type     string `json:"type"` // "user", "service_account", "system"
    ID       string `json:"id"`
    Name     string `json:"name"`
    Email    string `json:"email,omitempty"`
}

type Change struct {
    Field    string      `json:"field"`
    OldValue interface{} `json:"old_value"`
    NewValue interface{} `json:"new_value"`
}
```

### 事件分类

| 类别 | 前缀 | 示例 | 敏感度 |
|------|------|------|--------|
| **认证** | auth.* | auth.login, auth.logout, auth.mfa_enable | 中 |
| **授权** | rbac.* | rbac.role.assign, rbac.policy.create | 高 |
| **用户** | user.* | user.create, user.update, user.delete | 高 |
| **资源** | job.*, agent.* | job.create, agent.execute | 低-中 |
| **系统** | system.* | system.config.update, system.backup | 高 |
| **数据** | data.* | data.export, data.import | 高 |

## 模块架构

```
┌─────────────────────────────────────────────────────────────────┐
│                       Event Sources                              │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌──────────┐      │
│  │  API GW  │  │ Scheduler│  │  Worker  │  │  CRON    │      │
│  └────┬─────┘  └────┬─────┘  └────┬─────┘  └────┬─────┘      │
└───────┼─────────────┼─────────────┼─────────────┼──────────────┘
        │             │             │             │
        ▼             ▼             ▼             ▼
┌─────────────────────────────────────────────────────────────────┐
│                   Audit Collector (Buffer)                      │
│              ┌─────────────────────────────┐                   │
│              │    Kafka / Redis Stream      │                   │
│              │    (Async, High Throughput)  │                   │
│              └─────────────────────────────┘                   │
└────────────────────────────┬────────────────────────────────────┘
                             │
                             ▼
┌─────────────────────────────────────────────────────────────────┐
│                   Audit Processor                                │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐              │
│  │ Normalizer  │▶│  Enricher   │▶│  Persister  │              │
│  └─────────────┘  └─────────────┘  └─────────────┘              │
└────────────────────────────┬────────────────────────────────────┘
                             │
              ┌──────────────┼──────────────┐
              ▼              ▼              ▼
       ┌───────────┐  ┌───────────┐  ┌───────────┐
       │  Primary  │  │  Search   │  │  Cold     │
       │  Store    │  │  Index    │  │  Storage  │
       │ (Postgres)│  │ (OpenSearch)│ │ (S3/Blob) │
       └───────────┘  └───────────┘  └───────────┘
```

### 组件说明

#### 1. Audit Collector

```go
type AuditCollector interface {
    // 同步收集 (低延迟要求)
    Emit(ctx context.Context, event *AuditEvent) error
    
    // 批量收集 (高吞吐)
    EmitBatch(ctx context.Context, events []*AuditEvent) error
}
```

#### 2. Audit Processor

```go
type AuditProcessor interface {
    // 事件标准化
    Normalize(ctx context.Context, raw *RawAuditEvent) (*AuditEvent, error)
    
    // 事件富化 (添加上下文)
    Enrich(ctx context.Context, event *AuditEvent) (*AuditEvent, error)
    
    // 持久化存储
    Persist(ctx context.Context, event *AuditEvent) error
}
```

#### 3. Query Service

```go
type AuditQueryService interface {
    // 基础查询
    Query(ctx context.Context, filter AuditFilter) ([]AuditEvent, error)
    
    // 聚合分析
    Aggregate(ctx context.Context, agg AuditAggregation) ([]AggregateResult, error)
    
    // 导出
    Export(ctx context.Context, filter AuditFilter, format string) (io.ReadCloser, error)
}
```

## 事件收集

### API 层集成

```go
// Audit Middleware
func AuditMiddleware(collector AuditCollector) middleware.Middleware {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            // 记录请求开始
            start := time.Now()
            requestID := generateRequestID()
            
            // 包装 ResponseWriter 以捕获响应
            wrapped := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}
            
            next.ServeHTTP(wrapped, r)
            
            // 构建审计事件
            event := &AuditEvent{
                ID:           generateEventID(),
                OrgID:        getOrgID(r),
                Timestamp:    start,
                Actor:        getActor(r),
                Action:       getAction(r),
                ResourceType: getResourceType(r),
                ResourceID:   getResourceID(r),
                ClientIP:     getClientIP(r),
                UserAgent:    r.UserAgent(),
                RequestID:    requestID,
            }
            
            // 异步发送审计事件
            go func() {
                if err := collector.Emit(r.Context(), event); err != nil {
                    log.Error("audit emit failed", "error", err)
                }
            }()
        })
    }
}
```

### 业务层集成

```go
func (s *UserService) CreateUser(ctx context.Context, req *CreateUserRequest) (*User, error) {
    // 执行业务逻辑
    user, err := s.store.CreateUser(ctx, req)
    if err != nil {
        return nil, err
    }
    
    // 记录审计日志
    s.auditCollector.Emit(ctx, &AuditEvent{
        OrgID:        getOrgID(ctx),
        Timestamp:    time.Now(),
        Actor:        getActor(ctx),
        Action:       "user.create",
        ResourceType: "user",
        ResourceID:   user.ID,
        Details: map[string]interface{}{
            "email":      user.Email,
            "display_name": user.DisplayName,
        },
    })
    
    return user, nil
}
```

## 数据库 Schema

```sql
-- 审计事件主表
CREATE TABLE audit_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id UUID NOT NULL,
    timestamp TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    
    -- Actor
    actor_type VARCHAR(50) NOT NULL,
    actor_id VARCHAR(255) NOT NULL,
    actor_name VARCHAR(255),
    actor_email VARCHAR(255),
    
    -- Action
    action VARCHAR(100) NOT NULL,
    resource_type VARCHAR(100) NOT NULL,
    resource_id VARCHAR(255),
    
    -- Details (JSON)
    details JSONB,
    changes JSONB,
    
    -- Context
    client_ip INET,
    user_agent TEXT,
    request_id VARCHAR(255),
    trace_id VARCHAR(255),
    session_id VARCHAR(255),
    
    -- Partition key (按组织和时间分区)
    partition_key GENERATED ALWAYS AS (org_id || '_' || date_trunc('day', timestamp)) STORED
);

-- 主键索引
CREATE INDEX idx_audit_org_time ON audit_events(org_id, timestamp DESC);

-- 操作类型索引
CREATE INDEX idx_audit_action ON audit_events(org_id, action, timestamp DESC);

-- 资源索引
CREATE INDEX idx_audit_resource ON audit_events(org_id, resource_type, resource_id, timestamp DESC);

-- Actor 索引
CREATE INDEX idx_audit_actor ON audit_events(org_id, actor_id, timestamp DESC);

-- Client IP 索引 (用于安全分析)
CREATE INDEX idx_audit_client_ip ON audit_events(org_id, client_ip, timestamp DESC);

-- JSON 字段索引
CREATE INDEX idx_audit_details ON audit_events USING gin(details);
CREATE INDEX idx_audit_changes ON audit_events USING gin(changes);

-- 分区表 (按月分区)
CREATE TABLE audit_events_y2027m01 PARTITION OF audit_events
    FOR VALUES FROM ('2027-01-01') TO ('2027-02-01');

-- 审计日志归档表 (冷存储)
CREATE TABLE audit_events_archive (
    LIKE audit_events INCLUDING ALL
);

-- 合规报告视图
CREATE VIEW compliance_report_view AS
SELECT 
    org_id,
    date_trunc('day', timestamp) as date,
    action,
    resource_type,
    count(*) as event_count,
    count(DISTINCT actor_id) as unique_actors
FROM audit_events
WHERE timestamp > now() - interval '90 days'
GROUP BY org_id, date, action, resource_type;
```

## 查询与报表

### 审计查询 API

```go
type AuditFilter struct {
    OrgID       string     `json:"org_id"`
    StartTime   time.Time  `json:"start_time"`
    EndTime     time.Time  `json:"end_time"`
    Actions     []string   `json:"actions"`
    ActorIDs    []string   `json:"actor_ids"`
    ResourceType string    `json:"resource_type"`
    ResourceID  string     `json:"resource_id"`
    ClientIP    string     `json:"client_ip"`
    Keyword     string     `json:"keyword"`
    
    // 分页
    Page     int `json:"page"`
    PageSize int `json:"page_size"`
}

// 查询示例
// GET /api/v1/audit/events?start_time=2027-01-01&end_time=2027-01-31&actions=user.create,user.update&page=1&page_size=100
```

### 预置合规报告

| 报告名称 | 描述 | 周期 |
|----------|------|------|
| **用户活动报告** | 用户登录/登出、操作日志 | 每日 |
| **权限变更报告** | 角色分配、策略变更 | 每周 |
| **数据访问报告** | 敏感数据访问记录 | 每月 |
| **合规审计报告** | SOX/PCI/GDPR 合规性 | 按需 |

### 报告导出

```go
type ExportFormat string

const (
    ExportFormatJSON  ExportFormat = "json"
    ExportFormatCSV  ExportFormat = "csv"
    ExportFormatPDF  ExportFormat = "pdf"
)

// 导出示例
// POST /api/v1/audit/export
// {
//   "filter": {...},
//   "format": "pdf",
//   "report_type": "compliance_weekly"
// }
```

## 安全与合规

### 数据完整性

```go
// 审计事件签名 (防篡改)
type SignedAuditEvent struct {
    AuditEvent
    Signature string `json:"signature"`  // HMAC-SHA256
    KeyID     string `json:"key_id"`     // 用于验证的密钥 ID
}

// 定期哈希链密封
func SealDailyChain(events []AuditEvent) (*ChainSeal, error) {
    var prevHash []byte
    for _, e := range events {
        payload := e.ID + e.Timestamp.String() + string(prevHash)
        hash := sha256.Sum256([]byte(payload))
        prevHash = hash[:]
    }
    
    return &ChainSeal{
        RootHash: hex.EncodeToString(prevHash),
        KeyID:    currentSigningKeyID,
    }, nil
}
```

### 访问控制

- 审计日志读取权限仅限于 **Admin** 和 **Auditor** 角色
- 审计日志导出需要 **双重审批**
- 审计日志删除需要 **Owner** 授权

### 数据保留

| 数据类别 | 保留期限 | 存储位置 |
|----------|----------|----------|
| 90天内 | 90天 | Primary (PostgreSQL) |
| 90天-1年 | 9个月 | Search Index (OpenSearch) |
| 1年以上 | 7年 | Cold Storage (S3) |

## API 端点

| 方法 | 路径 | 描述 |
|------|------|------|
| GET | /api/v1/audit/events | 查询审计事件 |
| GET | /api/v1/audit/events/:id | 获取事件详情 |
| POST | /api/v1/audit/export | 导出审计报告 |
| GET | /api/v1/audit/reports | 列出预置报告 |
| POST | /api/v1/audit/reports/:id/generate | 生成报告 |
| GET | /api/v1/audit/reports/:id/download | 下载报告 |

## 相关文档

- [IAM 架构设计](iam.md)
- [RBAC 设计](rbac.md)
- [多租户架构](multi-tenant.md)
- [证据包导出方案](../roadmaps/EVIDENCE-PACKAGE-FOR-AUDITORS.md)
