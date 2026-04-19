# MCP Gateway

Aetheris CoRag 扩展的 MCP (Model Context Protocol) 网关，提供可复用的工具模板供开发者使用。

## 概述

本目录包含预构建的 MCP 工具实现，可与 Aetheris CoRag 代理运行时无缝集成。这些工具基于 `internal/tool` 包中的 `Tool` 接口实现，支持：

- 统一的 Schema 定义（JSON Schema 格式）
- 标准化的错误处理
- 集成到 CoRag 工具注册表
- 通过 MCP 协议暴露给 LLM 代理

## 可用工具

| 工具 | 描述 | 状态 |
|------|------|------|
| `mcp-github` | GitHub API 操作（Issues, PRs, Repos） | 模板 |
| `mcp-filesystem` | 本地文件系统操作（读/写/搜索） | 模板 |
| `mcp-web-search` | 网页搜索和内容提取 | 模板 |
| `mcp-database` | 数据库查询和操作 | 模板 |

## 快速开始

### 1. 导入工具

```go
import (
    "github.com/Colin4k1024/Aetheris/v2/tools/mcp-gateway/tools"
)
```

### 2. 注册到工具注册表

```go
// 创建工具实例
githubTool := tools.NewGitHubTool(&tools.GitHubConfig{
    Token: os.Getenv("GITHUB_TOKEN"),
})

// 注册到 CoRag 工具注册表
registry := registry.GetGlobal()
registry.Register(githubTool)
```

### 3. 通过 MCP 服务器调用

```go
// 创建 MCP 服务器
mcpServer := mcp.NewMCPServer(registry, gatekeeper)

// 列出所有可用工具
tools, _ := mcpServer.ListTools(ctx)

// 调用工具
result, _ := mcpServer.CallTool(ctx, "github.search_repos", map[string]any{
    "query": "aetheris coRag",
    "limit": 10,
})
```

## 工具接口

所有 MCP 网关工具都实现 `internal/tool.Tool` 接口：

```go
type Tool interface {
    Name() string                              // 工具名称
    Description() string                       // 工具描述
    Schema() tool.Schema                       // JSON Schema 参数定义
    Execute(ctx context.Context, input map[string]any) (ToolResult, error)
}
```

## 配置

每个工具支持通过配置文件或环境变量进行配置：

```yaml
# config.yaml
mcp:
  github:
    token: ${GITHUB_TOKEN}
    base_url: "https://api.github.com"
  
  filesystem:
    root_dir: "/data"
    allowed_paths:
      - "/data/docs"
      - "/data/uploads"
  
  web_search:
    api_key: ${SEARCH_API_KEY}
    engine: "google"
  
  database:
    dsn: "postgres://user:pass@localhost:5432/db"
```

## 错误处理

所有工具返回标准化的错误格式：

```go
type ToolResult struct {
    Content string `json:"content"` // 成功结果
    Err     string `json:"error,omitempty"` // 错误信息
}
```

错误类型包括：
- `execution_error`: 执行失败
- `timeout`: 操作超时
- `not_found`: 资源不存在
- `invalid_args`: 参数无效
- `permission_denied`: 权限不足

## 工具详情

### GitHub Tool

提供 GitHub API 访问能力：

```go
config := &GitHubConfig{
    Token:   "ghp_xxx",
    BaseURL: "https://api.github.com",
}
tool := NewGitHubTool(config)

// 可用操作:
// - search_repos: 搜索仓库
// - get_issue: 获取 Issue
// - create_issue: 创建 Issue
// - list_pulls: 列出 PRs
```

### Filesystem Tool

提供安全的文件系统操作：

```go
config := &FilesystemConfig{
    RootDir:     "/data",
    AllowedPaths: []string{"/data/docs", "/data/uploads"},
    MaxFileSize: 10 * 1024 * 1024, // 10MB
}
tool := NewFilesystemTool(config)

// 可用操作:
// - read_file: 读取文件
// - write_file: 写入文件
// - list_dir: 列出目录
// - search_files: 搜索文件
```

### Web Search Tool

提供网页搜索和内容提取：

```go
config := &WebSearchConfig{
    APIKey:  os.Getenv("SEARCH_API_KEY"),
    Engine:  "google",
    Timeout: 30 * time.Second,
}
tool := NewWebSearchTool(config)

// 可用操作:
// - search: 执行搜索
// - get_content: 提取页面内容
```

### Database Tool

提供数据库查询能力：

```go
config := &DatabaseConfig{
    DSN:      "postgres://user:pass@localhost:5432/db",
    Driver:   "postgres",
    Timeout:  30 * time.Second,
}
tool := NewDatabaseTool(config)

// 可用操作:
// - query: 执行查询
// - execute: 执行 DML/DDL
// - list_tables: 列出表
```

## 许可

本市场中的工具遵循与 Aetheris CoRag 相同的 Apache 2.0 许可证。
