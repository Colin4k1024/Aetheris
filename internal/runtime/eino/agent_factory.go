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
	"sync"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/compose"

	"rag-platform/pkg/config"
)

// AgentBuildConfig 单个 Agent 的构建配置（用于 AgentFactory）
type AgentBuildConfig struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Instruction string   `json:"instruction"`
	Type        string   `json:"type"`      // react, conversation, workflow
	Tools       []string `json:"tools"`     // 工具名列表（从 Registry 过滤）；空 = 全部
	MaxSteps    int      `json:"max_steps"` // ReAct 最大步数
	Streaming   bool     `json:"streaming"` // 是否启用流式
}

// AgentFactory 基于 Eino ADK 的 Agent 工厂：从配置 + 工具注册表 + LLM 构建 Runner。
// 所有 Agent 构建都经由此工厂，Aetheris 不再维护自定义 Plan→Execute 循环。
type AgentFactory struct {
	mu       sync.RWMutex
	engine   *Engine
	registry RuntimeToolRegistry
	bridge   *RegistryToolBridge

	// 已创建的 Runner 缓存
	runners map[string]*adk.Runner
}

// NewAgentFactory 创建 Agent 工厂
func NewAgentFactory(engine *Engine, registry RuntimeToolRegistry) *AgentFactory {
	return &AgentFactory{
		engine:   engine,
		registry: registry,
		bridge:   NewRegistryToolBridge(registry),
		runners:  make(map[string]*adk.Runner),
	}
}

// CreateAgent 根据配置创建 Agent Runner（带缓存）
func (f *AgentFactory) CreateAgent(ctx context.Context, cfg AgentBuildConfig) (*adk.Runner, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	if r, ok := f.runners[cfg.Name]; ok {
		return r, nil
	}

	runner, err := f.buildRunner(ctx, cfg, nil)
	if err != nil {
		return nil, err
	}
	f.runners[cfg.Name] = runner
	return runner, nil
}

// CreateAgentWithCheckpoint 创建带 Checkpoint 的 Agent Runner
func (f *AgentFactory) CreateAgentWithCheckpoint(ctx context.Context, cfg AgentBuildConfig, cps adk.CheckPointStore) (*adk.Runner, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	runner, err := f.buildRunner(ctx, cfg, cps)
	if err != nil {
		return nil, err
	}
	// checkpoint 版不缓存（每个 Job 可能有不同的 CheckpointStore）
	return runner, nil
}

// GetOrCreateFromConfig 从 agents.yaml 配置批量创建所有 Agent
func (f *AgentFactory) GetOrCreateFromConfig(ctx context.Context, agentsCfg *config.AgentsConfig) error {
	if agentsCfg == nil {
		return nil
	}
	for name, ac := range agentsCfg.Agents {
		cfg := AgentBuildConfig{
			Name:        name,
			Description: ac.Description,
			Instruction: ac.SystemPrompt,
			Type:        ac.Type,
			Tools:       ac.Tools,
			MaxSteps:    ac.MaxIterations,
			Streaming:   true,
		}
		if _, err := f.CreateAgent(ctx, cfg); err != nil {
			return fmt.Errorf("create agent %q failed: %w", name, err)
		}
	}
	return nil
}

// GetRunner 获取已创建的 Runner
func (f *AgentFactory) GetRunner(name string) (*adk.Runner, bool) {
	f.mu.RLock()
	defer f.mu.RUnlock()
	r, ok := f.runners[name]
	return r, ok
}

// ListAgents 返回所有已创建的 Agent 名称
func (f *AgentFactory) ListAgents() []string {
	f.mu.RLock()
	defer f.mu.RUnlock()
	names := make([]string, 0, len(f.runners))
	for n := range f.runners {
		names = append(names, n)
	}
	return names
}

func (f *AgentFactory) buildRunner(ctx context.Context, cfg AgentBuildConfig, cps adk.CheckPointStore) (*adk.Runner, error) {
	// 1. 获取 ChatModel
	chatModel, err := f.engine.CreateChatModel(ctx)
	if err != nil {
		return nil, fmt.Errorf("create chat model for agent %q: %w", cfg.Name, err)
	}

	// 2. 收集工具（默认 = Registry 全部 + Engine 内置；指定 tools = 过滤子集）
	einoTools := f.collectTools(cfg.Tools)

	// 3. 构建 ChatModelAgent 配置
	instruction := cfg.Instruction
	if instruction == "" {
		instruction = "你是一个有帮助的 AI 助手。"
	}

	agentCfg := &adk.ChatModelAgentConfig{
		Name:        cfg.Name,
		Description: cfg.Description,
		Instruction: instruction,
		ToolsConfig: adk.ToolsConfig{
			ToolsNodeConfig: compose.ToolsNodeConfig{
				Tools: einoTools,
			},
		},
	}
	if chatModel != nil {
		agentCfg.Model = chatModel.(model.ToolCallingChatModel)
	}

	// 4. 创建 Agent 和 Runner
	agent, err := adk.NewChatModelAgent(ctx, agentCfg)
	if err != nil {
		return nil, fmt.Errorf("create eino agent %q: %w", cfg.Name, err)
	}

	streaming := cfg.Streaming
	runnerCfg := adk.RunnerConfig{
		Agent:           agent,
		EnableStreaming: streaming,
	}
	if cps != nil {
		runnerCfg.CheckPointStore = cps
	}

	return adk.NewRunner(ctx, runnerCfg), nil
}

// collectTools 收集工具列表：将 Registry 中的工具 + Engine 内置工具合并
func (f *AgentFactory) collectTools(toolNames []string) []tool.BaseTool {
	// Engine 内置工具（retriever, generator, 文档系列）
	builtinTools := GetDefaultTools(f.engine)

	// Registry 工具（native + MCP）通过 bridge 转为 Eino tools
	registryTools := f.bridge.EinoTools()

	// 合并并去重（以名字为 key）
	seen := make(map[string]bool)
	var merged []tool.BaseTool

	addTool := func(t tool.BaseTool) {
		info, err := t.Info(context.Background())
		if err != nil {
			return
		}
		if seen[info.Name] {
			return
		}
		seen[info.Name] = true
		merged = append(merged, t)
	}

	// 如果指定了工具列表，则只选取匹配的
	if len(toolNames) > 0 {
		wanted := make(map[string]bool, len(toolNames))
		for _, n := range toolNames {
			wanted[n] = true
		}

		for _, t := range builtinTools {
			info, err := t.Info(context.Background())
			if err != nil {
				continue
			}
			if wanted[info.Name] {
				addTool(t)
			}
		}
		for _, t := range registryTools {
			info, err := t.Info(context.Background())
			if err != nil {
				continue
			}
			if wanted[info.Name] {
				addTool(t)
			}
		}
	} else {
		// 全部工具
		for _, t := range builtinTools {
			addTool(t)
		}
		for _, t := range registryTools {
			addTool(t)
		}
	}

	return merged
}
