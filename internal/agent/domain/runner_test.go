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

package domain

import (
	"testing"
)

func TestNewRunner(t *testing.T) {
	runner := NewRunner("job-1", "session-1")
	if runner.JobID != "job-1" {
		t.Errorf("expected job ID job-1, got %s", runner.JobID)
	}
	if runner.SessionID != "session-1" {
		t.Errorf("expected session ID session-1, got %s", runner.SessionID)
	}
	if runner.Status != RunnerStatusIdle {
		t.Errorf("expected status idle, got %v", runner.Status)
	}
	if runner.CurrentStepIndex != -1 {
		t.Errorf("expected current step index -1, got %d", runner.CurrentStepIndex)
	}
}

func TestRunner_AddStep(t *testing.T) {
	runner := NewRunner("job-1", "session-1")

	runner.AddStep(&Step{
		ID:       "step-1",
		NodeID:   "node-1",
		NodeType: "tool",
	})

	runner.AddStep(&Step{
		ID:       "step-2",
		NodeID:   "node-2",
		NodeType: "llm",
	})

	if len(runner.Steps) != 2 {
		t.Errorf("expected 2 steps, got %d", len(runner.Steps))
	}
}

func TestRunner_ExecuteStep(t *testing.T) {
	runner := NewRunner("job-1", "session-1")
	runner.AddStep(&Step{ID: "step-1", NodeID: "node-1"})
	runner.AddStep(&Step{ID: "step-2", NodeID: "node-2"})

	// 第一次执行
	err := runner.ExecuteStep()
	if err != nil {
		t.Fatalf("ExecuteStep failed: %v", err)
	}
	if runner.CurrentStepIndex != 0 {
		t.Errorf("expected current step index 0, got %d", runner.CurrentStepIndex)
	}
	if runner.Status != RunnerStatusRunning {
		t.Errorf("expected status running, got %v", runner.Status)
	}
	currentStep := runner.CurrentStep()
	if currentStep.Status != StepStatusRunning {
		t.Errorf("expected step status running, got %v", currentStep.Status)
	}
}

func TestRunner_CompleteStep(t *testing.T) {
	runner := NewRunner("job-1", "session-1")
	runner.AddStep(&Step{ID: "step-1", NodeID: "node-1"})
	runner.ExecuteStep()

	runner.CompleteStep(map[string]interface{}{"result": "ok"})

	currentStep := runner.CurrentStep()
	if currentStep.Status != StepStatusCompleted {
		t.Errorf("expected step status completed, got %v", currentStep.Status)
	}
	if currentStep.Output["result"] != "ok" {
		t.Errorf("expected output result ok, got %v", currentStep.Output["result"])
	}
}

func TestRunner_FailStep(t *testing.T) {
	runner := NewRunner("job-1", "session-1")
	runner.AddStep(&Step{ID: "step-1", NodeID: "node-1"})
	runner.ExecuteStep()

	runner.FailStep("some error")

	if !runner.HasFailed() {
		t.Error("expected has failed")
	}
	currentStep := runner.CurrentStep()
	if currentStep.Status != StepStatusFailed {
		t.Errorf("expected step status failed, got %v", currentStep.Status)
	}
	if currentStep.Error != "some error" {
		t.Errorf("expected error 'some error', got %s", currentStep.Error)
	}
}

func TestRunner_SaveCheckpoint(t *testing.T) {
	runner := NewRunner("job-1", "session-1")
	runner.AddStep(&Step{ID: "step-1", NodeID: "node-1"})
	runner.AddStep(&Step{ID: "step-2", NodeID: "node-2"})
	runner.ExecuteStep()
	runner.CompleteStep(nil)

	cp := runner.SaveCheckpoint("cursor-1")

	if cp == nil {
		t.Fatal("expected checkpoint")
	}
	if cp.JobID != "job-1" {
		t.Errorf("expected job ID job-1, got %s", cp.JobID)
	}
	if cp.Cursor != "cursor-1" {
		t.Errorf("expected cursor cursor-1, got %s", cp.Cursor)
	}
	if cp.StepIndex != 0 {
		t.Errorf("expected step index 0, got %d", cp.StepIndex)
	}
}

func TestRunner_LoadCheckpoint(t *testing.T) {
	runner := NewRunner("job-1", "session-1")
	runner.AddStep(&Step{ID: "step-1", NodeID: "node-1"})
	runner.AddStep(&Step{ID: "step-2", NodeID: "node-2"})
	runner.ExecuteStep()
	runner.CompleteStep(nil)

	cp := runner.SaveCheckpoint("cursor-1")

	// 创建新 Runner 并从 checkpoint 恢复
	runner2 := NewRunner("job-1", "session-1")
	err := runner2.LoadCheckpoint(cp)
	if err != nil {
		t.Fatalf("LoadCheckpoint failed: %v", err)
	}
	if runner2.CurrentStepIndex != 0 {
		t.Errorf("expected current step index 0, got %d", runner2.CurrentStepIndex)
	}
	if runner2.Status != RunnerStatusPaused {
		t.Errorf("expected status paused, got %v", runner2.Status)
	}
}

func TestRunner_Progress(t *testing.T) {
	runner := NewRunner("job-1", "session-1")
	runner.AddStep(&Step{ID: "step-1"})
	runner.AddStep(&Step{ID: "step-2"})
	runner.AddStep(&Step{ID: "step-3"})

	completed, total := runner.Progress()
	if completed != 0 {
		t.Errorf("expected 0 completed, got %d", completed)
	}
	if total != 3 {
		t.Errorf("expected 3 total, got %d", total)
	}

	runner.ExecuteStep()
	runner.CompleteStep(nil)

	completed, total = runner.Progress()
	if completed != 1 {
		t.Errorf("expected 1 completed, got %d", completed)
	}

	// 完成所有步骤 - 通过多次 ExecuteStep + CompleteStep
	runner.Status = RunnerStatusCompleted
	completed, total = runner.Progress()
	if completed != 3 {
		t.Errorf("expected 3 completed after all done, got %d", completed)
	}
}

func TestRunner_IsCompleted(t *testing.T) {
	runner := NewRunner("job-1", "session-1")
	runner.AddStep(&Step{ID: "step-1"})

	if runner.IsCompleted() {
		t.Error("expected not completed initially")
	}

	runner.ExecuteStep()
	runner.CompleteStep(nil)

	if runner.IsCompleted() {
		t.Error("expected not completed after one step")
	}
}

func TestRunner_Reset(t *testing.T) {
	runner := NewRunner("job-1", "session-1")
	runner.AddStep(&Step{ID: "step-1"})
	runner.ExecuteStep()
	runner.CompleteStep(nil)
	runner.SaveCheckpoint("cursor-1")

	runner.Reset()

	if runner.Status != RunnerStatusIdle {
		t.Errorf("expected status idle, got %v", runner.Status)
	}
	if runner.CurrentStepIndex != -1 {
		t.Errorf("expected current step index -1, got %d", runner.CurrentStepIndex)
	}
	if runner.Checkpoint != nil {
		t.Error("expected checkpoint to be nil after reset")
	}
}
