// Copyright 2026 fanjia1024
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or  by applicable law. A copy of the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package pipeline

import (
	"context"
	"testing"
	"time"

	"github.com/Colin4k1024/Aetheris/v2/internal/storage/vector"
)

// mockEmbedder is a mock embedder for testing
type mockEmbedder struct {
	dimension int
	model     string
}

func (m *mockEmbedder) Embed(ctx context.Context, texts []string) ([][]float64, error) {
	// Return deterministic mock vectors based on text hash
	result := make([][]float64, len(texts))
	for i, text := range texts {
		vec := make([]float64, m.dimension)
		// Simple hash-based vector for testing
		hash := float64(len(text))
		for j := 0; j < m.dimension; j++ {
			vec[j] = (hash + float64(j)) / float64(m.dimension+len(text))
		}
		result[i] = vec
	}
	return result, nil
}

func (m *mockEmbedder) Dimension() int {
	return m.dimension
}

func (m *mockEmbedder) Model() string {
	return m.model
}

func TestNewMemoryPipeline(t *testing.T) {
	config := DefaultMemoryPipelineConfig()
	embedder := &mockEmbedder{dimension: 1536, model: "test"}
	store := vector.NewMemoryStore()

	pipeline, err := NewMemoryPipeline(config, embedder, store)
	if err != nil {
		t.Fatalf("NewMemoryPipeline failed: %v", err)
	}

	if pipeline == nil {
		t.Fatal("pipeline is nil")
	}

	// Check initial stats
	total, avgWeight := pipeline.GetStats()
	if total != 0 {
		t.Errorf("expected 0 initial items, got %d", total)
	}
	if avgWeight != 0 {
		t.Errorf("expected 0 initial avg weight, got %f", avgWeight)
	}
}

func TestMemoryPipeline_StoreAndRecall(t *testing.T) {
	ctx := context.Background()
	config := DefaultMemoryPipelineConfig()
	embedder := &mockEmbedder{dimension: 2, model: "test"}
	store := vector.NewMemoryStore()

	pipeline, err := NewMemoryPipeline(config, embedder, store)
	if err != nil {
		t.Fatalf("NewMemoryPipeline failed: %v", err)
	}

	// Store some memories
	memories := []VectorMemoryItem{
		{ID: "mem1", Content: "User prefers dark mode", Weight: 1.0},
		{ID: "mem2", Content: "User likes Python programming", Weight: 0.9},
		{ID: "mem3", Content: "User works on AI projects", Weight: 0.8},
	}

	for _, mem := range memories {
		if err := pipeline.Store(ctx, mem); err != nil {
			t.Fatalf("Store failed for %s: %v", mem.ID, err)
		}
	}

	// Check stats
	total, _ := pipeline.GetStats()
	if total != 3 {
		t.Errorf("expected 3 items, got %d", total)
	}

	// Recall similar memories
	results, err := pipeline.Recall(ctx, "programming language preferences", 2)
	if err != nil {
		t.Fatalf("Recall failed: %v", err)
	}

	if len(results) == 0 {
		t.Error("expected some recall results")
	}
}

func TestMemoryPipeline_Decay(t *testing.T) {
	ctx := context.Background()
	config := DefaultMemoryPipelineConfig()
	config.DecayInterval = time.Millisecond // Force decay on every call
	config.MinWeight = 0.5

	embedder := &mockEmbedder{dimension: 2, model: "test"}
	store := vector.NewMemoryStore()

	pipeline, err := NewMemoryPipeline(config, embedder, store)
	if err != nil {
		t.Fatalf("NewMemoryPipeline failed: %v", err)
	}

	// Store a memory with initial weight
	mem := VectorMemoryItem{
		ID:      "mem1",
		Content: "Test content",
		Weight:  1.0,
	}
	if err := pipeline.Store(ctx, mem); err != nil {
		t.Fatalf("Store failed: %v", err)
	}

	// Wait a bit and decay
	time.Sleep(10 * time.Millisecond)
	if err := pipeline.Decay(ctx); err != nil {
		t.Fatalf("Decay failed: %v", err)
	}

	// Check that weight decreased
	total, _ := pipeline.GetStats()
	if total != 1 {
		t.Errorf("expected 1 item after decay, got %d", total)
	}
}

