# Eino ADK 集成说明

本文说明项目中 [Eino ADK](https://www.cloudwego.io/zh/docs/eino/core_modules/eino_adk/) 的接入方式、AgentFactory 配置驱动创建、Tool Bridge 工具桥接，以及对话/恢复/流式接口用法。

## 架构概览

Aetheris 的 Agent 构建完全基于 Eino 框架，核心组件：

```
configs/agents.yaml
  → AgentFactory.GetOrCreateFromConfig()
    → 为每个 agent 构建 Eino ADK Runner
      → collectTools(): Engine 内置工具 + Registry 工具（via ToolBridge）
      → ChatModelAgent + ToolsConfig
      → adk.Runner (cached)

Tool Registry (native + MCP)
  → RegistryToolBridge.EinoTools()
    → registryToolAdapter (implements tool.InvokableTool)
      → Info() 将 Schema 映射为 Eino ParameterInfo
      → InvokableRun() 反序列化 JSON 参数并调用 Tool.Execute()
```

### 核心类型

| 类型 | 包 | 说明 |
|------|-----|------|
| `AgentFactory` | `internal/runtime/eino` | Agent 工厂：从配置 + 工具注册表 + LLM 构建 Runner |
| `AgentBuildConfig` | `internal/runtime/eino` | 单个 Agent 的构建配置（Name, Tools, MaxSteps 等） |
| `RegistryToolBridge` | `internal/runtime/eino` | 将 RuntimeToolRegistry 中的工具转为 Eino InvokableTool |
| `RuntimeTool` | `internal/runtime/eino` | 工具接口（Name/Description/Schema/Execute） |
| `RuntimeToolRegistry` | `internal/runtime/eino` | 工具注册表接口（List） |

## AgentFactory — 配置驱动的 Agent 创建

所有 Agent 构建都经由 `AgentFactory`，不再使用旧的 `runtime.NewAgent()`。

### 启动流程（`internal/app/api/app.go`）

1. 创建 `AgentFactory`：`eino.NewAgentFactory(engine, registry)`
2. 从 `configs/agents.yaml` 加载所有 agent 定义：`agentFactory.GetOrCreateFromConfig(ctx, agentsCfg)`
3. 每个 agent 自动获得：ChatModel + 工具集（内置 + Registry） + SystemPrompt
4. 通过 `agentFactory.GetRunner("agent_name")` 获取执行器

### 编程式创建

```go
runner, err := agentFactory.CreateAgent(ctx, eino.AgentBuildConfig{
    Name:        "my_agent",
    Description: "自定义 Agent",
    Instruction: "你是一个有帮助的 AI 助手。",
    Type:        "react",
    Tools:       []string{"web_search", "calculator"}, // 空 = 全部
    MaxSteps:    10,
    Streaming:   true,
})
```

## Tool Bridge — 工具桥接层

`RegistryToolBridge` 解决了 Aetheris 工具体系与 Eino 工具体系的对接：

- **输入**：`RuntimeToolRegistry`（包含所有 Native + MCP 工具）
- **输出**：`[]tool.BaseTool`（Eino 格式，可直接传入 ADK Agent）
- **Session 传递**：通过 `WithSession(ctx, sess)` / `sessionFromContext(ctx)` 在 context 中传递 Session

### Schema 映射

工具的 `Schema() map[string]any` 自动映射为 Eino `ParameterInfo`：

- 支持 JSON Schema 格式（`{"type":"object","properties":{...},"required":[...]}`）
- 支持简单 key-value 格式（`{"query":"搜索关键词"}`）
- 类型映射：string → `schema.String`, integer → `schema.Integer`, number → `schema.Number`, boolean → `schema.Boolean`, array → `schema.Array`, object → `schema.Object`

## ADK 对话路径

> Runtime-first 说明：系统对外 canonical 提交路径是 `/api/runs/*` + `/api/jobs/*`。
> ADK 专项接口（`/api/agent/*`）用于兼容与专项能力。

当配置中 **agent.adk.enabled** 未设为 `false` 时，以下接口由 **ADK Runner** 执行：

- **POST /api/agent/run** — 单次对话，请求体 `{"query":"...", "session_id":"可选"}`，返回 `answer`、`session_id`、`steps` 等。
- **POST /api/agent/resume** — 从 checkpoint 恢复，请求体 `{"checkpoint_id":"...", "session_id":"可选"}`。
- **POST /api/agent/stream** — 流式对话，请求体同 run，响应为 SSE（`text/event-stream`），事件中含 `answer`、`session_id`。

Session 历史会转换为 ADK 消息（最近 20 轮）传入 Runner，执行结果写回 Session 并保存。

## CheckPointStore 与中断/恢复

Runner 使用 **CheckPointStore**（当前为内存实现）保存中断点。当 Agent 内调用 `adk.Interrupt(ctx, info)` 时，框架会写入 checkpoint，调用方可通过 **POST /api/agent/resume** 传入返回的 `checkpoint_id` 恢复执行。配置项 **agent.adk.checkpoint_store** 目前仅支持 `memory`，后续可扩展为 postgres/redis 等持久化存储。

## 禁用 ADK

在 **configs/api.yaml** 中设置：

```yaml
agent:
  adk:
    enabled: false
```

则 **POST /api/agent/run** 将使用原 Plan→Execute Agent（Planner + Executor + Tools），**/api/agent/resume** 与 **/api/agent/stream** 会返回 503（ADK Runner 未配置）。

## Multi-Agent 支持（已实现）

通过 `configs/agents.yaml` 定义多个命名 Agent，`AgentFactory` 在启动时批量创建：

```yaml
agents:
  react:
    type: "react"
    tools: ["web_search", "calculator"]
    system_prompt: "..."
  deer:
    type: "deer"
    tools: []  # 空 = 使用全部可用工具
    system_prompt: "..."
```

每个 Agent 获得独立的 Eino ADK Runner，可通过 `agentFactory.GetRunner("react")` 或 `agentFactory.GetRunner("deer")` 分别调用。`agentFactory.ListAgents()` 返回所有已创建的 Agent 名称。

参见 [Eino ADK Agent 实现](https://www.cloudwego.io/zh/docs/eino/core_modules/eino_adk/agent_implementation/)。
