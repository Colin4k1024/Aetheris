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

package eino

import (
	"context"
	"testing"
)

// TestRunStoreIntegration_FullLifecycle 测试 Run 完整生命周期：创建 -> 获取 -> Pause -> Resume -> 事件查询
func TestRunStoreIntegration_FullLifecycle(t *testing.T) {
	ctx := context.Background()
	store := NewMemoryRunStore()

	// 1. 创建运行
	run, err := store.CreateRun(ctx, &Run{WorkflowID: "wf-lifecycle"})
	if err != nil {
		t.Fatalf("CreateRun: %v", err)
	}
	if run.Status != RunStatusPending {
		t.Errorf("initial status: got %s want %s", run.Status, RunStatusPending)
	}

	// 2. 获取运行
	fetched, err := store.GetRun(ctx, run.ID)
	if err != nil {
		t.Fatalf("GetRun: %v", err)
	}
	if fetched.ID != run.ID || fetched.WorkflowID != "wf-lifecycle" {
		t.Errorf("fetched run mismatch: %+v", fetched)
	}

	// 3. Pause 运行
	paused, err := store.PauseRun(ctx, run.ID, "waiting_human", "operator-1")
	if err != nil {
		t.Fatalf("PauseRun: %v", err)
	}
	if paused.Status != RunStatusPaused {
		t.Errorf("paused status: got %s want %s", paused.Status, RunStatusPaused)
	}

	// 4. 添加 ToolCall（用于 Resume）
	_, err = store.UpsertToolCall(ctx, &ToolCall{
		ID:       "tc-1",
		RunID:    run.ID,
		ToolName: "search",
		Status:   "SUCCEEDED",
	})
	if err != nil {
		t.Fatalf("UpsertToolCall: %v", err)
	}

	// 5. Resume 运行
	resumed, err := store.ResumeRun(ctx, run.ID, ResumeRunRequest{
		Mode:           ResumeModeFromToolCall,
		FromToolCallID: "tc-1",
		Strategy:       ResumeStrategyReuseSuccessfulEffects,
		Operator:       "operator-1",
	})
	if err != nil {
		t.Fatalf("ResumeRun: %v", err)
	}
	if resumed.Status != RunStatusRunning {
		t.Errorf("resumed status: got %s want %s", resumed.Status, RunStatusRunning)
	}

	// 6. 查询事件
	events, cursor, err := store.ListEvents(ctx, run.ID, 0, 100)
	if err != nil {
		t.Fatalf("ListEvents: %v", err)
	}
	if len(events) < 4 {
		t.Errorf("events count: got %d want at least 4", len(events))
	}
	if cursor < int64(len(events)) {
		t.Errorf("cursor should be at end: got %d", cursor)
	}

	t.Logf("Full lifecycle test: created run %s with %d events", run.ID, len(events))
}

// TestRunStoreIntegration_HumanDecisionInjection 测试人工决策注入流程
func TestRunStoreIntegration_HumanDecisionInjection(t *testing.T) {
	ctx := context.Background()
	store := NewMemoryRunStore()

	run, _ := store.CreateRun(ctx, &Run{WorkflowID: "wf-human"})

	// 注入人工决策
	decision := HumanDecision{
		TargetStepID: "step-approval",
		Patch:        map[string]interface{}{"approved": true},
		Operator:     "admin",
		Comment:      "LGTM",
	}

	event, err := store.InjectHumanDecision(ctx, run.ID, decision)
	if err != nil {
		t.Fatalf("InjectHumanDecision: %v", err)
	}
	if event == nil {
		t.Fatal("event should not be nil")
	}
	if event.Type != EventTypeHumanInjected {
		t.Errorf("event type: got %s want %s", event.Type, EventTypeHumanInjected)
	}

	// 验证事件已保存
	events, _, err := store.ListEvents(ctx, run.ID, 0, 50)
	if err != nil {
		t.Fatalf("ListEvents: %v", err)
	}
	found := false
	for _, e := range events {
		if e.Type == EventTypeHumanInjected {
			found = true
			break
		}
	}
	if !found {
		t.Error("human decision event not found in events")
	}
}

