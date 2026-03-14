# Streaming Example

This example demonstrates how to use streaming responses with Aetheris agents.

## Overview

Learn how to handle streaming outputs from LLMs:

- Real-time response streaming
- Chunk processing
- Stream handling patterns

## Prerequisites

- Go 1.25.7+
- **Cloud LLM**: Set `DASHSCOPE_API_KEY` (Qwen) or `OPENAI_API_KEY` (OpenAI)

## Usage

### With Qwen (Recommended)

```bash
export DASHSCOPE_API_KEY="your-api-key"
cd examples/streaming
go run .
```

### With OpenAI

```bash
export OPENAI_API_KEY="your-api-key"
cd examples/streaming
go run .
```

## Key Concepts

- Stream configuration
- Chunk processing
- Real-time output handling