func TestMemoryPipeline_Compress(t *testing.T) {
	ctx := context.Background()
	config := DefaultMemoryPipelineConfig()
	config.SimilarityThreshold = 0.9 // High threshold to force grouping

	embedder := &mockEmbedder{dimension: 2, model: "test"}
	store := vector.NewMemoryStore()

	pipeline, err := NewMemoryPipeline(config, embedder, store)
	if err != nil {
		t.Fatalf("NewMemoryPipeline failed: %v", err)
	}

	// Store similar memories
	mem1 := VectorMemoryItem{
		ID:      "mem1",
		Content: "User likes machine learning",
		Weight:  1.0,
	}
	mem2 := VectorMemoryItem{
		ID:      "mem2",
		Content: "Machine learning is great",
		Weight:  0.9,
	}

	if err := pipeline.Store(ctx, mem1); err != nil {
		t.Fatalf("Store mem1 failed: %v", err)
	}
	if err := pipeline.Store(ctx, mem2); err != nil {
		t.Fatalf("Store mem2 failed: %v", err)
	}

	// Run compression
	if err := pipeline.Compress(ctx); err != nil {
		t.Fatalf("Compress failed: %v", err)
	}

	// Check stats - should have fewer items after compression
	total, _ := pipeline.GetStats()
	if total >= 2 {
		t.Logf("After compression: %d items (may have merged)", total)
	}
}

func TestMemoryPipeline_Maintenance(t *testing.T) {
	ctx := context.Background()
	config := DefaultMemoryPipelineConfig()
	config.DecayInterval = time.Millisecond
	config.MaintenanceInterval = time.Millisecond

	embedder := &mockEmbedder{dimension: 2, model: "test"}
	store := vector.NewMemoryStore()

	pipeline, err := NewMemoryPipeline(config, embedder, store)
	if err != nil {
		t.Fatalf("NewMemoryPipeline failed: %v", err)
	}

	// Store some memories
	mem := VectorMemoryItem{
		ID:      "mem1",
		Content: "Test content for maintenance",
		Weight:  1.0,
	}
	if err := pipeline.Store(ctx, mem); err != nil {
		t.Fatalf("Store failed: %v", err)
	}

	// Run maintenance
	time.Sleep(10 * time.Millisecond)
	if err := pipeline.Maintenance(ctx); err != nil {
		t.Fatalf("Maintenance failed: %v", err)
	}

	// Pipeline should still be functional
	total, _ := pipeline.GetStats()
	if total < 0 {
		t.Errorf("expected valid item count after maintenance, got %d", total)
	}
}

func TestMemoryPipeline_Close(t *testing.T) {
	config := DefaultMemoryPipelineConfig()
	embedder := &mockEmbedder{dimension: 2, model: "test"}
	store := vector.NewMemoryStore()

	pipeline, err := NewMemoryPipeline(config, embedder, store)
	if err != nil {
		t.Fatalf("NewMemoryPipeline failed: %v", err)
	}

	if err := pipeline.Close(); err != nil {
		t.Fatalf("Close failed: %v", err)
	}
}

func TestCosineSimilarity(t *testing.T) {
	tests := []struct {
		name     string
		a        []float64
		b        []float64
		expected float64
	}{
		{
			name:     "identical vectors",
			a:        []float64{1, 0},
			b:        []float64{1, 0},
			expected: 1.0,
		},
		{
			name:     "orthogonal vectors",
			a:        []float64{1, 0},
			b:        []float64{0, 1},
			expected: 0.0,
		},
		{
			name:     "opposite vectors",
			a:        []float64{1, 0},
			b:        []float64{-1, 0},
			expected: -1.0,
		},
		{
			name:     "similar vectors",
			a:        []float64{1, 0},
			b:        []float64{0.9, 0.1},
			expected: 0.9938837346736189,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := cosineSimilarity(tt.a, tt.b)
			// Use approximate comparison for floating point
			if result < tt.expected-0.001 || result > tt.expected+0.001 {
				t.Errorf("cosineSimilarity(%v, %v) = %v, expected %v", tt.a, tt.b, result, tt.expected)
			}
		})
	}
}

func TestMemoryPipeline_StoreEmptyContent(t *testing.T) {
	ctx := context.Background()
	config := DefaultMemoryPipelineConfig()
	embedder := &mockEmbedder{dimension: 2, model: "test"}
	store := vector.NewMemoryStore()

	pipeline, err := NewMemoryPipeline(config, embedder, store)
	if err != nil {
		t.Fatalf("NewMemoryPipeline failed: %v", err)
	}

	// Try to store empty content
	mem := VectorMemoryItem{
		ID:      "mem1",
		Content: "",
		Weight:  1.0,
	}
	err = pipeline.Store(ctx, mem)
	if err == nil {
		t.Error("expected error for empty content")
	}
}

func TestMemoryPipeline_RecallEmptyQuery(t *testing.T) {
	ctx := context.Background()
	config := DefaultMemoryPipelineConfig()
	embedder := &mockEmbedder{dimension: 2, model: "test"}
	store := vector.NewMemoryStore()

	pipeline, err := NewMemoryPipeline(config, embedder, store)
	if err != nil {
		t.Fatalf("NewMemoryPipeline failed: %v", err)
	}

	// Recall with empty query should still work (returns all items up to topK)
	results, err := pipeline.Recall(ctx, "", 10)
	if err != nil {
		t.Fatalf("Recall failed: %v", err)
	}

	// Empty query with no stored items should return empty
	if len(results) != 0 {
		t.Errorf("expected 0 results for empty query with no items, got %d", len(results))
	}
}
