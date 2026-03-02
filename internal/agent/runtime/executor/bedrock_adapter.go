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

	"rag-platform/internal/agent/planner"
	"rag-platform/internal/agent/runtime"
)

// BedrockClient AWS Bedrock Agents 桥接接口。
type BedrockClient interface {
	// CreateAgentSession 创建新的 Agent Session
	CreateAgentSession(ctx context.Context, agentID string, sessionConfig map[string]any) (string, error)
	// Invoke 同步调用 Agent
	Invoke(ctx context.Context, agentID, sessionID string, input map[string]any) (map[string]any, error)
	// InvokeWithResponseStream 流式调用 Agent
	InvokeWithResponseStream(ctx context.Context, agentID, sessionID string, input map[string]any, onChunk func(chunk map[string]any) error) error
	// GetSession 获取 Session 状态
	GetAgentSession(ctx context.Context, agentID, sessionID string) (map[string]any, error)
}

// BedrockErrorCode Bedrock error 分类。
type BedrockErrorCode string

const (
	BedrockErrorRetryable BedrockErrorCode = "retryable"
	BedrockErrorPermanent BedrockErrorCode = "permanent"
	BedrockErrorWait      BedrockErrorCode = "wait"
)

// BedrockError Bedrock 适配层 error。
type BedrockError struct {
	Code    BedrockErrorCode
	Message string
	Err     error
}

func (e *BedrockError) Error() string {
	if e == nil {
		return "bedrock error"
	}
	if e.Message != "" {
		return e.Message
	}
	if e.Err != nil {
		return e.Err.Error()
	}
	return "bedrock error"
}

func (e *BedrockError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}

// mapBedrockError 将 Bedrock 错误映射为适配层错误
func mapBedrockError(taskID string, err error) *BedrockError {
	if err == nil {
		return nil
	}
	return &BedrockError{
		Code:    BedrockErrorPermanent,
		Message: err.Error(),
		Err:     err,
	}
}

// BedrockNodeAdapter 将 bedrock 型 TaskNode 转为 DAG 节点。
type BedrockNodeAdapter struct {
	Client           BedrockClient
	CommandEventSink CommandEventSink
	EffectStore      EffectStore
}

func (a *BedrockNodeAdapter) runNode(ctx context.Context, taskID string, cfg map[string]any, p *AgentDAGPayload) (*AgentDAGPayload, error) {
	if a.Client == nil {
		return nil, fmt.Errorf("BedrockNodeAdapter: Client not configured")
	}
	if p == nil {
		p = &AgentDAGPayload{}
	}
	if p.Results == nil {
		p.Results = make(map[string]any)
	}
	jobID := JobIDFromContext(ctx)

	// Effect Store replay
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

	agentID, _ := cfg["agent_id"].(string)
	sessionID, _ := cfg["session_id"].(string)

	input := map[string]any{
		"goal":   p.Goal,
		"config": cfg,
	}
	if len(p.Results) > 0 {
		input["results"] = p.Results
	}

	if a.CommandEventSink != nil && jobID != "" {
		inputBytes, _ := json.Marshal(input)
		_ = a.CommandEventSink.AppendCommandEmitted(ctx, jobID, taskID, taskID, "bedrock", inputBytes)
	}

	var out map[string]any
	var err error

	// 如果没有 session，先创建
	if sessionID == "" {
		sessionID, err = a.Client.CreateAgentSession(ctx, agentID, nil)
		if err != nil {
			if mapped := mapBedrockError(taskID, err); mapped != nil {
				return nil, mapped
			}
			return nil, err
		}
	}

	out, err = a.Client.Invoke(ctx, agentID, sessionID, input)
	if err != nil {
		if mapped := mapBedrockError(taskID, err); mapped != nil {
			return nil, mapped
		}
		return nil, err
	}
	if out == nil {
		out = map[string]any{}
	}

	outputBytes, _ := json.Marshal(out)

	// Effect Store
	if a.EffectStore != nil && jobID != "" {
		inputBytes, _ := json.Marshal(input)
		_ = a.EffectStore.PutEffect(ctx, &EffectRecord{
			JobID:     jobID,
			CommandID: taskID,
			Kind:      EffectKindTool,
			Input:     inputBytes,
			Output:    outputBytes,
			Metadata:  map[string]any{"adapter": "bedrock", "session_id": sessionID},
		})
	}

	// Command committed
	if a.CommandEventSink != nil && jobID != "" {
		_ = a.CommandEventSink.AppendCommandCommitted(ctx, jobID, taskID, taskID, outputBytes, "")
	}

	p.Results[taskID] = out
	return p, nil
}

// ToDAGNode 实现 NodeAdapter
func (a *BedrockNodeAdapter) ToDAGNode(task *planner.TaskNode, agent *runtime.Agent) (*compose.Lambda, error) {
	if a.Client == nil {
		return nil, fmt.Errorf("BedrockNodeAdapter: Client not configured")
	}
	taskID, cfg := task.ID, task.Config
	if cfg == nil {
		cfg = make(map[string]any)
	}
	return compose.InvokableLambda(func(ctx context.Context, p *AgentDAGPayload) (*AgentDAGPayload, error) {
		return a.runNode(ctx, taskID, cfg, p)
	}), nil
}

// ToNodeRunner 实现 NodeAdapter
func (a *BedrockNodeAdapter) ToNodeRunner(task *planner.TaskNode, agent *runtime.Agent) (NodeRunner, error) {
	if a.Client == nil {
		return nil, fmt.Errorf("BedrockNodeAdapter: Client not configured")
	}
	taskID, cfg := task.ID, task.Config
	if cfg == nil {
		cfg = make(map[string]any)
	}
	return func(ctx context.Context, p *AgentDAGPayload) (*AgentDAGPayload, error) {
		return a.runNode(ctx, taskID, cfg, p)
	}, nil
}
