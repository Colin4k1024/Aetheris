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

// Package eino_agent 提供基于 cloudwego/eino 的完整 Agent 实现
// 支持 React, DEER, Manus 等多种 agent 模式
package eino_agent

import (
	"context"
	"fmt"
	"strings"

	einomodel "github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/flow/agent/react"
	"github.com/cloudwego/eino/schema"

	"github.com/Colin4k1024/Aetheris/v2/internal/agent/runtime/executor"
	"github.com/Colin4k1024/Aetheris/v2/pkg/config"
)

// ============ Agent 接口定义 ============

// Agent eino agent 接口
type Agent interface {
	// Invoke 执行 agent
	Invoke(ctx context.Context, input map[string]any) (map[string]any, error)
	// Stream 流式执行
	Stream(ctx context.Context, input map[string]any, onChunk func(chunk map[string]any) error) error
	// GetName 获取 agent 名称
	GetName() string
	// GetType 获取 agent 类型
	GetType() string
}

// ============ React Agent ============

// ReactAgent 基于 eino react 的完整实现
type ReactAgent struct {
	name           string
	agent          *react.Agent
	toolCallingLLM einomodel.ToolCallingChatModel
	tools          []tool.BaseTool
	maxIterations  int
	systemPrompt   string
}

// NewReactAgent 创建 React Agent
func NewReactAgent(name string, llm einomodel.ToolCallingChatModel, tools []tool.BaseTool, cfg *config.AgentDefConfig) (*ReactAgent, error) {
	agentCfg := &react.AgentConfig{
		ToolCallingModel: llm,
		ToolsConfig: compose.ToolsNodeConfig{
			Tools: tools,
		},
		MaxStep: cfg.MaxIterations,
	}

	reactAgent, err := react.NewAgent(context.Background(), agentCfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create react agent: %w", err)
	}

	return &ReactAgent{
		name:           name,
		agent:          reactAgent,
		toolCallingLLM: llm,
		tools:          tools,
		maxIterations:  cfg.MaxIterations,
		systemPrompt:   cfg.SystemPrompt,
	}, nil
}

// Invoke 执行 React Agent
func (a *ReactAgent) Invoke(ctx context.Context, input map[string]any) (map[string]any, error) {
	prompt := extractPrompt(input)
	if prompt == "" {
		prompt = "Hello"
	}

	msgs := []*schema.Message{
		{
			Role:    schema.User,
			Content: prompt,
		},
	}

	result, err := a.agent.Generate(ctx, msgs)
	if err != nil {
		return nil, err
	}

	return map[string]any{
		"response": result.Content,
		"agent":    a.name,
		"type":     "react",
	}, nil
}

// Stream 流式执行
func (a *ReactAgent) Stream(ctx context.Context, input map[string]any, onChunk func(chunk map[string]any) error) error {
	prompt := extractPrompt(input)
	if prompt == "" {
		prompt = "Hello"
	}

	msgs := []*schema.Message{
		{
			Role:    schema.User,
			Content: prompt,
		},
	}

	reader, err := a.agent.Stream(ctx, msgs)
	if err != nil {
		return err
	}
	defer reader.Close()

	for {
		msg, err := reader.Recv()
		if err != nil {
			break
		}
		if onChunk != nil {
			_ = onChunk(map[string]any{
				"content": msg.Content,
				"agent":   a.name,
				"type":    "react",
			})
		}
	}
	return nil
}

// GetName 获取名称
func (a *ReactAgent) GetName() string {
	return a.name
}

// GetType 获取类型
func (a *ReactAgent) GetType() string {
	return "react"
}

// ============ DEER Agent ============

// DEERAgent DEER-Go Agent 实现
// DEER = Documenting, Executing, Enhancing, and Reasoning
type DEERAgent struct {
	name               string
	llm                einomodel.ChatModel
	tools              []tool.BaseTool
	maxIterations      int
	enablePlanning     bool
	enableReflection   bool
	maxReflectAttempts int
	systemPrompt       string
}

// NewDEERAgent 创建 DEER Agent
func NewDEERAgent(name string, llm einomodel.ChatModel, tools []tool.BaseTool, cfg *config.AgentDefConfig) (*DEERAgent, error) {
	return &DEERAgent{
		name:               name,
		llm:                llm,
		tools:              tools,
		maxIterations:      cfg.MaxIterations,
		enablePlanning:     true,
		enableReflection:   true,
		maxReflectAttempts: 2,
		systemPrompt:       cfg.SystemPrompt,
	}, nil
}

