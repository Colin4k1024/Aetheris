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
)

func TestNewStructuralSplitter(t *testing.T) {
	ss := NewStructuralSplitter()
	if ss == nil {
		t.Fatal("NewStructuralSplitter should not return nil")
	}
	if ss.Name() != "structural_splitter" {
		t.Errorf("expected name 'structural_splitter', got '%s'", ss.Name())
	}
}

func TestStructuralSplitter_Split(t *testing.T) {
	ss := NewStructuralSplitter()

	content := "This is paragraph one.\n\nThis is paragraph two."
	chunks, err := ss.Split(content, nil)
	if err != nil {
		t.Fatalf("Split error: %v", err)
	}
	if len(chunks) == 0 {
		t.Error("expected at least one chunk")
	}
}

func TestStructuralSplitter_Split_WithOptions(t *testing.T) {
	ss := NewStructuralSplitter()

	content := "Short content"
	chunks, err := ss.Split(content, map[string]interface{}{
		"chunk_size":    100,
		"chunk_overlap": 10,
	})
	if err != nil {
		t.Fatalf("Split error: %v", err)
	}
	if len(chunks) == 0 {
		t.Error("expected at least one chunk")
	}
}

func TestStructuralSplitter_Split_Empty(t *testing.T) {
	ss := NewStructuralSplitter()

	chunks, err := ss.Split("", nil)
	if err != nil {
		t.Fatalf("Split error: %v", err)
	}
	if len(chunks) != 0 {
		t.Errorf("expected 0 chunks for empty content, got %d", len(chunks))
	}
}

func TestStructuralSplitter_Split_MultipleParagraphs(t *testing.T) {
	ss := NewStructuralSplitter()

	content := "Para one.\nPara two.\n\nPara three."
	chunks, err := ss.Split(content, map[string]interface{}{
		"chunk_size": 50,
	})
	if err != nil {
		t.Fatalf("Split error: %v", err)
	}
	if len(chunks) == 0 {
		t.Error("expected at least one chunk")
	}
}

func TestStructuralSplitter_Split_LongParagraph(t *testing.T) {
	ss := NewStructuralSplitter()

	// Create a long paragraph that exceeds chunk_size
	longContent := ""
	for i := 0; i < 50; i++ {
		longContent += "This is a very long paragraph that contains a lot of text. "
	}

	chunks, err := ss.Split(longContent, map[string]interface{}{
		"chunk_size":    100,
		"chunk_overlap": 20,
	})
	if err != nil {
		t.Fatalf("Split error: %v", err)
	}
	// Should produce multiple chunks from the long paragraph
	if len(chunks) <= 1 {
		t.Errorf("expected multiple chunks from long paragraph, got %d", len(chunks))
	}
}
