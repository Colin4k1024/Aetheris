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

package executor

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"

	"rag-platform/internal/agent/planner"
	"rag-platform/internal/agent/runtime"
)

// EinoChatModel eino ChatModel 接口
type EinoChatModel interface {
	Generate(ctx context.Context, input []*schema.Message) (*schema.Message, error)
	Stream(ctx context.Context, input []*schema.Message) (*schema.StreamReader[*schema.Message], error)
}

// EinoToolCallingChatModel 支持工具调用的 LLM 接口
type EinoToolCallingChatModel interface {
	EinoChatModel
	WithTools(tools []*schema.ToolInfo) (EinoToolCallingChatModel, error)
}

// ToolExecutor 工具执行器接口
type ToolExecutor interface {
	Execute(ctx context.Context, toolName string, input map[string]any) (string, error)
}

// EinoNodeAdapter 将 eino agent 型 TaskNode 转为 DAG 节点
// 实现真正的 ReAct 循环: Think -> Action -> Observe -> Think -> ... -> Final
type EinoNodeAdapter struct {
	// LLM 是 eino ChatModel，用于执行 agent
	LLM            EinoChatModel
	ToolCallingLLM EinoToolCallingChatModel // 支持工具调用的 LLM
	Tools          ToolExecutor              // 工具执行器
	CommandEventSink CommandEventSink
	EffectStore      EffectStore
	// NodeType 指定 agent 类型：eino_react, eino_deer, eino_manus
	NodeType string
	// MaxIterations 最大迭代次数
	MaxIterations int
}

