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
// 支持 flow/agent (react, deer-go, manus), adk, compose (chain, graph, workflow) 等模式
package eino_examples

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"

	"rag-platform/internal/agent/planner"
	"rag-platform/internal/agent/runtime/executor"
)

// ChatModel eino ChatModel 接口 (简化版)
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

// EinoExampleAdapter eino-examples 适配器接口
type EinoExampleAdapter interface {
	// Invoke 执行智能体/工作流
	Invoke(ctx context.Context, input map[string]any) (map[string]any, error)
	// Stream 流式执行
	Stream(ctx context.Context, input map[string]any, onChunk func(chunk map[string]any) error) error
	// GetState 获取状态
	GetState(ctx context.Context) (map[string]any, error)
}

// ReactAgentAdapter ReAct 智能体适配器
// 对应 eino-examples/flow/agent/react
type ReactAgentAdapter struct {
	Model        ChatModel
	Tools        []tool.InvokableTool
	SystemPrompt string
}

// NewReactAgentAdapter 创建 ReAct 智能体适配器
func NewReactAgentAdapter(model ChatModel, tools []tool.InvokableTool, opts ...Option) *ReactAgentAdapter {
	return &ReactAgentAdapter{
		Model:        model,
		Tools:        tools,
		SystemPrompt: "你是一个有帮助的助手。",
	}
}

// Invoke 执行 ReAct 智能体
func (a *ReactAgentAdapter) Invoke(ctx context.Context, input map[string]any) (map[string]any, error) {
	if a.Model == nil {
		return nil, fmt.Errorf("ReactAgentAdapter: Model not configured")
	}

	userMsg := &schema.Message{
		Role:    schema.User,
		Content: fmt.Sprintf("%v", input["prompt"]),
	}

	msgs := []*schema.Message{userMsg}
	resp, err := a.Model.Generate(ctx, msgs)
	if err != nil {
		return nil, err
	}

	return map[string]any{
		"response": resp.Content,
	}, nil
}

// Stream 流式执行 ReAct 智能体
func (a *ReactAgentAdapter) Stream(ctx context.Context, input map[string]any, onChunk func(chunk map[string]any) error) error {
	if a.Model == nil {
		return fmt.Errorf("ReactAgentAdapter: Model not configured")
	}

	userMsg := &schema.Message{
		Role:    schema.User,
		Content: fmt.Sprintf("%v", input["prompt"]),
	}

	msgs := []*schema.Message{userMsg}
	reader, err := a.Model.Stream(ctx, msgs)
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
			_ = onChunk(map[string]any{"content": msg.Content})
		}
	}
	return nil
}

// GetState 获取 ReAct 智能体状态
func (a *ReactAgentAdapter) GetState(ctx context.Context) (map[string]any, error) {
	return map[string]any{
		"status": "ready",
	}, nil
}

// DEERAgentAdapter DEER-Go 智能体适配器
// 对应 eino-examples/flow/agent/deer-go
type DEERAgentAdapter struct {
	Model        ChatModel
	Tools        []tool.InvokableTool
	SystemPrompt string
}

// NewDEERAgentAdapter 创建 DEER-Go 智能体适配器
func NewDEERAgentAdapter(model ChatModel, tools []tool.InvokableTool, opts ...Option) *DEERAgentAdapter {
	return &DEERAgentAdapter{
		Model:        model,
		Tools:        tools,
		SystemPrompt: "你是一个有帮助的助手。",
	}
}

// Invoke 执行 DEER-Go 智能体
func (a *DEERAgentAdapter) Invoke(ctx context.Context, input map[string]any) (map[string]any, error) {
	if a.Model == nil {
		return nil, fmt.Errorf("DEERAgentAdapter: Model not configured")
	}

	userMsg := &schema.Message{
		Role:    schema.User,
		Content: fmt.Sprintf("%v", input["prompt"]),
	}

	msgs := []*schema.Message{userMsg}
	resp, err := a.Model.Generate(ctx, msgs)
	if err != nil {
		return nil, err
	}

	return map[string]any{
		"response": resp.Content,
	}, nil
}

// Stream 流式执行
func (a *DEERAgentAdapter) Stream(ctx context.Context, input map[string]any, onChunk func(chunk map[string]any) error) error {
	_, err := a.Invoke(ctx, input) // 简化实现
	return err
}

// GetState 获取状态
func (a *DEERAgentAdapter) GetState(ctx context.Context) (map[string]any, error) {
	return map[string]any{
		"status": "ready",
	}, nil
}

// ManusAgentAdapter Manus 智能体适配器
// 对应 eino-examples/flow/agent/manus
type ManusAgentAdapter struct {
	Model ChatModel
	Tools []tool.InvokableTool
}

// NewManusAgentAdapter 创建 Manus 智能体适配器
func NewManusAgentAdapter(model ChatModel, tools []tool.InvokableTool) *ManusAgentAdapter {
	return &ManusAgentAdapter{
		Model: model,
		Tools: tools,
	}
}

