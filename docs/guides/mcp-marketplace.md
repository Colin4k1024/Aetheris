# MCP Tool Marketplace

> Discover and integrate pre-built MCP tools into your Aetheris agent runtime.

## Available Tools

### 🔌 GitHub Integration

**`mcp-github`** — Interact with GitHub repositories, issues, and pull requests.

```go
githubTool := mcpgithub.NewGitHubTool(&mcpgithub.GitHubConfig{
    Token: os.Getenv("GITHUB_TOKEN"),
})
```

**Capabilities:**
- `search_repos` — Search repositories by keyword
- `get_issue` / `create_issue` — Read and create GitHub issues
- `list_pulls` — List pull requests with filtering
- `get_file` / `create_pr` — File operations and PR creation

**Install:**
```bash
go get github.com/Colin4k1024/Aetheris/v2/tools/mcp-gateway/tools/mcp-github
```

---

### 📁 Filesystem

**`mcp-filesystem`** — Secure local file operations with path traversal protection.

```go
fsTool := mcpfs.NewFilesystemTool(&mcpfs.FilesystemConfig{
    RootDir:     "/data",
    AllowedPaths: []string{"/data/docs", "/data/uploads"},
    MaxFileSize: 10 * 1024 * 1024,
})
```

**Capabilities:**
- `read_file` / `write_file` — Safe file I/O
- `list_dir` — Directory listing
- `search_files` — Full-text file search

**Install:**
```bash
go get github.com/Colin4k1024/Aetheris/v2/tools/mcp-gateway/tools/mcp-filesystem
```

---

### 🔍 Web Search

**`mcp-web-search`** — Search the web and extract page content.

```go
searchTool := mcpsearch.NewWebSearchTool(&mcpsearch.WebSearchConfig{
    APIKey:  os.Getenv("SEARCH_API_KEY"),
    Engine:  "google",
    Timeout: 30 * time.Second,
})
```

**Capabilities:**
- `search` — Execute web search across multiple engines
- `get_content` — Extract and parse page content

**Install:**
```bash
go get github.com/Colin4k1024/Aetheris/v2/tools/mcp-gateway/tools/mcp-web-search
```

---

### 🗄️ Database

**`mcp-database`** — Query PostgreSQL, MySQL and other databases with parameterized queries.

```go
dbTool := mcpdb.NewDatabaseTool(&mcpdb.DatabaseConfig{
    DSN:     "postgres://user:pass@localhost:5432/db",
    Driver:  "postgres",
    Timeout: 30 * time.Second,
})
```

**Capabilities:**
- `query` — Execute parameterized SELECT queries
- `execute` — Run DML/DDL statements
- `list_tables` — Discover database schema

**Install:**
```bash
go get github.com/Colin4k1024/Aetheris/v2/tools/mcp-gateway/tools/mcp-database
```

---

## Quick Start

### 1. Register tools

```go
import (
    mcpgithub "github.com/Colin4k1024/Aetheris/v2/tools/mcp-gateway/tools/mcp-github"
    "github.com/Colin4k1024/Aetheris/v2/internal/tool/registry"
)

reg := registry.GetGlobal()
reg.Register(mcpgithub.NewGitHubTool(&mcpgithub.GitHubConfig{
    Token: os.Getenv("GITHUB_TOKEN"),
}))
```

### 2. Call via MCP protocol

```go
import (
    "github.com/Colin4k1024/Aetheris/v2/internal/tool/mcp"
    "github.com/Colin4k1024/Aetheris/v2/internal/tool/gatekeeper"
)

gk := gatekeeper.New(
    gatekeeper.WithAllowedHosts([]string{"api.github.com"}),
    gatekeeper.WithTypeValidation(true),
)

server := mcp.NewMCPServer(reg, gk)
result, _ := server.CallTool(ctx, "mcp-github", map[string]any{
    "action": "search_repos",
    "query":  "aetheris agent runtime golang",
    "limit":  5,
})
```

### 3. Or use directly

```go
// Direct tool execution (no MCP protocol overhead)
result, err := githubTool.Execute(ctx, map[string]any{
    "action": "search_repos",
    "query":  "agent runtime golang",
    "limit":  5,
})
```

## Registry File

See [registry.yaml](../../tools/mcp-gateway/registry.yaml) for full tool清单 and configuration schemas.

## Security

All tools are validated through the Gatekeeper before execution:

- **Network**: Allowed hosts must be explicitly configured
- **Filesystem**: Path traversal attacks prevented via allowlist
- **Database**: Parameterized queries prevent SQL injection
- **Rate Limiting**: Per-tool rate limits configurable

See [Security Guide](../mcp/security-guide.md) for production configuration.

## Contributing a New Tool

1. Create `tools/mcp-gateway/tools/mcp-yourtool/`
2. Implement the `tool.Tool` interface
3. Add `manifest.yaml` with metadata
4. Register in [registry.yaml](../../tools/mcp-gateway/registry.yaml)
5. Add example usage to this page

See [CONTRIBUTING.md](../../tools/mcp-gateway/CONTRIBUTING.md) for full guide.

## See Also

- [MCP Integration Guide](../mcp/integration.md)
- [MCP Implementation](../mcp/implementation.md)
- [OpenAPI Spec](../../tools/mcp-gateway/openapi.yaml)
