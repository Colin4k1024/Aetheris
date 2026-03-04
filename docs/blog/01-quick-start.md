# Aetheris 入门 - 5 分钟快速开始

> 本文将帮助你在 5 分钟内快速启动 Aetheris 服务，创建第一个 Agent 并完成首次对话。

## 什么是 Aetheris？

Aetheris（又称 CoRag）是一个面向 AI 智能体的**持久化运行时**，提供：
- 任务规划与执行（Planner + Runner）
- 崩溃恢复与重试（Scheduler Lease Fencing）
- 可观测性（Trace/Replay/Evidence Chain）
- 企业级特性（RBAC、审计、数据脱敏）

## 快速开始

### 1. 克隆并构建

```bash
git clone https://github.com/aetheris-ai/CoRag.git
cd CoRag
make build
```

### 2. 启动服务（快速体验模式）

使用内存模式，无需 PostgreSQL：

```bash
# 确保 configs/api.yaml 中 jobstore.type 为 memory
make run
```

验证服务启动：

```bash
curl http://localhost:8080/api/health
# {"status":"ok"}
```

### 3. 创建第一个 Agent

```bash
curl -X POST http://localhost:8080/api/agents \
  -H "Content-Type: application/json" \
  -d '{"name":"my-first-agent"}'
```

返回示例：
```json
{"id":"agent-abc123","name":"my-first-agent","created_at":"..."}
```

记录返回的 `agent_id`（如 `agent-abc123`）。

### 4. 发送消息

```bash
curl -X POST http://localhost:8080/api/agents/agent-abc123/message \
  -H "Content-Type: application/json" \
  -d '{"message":"你好！请介绍一下你自己"}'
```

返回 202 Accepted：
```json
{"job_id":"job-xyz789","status":"pending"}
```

### 5. 轮询结果

```bash
# 替换为实际的 job_id
curl http://localhost:8080/api/agents/agent-abc123/jobs/job-xyz789
```

当 `status` 变为 `completed` 时，响应中包含 `result` 字段即是对话结果。

### 6. 查看 Trace（可选）

打开浏览器访问：
```
http://localhost:8080/api/jobs/job-xyz789/trace/page
```

可以看到执行时间线、节点、工具调用等详细信息。

## 下一步

- 深入阅读：[完整功能测试指南](./guides/get-started.md)
- 了解架构：[核心概念](./concepts/devops.md)
- 企业功能：[RBAC 配置](./guides/m2-rbac-guide.md)
