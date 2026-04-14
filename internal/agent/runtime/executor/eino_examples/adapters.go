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

// Package eino_examples 提供 eino-examples (github.com/cloudwego/eino-examples) 的集成适配器
// 支持 flow/agent (react, deer-go, manus), compose (chain, graph, workflow) 等模式
package eino_examples

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/cloudwego/eino/schema"

	"github.com/Colin4k1024/Aetheris/v2/internal/agent/planner"
	"github.com/Colin4k1024/Aetheris/v2/internal/agent/runtime/executor"
)

// ============ ChatModel 接口定义 ============

// ChatModel eino ChatModel 接口
type ChatModel interface {
	Generate(ctx context.Context, input []*schema.Message, opts ...Option) (*schema.Message, error)
	Stream(ctx context.Context, input []*schema.Message, opts ...Option) (*schema.StreamReader[*schema.Message], error)
}

// Option ChatModel 选项
type Option func(*Options)

// Options ChatModel 选项
type Options struct {
	Temperature float64
	MaxTokens   int
	Model       string
}

// WithTemperature 设置温度
func WithTemperature(t float64) Option {
	return func(o *Options) {
		o.Temperature = t
	}
}

// WithMaxTokens 设置最大 token 数
func WithMaxTokens(n int) Option {
	return func(o *Options) {
		o.MaxTokens = n
	}
}

// WithModel 设置模型名称
func WithModel(model string) Option {
	return func(o *Options) {
		o.Model = model
	}
}

// ============ EinoExampleAdapter 接口 ============

// EinoExampleAdapter eino-examples 适配器接口
type EinoExampleAdapter interface {
	// Invoke 执行智能体/工作流
	Invoke(ctx context.Context, input map[string]any) (map[string]any, error)
	// Stream 流式执行
	Stream(ctx context.Context, input map[string]any, onChunk func(chunk map[string]any) error) error
	// GetState 获取状态
	GetState(ctx context.Context) (map[string]any, error)
}

// ============ ReactAgentAdapter ============

// ReactAgentAdapter ReAct 智能体适配器
// 对应 eino-examples/flow/agent/react
// 实现 Think -> Action -> Observe 循环
type ReactAgentAdapter struct {
	Model         ChatModel
	Tools         []interface{}
	SystemPrompt  string
	MaxIterations int
	opts          Options
}

// NewReactAgentAdapter 创建 ReAct 智能体适配器
func NewReactAgentAdapter(model ChatModel, tools []interface{}, opts ...Option) *ReactAgentAdapter {
	a := &ReactAgentAdapter{
		Model:         model,
		Tools:         tools,
		SystemPrompt:  "You are a helpful AI assistant.",
		MaxIterations: 10,
	}

	for _, opt := range opts {
		opt(&a.opts)
	}

	return a
}

// Invoke 执行 ReAct 智能体
func (a *ReactAgentAdapter) Invoke(ctx context.Context, input map[string]any) (map[string]any, error) {
	if a.Model == nil {
		return nil, fmt.Errorf("ReactAgentAdapter: Model not configured")
	}

	goal := a.extractMessage(input)
	if goal == "" {
		goal = "Hello"
	}

	// ReAct 循环：Think -> Action -> Observe
	history := make([]*schema.Message, 0)
	var finalResult string

	for iteration := 0; iteration < a.MaxIterations; iteration++ {
		// 构建上下文
		prompt := a.buildReactPrompt(goal, history)

		msg := &schema.Message{
			Role:    schema.User,
			Content: prompt,
		}

		resp, err := a.Model.Generate(ctx, []*schema.Message{msg})
		if err != nil {
			return nil, err
		}

		// 检查是否有工具调用
		if len(resp.ToolCalls) > 0 {
			history = append(history, &schema.Message{
				Role:      schema.Assistant,
				Content:   resp.Content,
				ToolCalls: resp.ToolCalls,
			})

			// 执行工具
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
			// 没有工具调用，返回结果
			finalResult = resp.Content
			break
		}
	}

	if finalResult == "" {
		finalResult = "Max iterations reached"
	}

	return map[string]any{
		"response": finalResult,
		"history":  history,
	}, nil
}

