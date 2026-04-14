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

package query

import (
	"testing"
	"time"

	"github.com/Colin4k1024/Aetheris/v2/internal/pipeline/common"
)

func TestNewResponder(t *testing.T) {
	r := NewResponder()
	if r == nil {
		t.Fatal("NewResponder should not return nil")
	}
	if r.Name() != "responder" {
		t.Errorf("expected 'responder', got '%s'", r.Name())
	}
}

func TestResponder_Validate(t *testing.T) {
	r := NewResponder()

	// Test nil input
	err := r.Validate(nil)
	if err == nil {
		t.Error("expected error for nil input")
	}

	// Test valid input
	err = r.Validate(&common.GenerationResult{})
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	// Test invalid type
	err = r.Validate("invalid")
	if err == nil {
		t.Error("expected error for invalid type")
	}
}

func TestResponder_Execute(t *testing.T) {
	r := NewResponder()

	input := &common.GenerationResult{
		Answer:      "test answer",
		References:  []string{"ref1", "ref2"},
		ProcessTime: 100 * time.Millisecond,
	}

	result, err := r.Execute(&common.PipelineContext{}, input)
	if err != nil {
		t.Fatalf("Execute error: %v", err)
	}

	resp, ok := result.(*Response)
	if !ok {
		t.Fatalf("expected *Response, got %T", result)
	}

	if resp.Answer != "test answer" {
		t.Errorf("expected 'test answer', got '%s'", resp.Answer)
	}
}

func TestResponder_Execute_NilInput(t *testing.T) {
	r := NewResponder()

	_, err := r.Execute(&common.PipelineContext{}, nil)
	if err == nil {
		t.Error("expected error for nil input")
	}
}

func TestResponder_ProcessQuery(t *testing.T) {
	r := NewResponder()

	query := &common.Query{
		ID:   "test-id",
		Text: "test query",
	}

	result, err := r.ProcessQuery(query)
	if err != nil {
		t.Fatalf("ProcessQuery error: %v", err)
	}

	if result.ID != "test-id" {
		t.Errorf("expected 'test-id', got '%s'", result.ID)
	}
}
