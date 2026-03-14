# Basic Agent Example

A simple example demonstrating how to create and run a basic agent using the Aetheris SDK.

## Overview

This example shows the fundamental patterns for building agents with Aetheris:

- Creating an agent
- Registering tools
- Running the agent

## Prerequisites

- Go 1.25.7+
- **Cloud LLM**: Set `DASHSCOPE_API_KEY` (Qwen) or `OPENAI_API_KEY` (OpenAI)

## Usage

### With Qwen (Recommended)

```bash
export DASHSCOPE_API_KEY="your-api-key"
cd examples/basic_agent
go run .
```

### With OpenAI

```bash
export OPENAI_API_KEY="your-api-key"
cd examples/basic_agent
go run .
```

## Key Concepts

- Agent creation with `NewAgent`
- Tool registration
- Basic execution flow