// runNode 执行真正的 ReAct Agent 循环
func (a *EinoNodeAdapter) runNode(ctx context.Context, taskID string, cfg map[string]any, agent *runtime.Agent, p *AgentDAGPayload) (*AgentDAGPayload, error) {
	// 验证 LLM
	llm := a.LLM
	if a.ToolCallingLLM != nil {
		llm = a.ToolCallingLLM
	}
	if llm == nil {
		return nil, fmt.Errorf("EinoNodeAdapter: LLM not configured")
	}

	if p == nil {
		p = &AgentDAGPayload{}
	}
	if p.Results == nil {
		p.Results = make(map[string]any)
	}

	jobID := JobIDFromContext(ctx)

	// 检查 effect store 是否需要重放
	if a.EffectStore != nil && jobID != "" {
		eff, err := a.EffectStore.GetEffectByJobAndCommandID(ctx, jobID, taskID)
		if err == nil && eff != nil && len(eff.Output) > 0 {
			var out map[string]any
			if json.Unmarshal(eff.Output, &out) == nil {
				p.Results[taskID] = out
				return p, nil
			}
		}
	}

	// 获取配置
	maxIterations := a.MaxIterations
	if maxIterations <= 0 {
		maxIterations = 10 // 默认最大 10 轮
	}
	if maxIterations > 20 {
		maxIterations = 20 // 最大限制
	}

	// 获取系统提示词
	systemPrompt := "You are a helpful assistant that can use tools to answer questions."
	if sp, ok := cfg["system_prompt"].(string); ok && sp != "" {
		systemPrompt = sp
	}

	// 获取可用工具
	tools := a.getTools(agent, cfg)
	var toolLLM EinoToolCallingChatModel
	if a.ToolCallingLLM != nil && len(tools) > 0 {
		var err error
		toolLLM, err = a.ToolCallingLLM.WithTools(tools)
		if err != nil {
			return nil, fmt.Errorf("EinoNodeAdapter: bind tools failed: %w", err)
		}
	} else {
		toolLLM = a.ToolCallingLLM
	}

	// 构建消息历史
	messages := a.buildMessages(p.Goal, systemPrompt)

	// 发送开始事件
	if a.CommandEventSink != nil && jobID != "" {
		inputBytes, _ := json.Marshal(map[string]any{
			"goal":    p.Goal,
			"type":    a.NodeType,
			"system":  systemPrompt,
			"tools":   len(tools),
		})
		_ = a.CommandEventSink.AppendCommandEmitted(ctx, jobID, taskID, taskID, a.NodeType, inputBytes)
	}

	// ReAct 循环
	var finalOutput string
	var iterations int
	var toolCallsMade []map[string]any

	for iterations = 0; iterations < maxIterations; iterations++ {
		// 1. Think: 调用 LLM 获取响应
		var resp *schema.Message
		var err error

		// 如果有工具可用且绑定成功，使用工具调用模式
		if toolLLM != nil && len(tools) > 0 {
			resp, err = toolLLM.Generate(ctx, messages)
		} else {
			// 没有工具，使用普通模式
			resp, err = llm.Generate(ctx, messages)
		}

		if err != nil {
			if p.Results == nil {
				p.Results = make(map[string]any)
			}
			p.Results[taskID] = map[string]any{"error": err.Error(), "iterations": iterations}
			return p, fmt.Errorf("EinoNodeAdapter: LLM generate failed: %w", err)
		}

		// 2. 检查是否有 tool calls
		if resp != nil && len(resp.ToolCalls) > 0 {
			// 3. Action: 执行每个工具
			for _, tc := range resp.ToolCalls {
				toolName := tc.Function.Name
				toolArgs := tc.Function.Arguments

				// 解析参数
				var args map[string]any
				if err := json.Unmarshal([]byte(toolArgs), &args); err != nil {
					args = map[string]any{"raw": toolArgs}
				}

				// 记录工具调用
				toolCallsMade = append(toolCallsMade, map[string]any{
					"name":       toolName,
					"arguments":  args,
					"id":         tc.ID,
				})

				// 发送工具调用开始事件
				if a.CommandEventSink != nil && jobID != "" {
					toolInputBytes, _ := json.Marshal(map[string]any{
						"tool_name": toolName,
						"arguments": args,
					})
					_ = a.CommandEventSink.AppendCommandEmitted(ctx, jobID, taskID, taskID, "tool", toolInputBytes)
				}

				// 执行工具
				var toolResult string
				if a.Tools != nil {
					result, err := a.Tools.Execute(ctx, toolName, args)
					if err != nil {
						toolResult = fmt.Sprintf(`{"error": "%s"}`, err.Error())
					} else {
						toolResult = result
					}
				} else {
					toolResult = `{"error": "no tool executor configured"}`
				}

				// 发送工具返回事件
				if a.CommandEventSink != nil && jobID != "" {
					toolResultBytes, _ := json.Marshal(map[string]any{
						"tool_name": toolName,
						"result":    toolResult,
					})
					_ = a.CommandEventSink.AppendCommandCommitted(ctx, jobID, taskID, taskID, toolResultBytes, "")
				}

				// 4. Observe: 添加工具结果到消息历史
				toolMsg := &schema.Message{
					Role:      schema.Tool,
					ToolCallID: tc.ID,
					ToolName:  toolName,
					Content:   toolResult,
				}
				messages = append(messages, toolMsg)
			}

			// 继续循环，让 LLM 基于工具结果生成最终答案
			continue
		}

		// 没有 tool calls，说明得到最终答案
		if resp != nil && resp.Content != "" {
			finalOutput = resp.Content
			break
		}

		// LLM 返回空内容，可能需要继续
		if resp == nil || (resp.Content == "" && len(resp.ToolCalls) == 0) {
			// 尝试继续一轮
			continue
		}
	}

	// 检查是否超过最大迭代
	if iterations >= maxIterations {
		finalOutput = fmt.Sprintf("[Max iterations reached: %d] %s", maxIterations, finalOutput)
	}

	// 构建结果
	agentResult := map[string]any{
		"output":            finalOutput,
		"type":              a.NodeType,
		"model":             "eino",
		"iterations":        iterations,
		"tool_calls_made":   toolCallsMade,
		"had_tool_calls":    len(toolCallsMade) > 0,
	}

	// 存储 effect
	if a.EffectStore != nil && jobID != "" {
		inputBytes, _ := json.Marshal(map[string]any{"goal": p.Goal})
		outputBytes, _ := json.Marshal(agentResult)
		_ = a.EffectStore.PutEffect(ctx, &EffectRecord{
			JobID:     jobID,
			CommandID: taskID,
			Kind:      EffectKindLLM,
			Input:     inputBytes,
			Output:    outputBytes,
			Metadata:  map[string]any{"adapter": "eino_" + a.NodeType, "iterations": iterations},
		})
	}

	// 发送完成事件
	if a.CommandEventSink != nil && jobID != "" {
		outputBytes, _ := json.Marshal(agentResult)
		_ = a.CommandEventSink.AppendCommandCommitted(ctx, jobID, taskID, taskID, outputBytes, "")
	}

	p.Results[taskID] = agentResult
	return p, nil
}

// buildMessages 构建消息历史
func (a *EinoNodeAdapter) buildMessages(goal, systemPrompt string) []*schema.Message {
	var messages []*schema.Message

	// 添加系统提示
	if systemPrompt != "" {
		messages = append(messages, &schema.Message{
			Role:    schema.System,
			Content: systemPrompt,
		})
	}

	// 添加用户消息
	messages = append(messages, &schema.Message{
		Role:    schema.User,
		Content: goal,
	})

	return messages
}

// getTools 获取可用工具列表
func (a *EinoNodeAdapter) getTools(agent *runtime.Agent, cfg map[string]any) []*schema.ToolInfo {
	var tools []*schema.ToolInfo

	// 如果有工具执行器，提供默认工具
	if a.Tools != nil {
		// 计算器工具
		calculatorTool := &schema.ToolInfo{
			Name: "calculator",
			Desc: "执行数学计算。支持操作: add(加), subtract(减), multiply(乘), divide(除)。输入格式: {\"operation\": \"add\", \"value1\": 1, \"value2\": 2}",
		}
		tools = append(tools, calculatorTool)

		// 搜索工具
		searchTool := &schema.ToolInfo{
			Name: "search",
			Desc: "搜索信息。输入格式: {\"query\": \"关键词\"}",
		}
		tools = append(tools, searchTool)

		// 天气工具
		weatherTool := &schema.ToolInfo{
			Name: "weather",
			Desc: "查询城市天气。输入格式: {\"city\": \"城市名\"}",
		}
		tools = append(tools, weatherTool)
	}

	// 从配置中获取自定义工具
	if toolConfigs, ok := cfg["tools"].([]map[string]any); ok {
		for _, tc := range toolConfigs {
			if name, ok := tc["name"].(string); ok {
				desc := ""
				if d, ok := tc["description"].(string); ok {
					desc = d
				}
				tools = append(tools, &schema.ToolInfo{
					Name: name,
					Desc: desc,
				})
			}
		}
	}

	return tools
}

