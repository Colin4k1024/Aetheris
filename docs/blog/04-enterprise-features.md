# Aetheris 企业级功能指南

> 本文介绍 Aetheris v2.2.0+ 的企业级功能：RBAC 权限管理、审计日志、数据脱敏与留存策略。

## 概述

Aetheris 从 v2.0 起提供完整的企业级特性：

| 特性 | 版本 | 说明 |
|------|------|------|
| **RBAC** | v2.0+ | 角色与权限控制 |
| **审计日志** | v2.0+ | 操作记录与取证 |
| **数据脱敏** | v2.0+ | 敏感信息自动脱敏 |
| **数据留存** | v2.0+ | Tombstone 与自动清理 |
| **Region 调度** | v2.2.0 | 区域感知调度 |
| **SLA/Quota** | v2.2.0 | 限流与配额管理 |

## 1. RBAC 权限管理

### 角色模型

Aetheris 支持以下内置角色：

| 角色 | 权限 |
|------|------|
| **admin** | 全部权限 |
| **operator** | 运维操作（重启、监控） |
| **developer** | Agent 管理、部署 |
| **viewer** | 只读（查询、Trace） |

### API 示例

```bash
# 创建角色
curl -X POST http://localhost:8080/api/rbac/role \
  -H "Content-Type: application/json" \
  -d '{
    "name": "agent-operator",
    "permissions": ["agent:create", "agent:delete", "job:cancel"]
  }'

# 分配角色给用户
curl -X POST http://localhost:8080/api/rbac/assignment \
  -H "Content-Type: application/json" \
  -d '{"user_id":"user-123","role":"agent-operator"}'

# 权限检查
curl -X POST http://localhost:8080/api/rbac/check \
  -H "Content-Type: application/json" \
  -d '{"user_id":"user-123","permission":"agent:delete"}'
```

### 配置 PostgreSQL RoleStore

```yaml
# configs/api.yaml
rbac:
  store: postgres
  role_store_dsn: "postgres://user:pass@host:5432/rbac?sslmode=require"
```

详细配置见 [RBAC 配置指南](../guides/m2-rbac-guide.md)。

## 2. 审计日志

### 审计事件类型

| 事件 | 说明 |
|------|------|
| `agent_created` | Agent 创建 |
| `agent_deleted` | Agent 删除 |
| `job_started` | Job 启动 |
| `job_completed` | Job 完成 |
| `job_failed` | Job 失败 |
| `tool_called` | 工具调用 |
| `rbac_changed` | 权限变更 |

### 查询审计日志

```bash
# 按时间范围查询
curl "http://localhost:8080/api/audit/logs?start=2026-01-01&end=2026-03-01"

# 按 Agent 过滤
curl "http://localhost:8080/api/audit/logs?agent_id=agent-123"

# 按事件类型过滤
curl "http://localhost:8080/api/audit/logs?event=tool_called"
```

### 导出审计报告

```bash
# 导出为 CSV
curl "http://localhost:8080/api/audit/export?format=csv" -o audit-2026.csv

# 导出为 JSON
curl "http://localhost:8080/api/audit/export?format=json" -o audit-2026.json
```

详细说明见 [审计与取证 API](../guides/m3-forensics-api-guide.md)。

## 3. 数据脱敏

### 脱敏策略

Aetheris 支持配置敏感字段自动脱敏：

```yaml
# configs/api.yaml
redaction:
  enabled: true
  rules:
    - field: "phone"
      pattern: "1[3-9]\\d{9}"
      replacement: "1*******"
    - field: "email"
      pattern: "[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\\.[a-zA-Z]{2,}"
      replacement: "***@***.***"
    - field: "credit_card"
      pattern: "\\d{16}"
      replacement: "************"
```

### API 使用

```bash
# 创建脱敏规则
curl -X POST http://localhost:8080/api/redaction/rules \
  -H "Content-Type: application/json" \
  -d '{
    "name": "phone-masking",
    "field": "message",
    "pattern": "1[3-9]\\d{9}",
    "replacement": "1*******"
  }'
```

