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

package runtime

import (
	"context"
	"sync"
	"time"

	"github.com/google/uuid"
)

// AgentStatus Agent 运行状态
type AgentStatus int

const (
	StatusIdle AgentStatus = iota
	StatusRunning
	StatusWaitingTool
	StatusSuspended
	StatusFailed
)

func (s AgentStatus) String() string {
	switch s {
	case StatusIdle:
		return "idle"
	case StatusRunning:
		return "running"
	case StatusWaitingTool:
		return "waiting_tool"
	case StatusSuspended:
		return "suspended"
	case StatusFailed:
		return "failed"
	default:
		return "unknown"
	}
}

// Agent 第一公民：具有状态、记忆、目标，可被调度执行的长期实体
type Agent struct {
	ID        string
	Name      string
	TenantID  string
	CreatedAt time.Time

	Session *Session
	Memory  MemoryProvider
	Planner PlannerProvider
	Tools   ToolsProvider

	Status AgentStatus
	mu     sync.RWMutex
}

// MemoryProvider 提供 Memory 能力（如 agent/memory.CompositeMemory）
type MemoryProvider interface {
	Recall(ctx interface{}, query string) (interface{}, error)
	Store(ctx interface{}, item interface{}) error
}

// PlannerProvider 提供规划能力（如 agent/planner.Planner）
type PlannerProvider interface {
	Plan(ctx interface{}, goal string, mem interface{}) (interface{}, error)
}

// ToolsProvider 提供工具注册表（如 agent/tools.Registry）
type ToolsProvider interface {
	Get(name string) (interface{}, bool)
	List() []interface{}
}

// NewAgent 创建新 Agent
func NewAgent(id, name string, session *Session, memory MemoryProvider, planner PlannerProvider, tools ToolsProvider) *Agent {
	now := time.Now()
	if id == "" {
		id = "agent-" + uuid.New().String()
	}
	if name == "" {
		name = id
	}
	return &Agent{
		ID:        id,
		Name:      name,
		CreatedAt: now,
		Session:   session,
		Memory:    memory,
		Planner:   planner,
		Tools:     tools,
		Status:    StatusIdle,
	}
}

// SetStatus 设置状态
func (a *Agent) SetStatus(s AgentStatus) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.Status = s
}

// GetStatus 读取状态
func (a *Agent) GetStatus() AgentStatus {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.Status
}

// Take 原子性地获取执行权：若状态为 Idle/Suspended 则转为 Running，返回 true
// 若状态为 Running/WaitingTool/Failed，返回 false（已被占用）
// RTN-05: 实现 Agent 并发模型的 Take/Release 语义
func (a *Agent) Take() bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	switch a.Status {
	case StatusIdle, StatusSuspended:
		a.Status = StatusRunning
		return true
	default:
		return false
	}
}

// Release 释放执行权：将状态从 Running/WaitingTool 转回 Idle
// RTN-05: 实现 Agent 并发模型的 Take/Release 语义
func (a *Agent) Release() {
	a.mu.Lock()
	defer a.mu.Unlock()
	switch a.Status {
	case StatusRunning, StatusWaitingTool:
		a.Status = StatusIdle
	}
}

// TakeWithWait 尝试获取执行权，若被占用则等待直到获取或上下文取消
// 返回是否成功获取（ctx 取消时返回 false）
func (a *Agent) TakeWithWait(ctx context.Context) bool {
	for {
		if a.Take() {
			return true
		}
		select {
		case <-ctx.Done():
			return false
		case <-time.After(10 * time.Millisecond):
			// 短暂等待后重试，避免 busy loop
		}
	}
}
