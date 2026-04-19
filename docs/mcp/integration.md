# MCP Gateway 集成指南

> 本指南帮助你在 Aetheris 项目中集成并使用 MCP Gateway 工具。

## 概述

MCP Gateway 提供了四类可复用的 MCP 工具，可直接注册到 Aetheris 运行时：

| 工具 | 包路径 | 用途 |
|------|--------|------|
| `mcp-github` | `tools/mcp-gateway/tools/mcp-github` | GitHub API 操作 |
| `mcp-filesystem` | `tools/mcp-gateway/tools/mcp-filesystem` | 本地文件系统安全操作 |
| `mcp-web-search` | `tools/mcp-gateway/tools/mcp-web-search` | 网页搜索与内容提取 |
| `mcp-database` | `tools/mcp-gateway/tools/mcp-database` | 数据库查询 |

## 快速集成

### Step 1: 安装依赖

```bash
go get github.com/Colin4k1024/Aetheris/v2/tools/mcp-gateway/tools/mcp-github
go get github.com/google/go-github/v45
```

### Step 2: 注册工具到 Registry

```go
import (
    mcpgithub "github.com/Colin4k1024/Aetheris/v2/tools/mcp-gateway/tools/mcp-github"
    "github.com/Colin4k1024/Aetheris/v2/internal/tool/registry"
)

reg := registry.GetGlobal()

// 注册 GitHub MCP 工具
githubTool := mcpgithub.NewGitHubTool(&mcpgithub.GitHubConfig{
    Token: os.Getenv("GITHUB_TOKEN"),
})
reg.Register(githubTool)
```

### Step 3: 通过 MCP 服务器调用

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

// 列出所有可用工具
tools, _ := server.ListTools(ctx)
for _, t := range tools {
    fmt.Println(t.Name, "-", t.Description)
}

// 通过 MCP 协议调用工具
result, _ := server.CallTool(ctx, "mcp-github", map[string]any{
    "action": "search_repos",
    "query":  "agent runtime golang",
    "limit":  5,
})
fmt.Println(result)
```

## MCP Gateway 工具详解

### GitHub Tool (`mcp-github`)

```go
githubTool := mcpgithub.NewGitHubTool(&mcpgithub.GitHubConfig{
    Token:   os.Getenv("GITHUB_TOKEN"),
    BaseURL: "https://api.github.com",  // 可选：GitHub Enterprise
    Timeout: 30 * time.Second,
})
```

**可用 action**:

| action | 参数 | 说明 |
|--------|------|------|
| `search_repos` | `query`, `limit` | 搜索仓库 |
| `get_issue` | `owner`, `repo`, `issue_number` | 获取 Issue |
| `create_issue` | `owner`, `repo`, `title`, `body` | 创建 Issue |
| `list_pulls` | `owner`, `repo`, `state`, `limit` | 列出 PRs |
| `get_file` | `owner`, `repo`, `path`, `ref` | 获取文件内容 |
| `create_pr` | `owner`, `repo`, `title`, `body`, `head`, `base` | 创建 PR |

**示例**:

```go
// 搜索仓库
result, _ := githubTool.Execute(ctx, map[string]any{
    "action": "search_repos",
    "query":  "aetheris agent runtime",
    "limit":  10,
})

// 创建 Issue
result, _ := githubTool.Execute(ctx, map[string]any{
    "action":      "create_issue",
    "owner":       "Colin4k1024",
    "repo":        "Aetheris",
    "title":       "Bug: tool fails on large files",
    "body":        "Steps to reproduce...",
})
```

### Filesystem Tool (`mcp-filesystem`)

```go
import mcpfs "github.com/Colin4k1024/Aetheris/v2/tools/mcp-gateway/tools/mcp-filesystem"

fsTool := mcpfs.NewFilesystemTool(&mcpfs.FilesystemConfig{
    RootDir:     "/data",
    AllowedPaths: []string{"/data/docs", "/data/uploads"},
    MaxFileSize: 10 * 1024 * 1024, // 10MB
})
```

**可用 action**:

| action | 参数 | 说明 |
|--------|------|------|
| `read_file` | `path` | 读取文件 |
| `write_file` | `path`, `content` | 写入文件 |
| `list_dir` | `path` | 列出目录 |
| `search_files` | `query`, `path` | 搜索文件 |

### Web Search Tool (`mcp-web-search`)

```go
import mcpsearch "github.com/Colin4k1024/Aetheris/v2/tools/mcp-gateway/tools/mcp-web-search"

searchTool := mcpsearch.NewWebSearchTool(&mcpsearch.WebSearchConfig{
    APIKey:  os.Getenv("SEARCH_API_KEY"),
    Engine:  "google",
    Timeout: 30 * time.Second,
})
```

### Database Tool (`mcp-database`)

```go
import mcpdb "github.com/Colin4k1024/Aetheris/v2/tools/mcp-gateway/tools/mcp-database"

dbTool := mcpdb.NewDatabaseTool(&mcpdb.DatabaseConfig{
    DSN:     "postgres://user:pass@localhost:5432/db",
    Driver:  "postgres",
    Timeout: 30 * time.Second,
})
```

## 通过 agents.yaml 配置

在 `configs/agents.yaml` 中声明 MCP 工具：

```yaml
agents:
  github_agent:
    type: "react"
    llm: "default"
    tools:
      - "mcp-github"
    system_prompt: |
      你是一个 GitHub 助手，可以搜索仓库、查看 Issue 和创建 PR。
```

## MCP 服务器传输模式

### Stdio 模式（本地）

适用于 CLI 工具和本地开发：

```bash
aetheris mcp-server --transport stdio
```

### HTTP + SSE 模式（生产）

适用于 Web 服务：

```go
server := mcp.NewMCPServer(reg, gk)
// 集成到 HTTP 路由
http.HandleFunc("/mcp", func(w http.ResponseWriter, r *http.Request) {
    server.HandleHTTP(w, r)
})
```

## 最佳实践

1. **始终配置 Gatekeeper**：限制允许的 hosts，防止 SSRF 攻击
2. **使用环境变量存储 Token**：不要硬编码敏感信息
3. **设置合理的 Timeout**：避免工具调用阻塞
4. **错误处理**：检查 `ToolResult.Err` 字段

## See Also

- [MCP Implementation Guide](implementation.md) — MCP 协议内部实现
- [MCP Security Guide](security-guide.md) — 安全配置
- [Tools Registry](../../tools/mcp-gateway/registry.yaml) — 完整工具清单
- [OpenAPI Spec](../../tools/mcp-gateway/openapi.yaml) — MCP Gateway API 规范