// buildReactPrompt 构建 ReAct prompt
func (a *ReactAgentAdapter) buildReactPrompt(goal string, history []*schema.Message) string {
	var sb strings.Builder

	sb.WriteString("Task: ")
	sb.WriteString(goal)
	sb.WriteString("\n\n")

	sb.WriteString("You can use tools to help complete the task. ")
	sb.WriteString("Think about your next action and use tools when needed.\n")

	sb.WriteString("\n=== Conversation History ===\n")
	for _, msg := range history {
		switch msg.Role {
		case schema.User:
			sb.WriteString(fmt.Sprintf("User: %s\n", msg.Content))
		case schema.Assistant:
			if len(msg.ToolCalls) > 0 {
				for _, tc := range msg.ToolCalls {
					sb.WriteString(fmt.Sprintf("Assistant: I'll use tool %s with arguments: %s\n", tc.Function.Name, tc.Function.Arguments))
				}
			} else {
				sb.WriteString(fmt.Sprintf("Assistant: %s\n", msg.Content))
			}
		case schema.Tool:
			sb.WriteString(fmt.Sprintf("Tool Result: %s\n", msg.Content))
		}
	}

	sb.WriteString("\nProvide your response (use a tool if needed): ")
	return sb.String()
}

// executeTool 执行工具
func (a *ReactAgentAdapter) executeTool(ctx context.Context, toolName string, args string) (string, error) {
	// 简化实现：记录工具调用
	return fmt.Sprintf("Tool %s called with args: %s", toolName, args), nil
}

