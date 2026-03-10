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

package job

import (
	"context"
	"testing"
)

func TestJobStoreMem_Create_Get(t *testing.T) {
	ctx := context.Background()
	s := NewJobStoreMem()
	j := &Job{AgentID: "agent-1", Goal: "goal1", Status: StatusPending}
	id, err := s.Create(ctx, j)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if id == "" {
		t.Fatal("Create returned empty id")
	}
	got, err := s.Get(ctx, id)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got == nil || got.ID != id || got.AgentID != "agent-1" || got.Goal != "goal1" || got.Status != StatusPending {
		t.Errorf("Get: %+v", got)
	}
}

func TestJobStoreMem_ListByAgent(t *testing.T) {
	ctx := context.Background()
	s := NewJobStoreMem()
	_, _ = s.Create(ctx, &Job{AgentID: "agent-1", Goal: "g1"})
	_, _ = s.Create(ctx, &Job{AgentID: "agent-1", Goal: "g2"})
	_, _ = s.Create(ctx, &Job{AgentID: "agent-2", Goal: "g3"})
	list, err := s.ListByAgent(ctx, "agent-1", "")
	if err != nil {
		t.Fatalf("ListByAgent: %v", err)
	}
	if len(list) != 2 {
		t.Errorf("expected 2 jobs for agent-1, got %d", len(list))
	}
}

func TestJobStoreMem_UpdateStatus_UpdateCursor(t *testing.T) {
	ctx := context.Background()
	s := NewJobStoreMem()
	id, _ := s.Create(ctx, &Job{AgentID: "a1", Goal: "g"})
	if err := s.UpdateStatus(ctx, id, StatusRunning); err != nil {
		t.Fatalf("UpdateStatus: %v", err)
	}
	got, _ := s.Get(ctx, id)
	if got.Status != StatusRunning {
		t.Errorf("expected StatusRunning, got %v", got.Status)
	}
	if err := s.UpdateCursor(ctx, id, "cp-1"); err != nil {
		t.Fatalf("UpdateCursor: %v", err)
	}
	got, _ = s.Get(ctx, id)
	if got.Cursor != "cp-1" {
		t.Errorf("expected cursor cp-1, got %q", got.Cursor)
	}
}

func TestJobStoreMem_ClaimNextPending(t *testing.T) {
	ctx := context.Background()
	s := NewJobStoreMem()
	id1, _ := s.Create(ctx, &Job{AgentID: "a1", Goal: "g1"})
	id2, _ := s.Create(ctx, &Job{AgentID: "a1", Goal: "g2"})

	j, err := s.ClaimNextPending(ctx)
	if err != nil || j == nil {
		t.Fatalf("ClaimNextPending: %v, j=%v", err, j)
	}
	if j.ID != id1 || j.Status != StatusRunning {
		t.Errorf("first claim: id=%s status=%v", j.ID, j.Status)
	}

	j2, _ := s.ClaimNextPending(ctx)
	if j2 == nil || j2.ID != id2 {
		t.Errorf("second claim: %+v", j2)
	}

	j3, _ := s.ClaimNextPending(ctx)
	if j3 != nil {
		t.Errorf("expected nil when no pending, got %+v", j3)
	}
}

func TestJobStoreMem_Requeue(t *testing.T) {
	ctx := context.Background()
	s := NewJobStoreMem()
	id, _ := s.Create(ctx, &Job{AgentID: "a1", Goal: "g"})
	j, _ := s.ClaimNextPending(ctx)
	if j.ID != id {
		t.Fatalf("claimed wrong job")
	}
	if err := s.Requeue(ctx, j); err != nil {
		t.Fatalf("Requeue: %v", err)
	}
	got, _ := s.Get(ctx, id)
	if got.Status != StatusPending || got.RetryCount != 1 {
		t.Errorf("after Requeue: status=%v retry_count=%d", got.Status, got.RetryCount)
	}
	// 应能再次被 Claim
	j2, _ := s.ClaimNextPending(ctx)
	if j2 == nil || j2.ID != id {
		t.Errorf("requeued job not claimable: %+v", j2)
	}
}

func TestJobStoreMem_Get_NotFound(t *testing.T) {
	ctx := context.Background()
	s := NewJobStoreMem()
	got, err := s.Get(ctx, "nonexistent")
	if err != nil || got != nil {
		t.Errorf("Get nonexistent: err=%v got=%v", err, got)
	}
}

func TestJobStatus_String(t *testing.T) {
	if StatusPending.String() != "pending" || StatusRunning.String() != "running" ||
		StatusCompleted.String() != "completed" || StatusFailed.String() != "failed" ||
		StatusCancelled.String() != "cancelled" {
		t.Errorf("JobStatus.String mismatch")
	}
}

