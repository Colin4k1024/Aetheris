# AGENTS.md

This document provides guidelines for AI agents working with the Aetheris codebase.

## Project Overview

**Aetheris** (CoRag) is an execution runtime for intelligent agents — a durable, replayable, and observable environment where AI agents can plan, execute, pause, resume, and recover long-running tasks.

Key technologies:

- **Go 1.25.7** (see `go.mod` and CI)
- **Go module**: `rag-platform` (import path for all internal packages)
- **Cloudwego eino**: Workflow/DAG execution, Agent scheduling, Pipeline orchestration
- **Hertz**: HTTP framework for REST APIs
- **Viper**: Configuration management
- **PostgreSQL**: JobStore (event-sourced durable history)
- **Redis**: Cache, RAG, Vector Index

## Build, Lint, and Test Commands

### Build

```bash
# Build all binaries
go build ./...

# Build specific binary
go build -o bin/api ./cmd/api
go build -o bin/worker ./cmd/worker
go build -o bin/aetheris ./cmd/cli

# Build with race detector
go build -race ./...
```

### Run

```bash
# API service (default :8080)
go run ./cmd/api

# Worker service
go run ./cmd/worker

# CLI tool
go run ./cmd/cli

# With custom config
CONFIG_PATH=/path/to/config.yaml go run ./cmd/api
```

### Testing

```bash
# Run all tests
go test ./...

# Run tests with coverage
go test -coverprofile=coverage.out ./...

# View coverage in browser
go tool cover -html=coverage.out

# Run single test file
go test ./internal/pipeline/query/... -v

# Run single test function
go test -run TestQueryPipeline_ValidQuery ./internal/pipeline/query/...

# Run tests with race detector
go test -race ./...

# Run key integration tests (runtime + http)
go test -v ./internal/agent/runtime/executor ./internal/api/http
```

### Vet and Lint

```bash
# Go vet
go vet ./...

# Format code
gofmt -w .
gofmt -d .  # Show diff

# Static analysis
go run golang.org/x/tools/go/analysis/cmd/vet@latest ./...
```

### Dependencies

```bash
# Download dependencies
go mod download

# Tidy dependencies
go mod tidy

# Verify dependencies
go mod verify

# List dependencies
go list -m all
```

### Makefile Commands

The project provides a Makefile for convenient build and startup:

```bash
make help              # Show help
make build             # Build api, worker, and cli into bin/
make run               # Build and start API + Worker in background (one-command startup)
make run-api           # Build and start only API in background
make run-worker        # Build and start only Worker in background
make stop              # Stop API and Worker started by make run
make clean             # Remove bin/
make test              # Run tests
make test-integration  # Run key integration tests (runtime + http)
make docker-build      # Build runtime container image
make docker-run        # Start local 2.0 stack via Compose
make docker-stop       # Stop local 2.0 stack
make vet               # go vet
make fmt               # gofmt -w
make tidy              # go mod tidy
```

## Code Style Guidelines

### Imports

Organize imports in three groups with blank lines between:

1. Standard library
2. External packages (github.com/xxx)
3. Internal packages (rag-platform/xxx)

```go
import (
    "context"
    "fmt"
    "time"

    "github.com/cloudwego/hertz/pkg/app"
    "github.com/cloudwego/hertz/pkg/common/hlog"

    appcore "rag-platform/internal/app"
    "rag-platform/internal/runtime/eino"
)
```

### Formatting

- Use `gofmt` for automatic formatting
- Indent with tabs, not spaces
- No trailing whitespace
- Max line length: ~120 characters (soft limit)

### Naming Conventions

- **Packages**: lowercase, concise, meaningful (e.g., `app`, `pipeline`, `storage`)
- **Files**: lowercase with underscores only if needed for naming (e.g., `workflow.go`)
- **Exported types/functions**: PascalCase (e.g., `Workflow`, `CreateWorkflow`)
- **Unexported**: camelCase (e.g., `engine`, `parseDefaultKey`)
- **Interfaces**: Simple noun or verb+noun pattern (e.g., `Client`, `Retriever`)
- **Constants**: PascalCase or SCREAMING_SNAKE_CASE for constants (e.g., `ErrNotFound`, `MaxRetries`)
- **Variables**: camelCase, avoid single letters except loop indices

### Error Handling

