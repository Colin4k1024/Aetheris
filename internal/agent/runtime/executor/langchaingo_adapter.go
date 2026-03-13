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

	"github.com/cloudwego/eino/compose"
	"github.com/tmc/langchaingo/agents"
	"github.com/tmc/langchaingo/chains"
	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/schema"
	"github.com/tmc/langchaingo/tools"

	"rag-platform/internal/agent/planner"
	"rag-platform/internal/agent/runtime"
)

// LangChainGoClient LangChainGo 桥接接口
type LangChainGoClient interface {
	// CreateAgent 创建 agent 实例
	CreateAgent(ctx context.Context, config map[string]any) (agents.Agent, []tools.Tool, error)
	// GetLLM 获取 LLM 实例
	GetLLM(ctx context.Context, config map[string]any) (llms.LLM, error)
}

// LangChainGoNodeAdapter 将 langchaingo 型 TaskNode 转为 DAG 节点
type LangChainGoNodeAdapter struct {
	Client           LangChainGoClient
	CommandEventSink CommandEventSink
	EffectStore      EffectStore
}

func (a *LangChainGoNodeAdapter) runNode(ctx context.Context, taskID string, cfg map[string]any, p *AgentDAGPayload) (*AgentDAGPayload, error) {
	if a.Client == nil {
		return nil, fmt.Errorf("LangChainGoNodeAdapter: Client not configured")
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

	// 创建 agent 和工具列表
	agent, _, err := a.Client.CreateAgent(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("LangChainGoNodeAdapter: failed to create agent: %w", err)
	}

	// 构建输入
	input := map[string]any{
		"input": p.Goal,
	}
	if len(p.Results) > 0 {
		input["agent_scratchpad"] = p.Results
	}

	// 发送命令事件
	if a.CommandEventSink != nil && jobID != "" {
		inputBytes, _ := json.Marshal(input)
		_ = a.CommandEventSink.AppendCommandEmitted(ctx, jobID, taskID, taskID, "langchaingo", inputBytes)
	}

	// 创建 executor 并执行
	executor := agents.NewExecutor(agent)
	result, err := chains.Call(ctx, executor, input)
	if err != nil {
		// 映射错误
		if p.Results == nil {
			p.Results = make(map[string]any)
		}
		p.Results[taskID] = map[string]any{"error": err.Error()}
		return nil, err
	}

	// 提取输出
	output := ""
	if v, ok := result["output"]; ok {
		if s, ok := v.(string); ok {
			output = s
		} else {
			output = fmt.Sprintf("%v", v)
		}
	}

	agentResult := map[string]any{
		"output":             output,
		"intermediate_steps": result["intermediate_steps"],
	}

	// 存储 effect
	if a.EffectStore != nil && jobID != "" {
		inputBytes, _ := json.Marshal(input)
		outputBytes, _ := json.Marshal(agentResult)
		_ = a.EffectStore.PutEffect(ctx, &EffectRecord{
			JobID:     jobID,
			CommandID: taskID,
			Kind:      EffectKindTool,
			Input:     inputBytes,
			Output:    outputBytes,
			Metadata:  map[string]any{"adapter": "langchaingo"},
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

// ToDAGNode 实现 NodeAdapter
func (a *LangChainGoNodeAdapter) ToDAGNode(task *planner.TaskNode, _ *runtime.Agent) (*compose.Lambda, error) {
	taskID := task.ID
	cfg := task.Config
	if cfg == nil {
		cfg = map[string]any{}
	}
	return compose.InvokableLambda[*AgentDAGPayload, *AgentDAGPayload](func(ctx context.Context, p *AgentDAGPayload) (*AgentDAGPayload, error) {
		return a.runNode(ctx, taskID, cfg, p)
	}), nil
}

// ToNodeRunner 实现 NodeAdapter
func (a *LangChainGoNodeAdapter) ToNodeRunner(task *planner.TaskNode, _ *runtime.Agent) (NodeRunner, error) {
	taskID := task.ID
	cfg := task.Config
	if cfg == nil {
		cfg = map[string]any{}
	}
	return func(ctx context.Context, p *AgentDAGPayload) (*AgentDAGPayload, error) {
		return a.runNode(ctx, taskID, cfg, p)
	}, nil
}

// Ensure schema.AgentFinish is used to avoid import error
var _ = schema.AgentFinish{}
