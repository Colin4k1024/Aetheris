# SDK Agent Example

A comprehensive example demonstrating the Aetheris SDK for building agents.

## Overview

This example showcases the full SDK capabilities:

- High-level Agent API
- Tool registration
- Runtime context usage
- Comparison with Job/Runner patterns

## Prerequisites

- Go 1.25.7+
- **Cloud LLM**: Set `DASHSCOPE_API_KEY` (Qwen) or `OPENAI_API_KEY` (OpenAI)

## Usage

### With Qwen (Recommended)

```bash
export DASHSCOPE_API_KEY="your-api-key"
cd examples/sdk_agent
go run .
```

### With OpenAI

```bash
export OPENAI_API_KEY="your-api-key"
cd examples/sdk_agent
go run .
```

## Key Concepts

- `NewAgent()` API
- `RegisterTool()` for custom tools
- `Run()` for execution
- SDK vs Job/Runner comparison