func TestJobStoreMem_ClaimNextPendingForWorker(t *testing.T) {
	ctx := context.Background()
	s := NewJobStoreMem()
	// 无能力要求的 Job
	id1, _ := s.Create(ctx, &Job{AgentID: "a1", Goal: "g1"})
	// 需要 llm 的 Job
	id2, _ := s.Create(ctx, &Job{AgentID: "a1", Goal: "g2", RequiredCapabilities: []string{"llm"}})
	// 需要 llm,tool 的 Job
	id3, _ := s.Create(ctx, &Job{AgentID: "a1", Goal: "g3", RequiredCapabilities: []string{"llm", "tool"}})

	// capabilities 为空等价于不按能力过滤，应拿到 id1
	j, err := s.ClaimNextPendingForWorker(ctx, "", nil, "")
	if err != nil || j == nil {
		t.Fatalf("ClaimNextPendingForWorker(nil): %v, j=%v", err, j)
	}
	if j.ID != id1 {
		t.Errorf("expected first job (no caps), got %s", j.ID)
	}

	// 仅有 tool 的 Worker 不应拿到需要 llm 或 llm+tool 的 Job，应拿到无能力要求的下一个（id2 需要 llm 不匹配，id3 需要 llm+tool 不匹配，但 id2 和 id3 都在 pending）
	j2, _ := s.ClaimNextPendingForWorker(ctx, "", []string{"tool"}, "")
	if j2 != nil {
		t.Errorf("worker [tool] should not get job requiring llm or llm+tool, got %s", j2.ID)
	}

	// 有 llm 的 Worker 可拿 id2
	j3, _ := s.ClaimNextPendingForWorker(ctx, "", []string{"llm"}, "")
	if j3 == nil || j3.ID != id2 {
		t.Errorf("worker [llm] expected id2, got %+v", j3)
	}

	// 有 llm,tool 的 Worker 可拿 id3
	j4, _ := s.ClaimNextPendingForWorker(ctx, "", []string{"llm", "tool"}, "")
	if j4 == nil || j4.ID != id3 {
		t.Errorf("worker [llm,tool] expected id3, got %+v", j4)
	}

	j5, _ := s.ClaimNextPendingForWorker(ctx, "", []string{"llm", "tool", "rag"}, "")
	if j5 != nil {
		t.Errorf("no more pending, got %+v", j5)
	}
}

// TestJob_StateTransitions 测试 Job 充血模型状态转换
func TestJob_StateTransitions(t *testing.T) {
	// Test valid transitions
	t.Run("Pending_to_Running", func(t *testing.T) {
		j := NewJob("j1", "agent-1", "tenant-1", "test goal")
		if j.Status != StatusPending {
			t.Fatalf("expected StatusPending, got %v", j.Status)
		}
		err := j.Start()
		if err != nil {
			t.Fatalf("Start() failed: %v", err)
		}
		if j.Status != StatusRunning {
			t.Errorf("expected StatusRunning, got %v", j.Status)
		}
		event := j.PendingEvent()
		if event == nil || event.Type != "job_running" {
			t.Errorf("expected job_running event, got %+v", event)
		}
	})

	t.Run("Running_to_Completed", func(t *testing.T) {
		j := NewJob("j1", "agent-1", "tenant-1", "test goal")
		_ = j.Start()
		err := j.Complete()
		if err != nil {
			t.Fatalf("Complete() failed: %v", err)
		}
		if j.Status != StatusCompleted {
			t.Errorf("expected StatusCompleted, got %v", j.Status)
		}
	})

	t.Run("Running_to_Failed", func(t *testing.T) {
		j := NewJob("j1", "agent-1", "tenant-1", "test goal")
		_ = j.Start()
		err := j.Fail()
		if err != nil {
			t.Fatalf("Fail() failed: %v", err)
		}
		if j.Status != StatusFailed {
			t.Errorf("expected StatusFailed, got %v", j.Status)
		}
	})

	t.Run("Running_to_Waiting", func(t *testing.T) {
		j := NewJob("j1", "agent-1", "tenant-1", "test goal")
		_ = j.Start()
		err := j.Wait()
		if err != nil {
			t.Fatalf("Wait() failed: %v", err)
		}
		if j.Status != StatusWaiting {
			t.Errorf("expected StatusWaiting, got %v", j.Status)
		}
	})

	t.Run("Running_to_Parked", func(t *testing.T) {
		j := NewJob("j1", "agent-1", "tenant-1", "test goal")
		_ = j.Start()
		err := j.Park("long wait")
		if err != nil {
			t.Fatalf("Park() failed: %v", err)
		}
		if j.Status != StatusParked {
			t.Errorf("expected StatusParked, got %v", j.Status)
		}
	})

	t.Run("Parked_to_Resumed", func(t *testing.T) {
		j := NewJob("j1", "agent-1", "tenant-1", "test goal")
		_ = j.Start()
		_ = j.Park("long wait")
		err := j.Resume()
		if err != nil {
			t.Fatalf("Resume() failed: %v", err)
		}
		if j.Status != StatusRunning {
			t.Errorf("expected StatusRunning after Resume, got %v", j.Status)
		}
	})

	t.Run("Running_to_Retrying", func(t *testing.T) {
		j := NewJob("j1", "agent-1", "tenant-1", "test goal")
		_ = j.Start()
		err := j.Retry()
		if err != nil {
			t.Fatalf("Retry() failed: %v", err)
		}
		if j.Status != StatusRetrying {
			t.Errorf("expected StatusRetrying, got %v", j.Status)
		}
		if j.RetryCount != 1 {
			t.Errorf("expected RetryCount=1, got %d", j.RetryCount)
		}
	})

	t.Run("Pending_to_Cancelled", func(t *testing.T) {
		j := NewJob("j1", "agent-1", "tenant-1", "test goal")
		err := j.Cancel()
		if err != nil {
			t.Fatalf("Cancel() failed: %v", err)
		}
		if j.Status != StatusCancelled {
			t.Errorf("expected StatusCancelled, got %v", j.Status)
		}
		if j.CancelRequestedAt.IsZero() {
			t.Errorf("CancelRequestedAt should be set")
		}
	})
}

