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
	"testing"

	"rag-platform/pkg/config"
)

func TestNewAgentFactory(t *testing.T) {
	reg := &mockRegistry{tools: []RuntimeTool{
		&mockRuntimeTool{name: "search", desc: "搜索"},
	}}

	factory := NewAgentFactory(nil, reg)
	if factory == nil {
		t.Fatal("factory should not be nil")
	}
	if factory.bridge == nil {
		t.Error("bridge should not be nil")
	}
}

func TestAgentFactory_ListAgents_Empty(t *testing.T) {
	factory := NewAgentFactory(nil, &mockRegistry{})
	agents := factory.ListAgents()
	if len(agents) != 0 {
		t.Errorf("expected 0 agents, got %d", len(agents))
	}
}

func TestAgentFactory_GetRunner_NotFound(t *testing.T) {
	factory := NewAgentFactory(nil, &mockRegistry{})
	_, ok := factory.GetRunner("nonexistent")
	if ok {
		t.Error("expected not found")
	}
}

func TestAgentFactory_GetOrCreateFromConfig_NilConfig(t *testing.T) {
	factory := NewAgentFactory(nil, &mockRegistry{})
	err := factory.GetOrCreateFromConfig(nil, nil)
	if err != nil {
		t.Errorf("nil config should not error: %v", err)
	}
}

func TestAgentFactory_GetOrCreateFromConfig_EmptyAgents(t *testing.T) {
	factory := NewAgentFactory(nil, &mockRegistry{})
	cfg := &config.AgentsConfig{
		Agents: map[string]config.AgentDefConfig{},
	}
	err := factory.GetOrCreateFromConfig(nil, cfg)
	if err != nil {
		t.Errorf("empty agents should not error: %v", err)
	}
}

func TestAgentFactory_CollectTools(t *testing.T) {
	reg := &mockRegistry{tools: []RuntimeTool{
		&mockRuntimeTool{name: "search", desc: "搜索"},
		&mockRuntimeTool{name: "calculator", desc: "计算器"},
	}}

	factory := NewAgentFactory(nil, reg)

	// 全部工具
	all := factory.collectTools(nil)
	// Should include both registry tools + engine defaults (placeholder)
	if len(all) == 0 {
		t.Error("expected tools, got 0")
	}

	// 指定工具子集
	subset := factory.collectTools([]string{"search"})
	found := false
	for _, tt := range subset {
		info, _ := tt.Info(nil)
		if info.Name == "search" {
			found = true
		}
		if info.Name == "calculator" {
			t.Error("calculator should not be included")
		}
	}
	if !found {
		t.Error("search tool should be included")
	}
}
