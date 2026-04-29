# Quickstart — 5 分钟入门 Aetheris

> 本指南帮助你快速启动 Aetheris Runtime 并运行第一个 Agent。  
> 内嵌模式（Embedded）无需 Docker 或外部数据库，适合本地快速体验。

## 前置条件

- **Go 1.21+**（推荐 Go 1.26）
- **Git**
- 可选：OpenAI / Ollama / 任何兼容 OpenAI API 的模型服务（用于 LLM 调用）

---

## 步骤 1: 克隆并构建

```bash
git clone https://github.com/Colin4k1024/Aetheris.git
cd Aetheris
go mod download
make build
```

构建产物在 `bin/`：`api`、`worker`、`aetheris`（CLI）。

---

## 步骤 2: 以内嵌模式启动 API（无需 Docker）

内嵌模式使用本地 SQLite 作为 Job Store，无需外部依赖：

```bash
# 终端 1：启动 API
CONFIG_PATH=configs/api.embedded.yaml go run ./cmd/api
```

```bash
# 终端 2：健康检查
curl http://localhost:8080/api/health
```

预期输出：
```json
{"status":"ok","version":"2.3.0"}
```

若想同时启动 Worker（Agent 任务执行器）：

```bash
# 终端 3：启动 Worker
CONFIG_PATH=configs/worker.embedded.yaml go run ./cmd/worker
```

---

## 步骤 3: 运行内置示例

### 最简单的 Agent（chat 示例）

```bash
cd examples/basic_agent
OPENAI_API_KEY=your-key-here \
OPENAI_BASE_URL=https://api.openai.com/v1 \
go run main.go
```

> 使用 Ollama 本地模型：
> ```bash
> OPENAI_API_KEY=ollama \
> OPENAI_BASE_URL=http://localhost:11434/v1 \
> go run main.go
> ```

### 带工具调用的 Agent

```bash
cd examples/eino_agent_with_tools
OPENAI_API_KEY=your-key-here go run main.go
```

### Human-in-the-Loop（人工审批）

```bash
cd examples/human_approval_agent
OPENAI_API_KEY=your-key-here go run main.go
```

---

## 步骤 4: 通过 HTTP API 创建并运行 Agent Job

```bash
# 创建一个 Agent Job
curl -X POST http://localhost:8080/api/v1/agents \
  -H "Content-Type: application/json" \
  -d '{
    "name": "code-reviewer",
    "system_prompt": "你是一个专业的代码审查助手，检查代码的潜在问题和最佳实践。"
  }'

# 提交任务
curl -X POST http://localhost:8080/api/v1/agents/code-reviewer/run \
  -H "Content-Type: application/json" \
  -d '{
    "user_message": "请审查这段代码：\n\ndef fib(n):\n    return fib(n-1) + fib(n-2)"
  }'
```

---

## 步骤 5: 体验崩溃恢复（可选）

这是 Aetheris 的核心能力——任务在 Worker 崩溃后自动恢复：

```bash
# 1. 提交一个长时间运行的任务（先在步骤 3 的 Agent 上提交）
./bin/aetheris jobs list      # 获取 job_id

# 2. 强制杀死 Worker（模拟崩溃）
kill $(pgrep -f "cmd/worker")

# 3. 重新启动 Worker
CONFIG_PATH=configs/worker.embedded.yaml go run ./cmd/worker

# 4. 观察任务自动恢复
./bin/aetheris trace <job_id>
```

---

## 步骤 6: 查看执行事件流

```bash
# 列出所有 Job
./bin/aetheris jobs list

# 查看某个 Job 的完整事件溯源历史
./bin/aetheris trace <job_id>
```

---

## 使用完整 PostgreSQL 模式

若需要完整的持久化和多 Worker 能力：

```bash
# 需要设置 API_MIDDLEWARE_JWT_KEY（任意字符串）
cp deployments/compose/.env.example deployments/compose/.env
# 编辑 .env，至少填写 API_MIDDLEWARE_JWT_KEY

docker compose -f deployments/compose/docker-compose.yml up -d
curl http://localhost:8080/api/health
```

---

## 下一步

- [MCP Gateway 集成](../mcp/integration.md) — 用 MCP 工具扩展 Agent 能力
- [事件溯源设计](../../design/core.md) — 深入理解 Aetheris 核心原理
- [Human-in-the-Loop 示例](../../examples/human_approval_agent/) — 构建需要人工审批的 Agent
- [多 Agent 协作示例](../../examples/multi_agent_collaboration/) — Supervisor + Worker 模式
