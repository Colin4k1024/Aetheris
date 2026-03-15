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

	"rag-platform/internal/pipeline/common"
)

func TestNewReranker(t *testing.T) {
	r := NewReranker(5, 0.5)
	if r == nil {
		t.Fatal("expected non-nil reranker")
	}
	if r.Name() != "reranker" {
		t.Errorf("expected name reranker, got %s", r.Name())
	}
}

func TestNewReranker_Defaults(t *testing.T) {
	r := NewReranker(0, 0)
	if r.topK != 5 {
		t.Errorf("expected default topK 5, got %d", r.topK)
	}
	if r.scoreThreshold != 0.5 {
		t.Errorf("expected default scoreThreshold 0.5, got %f", r.scoreThreshold)
	}
}

func TestReranker_Validate(t *testing.T) {
	r := NewReranker(5, 0.5)

	err := r.Validate(nil)
	if err == nil {
		t.Error("expected error for nil input")
	}

	err = r.Validate("invalid")
	if err == nil {
		t.Error("expected error for invalid type")
	}

	err = r.Validate(&common.RetrievalResult{})
	if err != nil {
		t.Errorf("expected no error for valid input, got %v", err)
	}
}

func TestReranker_ProcessQuery(t *testing.T) {
	r := NewReranker(5, 0.5)

	query := &common.Query{Text: "test query"}
	result, err := r.ProcessQuery(query)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}
}

func TestReranker_Rerank(t *testing.T) {
	r := NewReranker(2, 0.5)

	result := &common.RetrievalResult{
		Chunks: []common.Chunk{
			{ID: "chunk-1", Content: "content 1"},
			{ID: "chunk-2", Content: "content 2"},
			{ID: "chunk-3", Content: "content 3"},
		},
		Scores:     []float64{0.3, 0.8, 0.6},
		TotalCount: 3,
	}

	reranked, err := r.rerank(result)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if reranked.TotalCount != 2 {
		t.Errorf("expected 2 chunks after rerank, got %d", reranked.TotalCount)
	}
	// Should be sorted by score descending
	if reranked.Scores[0] != 0.8 {
		t.Errorf("expected first score 0.8, got %f", reranked.Scores[0])
	}
}

func TestReranker_Rerank_FilterLowScores(t *testing.T) {
	r := NewReranker(5, 0.5)

	result := &common.RetrievalResult{
		Chunks: []common.Chunk{
			{ID: "chunk-1", Content: "content 1"},
			{ID: "chunk-2", Content: "content 2"},
			{ID: "chunk-3", Content: "content 3"},
		},
		Scores:     []float64{0.3, 0.8, 0.4},
		TotalCount: 3,
	}

	reranked, err := r.rerank(result)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Only chunk-2 (score 0.8) should pass threshold
	if reranked.TotalCount != 1 {
		t.Errorf("expected 1 chunk after filtering, got %d", reranked.TotalCount)
	}
}

func TestReranker_Rerank_EmptyChunks(t *testing.T) {
	r := NewReranker(5, 0.5)

	result := &common.RetrievalResult{
		Chunks:     []common.Chunk{},
		Scores:     []float64{},
		TotalCount: 0,
	}

	reranked, err := r.rerank(result)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if reranked.TotalCount != 0 {
		t.Errorf("expected 0 chunks, got %d", reranked.TotalCount)
	}
}

func TestReranker_Rerank_NilResult(t *testing.T) {
	r := NewReranker(5, 0.5)

	reranked, err := r.rerank(nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if reranked != nil {
		t.Error("expected nil result for nil input")
	}
}

func TestReranker_SetTopK(t *testing.T) {
	r := NewReranker(5, 0.5)

	r.SetTopK(10)
	if r.topK != 10 {
		t.Errorf("expected topK 10, got %d", r.topK)
	}

	r.SetTopK(0)
	if r.topK != 10 {
		t.Errorf("expected topK to remain 10, got %d", r.topK)
	}
}

func TestReranker_SetScoreThreshold(t *testing.T) {
	r := NewReranker(5, 0.5)

	r.SetScoreThreshold(0.8)
	if r.scoreThreshold != 0.8 {
		t.Errorf("expected scoreThreshold 0.8, got %f", r.scoreThreshold)
	}

	r.SetScoreThreshold(0)
	if r.scoreThreshold != 0.8 {
		t.Errorf("expected scoreThreshold to remain 0.8, got %f", r.scoreThreshold)
	}
}

func TestChunkScorePair_Len(t *testing.T) {
	pair := &chunkScorePair{
		chunks: []common.Chunk{{ID: "1"}, {ID: "2"}},
		scores: []float64{0.5, 0.8},
	}
	if pair.Len() != 2 {
		t.Errorf("expected length 2, got %d", pair.Len())
	}
}

func TestChunkScorePair_Less(t *testing.T) {
	pair := &chunkScorePair{
		chunks: []common.Chunk{{ID: "1"}, {ID: "2"}},
		scores: []float64{0.8, 0.5},
	}
	// Higher score (0.8) at index 0 should be "less than" lower score (0.5) at index 1
	// because in descending sort, we want index 0 to come before index 1
	if !pair.Less(0, 1) {
		t.Error("expected 0 to be less than 1 (higher score first)")
	}
	// Reverse: lower score at 0 should NOT be less than higher score at 1
	if pair.Less(1, 0) {
		t.Error("expected 1 NOT to be less than 0")
	}
}

func TestChunkScorePair_Swap(t *testing.T) {
	pair := &chunkScorePair{
		chunks: []common.Chunk{{ID: "1"}, {ID: "2"}},
		scores: []float64{0.5, 0.8},
	}
	pair.Swap(0, 1)
	if pair.chunks[0].ID != "2" || pair.scores[0] != 0.8 {
		t.Error("swap did not work correctly")
	}
}

func TestReranker_Execute(t *testing.T) {
	r := NewReranker(2, 0.5)

	result := &common.RetrievalResult{
		Chunks: []common.Chunk{
			{ID: "chunk-1", Content: "content 1"},
			{ID: "chunk-2", Content: "content 2"},
		},
		Scores:     []float64{0.3, 0.8},
		TotalCount: 2,
	}

	ctx := &common.PipelineContext{}
	reranked, err := r.Execute(ctx, result)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if reranked == nil {
		t.Fatal("expected non-nil result")
	}
}

func TestReranker_Execute_InvalidInput(t *testing.T) {
	r := NewReranker(5, 0.5)

	ctx := &common.PipelineContext{}
	_, err := r.Execute(ctx, "invalid")
	if err == nil {
		t.Error("expected error for invalid input")
	}
}
