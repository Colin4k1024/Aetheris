# Agent SDK 使用说明

`pkg/agent/sdk` 提供高层 Agent API，屏蔽 Job、Event、Planner、Runner 等底层概念，适合应用开发者「提交任务、取回回答」的用法。

> Runtime-first：新接入默认对接 `/api/runs/*` + `/api/jobs/*`。`/api/agents/*` 仅作为兼容迁移 facade。

## 推荐用法

```go
agent := sdk.NewAgent(runtime, "my-agent-id")
agent.RegisterTool("search", searchTool)
answer, err := agent.Run(ctx, "用户问题")
```

- **runtime** 实现 [AgentRuntime](pkg/agent/sdk/runtime.go)：`Submit(ctx, agentID, goal, sessionID)`、`WaitCompleted(ctx, jobID)`。由应用层注入（封装 JobStore + Runner 或 HTTP 客户端）。
- **RegisterTool**：若 runtime 实现 [ToolRegistrar](pkg/agent/sdk/runtime.go)，工具会注册到该 Agent；否则忽略。
- **Run**：内部 Submit → WaitCompleted，返回最终回答；超时由 `WithWaitTimeout` 或 context 控制。

## 与底层的关系

| SDK                        | 底层                                                                                    |
| -------------------------- | --------------------------------------------------------------------------------------- |
| Agent.Run(ctx, query)      | JobStore.Create → Scheduler/Worker 拉取 → Runner.RunForJob → Session 最后一条 assistant |
| AgentRuntime.Submit        | Run/Job 提交（runtime-first；可兼容 Agent facade）                                      |
| AgentRuntime.WaitCompleted | 轮询 Job 状态或 Watch 事件，完成后从 Session/Job 取回答                                 |

对接真实 API 时，实现一个 AgentRuntime：

- **推荐**：`Submit` 调用 `POST /api/runs`，并保存 `run_id`（可选）+ `job_id`（如业务返回/映射提供）。
- **状态查询**：`WaitCompleted` 轮询 `GET /api/jobs/:id`（以及按需读取 `GET /api/runs/:id`）。
- **兼容路径**：如使用 `POST /api/agents/:id/message`，可读取响应内 `runtime_submission` 取得 canonical runtime 映射信息。

## 示例

- [examples/sdk_agent](examples/sdk_agent) — 使用 MockRuntime 的极简示例，可直接 `go run ./examples/sdk_agent`。

## AgentFactory 集成（推荐）

对于新项目，推荐通过 `AgentFactory` 配置驱动创建 Agent，而非手动组装 Planner + Executor：

```go
import "rag-platform/internal/runtime/eino"

// 获取已从 agents.yaml 加载的 Runner
runner, ok := agentFactory.GetRunner("my_agent")
if !ok {
    log.Fatal("agent not found")
}

// 或编程式创建
runner, err := agentFactory.CreateAgent(ctx, eino.AgentBuildConfig{
    Name:        "custom_agent",
    Instruction: "你是一个有帮助的 AI 助手。",
    Type:        "react",
    Tools:       []string{"web_search"}, // 空 = 全部工具
    MaxSteps:    10,
    Streaming:   true,
})
```

`AgentFactory` 的优势：

- **配置驱动**：Agent 定义在 `configs/agents.yaml`，无需修改代码即可新增/调整 Agent
- **工具自动桥接**：通过 `RegistryToolBridge` 自动将 `RuntimeTool` 转为 Eino `InvokableTool`
- **Runner 缓存**：同名 Agent 复用已创建的 Runner 实例
- **工具过滤**：通过 `tools:` 字段限制每个 Agent 可用的工具子集

与 SDK 的关系：SDK 的 `AgentRuntime.Submit` 最终通过 `AgentFactory` 获取 Runner 执行。新接入建议直接使用 `AgentFactory`，SDK 层面做轻量封装。

详见 [concepts/adk.md](../concepts/adk.md)。

## Step Programming Model（2.0）

编写 Step 时须遵守强约束，否则 Replay 与 at-most-once 保证失效。

**允许**：纯计算、调用 Tool（经 Runtime 执行）、读 runtime state（通过 `sdk.Now(ctx)`、`sdk.UUID(ctx)`、`sdk.HTTP(ctx, ...)`、`sdk.JobID(ctx)`、`sdk.StepID(ctx)`）。

**禁止**：goroutine、channel、time.Sleep、直接 `time.Now()`/`uuid.New()`/`http.Get`、裸外部 IO。

- **step.go**：`StepFunc` 类型与契约说明。
- **runtime_context.go**：`Now(ctx)`、`UUID(ctx)`、`HTTP(ctx, effectID, doRequest)`、`JobID(ctx)`、`StepID(ctx)`；Runner 通过 `WithRuntimeContext(ctx, impl)` 注入实现，Replay 时从事件注入。
- **tool.go**：Tool 须经 Runtime 执行并记录。

详见 [design/step-contract.md](../design/step-contract.md)。

## 参考

- [usage.md](usage.md) — API 与 Job 流程
- [pkg/agent](pkg/agent) — 非 SDK 的 Agent 门面（Planner + Executor + Registry）
- [design/step-contract.md](../design/step-contract.md) — Step 语义契约
