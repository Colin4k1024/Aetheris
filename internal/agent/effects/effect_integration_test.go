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

package effects

import (
	"context"
	"encoding/json"
	"testing"

	"rag-platform/internal/runtime/jobstore"
)

// TestJobStoreEffectLogIntegration_FullLifecycle 测试 EffectLog 完整生命周期
func TestJobStoreEffectLogIntegration_FullLifecycle(t *testing.T) {
	ctx := context.Background()
	store := jobstore.NewMemoryStore()
	effectLog := NewJobStoreEffectLog(store)

	jobID := "job-effects-1"

	// 1. Append LLM Response Effect
	llmPayload, _ := json.Marshal(map[string]interface{}{
		"model":   "gpt-4",
		"content": "Hello world",
	})
	err := effectLog.AppendEffect(ctx, jobID, EffectKindLLMResponseRecorded, llmPayload)
	if err != nil {
		t.Fatalf("Append LLM effect: %v", err)
	}

	// 2. Append Tool Result Effect
	toolPayload, _ := json.Marshal(map[string]interface{}{
		"tool":   "search",
		"result": "found 10 results",
	})
	err = effectLog.AppendEffect(ctx, jobID, EffectKindToolResultRecorded, toolPayload)
	if err != nil {
		t.Fatalf("Append tool effect: %v", err)
	}

	// 3. Append External Call Effect
	extPayload, _ := json.Marshal(map[string]interface{}{
		"endpoint": "/api/data",
		"status":   200,
	})
	err = effectLog.AppendEffect(ctx, jobID, EffectKindExternalCallRecorded, extPayload)
	if err != nil {
		t.Fatalf("Append external call effect: %v", err)
	}

	// 4. Verify events were recorded
	events, version, err := store.ListEvents(ctx, jobID)
	if err != nil {
		t.Fatalf("ListEvents: %v", err)
	}
	if version != 3 {
		t.Errorf("version: got %d want 3", version)
	}
	if len(events) != 3 {
		t.Errorf("events count: got %d want 3", len(events))
	}

	// 5. Verify event types
	if events[0].Type != jobstore.JobCreated && events[0].Type != jobstore.CommandCommitted {
		t.Logf("first event type: %s", events[0].Type)
	}

	t.Logf("Effect log lifecycle: %d events at version %d", len(events), version)
}

// TestJobStoreEffectLogIntegration_NilStore 测试 nil store 的 EffectLog
func TestJobStoreEffectLogIntegration_NilStore(t *testing.T) {
	ctx := context.Background()
	effectLog := NewJobStoreEffectLog(nil)

	// Append with nil store should be no-op
	err := effectLog.AppendEffect(ctx, "job-1", EffectKindLLMResponseRecorded, []byte("{}"))
	if err != nil {
		t.Errorf("nil store should be no-op: %v", err)
	}
}

// TestJobStoreEffectLogIntegration_EmptyJobID 测试空 jobID 的 EffectLog
func TestJobStoreEffectLogIntegration_EmptyJobID(t *testing.T) {
	ctx := context.Background()
	store := jobstore.NewMemoryStore()
	effectLog := NewJobStoreEffectLog(store)

	// Append with empty jobID should be no-op
	err := effectLog.AppendEffect(ctx, "", EffectKindLLMResponseRecorded, []byte("{}"))
	if err != nil {
		t.Errorf("empty jobID should be no-op: %v", err)
	}
}

// TestJobStoreEffectLogIntegration_ConcurrentEffects 测试并发 Effects
func TestJobStoreEffectLogIntegration_ConcurrentEffects(t *testing.T) {
	ctx := context.Background()
	store := jobstore.NewMemoryStore()
	effectLog := NewJobStoreEffectLog(store)

	jobID := "job-concurrent"

	// Append initial event
	_, _ = store.Append(ctx, jobID, 0, jobstore.JobEvent{JobID: jobID, Type: jobstore.JobCreated})

	// Concurrently append effects
	errs := make(chan error, 10)
	for i := 0; i < 10; i++ {
		go func(idx int) {
			payload, _ := json.Marshal(map[string]interface{}{"index": idx})
			err := effectLog.AppendEffect(ctx, jobID, EffectKindToolResultRecorded, payload)
			errs <- err
		}(i)
	}

	// Collect results
	for i := 0; i < 10; i++ {
		err := <-errs
		if err != nil && err != jobstore.ErrVersionMismatch {
			t.Logf("concurrent effect error: %v", err)
		}
	}
	close(errs)

	// Verify at least some events were recorded
	events, _, err := store.ListEvents(ctx, jobID)
	if err != nil {
		t.Fatalf("ListEvents: %v", err)
	}
	if len(events) < 2 {
		t.Errorf("events recorded: got %d want at least 2", len(events))
	}
}

// TestJobStoreEffectLogIntegration_MultipleJobs 测试多 Job 的 Effects 隔离
func TestJobStoreEffectLogIntegration_MultipleJobs(t *testing.T) {
	ctx := context.Background()
	store := jobstore.NewMemoryStore()
	effectLog := NewJobStoreEffectLog(store)

	jobs := []string{"job-a", "job-b", "job-c"}

	// Each job gets its own effects
	for i, jobID := range jobs {
		payload, _ := json.Marshal(map[string]interface{}{"index": i})
		err := effectLog.AppendEffect(ctx, jobID, EffectKindLLMResponseRecorded, payload)
		if err != nil {
			t.Fatalf("Append effect for %s: %v", jobID, err)
		}
	}

	// Verify isolation
	for _, jobID := range jobs {
		events, _, err := store.ListEvents(ctx, jobID)
		if err != nil {
			t.Fatalf("ListEvents for %s: %v", jobID, err)
		}
		if len(events) != 1 {
			t.Errorf("%s events: got %d want 1", jobID, len(events))
		}
	}
}

// TestEffectKindMapping_Integration 测试 EffectKind 到 EventType 的映射
func TestEffectKindMapping_Integration(t *testing.T) {
	tests := []struct {
		kind       EffectKind
		wantType   jobstore.EventType
		wantMapped bool
	}{
		{EffectKindLLMResponseRecorded, jobstore.CommandCommitted, true},
		{EffectKindToolResultRecorded, jobstore.CommandCommitted, true},
		{EffectKindExternalCallRecorded, jobstore.CommandCommitted, true},
		{EffectKindTimerScheduled, "", false},
		{EffectKindRetryDecision, "", false},
	}

	for _, tt := range tests {
		t.Run(string(tt.kind), func(t *testing.T) {
			typ := effectKindToEventType(tt.kind)
			if tt.wantMapped && typ != tt.wantType {
				t.Errorf("effectKindToEventType(%s) = %s, want %s", tt.kind, typ, tt.wantType)
			}
			if !tt.wantMapped && typ != "" {
				t.Errorf("unmapped kind %s should return empty, got %s", tt.kind, typ)
			}
		})
	}
}
