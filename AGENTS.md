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
