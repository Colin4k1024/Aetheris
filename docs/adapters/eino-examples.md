# Eino Examples Adapter

This document describes the eino_examples integration in CoRag, which provides adapters for [cloudwego/eino-examples](https://github.com/cloudwego/eino-examples) patterns.

## Overview

The `internal/agent/runtime/executor/eino_examples` package provides adapters for various eino agent and workflow patterns:

| Pattern | Adapter | Description |
|---------|---------|-------------|
| **ReAct Agent** | `ReactAgentAdapter` | Reasoning + Action pattern with tool usage |
| **DEER-Go Agent** | `DEERAgentAdapter` | DEER-Go agent pattern |
| **Manus Agent** | `ManusAgentAdapter` | Manus agent pattern |
| **ADK** | `ADKAdapter` | Google ADK agent pattern |
| **Chain** | `ChainAdapter` | Sequential pipeline composition |
| **Graph** | `GraphAdapter` | DAG-based composition with edges |
| **Workflow** | `WorkflowAdapter` | Flexible workflow composition |

## Installation

The eino_examples package is included in the CoRag module. Ensure you have the dependencies:

```bash
go mod tidy
```

## Quick Start

### 1. Create a ChatModel

```go
import (
    eino_examples "rag-platform/internal/agent/runtime/executor/eino_examples"
    "rag-platform/internal/model/llm"
)

// Using Ollama (local LLM)
client, _ := llm.NewOllamaClient("llama3", "http://localhost:11434")
model := eino_examples.NewOllamaChatModel(client)

// Using OpenAI
client, _ := llm.NewClient("openai", "gpt-4", apiKey, "")
model := eino_examples.NewOpenAIChatModel(client)

// Using Claude
client, _ := llm.NewClient("claude", "claude-3-haiku-20240307", apiKey, "")
model := eino_examples.NewClaudeChatModel(client)
```

### 2. Create Agent Adapter

```go
// ReAct Agent with tools
adapter := eino_examples.NewReactAgentAdapter(model, tools,
    eino_examples.WithTemperature(0.7),
    eino_examples.WithMaxTokens(2048),
)

// DEER-Go Agent
deerAdapter := eino_examples.NewDEERAgentAdapter(model, tools)

// Manus Agent
manusAdapter := eino_examples.NewManusAgentAdapter(model, tools)
```

### 3. Execute

```go
ctx := context.Background()

result, err := adapter.Invoke(ctx, map[string]any{
    "prompt": "Calculate 15 + 27 using the calculator tool",
})
if err != nil {
    log.Fatal(err)
}

fmt.Println(result["response"])
```

## Examples

### ReAct Agent with Tools

See `examples/eino_agent_with_tools/` for a complete example:

```bash
go run ./examples/eino_agent_with_tools/main.go
```

This example demonstrates:
- Creating a calculator tool
- Using ReAct agent with tool calling
- Integrating with CoRag executor

### Chain Composition

```go
chain := eino_examples.NewChainAdapter()

chain.AddNode("step1", func(ctx context.Context, input any) (any, error) {
    return map[string]any{"result": "step1 done"}, nil
})

chain.AddNode("step2", func(ctx context.Context, input any) (any, error) {
    return map[string]any{"result": "step2 done"}, nil
})

result, _ := chain.Invoke(ctx, map[string]any{"input": "test"})
```

### Graph Composition

```go
graph := eino_examples.NewGraphAdapter()

graph.AddNode("input", inputNode)
graph.AddNode("process", processNode)
graph.AddNode("output", outputNode)

graph.AddEdge("input", "process")
graph.AddEdge("process", "output")
graph.SetEntry("input")

result, _ := graph.Invoke(ctx, map[string]any{"data": "test"})
```

## LLM Provider Configuration

### Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `OLLAMA_MODEL` | Ollama model name | `llama3` |
| `OLLAMA_BASE_URL` | Ollama API endpoint | `http://localhost:11434` |
| `OPENAI_API_KEY` | OpenAI API key | - |
| `OPENAI_MODEL` | OpenAI model name | `gpt-3.5-turbo` |
| `ANTHROPIC_API_KEY` | Anthropic Claude API key | - |

### Using Docker Compose

In `docker-compose.v2.yml`, configure the LLM provider:

```yaml
environment:
  - MODEL_DEFAULTS_LLM=openai.gpt_35_turbo
  - MODEL_LLM_PROVIDERS_OPENAI_BASE_URL=http://host.docker.internal:11434/v1
  - MODEL_LLM_PROVIDERS_OPENAI_API_KEY=ollama
  - MODEL_LLM_PROVIDERS_OPENAI_MODELS_GPT_35_TURBO_NAME=llama3
```

## Token Metrics

The eino_examples adapters automatically track token usage:

```go
// Tokens are recorded via Prometheus metrics
metrics.LLMTokensTotal.WithLabelValues("input").Add(float64(inputTokens))
metrics.LLMTokensTotal.WithLabelValues("output").Add(float64(outputTokens))
```

View metrics at: `http://localhost:9093/metrics` (Worker metrics port)

## API Reference

### ChatModel Interface

```go
type ChatModel interface {
    Generate(ctx context.Context, input []*schema.Message, opts ...Option) (*schema.Message, error)
    Stream(ctx context.Context, input []*schema.Message, opts ...Option) (*schema.StreamReader[*schema.Message], error)
}
```

### EinoExampleAdapter Interface

```go
type EinoExampleAdapter interface {
    Invoke(ctx context.Context, input map[string]any) (map[string]any, error)
    Stream(ctx context.Context, input map[string]any, onChunk func(chunk map[string]any) error) error
    GetState(ctx context.Context) (map[string]any, error)
}
```

### Options

| Option | Description |
|--------|-------------|
| `WithTemperature(t float64)` | Set LLM temperature (0.0-2.0) |
| `WithMaxTokens(n int)` | Set max tokens for response |

## Related Documentation

- [Cloudwego Eino](https://github.com/cloudwego/eino) - Eino framework
- [Eino Examples](https://github.com/cloudwego/eino-examples) - Example agents and workflows
- [Config Reference](../reference/config.md) - LLM configuration
- [Usage Guide](../guides/usage.md) - General usage patterns
