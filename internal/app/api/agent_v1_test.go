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

package api

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Colin4k1024/Aetheris/v2/internal/agent/memory"
	"github.com/Colin4k1024/Aetheris/v2/internal/agent/planner"
	"github.com/Colin4k1024/Aetheris/v2/internal/agent/runtime"
	"github.com/Colin4k1024/Aetheris/v2/internal/agent/tools"
	"github.com/Colin4k1024/Aetheris/v2/pkg/config"
)

func TestMemoryProviderAdapter_Recall(t *testing.T) {
	mockMem := &mockMemoryForAdapter{}
	adapter := &memoryProviderAdapter{m: mockMem}

	result, err := adapter.Recall(context.Background(), "test query")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if result == nil {
		t.Error("expected non-nil result")
	}
}

func TestMemoryProviderAdapter_Recall_NonContext(t *testing.T) {
	mockMem := &mockMemoryForAdapter{}
	adapter := &memoryProviderAdapter{m: mockMem}

	// Pass non-context type
	result, err := adapter.Recall("not a context", "test query")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if result == nil {
		t.Error("expected non-nil result")
	}
}

func TestMemoryProviderAdapter_Store(t *testing.T) {
	mockMem := &mockMemoryForAdapter{}
	adapter := &memoryProviderAdapter{m: mockMem}

	item := memory.MemoryItem{
		Type:    "working",
		Content: "test",
		At:      time.Now(),
	}
	err := adapter.Store(context.Background(), item)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestMemoryProviderAdapter_Store_NonContext(t *testing.T) {
	mockMem := &mockMemoryForAdapter{}
	adapter := &memoryProviderAdapter{m: mockMem}

	item := memory.MemoryItem{
		Type:    "working",
		Content: "test",
		At:      time.Now(),
	}
	err := adapter.Store("not a context", item)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestMemoryProviderAdapter_Store_NonMemoryItem(t *testing.T) {
	mockMem := &mockMemoryForAdapter{}
	adapter := &memoryProviderAdapter{m: mockMem}

	err := adapter.Store(context.Background(), "not a memory item")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestPlanGoalForJobFuncWithExternalAgents(t *testing.T) {
	manager := runtime.NewManager()
	toolsReg := tools.NewRegistry()
	cfg := &config.AgentsConfig{Agents: map[string]config.AgentDefConfig{
		"customer_support_bot": {
			Type: "external_http",
			External: config.AgentExternalConfig{
				URL:     "http://customer-bot:9000/invoke",
				Timeout: "120s",
			},
		},
	}}
	if err := RegisterConfiguredAgents(context.Background(), manager, planner.NewRulePlanner(), toolsReg, cfg); err != nil {
		t.Fatalf("RegisterConfiguredAgents returned error: %v", err)
	}
	agent, _ := manager.Get(context.Background(), "customer_support_bot")
	if agent == nil {
		t.Fatalf("expected configured external agent to be registered with stable id")
	}

	planFn := PlanGoalForJobFuncWithExternalAgents(manager, planner.NewRulePlanner(), cfg)
	graph, err := planFn(context.Background(), "customer_support_bot", "hello")
	if err != nil {
		t.Fatalf("planFn returned error: %v", err)
	}
	if len(graph.Nodes) != 1 {
		t.Fatalf("expected one node, got %d", len(graph.Nodes))
	}
	node := graph.Nodes[0]
	if node.Type != planner.NodeTool || node.ToolName != ExternalAgentCallToolName {
		t.Fatalf("expected external_agent_call tool node, got type=%s tool=%s", node.Type, node.ToolName)
	}
	if node.Config["agent_id"] != "customer_support_bot" {
		t.Errorf("expected agent_id config, got %v", node.Config["agent_id"])
	}
	if node.Config["message"] != "hello" {
		t.Errorf("expected message config, got %v", node.Config["message"])
	}
}

func TestPlanGoalForJobFuncWithFrameworkAgentAlias(t *testing.T) {
	manager := runtime.NewManager()
	toolsReg := tools.NewRegistry()
	cfg := &config.AgentsConfig{Agents: map[string]config.AgentDefConfig{
		"research_graph": {
			Type: "langgraph",
			External: config.AgentExternalConfig{
				URL:     "http://langgraph-agent:9000/invoke",
				Timeout: "120s",
			},
		},
	}}
	if err := RegisterConfiguredAgents(context.Background(), manager, planner.NewRulePlanner(), toolsReg, cfg); err != nil {
		t.Fatalf("RegisterConfiguredAgents returned error: %v", err)
	}

	planFn := PlanGoalForJobFuncWithExternalAgents(manager, planner.NewRulePlanner(), cfg)
	graph, err := planFn(context.Background(), "research_graph", "research this")
	if err != nil {
		t.Fatalf("planFn returned error: %v", err)
	}
	if len(graph.Nodes) != 1 {
		t.Fatalf("expected one node, got %d", len(graph.Nodes))
	}
	node := graph.Nodes[0]
	if node.Type != planner.NodeTool || node.ToolName != ExternalAgentCallToolName {
		t.Fatalf("expected external_agent_call tool node, got type=%s tool=%s", node.Type, node.ToolName)
	}
	if node.Config["framework"] != "langgraph" {
		t.Errorf("expected framework=langgraph, got %v", node.Config["framework"])
	}
}

func TestPlanGoalForJobFuncWithEmbeddedFrameworkManifest(t *testing.T) {
	manager := runtime.NewManager()
	toolsReg := tools.NewRegistry()
	tmpDir := t.TempDir()
	manifestPath := filepath.Join(tmpDir, "agent.manifest.json")
	manifest := `{
		"schema_version":"aetheris.framework.v1",
		"name":"research_agent",
		"framework":"langchain",
		"input_node":"load_question",
		"output_node":"final_answer",
		"nodes":[
			{"id":"load_question","kind":"remote_callable","callable":"load_question","config":{}},
			{"id":"reason","kind":"runtime_llm","config":{"prompt_key":"load_question"}},
			{"id":"search","kind":"runtime_tool","tool_name":"knowledge.search","config":{"top_k":3}},
			{"id":"final_answer","kind":"remote_callable","callable":"final_answer","config":{}}
		],
		"edges":[
			{"from":"load_question","to":"reason"},
			{"from":"reason","to":"search"},
			{"from":"search","to":"final_answer"}
		]
	}`
	if err := os.WriteFile(manifestPath, []byte(manifest), 0644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}
	cfg := &config.AgentsConfig{Agents: map[string]config.AgentDefConfig{
		"research_agent": {
			Type: "langchain",
			External: config.AgentExternalConfig{
				Mode:         "embedded",
				URL:          "http://framework-agent:9000",
				ManifestPath: manifestPath,
			},
		},
	}}
	if err := RegisterConfiguredAgents(context.Background(), manager, planner.NewRulePlanner(), toolsReg, cfg); err != nil {
		t.Fatalf("RegisterConfiguredAgents returned error: %v", err)
	}

	planFn := PlanGoalForJobFuncWithExternalAgents(manager, planner.NewRulePlanner(), cfg)
	graph, err := planFn(context.Background(), "research_agent", "research this")
	if err != nil {
		t.Fatalf("planFn returned error: %v", err)
	}
	if len(graph.Nodes) != 4 {
		t.Fatalf("expected four manifest nodes, got %d", len(graph.Nodes))
	}
	if graph.Nodes[0].Type != planner.NodeFrameworkCallable {
		t.Errorf("expected first node framework_callable, got %s", graph.Nodes[0].Type)
	}
	if graph.Nodes[1].Type != planner.NodeLLM {
		t.Errorf("expected second node llm, got %s", graph.Nodes[1].Type)
	}
	if graph.Nodes[2].Type != planner.NodeTool || graph.Nodes[2].ToolName != "knowledge.search" {
		t.Errorf("expected runtime tool knowledge.search, got type=%s tool=%s", graph.Nodes[2].Type, graph.Nodes[2].ToolName)
	}
	if len(graph.Edges) != 3 {
		t.Errorf("expected three manifest edges, got %d", len(graph.Edges))
	}
}

// Mock implementations

type mockMemoryForAdapter struct {
	recallResult []memory.MemoryItem
}

func (m *mockMemoryForAdapter) Recall(ctx context.Context, query string) ([]memory.MemoryItem, error) {
	return []memory.MemoryItem{{Type: "working", Content: "test", At: time.Now()}}, nil
}

func (m *mockMemoryForAdapter) Store(ctx context.Context, item memory.MemoryItem) error {
	return nil
}
