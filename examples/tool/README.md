# Tool Example

This example demonstrates how to register and use custom tools with Aetheris agents.

## Overview

Learn how to extend agent capabilities with custom tools:

- Tool registration
- Tool implementation
- Tool result handling

## Prerequisites

- Go 1.25.7+
- **Cloud LLM**: Set `DASHSCOPE_API_KEY` (Qwen) or `OPENAI_API_KEY` (OpenAI)

## Usage

### With Qwen (Recommended)

```bash
export DASHSCOPE_API_KEY="your-api-key"
cd examples/tool
go run .
```

### With OpenAI

```bash
export OPENAI_API_KEY="your-api-key"
cd examples/tool
go run .
```

## Key Concepts

- Custom tool creation
- Tool interface implementation
- Tool result format
