# MCP 集成

本目录包含 CoRag 项目的 MCP (Model Context Protocol) 协议实现。

## 模块结构

```
internal/tool/
├── mcp/           # MCP 协议实现
│   ├── server.go  # MCP 服务器
│   └── plugin.go  # 插件支持
├── gatekeeper/    # 本地护栏（参数验证）
├── types/         # 共享类型定义
├── lint/          # 工具描述符 lint 检查
└── descriptor.go  # 强类型工具描述符
```

## 快速开始

```go
import (
    "rag-platform/internal/tool"
    "rag-platform/internal/tool/mcp"
    "rag-platform/internal/tool/gatekeeper"
    "rag-platform/internal/tool/registry"
)

// 1. 创建工具注册表
reg := registry.New()
reg.Register(yourTool)

// 2. 创建 Gatekeeper（安全验证）
gk := gatekeeper.New(
    gatekeeper.WithAllowedHosts([]string{"api.example.com"}),
    gatekeeper.WithTypeValidation(true),
)

// 3. 创建 MCP 服务器
server := mcp.NewMCPServer(reg, gk)

// 4. 使用
tools, _ := server.ListTools(ctx)
result, _ := server.CallTool(ctx, "tool.name", params)
```

## 文档

- [MCP 协议实现指南](docs/mcp/implementation.md)
- [MCP 协议安全指南](docs/mcp/security-guide.md)
