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

// MyLlamaIndexClient 是一个示例 LlamaIndex 客户端实现
// 在实际使用中，你可以连接到真实的 LlamaIndex API (如 LlamaCloud) 或本地部署
type MyLlamaIndexClient struct {
	APIEndpoint string
	APIKey      string
}

// Invoke 调用 LlamaIndex Agent/ChatEngine
func (c *MyLlamaIndexClient) Invoke(ctx context.Context, input map[string]any) (map[string]any, error) {
	// 示例：实际使用中通过 HTTP 调用 LlamaIndex API
	// req, _ := json.Marshal(input)
	// http.Post(c.APIEndpoint+"/chat", "application/json", bytes.NewReader(req))

	// 示例响应
	goal, _ := input["goal"].(string)

	// 模拟 LlamaIndex Agent 执行逻辑
	result := map[string]any{
		"status":   "completed",
		"goal":     goal,
		"response": "Processed query using LlamaIndex Agent",
		"sources":  []string{"doc1", "doc2"},
	}

	return result, nil
}

// Stream 流式调用 LlamaIndex
func (c *MyLlamaIndexClient) Stream(ctx context.Context, input map[string]any, onChunk func(chunk map[string]any) error) error {
	// 示例：流式处理
	chunks := []map[string]any{
		{"chunk": "Analyzing query", "progress": 0.2},
		{"chunk": "Retrieving context", "progress": 0.5},
		{"chunk": "Generating response", "progress": 0.8},
		{"chunk": "Final answer", "progress": 1.0},
	}

	for _, chunk := range chunks {
		if err := onChunk(chunk); err != nil {
			return err
		}
	}
	return nil
}

// GetState 获取 Agent 状态
func (c *MyLlamaIndexClient) GetState(ctx context.Context, sessionID string) (map[string]any, error) {
	return map[string]any{
		"session_id": sessionID,
		"status":     "idle",
	}, nil
}

func main() {
	// 1. 创建 LlamaIndex 客户端
	client := &MyLlamaIndexClient{
		APIEndpoint: "http://localhost:8000",
		APIKey:      "your-api-key",
	}

	// 2. 创建 LlamaIndex 节点适配器
	llamaIndexAdapter := &agentexec.LlamaIndexNodeAdapter{
		Client:      client,
		EffectStore: nil, // 生产环境应配置 EffectStore
	}

	// 3. 创建 TaskGraph
	taskGraph := &planner.TaskGraph{
		Nodes: []planner.TaskNode{
			{
				ID:   "agent_node",
				Type: planner.NodeLlamaIndex, // 使用 llamaindex 节点类型
				Config: map[string]any{
					"model": "gpt-4",
				},
			},
		},
		Edges: []planner.TaskEdge{},
	}

	log.Printf("Created LlamaIndex TaskGraph: %+v", taskGraph)
	log.Printf("Adapter: %+v", llamaIndexAdapter)

	// 4. 演示调用
	ctx := context.Background()
	input := map[string]any{
		"goal": "What is the capital of France?",
	}

	result, err := client.Invoke(ctx, input)
	if err != nil {
		log.Fatalf("Invoke failed: %v", err)
	}

	fmt.Printf("Result: %+v\n", result)
}
