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

package vector

import (
	"context"
	"testing"
)

func TestMemoryStore_Create_Add_Search(t *testing.T) {
	ctx := context.Background()
	s := NewMemoryStore()
	idx := &Index{Name: "idx1", Dimension: 2, Distance: "cosine"}
	if err := s.Create(ctx, idx); err != nil {
		t.Fatalf("Create: %v", err)
	}
	vecs := []*Vector{
		{ID: "v1", Values: []float64{1, 0}},
		{ID: "v2", Values: []float64{0, 1}},
	}
	if err := s.Add(ctx, "idx1", vecs); err != nil {
		t.Fatalf("Add: %v", err)
	}
	results, err := s.Search(ctx, "idx1", []float64{1, 0}, &SearchOptions{TopK: 2})
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(results) < 1 {
		t.Fatalf("Search: expected at least 1 result, got %d", len(results))
	}
	if results[0].ID != "v1" {
		t.Errorf("Search: expected v1 first (cosine sim), got %s", results[0].ID)
	}
}

func TestMemoryStore_Create_DuplicateIndex(t *testing.T) {
	ctx := context.Background()
	s := NewMemoryStore()
	idx := &Index{Name: "x", Dimension: 2}
	_ = s.Create(ctx, idx)
	err := s.Create(ctx, idx)
	if err == nil {
		t.Error("Create duplicate index should error")
	}
}

func TestMemoryStore_Add_IndexNotFound(t *testing.T) {
	ctx := context.Background()
	s := NewMemoryStore()
	err := s.Add(ctx, "missing", []*Vector{{ID: "v1", Values: []float64{1}}})
	if err == nil {
		t.Error("Add to missing index should error")
	}
}

func TestMemoryStore_Add_DimensionMismatch(t *testing.T) {
	ctx := context.Background()
	s := NewMemoryStore()
	_ = s.Create(ctx, &Index{Name: "i", Dimension: 2})
	err := s.Add(ctx, "i", []*Vector{{ID: "v1", Values: []float64{1, 0, 0}}})
	if err == nil {
		t.Error("Add with wrong dimension should error")
	}
}

func TestMemoryStore_Search_IndexNotFound(t *testing.T) {
	ctx := context.Background()
	s := NewMemoryStore()
	_, err := s.Search(ctx, "missing", []float64{1}, nil)
	if err == nil {
		t.Error("Search missing index should error")
	}
}

func TestMemoryStore_Get(t *testing.T) {
	ctx := context.Background()
	s := NewMemoryStore()
	_ = s.Create(ctx, &Index{Name: "idx1", Dimension: 2})
	_ = s.Add(ctx, "idx1", []*Vector{{ID: "v1", Values: []float64{1, 0}}})

	vec, err := s.Get(ctx, "idx1", "v1")
	if err != nil {
		t.Errorf("Get failed: %v", err)
	}
	if vec.ID != "v1" {
		t.Errorf("expected v1, got %s", vec.ID)
	}
}

func TestMemoryStore_Get_IndexNotFound(t *testing.T) {
	ctx := context.Background()
	s := NewMemoryStore()
	_, err := s.Get(ctx, "missing", "v1")
	if err == nil {
		t.Error("Get from missing index should error")
	}
}

func TestMemoryStore_Get_VectorNotFound(t *testing.T) {
	ctx := context.Background()
	s := NewMemoryStore()
	_ = s.Create(ctx, &Index{Name: "idx1", Dimension: 2})
	_, err := s.Get(ctx, "idx1", "nonexistent")
	if err == nil {
		t.Error("Get nonexistent vector should error")
	}
}

func TestMemoryStore_Delete(t *testing.T) {
	ctx := context.Background()
	s := NewMemoryStore()
	_ = s.Create(ctx, &Index{Name: "idx1", Dimension: 2})
	_ = s.Add(ctx, "idx1", []*Vector{{ID: "v1", Values: []float64{1, 0}}})

	err := s.Delete(ctx, "idx1", "v1")
	if err != nil {
		t.Errorf("Delete failed: %v", err)
	}
}

func TestMemoryStore_DeleteIndex(t *testing.T) {
	ctx := context.Background()
	s := NewMemoryStore()
	_ = s.Create(ctx, &Index{Name: "idx1", Dimension: 2})

	err := s.DeleteIndex(ctx, "idx1")
	if err != nil {
		t.Errorf("DeleteIndex failed: %v", err)
	}
}

func TestMemoryStore_ListIndexes(t *testing.T) {
	ctx := context.Background()
	s := NewMemoryStore()
	_ = s.Create(ctx, &Index{Name: "idx1", Dimension: 2})
	_ = s.Create(ctx, &Index{Name: "idx2", Dimension: 2})

	indexes, err := s.ListIndexes(ctx)
	if err != nil {
		t.Errorf("ListIndexes failed: %v", err)
	}
	if len(indexes) != 2 {
		t.Errorf("expected 2 indexes, got %d", len(indexes))
	}
}

func TestMemoryStore_Close(t *testing.T) {
	s := NewMemoryStore()
	err := s.Close()
	if err != nil {
		t.Errorf("Close failed: %v", err)
	}
}

func TestMemoryStore_Search_WithThreshold(t *testing.T) {
	ctx := context.Background()
	s := NewMemoryStore()
	_ = s.Create(ctx, &Index{Name: "idx1", Dimension: 2, Distance: "cosine"})
	_ = s.Add(ctx, "idx1", []*Vector{
		{ID: "v1", Values: []float64{1, 0}},
		{ID: "v2", Values: []float64{0, 1}},
	})

	results, err := s.Search(ctx, "idx1", []float64{1, 0}, &SearchOptions{TopK: 10, Threshold: 0.9})
	if err != nil {
		t.Errorf("Search failed: %v", err)
	}
	// v1 should have high similarity, v2 should be below threshold
	if len(results) > 1 {
		t.Errorf("expected at most 1 result with threshold 0.9, got %d", len(results))
	}
}

func TestMemoryStore_Search_WithFilter(t *testing.T) {
	ctx := context.Background()
	s := NewMemoryStore()
	_ = s.Create(ctx, &Index{Name: "idx1", Dimension: 2, Distance: "cosine"})
	_ = s.Add(ctx, "idx1", []*Vector{
		{ID: "v1", Values: []float64{1, 0}, Metadata: map[string]string{"type": "a"}},
		{ID: "v2", Values: []float64{0, 1}, Metadata: map[string]string{"type": "b"}},
	})

	results, err := s.Search(ctx, "idx1", []float64{1, 0}, &SearchOptions{TopK: 10, Filter: map[string]string{"type": "a"}})
	if err != nil {
		t.Errorf("Search failed: %v", err)
	}
	if len(results) != 1 {
		t.Errorf("expected 1 result with filter, got %d", len(results))
	}
	if results[0].ID != "v1" {
		t.Errorf("expected v1, got %s", results[0].ID)
	}
}

func TestMemoryStore_Search_IncludeVectors(t *testing.T) {
	ctx := context.Background()
	s := NewMemoryStore()
	_ = s.Create(ctx, &Index{Name: "idx1", Dimension: 2, Distance: "cosine"})
	_ = s.Add(ctx, "idx1", []*Vector{{ID: "v1", Values: []float64{1, 0}}})

	results, err := s.Search(ctx, "idx1", []float64{1, 0}, &SearchOptions{TopK: 1, IncludeVectors: true})
	if err != nil {
		t.Errorf("Search failed: %v", err)
	}
	if len(results) > 0 && results[0].Values == nil {
		t.Error("expected vectors to be included")
	}
}
