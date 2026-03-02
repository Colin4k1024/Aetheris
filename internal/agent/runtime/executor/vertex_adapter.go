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

// VertexClient Vertex AI Agent Engine 桥接接口。
type VertexClient interface {
	// CreateSession 创建新的 Agent Session
	CreateSession(ctx context.Context, agent string, sessionConfig map[string]any) (string, error)
	// Execute 执行 Agent，返回最终结果
	Execute(ctx context.Context, agent, sessionID string, input map[string]any) (map[string]any, error)
	// Stream 流式执行
	Stream(ctx context.Context, agent, sessionID string, input map[string]any, onChunk func(chunk map[string]any) error) error
	// GetSession 获取 Session 状态
	GetSession(ctx context.Context, agent, sessionID string) (map[string]any, error)
}

// VertexErrorCode Vertex error 分类。
type VertexErrorCode string

const (
	VertexErrorRetryable VertexErrorCode = "retryable"
	VertexErrorPermanent VertexErrorCode = "permanent"
	VertexErrorWait      VertexErrorCode = "wait"
)

// VertexError Vertex 适配层 error。
type VertexError struct {
	Code    VertexErrorCode
	Message string
	Err     error
}

func (e *VertexError) Error() string {
	if e == nil {
		return "vertex error"
	}
	if e.Message != "" {
		return e.Message
	}
	if e.Err != nil {
		return e.Err.Error()
	}
	return "vertex error"
}

func (e *VertexError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}

// mapVertexError 将 Vertex 错误映射为适配层错误
func mapVertexError(taskID string, err error) *VertexError {
	if err == nil {
		return nil
	}
	return &VertexError{
		Code:    VertexErrorPermanent,
		Message: err.Error(),
		Err:     err,
	}
}

// VertexNodeAdapter 将 vertex 型 TaskNode 转为 DAG 节点。
type VertexNodeAdapter struct {
	Client           VertexClient
	CommandEventSink CommandEventSink
	EffectStore      EffectStore
}

func (a *VertexNodeAdapter) runNode(ctx context.Context, taskID string, cfg map[string]any, p *AgentDAGPayload) (*AgentDAGPayload, error) {
	if a.Client == nil {
		return nil, fmt.Errorf("VertexNodeAdapter: Client not configured")
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

	agent, _ := cfg["agent"].(string)
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
		_ = a.CommandEventSink.AppendCommandEmitted(ctx, jobID, taskID, taskID, "vertex", inputBytes)
	}

	var out map[string]any
	var err error

	// 如果没有 session，先创建
	if sessionID == "" {
		sessionID, err = a.Client.CreateSession(ctx, agent, nil)
		if err != nil {
			if mapped := mapVertexError(taskID, err); mapped != nil {
				return nil, mapped
			}
			return nil, err
		}
	}

	out, err = a.Client.Execute(ctx, agent, sessionID, input)
	if err != nil {
		if mapped := mapVertexError(taskID, err); mapped != nil {
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
			Metadata:  map[string]any{"adapter": "vertex", "session_id": sessionID},
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
func (a *VertexNodeAdapter) ToDAGNode(task *planner.TaskNode, agent *runtime.Agent) (*compose.Lambda, error) {
	if a.Client == nil {
		return nil, fmt.Errorf("VertexNodeAdapter: Client not configured")
	}
	taskID, cfg := task.ID, task.Config
	if cfg == nil {
		cfg = make(map[string]any)
	}
	return compose.InvokableLambda[*AgentDAGPayload, *AgentDAGPayload](func(ctx context.Context, p *AgentDAGPayload) (*AgentDAGPayload, error) {
		return a.runNode(ctx, taskID, cfg, p)
	}), nil
}

// ToNodeRunner 实现 NodeAdapter
func (a *VertexNodeAdapter) ToNodeRunner(task *planner.TaskNode, agent *runtime.Agent) (NodeRunner, error) {
	if a.Client == nil {
		return nil, fmt.Errorf("VertexNodeAdapter: Client not configured")
	}
	taskID, cfg := task.ID, task.Config
	if cfg == nil {
		cfg = make(map[string]any)
	}
	return func(ctx context.Context, p *AgentDAGPayload) (*AgentDAGPayload, error) {
		return a.runNode(ctx, taskID, cfg, p)
	}, nil
}
