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
	"context"
	"testing"
)

type mockEmbedder struct {
	vectors [][]float64
	err     error
}

func (m *mockEmbedder) Embed(ctx context.Context, texts []string) ([][]float64, error) {
	return m.vectors, m.err
}

func TestNewSemanticSplitter(t *testing.T) {
	embedder := &mockEmbedder{}
	s := NewSemanticSplitter(embedder)
	if s == nil {
		t.Fatal("expected non-nil SemanticSplitter")
	}
	if s.Name() != "semantic_splitter" {
		t.Errorf("expected semantic_splitter, got %s", s.Name())
	}
}

func TestNewSemanticSplitter_NilEmbedder(t *testing.T) {
	s := NewSemanticSplitter(nil)
	if s == nil {
		t.Fatal("expected non-nil SemanticSplitter")
	}
	if s.embedder != nil {
		t.Error("expected nil embedder")
	}
}

func TestSemanticSplitter_Split(t *testing.T) {
	embedder := &mockEmbedder{}
	s := NewSemanticSplitter(embedder)

	content := "This is the first sentence. This is the second sentence. This is the third sentence."
	chunks, err := s.Split(content, map[string]interface{}{
		"chunk_size":    50,
		"chunk_overlap": 10,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(chunks) == 0 {
		t.Error("expected at least one chunk")
	}
}

func TestSemanticSplitter_Split_EmptyContent(t *testing.T) {
	s := NewSemanticSplitter(nil)

	chunks, err := s.Split("", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(chunks) != 0 {
		t.Errorf("expected 0 chunks for empty content, got %d", len(chunks))
	}
}

func TestSemanticSplitter_Split_DefaultOptions(t *testing.T) {
	s := NewSemanticSplitter(nil)

	content := "This is a test content."
	chunks, err := s.Split(content, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(chunks) == 0 {
		t.Error("expected at least one chunk")
	}
}

func TestSemanticSplitter_SplitBySentence(t *testing.T) {
	s := NewSemanticSplitter(nil)

	content := "First sentence. Second sentence! Third sentence? Fourth sentence\nFifth sentence."
	sentences := s.splitBySentence(content)
	if len(sentences) == 0 {
		t.Error("expected at least one sentence")
	}
}

func TestSemanticSplitter_SplitBySentence_Empty(t *testing.T) {
	s := NewSemanticSplitter(nil)

	sentences := s.splitBySentence("")
	if len(sentences) != 0 {
		t.Errorf("expected 0 sentences for empty content, got %d", len(sentences))
	}
}

func TestSemanticSplitter_CreateChunk(t *testing.T) {
	s := NewSemanticSplitter(nil)

	chunk := s.createChunk("test content", 0)
	if chunk.Content != "test content" {
		t.Errorf("expected content 'test content', got %s", chunk.Content)
	}
	if chunk.Index != 0 {
		t.Errorf("expected index 0, got %d", chunk.Index)
	}
	if chunk.ID == "" {
		t.Error("expected non-empty ID")
	}
	if chunk.Metadata["splitter"] != "semantic" {
		t.Errorf("expected splitter semantic, got %v", chunk.Metadata["splitter"])
	}
}

func TestSemanticSplitter_CalculateSimilarity_NoEmbedder(t *testing.T) {
	s := NewSemanticSplitter(nil)

	sim, err := s.calculateSemanticSimilarity(nil, "text1", "text2")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if sim != 0.5 {
		t.Errorf("expected 0.5 for nil embedder, got %f", sim)
	}
}

func TestSemanticSplitter_CalculateSimilarity_EmptyText(t *testing.T) {
	embedder := &mockEmbedder{}
	s := NewSemanticSplitter(embedder)

	sim, err := s.calculateSemanticSimilarity(nil, "", "text2")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if sim != 0.5 {
		t.Errorf("expected 0.5 for empty text, got %f", sim)
	}
}

func TestCosineSimilarity(t *testing.T) {
	// Test identical vectors
	a := []float64{1.0, 0.0}
	b := []float64{1.0, 0.0}
	sim := cosineSimilarity(a, b)
	if sim != 1.0 {
		t.Errorf("expected 1.0 for identical vectors, got %f", sim)
	}

	// Test orthogonal vectors
	a = []float64{1.0, 0.0}
	b = []float64{0.0, 1.0}
	sim = cosineSimilarity(a, b)
	if sim != 0.0 {
		t.Errorf("expected 0.0 for orthogonal vectors, got %f", sim)
	}

	// Test opposite vectors
	a = []float64{1.0, 0.0}
	b = []float64{-1.0, 0.0}
	sim = cosineSimilarity(a, b)
	if sim != -1.0 {
		t.Errorf("expected -1.0 for opposite vectors, got %f", sim)
	}
}

func TestCosineSimilarity_DifferentLengths(t *testing.T) {
	a := []float64{1.0, 0.0}
	b := []float64{1.0}
	sim := cosineSimilarity(a, b)
	if sim != 0 {
		t.Error("expected 0 for different length vectors")
	}
}

func TestCosineSimilarity_Empty(t *testing.T) {
	a := []float64{}
	b := []float64{}
	sim := cosineSimilarity(a, b)
	if sim != 0 {
		t.Error("expected 0 for empty vectors")
	}
}

func TestSemanticSplitter_MergeBySemantics(t *testing.T) {
	s := NewSemanticSplitter(nil)

	sentences := []string{"Sentence one.", "Sentence two.", "Sentence three."}
	chunks := s.mergeBySemantics(sentences, 50, 10, 0.3)
	if len(chunks) == 0 {
		t.Error("expected at least one chunk")
	}
}
