# Troubleshooting Guide

> **版本**: v2.3.0+

本指南帮助您诊断和解决使用 Aetheris 过程中遇到的常见问题。

---

## 目录

1. [服务启动问题](#1-服务启动问题)
2. [Job 执行问题](#2-job-执行问题)
3. [认证与权限问题](#3-认证与权限问题)
4. [RAG 相关问题](#4-rag-相关问题)
5. [性能问题](#5-性能问题)
6. [数据问题](#6-数据问题)
7. [调试技巧](#7-调试技巧)

---

## 1. 服务启动问题

### 1.1 API 无法启动

**症状**: `go run ./cmd/api` 失败

**常见原因**:

- 端口 8080 被占用
- 配置文件格式错误
- 缺少必要依赖

**解决方案**:

```bash
# 检查端口占用
lsof -i :8080

# 查看详细错误日志
go run ./cmd/api 2>&1

# 验证配置文件
cat configs/api.yaml
cat configs/model.yaml
```

### 1.2 无法连接 PostgreSQL

**症状**: 报错 `dial tcp: connection refused`

**解决方案**:

```bash
# 启动 PostgreSQL (Docker)
docker run -d --name aetheris-pg -p 5432:5432 \
  -e POSTGRES_USER=aetheris -e POSTGRES_PASSWORD=aetheris \
  -e POSTGRES_DB=aetheris postgres:15-alpine

# 或使用项目 Compose
docker compose -f deployments/compose/docker-compose.yml up -d postgres

# 验证连接
psql -h localhost -p 5432 -U aetheris -d aetheris
```

### 1.3 缺少 LLM 配置

**症状**: 报错 `no LLM configured` 或 `model not found`

**解决方案**:

确保 `configs/model.yaml` 中配置了 LLM：

```yaml
defaults:
  llm: "openai.gpt_35_turbo"
  embedding: "openai.text_embedding_ada_002"

providers:
  openai:
    api_key: "${OPENAI_API_KEY}"
    models:
      - name: "gpt_35_turbo"
        type: "llm"
      - name: "text_embedding_ada_002"
        type: "embedding"
```

或使用 Qwen:

```yaml
defaults:
  llm: "qwen.qwen3_max"
  embedding: "qwen.text_embedding"

providers:
  qwen:
    api_key: "${DASHSCOPE_API_KEY}"
    models:
      - name: "qwen3_max"
        type: "llm"
      - name: "text_embedding"
        type: "embedding"
```

---

## 2. Job 执行问题

### 2.1 Job 一直处于 Pending

**症状**: Job 状态长时间停留在 `pending`

**常见原因**:

- 没有 Worker 进程运行（使用 PostgreSQL 时）
- Worker 无法连接到 PostgreSQL
- Scheduler 未启动（内存模式）

**解决方案**:

```bash
# 检查 Worker 是否运行
ps aux | grep worker

# 启动 Worker
go run ./cmd/worker

# 或使用 Docker Compose 启动完整栈
./scripts/local-2.0-stack.sh start
```

### 2.2 Job 失败 (Failed)

**症状**: Job 状态为 `failed`

**排查步骤**:

```bash
# 查看 Job 事件
curl http://localhost:8080/api/jobs/<job_id>/events

# 查看 Job Trace
curl http://localhost:8080/api/jobs/<job_id>/trace
```

**常见原因**:

- LLM API 调用失败（检查 API Key）
- 工具执行失败（检查工具配置）
- 超过最大重试次数

### 2.3 重复执行 (Duplicate Execution)

**症状**: 同一个 Job 被执行多次

**解决方案**:

Aetheris 使用 Tool Ledger 确保 at-most-once 执行。检查:

1. 确认使用 `jobstore.type=postgres`（生产环境）
2. 确保只有单一 Scheduler 在运行
3. 检查是否有多个 Worker 尝试执行同一 Job

### 2.4 Worker 崩溃后 Job 未恢复

**症状**: Worker 崩溃后 Job 状态卡住

**解决方案**:

```bash
# 确认 PostgreSQL 中的 Job 状态
# Job 应该自动被其他 Worker 接收

# 手动检查 Job 状态
curl http://localhost:8080/api/jobs/<job_id>

# 如果 Job 卡住，可以手动重置
# (需要直接操作数据库)
```

---

## 3. 认证与权限问题

### 3.1 返回 401 Unauthorized

**症状**: API 请求返回 401

**常见原因**:

- 生产模式下未提供 JWT token
- JWT secret 配置错误
- Token 过期

**解决方案**:

```bash
# 获取 token (如果启用了登录)
curl -X POST http://localhost:8080/api/login \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"admin"}'

# 使用 token 请求
curl -H "Authorization: Bearer <token>" \
  http://localhost:8080/api/agents
```

### 3.2 返回 403 Forbidden

**症状**: 已认证但无权限

**解决方案**:

检查 RBAC 配置和用户角色:

```bash
# 查看当前用户角色
curl -H "Authorization: Bearer <token>" \
  http://localhost:8080/api/rbac/role

# 检查权限
curl -X POST -H "Authorization: Bearer <token>" \
  http://localhost:8080/api/rbac/check \
  -H "Content-Type: application/json" \
  -d '{"permission":"job:create"}'
```

### 3.3 生产模式验证失败

**症状**: 报错 `production mode requires...`

**解决方案**:

生产环境需要配置:

```yaml
# configs/api.yaml
app:
  env: "production" # 或设置 AETHERIS_ENV=production

auth:
  jwt:
    secret: "your-secure-secret-key" # 必须修改默认密钥
    enabled: true

jobstore:
  type: "postgres" # 生产环境必须使用 PostgreSQL
  postgres:
    dsn: "postgres://user:pass@host:5432/db?sslmode=require"

cors:
  allowed_origins:
    - "https://your-domain.com" # 不能使用 *
```

---

## 4. RAG 相关问题

### 4.1 文档上传失败

**症状**: `POST /api/documents/upload` 返回错误

**排查步骤**:

```bash
# 检查支持的格式
# 支持: PDF, Markdown, TXT, HTML 等

# 检查文件大小限制
# 默认 10MB，可在配置中修改

# 查看详细错误
curl -v -X POST http://localhost:8080/api/documents/upload \
  -F "file=@./document.pdf"
```

### 4.2 检索结果不准确

**解决方案**:

1. 调整 `top_k` 参数
2. 检查文档分块大小
3. 验证 embedding 模型配置

```bash
# 测试检索
curl -X POST http://localhost:8080/api/query \
  -H "Content-Type: application/json" \
  -d '{"query":"your question","top_k":10}'
```

### 4.3 异步上传任务卡住

**症状**: `POST /api/documents/upload/async` 返回 task_id 但状态不更新

**解决方案**:

```bash
# 检查上传状态
curl http://localhost:8080/api/documents/upload/status/<task_id>

# 检查 Worker 日志
# 确认 ingest pipeline 正常运行
```

---

## 5. 性能问题

### 5.1 API 响应慢

**排查步骤**:

1. 检查数据库连接池配置
2. 查看是否有慢查询
3. 检查系统资源 (CPU, 内存)

```bash
# 查看系统指标
curl http://localhost:8080/api/system/metrics

# 查看 Worker 状态
curl http://localhost:8080/api/system/workers
```

### 5.2 向量检索慢

**解决方案**:

- 使用更快的 embedding 模型
- 调整批量大小
- 使用专用向量数据库 (Milvus, Pinecone 等)

---

## 6. 数据问题

### 6.1 数据丢失

**症状**: 重启后数据丢失

**原因**: 使用内存存储

**解决方案**:

配置持久化存储:

```yaml
# configs/api.yaml
storage:
  metadata:
    type: "postgres"
    dsn: "postgres://user:pass@host:5432/db"
  vector:
    type: "milvus"
    collection: "aetheris_docs"
```

### 6.2 事件流不一致

**症状**: `GET /api/jobs/:id/verify` 失败

**解决方案**:

```bash
# 检查一致性
curl http://localhost:8080/api/forensics/consistency/<job_id>

# 导出完整事件链
curl http://localhost:8080/api/jobs/<job_id>/export
```

---

## 7. 调试技巧

### 7.1 启用详细日志

```bash
# 设置日志级别
export AETHERIS_LOG_LEVEL=debug

# 或在配置中
# configs/api.yaml
logging:
  level: "debug"
```

### 7.2 使用 CLI 调试

```bash
# 创建 Agent（legacy facade）
go run ./cmd/cli agent create my-agent

# 发送消息（legacy facade）
go run ./cmd/cli chat <agent_id>

# 查看 Job 状态（legacy facade）
go run ./cmd/cli jobs <agent_id>

# 查看 Trace
go run ./cmd/cli trace <job_id>

# 查看 Replay
go run ./cmd/cli replay <job_id>
```

### 7.3 使用 Trace 页面

浏览器打开: `http://localhost:8080/api/jobs/<job_id>/trace/page`

### 7.4 查看 observability

```bash
# 查看概览
curl http://localhost:8080/api/observability/summary

# 查看卡住的 Job
curl http://localhost:8080/api/observability/stuck
```

---

## 获取帮助

如果以上方案无法解决您的问题:

1. 查看 [GitHub Issues](https://github.com/Colin4k1024/Aetheris/issues)
2. 加入 [Discord](https://discord.gg/PrrK2Mua)
3. 在 [Discussions](https://github.com/Colin4k1024/Aetheris/discussions) 提问
4. 查看 [design docs](../design/) 了解架构细节