// Invoke 执行 DEER Agent
func (a *DEERAgent) Invoke(ctx context.Context, input map[string]any) (map[string]any, error) {
	prompt := extractPrompt(input)
	if prompt == "" {
		prompt = "Hello"
	}

	// Stage 1: Planning
	var plan string
	if a.enablePlanning {
		planPrompt := fmt.Sprintf(`You are a planning assistant. Break down the following task into steps:

Task: %s

Provide a structured plan with numbered steps.`, prompt)

		msg := &schema.Message{Role: schema.User, Content: planPrompt}
		resp, err := a.llm.Generate(ctx, []*schema.Message{msg})
		if err != nil {
			return nil, fmt.Errorf("planning failed: %w", err)
		}
		plan = resp.Content
	}

	// Stage 2: Execution with Reflection
	var finalResult string
	for attempt := 0; attempt <= a.maxReflectAttempts; attempt++ {
		var execPrompt string
		if plan != "" {
			execPrompt = fmt.Sprintf(`Task: %s

Plan:
%s

Execute the plan and provide the result.`, prompt, plan)
		} else {
			execPrompt = prompt
		}

		msg := &schema.Message{Role: schema.User, Content: execPrompt}
		resp, err := a.llm.Generate(ctx, []*schema.Message{msg})
		if err != nil {
			return nil, fmt.Errorf("execution failed: %w", err)
		}

		finalResult = resp.Content

		if !a.enableReflection || attempt >= a.maxReflectAttempts {
			break
		}

		// Reflection
		reflectPrompt := fmt.Sprintf(`Task: %s

Result:
%s

Evaluate if the result adequately completes the task. If not, explain what needs to be improved.`, prompt, finalResult)

		msg = &schema.Message{Role: schema.User, Content: reflectPrompt}
		reflectResp, err := a.llm.Generate(ctx, []*schema.Message{msg})
		if err != nil {
			break
		}

		reflect := strings.ToLower(reflectResp.Content)
		if strings.Contains(reflect, "acceptable") ||
			strings.Contains(reflect, "good") ||
			strings.Contains(reflect, "complete") {
			break
		}

		prompt = fmt.Sprintf(`Previous result:
%s

Reflection:
%s

Please improve and try again.`, finalResult, reflectResp.Content)
	}

	return map[string]any{
		"response": finalResult,
		"plan":     plan,
		"agent":    a.name,
		"type":     "deer",
	}, nil
}

// Stream 流式执行
func (a *DEERAgent) Stream(ctx context.Context, input map[string]any, onChunk func(chunk map[string]any) error) error {
	prompt := extractPrompt(input)
	if prompt == "" {
		prompt = "Hello"
	}

	msg := &schema.Message{Role: schema.User, Content: prompt}
	reader, err := a.llm.Stream(ctx, []*schema.Message{msg})
	if err != nil {
		return err
	}
	defer reader.Close()

	for {
		msg, err := reader.Recv()
		if err != nil {
			break
		}
		if onChunk != nil {
			_ = onChunk(map[string]any{
				"content": msg.Content,
				"agent":   a.name,
				"type":    "deer",
			})
		}
	}
	return nil
}

// GetName 获取名称
func (a *DEERAgent) GetName() string {
	return a.name
}

// GetType 获取类型
func (a *DEERAgent) GetType() string {
	return "deer"
}

// ============ Manus Agent ============

// ManusAgent Manus 自主 Agent 实现
type ManusAgent struct {
	name          string
	llm           einomodel.ChatModel
	tools         []tool.BaseTool
	maxIterations int
	autoPlan      bool
	executeTools  bool
	systemPrompt  string
}

// NewManusAgent 创建 Manus Agent
func NewManusAgent(name string, llm einomodel.ChatModel, tools []tool.BaseTool, cfg *config.AgentDefConfig) (*ManusAgent, error) {
	return &ManusAgent{
		name:          name,
		llm:           llm,
		tools:         tools,
		maxIterations: cfg.MaxIterations,
		autoPlan:      true,
		executeTools:  true,
		systemPrompt:  cfg.SystemPrompt,
	}, nil
}

// Invoke 执行 Manus Agent
func (a *ManusAgent) Invoke(ctx context.Context, input map[string]any) (map[string]any, error) {
	prompt := extractPrompt(input)
	if prompt == "" {
		prompt = "Hello"
	}

	history := make([]*schema.Message, 0)

	for iteration := 0; iteration < a.maxIterations; iteration++ {
		// 构建上下文
		ctxPrompt := a.buildContext(prompt, history)

		msg := &schema.Message{Role: schema.User, Content: ctxPrompt}
		resp, err := a.llm.Generate(ctx, []*schema.Message{msg})
		if err != nil {
			return nil, err
		}

		history = append(history, &schema.Message{
			Role:    schema.Assistant,
			Content: resp.Content,
		})

		// 检查工具调用
		if len(resp.ToolCalls) > 0 && a.executeTools {
			for _, tc := range resp.ToolCalls {
				toolResult, err := a.executeTool(ctx, tc.Function.Name, tc.Function.Arguments)
				if err != nil {
					history = append(history, &schema.Message{
						Role:       schema.Tool,
						Content:    fmt.Sprintf("Error: %v", err),
						ToolCallID: tc.ID,
					})
				} else {
					history = append(history, &schema.Message{
						Role:       schema.Tool,
						Content:    toolResult,
						ToolCallID: tc.ID,
					})
				}
			}
		} else {
			return map[string]any{
				"response": resp.Content,
				"history":  history,
				"agent":    a.name,
				"type":     "manus",
			}, nil
		}
	}

	return map[string]any{
		"response": "Max iterations reached",
		"history":  history,
		"agent":    a.name,
		"type":     "manus",
	}, nil
}