// TestJob_InvalidStateTransitions 测试无效状态转换
func TestJob_InvalidStateTransitions(t *testing.T) {
	t.Run("Pending_to_Completed_invalid", func(t *testing.T) {
		j := NewJob("j1", "agent-1", "tenant-1", "test goal")
		err := j.Complete()
		if err == nil {
			t.Fatalf("expected error for invalid transition from Pending to Completed")
		}
		if !IsInvalidStateTransition(err) {
			t.Errorf("expected InvalidStateTransitionError, got %v", err)
		}
	})

	t.Run("Completed_to_Running_invalid", func(t *testing.T) {
		j := NewJob("j1", "agent-1", "tenant-1", "test goal")
		_ = j.Start()
		_ = j.Complete()
		err := j.Start()
		if err == nil {
			t.Fatalf("expected error for invalid transition from Completed to Running")
		}
	})

	t.Run("Failed_to_Completed_invalid", func(t *testing.T) {
		j := NewJob("j1", "agent-1", "tenant-1", "test goal")
		_ = j.Start()
		_ = j.Fail()
		err := j.Complete()
		if err == nil {
			t.Fatalf("expected error for invalid transition from Failed to Completed")
		}
	})

	t.Run("Cancelled_to_Running_invalid", func(t *testing.T) {
		j := NewJob("j1", "agent-1", "tenant-1", "test goal")
		_ = j.Cancel()
		err := j.Start()
		if err == nil {
			t.Fatalf("expected error for invalid transition from Cancelled to Running")
		}
	})
}

// TestJob_NewJobFactory 测试工厂方法
func TestJob_NewJobFactory(t *testing.T) {
	j := NewJob("job-123", "agent-abc", "tenant-x", "my goal")
	if j.ID != "job-123" {
		t.Errorf("expected ID job-123, got %s", j.ID)
	}
	if j.AgentID != "agent-abc" {
		t.Errorf("expected AgentID agent-abc, got %s", j.AgentID)
	}
	if j.TenantID != "tenant-x" {
		t.Errorf("expected TenantID tenant-x, got %s", j.TenantID)
	}
	if j.Goal != "my goal" {
		t.Errorf("expected Goal my goal, got %s", j.Goal)
	}
	if j.Status != StatusPending {
		t.Errorf("expected StatusPending, got %v", j.Status)
	}
	if j.CreatedAt.IsZero() {
		t.Errorf("CreatedAt should be set")
	}
	if j.UpdatedAt.IsZero() {
		t.Errorf("UpdatedAt should be set")
	}
}

// TestJobStoreMem_SetWaiting 测试 Waiting 状态
func TestJobStoreMem_SetWaiting(t *testing.T) {
	ctx := context.Background()
	s := NewJobStoreMem()
	id, _ := s.Create(ctx, &Job{AgentID: "a1", Goal: "g"})

	// 设置为 Waiting
	err := s.SetWaiting(ctx, id, "corr-key-1", "human", "need approval")
	if err != nil {
		t.Fatalf("SetWaiting failed: %v", err)
	}

	// 验证状态
	j, _ := s.Get(ctx, id)
	if j.Status != StatusWaiting {
		t.Errorf("expected StatusWaiting, got %v", j.Status)
	}

	// Waiting 的 Job 不应该被普通 Claim 拉取到
	j2, _ := s.ClaimNextPending(ctx)
	if j2 != nil {
		t.Errorf("Waiting job should not be claimable via ClaimNextPending, got %+v", j2)
	}

	// 通过 Wakeup 唤醒
	j3, err := s.WakeupJob(ctx, "corr-key-1")
	if err != nil {
		t.Fatalf("WakeupJob failed: %v", err)
	}
	if j3 == nil || j3.ID != id {
		t.Errorf("expected to wakeup job %s, got %+v", id, j3)
	}
	if j3.Status != StatusPending {
		t.Errorf("after wakeup, expected StatusPending, got %v", j3.Status)
	}
}

