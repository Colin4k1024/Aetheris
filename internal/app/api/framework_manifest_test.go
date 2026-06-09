package api

import (
	"testing"

	"github.com/Colin4k1024/Aetheris/v2/internal/agent/planner"
	"github.com/Colin4k1024/Aetheris/v2/pkg/config"
)

func TestFrameworkManifestToTaskGraph(t *testing.T) {
	agent := config.AgentDefConfig{
		Type: "langchain",
		External: config.AgentExternalConfig{
			Mode: "embedded",
			URL:  "http://framework-agent:9000",
		},
	}
	manifest := &FrameworkManifest{
		SchemaVersion: frameworkManifestSchemaV1,
		Name:          "research_agent",
		Framework:     "langchain",
		Nodes: []FrameworkManifestNode{
			{ID: "load", Kind: "remote_callable", Callable: "load_question"},
			{ID: "reason", Kind: "runtime_llm", Config: map[string]any{"prompt_key": "load"}},
			{ID: "search", Kind: "runtime_tool", ToolName: "knowledge.search"},
			{ID: "approval", Kind: "approval"},
			{ID: "final", Kind: "remote_callable", Callable: "final_answer"},
		},
		Edges: []FrameworkManifestEdge{
			{From: "load", To: "reason"},
			{From: "reason", To: "search"},
			{From: "search", To: "approval"},
			{From: "approval", To: "final"},
		},
	}
	graph, err := FrameworkManifestToTaskGraph("research_agent", agent, manifest)
	if err != nil {
		t.Fatalf("FrameworkManifestToTaskGraph returned error: %v", err)
	}
	if len(graph.Nodes) != 5 {
		t.Fatalf("expected 5 nodes, got %d", len(graph.Nodes))
	}
	if graph.Nodes[0].Type != planner.NodeFrameworkCallable {
		t.Fatalf("expected remote_callable to map to framework_callable, got %s", graph.Nodes[0].Type)
	}
	if graph.Nodes[1].Type != planner.NodeLLM {
		t.Fatalf("expected runtime_llm to map to llm, got %s", graph.Nodes[1].Type)
	}
	if graph.Nodes[2].Type != planner.NodeTool || graph.Nodes[2].ToolName != "knowledge.search" {
		t.Fatalf("expected runtime_tool to map to knowledge.search, got %+v", graph.Nodes[2])
	}
	if graph.Nodes[3].Type != planner.NodeApproval {
		t.Fatalf("expected approval to map to approval, got %s", graph.Nodes[3].Type)
	}
	if len(graph.Edges) != 4 {
		t.Fatalf("expected 4 edges, got %d", len(graph.Edges))
	}
}

func TestValidateFrameworkManifestRejectsUnknownKind(t *testing.T) {
	agent := config.AgentDefConfig{Type: "langchain", External: config.AgentExternalConfig{Mode: "embedded"}}
	err := ValidateFrameworkManifest("bad_agent", agent, &FrameworkManifest{
		SchemaVersion: frameworkManifestSchemaV1,
		Name:          "bad_agent",
		Framework:     "langchain",
		Nodes:         []FrameworkManifestNode{{ID: "n1", Kind: "mystery"}},
	})
	if err == nil {
		t.Fatalf("expected unknown kind validation error")
	}
}

func TestValidateFrameworkManifestRejectsMissingInputOutputNodes(t *testing.T) {
	agent := config.AgentDefConfig{Type: "langchain", External: config.AgentExternalConfig{
		Mode: "embedded",
		URL:  "http://framework-agent:9000",
	}}
	manifest := &FrameworkManifest{
		SchemaVersion: frameworkManifestSchemaV1,
		Name:          "bad_agent",
		Framework:     "langchain",
		InputNode:     "missing_input",
		OutputNode:    "final",
		Nodes: []FrameworkManifestNode{
			{ID: "final", Kind: "remote_callable", Callable: "final_answer"},
		},
	}

	if err := ValidateFrameworkManifest("bad_agent", agent, manifest); err == nil {
		t.Fatalf("expected missing input_node validation error")
	}

	manifest.InputNode = "final"
	manifest.OutputNode = "missing_output"
	if err := ValidateFrameworkManifest("bad_agent", agent, manifest); err == nil {
		t.Fatalf("expected missing output_node validation error")
	}
}

func TestValidateFrameworkManifestRejectsUnsafeNodeID(t *testing.T) {
	agent := config.AgentDefConfig{Type: "langchain", External: config.AgentExternalConfig{
		Mode: "embedded",
		URL:  "http://framework-agent:9000",
	}}
	err := ValidateFrameworkManifest("bad_agent", agent, &FrameworkManifest{
		SchemaVersion: frameworkManifestSchemaV1,
		Name:          "bad_agent",
		Framework:     "langchain",
		Nodes: []FrameworkManifestNode{
			{ID: "nested/load", Kind: "remote_callable", Callable: "load_question"},
		},
	})
	if err == nil {
		t.Fatalf("expected unsafe node id validation error")
	}
}