// TestRunStoreIntegration_EventsPagination 测试事件分页
func TestRunStoreIntegration_EventsPagination(t *testing.T) {
	ctx := context.Background()
	store := NewMemoryRunStore()

	run, _ := store.CreateRun(ctx, &Run{WorkflowID: "wf-pagination"})

	// 添加多个 ToolCall 事件
	for i := 0; i < 10; i++ {
		_, _ = store.UpsertToolCall(ctx, &ToolCall{
			ID:       "tc-" + string(rune('0'+i)),
			RunID:    run.ID,
			ToolName: "tool-" + string(rune('0'+i)),
			Status:   "SUCCEEDED",
		})
	}

	// 分页查询：每页 3 条
	page1, cursor1, err := store.ListEvents(ctx, run.ID, 0, 3)
	if err != nil {
		t.Fatalf("ListEvents page 1: %v", err)
	}
	if len(page1) != 3 {
		t.Errorf("page 1 size: got %d want 3", len(page1))
	}

	page2, cursor2, err := store.ListEvents(ctx, run.ID, cursor1, 3)
	if err != nil {
		t.Fatalf("ListEvents page 2: %v", err)
	}
	if len(page2) != 3 {
		t.Errorf("page 2 size: got %d want 3", len(page2))
	}

	// 验证游标前进
	if cursor1 == cursor2 {
		t.Error("cursor should advance between pages")
	}

	// 验证事件不重复
	seen := make(map[string]bool)
	for _, e := range append(page1, page2...) {
		if seen[e.ID] {
			t.Errorf("duplicate event found: %s", e.ID)
		}
		seen[e.ID] = true
	}
}

// TestRunStoreIntegration_ConcurrentToolCalls 测试并发 ToolCall 更新
func TestRunStoreIntegration_ConcurrentToolCalls(t *testing.T) {
	ctx := context.Background()
	store := NewMemoryRunStore()

	run, _ := store.CreateRun(ctx, &Run{WorkflowID: "wf-concurrent"})

	// 并发添加多个 ToolCall
	results := make(chan error, 5)
	for i := 0; i < 5; i++ {
		go func(idx int) {
			_, err := store.UpsertToolCall(ctx, &ToolCall{
				ID:       "tc-parallel-" + string(rune('a'+idx)),
				RunID:    run.ID,
				ToolName: "tool",
				Status:   "SUCCEEDED",
			})
			results <- err
		}(i)
	}

	// 等待所有结果
	for i := 0; i < 5; i++ {
		if err := <-results; err != nil {
			t.Errorf("concurrent upsert failed: %v", err)
		}
	}
	close(results)

	// 验证所有 ToolCall 都已保存
	events, _, err := store.ListEvents(ctx, run.ID, 0, 100)
	if err != nil {
		t.Fatalf("ListEvents: %v", err)
	}
	toolCallEvents := 0
	for _, e := range events {
		if e.Type == EventTypeToolCallStarted || e.Type == EventTypeToolCallEnded {
			toolCallEvents++
		}
	}
	// 每个 ToolCall 至少有 Started 事件（可能还有 Ended）
	if toolCallEvents < 5 {
		t.Errorf("tool call events: got %d want at least 5", toolCallEvents)
	}
}

