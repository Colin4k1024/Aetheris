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

package eino

import (
	"context"
	"fmt"
	"time"

	"github.com/cloudwego/eino/adk"
)

// ContextManager 上下文管理器
type ContextManager struct {
	runners map[string]*adk.Runner
}

// NewContextManager 创建新的上下文管理器
func NewContextManager() *ContextManager {
	return &ContextManager{
		runners: make(map[string]*adk.Runner),
	}
}

// RegisterRunner 注册 Runner
func (cm *ContextManager) RegisterRunner(name string, runner *adk.Runner) {
	cm.runners[name] = runner
}

// GetRunner 获取 Runner
func (cm *ContextManager) GetRunner(name string) (*adk.Runner, error) {
	runner, exists := cm.runners[name]
	if !exists {
		return nil, fmt.Errorf("runner %s not found", name)
	}
	return runner, nil
}

// ExecuteQuery 执行查询
func (cm *ContextManager) ExecuteQuery(ctx context.Context, runnerName, query string) (chan *adk.AgentEvent, error) {
	r, err := cm.GetRunner(runnerName)
	if err != nil {
		return nil, err
	}

	iter := r.Query(ctx, query)
	eventCh := make(chan *adk.AgentEvent)

	go func() {
		defer close(eventCh)
		for {
			// Use non-blocking check for context cancellation
			select {
			case <-ctx.Done():
				return
			default:
			}

			event, ok := iter.Next()
			if !ok {
				return
			}

			// Use non-blocking send to avoid blocking on slow consumer
			select {
			case <-ctx.Done():
				return
			case eventCh <- event:
			}

			// Small sleep to avoid busy loop when no events are available
			// This balances responsiveness with CPU usage
			time.Sleep(10 * time.Millisecond)
		}
	}()

	return eventCh, nil
}

// ExecuteTool 执行工具。需通过已注册的 Runner 执行；未配置时返回错误，不返回模拟结果。
func (cm *ContextManager) ExecuteTool(ctx context.Context, runnerName, toolName, input string) (string, error) {
	runner, err := cm.GetRunner(runnerName)
	if err != nil {
		return "", err
	}
	if runner == nil {
		return "", fmt.Errorf("runner %q is not configured, cannot execute tool %q", runnerName, toolName)
	}
	// Runner 接口不支持直接执行单个工具；返回不支持错误
	return "", fmt.Errorf("direct tool execution via ContextManager is not supported; use Runner workflow instead (tool: %s)", toolName)
}
