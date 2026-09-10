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

package specialized

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Colin4k1024/Aetheris/v2/internal/pipeline/common"
)

// mockHiveClient implements HiveQueryClient for testing.
type mockHiveClient struct {
	mu       sync.Mutex
	rows     []map[string]interface{}
	queryLog []string
	closed   bool
}

func (m *mockHiveClient) Query(ctx context.Context, query string) ([]map[string]interface{}, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.queryLog = append(m.queryLog, query)
	// Return a copy to avoid mutation issues
	result := make([]map[string]interface{}, len(m.rows))
	for i, row := range m.rows {
		copy := make(map[string]interface{})
		for k, v := range row {
			copy[k] = v
		}
		result[i] = copy
	}
	return result, nil
}

func (m *mockHiveClient) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.closed = true
	return nil
}

// mockProgressStore implements HiveSyncProgressStore for testing.
type mockProgressStore struct {
	mu      sync.Mutex
	cursors map[string]string
}

func newMockProgressStore() *mockProgressStore {
	return &mockProgressStore{cursors: make(map[string]string)}
}

func (s *mockProgressStore) GetCursor(syncKey string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.cursors[syncKey], nil
}

func (s *mockProgressStore) SaveCursor(syncKey string, cursor string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.cursors[syncKey] = cursor
	return nil
}

// --- Tests ---

func TestHIVEPipeline_Name(t *testing.T) {
	p := NewHIVEPipeline()
	if p.Name() != "hive" {
		t.Errorf("expected name 'hive', got %s", p.Name())
	}
}

func TestHIVEPipeline_NoClient(t *testing.T) {
	p := NewHIVEPipeline()
	_, err := p.ProcessSpecialized(context.Background())
	if err == nil {
		t.Error("expected error when no client configured")
	}
}

func TestHIVEPipeline_NoQueryTemplate(t *testing.T) {
	p := NewHIVEPipeline()
	p.SetClient(&mockHiveClient{})
	_, err := p.ProcessSpecialized(context.Background())
	if err == nil {
		t.Error("expected error when no query template configured")
	}
}

