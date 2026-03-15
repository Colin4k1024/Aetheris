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
	"context"
	"testing"

	"rag-platform/internal/pipeline/common"
	"rag-platform/internal/storage/vector"
)

// mockVectorStore is a mock implementation of vector.Store
type mockVectorStore struct {
	searchResult []*vector.SearchResult
	searchErr    error
}

func (m *mockVectorStore) Create(ctx context.Context, index *vector.Index) error {
	return nil
}

func (m *mockVectorStore) Add(ctx context.Context, indexName string, vectors []*vector.Vector) error {
	return nil
}

func (m *mockVectorStore) Search(ctx context.Context, indexName string, query []float64, options *vector.SearchOptions) ([]*vector.SearchResult, error) {
	return m.searchResult, m.searchErr
}

func (m *mockVectorStore) Get(ctx context.Context, indexName string, id string) (*vector.Vector, error) {
	return nil, nil
}

func (m *mockVectorStore) Delete(ctx context.Context, indexName string, id string) error {
	return nil
}

func (m *mockVectorStore) DeleteIndex(ctx context.Context, indexName string) error {
	return nil
}

func (m *mockVectorStore) ListIndexes(ctx context.Context) ([]string, error) {
	return nil, nil
}

func (m *mockVectorStore) Close() error {
	return nil
}

func TestNewRetriever(t *testing.T) {
	store := &mockVectorStore{}
	r := NewRetriever(store, "test-index", 5, 0.5)
	if r == nil {
		t.Fatal("expected non-nil retriever")
	}
	if r.Name() != "retriever" {
		t.Errorf("expected name retriever, got %s", r.Name())
	}
}

func TestNewRetriever_Defaults(t *testing.T) {
	store := &mockVectorStore{}
	r := NewRetriever(store, "", 0, 0)
	if r.topK != 10 {
		t.Errorf("expected default topK 10, got %d", r.topK)
	}
	if r.scoreThreshold != 0.3 {
		t.Errorf("expected default scoreThreshold 0.3, got %f", r.scoreThreshold)
	}
	if r.indexName != "default" {
		t.Errorf("expected default indexName default, got %s", r.indexName)
	}
}

func TestRetriever_Validate(t *testing.T) {
	store := &mockVectorStore{}
	r := NewRetriever(store, "test", 5, 0.5)

	err := r.Validate(nil)
	if err == nil {
		t.Error("expected error for nil input")
	}

	err = r.Validate("invalid")
	if err == nil {
		t.Error("expected error for invalid type")
	}

	err = r.Validate(&common.Query{})
	if err != nil {
		t.Errorf("expected no error for valid query, got %v", err)
	}
}

func TestRetriever_Validate_NoStore(t *testing.T) {
	r := &Retriever{vectorStore: nil}
	err := r.Validate(&common.Query{})
	if err == nil {
		t.Error("expected error for nil store")
	}
}

func TestRetriever_ProcessQuery_NoEmbedding(t *testing.T) {
	store := &mockVectorStore{}
	r := NewRetriever(store, "test", 5, 0.5)

	query := &common.Query{
		Text: "test query",
	}
	_, err := r.ProcessQuery(query)
	if err == nil {
		t.Error("expected error for query without embedding")
	}
}

func TestRetriever_ProcessQuery_WithResults(t *testing.T) {
	store := &mockVectorStore{
		searchResult: []*vector.SearchResult{
			{
				ID:    "chunk-1",
				Score: 0.9,
				Metadata: map[string]string{
					"content":     "test content",
					"document_id": "doc-1",
					"index":       "0",
					"token_count": "100",
				},
			},
		},
	}
	r := NewRetriever(store, "test", 5, 0.5)

	query := &common.Query{
		Text:      "test query",
		Embedding: []float64{0.1, 0.2, 0.3},
	}
	result, err := r.ProcessQuery(query)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if result.TotalCount != 1 {
		t.Errorf("expected 1 result, got %d", result.TotalCount)
	}
}

func TestRetriever_Execute(t *testing.T) {
	store := &mockVectorStore{
		searchResult: []*vector.SearchResult{
			{
				ID:    "chunk-1",
				Score: 0.9,
				Metadata: map[string]string{
					"content":     "test content",
					"document_id": "doc-1",
				},
			},
		},
	}
	r := NewRetriever(store, "test", 5, 0.5)

	query := &common.Query{
		Text:      "test query",
		Embedding: []float64{0.1, 0.2, 0.3},
	}
	ctx := &common.PipelineContext{}
	result, err := r.Execute(ctx, query)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}
}

func TestRetriever_Execute_InvalidInput(t *testing.T) {
	store := &mockVectorStore{}
	r := NewRetriever(store, "test", 5, 0.5)

	ctx := &common.PipelineContext{}
	_, err := r.Execute(ctx, "invalid")
	if err == nil {
		t.Error("expected error for invalid input")
	}
}

func TestRetriever_SetVectorStore(t *testing.T) {
	store := &mockVectorStore{}
	r := NewRetriever(store, "test", 5, 0.5)

	newStore := &mockVectorStore{}
	r.SetVectorStore(newStore)
	if r.vectorStore != newStore {
		t.Error("vector store not set correctly")
	}
}

func TestRetriever_GetVectorStore(t *testing.T) {
	store := &mockVectorStore{}
	r := NewRetriever(store, "test", 5, 0.5)

	got := r.GetVectorStore()
	if got != store {
		t.Error("GetVectorStore returned wrong store")
	}
}

func TestRetriever_SetIndexName(t *testing.T) {
	store := &mockVectorStore{}
	r := NewRetriever(store, "test", 5, 0.5)

	r.SetIndexName("new-index")
	if r.indexName != "new-index" {
		t.Errorf("expected indexName new-index, got %s", r.indexName)
	}

	r.SetIndexName("")
	if r.indexName != "new-index" {
		t.Errorf("expected indexName to remain new-index, got %s", r.indexName)
	}
}

func TestRetriever_SetTopK(t *testing.T) {
	store := &mockVectorStore{}
	r := NewRetriever(store, "test", 5, 0.5)

	r.SetTopK(20)
	if r.topK != 20 {
		t.Errorf("expected topK 20, got %d", r.topK)
	}

	r.SetTopK(0)
	if r.topK != 20 {
		t.Errorf("expected topK to remain 20, got %d", r.topK)
	}
}

func TestRetriever_SetScoreThreshold(t *testing.T) {
	store := &mockVectorStore{}
	r := NewRetriever(store, "test", 5, 0.5)

	r.SetScoreThreshold(0.8)
	if r.scoreThreshold != 0.8 {
		t.Errorf("expected scoreThreshold 0.8, got %f", r.scoreThreshold)
	}

	r.SetScoreThreshold(0)
	if r.scoreThreshold != 0.8 {
		t.Errorf("expected scoreThreshold to remain 0.8, got %f", r.scoreThreshold)
	}
}
