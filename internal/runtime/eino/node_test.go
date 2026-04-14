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
	"testing"

	"github.com/Colin4k1024/Aetheris/v2/internal/pipeline/common"
)

type mockStage struct {
	name   string
	input  interface{}
	output interface{}
	err    error
}

func (m *mockStage) Name() string {
	return m.name
}

func (m *mockStage) Execute(ctx *common.PipelineContext, input interface{}) (interface{}, error) {
	m.input = input
	return m.output, m.err
}

func (m *mockStage) Validate(input interface{}) error {
	return nil
}

func TestNewNode(t *testing.T) {
	stage := &mockStage{name: "test"}
	node := NewNode("node1", stage)
	if node == nil {
		t.Fatal("expected non-nil node")
	}
	if node.Name() != "node1" {
		t.Errorf("expected node1, got %s", node.Name())
	}
}

func TestNode_Name(t *testing.T) {
	stage := &mockStage{name: "test"}
	node := NewNode("mynode", stage)
	if node.Name() != "mynode" {
		t.Errorf("expected mynode, got %s", node.Name())
	}
}

func TestNode_Stage(t *testing.T) {
	stage := &mockStage{name: "test"}
	node := NewNode("node1", stage)
	if node.Stage() != stage {
		t.Error("Stage() should return the same stage")
	}
}

func TestNode_Run(t *testing.T) {
	stage := &mockStage{
		name:   "test",
		output: "processed",
	}
	node := NewNode("node1", stage)
	ctx := context.Background()
	pipeCtx := &common.PipelineContext{}

	result, err := node.Run(ctx, pipeCtx, "input")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if result != "processed" {
		t.Errorf("expected processed, got %v", result)
	}
}

func TestNode_Run_NilStage(t *testing.T) {
	node := NewNode("node1", nil)
	ctx := context.Background()
	pipeCtx := &common.PipelineContext{}

	result, err := node.Run(ctx, pipeCtx, "input")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if result != "input" {
		t.Errorf("expected input, got %v", result)
	}
}

func TestNode_Run_Error(t *testing.T) {
	stage := &mockStage{
		name:   "test",
		output: nil,
		err:    context.DeadlineExceeded,
	}
	node := NewNode("node1", stage)
	ctx := context.Background()
	pipeCtx := &common.PipelineContext{}

	_, err := node.Run(ctx, pipeCtx, "input")
	if err != context.DeadlineExceeded {
		t.Errorf("expected DeadlineExceeded, got %v", err)
	}
}