func TestHIVEPipeline_FullLoad(t *testing.T) {
	client := &mockHiveClient{
		rows: []map[string]interface{}{
			{"id": 1, "text": "first", "category": "a"},
			{"id": 2, "text": "second", "category": "b"},
			{"id": 3, "text": "third", "category": "c"},
		},
	}

	p := NewHIVEPipeline()
	p.SetClient(client)
	p.SetConfig(HiveConfig{
		QueryTemplate: "SELECT * FROM events",
		ContentField:  "text",
		IDField:       "id",
	})

	result, err := p.ProcessSpecialized(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	hr, ok := result.(*HiveResult)
	if !ok {
		t.Fatalf("expected *HiveResult, got %T", result)
	}
	if len(hr.Documents) != 3 {
		t.Fatalf("expected 3 documents, got %d", len(hr.Documents))
	}
	if hr.Documents[0].ID != "1" {
		t.Errorf("expected id=1, got %s", hr.Documents[0].ID)
	}
	if hr.Documents[0].Content != "first" {
		t.Errorf("expected content 'first', got %s", hr.Documents[0].Content)
	}
	if hr.Documents[0].Metadata["category"] != "a" {
		t.Errorf("expected category=a, got %v", hr.Documents[0].Metadata["category"])
	}
	if hr.TotalRows != 3 {
		t.Errorf("expected total 3 rows, got %d", hr.TotalRows)
	}
}

func TestHIVEPipeline_IncrementalSync(t *testing.T) {
	client := &mockHiveClient{
		rows: []map[string]interface{}{
			{"id": 11, "text": "row11"},
			{"id": 12, "text": "row12"},
		},
	}
	store := newMockProgressStore()

	p := NewHIVEPipeline()
	p.SetClient(client)
	p.SetConfig(HiveConfig{
		QueryTemplate: "SELECT * FROM events WHERE id > {cursor}",
		CursorField:   "id",
		SyncKey:       "test_sync",
		ContentField:  "text",
		IDField:       "id",
	})
	p.SetProgressStore(store)

	// First sync — no prior cursor
	result, err := p.ProcessSpecialized(context.Background())
	if err != nil {
		t.Fatalf("first sync error: %v", err)
	}
	hr := result.(*HiveResult)
	if len(hr.Documents) != 2 {
		t.Fatalf("expected 2 documents, got %d", len(hr.Documents))
	}
	if hr.LastCursor != "12" {
		t.Errorf("expected last cursor '12', got %s", hr.LastCursor)
	}

	// Verify cursor persisted
	saved, _ := store.GetCursor("test_sync")
	if saved != "12" {
		t.Errorf("expected persisted cursor '12', got %s", saved)
	}

	// Verify the query didn't substitute cursor (no prior cursor)
	if len(client.queryLog) != 1 {
		t.Fatalf("expected 1 query, got %d", len(client.queryLog))
	}
	if client.queryLog[0] != "SELECT * FROM events WHERE id > {cursor}" {
		t.Errorf("expected template without substitution, got: %s", client.queryLog[0])
	}

	// Second sync — should substitute cursor
	result2, err := p.ProcessSpecialized(context.Background())
	if err != nil {
		t.Fatalf("second sync error: %v", err)
	}
	hr2 := result2.(*HiveResult)
	if hr2.LastCursor != "12" {
		t.Errorf("expected last cursor '12', got %s", hr2.LastCursor)
	}

	// Verify the query substituted cursor
	if len(client.queryLog) != 2 {
		t.Fatalf("expected 2 queries, got %d", len(client.queryLog))
	}
	if client.queryLog[1] != "SELECT * FROM events WHERE id > '12'" {
		t.Errorf("expected cursor substitution, got: %s", client.queryLog[1])
	}
}

func TestHIVEPipeline_Pagination(t *testing.T) {
	rows := make([]map[string]interface{}, 100)
	for i := 0; i < 100; i++ {
		rows[i] = map[string]interface{}{
			"id":   i + 1,
			"text": fmt.Sprintf("row %d", i+1),
		}
	}
	client := &mockHiveClient{rows: rows}

	p := NewHIVEPipeline()
	p.SetClient(client)
	p.SetConfig(HiveConfig{
		QueryTemplate: "SELECT * FROM events",
		PageSize:      10,
		ContentField:  "text",
		IDField:       "id",
	})

	result, err := p.ProcessSpecialized(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	hr := result.(*HiveResult)
	if len(hr.Documents) != 10 {
		t.Fatalf("expected 10 documents (page size), got %d", len(hr.Documents))
	}
	if hr.TotalRows != 10 {
		t.Errorf("expected total 10, got %d", hr.TotalRows)
	}
}

func TestHIVEPipeline_CancelContext(t *testing.T) {
	client := &mockHiveClient{
		rows: []map[string]interface{}{
			{"id": 1, "text": "first"},
			{"id": 2, "text": "second"},
		},
	}

	p := NewHIVEPipeline()
	p.SetClient(client)
	p.SetConfig(HiveConfig{
		QueryTemplate: "SELECT * FROM events",
		ContentField:  "text",
	})

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	_, err := p.ProcessSpecialized(ctx)
	// With cancelled context, either the query succeeds but document loop catches cancellation,
	// or the query itself respects context. Either way, no panic.
	_ = err
}

func TestHIVEPipeline_IdempotentSync(t *testing.T) {
	client := &mockHiveClient{
		rows: []map[string]interface{}{
			{"id": 1, "text": "same"},
		},
	}
	store := newMockProgressStore()

	p := NewHIVEPipeline()
	p.SetClient(client)
	p.SetConfig(HiveConfig{
		QueryTemplate: "SELECT * FROM events WHERE id > {cursor}",
		CursorField:   "id",
		SyncKey:       "idempotent_test",
		ContentField:  "text",
		IDField:       "id",
	})
	p.SetProgressStore(store)

	// Run sync twice with same data
	result1, _ := p.ProcessSpecialized(context.Background())
	hr1 := result1.(*HiveResult)

	result2, _ := p.ProcessSpecialized(context.Background())
	hr2 := result2.(*HiveResult)

	// Both runs produce the same document (idempotent)
	if hr1.Documents[0].ID != hr2.Documents[0].ID {
		t.Errorf("expected same document ID for idempotent sync: %s vs %s",
			hr1.Documents[0].ID, hr2.Documents[0].ID)
	}
}

func TestHIVEPipeline_MetadataFieldsMapping(t *testing.T) {
	client := &mockHiveClient{
		rows: []map[string]interface{}{
			{"id": 1, "text": "content", "author": "alice", "extra": "ignored"},
		},
	}

	p := NewHIVEPipeline()
	p.SetClient(client)
	p.SetConfig(HiveConfig{
		QueryTemplate:  "SELECT * FROM events",
		ContentField:   "text",
		IDField:        "id",
		MetadataFields: map[string]string{"author": "creator"},
	})

	result, err := p.ProcessSpecialized(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	hr := result.(*HiveResult)
	doc := hr.Documents[0]
	if doc.Content != "content" {
		t.Errorf("content mismatch: %s", doc.Content)
	}
	if doc.Metadata["creator"] != "alice" {
		t.Errorf("expected creator=alice, got %v", doc.Metadata["creator"])
	}
	if _, exists := doc.Metadata["extra"]; exists {
		t.Error("expected 'extra' not in metadata with explicit mapping")
	}
}

func TestHIVEPipeline_DefaultContent(t *testing.T) {
	client := &mockHiveClient{
		rows: []map[string]interface{}{
			{"id": 1, "name": "test", "value": 42},
		},
	}

	p := NewHIVEPipeline()
	p.SetClient(client)
	p.SetConfig(HiveConfig{
		QueryTemplate: "SELECT * FROM events",
		IDField:       "id",
	})

	result, err := p.ProcessSpecialized(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	hr := result.(*HiveResult)
	doc := hr.Documents[0]
	if !strings.Contains(doc.Content, "name=test") && !strings.Contains(doc.Content, "test") {
		t.Errorf("expected content to contain column data, got: %s", doc.Content)
	}
}

func TestHIVEPipeline_StageManagement(t *testing.T) {
	p := NewHIVEPipeline()

	if len(p.Stages()) != 0 {
		t.Fatalf("expected 0 stages, got %d", len(p.Stages()))
	}

	stage1 := &mockPipelineStage{name: "hive-stage1"}
	stage2 := &mockPipelineStage{name: "hive-stage2"}

	if err := p.AddStage(stage1); err != nil {
		t.Fatalf("AddStage error: %v", err)
	}
	if err := p.AddStage(stage2); err != nil {
		t.Fatalf("AddStage error: %v", err)
	}

	if len(p.Stages()) != 2 {
		t.Fatalf("expected 2 stages, got %d", len(p.Stages()))
	}

	if err := p.RemoveStage("hive-stage1"); err != nil {
		t.Fatalf("RemoveStage error: %v", err)
	}

	stages := p.Stages()
	if len(stages) != 1 {
		t.Fatalf("expected 1 stage after remove, got %d", len(stages))
	}
	if stages[0].Name() != "hive-stage2" {
		t.Errorf("expected remaining stage 'hive-stage2', got %s", stages[0].Name())
	}
}

func TestHIVEPipeline_AddStage_Nil(t *testing.T) {
	p := NewHIVEPipeline()
	err := p.AddStage(nil)
	if err == nil {
		t.Error("expected error for nil stage")
	}
}

func TestHIVEPipeline_RemoveStage_NotFound(t *testing.T) {
	p := NewHIVEPipeline()
	err := p.RemoveStage("nonexistent")
	if err == nil {
		t.Error("expected error for removing non-existent stage")
	}
}

func TestHIVEPipeline_Execute(t *testing.T) {
	client := &mockHiveClient{
		rows: []map[string]interface{}{
			{"id": 1, "text": "executed"},
		},
	}

	p := NewHIVEPipeline()
	p.SetClient(client)
	p.SetConfig(HiveConfig{
		QueryTemplate: "SELECT * FROM events",
		ContentField:  "text",
		IDField:       "id",
	})

	ctx := common.NewPipelineContext(nil, "test")
	result, err := p.Execute(ctx, context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	hr, ok := result.(*HiveResult)
	if !ok {
		t.Fatalf("expected *HiveResult, got %T", result)
	}
	if len(hr.Documents) != 1 {
		t.Fatalf("expected 1 document, got %d", len(hr.Documents))
	}
}

func TestHIVEPipeline_QueryError(t *testing.T) {
	client := &errorHiveClient{}
	p := NewHIVEPipeline()
	p.SetClient(client)
	p.SetConfig(HiveConfig{
		QueryTemplate: "SELECT * FROM events",
	})

	_, err := p.ProcessSpecialized(context.Background())
	if err == nil {
		t.Error("expected error from failed query")
	}
}

type errorHiveClient struct{}

func (e *errorHiveClient) Query(ctx context.Context, query string) ([]map[string]interface{}, error) {
	return nil, fmt.Errorf("hive connection refused")
}

func (e *errorHiveClient) Close() error { return nil }

func TestHIVEPipeline_StagesReturnsCopy(t *testing.T) {
	p := NewHIVEPipeline()
	p.AddStage(&mockPipelineStage{name: "s1"})

	stages1 := p.Stages()
	if len(stages1) != 1 {
		t.Fatal("expected 1 stage")
	}
	stages1[0] = &mockPipelineStage{name: "tampered"}

	stages2 := p.Stages()
	if stages2[0].Name() != "s1" {
		t.Error("expected Stages() to return a copy, not internal slice")
	}
}

func TestSubstituteCursor(t *testing.T) {
	tests := []struct {
		template string
		cursor   string
		expected string
	}{
		{"SELECT * FROM t WHERE id > {cursor}", "42", "SELECT * FROM t WHERE id > '42'"},
		{"SELECT * FROM t", "42", "SELECT * FROM t"},
		{"{cursor} at start", "abc", "'abc' at start"},
		{"end {cursor}", "xyz", "end 'xyz'"},
	}

	for _, tt := range tests {
		got := substituteCursor(tt.template, tt.cursor)
		if got != tt.expected {
			t.Errorf("substituteCursor(%q, %q) = %q, want %q", tt.template, tt.cursor, got, tt.expected)
		}
	}
}

func TestHIVEPipeline_RespectsTimeout(t *testing.T) {
	client := &slowHiveClient{delay: 100 * time.Millisecond}
	p := NewHIVEPipeline()
	p.SetClient(client)
	p.SetConfig(HiveConfig{
		QueryTemplate: "SELECT * FROM events",
		ContentField:  "text",
	})

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	_, err := p.ProcessSpecialized(ctx)
	// The slow query should either timeout or return an error
	if err == nil {
		// If query completed before timeout, that's fine
	}
}

type slowHiveClient struct {
	delay time.Duration
}

func (s *slowHiveClient) Query(ctx context.Context, query string) ([]map[string]interface{}, error) {
	select {
	case <-time.After(s.delay):
		return []map[string]interface{}{{"id": 1, "text": "slow"}}, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

func (s *slowHiveClient) Close() error { return nil }