// Invoke 执行 Manus 智能体
func (a *ManusAgentAdapter) Invoke(ctx context.Context, input map[string]any) (map[string]any, error) {
	if a.Model == nil {
		return nil, fmt.Errorf("ManusAgentAdapter: Model not configured")
	}

	userMsg := &schema.Message{
		Role:    schema.User,
		Content: fmt.Sprintf("%v", input["prompt"]),
	}

	msgs := []*schema.Message{userMsg}
	resp, err := a.Model.Generate(ctx, msgs)
	if err != nil {
		return nil, err
	}

	return map[string]any{
		"response": resp.Content,
	}, nil
}

// Stream 流式执行
func (a *ManusAgentAdapter) Stream(ctx context.Context, input map[string]any, onChunk func(chunk map[string]any) error) error {
	if a.Model == nil {
		return fmt.Errorf("ManusAgentAdapter: Model not configured")
	}

	userMsg := &schema.Message{
		Role:    schema.User,
		Content: fmt.Sprintf("%v", input["prompt"]),
	}

	msgs := []*schema.Message{userMsg}
	reader, err := a.Model.Stream(ctx, msgs)
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
			_ = onChunk(map[string]any{"content": msg.Content})
		}
	}
	return nil
}

// GetState 获取状态
func (a *ManusAgentAdapter) GetState(ctx context.Context) (map[string]any, error) {
	return map[string]any{
		"status": "ready",
	}, nil
}

// ADKAdapter ADK 模式适配器
// 对应 eino-examples/adk
type ADKAdapter struct {
	Agent      interface{} // adk.Agent
	Checkpoint interface{} // compose.CheckPointStore
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

// ChainAdapter Chain 组合适配器
// 对应 eino-examples/compose/chain
type ChainAdapter struct {
	Nodes map[string]func(ctx context.Context, input any) (any, error)
}

// NewChainAdapter 创建 Chain 适配器
func NewChainAdapter() *ChainAdapter {
	return &ChainAdapter{
		Nodes: make(map[string]func(ctx context.Context, input any) (any, error)),
	}
}

// AddNode 添加节点
func (c *ChainAdapter) AddNode(id string, node func(ctx context.Context, input any) (any, error)) {
	c.Nodes[id] = node
}

// Invoke 执行 Chain
func (c *ChainAdapter) Invoke(ctx context.Context, input map[string]any) (map[string]any, error) {
	// 简化实现：直接返回输入
	return input, nil
}

// Stream 流式执行
func (c *ChainAdapter) Stream(ctx context.Context, input map[string]any, onChunk func(chunk map[string]any) error) error {
	return nil
}

// GetState 获取状态
func (c *ChainAdapter) GetState(ctx context.Context) (map[string]any, error) {
	return map[string]any{
		"status": "ready",
	}, nil
}

// GraphAdapter Graph 组合适配器
// 对应 eino-examples/compose/graph
type GraphAdapter struct {
	Nodes map[string]func(ctx context.Context, input any) (any, error)
	Edges [][]string // [from, to]
	Entry string
}

// NewGraphAdapter 创建 Graph 适配器
func NewGraphAdapter() *GraphAdapter {
	return &GraphAdapter{
		Nodes: make(map[string]func(ctx context.Context, input any) (any, error)),
		Edges: make([][]string, 0),
	}
}

// AddNode 添加节点
func (g *GraphAdapter) AddNode(id string, node func(ctx context.Context, input any) (any, error)) {
	g.Nodes[id] = node
}

// AddEdge 添加边
func (g *GraphAdapter) AddEdge(from, to string) {
	g.Edges = append(g.Edges, []string{from, to})
}

// SetEntry 设置入口节点
func (g *GraphAdapter) SetEntry(id string) {
	g.Entry = id
}

// Invoke 执行 Graph
func (g *GraphAdapter) Invoke(ctx context.Context, input map[string]any) (map[string]any, error) {
	return input, nil
}

// Stream 流式执行
func (g *GraphAdapter) Stream(ctx context.Context, input map[string]any, onChunk func(chunk map[string]any) error) error {
	return nil
}

// GetState 获取状态
func (g *GraphAdapter) GetState(ctx context.Context) (map[string]any, error) {
	return map[string]any{
		"status": "ready",
	}, nil
}

// WorkflowAdapter Workflow 组合适配器
// 对应 eino-examples/compose/workflow
type WorkflowAdapter struct {
	Nodes map[string]func(ctx context.Context, input any) (any, error)
}

// NewWorkflowAdapter 创建 Workflow 适配器
func NewWorkflowAdapter() *WorkflowAdapter {
	return &WorkflowAdapter{
		Nodes: make(map[string]func(ctx context.Context, input any) (any, error)),
	}
}

// AddNode 添加节点
func (w *WorkflowAdapter) AddNode(id string, node func(ctx context.Context, input any) (any, error)) {
	w.Nodes[id] = node
}

// Invoke 执行 Workflow
func (w *WorkflowAdapter) Invoke(ctx context.Context, input map[string]any) (map[string]any, error) {
	return input, nil
}

// Stream 流式执行
func (w *WorkflowAdapter) Stream(ctx context.Context, input map[string]any, onChunk func(chunk map[string]any) error) error {
	return nil
}

// GetState 获取状态
func (w *WorkflowAdapter) GetState(ctx context.Context) (map[string]any, error) {
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
