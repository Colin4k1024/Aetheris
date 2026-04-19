# MCP 协议实现指南

## 什么是 MCP

MCP (Model Context Protocol) 是一个用于 AI 模型与外部工具/资源交互的标准协议。

## CoRag MCP 实现

### 目录结构

```
internal/tool/
├── mcp/
│   ├── server.go      # MCP 服务器实现
│   ├── plugin.go       # 插件支持
│   └── ...
├── gatekeeper/
│   ├── gatekeeper.go   # 参数验证和安全检查
│   └── gatekeeper_test.go
├── types/
│   └── errors.go       # 错误类型定义
└── descriptor.go       # 工具描述符（强类型 Schema）
```

### 快速开始

#### 1. 创建 MCP 服务器

```go
import (
    "github.com/Colin4k1024/Aetheris/v2/internal/tool"
    "github.com/Colin4k1024/Aetheris/v2/internal/tool/mcp"
    "github.com/Colin4k1024/Aetheris/v2/internal/tool/gatekeeper"
    "github.com/Colin4k1024/Aetheris/v2/internal/tool/registry"
)

// 创建工具注册表
reg := registry.New()

// 注册工具
builtin.RegisterBuiltin(reg, engine, generator)

// 创建 Gatekeeper（参数验证）
gk := gatekeeper.New(
    gatekeeper.WithAllowedHosts([]string{"api.example.com"}),
    gatekeeper.WithTypeValidation(true),
)

// 创建 MCP 服务器
server := mcp.NewMCPServer(reg, gk)
```

#### 2. 使用 MCP 服务器

```go
// 初始化
result, err := server.Initialize(ctx, initParams)

// 列出工具
tools, err := server.ListTools(ctx)

// 调用工具
result, err := server.CallTool(ctx, "http.request", map[string]any{
    "method": "GET",
    "url":    "https://api.example.com/data",
})
```

### 工具开发

#### 基本工具模板

```go
package mytool

import (
    "context"
    "github.com/Colin4k1024/Aetheris/v2/internal/tool"
)

type MyTool struct{}

func NewMyTool() *MyTool {
    return &MyTool{}
}

func (t *MyTool) Name() string {
    return "my.action"
}

func (t *MyTool) Description() string {
    return "执行特定操作的工具"
}

func (t *MyTool) Schema() tool.Schema {
    return tool.Schema{
        Type:        "object",
        Description: "输入参数",
        Properties: map[string]tool.SchemaProperty{
            "param1": {Type: "string", Description: "参数1描述"},
            "param2": {Type: "integer", Description: "参数2描述"},
        },
        Required: []string{"param1"},
    }
}

func (t *MyTool) Execute(ctx context.Context, input map[string]any) (tool.ToolResult, error) {
    param1, _ := input["param1"].(string)
    // 业务逻辑
    return tool.ToolResult{Content: "success"}, nil
}
```

#### 注册工具

```go
reg := registry.New()
reg.Register(mytool.NewMyTool())
```

### 强类型 Schema

使用 `ToolDescriptor` 定义强类型工具描述：

```go
descriptor := &tool.ToolDescriptor{
    Name:        "database.query",
    Version:     "1.0.0",
    Description: "执行数据库查询",
    Parameters: tool.ParameterConstraint{
        Type: "object",
        Properties: map[string]tool.ParameterConstraint{
            "query": {
                Type:        "string",
                Description: "SQL 查询语句",
                MinLength:   ptr(1),
                MaxLength:   ptr(10000),
            },
            "limit": {
                Type:        "integer",
                Description: "返回结果数量限制",
                Minimum:     ptr(1.0),
                Maximum:     ptr(1000.0),
            },
        },
        Required: []string{"query"},
    },
    Security: tool.SecurityConfig{
        RequireAuth:   true,
        MaxRequestSize: 1024 * 1024,
    },
}

// 验证
if errs := descriptor.Validate(); len(errs) > 0 {
    for _, e := range errs {
        fmt.Println(e)
    }
}

// 转换为 Tool 接口使用的 Schema
schema := descriptor.ToSchema()
```

### Gatekeeper 配置

```go
// 基础配置
gk := gatekeeper.New()

// 启用所有安全检查
gk := gatekeeper.New(
    gatekeeper.WithAllowedHosts([]string{
        "api.github.com",
        "api.openai.com",
    }),
    gatekeeper.WithBlockedPatterns([]string{
        "*.internal",
        "169.254.169.254",  // AWS 元数据
    }),
    gatekeeper.WithNetworkValidation(true),
    gatekeeper.WithTypeValidation(true),
)

// 验证参数
err := gk.Validate("toolName", params, schema)
if err != nil {
    // 处理验证错误
}
```

### MCP 插件

#### 创建插件

```go
// manifest.json
{
    "name": "my-mcp-plugin",
    "version": "1.0.0",
    "description": "我的 MCP 插件",
    "author": "yourname",
    "tools": ["mytool.action1", "mytool.action2"]
}
```

#### 加载插件

```go
loader := mcp.NewPluginLoader("./plugins")
plugins, err := loader.LoadAll()
for _, p := range plugins {
    err := p.Register(registry)
}
```

## API 参考

### MCP 服务器方法

| 方法 | 描述 |
|------|------|
| `Initialize` | 初始化 MCP 连接 |
| `ListTools` | 列出所有可用工具 |
| `CallTool` | 调用指定工具 |

### 工具接口

```go
type Tool interface {
    Name() string
    Description() string
    Schema() Schema
    Execute(ctx context.Context, input map[string]any) (ToolResult, error)
}
```

### Gatekeeper 验证

| 验证类型 | 描述 |
|----------|------|
| `validateRequired` | 检查必需参数 |
| `validateTypes` | 验证参数类型 |
| `validateHTTPRequest` | 验证 HTTP 请求安全 |
| `validateFileRead` | 验证文件读取路径 |
| `validateFileWrite` | 验证文件写入路径 |
| `validateDBQuery` | 验证 SQL 查询安全 |

## 最佳实践

1. **始终验证输入**: 使用 Gatekeeper 进行参数验证
2. **限制权限**: 使用最小权限原则配置 allowedHosts
3. **记录日志**: 记录所有工具调用以便审计
4. **使用强类型**: 使用 ToolDescriptor 定义工具
5. **超时设置**: 为所有网络请求设置超时
