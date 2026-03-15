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

package planner

import (
	"context"
	"testing"
)

type mockToolRunner struct{}

func (m *mockToolRunner) Execute(ctx context.Context, toolName string, input map[string]any) (string, error) {
	return "tool result", nil
}

type mockWorkflowRunner struct{}

func (m *mockWorkflowRunner) ExecuteWorkflow(ctx context.Context, name string, params map[string]any) (interface{}, error) {
	return "workflow result", nil
}

type mockLLMRunner struct{}

func (m *mockLLMRunner) Generate(ctx context.Context, prompt string) (string, error) {
	return "llm response", nil
}

func TestNewTaskGraphExecutor(t *testing.T) {
	executor := NewTaskGraphExecutor(&mockToolRunner{}, &mockWorkflowRunner{}, &mockLLMRunner{})

	if executor == nil {
		t.Fatal("executor should not be nil")
	}

	if executor.Tools == nil {
		t.Error("Tools should be set")
	}

	if executor.Workflow == nil {
		t.Error("Workflow should be set")
	}

	if executor.LLM == nil {
		t.Error("LLM should be set")
	}
}
