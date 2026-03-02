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

// LlamaIndexClient LlamaIndex 桥接接口，支持 Agent/ChatEngine 调用。
type LlamaIndexClient interface {
	// Invoke 执行 Agent/ChatEngine，返回最终结果
	Invoke(ctx context.Context, input map[string]any) (map[string]any, error)
	// Stream 流式执行，onChunk 每收到一个 chunk 调用一次
	Stream(ctx context.Context, input map[string]any, onChunk func(chunk map[string]any) error) error
	// GetState 获取当前 Agent 状态（如果支持）
	GetState(ctx context.Context, sessionID string) (map[string]any, error)
}

// LlamaIndexErrorCode LlamaIndex error 分类。
type LlamaIndexErrorCode string

const (
	LlamaIndexErrorRetryable LlamaIndexErrorCode = "retryable"
	LlamaIndexErrorPermanent LlamaIndexErrorCode = "permanent"
	LlamaIndexErrorWait      LlamaIndexErrorCode = "wait"
)

// LlamaIndexError LlamaIndex 适配层 error；Runner/Adapter 可据此映射到 StepResultType 或等待 signal。
type LlamaIndexError struct {
	Code           LlamaIndexErrorCode
	Message        string
	CorrelationKey string
	Reason         string
	Err            error
}

func (e *LlamaIndexError) Error() string {
	if e == nil {
		return "llamaindex error"
	}
	if e.Message != "" {
		return e.Message
	}
	if e.Err != nil {
		return e.Err.Error()
	}
	return "llamaindex error"
}

func (e *LlamaIndexError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}

// mapLlamaIndexError 将 LlamaIndex 错误映射为适配层错误
func mapLlamaIndexError(taskID string, err error) *LlamaIndexError {
	if err == nil {
		return nil
	}
	// 根据错误类型判断是否可重试
	// 这里可以根据实际的 LlamaIndex 错误类型进行映射
	return &LlamaIndexError{
		Code:    LlamaIndexErrorPermanent,
		Message: err.Error(),
		Err:     err,
	}
}

// LlamaIndexNodeAdapter 将 llamaindex 型 TaskNode 转为 DAG 节点；支持 command 事件、effect 存储与 error 分类映射。
type LlamaIndexNodeAdapter struct {
	Client           LlamaIndexClient
	CommandEventSink CommandEventSink
	EffectStore      EffectStore
}

func (a *LlamaIndexNodeAdapter) runNode(ctx context.Context, taskID string, cfg map[string]any, p *AgentDAGPayload) (*AgentDAGPayload, error) {
	if a.Client == nil {
		return nil, fmt.Errorf("LlamaIndexNodeAdapter: Client not configured")
	}
	if p == nil {
		p = &AgentDAGPayload{}
	}
	if p.Results == nil {
		p.Results = make(map[string]any)
	}
	jobID := JobIDFromContext(ctx)

	// Effect Store replay: 如果已有执行结果，直接注入
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

	input := map[string]any{
		"goal":   p.Goal,
		"config": cfg,
	}
	if len(p.Results) > 0 {
		input["results"] = p.Results
	}
	if cfg != nil {
		if v, ok := cfg["input"].(map[string]any); ok {
			for k, val := range v {
				input[k] = val
			}
		}
	}

	if a.CommandEventSink != nil && jobID != "" {
		inputBytes, _ := json.Marshal(input)
		_ = a.CommandEventSink.AppendCommandEmitted(ctx, jobID, taskID, taskID, "llamaindex", inputBytes)
	}

	out, err := a.Client.Invoke(ctx, input)
	if err != nil {
		if mapped := mapLlamaIndexError(taskID, err); mapped != nil {
			return nil, mapped
		}
		return nil, err
	}
	if out == nil {
		out = map[string]any{}
	}
	outputBytes, _ := json.Marshal(out)

	// Effect Store: 记录执行结果供 replay
	if a.EffectStore != nil && jobID != "" {
		inputBytes, _ := json.Marshal(input)
		_ = a.EffectStore.PutEffect(ctx, &EffectRecord{
			JobID:     jobID,
			CommandID: taskID,
			Kind:      EffectKindTool,
			Input:     inputBytes,
			Output:    outputBytes,
			Metadata:  map[string]any{"adapter": "llamaindex"},
		})
	}

	// Command committed: 标记执行完成
	if a.CommandEventSink != nil && jobID != "" {
		_ = a.CommandEventSink.AppendCommandCommitted(ctx, jobID, taskID, taskID, outputBytes, "")
	}

	p.Results[taskID] = out
	return p, nil
}

// ToDAGNode 实现 NodeAdapter
func (a *LlamaIndexNodeAdapter) ToDAGNode(task *planner.TaskNode, agent *runtime.Agent) (*compose.Lambda, error) {
	if a.Client == nil {
		return nil, fmt.Errorf("LlamaIndexNodeAdapter: Client not configured")
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
func (a *LlamaIndexNodeAdapter) ToNodeRunner(task *planner.TaskNode, agent *runtime.Agent) (NodeRunner, error) {
	if a.Client == nil {
		return nil, fmt.Errorf("LlamaIndexNodeAdapter: Client not configured")
	}
	taskID, cfg := task.ID, task.Config
	if cfg == nil {
		cfg = make(map[string]any)
	}
	return func(ctx context.Context, p *AgentDAGPayload) (*AgentDAGPayload, error) {
		return a.runNode(ctx, taskID, cfg, p)
	}, nil
}
