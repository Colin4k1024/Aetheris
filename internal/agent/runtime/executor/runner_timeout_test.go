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

package executor

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"rag-platform/internal/agent/planner"
	"rag-platform/internal/agent/runtime"
	"rag-platform/internal/runtime/jobstore"
)

type timeoutNodeSink struct {
	mu                 sync.Mutex
	lastNodeID         string
	lastResultType     StepResultType
	lastReason         string
	lastFinishedCalled bool
}

func (s *timeoutNodeSink) AppendNodeStarted(ctx context.Context, jobID string, nodeID string, attempt int, workerID string) error {
	return nil
}

func (s *timeoutNodeSink) AppendNodeFinished(ctx context.Context, jobID string, nodeID string, payloadResults []byte, durationMs int64, state string, attempt int, resultType StepResultType, reason string, stepID string, inputHash string) error {
	s.mu.Lock()
	s.lastNodeID = nodeID
	s.lastResultType = resultType
	s.lastReason = reason
	s.lastFinishedCalled = true
	s.mu.Unlock()
	return nil
}

func (s *timeoutNodeSink) AppendStepCommitted(ctx context.Context, jobID string, nodeID string, stepID string, commandID string, idempotencyKey string) error {
	return nil
}

func (s *timeoutNodeSink) AppendStateCheckpointed(ctx context.Context, jobID string, nodeID string, stateBefore, stateAfter []byte, opts *StateCheckpointOpts) error {
	return nil
}

func (s *timeoutNodeSink) AppendJobWaiting(ctx context.Context, jobID string, nodeID string, waitKind, reason string, expiresAt time.Time, correlationKey string, resumptionContext []byte) error {
	return nil
}

func (s *timeoutNodeSink) AppendReasoningSnapshot(ctx context.Context, jobID string, payload []byte) error {
	return nil
}

func (s *timeoutNodeSink) AppendStepCompensated(ctx context.Context, jobID string, nodeID string, stepID string, commandID string, reason string) error {
	return nil
}

func (s *timeoutNodeSink) AppendMemoryRead(ctx context.Context, jobID string, nodeID string, stepIndex int, memoryType, keyOrScope, summary string) error {
	return nil
}

func (s *timeoutNodeSink) AppendMemoryWrite(ctx context.Context, jobID string, nodeID string, stepIndex int, memoryType, keyOrScope, summary string) error {
	return nil
}

func (s *timeoutNodeSink) AppendPlanEvolution(ctx context.Context, jobID string, planVersion int, diffSummary string) error {
	return nil
}

func (s *timeoutNodeSink) snapshot() (string, StepResultType, string, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.lastNodeID, s.lastResultType, s.lastReason, s.lastFinishedCalled
}

// TestRunnerParallelLevel_TimeoutClassifiedAsRetryableFailure 验证 step timeout 被映射为 retryable_failure，并写入 step timeout reason。
func TestRunnerParallelLevel_TimeoutClassifiedAsRetryableFailure(t *testing.T) {
	r := NewRunner(nil)
	r.SetStepTimeout(20 * time.Millisecond)
	jobStore := &fakeJobStoreForRunner{}
	r.jobStore = jobStore
	sink := &timeoutNodeSink{}
	r.SetNodeEventSink(sink)

	steps := []SteppableStep{{
		NodeID:   "n-timeout",
		NodeType: planner.NodeWorkflow,
		Run: func(ctx context.Context, p *AgentDAGPayload) (*AgentDAGPayload, error) {
			<-ctx.Done()
			return nil, ctx.Err()
		},
	}}
	batch := []int{0}
	g := &planner.TaskGraph{Nodes: []planner.TaskNode{{ID: "n-timeout", Type: planner.NodeWorkflow}}}
	payload := &AgentDAGPayload{Goal: "timeout-case", Results: map[string]any{}}
	j := &JobForRunner{ID: "job-timeout", AgentID: "a1"}

	err := r.runParallelLevel(context.Background(), j, steps, batch, g, payload, nil, nil, map[string]struct{}{}, nil, "d1", "")
	if err == nil {
		t.Fatal("runParallelLevel should fail on step timeout")
	}
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("error should wrap context deadline exceeded, got: %v", err)
	}

	nodeID, resultType, reason, called := sink.snapshot()
	if !called {
		t.Fatal("node_finished should be emitted on timeout")
	}
	if nodeID != "n-timeout" {
		t.Fatalf("node_finished node_id = %q, want %q", nodeID, "n-timeout")
	}
	if resultType != StepResultRetryableFailure {
		t.Fatalf("result_type = %q, want %q", resultType, StepResultRetryableFailure)
	}
	if reason != "step timeout" {
		t.Fatalf("reason = %q, want %q", reason, "step timeout")
	}
	gotJobID, gotStatus := jobStore.getLast()
	if gotJobID != "job-timeout" {
		t.Fatalf("UpdateStatus jobID = %q, want %q", gotJobID, "job-timeout")
	}
	const statusFailed = 3
	if gotStatus != statusFailed {
		t.Fatalf("UpdateStatus status = %d, want %d (Failed)", gotStatus, statusFailed)
	}
}

type staleParallelNodeSink struct{}

func (staleParallelNodeSink) AppendNodeStarted(ctx context.Context, jobID string, nodeID string, attempt int, workerID string) error {
	return nil
}

func (staleParallelNodeSink) AppendNodeFinished(ctx context.Context, jobID string, nodeID string, payloadResults []byte, durationMs int64, state string, attempt int, resultType StepResultType, reason string, stepID string, inputHash string) error {
	return jobstore.ErrStaleAttempt
}

func (staleParallelNodeSink) AppendStepCommitted(ctx context.Context, jobID string, nodeID string, stepID string, commandID string, idempotencyKey string) error {
	return nil
}

