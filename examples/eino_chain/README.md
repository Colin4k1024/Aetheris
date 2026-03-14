# Eino Chain Example

This example demonstrates how to use Chain and Workflow composition patterns with eino_examples in Aetheris.

## Overview

This example shows:
- Chain composition - sequential node execution
- Workflow composition - conditional branching
- State passing between nodes
- Integration with Aetheris executor

## Prerequisites

- Go 1.25.7+
- **Cloud LLM**: Set `DASHSCOPE_API_KEY` (Qwen) or `OPENAI_API_KEY` (OpenAI)

## Running

### With Qwen (Recommended)

```bash
export DASHSCOPE_API_KEY="your-api-key"
go run ./examples/eino_chain/main.go
```

### With OpenAI

```bash
export OPENAI_API_KEY="your-api-key"
go run ./examples/eino_chain/main.go
```

## Chain Pattern

The Chain pattern executes nodes sequentially, passing output from one node to the next:

```go
chain := eino_examples.NewChainAdapter()

chain.AddNode("step1", func(ctx context.Context, input any) (any, error) {
    // Process input
    return map[string]any{"result": "step1 done"}, nil
})

chain.AddNode("step2", func(ctx context.Context, input any) (any, error) {
    // Use output from step1
    return map[string]any{"result": "step2 done"}, nil
})

result, _ := chain.Invoke(ctx, map[string]any{"input": "test"})
```

## Workflow Pattern

The Workflow pattern supports conditional branching and flexible flow control:

```go
workflow := eino_examples.NewWorkflowAdapter()

workflow.AddNode("input", inputNode)
workflow.AddNode("process_a", processANode)
workflow.AddNode("process_b", processBNode)
workflow.AddNode("output", outputNode)
```

## Related Documentation

- [Eino Examples Adapter](../docs/adapters/eino-examples.md)
