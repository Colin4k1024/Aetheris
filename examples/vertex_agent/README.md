# Vertex AI Agent Engine Example

本示例展示如何在 Aetheris 中使用 Vertex AI Agent Engine 适配器。

## 概述

Vertex AI 适配器允许你将 Google Cloud Vertex AI 的托管 Agent 集成到 Aetheris 运行时。

## 快速开始

```bash
cd examples/vertex_agent
go run main.go
```

## 使用方法

### 1. 实现 VertexClient 接口

```go
type VertexClient interface {
    CreateSession(ctx context.Context, agent string, sessionConfig map[string]any) (string, error)
    Execute(ctx context.Context, agent, sessionID string, input map[string]any) (map[string]any, error)
    Stream(ctx context.Context, agent, sessionID string, input map[string]any, onChunk func(chunk map[string]any) error) error
    GetSession(ctx context.Context, agent, sessionID string) (map[string]any, error)
}
```

### 2. 创建 VertexNodeAdapter

```go
adapter := &VertexNodeAdapter{
    Client:      yourClient,
    EffectStore: nil, // 生产环境配置
}
```

### 3. 在 TaskGraph 中使用

```go
taskGraph := &planner.TaskGraph{
    Nodes: []planner.TaskNode{
        {
            ID:   "vertex_agent",
            Type: planner.NodeVertex,
            Config: map[string]any{
                "agent": "your-agent-name",
            },
        },
    },
}
```

## 连接到真实 Vertex AI

实际使用中，使用 Google Cloud SDK：

```go
type RealVertexClient struct {
    ProjectID string
    Location  string
    client    *agentengine.Client
}

func (c *RealVertexClient) Execute(ctx context.Context, agent, sessionID string, input map[string]any) (map[string]any, error) {
    // 使用 Vertex AI Agent Engine SDK
    resp, err := c.client.ExecuteAgent(ctx, &agentengine.ExecuteAgentRequest{
        Agent:      agent,
        SessionId:  sessionID,
        Input:      input,
    })
    return resp.Output, err
}
```

## 配置要求

- Google Cloud Project
- Vertex AI Agent Engine API 已启用
- 适当的 IAM 权限