func (staleParallelNodeSink) AppendStateCheckpointed(ctx context.Context, jobID string, nodeID string, stateBefore, stateAfter []byte, opts *StateCheckpointOpts) error {
	return nil
}

func (staleParallelNodeSink) AppendJobWaiting(ctx context.Context, jobID string, nodeID string, waitKind, reason string, expiresAt time.Time, correlationKey string, resumptionContext []byte) error {
	return nil
}

func (staleParallelNodeSink) AppendReasoningSnapshot(ctx context.Context, jobID string, payload []byte) error {
	return nil
}

func (staleParallelNodeSink) AppendStepCompensated(ctx context.Context, jobID string, nodeID string, stepID string, commandID string, reason string) error {
	return nil
}

func (staleParallelNodeSink) AppendMemoryRead(ctx context.Context, jobID string, nodeID string, stepIndex int, memoryType, keyOrScope, summary string) error {
	return nil
}

func (staleParallelNodeSink) AppendMemoryWrite(ctx context.Context, jobID string, nodeID string, stepIndex int, memoryType, keyOrScope, summary string) error {
	return nil
}

func (staleParallelNodeSink) AppendPlanEvolution(ctx context.Context, jobID string, planVersion int, diffSummary string) error {
	return nil
}

// TestRunnerParallelLevel_SuccessPathNodeFinishStaleAttempt 验证并行成功分支在写 node_finished 时若被 stale-attempt 拒绝，会直接失败收敛。
func TestRunnerParallelLevel_SuccessPathNodeFinishStaleAttempt(t *testing.T) {
	r := NewRunner(nil)
	jobStore := &fakeJobStoreForRunner{}
	r.jobStore = jobStore
	r.SetNodeEventSink(staleParallelNodeSink{})

	var callCount int32
	steps := []SteppableStep{
		{NodeID: "n1", NodeType: planner.NodeWorkflow, Run: func(ctx context.Context, p *AgentDAGPayload) (*AgentDAGPayload, error) {
			atomic.AddInt32(&callCount, 1)
			p.Results["n1"] = "ok-1"
			return p, nil
		}},
		{NodeID: "n2", NodeType: planner.NodeWorkflow, Run: func(ctx context.Context, p *AgentDAGPayload) (*AgentDAGPayload, error) {
			atomic.AddInt32(&callCount, 1)
			p.Results["n2"] = "ok-2"
			return p, nil
		}},
	}
	batch := []int{0, 1}
	g := &planner.TaskGraph{Nodes: []planner.TaskNode{{ID: "n1", Type: planner.NodeWorkflow}, {ID: "n2", Type: planner.NodeWorkflow}}}
	payload := &AgentDAGPayload{Goal: "parallel-success", Results: map[string]any{}}
	j := &JobForRunner{ID: "job-parallel-stale-success", AgentID: "a1"}

	err := r.runParallelLevel(context.Background(), j, steps, batch, g, payload, &runtime.Agent{ID: "a1"}, nil, map[string]struct{}{}, nil, "d1", "")
	if !errors.Is(err, jobstore.ErrStaleAttempt) {
		t.Fatalf("expected ErrStaleAttempt, got %v", err)
	}
	if atomic.LoadInt32(&callCount) != 2 {
		t.Fatalf("expected both parallel steps to execute before stale attempt rejection, got %d", callCount)
	}
	gotJobID, gotStatus := jobStore.getLast()
	if gotJobID != "job-parallel-stale-success" {
		t.Fatalf("UpdateStatus jobID = %q, want %q", gotJobID, "job-parallel-stale-success")
	}
	const statusFailed = 3
	if gotStatus != statusFailed {
		t.Fatalf("UpdateStatus status = %d, want %d (Failed)", gotStatus, statusFailed)
	}
}

// TestRunnerParallelLevel_FailurePathNodeFinishStaleAttempt 验证并行失败分支在写失败 node_finished 时若被 stale-attempt 拒绝，会优先返回 fencing 错误。
func TestRunnerParallelLevel_FailurePathNodeFinishStaleAttempt(t *testing.T) {
	r := NewRunner(nil)
	jobStore := &fakeJobStoreForRunner{}
	r.jobStore = jobStore
	r.SetNodeEventSink(staleParallelNodeSink{})

	steps := []SteppableStep{{
		NodeID:   "n-fail",
		NodeType: planner.NodeWorkflow,
		Run: func(ctx context.Context, p *AgentDAGPayload) (*AgentDAGPayload, error) {
			return nil, context.DeadlineExceeded
		},
	}}
	batch := []int{0}
	g := &planner.TaskGraph{Nodes: []planner.TaskNode{{ID: "n-fail", Type: planner.NodeWorkflow}}}
	payload := &AgentDAGPayload{Goal: "parallel-fail", Results: map[string]any{}}
	j := &JobForRunner{ID: "job-parallel-stale-failure", AgentID: "a1"}

	err := r.runParallelLevel(context.Background(), j, steps, batch, g, payload, nil, nil, map[string]struct{}{}, nil, "d1", "")
	if !errors.Is(err, jobstore.ErrStaleAttempt) {
		t.Fatalf("expected ErrStaleAttempt, got %v", err)
	}
	gotJobID, gotStatus := jobStore.getLast()
	if gotJobID != "job-parallel-stale-failure" {
		t.Fatalf("UpdateStatus jobID = %q, want %q", gotJobID, "job-parallel-stale-failure")
	}
	const statusFailed = 3
	if gotStatus != statusFailed {
		t.Fatalf("UpdateStatus status = %d, want %d (Failed)", gotStatus, statusFailed)
	}
}
