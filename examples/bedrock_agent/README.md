# AWS Bedrock Agents Example

本示例展示如何在 Aetheris 中使用 AWS Bedrock Agents 适配器。

## 概述

Bedrock 适配器允许你将 AWS Bedrock 的托管 Agent 集成到 Aetheris 运行时。

## 快速开始

```bash
cd examples/bedrock_agent
go run main.go
```

## 使用方法

### 1. 实现 BedrockClient 接口

```go
type BedrockClient interface {
    CreateAgentSession(ctx context.Context, agentID string, sessionConfig map[string]any) (string, error)
    Invoke(ctx context.Context, agentID, sessionID string, input map[string]any) (map[string]any, error)
    InvokeWithResponseStream(ctx context.Context, agentID, sessionID string, input map[string]any, onChunk func(chunk map[string]any) error) error
    GetAgentSession(ctx context.Context, agentID, sessionID string) (map[string]any, error)
}
```

### 2. 创建 BedrockNodeAdapter

```go
adapter := &BedrockNodeAdapter{
    Client:      yourClient,
    EffectStore: nil, // 生产环境配置
}
```

### 3. 在 TaskGraph 中使用

```go
taskGraph := &planner.TaskGraph{
    Nodes: []planner.TaskNode{
        {
            ID:   "bedrock_agent",
            Type: planner.NodeBedrock,
            Config: map[string]any{
                "agent_id": "your-agent-id",
            },
        },
    },
}
```

## 连接到真实 Bedrock

实际使用中，使用 AWS SDK：

```go
type RealBedrockClient struct {
    Region string
    client *bedrockagent.Client
}

func (c *RealBedrockClient) Invoke(ctx context.Context, agentID, sessionID string, input map[string]any) (map[string]any, error) {
    // 使用 AWS Bedrock Agents SDK
    resp, err := c.client.InvokeAgent(ctx, &bedrockagent.InvokeAgentInput{
        AgentId:     agentID,
        SessionId:   sessionID,
        InputText:   input["goal"].(string),
    })
    return map[string]any{
        "response": resp.Completion,
    }, err
}
```

## 配置要求

- AWS 账户
- Bedrock Agents API 已启用
- 适当的 IAM 权限 (bedrock-agent:*)