详细配置见 [数据脱敏指南](../guides/m2-redaction-guide.md)。

## 4. 数据留存策略

### 策略配置

```yaml
# configs/api.yaml
retention:
  enabled: true
  policies:
    - name: "short-term"
      job_status: ["completed", "failed"]
      max_age_days: 30
      action: "archive"  # archive 或 delete
    - name: "long-term"
      job_status: ["completed"]
      max_age_days: 365
      action: "tombstone"  # 保留元数据，删除实际数据
```

### Tombstone 机制

Tombstone 保留：
- Job ID、创建时间、完成时间
- 执行结果摘要
- 审计日志引用

删除：
- 实际的 LLM 输入/输出
- 工具调用参数与结果
- 中间状态

```bash
# 查询 Tombstone Job
curl http://localhost:8080/api/jobs/job-xxx
# 返回: {"id":"job-xxx","status":"completed","tombstone":true,...}
```

详细说明见 [留存策略指南](../guides/m2-retention-guide.md)。

## 5. Region 感知调度 (v2.2.0)

### 配置 Region

```yaml
# configs/worker.yaml
scheduler:
  region: "us-east-1"
  allowed_regions: ["us-east-1", "us-west-2"]
```

### Region-Aware Job

```bash
# 创建指定 Region 的 Job
curl -X POST http://localhost:8080/api/agents/agent-xxx/message \
  -H "Content-Type: application/json" \
  -d '{"message":"处理任务","region":"us-west-2"}'
```

Worker 会优先调度到相同 Region 的节点执行，减少延迟。

## 6. SLA/Quota 管理 (v2.2.0)

### 配置 Quota

```bash
# 为 Agent 配置速率限制
curl -X POST http://localhost:8080/api/sla/quotas \
  -H "Content-Type: application/json" \
  -d '{
    "agent_id": "agent-123",
    "max_rpm": 100,        # 每分钟最大请求数
    "max_daily": 10000,    # 每日最大请求数
    "burst": 20            # 突发上限
  }'
```

### SLO 监控

```bash
# 获取 SLO 状态
curl http://localhost:8080/api/slo/status

# 返回示例
{
  "agent_id": "agent-123",
  "current_rpm": 45,
  "limit_rpm": 100,
  "daily_usage": 3200,
  "daily_limit": 10000,
  "slo_compliant": true
}
```

### 超过配额处理

```yaml
# configs/api.yaml
sla:
  on_quota_exceeded: "queue"  # queue, reject, or throttle
  throttle_delay_ms: 1000      # 延迟时间
```

## 7. 证据包导出

对于需要提供执行证据的场景：

```bash
# 导出 Job 证据包
curl -X POST http://localhost:8080/api/jobs/job-xxx/export \
  -o evidence-job-xxx.zip

# 验证证据包
aetheris verify evidence-job-xxx.zip
```

证据包包含：
- 完整事件链（带哈希）
- 审计日志
- 执行状态快照
- 工具调用记录

详细说明见 [证据包与取证](../guides/m3-evidence-graph-guide.md)。

## 总结

Aetheris 企业级功能完整覆盖：

1. **RBAC** — 细粒度权限控制
2. **审计日志** — 完整操作记录
3. **数据脱敏** — 敏感信息保护
4. **留存策略** — 平衡存储成本与合规
5. **Region 调度** — 全球化部署优化
6. **SLA/Quota** — 资源管控
7. **证据包** — 审计与取证

这些功能使 Aetheris 适用于金融、医疗、政府等高合规要求场景。

## 延伸阅读

- [RBAC 配置详解](../guides/m2-rbac-guide.md)
- [审计与取证 API](../guides/m3-forensics-api-guide.md)
- [数据脱敏配置](../guides/m2-redaction-guide.md)
- [留存策略配置](../guides/m2-retention-guide.md)
