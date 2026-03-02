// Copyright 2026 fanjia1024
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"context"
	"fmt"
	"log"

	"rag-platform/internal/agent/planner"
	agentexec "rag-platform/internal/agent/runtime/executor"
)

// MyBedrockClient 是一个示例 AWS Bedrock Agents 客户端实现
// 在实际使用中，你需要使用 AWS SDK 连接 Bedrock
type MyBedrockClient struct {
	Region string
}

// CreateAgentSession 创建新的 Agent Session
func (c *MyBedrockClient) CreateAgentSession(ctx context.Context, agentID string, sessionConfig map[string]any) (string, error) {
	// 示例：调用 Bedrock Agents API 创建 session
	return "session-" + agentID + "-123", nil
}

// Invoke 同步调用 Agent
func (c *MyBedrockClient) Invoke(ctx context.Context, agentID, sessionID string, input map[string]any) (map[string]any, error) {
	// 示例响应
	goal, _ := input["goal"].(string)

	result := map[string]any{
		"status":     "completed",
		"goal":       goal,
		"response":   "Processed by AWS Bedrock Agents",
		"session_id": sessionID,
		"agent_id":   agentID,
	}

	return result, nil
}

// InvokeWithResponseStream 流式调用 Agent
func (c *MyBedrockClient) InvokeWithResponseStream(ctx context.Context, agentID, sessionID string, input map[string]any, onChunk func(chunk map[string]any) error) error {
	chunks := []map[string]any{
		{"chunk": "Initializing Bedrock Agent", "progress": 0.2},
		{"chunk": "Processing request", "progress": 0.5},
		{"chunk": "Generating response", "progress": 0.8},
		{"chunk": "Complete", "progress": 1.0},
	}

	for _, chunk := range chunks {
		if err := onChunk(chunk); err != nil {
			return err
		}
	}
	return nil
}

// GetAgentSession 获取 Session 状态
func (c *MyBedrockClient) GetAgentSession(ctx context.Context, agentID, sessionID string) (map[string]any, error) {
	return map[string]any{
		"session_id": sessionID,
		"agent_id":   agentID,
		"status":     "idle",
	}, nil
}

func main() {
	// 1. 创建 Bedrock 客户端
	client := &MyBedrockClient{
		Region: "us-east-1",
	}

	// 2. 创建 Bedrock 节点适配器
	bedrockAdapter := &agentexec.BedrockNodeAdapter{
		Client:      client,
		EffectStore: nil,
	}

	// 3. 创建 TaskGraph
	taskGraph := &planner.TaskGraph{
		Nodes: []planner.TaskNode{
			{
				ID:   "bedrock_agent",
				Type: planner.NodeBedrock,
				Config: map[string]any{
					"agent_id": "my-agent-id",
				},
			},
		},
		Edges: []planner.TaskEdge{},
	}

	log.Printf("Created Bedrock TaskGraph: %+v", taskGraph)
	log.Printf("Adapter: %+v", bedrockAdapter)

	// 4. 演示调用
	ctx := context.Background()
	input := map[string]any{
		"goal": "Analyze this data",
	}

	result, err := client.Invoke(ctx, "my-agent-id", "session-123", input)
	if err != nil {
		log.Fatalf("Invoke failed: %v", err)
	}

	fmt.Printf("Result: %+v\n", result)
}