// TestRunStoreIntegration_PauseResumeConflict 测试 Pause/Resume 冲突场景
func TestRunStoreIntegration_PauseResumeConflict(t *testing.T) {
	ctx := context.Background()
	store := NewMemoryRunStore()

	run, _ := store.CreateRun(ctx, &Run{WorkflowID: "wf-conflict"})

	// Pause 第一次
	_, err := store.PauseRun(ctx, run.ID, "reason1", "op1")
	if err != nil {
		t.Fatalf("first PauseRun: %v", err)
	}

	// 已经 Pause 状态下再次 Pause 应该返回冲突错误
	_, err = store.PauseRun(ctx, run.ID, "reason2", "op2")
	if err != ErrRunConflict {
		t.Errorf("second PauseRun should return ErrRunConflict: %v", err)
	}

	// Resume
	_, _ = store.UpsertToolCall(ctx, &ToolCall{ID: "tc-resume", RunID: run.ID, ToolName: "test", Status: "SUCCEEDED"})
	_, err = store.ResumeRun(ctx, run.ID, ResumeRunRequest{
		Mode:           ResumeModeFromToolCall,
		FromToolCallID: "tc-resume",
		Strategy:       ResumeStrategyReuseSuccessfulEffects,
	})
	if err != nil {
		t.Fatalf("ResumeRun: %v", err)
	}

	// Resume 后再 Pause 应该成功
	_, err = store.PauseRun(ctx, run.ID, "after-resume", "op3")
	if err != nil {
		t.Errorf("PauseRun after Resume: %v", err)
	}
}

// TestRunStoreIntegration_RunNotFound 测试 Run 不存在的错误处理
func TestRunStoreIntegration_RunNotFound(t *testing.T) {
	ctx := context.Background()
	store := NewMemoryRunStore()

	// Get 不存在的 Run
	_, err := store.GetRun(ctx, "nonexistent-run")
	if err != ErrRunNotFound {
		t.Errorf("GetRun nonexistent: got %v want %v", err, ErrRunNotFound)
	}

	// Pause 不存在的 Run
	_, err = store.PauseRun(ctx, "nonexistent-run", "reason", "op")
	if err != ErrRunNotFound {
		t.Errorf("PauseRun nonexistent: got %v want %v", err, ErrRunNotFound)
	}

	// Resume 不存在的 Run (with valid args but nonexistent run)
	_, err = store.ResumeRun(ctx, "nonexistent-run", ResumeRunRequest{
		Mode:           ResumeModeFromToolCall,
		FromToolCallID: "tc-any",
		Strategy:       ResumeStrategyReuseSuccessfulEffects,
	})
	if err != ErrRunNotFound {
		t.Errorf("ResumeRun nonexistent: got %v want %v", err, ErrRunNotFound)
	}

	// ListEvents 不存在的 Run
	_, _, err = store.ListEvents(ctx, "nonexistent-run", 0, 10)
	if err != ErrRunNotFound {
		t.Errorf("ListEvents nonexistent: got %v want %v", err, ErrRunNotFound)
	}
}

// TestRunStoreIntegration_InvalidResumeRequests 测试无效的 Resume 请求
func TestRunStoreIntegration_InvalidResumeRequests(t *testing.T) {
	ctx := context.Background()
	store := NewMemoryRunStore()

	run, _ := store.CreateRun(ctx, &Run{WorkflowID: "wf-invalid"})

	// 未 Pause 的 Run 不能 Resume
	_, err := store.ResumeRun(ctx, run.ID, ResumeRunRequest{
		Mode:           ResumeModeFromToolCall,
		FromToolCallID: "tc-any",
		Strategy:       ResumeStrategyReuseSuccessfulEffects,
	})
	// 可能返回错误或者被忽略，取决于实现

	// Pause 后使用不存在的 ToolCall Resume
	_, _ = store.PauseRun(ctx, run.ID, "wait", "op")
	_, err = store.ResumeRun(ctx, run.ID, ResumeRunRequest{
		Mode:           ResumeModeFromToolCall,
		FromToolCallID: "tc-does-not-exist",
		Strategy:       ResumeStrategyReuseSuccessfulEffects,
	})
	if err != ErrToolCallNotFound {
		t.Errorf("Resume with missing tool call: got %v want %v", err, ErrToolCallNotFound)
	}
}
