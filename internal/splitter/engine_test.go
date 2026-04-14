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

package splitter

import (
	"testing"

	"github.com/Colin4k1024/Aetheris/v2/internal/pipeline/common"
)

type testSplitter struct {
	nameValue string
}

func (t *testSplitter) Split(content string, options map[string]interface{}) ([]common.Chunk, error) {
	return []common.Chunk{{ID: "chunk-1", Content: content}}, nil
}

func (t *testSplitter) Name() string {
	return t.nameValue
}

func TestNewEngine(t *testing.T) {
	engine := NewEngine(nil)
	if engine == nil {
		t.Fatal("expected non-nil Engine")
	}
	if engine.Name() != "splitter_engine" {
		t.Errorf("expected splitter_engine, got %s", engine.Name())
	}
}

func TestNewEngine_WithEmbedder(t *testing.T) {
	embedder := &mockEmbedder{}
	engine := NewEngine(embedder)
	if engine == nil {
		t.Fatal("expected non-nil Engine")
	}
}

func TestEngine_GetSplitter(t *testing.T) {
	engine := NewEngine(nil)

	splitter, err := engine.GetSplitter("structural")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if splitter == nil {
		t.Fatal("expected non-nil splitter")
	}

	_, err = engine.GetSplitter("nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent splitter")
	}
}

func TestEngine_GetSplitters(t *testing.T) {
	engine := NewEngine(nil)

	splitters := engine.GetSplitters()
	if len(splitters) == 0 {
		t.Error("expected at least one splitter")
	}
}

func TestEngine_AddSplitter(t *testing.T) {
	engine := NewEngine(nil)

	customSplitter := &testSplitter{nameValue: "custom"}
	engine.AddSplitter("custom", customSplitter)

	splitter, err := engine.GetSplitter("custom")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if splitter.Name() != "custom" {
		t.Errorf("expected custom, got %s", splitter.Name())
	}
}

func TestEngine_Split(t *testing.T) {
	engine := NewEngine(nil)

	chunks, err := engine.Split("test content", "structural", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(chunks) == 0 {
		t.Error("expected at least one chunk")
	}

	_, err = engine.Split("test", "nonexistent", nil)
	if err == nil {
		t.Error("expected error for nonexistent splitter")
	}
}

func TestEngine_SplitDocument(t *testing.T) {
	engine := NewEngine(nil)

	doc := &common.Document{
		ID:       "doc-1",
		Content:  "test content",
		Metadata: make(map[string]interface{}),
	}

	result, err := engine.SplitDocument(doc, "structural", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if len(result.Chunks) == 0 {
		t.Error("expected at least one chunk")
	}
	if result.Metadata["chunked"] != true {
		t.Error("expected chunked to be true")
	}
}

func TestEngine_SplitDocument_Error(t *testing.T) {
	engine := NewEngine(nil)

	doc := &common.Document{
		ID:       "doc-1",
		Content:  "test content",
		Metadata: make(map[string]interface{}),
	}

	_, err := engine.SplitDocument(doc, "nonexistent", nil)
	if err == nil {
		t.Error("expected error for nonexistent splitter")
	}
}
