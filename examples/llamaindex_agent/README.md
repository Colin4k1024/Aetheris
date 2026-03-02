# LlamaIndex Agent Example

本示例展示如何在 Aetheris 中使用 LlamaIndex 适配器。

## 概述

LlamaIndex 适配器允许你将 LlamaIndex Agent/ChatEngine 集成到 Aetheris 运行时。

## 快速开始

```bash
cd examples/llamaindex_agent
go run main.go
```

## 使用方法

### 1. 实现 LlamaIndexClient 接口

```go
type LlamaIndexClient interface {
    Invoke(ctx context.Context, input map[string]any) (map[string]any, error)
    Stream(ctx context.Context, input map[string]any, onChunk func(chunk map[string]any) error) error
    GetState(ctx context.Context, sessionID string) (map[string]any, error)
}
```

### 2. 创建 LlamaIndexNodeAdapter

```go
adapter := &LlamaIndexNodeAdapter{
    Client:      yourClient,
    EffectStore: nil, // 生产环境配置
}
```

### 3. 在 TaskGraph 中使用

```go
taskGraph := &planner.TaskGraph{
    Nodes: []*planner.TaskNode{
        {
            ID:   "agent_node",
            Type: planner.NodeLlamaIndex,
            Config: map[string]any{
                "model": "gpt-4",
            },
        },
    },
}
```

## 连接到真实 LlamaIndex

实际使用中，你需要实现真正的 LlamaIndex 客户端：

```go
type RealLlamaIndexClient struct {
    APIEndpoint string
    APIKey      string
}

func (c *RealLlamaIndexClient) Invoke(ctx context.Context, input map[string]any) (map[string]any, error) {
    // 使用 HTTP 调用 LlamaIndex API
    // 例如：LlamaCloud, local server 等
    req, _ := json.Marshal(input)
    resp, err := http.Post(c.APIEndpoint+"/chat", "application/json", bytes.NewReader(req))
    // 处理响应...
    return result, err
}
```
