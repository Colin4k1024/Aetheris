# Plan-Execute Agent Example

This example demonstrates the Plan-Execute Agent pattern in eino.

## Overview

Plan-Execute Agent is a two-phase agent pattern:
1. **Plan Phase**: Analyze the task and create an execution plan
2. **Execute Phase**: Execute tasks step by step according to the plan

## Architecture

```
                        User Query
                           │
                           ▼
┌─────────────────────────────────────────────────────────────┐
│  ┌─────────────────────────────────────────────────────┐   │
│  │                   PLAN 阶段                          │   │
│  │  1. Analyze task goal                               │   │
│  │  2. Create execution plan                           │   │
│  │  3. Evaluate plan feasibility                      │   │
│  └─────────────────────────────────────────────────────┘   │
└─────────────────────────┬───────────────────────────────────┘
                          │
                          ▼
┌─────────────────────────────────────────────────────────────┐
│  ┌─────────────────────────────────────────────────────┐   │
│  │                 EXECUTE 阶段                         │   │
│  │  Loop:                                            │   │
│  │   1. Execute current step                         │   │
│  │   2. Check result                                 │   │
│  │   3. Adjust plan if needed                        │   │
│  │   4. Continue to next step                        │   │
│  └─────────────────────────────────────────────────────┘   │
└─────────────────────────┬───────────────────────────────────┘
                          │
                          ▼
                   Final Result
```

## When to Use Plan-Execute Agent

- Complex multi-step tasks
- Tasks where planning is important before execution
- Tasks that may need to adapt based on results
- When you need visibility into the execution steps

## Advantages

- **Think before act**: Plan first to ensure correct direction
- **Dynamic adjustment**: Adapt plan based on execution results
- **Traceable**: Each step has clear status and results
- **Suitable for complex tasks**: Multi-step tasks especially benefit
- **Intervention**: Users can review and intervene in the plan

## Running

```bash
go run ./examples/plan_execute_agent/main.go
```

## Related Documentation

- [Eino ADK](https://www.cloudwego.io/zh/docs/eino/core_modules/eino_adk/)
- [Plan-Execute Agent](https://www.cloudwego.io/zh/docs/eino/core_modules/eino_adk/eino_adk_agents/)
