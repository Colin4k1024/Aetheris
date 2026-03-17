# MCP 协议安全指南

## 概述

本文档描述了 CoRag 项目中 MCP (Model Context Protocol) 集成的安全最佳实践和实现细节。

## 核心安全原则

### 1. 最小权限原则

每个工具应仅具有完成其任务所需的最小权限：

- **文件访问工具**: 仅允许访问工作目录
- **网络请求工具**: 限制可访问的域名/IP
- **数据库工具**: 使用只读连接或受限用户

### 2. 参数验证

所有工具输入必须经过严格的参数验证：

```go
// 使用 Gatekeeper 进行参数验证
gk := gatekeeper.New(
    gatekeeper.WithAllowedHosts([]string{"api.example.com"}),
    gatekeeper.WithTypeValidation(true),
)

err := gk.Validate("http.request", params, schema)
if err != nil {
    return err
}
```

### 3. 网络安全

#### 3.1 URL 白名单

```go
// 只允许访问特定域名
allowedHosts := []string{
    "api.github.com",
    "api.openai.com",
    "your-company.com",
}

// 阻止内部网络
blockedPatterns := []string{
    "*.internal",
    "192.168.*",
    "10.*",
    "localhost",
}
```

#### 3.2 防止 SSRF 攻击

- 验证 URL scheme（仅允许 http/https）
- 解析并验证最终 IP 地址
- 阻止对内部服务的请求

### 4. 文件系统安全

#### 4.1 路径遍历防护

```go
// 禁止路径遍历
if strings.Contains(path, "..") {
    return ErrPathTraversal
}

// 限制可访问目录
allowedDirs := []string{"/workspace", "/data"}
if !isInAllowedDir(path, allowedDirs) {
    return ErrAccessDenied
}
```

#### 4.2 危险路径保护

禁止写入系统目录：
- `/etc/`
- `/usr/bin/`
- `/usr/sbin/`
- `C:\Windows\`

### 5. 数据库安全

#### 5.1 SQL 注入防护

- 使用参数化查询
- 限制数据库用户权限
- 禁止危险操作（DROP, DELETE, TRUNCATE）

```go
// 检查危险 SQL 命令
dangerous := []string{"DROP ", "DELETE ", "TRUNCATE ", "ALTER "}
for _, cmd := range dangerous {
    if strings.Contains(strings.ToUpper(query), cmd) {
        return ErrDangerousSQL
    }
}
```

### 6. 速率限制

防止滥用和 DoS 攻击：

```go
type RateLimitConfig struct {
    RequestsPerMinute int // 每分钟请求数
    Burst             int // 突发请求数
}
```

## 工具描述 Schema

所有工具必须提供强类型的参数描述：

```go
descriptor := &tool.ToolDescriptor{
    Name:        "http.request",
    Version:     "1.0.0",
    Description: "发送 HTTP 请求",
    Parameters: tool.ParameterConstraint{
        Type: "object",
        Properties: map[string]tool.ParameterConstraint{
            "method": {
                Type:        "string",
                Description: "HTTP 方法",
                Enum:        []any{"GET", "POST", "PUT", "DELETE"},
            },
            "url": {
                Type:        "string",
                Description: "请求 URL",
                Pattern:     "^https?://",
            },
        },
        Required: []string{"method", "url"},
    },
    Security: tool.SecurityConfig{
        RequireAuth:   true,
        AllowedHosts: []string{"api.example.com"},
        MaxRequestSize: 1024 * 1024, // 1MB
        Timeout:       30000, // 30s
    },
}
```

## MCP 协议交互

### 安全握手流程

1. **初始化**: 客户端发送 `initialize` 请求
2. **能力交换**: 服务器返回支持的 capability
3. **工具列表**: 客户端请求可用工具
4. **工具调用**: 客户端调用工具（经过验证）

### 请求示例

```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "tools/call",
  "params": {
    "name": "http.request",
    "arguments": {
      "method": "GET",
      "url": "https://api.example.com/data"
    }
  }
}
```

## 审计日志

记录所有工具调用：

```go
type AuditLog struct {
    Timestamp   time.Time
    ToolName    string
    Parameters  map[string]any
    Result      string
    UserID      string
    IPAddress   string
}
```

## 常见安全威胁

| 威胁 | 防护措施 |
|------|----------|
| SSRF | URL 白名单、IP 黑名单 |
| 路径遍历 | 路径验证、目录限制 |
| SQL 注入 | 参数化查询、危险命令过滤 |
| 资源耗尽 | 超时、请求大小限制 |
| 权限提升 | 最小权限原则 |

## 配置建议

### 开发环境
```yaml
security:
  networkValidation: true
  typeValidation: true
  allowedHosts:
    - "localhost"
    - "*.dev.example.com"
```

### 生产环境
```yaml
security:
  networkValidation: true
  typeValidation: true
  allowedHosts:
    - "api.production.com"
  blockedPatterns:
    - "*.internal"
    - "169.254.169.254"  # 云元数据
  maxRequestSize: 5242880  # 5MB
  timeout: 30000
  rateLimit:
    requestsPerMinute: 100
    burst: 10
```

## 总结

MCP 集成的安全性需要多层次防护：

1. **入口层**: 参数验证（Gatekeeper）
2. **工具层**: 权限控制和资源限制
3. **网络层**: URL 白名单和 SSRF 防护
4. **审计层**: 完整日志记录

遵循本文档的安全最佳实践，可以显著降低安全风险。
