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

// AutoGenClient AutoGen 桥接接口
// 将 AutoGen agent 封装为 Aetheris 可调用的服务
type AutoGenClient struct {
	Endpoint string
	APIKey   string
}

// AutoGenMessage AutoGen 消息格式
type AutoGenMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// AutoGenConfig AutoGen 配置
type AutoGenConfig struct {
	Agents    []string  // Agent 名称列表
	MaxTurns  int       // 最大对话轮次
	LLMConfig LLMConfig // LLM 配置
}

// LLMConfig LLM 配置
type LLMConfig struct {
	Model       string
	APIKey      string
	APIBase     string
	Temperature float64
}

// Invoke 执行 AutoGen conversation
func (c *AutoGenClient) Invoke(ctx context.Context, input map[string]any) (map[string]any, error) {
	// 解析输入
	messages, ok := input["messages"].([]map[string]any)
	if !ok {
		messages = []map[string]any{}
	}

	// 添加用户消息
	userMsg, _ := input["goal"].(string)
	if userMsg != "" {
		messages = append(messages, map[string]any{
			"role":    "user",
			"content": userMsg,
		})
	}

	// 模拟 AutoGen 执行
	// 实际使用中调用 AutoGen 的 GroupChat 或 ConversableAgent
	result := map[string]any{
		"status":       "completed",
		"last_message": "Processed by AutoGen agents",
		"agent_count":  len(c.GetAgentNames()),
		"messages":     messages,
	}

	// 模拟需要人类输入的场景
	// return nil, &AutoGenError{
	//     Code:           AutoGenErrorNeedsInput,
	//     Message:        "需要人类输入",
	//     InputRequest:   "请确认是否继续",
	// }

	return result, nil
}

// GetAgentNames 获取配置的 Agent 名称
func (c *AutoGenClient) GetAgentNames() []string {
	return []string{"assistant", "critic", "executor"}
}

// Stream 流式执行
func (c *AutoGenClient) Stream(ctx context.Context, input map[string]any, onChunk func(chunk map[string]any) error) error {
	chunks := []map[string]any{
		{"agent": "assistant", "chunk": "Analyzing request...", "progress": 0.3},
		{"agent": "critic", "chunk": "Reviewing plan...", "progress": 0.6},
		{"agent": "executor", "chunk": "Executing action...", "progress": 1.0},
	}

	for _, chunk := range chunks {
		if err := onChunk(chunk); err != nil {
			return err
		}
	}
	return nil
}

// AutoGenNodeAdapter AutoGen 节点适配器
type AutoGenNodeAdapter struct {
	Client      AutoGenClientInterface
	EffectStore agentexec.EffectStore
}

// AutoGenClientInterface AutoGen 客户端接口
type AutoGenClientInterface interface {
	Invoke(ctx context.Context, input map[string]any) (map[string]any, error)
	Stream(ctx context.Context, input map[string]any, onChunk func(chunk map[string]any) error) error
}

// AutoGenError AutoGen 错误类型
type AutoGenError struct {
	Code          string `json:"code"`
	Message       string `json:"message"`
	InputRequest  string `json:"input_request,omitempty"`
	CorrelationID string `json:"correlation_id,omitempty"`
}

const (
	AutoGenErrorNeedsInput = "needs_input"
	AutoGenErrorRetryable  = "retryable"
	AutoGenErrorPermanent  = "permanent"
)

func (e *AutoGenError) Error() string {
	return e.Message
}

// MapAutoGenError 将 AutoGen 错误映射到 Aetheris 错误
func MapAutoGenError(err error) error {
	var autoGenErr *AutoGenError
	if !As(err, &autoGenErr) {
		return nil
	}

	switch autoGenErr.Code {
	case AutoGenErrorNeedsInput:
		return &agentexec.SignalWaitRequired{
			CorrelationKey: autoGenErr.CorrelationID,
			Reason:         autoGenErr.InputRequest,
		}
	case AutoGenErrorRetryable:
		return &agentexec.StepFailure{
			Type:  agentexec.StepResultRetryableFailure,
			Inner: err,
		}
	case AutoGenErrorPermanent:
		return &agentexec.StepFailure{
			Type:  agentexec.StepResultPermanentFailure,
			Inner: err,
		}
	}
	return nil
}

// 辅助函数
func As(err error, target interface{}) bool {
	return false // 简化实现
}

func main() {
	// 1. 创建 AutoGen 客户端
	client := &AutoGenClient{
		Endpoint: "http://localhost:8001",
		APIKey:   "your-api-key",
	}

	// 2. 配置 AutoGen agents
	config := AutoGenConfig{
		Agents:   client.GetAgentNames(),
		MaxTurns: 10,
		LLMConfig: LLMConfig{
			Model:       "gpt-4",
			Temperature: 0.7,
		},
	}

	// 3. 创建 Aetheris TaskGraph
	taskGraph := &planner.TaskGraph{
		Nodes: []planner.TaskNode{
			{
				ID:   "analyze",
				Type: planner.NodeLLM, // 使用 LLM 节点进行初步分析
				Config: map[string]any{
					"prompt": "Analyze the user request and create a plan",
				},
			},
			{
				ID:   "autogen_chat",
				Type: planner.NodeLangGraph, // 复用 LangGraph 节点类型，或创建新的 NodeAutoGen
				Config: map[string]any{
					"agents":    config.Agents,
					"max_turns": config.MaxTurns,
				},
			},
			{
				ID:   "human_review",
				Type: planner.NodeWait,
				Config: map[string]any{
					"wait_kind":       "signal",
					"correlation_key": "autogen-review",
				},
			},
			{
				ID:   "execute",
				Type: planner.NodeTool,
				Config: map[string]any{
					"tool_name": "final_executor",
				},
			},
		},
		Edges: []planner.TaskEdge{
			{From: "analyze", To: "autogen_chat"},
			{From: "autogen_chat", To: "human_review"},
			{From: "human_review", To: "execute"},
		},
	}

	// 4. 序列化
	graphBytes, err := json.MarshalIndent(taskGraph, "", "  ")
	if err != nil {
		log.Fatalf("Failed to marshal TaskGraph: %v", err)
	}

	fmt.Printf("AutoGen TaskGraph:\n%s\n", string(graphBytes))
	fmt.Println("\n✅ AutoGen adapter is ready!")
	fmt.Println("\nUsage with Aetheris:")
	fmt.Println("1. Start Aetheris API: go run ./cmd/api")
	fmt.Println("2. Register AutoGen client with DAGCompiler")
	fmt.Println("3. Create agent with this TaskGraph")
	fmt.Println("4. Submit job with goal")
}
