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
)

func TestCreateWorkflow(t *testing.T) {
	wf := CreateWorkflow("test-workflow", "A test workflow")
	if wf == nil {
		t.Fatal("expected non-nil workflow")
	}
}

func TestWorkflow_AddNode(t *testing.T) {
	wf := CreateWorkflow("test", "desc")

	// Add validate node
	err := wf.AddNode("validator", "validate", &NodeConfig{Name: "validator", Type: "validate"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Add format node
	err = wf.AddNode("formatter", "format", &NodeConfig{Name: "formatter", Type: "format"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Add generate node should fail without chatModel
	err = wf.AddNode("generator", "generate", &NodeConfig{Name: "generator", Type: "generate"})
	if err == nil {
		t.Error("expected error for generate node without chatModel")
	}

	// Add unknown node type
	err = wf.AddNode("unknown", "unknown-type", &NodeConfig{Name: "unknown", Type: "unknown-type"})
	if err == nil {
		t.Error("expected error for unknown node type")
	}
}

func TestWorkflow_AddEdge(t *testing.T) {
	wf := CreateWorkflow("test", "desc")

	// Add nodes first
	wf.AddNode("node1", "validate", &NodeConfig{Name: "node1", Type: "validate"})
	wf.AddNode("node2", "format", &NodeConfig{Name: "node2", Type: "format"})

	// Add edge
	err := wf.AddEdge("node1", "node2")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Note: AddEdge may not validate node existence, so we just test happy path
}

func TestInput_Output(t *testing.T) {
	input := Input{Query: "test query"}
	if input.Query != "test query" {
		t.Errorf("expected test query, got %s", input.Query)
	}

	output := Output{Result: "test result"}
	if output.Result != "test result" {
		t.Errorf("expected test result, got %s", output.Result)
	}
}

func TestWorkflowConfig(t *testing.T) {
	cfg := WorkflowConfig{
		Name:        "test-config",
		Description: "test description",
	}
	if cfg.Name != "test-config" {
		t.Errorf("expected test-config, got %s", cfg.Name)
	}
	if cfg.Description != "test description" {
		t.Errorf("expected test description, got %s", cfg.Description)
	}
}

func TestNodeConfig(t *testing.T) {
	cfg := NodeConfig{
		Name:        "test-node",
		Type:        "validate",
		Description: "test node description",
	}
	if cfg.Name != "test-node" {
		t.Errorf("expected test-node, got %s", cfg.Name)
	}
	if cfg.Type != "validate" {
		t.Errorf("expected validate, got %s", cfg.Type)
	}
}
