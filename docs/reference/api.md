# Aetheris API Reference

> 面向 Aetheris v2.x 用户的 HTTP API 快速参考。完整契约见 [api-contract.md](api-contract.md)。

## Base URL

```
http://localhost:8080/api
```

## 认证

```bash
# Header 认证（当前实现）
Authorization: Bearer <token>
```

## Jobs API

### 提交 Job

```http
POST /api/runs
Content-Type: application/json

{
  "workflow_id": "agent_message",
  "input": {
    "agent_id": "refund-agent",
    "message": "请退款订单 order-123"
  }
}
```

**Response:**

```json
{
  "job_id": "job-abc123",
  "status": "pending",
  "created_at": "2026-04-19T12:00:00Z"
}
```

### 查询 Job 状态

```http
GET /api/jobs/:id
```

```json
{
  "job_id": "job-abc123",
  "status": "parked",
  "agent_id": "refund-agent",
  "current_node": "wait_approval",
  "created_at": "2026-04-19T12:00:00Z"
}
```

**Job Status 枚举:** `pending` | `running` | `parked` | `completed` | `failed` | `cancelled`

### 发送 Signal（唤醒 Parked Job）

```http
POST /api/jobs/:id/signal
Content-Type: application/json

{
  "correlation_key": "approval-abc123",
  "payload": {
    "approved": true,
    "reason": "客户投诉合理"
  }
}
```

### 获取执行 Trace

```http
GET /api/jobs/:id/trace
```

```json
{
  "job_id": "job-abc123",
  "timeline": [
    {"time": "12:00:00", "event": "JobCreated"},
    {"time": "12:00:05", "event": "StepStarted", "node": "query_order"},
    {"time": "12:00:10", "event": "ToolCompleted", "node": "query_order"},
    {"time": "12:00:15", "event": "JobParked", "node": "wait_approval", "correlation_key": "approval-abc123"},
    {"time": "15:12:30", "event": "SignalReceived", "payload": {"approved": true}},
    {"time": "15:12:35", "event": "StepStarted", "node": "send_refund"},
    {"time": "15:12:40", "event": "JobCompleted"}
  ]
}
```

### 获取 Replay 数据

```http
GET /api/jobs/:id/replay
```

### 停止 Job

```http
POST /api/jobs/:id/stop
```

### 获取 Job 事件流

```http
GET /api/jobs/:id/events
```

## Agents API

### 创建 Agent

```http
POST /api/agents
Content-Type: application/json

{
  "name": "refund-agent",
  "description": "退款审批 Agent",
  "type": "react",
  "llm": "default",
  "tools": ["query_order", "send_refund"]
}
```

### 列出 Agents

```http
GET /api/agents
```

### 获取 Agent 状态

```http
GET /api/agents/:id/state
```

### 列出 Agent 的 Jobs

```http
GET /api/agents/:id/jobs?status=running
```

## Documents API

### 上传文档

```http
POST /api/documents/upload
Content-Type: multipart/form-data

file: <binary>
knowledge_id: "kb-123"
```

### 列出文档

```http
GET /api/documents?knowledge_id=kb-123
```

## MCP Tool API

> 通过 MCP Gateway 注册的工具通过 MCP 协议调用，不走 REST API。

```bash
# 通过 MCP 协议列出工具
mcp__tools__list

# 通过 MCP 协议调用工具
mcp__tools__call
{
  "name": "mcp-github",
  "arguments": {
    "action": "search_repos",
    "query": "aetheris golang",
    "limit": 5
  }
}
```

## System API

### 健康检查

```http
GET /health
```

### 版本信息

```http
GET /api/version
```

## 错误格式

所有错误返回统一格式：

```json
{
  "error": {
    "code": "JOB_NOT_FOUND",
    "message": "Job job-xxx not found",
    "details": {}
  }
}
```

**常见错误码:**

| Code | HTTP Status | 说明 |
|------|-------------|------|
| `JOB_NOT_FOUND` | 404 | Job 不存在 |
| `AGENT_NOT_FOUND` | 404 | Agent 不存在 |
| `INVALID_SIGNAL_KEY` | 400 | correlation_key 不匹配 |
| `JOB_NOT_PARKED` | 400 | Job 未在 Parked 状态，无法 Signal |
| `TOOL_NOT_FOUND` | 404 | 工具未注册 |
| `TOOL_EXECUTION_ERROR` | 500 | 工具执行失败 |
| `UNAUTHORIZED` | 401 | 未认证 |
| `RATE_LIMITED` | 429 | 请求过于频繁 |

## 速率限制

| 端点 | 限制 |
|------|------|
| `POST /api/runs` | 100 req/min |
| `POST /api/jobs/:id/signal` | 60 req/min |
| `GET /api/jobs/:id/*` | 300 req/min |

## See Also

- [API Contract](api-contract.md) — 完整 API 契约与兼容性承诺
- [Getting Started](../guides/get-started.md) — 端到端使用教程
- [Observability](../guides/observability.md) — Trace 与监控
