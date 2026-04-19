# eino × Aetheris Integration Story

> Aetheris uses [cloudwego/eino](https://github.com/cloudwego/eino) as its agent model adapter layer.
> This document explains how eino agents run on Aetheris for durable, production-grade execution.

## Overview

[eino](https://github.com/cloudwego/eino) is ByteDance's Go agent framework for building LLM applications.
Aetheris complements eino by providing the **execution runtime** — crash recovery, at-most-once tool execution,
and full audit trails that eino itself does not provide.

```
eino:  Build agents (prompting, tool calling, memory, chains)
Aetheris: Run agents (durability, reliability, observability)
```

## Quick Start

```bash
go get github.com/cloudwego/eino@latest
go get github.com/Colin4k1024/Aetheris/v2
```

```go
package main

import (
    "context"
    "fmt"
    "os"

    "github.com/cloudwego/eino-ext/components/model/dashscope"
    "github.com/cloudwego/eino/adk"
    "github.com/Colin4k1024/Aetheris/v2/internal/agent/runtime"
)

func main() {
    ctx := context.Background()

    // 1. Create eino agent (your existing eino code)
    chatModel, _ := dashscope.NewChatModel(ctx, &dashscope.ChatModelConfig{
        Model:  "qwen-plus",
        APIKey: os.Getenv("DASHSCOPE_API_KEY"),
    })
    agent, _ := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{Model: chatModel})

    // 2. Wrap with Aetheris runtime (adds durability)
    rt, _ := runtime.New(ctx, &runtime.Config{
        JobStore: "postgres",
        DSN:      os.Getenv("DATABASE_URL"),
        Effects:  true,  // At-Most-Once tool calls
    })

    // 3. Submit jobs — now crash-proof
    job, _ := rt.Submit(ctx, agent.Input{
        System:  "You are a code review assistant",
        Message: "Review this Go code for bugs...",
    })
    fmt.Printf("Job %s submitted, status: %s\n", job.ID, job.Status)
}
```

## Key Benefits

| eino alone | eino + Aetheris |
|------------|-----------------|
| Worker crash → job starts over | Worker crash → resumes from checkpoint |
| Tool timeout → retry blindly | Tool timeout → Effects Ledger prevents duplicates |
| No audit trail | Full Event Sourcing trace |
| Single worker | Multi-worker with lease fencing |

## Effects Ledger with eino Tools

For eino tools that must not execute twice:

```go
// Register tool with Aetheris Effects
rt.RegisterTool("payment_tool", paymentTool,
    runtime.WithIdempotencyKey(func(ctx context.Context, input any) string {
        return "payment-" + orderID // idempotency key
    }),
)

// On crash + retry, Aetheris skips re-execution
// Only one actual payment API call is made
```

## MCP Gateway for eino

Aetheris MCP Gateway provides pre-built tools for eino agents:

```go
import "github.com/Colin4k1024/Aetheris/v2/tools/mcp-gateway/tools/mcp-github"

githubTool := mcpgithub.NewGitHubTool(&mcpgithub.GitHubConfig{Token: token})
agent, _ := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
    Model: chatModel,
    Tools: []tool.BaseTool{githubTool},  // eino uses Aetheris MCP tools
})
```

## For the eino Community

- eino GitHub: https://github.com/cloudwego/eino
- Aetheris GitHub: https://github.com/Colin4k1024/Aetheris
- eino Discord: https://discord.gg/cloudwego (or check eino readme)

We welcome contributions and integration stories from the eino community!

---

*Submitted as an eino integration showcase. See full example in `examples/eino_agent_with_tools/`.*
