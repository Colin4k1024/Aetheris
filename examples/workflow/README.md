# Workflow Example

This example demonstrates how to build workflows with Aetheris.

## Overview

Learn how to create multi-step workflows:

- Workflow definition
- Step orchestration
- DAG-based execution

## Prerequisites

- Go 1.25.7+
- **Cloud LLM**: Set `DASHSCOPE_API_KEY` (Qwen) or `OPENAI_API_KEY` (OpenAI)

## Usage

### With Qwen (Recommended)

```bash
export DASHSCOPE_API_KEY="your-api-key"
cd examples/workflow
go run .
```

### With OpenAI

```bash
export OPENAI_API_KEY="your-api-key"
cd examples/workflow
go run .
```

## Key Concepts

- Workflow structure
- Step dependencies
- DAG execution
