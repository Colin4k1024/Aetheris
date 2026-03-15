# Supervisor Agent Example

This example demonstrates the Supervisor Agent pattern in eino.

## Overview

Supervisor Agent is a multi-agent architecture where a supervisory agent is responsible for:
1. Understanding user requests
2. Decomposing tasks into sub-tasks
3. Delegating sub-tasks to specialized sub-agents
4. Collecting results from sub-agents
5. Synthesizing results and returning to user

This pattern is similar to management in a company - the Supervisor coordinates specialist agents.

## Architecture

```
                        User Query
                           │
                           ▼
              ┌────────────────────────┐
              │   Supervisor Agent    │
              │  ┌──────────────────┐ │
              │  │ 1. Analyze       │ │
              │  │ 2. Decide        │ │
              │  │ 3. Delegate      │ │
              │  │ 4. Collect       │ │
              │  │ 5. Synthesize    │ │
              │  └──────────────────┘ │
              └──────────┬───────────┘
                         │
        ┌────────────────┼────────────────┐
        │                │                │
   ┌────▼────┐      ┌────▼────┐     ┌────▼────┐
   │Research │      │ Writer  │     │  Coder  │
   │ Agent   │      │ Agent   │     │ Agent   │
   └─────────┘      └─────────┘     └─────────┘
```

## When to Use Supervisor Agent

- Complex tasks that require multiple skills
- Tasks that can be parallelized
- When you need clear separation of concerns
- When you want to add/remove capabilities easily

## Running

```bash
go run ./examples/supervisor_agent/main.go
```

## Key Components

1. **Sub-Agents**: Specialized agents for different tasks (researcher, writer, coder)
2. **Delegate Tool**: Mechanism to delegate tasks to sub-agents
3. **Supervisor**: Coordinates the workflow and synthesizes results

## Related Documentation

- [Eino ADK](https://www.cloudwego.io/zh/docs/eino/core_modules/eino_adk/)
- [Supervisor Agent](https://www.cloudwego.io/zh/docs/eino/core_modules/eino_adk/eino_adk_agents_supervisor/)