// TestJobStoreMem_SetParked 测试 Parked 状态
func TestJobStoreMem_SetParked(t *testing.T) {
	ctx := context.Background()
	s := NewJobStoreMem()
	id, _ := s.Create(ctx, &Job{AgentID: "a1", Goal: "g"})

	// 设置为 Parked
	err := s.SetParked(ctx, id, "corr-key-2", "webhook", "long running task")
	if err != nil {
		t.Fatalf("SetParked failed: %v", err)
	}

	// 验证状态
	j, _ := s.Get(ctx, id)
	if j.Status != StatusParked {
		t.Errorf("expected StatusParked, got %v", j.Status)
	}

	// Parked 的 Job 不应该被普通 Claim 拉取到
	j2, _ := s.ClaimNextPending(ctx)
	if j2 != nil {
		t.Errorf("Parked job should not be claimable via ClaimNextPending, got %+v", j2)
	}

	// 通过 Wakeup 唤醒
	j3, err := s.WakeupJob(ctx, "corr-key-2")
	if err != nil {
		t.Fatalf("WakeupJob failed: %v", err)
	}
	if j3 == nil || j3.ID != id {
		t.Errorf("expected to wakeup job %s, got %+v", id, j3)
	}
	if j3.Status != StatusPending {
		t.Errorf("after wakeup, expected StatusPending, got %v", j3.Status)
	}
}

// TestJobStoreMem_ClaimParkedJob 测试直接认领 Parked Job
func TestJobStoreMem_ClaimParkedJob(t *testing.T) {
	ctx := context.Background()
	s := NewJobStoreMem()
	id, _ := s.Create(ctx, &Job{AgentID: "a1", Goal: "g"})

	// 设置为 Parked
	_ = s.SetParked(ctx, id, "corr-key-3", "signal", "")

	// 通过 ClaimParkedJob 直接认领
	j, err := s.ClaimParkedJob(ctx, id)
	if err != nil {
		t.Fatalf("ClaimParkedJob failed: %v", err)
	}
	if j == nil || j.ID != id {
		t.Errorf("expected to claim parked job %s, got %+v", id, j)
	}
	if j.Status != StatusRunning {
		t.Errorf("after claim, expected StatusRunning, got %v", j.Status)
	}
}

// TestJobStoreMem_WaitingAndParked_NotInNormalPoll 测试 Parked/Waiting 不参与普通轮询
func TestJobStoreMem_WaitingAndParked_NotInNormalPoll(t *testing.T) {
	ctx := context.Background()
	s := NewJobStoreMem()

	// 创建多个 Job
	id1, _ := s.Create(ctx, &Job{AgentID: "a1", Goal: "g1"})
	id2, _ := s.Create(ctx, &Job{AgentID: "a1", Goal: "g2"})
	id3, _ := s.Create(ctx, &Job{AgentID: "a1", Goal: "g3"})

	// 认领 id1
	j1, _ := s.ClaimNextPending(ctx)
	if j1 == nil || j1.ID != id1 {
		t.Fatalf("expected to claim job %s, got %+v", id1, j1)
	}

	// 将 id2 设置为 Waiting
	_ = s.SetWaiting(ctx, id2, "corr-2", "timer", "")

	// 将 id3 设置为 Parked
	_ = s.SetParked(ctx, id3, "corr-3", "webhook", "")

	// 尝试 ClaimNextPending，应该只能拿到已完成的 id1（现在是 Running，不在 pending 队列）
	// Waiting 和 Parked 的 Job 不在 pending 队列中
	j4, _ := s.ClaimNextPending(ctx)
	if j4 != nil {
		t.Errorf("should not get Waiting/Parked job from ClaimNextPending, got %+v", j4)
	}

	// 验证 id2 和 id3 仍在等待
	j2, _ := s.Get(ctx, id2)
	j3, _ := s.Get(ctx, id3)
	if j2.Status != StatusWaiting {
		t.Errorf("id2 should be StatusWaiting, got %v", j2.Status)
	}
	if j3.Status != StatusParked {
		t.Errorf("id3 should be StatusParked, got %v", j3.Status)
	}
}