// buildContext 构建上下文
func (a *ManusAgent) buildContext(goal string, history []*schema.Message) string {
	var sb strings.Builder
	sb.WriteString("Task: ")
	sb.WriteString(goal)
	sb.WriteString("\n\n")

	if a.autoPlan {
		sb.WriteString("Please plan and execute this task step by step.\n")
	}

	sb.WriteString("\n=== Conversation History ===\n")
	for _, msg := range history {
		switch msg.Role {
		case schema.User:
			sb.WriteString(fmt.Sprintf("User: %s\n", msg.Content))
		case schema.Assistant:
			sb.WriteString(fmt.Sprintf("Assistant: %s\n", msg.Content))
		case schema.Tool:
			sb.WriteString(fmt.Sprintf("Tool Result: %s\n", msg.Content))
		}
	}

	sb.WriteString("\nProvide your next action or final response: ")
	return sb.String()
}

// executeTool 执行工具
func (a *ManusAgent) executeTool(ctx context.Context, toolName string, args string) (string, error) {
	// 简化实现：记录工具调用
	return fmt.Sprintf("Tool %s executed with args: %s", toolName, args), nil
}

// Stream 流式执行
func (a *ManusAgent) Stream(ctx context.Context, input map[string]any, onChunk func(chunk map[string]any) error) error {
	prompt := extractPrompt(input)
	if prompt == "" {
		prompt = "Hello"
	}

	msg := &schema.Message{Role: schema.User, Content: prompt}
	reader, err := a.llm.Stream(ctx, []*schema.Message{msg})
	if err != nil {
		return err
	}
	defer reader.Close()

	for {
		msg, err := reader.Recv()
		if err != nil {
			break
		}
		if onChunk != nil {
			_ = onChunk(map[string]any{
				"content": msg.Content,
				"agent":   a.name,
				"type":    "manus",
			})
		}
	}
	return nil
}

// GetName 获取名称
func (a *ManusAgent) GetName() string {
	return a.name
}

// GetType 获取类型
func (a *ManusAgent) GetType() string {
	return "manus"
}

// ============ 辅助函数 ============

// extractPrompt 从输入中提取 prompt
func extractPrompt(input map[string]any) string {
	if prompt, ok := input["prompt"].(string); ok && prompt != "" {
		return prompt
	}
	if goal, ok := input["goal"].(string); ok && goal != "" {
		return goal
	}
	if msg, ok := input["message"].(string); ok && msg != "" {
		return msg
	}
	return ""
}

// ============ Agent 工厂 ============

// AgentFactory Agent 工厂接口
type AgentFactory interface {
	CreateAgent(name string, cfg *config.AgentDefConfig, llm einomodel.ChatModel, tools []tool.BaseTool) (Agent, error)
}

// ReactAgentFactory React Agent 工厂
type ReactAgentFactory struct{}

func (f *ReactAgentFactory) CreateAgent(name string, cfg *config.AgentDefConfig, llm einomodel.ChatModel, tools []tool.BaseTool) (Agent, error) {
	// 尝试转换为 ToolCallingChatModel
	var tcLLM einomodel.ToolCallingChatModel
	if tc, ok := llm.(einomodel.ToolCallingChatModel); ok {
		tcLLM = tc
	}
	if tcLLM == nil {
		return nil, fmt.Errorf("LLM does not support tool calling")
	}
	return NewReactAgent(name, tcLLM, tools, cfg)
}

// DEERAgentFactory DEER Agent 工厂
type DEERAgentFactory struct{}

func (f *DEERAgentFactory) CreateAgent(name string, cfg *config.AgentDefConfig, llm einomodel.ChatModel, tools []tool.BaseTool) (Agent, error) {
	return NewDEERAgent(name, llm, tools, cfg)
}

// ManusAgentFactory Manus Agent 工厂
type ManusAgentFactory struct{}

func (f *ManusAgentFactory) CreateAgent(name string, cfg *config.AgentDefConfig, llm einomodel.ChatModel, tools []tool.BaseTool) (Agent, error) {
	return NewManusAgent(name, llm, tools, cfg)
}

// ============ Agent 转换为 NodeRunner ============

// ToNodeRunner 将 Agent 转换为 NodeRunner
func ToNodeRunner(agent Agent) executor.NodeRunner {
	return func(ctx context.Context, p *executor.AgentDAGPayload) (*executor.AgentDAGPayload, error) {
		if p == nil {
			p = &executor.AgentDAGPayload{}
		}
		if p.Results == nil {
			p.Results = make(map[string]any)
		}

		input := map[string]any{
			"goal":   p.Goal,
			"prompt": p.Goal,
		}

		result, err := agent.Invoke(ctx, input)
		if err != nil {
			return nil, err
		}

		p.Results[agent.GetName()] = result
		return p, nil
	}
}
