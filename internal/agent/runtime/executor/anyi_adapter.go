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

// AnyiClient Anyi 桥接接口
type AnyiClient interface {
	CreateAgent(ctx context.Context, config map[string]any) (interface {
		Run(ctx context.Context, input string) (string, error)
	}, error)
}

// AnyiNodeAdapter 将 Anyi 型 TaskNode 转为 DAG 节点
type AnyiNodeAdapter struct {
	Client           AnyiClient
	CommandEventSink CommandEventSink
	EffectStore      EffectStore
}

func (a *AnyiNodeAdapter) runNode(ctx context.Context, taskID string, cfg map[string]any, p *AgentDAGPayload) (*AgentDAGPayload, error) {
	if a.Client == nil {
		return nil, fmt.Errorf("AnyiNodeAdapter: Client not configured")
	}
	if p == nil {
		p = &AgentDAGPayload{}
	}
	if p.Results == nil {
		p.Results = make(map[string]any)
	}
	jobID := JobIDFromContext(ctx)

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

	agent, err := a.Client.CreateAgent(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("AnyiNodeAdapter: failed to create agent: %w", err)
	}

	if a.CommandEventSink != nil && jobID != "" {
		inputBytes, _ := json.Marshal(map[string]any{"input": p.Goal})
		_ = a.CommandEventSink.AppendCommandEmitted(ctx, jobID, taskID, taskID, "anyi", inputBytes)
	}

	output, err := agent.Run(ctx, p.Goal)
	if err != nil {
		if p.Results == nil {
			p.Results = make(map[string]any)
		}
		p.Results[taskID] = map[string]any{"error": err.Error()}
		return nil, err
	}

	result := map[string]any{"output": output}

	if a.EffectStore != nil && jobID != "" {
		inputBytes, _ := json.Marshal(map[string]any{"input": p.Goal})
		outputBytes, _ := json.Marshal(result)
		_ = a.EffectStore.PutEffect(ctx, &EffectRecord{
			JobID: jobID, CommandID: taskID, Kind: EffectKindTool,
			Input: inputBytes, Output: outputBytes,
			Metadata: map[string]any{"adapter": "anyi"},
		})
	}

	if a.CommandEventSink != nil && jobID != "" {
		outputBytes, _ := json.Marshal(result)
		_ = a.CommandEventSink.AppendCommandCommitted(ctx, jobID, taskID, taskID, outputBytes, "")
	}

	p.Results[taskID] = result
	return p, nil
}

func (a *AnyiNodeAdapter) ToDAGNode(task *planner.TaskNode, _ *runtime.Agent) (*compose.Lambda, error) {
	taskID := task.ID
	cfg := task.Config
	if cfg == nil {
		cfg = map[string]any{}
	}
	return compose.InvokableLambda[*AgentDAGPayload, *AgentDAGPayload](func(ctx context.Context, p *AgentDAGPayload) (*AgentDAGPayload, error) {
		return a.runNode(ctx, taskID, cfg, p)
	}), nil
}

func (a *AnyiNodeAdapter) ToNodeRunner(task *planner.TaskNode, _ *runtime.Agent) (NodeRunner, error) {
	taskID := task.ID
	cfg := task.Config
	if cfg == nil {
		cfg = map[string]any{}
	}
	return func(ctx context.Context, p *AgentDAGPayload) (*AgentDAGPayload, error) {
		return a.runNode(ctx, taskID, cfg, p)
	}, nil
}
