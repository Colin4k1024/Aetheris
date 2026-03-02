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

// CrewAIClient CrewAI 桥接客户端
type CrewAIClient struct {
	Endpoint string
	APIKey   string
}

// CrewConfig CrewAI 配置
type CrewConfig struct {
	Agents  []AgentConfig // Agent 配置
	Tasks   []TaskConfig  // Task 配置
	Process ProcessType   // sequential | hierarchical
	Verbose bool          // 详细输出
}

// AgentConfig Agent 配置
type AgentConfig struct {
	Role      string
	Goal      string
	Backstory string
	Tools     []string
}

// TaskConfig Task 配置
type TaskConfig struct {
	Description    string
	Agent          string
	ExpectedOutput string
}

// ProcessType 执行流程类型
type ProcessType string

const (
	ProcessSequential   ProcessType = "sequential"
	ProcessHierarchical ProcessType = "hierarchical"
)

// Invoke 执行 CrewAI crew
func (c *CrewAIClient) Invoke(ctx context.Context, input map[string]any) (map[string]any, error) {
	// 解析输入
	goal, _ := input["goal"].(string)

	// 模拟 CrewAI 执行
	// 实际使用中调用 CrewAI 的 kickoff 方法
	result := map[string]any{
		"status":         "completed",
		"goal":           goal,
		"final_output":   "Processed by CrewAI agents",
		"agent_results":  c.getAgentOutputs(),
		"task_completed": len(c.Config().Tasks),
	}

	// 模拟需要人类审批的场景
	// return nil, &CrewAIError{
	//     Code:          CrewAIErrorAwaitApproval,
	//     Message:       "需要团队负责人审批",
	//     ApprovalLevel: "manager",
	// }

	return result, nil
}

// Config 获取 crew 配置
func (c *CrewAIClient) Config() CrewConfig {
	return CrewConfig{
		Agents: []AgentConfig{
			{Role: "Researcher", Goal: "Research and gather information", Backstory: "Expert researcher"},
			{Role: "Analyzer", Goal: "Analyze findings and provide insights", Backstory: "Data analyst expert"},
			{Role: "Writer", Goal: "Compile findings into a report", Backstory: "Technical writer"},
		},
		Tasks: []TaskConfig{
			{Description: "Research the topic", Agent: "Researcher", ExpectedOutput: "Research findings"},
			{Description: "Analyze results", Agent: "Analyzer", ExpectedOutput: "Analysis report"},
			{Description: "Write final report", Agent: "Writer", ExpectedOutput: "Final document"},
		},
		Process: ProcessSequential,
		Verbose: true,
	}
}

func (c *CrewAIClient) getAgentOutputs() []map[string]any {
	config := c.Config()
	results := make([]map[string]any, len(config.Agents))
	for i, agent := range config.Agents {
		results[i] = map[string]any{
			"role":   agent.Role,
			"status": "completed",
			"output": "Processed " + agent.Role + " task",
		}
	}
	return results
}

// Stream 流式执行
func (c *CrewAIClient) Stream(ctx context.Context, input map[string]any, onChunk func(chunk map[string]any) error) error {
	config := c.Config()

	// 模拟 agent 依次执行
	for i, task := range config.Tasks {
		chunk := map[string]any{
			"stage":    i + 1,
			"total":    len(config.Tasks),
			"task":     task.Description,
			"agent":    task.Agent,
			"progress": float64(i+1) / float64(len(config.Tasks)),
		}
		if err := onChunk(chunk); err != nil {
			return err
		}
	}
	return nil
}

// CrewAIError CrewAI 错误类型
type CrewAIError struct {
	Code          string `json:"code"`
	Message       string `json:"message"`
	ApprovalLevel string `json:"approval_level,omitempty"`
	CorrelationID string `json:"correlation_id,omitempty"`
}

const (
	CrewAIErrorAwaitApproval = "await_approval"
	CrewAIErrorRetryable     = "retryable"
	CrewAIErrorPermanent     = "permanent"
)

func (e *CrewAIError) Error() string {
	return e.Message
}

// MapCrewAIError 将 CrewAI 错误映射到 Aetheris
func MapCrewAIError(err error) error {
	var crewErr *CrewAIError
	if !As(err, &crewErr) {
		return nil
	}

	switch crewErr.Code {
	case CrewAIErrorAwaitApproval:
		return &agentexec.SignalWaitRequired{
			CorrelationKey: crewErr.CorrelationID,
			Reason:         crewErr.Message,
		}
	case CrewAIErrorRetryable:
		return &agentexec.StepFailure{
			Type:  agentexec.StepResultRetryableFailure,
			Inner: err,
		}
	case CrewAIErrorPermanent:
		return &agentexec.StepFailure{
			Type:  agentexec.StepResultPermanentFailure,
			Inner: err,
		}
	}
	return nil
}

func As(err error, target interface{}) bool {
	return false
}

func main() {
	// 1. 创建 CrewAI 客户端
	client := &CrewAIClient{
		Endpoint: "http://localhost:8002",
		APIKey:   "your-api-key",
	}

	config := client.Config()

	// 2. 构建 Aetheris TaskGraph
	// CrewAI 的 sequential 模式对应 Aetheris 的线性 TaskGraph
	// hierarchical 模式可以用条件分支实现
	taskGraph := buildTaskGraph(config)

	// 3. 序列化
	graphBytes, err := json.MarshalIndent(taskGraph, "", "  ")
	if err != nil {
		log.Fatalf("Failed to marshal TaskGraph: %v", err)
	}

	fmt.Printf("CrewAI TaskGraph:\n%s\n", string(graphBytes))
	fmt.Println("\n✅ CrewAI adapter is ready!")
	fmt.Println("\nUsage with Aetheris:")
	fmt.Println("1. Start Aetheris API: go run ./cmd/api")
	fmt.Println("2. Register CrewAI client with DAGCompiler")
	fmt.Println("3. Create agent with this TaskGraph")
	fmt.Println("4. Submit job with goal")
}

// buildTaskGraph 将 CrewAI 配置转换为 Aetheris TaskGraph
func buildTaskGraph(config CrewConfig) *planner.TaskGraph {
	nodes := make([]planner.TaskNode, 0, len(config.Tasks))
	edges := make([]planner.TaskEdge, 0, len(config.Tasks)-1)

	// 为每个 Task 创建一个节点
	for i, task := range config.Tasks {
		nodeID := fmt.Sprintf("task_%d", i)
		nodes = append(nodes, planner.TaskNode{
			ID:   nodeID,
			Type: planner.NodeTool, // 或创建专门的 NodeCrewAI
			Config: map[string]any{
				"tool_name":        "crewai_task",
				"task_description": task.Description,
				"agent_role":       task.Agent,
			},
		})

		// 添加边（sequential 模式）
		if i > 0 {
			edges = append(edges, planner.TaskEdge{
				From: fmt.Sprintf("task_%d", i-1),
				To:   nodeID,
			})
		}
	}

	// 添加人工审批节点（如果是 hierarchical 模式）
	if config.Process == ProcessHierarchical {
		approvalNode := planner.TaskNode{
			ID:   "manager_approval",
			Type: planner.NodeWait,
			Config: map[string]any{
				"wait_kind":       "signal",
				"correlation_key": "crewai-manager-approval",
			},
		}
		nodes = append(nodes, approvalNode)
		edges = append(edges, planner.TaskEdge{
			From: fmt.Sprintf("task_%d", len(config.Tasks)-1),
			To:   "manager_approval",
		})
	}

	return &planner.TaskGraph{
		Nodes: nodes,
		Edges: edges,
	}
}