- Use `pkg/errors` for error wrapping: `errors.Wrap(err, "message")`
- Use `errors.Wrapf` for formatted error messages
- Sentinel errors in `pkg/errors/errors.go`: `ErrNotFound`, `ErrInvalidArg`
- Return meaningful errors with context
- Handle errors at the appropriate level (don't ignore with `_`)
- Use `context.Context` for cancellation and timeouts
- Use `hlog.CtxErrorf` for logging errors in handlers

```go
if err != nil {
    return nil, fmt.Errorf("compile workflow failed: %w", err)
}

return nil, errors.Wrap(err, "failed to create client")
```

### Structs and Types

- Use struct tags for JSON serialization
- Use `binding` tags for Hertz request validation
- Keep structs focused and small

```go
type Query struct {
    ID        string                 `json:"id"`
    Text      string                 `json:"text"`
    Metadata  map[string]interface{} `json:"metadata"`
    CreatedAt time.Time              `json:"created_at"`
}
```

### Context Usage

- Pass `context.Context` as first parameter
- Use named context variables for clarity
- Check context cancellation in long-running operations

```go
func (h *Handler) Query(ctx context.Context, c *app.RequestContext) error {
    // ...
}
```

### Comments

- Use Chinese or English comments for public APIs and documentation (team preference)
- Comment exported types and functions
- Use sentence case for comments
- No commented-out code

```go
// Workflow 工作流
type Workflow struct {
    // ...
}

// CreateWorkflow 创建工作流
func CreateWorkflow(name, description string) *Workflow {
    // ...
}
```

### HTTP Handlers (Hertz)

- Use `consts.StatusXXX` for status codes
- Return consistent JSON response format
- Log errors with `hlog.CtxErrorf`
- Validate request parameters with `binding` tags

```go
func (h *Handler) Query(ctx context.Context, c *app.RequestContext) {
    var request struct {
        Query string `json:"query" binding:"required"`
        TopK  int    `json:"top_k"`
    }

    if err := c.BindJSON(&request); err != nil {
        c.JSON(consts.StatusBadRequest, map[string]string{
            "error": "请求参数错误",
        })
        return
    }
    // ...
}
```

### Testing

- Use table-driven tests when appropriate
- Test file naming: `xxx_test.go`
- Test function naming: `TestXxx`
- Use `t.Run` for sub-tests
- Prefer `require` over `assert` for clarity on failures

### Project Structure

```
cmd/              # Entry points (api, worker, cli, devops)
internal/         # Private application code
  agent/          # Agent runtime (execution, scheduling, recovery)
    agent.go      # Deprecated: legacy Agent struct; use eino.AgentFactory instead
  api/            # HTTP/gRPC API
  app/            # Application core (bootstrap, services)
  einoext/        # Cloudwego eino extensions
  ingestqueue/    # Document ingestion queue
  model/          # LLM, embedding, vision abstractions
  pipeline/       # Domain pipelines (query, specialized)
  runtime/        # Runtime core (eino workflow orchestration)
    eino/
      engine.go          # Eino Engine: workflow compilation, runner management
      agent_factory.go   # AgentFactory: config-driven Eino ADK agent creation (recommended)
      tool_bridge.go     # Tool Bridge: converts Aetheris tools → Eino InvokableTool
      workflow.go        # Workflow definition and compilation
      tools.go           # Built-in Eino tools (retriever, generator, etc.)
  splitter/       # Text splitting implementations
  storage/        # Data storage implementations
  tool/           # Tool definitions and implementations
pkg/              # Public libraries (errors moved to experimental/)
  config/         # Configuration
  log/            # Logging
  tracing/        # Tracing utilities
  experimental/   # Unused packages pending 3.0 or removal
configs/          # Configuration files
  agents.yaml    # Agent definitions (loaded by AgentFactory at startup)
examples/         # Example code
design/           # Design documentation (public in root; internal/ for implementation details)
deployments/      # Docker, K8s configurations
```

### Configuration

- Use Viper for configuration management
- YAML configuration files in `configs/`
- Support environment variable overrides
- Use `${VAR_NAME}` syntax in config for env var substitution

### Workflows and Pipelines

- All pipelines orchestrated via eino
- Workflows: DAG-based execution with nodes and edges
- Use `compose.NewGraph` for workflow definition
- Register workflows with the Engine

### Important Files

- `go.mod`: Module definition and dependencies
- `configs/*.yaml`: Configuration files
- `configs/agents.yaml`: Agent definitions (loaded by `AgentFactory` at startup)
- `internal/runtime/eino/engine.go`: Eino Engine — workflow compilation, runner management
- `internal/runtime/eino/agent_factory.go`: **AgentFactory** — config-driven Eino ADK agent creation (recommended entry point for all agent construction)
- `internal/runtime/eino/tool_bridge.go`: **Tool Bridge** — converts Aetheris `RuntimeTool` to Eino `InvokableTool` (resolves import cycle via interface abstraction)
- `internal/runtime/eino/workflow.go`: Workflow implementation
- `internal/api/http/handler.go`: HTTP handlers
- `internal/app/api/app.go`: API assembly — creates `AgentFactory`, wires tools, registers agents from config
- `internal/app/bootstrap.go`: Bootstrap and shared initialization
- `pkg/config/config.go`: Configuration types (includes `AgentDefConfig` with `Tools` field)
- `pkg/errors/errors.go`: Error utilities
- `internal/agent/agent.go`: **Deprecated** — legacy Agent struct; use `eino.AgentFactory` instead
- `internal/agent/runtime/executor.go`: Agent execution runtime (DAG compiler + runner)
- `Makefile`: Build and run commands (use `make run` to start all services)

## Common Tasks

### Adding a New Pipeline

1. Create `internal/pipeline/newpipeline/`
2. Implement `NewPipeline()` function
3. Register with Engine in `internal/app/bootstrap.go`
4. Add handler in `internal/api/http/handler.go`

### Adding a New Model Provider

1. Implement interface in `internal/model/llm/` or similar
2. Register provider in config
3. Use `NewLLMClientFromConfig` pattern

### Adding a New Agent (Config-Driven, Recommended)

All agent construction goes through `AgentFactory` using Eino ADK. The legacy `runtime.NewAgent()` is deprecated.

1. Define the agent in `configs/agents.yaml`:

```yaml
agents:
  my_agent:
    type: "react"
    description: "My custom agent"
    llm: "default"
    max_iterations: 10
    tools:                    # Optional: filter available tools; empty = all
      - "web_search"
      - "calculator"
    system_prompt: |
      You are a helpful assistant.
```

2. `AgentFactory.GetOrCreateFromConfig()` loads all agents at startup (in `internal/app/api/app.go`)
3. Access the runner via `agentFactory.GetRunner("my_agent")`
4. For programmatic creation, use `AgentFactory.CreateAgent(ctx, eino.AgentBuildConfig{...})`

### Adding a Custom Tool (via Tool Bridge)

1. Implement the `RuntimeTool` interface in your tool package:

```go
type MyTool struct{}

func (t *MyTool) Name() string            { return "my_tool" }
func (t *MyTool) Description() string      { return "Description" }
func (t *MyTool) Schema() map[string]any   { return map[string]any{...} }
func (t *MyTool) Execute(ctx context.Context, sess *session.Session, input map[string]any, state interface{}) (any, error) {
    // Tool logic here
    return "result", nil
}
```

2. Register with the tool registry: `registry.Register("my_tool", &MyTool{})`
3. The `RegistryToolBridge` automatically converts registered tools to Eino `InvokableTool`
4. `AgentFactory.collectTools()` merges registry tools + engine built-in tools for each agent
5. Optionally limit which agents see this tool via `tools:` list in `agents.yaml`

### Adding a New API Endpoint

1. Define request/response types in handler
2. Implement handler method
3. Register route in `internal/api/http/router.go`


<claude-mem-context>
# Memory Context

# [CoRag] recent context, 2026-05-26 1:07pm GMT+8

Legend: 🎯session 🔴bugfix 🟣feature 🔄refactor ✅change 🔵discovery ⚖️decision 🚨security_alert 🔐security_note
Format: ID TIME TYPE TITLE
Fetch details: get_observations([IDs]) | Search: mem-search skill

Stats: 50 obs (17,283t read) | 440,696t work | 96% savings

### May 8, 2026
2912 2:54p 🔵 Aetheris Go 框架适配器生态 — 8 种外部框架作为 TaskGraph 节点支持
2913 " 🔵 Aetheris 自定义 Agent 迁移模式 — 命令式代码→Tool+TaskGraph 标准化转换
2914 2:55p 🔵 AgentFactory 实现细节 — Checkpoint Runner 不缓存，默认 agents.yaml 定义 6 种 Agent
2916 3:03p 🔵 Aetheris 战略定位与 SDK 完整上下文确认
2917 " ⚖️ Aetheris "用户已有 Agent 接入"方向三项核心决策确立
2918 " 🔵 Aetheris SDK + NodeAdapter 执行层完整结构确认
2923 3:08p ⚖️ CoRag/Aetheris MVP 黑盒 Agent 接入策略确定
2928 3:12p 🔵 CoRag/Aetheris 黑盒 Agent 接入架构全景探索
2929 " 🔵 CoRag/Aetheris app.go 完整工具链初始化路径
2930 " 🔵 CoRag/Aetheris Runner 执行契约 — PlanGenerated 强制前置
2931 3:13p 🟣 openclaw-adapter 黑盒 HTTP Agent 接入 TDD RED 阶段 — 测试先行
2933 3:14p 🔵 CoRag 开发环境 Go 二进制路径需手动指定
2934 3:15p 🔵 TDD RED 状态确认 — external_http Agent 接入编译失败
2938 3:19p 🔵 CoRag/Aetheris TDD GREEN Phase — Session Resumed After Code Pull
2939 " 🟣 pkg/config: AgentExternalConfig + ValidateExternalAgents 实现落地
2940 " 🟣 runtime.Manager.Register() — 稳定 ID agent 注册方法新增
2941 " ✅ external_agent_tool_test.go — 返回类型断言从 map[string]any 修正为 tools.ToolResult
2943 3:21p 🟣 ExternalAgentCallTool 完整实现 — internal/app/api/external_agent_tool.go 新建
2944 " 🟣 RegisterConfiguredAgents + PlanGoalForJobFuncWithExternalAgents 实现落地
2946 3:22p 🟣 app.go 启动时完整接入 external_http agent — 工具注册、Agent 注册、Planner 路由三段全部接通
2947 3:23p 🟣 collectExternalAgentConfigs 辅助函数添加至 external_agent_tool.go
2948 " 🔴 app.go nil-safety fix — bootstrap.Config nil guard before AgentsConfig dereference
2949 " ✅ agent_dag.go loadLocalAgents — external_http case guard prevents unknown-type warning
2950 3:24p 🟣 AppendJobCompleted 携带 external agent 答案 — extractAnswerFromCommittedEvents 实现
2951 " 🟣 TDD GREEN 阶段完成 — pkg/config 和 internal/app/api 测试全部通过
2952 3:29p 🟣 external_http Agent Type — Phase 1 HTTP Blackbox Adapter
2953 " 🔴 app.go nil guard before AgentsConfig dereference
2954 " 🔴 ExternalAgentCallTool test type assertion fixed from map to tools.ToolResult
2955 " ⚖️ external_http reliability boundary: at-most-once only for outer tool call
2956 3:31p 🟣 node_sink_test.go — AppendJobCompleted answer extraction integration test
2957 3:32p 🟣 external_http full test suite GREEN — all 5 packages pass including new node_sink_test
2958 3:35p 🔵 CoRag/Aetheris working branch is main tracking origin/main
2959 4:16p 🔵 CoRag Project Review — Branch State and Directory Structure Confirmed
2960 4:17p 🔵 CoRag/Aetheris Project Structure — Multi-Language Monorepo with SLSA Release Pipeline
2961 " 🔵 CoRag/Aetheris Internal Design Documentation — Extensive Formal Spec Coverage
2962 " 🔵 Aetheris v2.3.0 Status Snapshot — Production-Ready Runtime, Integrated Compliance, Prototype Enterprise Lane
2963 " 🔵 Aetheris CI Pipeline — Go 1.26.1, 30% Coverage Threshold, Postgres Integration Tests
2964 4:18p 🔵 Aetheris Full Test Suite — All Packages Pass, Complete Package Layout Confirmed
2965 " 🔵 Local Build Environment — Go Module Cache Permission Issue and Linter Not Installed
2966 4:19p 🔵 Aetheris go.mod Dependency Stack — CloudWeGo Eino + Hertz, Full OpenTelemetry, Dual DB Support
2967 4:22p 🔵 Aetheris Technical Debt Scan — gRPC Unimplemented, Milvus/Pinecone Stubs, SAML Not Supported
2968 " 🔵 AGENTS.md Go Version Stale — Documents 1.25.7 While go.mod and CI Use 1.26.1
2969 4:24p 🔵 Aetheris Runtime Configuration — Key Environment Variables Including PLANNER_TYPE Switcher
2970 " 🔵 Go Version Documentation Drift Is Pervasive — 1.25.7 Appears in 10+ Files vs Actual 1.26.1
2971 " 🔵 Aetheris Latest Release is v2.5.3 — docs/STATUS.md Stale at v2.3.0+
2975 4:35p ✅ CoRag/Aetheris Go version requirement bumped to 1.26.1 across all docs
2976 " ✅ Milvus/Pinecone vector adapters clarified as non-production prototype placeholders
2977 " ✅ CoRag/Aetheris STATUS.md updated to v2.5.3 and CURRENT-STATUS-AND-FOCUS.md demoted to historical snapshot
2978 " 🔵 Remote branch codex/external-http-agent-intake diverged with 2 PR-review fix commits during local work
2994 5:15p ✅ hermes-agent-go v2.1.0 代码推送 GitHub 启动

Access 441k tokens of past work via get_observations([IDs]) or mem-search skill.
</claude-mem-context>