// ToDAGNode 实现 NodeAdapter
func (a *EinoNodeAdapter) ToDAGNode(task *planner.TaskNode, agent *runtime.Agent) (*compose.Lambda, error) {
	taskID := task.ID
	cfg := task.Config
	if cfg == nil {
		cfg = map[string]any{}
	}

	return compose.InvokableLambda[*AgentDAGPayload, *AgentDAGPayload](func(ctx context.Context, p *AgentDAGPayload) (*AgentDAGPayload, error) {
		return a.runNode(ctx, taskID, cfg, agent, p)
	}), nil
}

// ToNodeRunner 实现 NodeAdapter
func (a *EinoNodeAdapter) ToNodeRunner(task *planner.TaskNode, agent *runtime.Agent) (NodeRunner, error) {
	taskID := task.ID
	cfg := task.Config
	if cfg == nil {
		cfg = map[string]any{}
	}

	return func(ctx context.Context, p *AgentDAGPayload) (*AgentDAGPayload, error) {
		return a.runNode(ctx, taskID, cfg, agent, p)
	}, nil
}

// NewEinoNodeAdapter 创建 eino node adapter
func NewEinoNodeAdapter(llm EinoChatModel, toolLLM EinoToolCallingChatModel, tools ToolExecutor, nodeType string) *EinoNodeAdapter {
	return &EinoNodeAdapter{
		LLM:            llm,
		ToolCallingLLM: toolLLM,
		Tools:          tools,
		NodeType:       nodeType,
		MaxIterations:  10,
	}
}

// NewEinoNodeAdapterWithOptions 创建带选项的 adapter
func NewEinoNodeAdapterWithOptions(llm EinoChatModel, toolLLM EinoToolCallingChatModel, tools ToolExecutor, nodeType string, maxIterations int) *EinoNodeAdapter {
	return &EinoNodeAdapter{
		LLM:            llm,
		ToolCallingLLM: toolLLM,
		Tools:          tools,
		NodeType:       nodeType,
		MaxIterations:  maxIterations,
	}
}

// 确保接口实现正确
var _ NodeAdapter = (*EinoNodeAdapter)(nil)

// ============ 工具实现 ============

// BuiltinTools 内置工具实现
type BuiltinTools struct {
	calculator func(op string, v1, v2 int) (int, error)
	search     func(query string) (string, error)
	weather    func(city string) (string, error)
}

// NewBuiltinTools 创建内置工具
func NewBuiltinTools() *BuiltinTools {
	return &BuiltinTools{
		calculator: func(op string, v1, v2 int) (int, error) {
			switch op {
			case "add", "加", "+":
				return v1 + v2, nil
			case "subtract", "减", "-":
				return v1 - v2, nil
			case "multiply", "乘", "*":
				return v1 * v2, nil
			case "divide", "除", "/":
				if v2 == 0 {
					return 0, fmt.Errorf("division by zero")
				}
				return v1 / v2, nil
			default:
				return 0, fmt.Errorf("unknown operation: %s", op)
			}
		},
		search: func(query string) (string, error) {
			// 模拟搜索结果
			return fmt.Sprintf("Search results for '%s':\n1. Result A\n2. Result B\n3. Result C", query), nil
		},
		weather: func(city string) (string, error) {
			weatherMap := map[string]string{
				"beijing":   "晴, 25°C",
				"shanghai":  "多云, 28°C",
				"guangzhou": "雷阵雨, 32°C",
				"shenzhen":  "晴, 31°C",
				"hangzhou":  "晴, 26°C",
			}
			w := strings.ToLower(city)
			if result, ok := weatherMap[w]; ok {
				return fmt.Sprintf("%s: %s", city, result), nil
			}
			return fmt.Sprintf("%s: 天气数据未知", city), nil
		},
	}
}

// Execute 实现 ToolExecutor 接口
func (b *BuiltinTools) Execute(ctx context.Context, toolName string, input map[string]any) (string, error) {
	switch toolName {
	case "calculator":
		op, _ := input["operation"].(string)
		v1, _ := input["value1"].(float64)
		v2, _ := input["value2"].(float64)
		result, err := b.calculator(op, int(v1), int(v2))
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("%d", result), nil

	case "search":
		query, _ := input["query"].(string)
		return b.search(query)

	case "weather":
		city, _ := input["city"].(string)
		return b.weather(city)

	default:
		return "", fmt.Errorf("unknown tool: %s", toolName)
	}
}
