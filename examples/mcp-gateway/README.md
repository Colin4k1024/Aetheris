# MCP Gateway Example

 Demonstrates how to use Aetheris MCP Gateway tools with the agent runtime.

## Tools Available

| Tool | Description | Status |
|------|-------------|--------|
| `mcp-github` | GitHub API (search repos, issues, PRs, files) | Template |
| `mcp-filesystem` | Secure local file operations | Template |
| `mcp-web-search` | Web search and content extraction | Template |
| `mcp-database` | Database query (PostgreSQL, MySQL) | Template |

## Quick Start

```bash
# Run the example
go run ./examples/mcp-gateway

# Or build and run
go build -o bin/mcp-example ./examples/mcp-gateway
./bin/mcp-example
```

## Prerequisites

```bash
export GITHUB_TOKEN=ghp_your_token_here
```

## Example: Using GitHub MCP Tool

```go
package main

import (
    "context"
    "fmt"
    "os"

    "github.com/Colin4k1024/Aetheris/v2/tools/mcp-gateway/tools"
    "github.com/Colin4k1024/Aetheris/v2/tools/mcp-gateway/tools/mcp-github"
)

func main() {
    ctx := context.Background()

    // 1. Create GitHub tool
    githubTool := mcpgithub.NewGitHubTool(&mcpgithub.GitHubConfig{
        Token: os.Getenv("GITHUB_TOKEN"),
    })

    // 2. List available actions
    fmt.Println("Schema:", githubTool.Schema())

    // 3. Search repositories
    result, err := githubTool.Execute(ctx, map[string]any{
        "action": "search_repos",
        "query":  "aetheris agent runtime",
        "limit":  5,
    })
    if err != nil {
        fmt.Printf("Error: %v\n", err)
        return
    }
    fmt.Println("Result:", result)
}
```

## MCP Server Integration

For full MCP protocol support (stdio/HTTP+SSE transport):

```go
import (
    "github.com/Colin4k1024/Aetheris/v2/internal/tool/mcp"
    "github.com/Colin4k1024/Aetheris/v2/internal/tool/gatekeeper"
    "github.com/Colin4k1024/Aetheris/v2/internal/tool/registry"
)

// Create registry and register tools
reg := registry.New()
reg.Register(mcpgithub.NewGitHubTool(&mcpgithub.GitHubConfig{
    Token: os.Getenv("GITHUB_TOKEN"),
}))

// Create gatekeeper with security config
gk := gatekeeper.New(
    gatekeeper.WithAllowedHosts([]string{
        "api.github.com",
        "api.githubcopilot.com",
    }),
    gatekeeper.WithTypeValidation(true),
)

// Create MCP server
server := mcp.NewMCPServer(reg, gk)

// List tools via MCP protocol
tools, _ := server.ListTools(ctx)
for _, t := range tools {
    fmt.Println(t.Name, "-", t.Description)
}

// Call tool via MCP protocol
result, _ := server.CallTool(ctx, "mcp-github", map[string]any{
    "action": "search_repos",
    "query":  "agent runtime golang",
    "limit":  3,
})
fmt.Println(result)
```

## See Also

- [MCP Implementation Guide](../../docs/mcp/implementation.md)
- [MCP Security Guide](../../docs/mcp/security-guide.md)
- [Tools Registry](../../tools/mcp-gateway/registry.yaml)