// extractMessage 从输入中提取消息
func (a *ReactAgentAdapter) extractMessage(input map[string]any) string {
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

// Stream 流式执行 ReAct 智能体
func (a *ReactAgentAdapter) Stream(ctx context.Context, input map[string]any, onChunk func(chunk map[string]any) error) error {
	if a.Model == nil {
		return fmt.Errorf("ReactAgentAdapter: Model not configured")
	}

	goal := a.extractMessage(input)
	if goal == "" {
		goal = "Hello"
	}

	msg := &schema.Message{
		Role:    schema.User,
		Content: goal,
	}

	reader, err := a.Model.Stream(ctx, []*schema.Message{msg})
	if err != nil {
		return err
	}
	defer reader.Close()

	for {
		msg, err := reader.Recv()
		if err != nil {
			if err == io.EOF {
				break
			}
			return err
		}
		if onChunk != nil {
			_ = onChunk(map[string]any{"content": msg.Content})
		}
	}
	return nil
}

// GetState 获取 ReAct 智能体状态
func (a *ReactAgentAdapter) GetState(ctx context.Context) (map[string]any, error) {
	return map[string]any{
		"status":         "ready",
		"type":           "react",
		"max_iterations": a.MaxIterations,
	}, nil
}

// ============ DEERAgentAdapter ============

// DEERAgentAdapter DEER-Go 智能体适配器
// 对应 eino-examples/flow/agent/deer-go
// 实现 Planning -> Execution -> Reflection 循环
type DEERAgentAdapter struct {
	Model              ChatModel
	Tools              []interface{}
	SystemPrompt       string
	MaxIterations      int
	EnablePlanning     bool
	EnableReflection   bool
	MaxPlanIterations  int
	MaxReflectAttempts int
	opts               Options
}

// NewDEERAgentAdapter 创建 DEER-Go 智能体适配器
func NewDEERAgentAdapter(model ChatModel, tools []interface{}, opts ...Option) *DEERAgentAdapter {
	a := &DEERAgentAdapter{
		Model:              model,
		Tools:              tools,
		SystemPrompt:       "You are a helpful AI assistant.",
		MaxIterations:      15,
		EnablePlanning:     true,
		EnableReflection:   true,
		MaxPlanIterations:  3,
		MaxReflectAttempts: 2,
	}

	for _, opt := range opts {
		opt(&a.opts)
	}

	return a
}

// Invoke 执行 DEER-Go 智能体
func (a *DEERAgentAdapter) Invoke(ctx context.Context, input map[string]any) (map[string]any, error) {
	if a.Model == nil {
		return nil, fmt.Errorf("DEERAgentAdapter: Model not configured")
	}

	goal := a.extractMessage(input)
	if goal == "" {
		goal = "Hello"
	}

	// 阶段 1: Planning
	var plan string
	if a.EnablePlanning {
		planResult, err := a.runPlanning(ctx, goal)
		if err != nil {
			return nil, fmt.Errorf("planning failed: %w", err)
		}
		plan = planResult
	}

	// 阶段 2: Execution with Reflection
	result, err := a.runExecutionWithReflection(ctx, goal, plan)
	if err != nil {
		return nil, err
	}

	return map[string]any{
		"response": result,
		"plan":     plan,
	}, nil
}

// runPlanning 运行规划阶段
func (a *DEERAgentAdapter) runPlanning(ctx context.Context, goal string) (string, error) {
	planningPrompt := fmt.Sprintf(`You are a planning assistant. Break down the following task into steps:

Task: %s

Provide a structured plan with numbered steps.`, goal)

	msg := &schema.Message{
		Role:    schema.User,
		Content: planningPrompt,
	}

	resp, err := a.Model.Generate(ctx, []*schema.Message{msg})
	if err != nil {
		return "", err
	}

	return resp.Content, nil
}

// runExecutionWithReflection 运行执行阶段（带反思）
func (a *DEERAgentAdapter) runExecutionWithReflection(ctx context.Context, goal, plan string) (string, error) {
	var finalResult string

	for attempt := 0; attempt <= a.MaxReflectAttempts; attempt++ {
		var prompt string
		if plan != "" {
			prompt = fmt.Sprintf(`Task: %s

Plan:
%s

Execute the plan and provide the result.`, goal, plan)
		} else {
			prompt = goal
		}

		msg := &schema.Message{
			Role:    schema.User,
			Content: prompt,
		}

		resp, err := a.Model.Generate(ctx, []*schema.Message{msg})
		if err != nil {
			return "", err
		}

		finalResult = resp.Content

		if !a.EnableReflection || attempt >= a.MaxReflectAttempts {
			break
		}

		reflection, err := a.runReflection(ctx, goal, finalResult)
		if err != nil {
			break
		}

		if strings.Contains(strings.ToLower(reflection), "acceptable") ||
			strings.Contains(strings.ToLower(reflection), "good") ||
			strings.Contains(strings.ToLower(reflection), "complete") {
			break
		}

		goal = fmt.Sprintf(`Previous attempt result:
%s

Reflection:
%s

Please improve and try again.`, finalResult, reflection)
	}

	return finalResult, nil
}

// runReflection 运行反思阶段
func (a *DEERAgentAdapter) runReflection(ctx context.Context, goal, result string) (string, error) {
	reflectionPrompt := fmt.Sprintf(`Task: %s

Result:
%s

Evaluate if the result adequately completes the task. If not, explain what needs to be improved.`, goal, result)

	msg := &schema.Message{
		Role:    schema.User,
		Content: reflectionPrompt,
	}

	resp, err := a.Model.Generate(ctx, []*schema.Message{msg})
	if err != nil {
		return "", err
	}

	return resp.Content, nil
}

// extractMessage 从输入中提取消息
func (a *DEERAgentAdapter) extractMessage(input map[string]any) string {
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

// Stream 流式执行
func (a *DEERAgentAdapter) Stream(ctx context.Context, input map[string]any, onChunk func(chunk map[string]any) error) error {
	goal := a.extractMessage(input)
	if goal == "" {
		goal = "Hello"
	}

	msg := &schema.Message{
		Role:    schema.User,
		Content: goal,
	}

	reader, err := a.Model.Stream(ctx, []*schema.Message{msg})
	if err != nil {
		return err
	}
	defer reader.Close()

	for {
		msg, err := reader.Recv()
		if err != nil {
			if err == io.EOF {
				break
			}
			return err
		}
		if onChunk != nil {
			_ = onChunk(map[string]any{"content": msg.Content})
		}
	}
	return nil
}

// GetState 获取状态
func (a *DEERAgentAdapter) GetState(ctx context.Context) (map[string]any, error) {
	return map[string]any{
		"status":            "ready",
		"type":              "deer",
		"max_iterations":    a.MaxIterations,
		"enable_planning":   a.EnablePlanning,
		"enable_reflection": a.EnableReflection,
	}, nil
}

// ============ ManusAgentAdapter ============

// ManusAgentAdapter Manus 智能体适配器
// 对应 eino-examples/flow/agent/manus
// Manus 是一个更自主的 agent，能够规划和执行复杂任务
type ManusAgentAdapter struct {
	Model         ChatModel
	Tools         []interface{}
	SystemPrompt  string
	MaxIterations int
	AutoPlan      bool
	ExecuteTools  bool
	MaxToolCalls  int
	opts          Options
}

// NewManusAgentAdapter 创建 Manus 智能体适配器
func NewManusAgentAdapter(model ChatModel, tools []interface{}, opts ...Option) *ManusAgentAdapter {
	a := &ManusAgentAdapter{
		Model:         model,
		Tools:         tools,
		SystemPrompt:  "You are an autonomous agent capable of completing complex tasks.",
		MaxIterations: 20,
		AutoPlan:      true,
		ExecuteTools:  true,
		MaxToolCalls:  50,
	}

	for _, opt := range opts {
		opt(&a.opts)
	}

	return a
}

// Invoke 执行 Manus 智能体
func (a *ManusAgentAdapter) Invoke(ctx context.Context, input map[string]any) (map[string]any, error) {
	if a.Model == nil {
		return nil, fmt.Errorf("ManusAgentAdapter: Model not configured")
	}

	goal := a.extractMessage(input)
	if goal == "" {
		goal = "Hello"
	}

	history := make([]*schema.Message, 0)

	for iteration := 0; iteration < a.MaxIterations; iteration++ {
		prompt := a.buildContext(goal, history)

		msg := &schema.Message{
			Role:    schema.User,
			Content: prompt,
		}

		resp, err := a.Model.Generate(ctx, []*schema.Message{msg})
		if err != nil {
			return nil, err
		}

		history = append(history, &schema.Message{
			Role:    schema.Assistant,
			Content: resp.Content,
		})

		if len(resp.ToolCalls) > 0 && a.ExecuteTools {
			for _, tc := range resp.ToolCalls {
				if iteration >= a.MaxToolCalls {
					break
				}

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
			}, nil
		}
	}

	return map[string]any{
		"response": "Max iterations reached",
		"history":  history,
	}, nil
}

// buildContext 构建上下文
func (a *ManusAgentAdapter) buildContext(goal string, history []*schema.Message) string {
	var sb strings.Builder

	sb.WriteString("Task: ")
	sb.WriteString(goal)
	sb.WriteString("\n\n")

	if a.AutoPlan {
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
func (a *ManusAgentAdapter) executeTool(ctx context.Context, toolName string, args string) (string, error) {
	var params map[string]any
	if err := json.Unmarshal([]byte(args), &params); err != nil {
		return "", fmt.Errorf("failed to parse tool arguments: %w", err)
	}

	// 简化实现
	return fmt.Sprintf("Tool %s executed with params: %v", toolName, params), nil
}

// extractMessage 从输入中提取消息
func (a *ManusAgentAdapter) extractMessage(input map[string]any) string {
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

// Stream 流式执行
func (a *ManusAgentAdapter) Stream(ctx context.Context, input map[string]any, onChunk func(chunk map[string]any) error) error {
	goal := a.extractMessage(input)
	if goal == "" {
		goal = "Hello"
	}

	msg := &schema.Message{
		Role:    schema.User,
		Content: goal,
	}

	reader, err := a.Model.Stream(ctx, []*schema.Message{msg})
	if err != nil {
		return err
	}
	defer reader.Close()

	for {
		msg, err := reader.Recv()
		if err != nil {
			if err == io.EOF {
				break
			}
			return err
		}
		if onChunk != nil {
			_ = onChunk(map[string]any{"content": msg.Content})
		}
	}
	return nil
}

// GetState 获取状态
func (a *ManusAgentAdapter) GetState(ctx context.Context) (map[string]any, error) {
	return map[string]any{
		"status":         "ready",
		"type":           "manus",
		"max_iterations": a.MaxIterations,
		"auto_plan":      a.AutoPlan,
		"execute_tools":  a.ExecuteTools,
	}, nil
}

// ============ ChainAdapter ============

// ChainAdapter Chain 组合适配器
// 对应 eino-examples/compose/chain
type ChainAdapter struct {
	Nodes     map[string]func(ctx context.Context, input any) (any, error)
	Edges     [][]string
	EntryID   string
	OutputKey string
}

// NewChainAdapter 创建 Chain 适配器
func NewChainAdapter() *ChainAdapter {
	return &ChainAdapter{
		Nodes:     make(map[string]func(ctx context.Context, input any) (any, error)),
		Edges:     make([][]string, 0),
		EntryID:   "",
		OutputKey: "result",
	}
}

// AddNode 添加节点
func (c *ChainAdapter) AddNode(id string, node func(ctx context.Context, input any) (any, error)) {
	c.Nodes[id] = node
	if c.EntryID == "" {
		c.EntryID = id
	}
}

// AddEdge 添加边 (from -> to)
func (c *ChainAdapter) AddEdge(from, to string) {
	c.Edges = append(c.Edges, []string{from, to})
}

// SetOutputKey 设置输出键
func (c *ChainAdapter) SetOutputKey(key string) {
	c.OutputKey = key
}

// Invoke 执行 Chain
func (c *ChainAdapter) Invoke(ctx context.Context, input map[string]any) (map[string]any, error) {
	if c.EntryID == "" {
		return input, nil
	}

	currentNode := c.EntryID
	result := input

	for {
		node, ok := c.Nodes[currentNode]
		if !ok {
			break
		}

		output, err := node(ctx, result)
		if err != nil {
			return nil, fmt.Errorf("node %s failed: %w", currentNode, err)
		}

		// 合并结果
		if m, ok := output.(map[string]any); ok {
			for k, v := range m {
				result[k] = v
			}
		} else {
			result["result"] = output
		}

		nextNode := c.findNextNode(currentNode)
		if nextNode == "" {
			break
		}
		currentNode = nextNode
	}

	return result, nil
}

// findNextNode 查找下一个节点
func (c *ChainAdapter) findNextNode(current string) string {
	for _, edge := range c.Edges {
		if len(edge) >= 2 && edge[0] == current {
			return edge[1]
		}
	}
	return ""
}

// Stream 流式执行
func (c *ChainAdapter) Stream(ctx context.Context, input map[string]any, onChunk func(chunk map[string]any) error) error {
	result, err := c.Invoke(ctx, input)
	if err != nil {
		return err
	}
	if onChunk != nil {
		_ = onChunk(result)
	}
	return nil
}

// GetState 获取状态
func (c *ChainAdapter) GetState(ctx context.Context) (map[string]any, error) {
	return map[string]any{
		"status":     "ready",
		"type":       "chain",
		"node_count": len(c.Nodes),
	}, nil
}

// ============ GraphAdapter ============

// GraphAdapter Graph 组合适配器
// 对应 eino-examples/compose/graph
type GraphAdapter struct {
	Nodes     map[string]func(ctx context.Context, input any) (any, error)
	Edges     [][]string
	EntryID   string
	OutputKey string
}

// NewGraphAdapter 创建 Graph 适配器
func NewGraphAdapter() *GraphAdapter {
	return &GraphAdapter{
		Nodes:     make(map[string]func(ctx context.Context, input any) (any, error)),
		Edges:     make([][]string, 0),
		EntryID:   "",
		OutputKey: "result",
	}
}

// AddNode 添加节点
func (g *GraphAdapter) AddNode(id string, node func(ctx context.Context, input any) (any, error)) {
	g.Nodes[id] = node
	if g.EntryID == "" {
		g.EntryID = id
	}
}

// AddEdge 添加边 (from -> to)
func (g *GraphAdapter) AddEdge(from, to string) {
	g.Edges = append(g.Edges, []string{from, to})
}

// SetEntry 设置入口节点
func (g *GraphAdapter) SetEntry(id string) {
	g.EntryID = id
}

// SetOutputKey 设置输出键
func (g *GraphAdapter) SetOutputKey(key string) {
	g.OutputKey = key
}

// Invoke 执行 Graph (简化版)
func (g *GraphAdapter) Invoke(ctx context.Context, input map[string]any) (map[string]any, error) {
	if g.EntryID == "" {
		return input, nil
	}

	currentNode := g.EntryID
	result := input
	visited := make(map[string]bool)

	for {
		if currentNode == "" {
			break
		}

		if visited[currentNode] {
			break
		}
		visited[currentNode] = true

		node, ok := g.Nodes[currentNode]
		if !ok {
			break
		}

		output, err := node(ctx, result)
		if err != nil {
			return nil, fmt.Errorf("node %s failed: %w", currentNode, err)
		}

		// 合并结果
		if m, ok := output.(map[string]any); ok {
			for k, v := range m {
				result[k] = v
			}
		} else {
			result["result"] = output
		}

		nextNode := g.findNextNode(currentNode)
		if nextNode == "" {
			break
		}
		currentNode = nextNode
	}

	return result, nil
}

// findNextNode 查找下一个节点
func (g *GraphAdapter) findNextNode(current string) string {
	for _, edge := range g.Edges {
		if len(edge) >= 2 && edge[0] == current {
			return edge[1]
		}
	}
	return ""
}

// Stream 流式执行
func (g *GraphAdapter) Stream(ctx context.Context, input map[string]any, onChunk func(chunk map[string]any) error) error {
	result, err := g.Invoke(ctx, input)
	if err != nil {
		return err
	}
	if onChunk != nil {
		_ = onChunk(result)
	}
	return nil
}

// GetState 获取状态
func (g *GraphAdapter) GetState(ctx context.Context) (map[string]any, error) {
	return map[string]any{
		"status":     "ready",
		"type":       "graph",
		"node_count": len(g.Nodes),
	}, nil
}

// ============ WorkflowAdapter ============

// WorkflowAdapter Workflow 组合适配器
// 对应 eino-examples/compose/workflow
type WorkflowAdapter struct {
	Nodes []struct {
		ID string
		Fn func(ctx context.Context, input any) (any, error)
	}
	InputKey  string
	OutputKey string
}

// NewWorkflowAdapter 创建 Workflow 适配器
func NewWorkflowAdapter() *WorkflowAdapter {
	return &WorkflowAdapter{
		Nodes: make([]struct {
			ID string
			Fn func(ctx context.Context, input any) (any, error)
		}, 0),
		InputKey:  "input",
		OutputKey: "result",
	}
}

// AddNode 添加节点（按顺序）
func (w *WorkflowAdapter) AddNode(id string, fn func(ctx context.Context, input any) (any, error)) {
	w.Nodes = append(w.Nodes, struct {
		ID string
		Fn func(ctx context.Context, input any) (any, error)
	}{ID: id, Fn: fn})
}

// SetInputKey 设置输入键
func (w *WorkflowAdapter) SetInputKey(key string) {
	w.InputKey = key
}

// SetOutputKey 设置输出键
func (w *WorkflowAdapter) SetOutputKey(key string) {
	w.OutputKey = key
}

// Invoke 执行 Workflow (顺序执行)
func (w *WorkflowAdapter) Invoke(ctx context.Context, input map[string]any) (map[string]any, error) {
	result := input

	for i, node := range w.Nodes {
		output, err := node.Fn(ctx, result)
		if err != nil {
			return nil, fmt.Errorf("node %s (%d) failed: %w", node.ID, i, err)
		}

		// 合并结果
		if m, ok := output.(map[string]any); ok {
			for k, v := range m {
				result[k] = v
			}
		} else {
			result["result"] = output
		}
	}

	return result, nil
}

// Stream 流式执行
func (w *WorkflowAdapter) Stream(ctx context.Context, input map[string]any, onChunk func(chunk map[string]any) error) error {
	result, err := w.Invoke(ctx, input)
	if err != nil {
		return err
	}
	if onChunk != nil {
		_ = onChunk(result)
	}
	return nil
}

// GetState 获取状态
func (w *WorkflowAdapter) GetState(ctx context.Context) (map[string]any, error) {
	return map[string]any{
		"status":     "ready",
		"type":       "workflow",
		"node_count": len(w.Nodes),
	}, nil
}

// ============ 辅助函数 ============

// ADKAdapter ADK 模式适配器 (简化版)
// 对应 eino-examples/adk
type ADKAdapter struct {
	Agent      interface{}
	Checkpoint interface{}
}

// NewADKAdapter 创建 ADK 适配器
func NewADKAdapter(agent interface{}, checkpoint interface{}) *ADKAdapter {
	return &ADKAdapter{
		Agent:      agent,
		Checkpoint: checkpoint,
	}
}

// Invoke 执行 ADK Agent
func (a *ADKAdapter) Invoke(ctx context.Context, input map[string]any) (map[string]any, error) {
	return map[string]any{
		"response": "ADK response",
	}, nil
}

// Stream 流式执行
func (a *ADKAdapter) Stream(ctx context.Context, input map[string]any, onChunk func(chunk map[string]any) error) error {
	return nil
}

// GetState 获取状态
func (a *ADKAdapter) GetState(ctx context.Context) (map[string]any, error) {
	return map[string]any{
		"status": "ready",
	}, nil
}

// ToNodeRunner 将适配器转换为 NodeRunner
func ToNodeRunner(adapter EinoExampleAdapter) executor.NodeRunner {
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

		result, err := adapter.Invoke(ctx, input)
		if err != nil {
			return nil, err
		}

		p.Results["eino"] = result
		return p, nil
	}
}

// ConvertToPlannerTaskNode 将适配器转换为 planner.TaskNode
func ConvertToPlannerTaskNode(adapter EinoExampleAdapter, nodeType string, config map[string]any) *planner.TaskNode {
	return &planner.TaskNode{
		ID:     "eino_" + nodeType,
		Type:   nodeType,
		Config: config,
	}
}

// JSONMarshal JSON 序列化帮助函数
func JSONMarshal(v any) []byte {
	b, _ := json.Marshal(v)
	return b
}
