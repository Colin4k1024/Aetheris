# Agent Integration Examples

本目录包含 Aetheris 集成现有 Agent 的完整示例，覆盖三种主要集成场景。

## 快速开始

### 前置条件

- Go 1.26.1+
- Python 3.10+（Python 示例）
- Docker（可选，用于容器化部署）

### 1. External HTTP Agent（推荐）

最简单的集成方式，你的 Agent 只需暴露一个 HTTP endpoint。

```bash
# 1. 启动 Python Agent
cd python-agent
pip install -r requirements.txt
python app.py  # 监听 :9001

# 2. 启动 Aetheris（新终端）
cd ../..
make run-embedded

# 3. 提交任务
curl -X POST http://localhost:8080/api/agents/my_python_agent/message \
  -H "Content-Type: application/json" \
  -H "Idempotency-Key: test-001" \
  -d '{"message": "Hello from Aetheris!"}'
```

### 2. LangChain Agent

如果你的 Agent 使用 LangChain，可以使用框架类型别名。

```bash
# 1. 启动 LangChain Agent
cd langchain-agent
pip install -r requirements.txt
export OPENAI_API_KEY=your_key_here
python app.py  # 监听 :9002

# 2. 配置 Aetheris 使用 langchain 类型
# 编辑 configs/agents-langchain.yaml

# 3. 启动 Aetheris
CONFIG_PATH=configs/agents-langchain.yaml make run-embedded
```

### 3. Docker Compose 一键启动

```bash
docker-compose up -d
# 所有服务自动启动并连接
```

---

## 集成架构

```
┌─────────────────────────────────────────────────────────────┐
│                      Aetheris Runtime                       │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────────────┐ │
│  │   API Layer  │→│ Job Scheduler│→│   Worker (DAG Exec)  │ │
│  └─────────────┘  └─────────────┘  └─────────────────────┘ │
│         │                │                      │           │
│         ▼                ▼                      ▼           │
│  ┌─────────────────────────────────────────────────────────┐│
│  │              external_agent_call Tool                    ││
│  │  • Idempotency-Key 传递                                 ││
│  │  • Job-ID 追踪                                          ││
│  │  • 超时控制                                             ││
│  │  • 响应验证                                             ││
│  └─────────────────────────────────────────────────────────┘│
│                          │                                   │
└──────────────────────────┼───────────────────────────────────┘
                           │ HTTP POST
                           ▼
         ┌─────────────────────────────────────┐
         │     Your Existing Agent             │
         │  • Python / JS / Go / Any Language  │
         │  • LangChain / AutoGen / CrewAI     │
         │  • Custom Framework                 │
         └─────────────────────────────────────┘
```

---

## 示例目录

| 目录 | 说明 | Agent 类型 |
|------|------|-----------|
| `python-agent/` | 通用 Python HTTP Agent | `external_http` |
| `langchain-agent/` | LangChain ReAct Agent | `langchain` |
| `configs/` | Aetheris 配置示例 | - |

---

## 配置详解

### External HTTP Agent 配置

```yaml
agents:
  my_agent:
    type: "external_http"
    description: "My existing agent"
    external:
      url: "http://localhost:9001/invoke"      # 必填：Agent endpoint
      timeout: "120s"                           # 可选：超时时间
      token_env: "MY_AGENT_TOKEN"               # 可选：Bearer token 环境变量
      protocol: "json"                          # 可选：json 或 sse_legacy
```

### 框架类型别名

```yaml
agents:
  langchain_agent:
    type: "langchain"                           # 框架别名
    external:
      url: "http://localhost:9002/invoke"
      framework: "langchain"                    # 可选：框架标识
      
  langgraph_agent:
    type: "langgraph"
    external:
      url: "http://localhost:9003/invoke"
```

---

## Agent 接口规范

### 请求格式

```json
{
  "message": "用户的任务描述",
  "session_id": "可选的会话ID",
  "metadata": {
    "agent_id": "my_agent",
    "job_id": "job-xxx",
    "idempotency_key": "key-xxx"
  }
}
```

### 响应格式

```json
{
  "answer": "Agent 的回答",
  "final": true,
  "metadata": {
    "custom_field": "value"
  }
}
```

### HTTP Headers

Aetheris 会在请求中添加以下 Headers：

| Header | 说明 |
|--------|------|
| `Content-Type` | `application/json` |
| `Idempotency-Key` | 幂等键，用于去重 |
| `X-Aetheris-Job-ID` | Aetheris 任务 ID |
| `X-Aetheris-Agent-ID` | Agent 配置 ID |
| `Authorization` | Bearer token（如果配置了 `token_env`） |

---

## 高级用法

### 1. SSE 流式协议

适用于支持 SSE 的 Agent（如 superagent-base）：

```yaml
agents:
  sse_agent:
    type: "external_http"
    external:
      url: "http://localhost:8888/api/v1/chat/stream"
      protocol: "sse_legacy"
      agent_id: "research-agent"
```

### 2. 嵌入式模式

当你需要 Aetheris 管理 Agent 内部步骤时：

```yaml
agents:
  embedded_agent:
    type: "langchain"
    external:
      mode: "embedded"
      url: "http://localhost:9000"
      manifest_path: "./configs/framework-agents/my_agent.manifest.json"
```

### 3. 多 Agent 协作

```yaml
agents:
  researcher:
    type: "external_http"
    external:
      url: "http://research-agent:9001/invoke"
      
  writer:
    type: "external_http"
    external:
      url: "http://writer-agent:9002/invoke"
      
  reviewer:
    type: "external_http"
    external:
      url: "http://reviewer-agent:9003/invoke"
```

---

## 故障排查

### 常见问题

**Q: Agent 返回 401 Unauthorized**
A: 检查 `token_env` 对应的环境变量是否设置。

**Q: 请求超时**
A: 调整 `timeout` 配置，或检查 Agent 是否正常运行。

**Q: 幂等键冲突**
A: 每个任务使用唯一的 `Idempotency-Key`。

### 调试技巧

```bash
# 查看 Aetheris 日志
docker-compose logs -f aetheris

# 查看 Agent 日志
docker-compose logs -f python-agent

# 手动测试 Agent endpoint
curl -X POST http://localhost:9001/invoke \
  -H "Content-Type: application/json" \
  -d '{"message": "test"}'
```

---

## 下一步

- 阅读 [External HTTP Agent 完整文档](../../docs/adapters/external-http-agent.md)
- 了解 [LangChain 集成详情](../../docs/adapters/langchain.md)
- 查看 [生产部署指南](../../docs/guides/deployment.md)
