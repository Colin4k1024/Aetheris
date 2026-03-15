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

package api

import (
	"context"
	"testing"
	"time"

	"rag-platform/internal/agent/memory"
)

func TestMemoryProviderAdapter_Recall(t *testing.T) {
	mockMem := &mockMemoryForAdapter{}
	adapter := &memoryProviderAdapter{m: mockMem}

	result, err := adapter.Recall(context.Background(), "test query")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if result == nil {
		t.Error("expected non-nil result")
	}
}

func TestMemoryProviderAdapter_Recall_NonContext(t *testing.T) {
	mockMem := &mockMemoryForAdapter{}
	adapter := &memoryProviderAdapter{m: mockMem}

	// Pass non-context type
	result, err := adapter.Recall("not a context", "test query")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if result == nil {
		t.Error("expected non-nil result")
	}
}

func TestMemoryProviderAdapter_Store(t *testing.T) {
	mockMem := &mockMemoryForAdapter{}
	adapter := &memoryProviderAdapter{m: mockMem}

	item := memory.MemoryItem{
		Type:    "working",
		Content: "test",
		At:      time.Now(),
	}
	err := adapter.Store(context.Background(), item)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestMemoryProviderAdapter_Store_NonContext(t *testing.T) {
	mockMem := &mockMemoryForAdapter{}
	adapter := &memoryProviderAdapter{m: mockMem}

	item := memory.MemoryItem{
		Type:    "working",
		Content: "test",
		At:      time.Now(),
	}
	err := adapter.Store("not a context", item)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestMemoryProviderAdapter_Store_NonMemoryItem(t *testing.T) {
	mockMem := &mockMemoryForAdapter{}
	adapter := &memoryProviderAdapter{m: mockMem}

	err := adapter.Store(context.Background(), "not a memory item")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

// Mock implementations

type mockMemoryForAdapter struct {
	recallResult []memory.MemoryItem
}

func (m *mockMemoryForAdapter) Recall(ctx context.Context, query string) ([]memory.MemoryItem, error) {
	return []memory.MemoryItem{{Type: "working", Content: "test", At: time.Now()}}, nil
}

func (m *mockMemoryForAdapter) Store(ctx context.Context, item memory.MemoryItem) error {
	return nil
}
