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
	"encoding/json"
	"fmt"
	"log"

	"rag-platform/internal/agent/planner"
	agentexec "rag-platform/internal/agent/runtime/executor"
)

// MyLangGraphClient 是一个示例 LangGraph 客户端实现
// 在实际使用中，你可以连接到真实的 LangGraph API
type MyLangGraphClient struct {
	APIEndpoint string
	APIKey      string
}

// Invoke 调用 LangGraph 流程
func (c *MyLangGraphClient) Invoke(ctx context.Context, input map[string]any) (map[string]any, error) {
	// 示例：实际使用中通过 HTTP 调用 LangGraph API
	// req, _ := json.Marshal(map[string]any{
	//     "input": input,
	// })
	// http.Post(c.APIEndpoint+"/invoke", "application/json", bytes.NewReader(req))

	// 示例响应
	goal, _ := input["goal"].(string)

	// 模拟 LangGraph 执行逻辑
	result := map[string]any{
		"status":    "completed",
		"goal":      goal,
		"reasoning": "Analyzed the request and generated a response",
		"next_action": "respond",
	}

	// 模拟需要人类审批的场景
	// return nil, &LangGraphError{
	//     Code:           LangGraphErrorWait,
	//     CorrelationKey: "approval-" + generateUUID(),
	//     Message:        "需要人工审批",
	// }

	return result, nil
}

// Stream 流式调用 LangGraph
func (c *MyLangGraphClient) Stream(ctx context.Context, input map[string]any, onChunk func(chunk map[string]any) error) error {
	// 示例：流式处理
	chunks := []map[string]any{
		{"chunk": "Thinking", "progress": 0.2},
		{"chunk": "Processing", "progress": 0.5},
		{"chunk": "Finalizing", "progress": 1.0},
	}

	for _, chunk := range chunks {
		if err := onChunk(chunk); err != nil {
			return err
		}
	}
	return nil
}

// State 获取 LangGraph thread 状态
func (c *MyLangGraphClient) State(ctx context.Context, threadID string) (map[string]any, error) {
	return map[string]any{
		"thread_id": threadID,
		"status":    "idle",
	}, nil
}

func main() {
	// 1. 创建 LangGraph 客户端
	client := &MyLangGraphClient{
		APIEndpoint: "http://localhost:8000",
		APIKey:     "your-api-key",
	}

	// 2. 创建 LangGraph 节点适配器
	langGraphAdapter := &agentexec.LangGraphNodeAdapter{
		Client:      client,
		EffectStore: nil, // 生产环境应配置 EffectStore
	}

	// 3. 注册到节点适配器映射（在实际使用时传给 DAGCompiler）
	_ = map[string]agentexec.NodeAdapter{
		planner.NodeLangGraph: langGraphAdapter,
		planner.NodeTool:      &agentexec.ToolNodeAdapter{},
	}

	// 4. 创建 TaskGraph
	taskGraph := &planner.TaskGraph{
		Nodes: []planner.TaskNode{
			{
				ID:   "analyze",
				Type: planner.NodeLangGraph,
				Config: map[string]any{
					"input": map[string]any{
						"task": "analyze_request",
					},
				},
			},
			{
				ID:   "human_approval",
				Type: planner.NodeWait,
				Config: map[string]any{
					"wait_kind":       "signal",
					"correlation_key": "approval-001",
				},
			},
			{
				ID:   "execute",
				Type: planner.NodeTool,
				Config: map[string]any{
					"tool_name": "execute_action",
				},
			},
		},
		Edges: []planner.TaskEdge{
			{From: "analyze", To: "human_approval"},
			{From: "human_approval", To: "execute"},
		},
	}

	// 5. 序列化 TaskGraph（用于提交到 Aetheris）
	graphBytes, err := json.Marshal(taskGraph)
	if err != nil {
		log.Fatalf("Failed to marshal TaskGraph: %v", err)
	}

	fmt.Printf("TaskGraph JSON:\n%s\n", string(graphBytes))
	fmt.Println("\nLangGraph adapter is ready!")
	fmt.Println("To use with Aetheris:")
	fmt.Println("1. Start Aetheris API: go run ./cmd/api")
	fmt.Println("2. Create agent with this TaskGraph")
	fmt.Println("3. Submit job with goal")
}

// 辅助函数：模拟生成 UUID
func generateUUID() string {
	return "xxxxxxxx-xxxx-4xxx-yxxx-xxxxxxxxxxxx"
}
