# Contributing to MCP Tools Marketplace

感谢您对 MCP Tools Marketplace 的贡献！本文档提供了添加新工具的指南。

## 添加新工具的步骤

### 1. 创建工具目录

每个工具应有独立的目录：

```
tools/mcp-gateway/tools/mcp-{name}/
├── tool.go           # 主要实现
├── tool_test.go      # 单元测试
├── manifest.yaml     # 工具清单
└── README.md         # 工具文档
```

### 2. 实现 Tool 接口

```go
package mcp

import (
    "context"
    "github.com/Colin4k1024/Aetheris/v2/internal/tool"
)

// MyTool 自定义 MCP 工具
type MyTool struct {
    config *Config
}

// NewMyTool 创建工具实例
func NewMyTool(config *Config) *MyTool {
    return &MyTool{config: config}
}

// Name 返回工具名称
func (t *MyTool) Name() string {
    return "mcp.mytool"
}

// Description 返回工具描述
func (t *MyTool) Description() string {
    return "A description of what this tool does"
}

// Schema 返回参数 Schema
func (t *MyTool) Schema() tool.Schema {
    return tool.Schema{
        Type: "object",
        Properties: map[string]tool.SchemaProperty{
            "param1": {
                Type:        "string",
                Description: "First parameter",
            },
            "param2": {
                Type:        "integer",
                Description: "Second parameter",
            },
        },
        Required: []string{"param1"},
    }
}

// Execute 执行工具
func (t *MyTool) Execute(ctx context.Context, input map[string]any) (tool.ToolResult, error) {
    // 解析参数
    param1, _ := input["param1"].(string)
    
    // 执行业务逻辑
    result, err := t.doSomething(ctx, param1)
    if err != nil {
        return tool.ToolResult{Err: err.Error()}, nil
    }
    
    return tool.ToolResult{Content: result}, nil
}
```

### 3. 创建 manifest.yaml

```yaml
name: mcp-mytool
version: 1.0.0
description: A custom MCP tool
author: Your Name
category: integration

tools:
  - name: mcp.mytool
    description: Tool description
    parameters:
      - name: param1
        type: string
        required: true
        description: First parameter
      - name: param2
        type: integer
        required: false
        description: Second parameter

dependencies:
  - github.com/external/pkg

configuration:
  - name: api_key
    type: string
    required: true
    description: API key for external service
```

### 4. 添加工具测试

```go
package mcp

import (
    "context"
    "testing"
)

func TestMyTool_Execute(t *testing.T) {
    tool := NewMyTool(&Config{Param: "test"})
    
    tests := []struct {
        name    string
        input   map[string]any
        want    string
        wantErr bool
    }{
        {
            name:  "valid input",
            input: map[string]any{"param1": "value1"},
            want:  "expected result",
            wantErr: false,
        },
        {
            name:    "missing required param",
            input:   map[string]any{},
            want:    "",
            wantErr: true,
        },
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            result, err := tool.Execute(context.Background(), tt.input)
            if (err != nil) != tt.wantErr {
                t.Errorf("Execute() error = %v, wantErr %v", err, tt.wantErr)
                return
            }
            if result.Err != "" && !tt.wantErr {
                t.Errorf("Execute() error = %v", result.Err)
            }
        })
    }
}
```

### 5. 更新 registry.yaml

在 `registry.yaml` 中添加工具条目：

```yaml
tools:
  - name: mcp-mytool
    path: tools/mcp-gateway/tools/mcp-mytool
    status: template
    manifest: manifest.yaml
    category: integration
    tags:
      - github
      - api
    description: A custom MCP tool
```

## 工具开发规范

### 命名规范

- 工具名称格式: `mcp.{category}.{name}`
- 示例: `mcp-github.search_repos`, `mcp-filesystem.read_file`

### Schema 规范

- 使用 JSON Schema Draft-07 格式
- 必须定义 `type` 和 `properties`
- 必填参数必须在 `required` 数组中声明
- 每个参数需提供 `description`

### 错误处理规范

- 使用 `tool.ToolResult.Err` 返回业务错误
- 使用 `error` 返回系统错误（如网络故障）
- 错误信息应简洁明了，便于调试

### 安全性规范

- 敏感配置通过环境变量注入
- 文件操作需验证路径遍历
- 数据库操作需防止 SQL 注入
- API 调用需处理限流和认证

## 发布工具

1. 确保所有测试通过：`go test ./tools/mcp-gateway/tools/...`
2. 更新 README.md 中的工具列表
3. 提交 Pull Request 到 `main` 分支

## 工具审核清单

- [ ] 实现 `Tool` 接口的所有方法
- [ ] 提供完整的 Schema 定义
- [ ] 添加单元测试，覆盖主要路径
- [ ] 创建 manifest.yaml 清单文件
- [ ] 编写清晰的文档和使用示例
- [ ] 处理所有可能的错误情况
- [ ] 敏感配置使用环境变量
