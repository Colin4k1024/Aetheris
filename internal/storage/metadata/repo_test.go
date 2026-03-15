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

package metadata

import (
	"context"
	"testing"
)

func TestNewRepository(t *testing.T) {
	store := NewMemoryStore()
	repo := NewRepository(store)
	if repo == nil {
		t.Fatal("expected non-nil repository")
	}
}

func TestRepository_ListDocuments(t *testing.T) {
	store := NewMemoryStore()
	repo := NewRepository(store)

	// Create a document
	doc := &Document{ID: "doc-1", Name: "test document"}
	store.Create(context.Background(), doc)

	// List documents
	docs, err := repo.ListDocuments(context.Background(), nil, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(docs) != 1 {
		t.Errorf("expected 1 document, got %d", len(docs))
	}
}

func TestRepository_ListDocuments_DefaultPagination(t *testing.T) {
	store := NewMemoryStore()
	repo := NewRepository(store)

	docs, err := repo.ListDocuments(context.Background(), nil, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if docs == nil {
		t.Fatal("expected non-nil result")
	}
}

func TestRepository_GetDocument(t *testing.T) {
	store := NewMemoryStore()
	repo := NewRepository(store)

	doc := &Document{ID: "doc-1", Name: "test"}
	store.Create(context.Background(), doc)

	got, err := repo.GetDocument(context.Background(), "doc-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got == nil || got.ID != "doc-1" {
		t.Error("expected to get the document")
	}
}

func TestRepository_GetDocument_NotFound(t *testing.T) {
	store := NewMemoryStore()
	repo := NewRepository(store)

	got, err := repo.GetDocument(context.Background(), "nonexistent")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != nil {
		t.Error("expected nil for nonexistent document")
	}
}

func TestRepository_DeleteDocument(t *testing.T) {
	store := NewMemoryStore()
	repo := NewRepository(store)

	doc := &Document{ID: "doc-1"}
	store.Create(context.Background(), doc)

	err := repo.DeleteDocument(context.Background(), "doc-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got, _ := repo.GetDocument(context.Background(), "doc-1")
	if got != nil {
		t.Error("expected document to be deleted")
	}
}
