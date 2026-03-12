# Eino Agent with Tools Example

This example demonstrates how to run a ReAct agent with tools using the eino_examples adapter in CoRag.

## Overview

This example shows:
- Creating a ChatModel using Ollama (local LLM)
- Defining custom tools (calculator)
- Using ReAct agent pattern with tool calling
- Integrating with CoRag executor

## Prerequisites

- Go 1.26+
- Ollama running locally (e.g., `ollama serve` and `ollama pull llama3`)

## Running

```bash
# Set environment variables (optional, defaults are provided)
export OLLAMA_MODEL=llama3
export OLLAMA_BASE_URL=http://localhost:11434

# Run the example
go run ./examples/eino_agent_with_tools/main.go
```

## Code Structure

```go
// 1. Create LLM client
client, _ := llm.NewOllamaClient("llama3", "http://localhost:11434")
model := eino_examples.NewOllamaChatModel(client)

// 2. Define tools
calculatorTool := tool.NewTool(
    "calculator",
    "Performs basic arithmetic operations",
    CalculatorTool,
)

// 3. Create ReAct agent adapter
adapter := eino_examples.NewReactAgentAdapter(model, []tool.InvokableTool{calculatorTool})

// 4. Execute
result, err := adapter.Invoke(ctx, map[string]any{
    "prompt": "Calculate 15 + 27",
})
```

## Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `OLLAMA_MODEL` | Ollama model name | `llama3` |
| `OLLAMA_BASE_URL` | Ollama API endpoint | `http://localhost:11434` |

## Related Documentation

- [Eino Examples Adapter](../docs/adapters/eino-examples.md)
- [Ollama Documentation](https://github.com/ollama/ollama)
