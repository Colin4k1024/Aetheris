# Eino Agent with Tools Example

This example demonstrates how to run a ReAct agent with tools using the eino_examples adapter in Aetheris.

## Overview

This example shows:
- Creating a ChatModel using cloud LLM (Qwen/OpenAI) or local Ollama
- Defining custom tools (calculator, search, weather)
- Using ReAct agent pattern with tool calling
- Integrating with Aetheris executor

## Prerequisites

- Go 1.25.7+
- **Cloud LLM (Recommended)**: Set `DASHSCOPE_API_KEY` for Qwen or `OPENAI_API_KEY` for OpenAI
- **Local LLM (Optional)**: Ollama running locally (e.g., `ollama serve` and `ollama pull qwen3:30b`)

> **Note**: Some Ollama models (e.g., llama3) do not support tool calling. Use Qwen or OpenAI for production.

## Running

### With Qwen (Recommended)

```bash
# Set your API key
export DASHSCOPE_API_KEY="your-api-key"

# Run the example
go run ./examples/eino_agent_with_tools/main.go
```

### With OpenAI

```bash
export OPENAI_API_KEY="your-api-key"
go run ./examples/eino_agent_with_tools/main.go
```

### With Ollama

```bash
export OLLAMA_MODEL=qwen3:30b
export OLLAMA_BASE_URL=http://localhost:11434
go run ./examples/eino_agent_with_tools/main.go
```

## Code Structure

```go
// 1. Create LLM client (Qwen/OpenAI/Ollama)
client, _ := llm.NewOpenAIClient("qwen3-max", "your-api-key")
model := eino_examples.NewOpenAIChatModel(client)

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
| `DASHSCOPE_API_KEY` | Qwen API key (recommended) | - |
| `OPENAI_API_KEY` | OpenAI API key | - |
| `OLLAMA_MODEL` | Ollama model name | `qwen3:30b` |
| `OLLAMA_BASE_URL` | Ollama API endpoint | `http://localhost:11434` |

## Related Documentation

- [Eino Examples Adapter](../docs/adapters/eino-examples.md)
- [Model Configuration](../configs/model.yaml)
- [Qwen Documentation](https://dashscope.aliyuncs.com/)
- [OpenAI Documentation](https://platform.openai.com/)
- [Ollama Documentation](https://github.com/ollama/ollama)
