# Simple Chat Agent Example

A minimal chat agent example demonstrating conversational AI with Aetheris.

## Overview

This example shows how to build a simple conversational agent:

- Basic message handling
- Response generation
- Session management

## Prerequisites

- Go 1.25.7+
- **Cloud LLM**: Set `DASHSCOPE_API_KEY` (Qwen) or `OPENAI_API_KEY` (OpenAI)

## Usage

### With Qwen (Recommended)

```bash
export DASHSCOPE_API_KEY="your-api-key"
cd examples/simple_chat_agent
go run .
```

### With OpenAI

```bash
export OPENAI_API_KEY="your-api-key"
cd examples/simple_chat_agent
go run .
```

## Key Concepts

- Chat message format
- Basic agent responses
- Simple conversation flow
