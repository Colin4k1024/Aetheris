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
	"encoding/json"
	"sync"
	"testing"

	"github.com/cloudwego/eino/compose"
	"github.com/stretchr/testify/assert"

	"github.com/Colin4k1024/Aetheris/v2/internal/agent/planner"
	"github.com/Colin4k1024/Aetheris/v2/internal/agent/runtime"
)

// fakeJobStoreWithCursor tracks cursor and status for recovery testing
type fakeJobStoreWithCursor struct {
	mu          sync.Mutex
	statusByJob map[string]int
	cursorByJob map[string]string
}

func newFakeJobStoreWithCursor() *fakeJobStoreWithCursor {
	return &fakeJobStoreWithCursor{
		statusByJob: make(map[string]int),
		cursorByJob: make(map[string]string),
	}
}

func (f *fakeJobStoreWithCursor) UpdateCursor(ctx context.Context, jobID string, cursor string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.cursorByJob[jobID] = cursor
	return nil
}

func (f *fakeJobStoreWithCursor) UpdateStatus(ctx context.Context, jobID string, status int) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.statusByJob[jobID] = status
	return nil
}

func (f *fakeJobStoreWithCursor) GetCursor(jobID string) string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.cursorByJob[jobID]
}

func (f *fakeJobStoreWithCursor) GetStatus(jobID string) int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.statusByJob[jobID]
}

// countingNodeAdapter tracks how many times each node is executed
type countingNodeAdapter struct {
	mu       sync.Mutex
	counters map[string]int64
	results  map[string]any
}

func newCountingNodeAdapter() *countingNodeAdapter {
	return &countingNodeAdapter{
		counters: make(map[string]int64),
		results:  make(map[string]any),
	}
}

func (a *countingNodeAdapter) ToDAGNode(task *planner.TaskNode, _ *runtime.Agent) (*compose.Lambda, error) {
	return compose.InvokableLambda[*AgentDAGPayload, *AgentDAGPayload](func(ctx context.Context, p *AgentDAGPayload) (*AgentDAGPayload, error) {
		a.mu.Lock()
		a.counters[task.ID]++
		a.mu.Unlock()
		if p == nil {
			p = &AgentDAGPayload{}
		}
		if p.Results == nil {
			p.Results = make(map[string]any)
		}
		result := "result-" + task.ID
		a.results[task.ID] = result
		p.Results[task.ID] = result
		return p, nil
	}), nil
}

func (a *countingNodeAdapter) ToNodeRunner(task *planner.TaskNode, _ *runtime.Agent) (NodeRunner, error) {
	return func(ctx context.Context, p *AgentDAGPayload) (*AgentDAGPayload, error) {
		a.mu.Lock()
		a.counters[task.ID]++
		a.mu.Unlock()
		if p == nil {
			p = &AgentDAGPayload{}
		}
		if p.Results == nil {
			p.Results = make(map[string]any)
		}
		result := "result-" + task.ID
		a.results[task.ID] = result
		p.Results[task.ID] = result
		return p, nil
	}, nil
}

func (a *countingNodeAdapter) GetCount(nodeID string) int64 {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.counters[nodeID]
}

// TestCheckpointRecovery_JobContinuesFromCheckpoint tests that after a checkpoint is saved,
// recovery will continue from where the job left off and not re-execute completed steps.
func TestCheckpointRecovery_JobContinuesFromCheckpoint(t *testing.T) {
	ctx := context.Background()

	// Create counting adapter
	adapter := newCountingNodeAdapter()

	// Build task graph: n1 -> n2 -> n3
	taskGraph := &planner.TaskGraph{
		Nodes: []planner.TaskNode{
			{ID: "n1", Type: planner.NodeLLM},
			{ID: "n2", Type: planner.NodeLLM},
			{ID: "n3", Type: planner.NodeLLM},
		},
		Edges: []planner.TaskEdge{
			{From: "n1", To: "n2"},
			{From: "n2", To: "n3"},
		},
	}

	// Create in-memory checkpoint and job stores
	cpStore := runtime.NewCheckpointStoreMem()
	jobStore := newFakeJobStoreWithCursor()

	// Create compiler with our counting adapter
	compiler := NewCompiler(map[string]NodeAdapter{
		planner.NodeLLM: adapter,
	})

	agent := &runtime.Agent{ID: "agent-1"}
	job := &JobForRunner{ID: "job-recovery-test", AgentID: "agent-1", Goal: "test goal", Cursor: ""}

	graphBytes, err := taskGraph.Marshal()
	assert.NoError(t, err)

	// Simulate: save a checkpoint after completing n1 (as if job was interrupted after n1)
	// The checkpoint stores CursorNode="n1" which means execution should resume from n2
	cp := runtime.NewNodeCheckpoint("agent-1", "", job.ID, "n1", graphBytes, []byte(`{"n1":"result-n1"}`), nil)
	cpID, err := cpStore.Save(ctx, cp)
	assert.NoError(t, err)

	// Update job cursor to point to the checkpoint (simulating job interruption)
	job.Cursor = cpID

	// Now recover: create a new runner and resume from checkpoint
	runner := NewRunner(compiler)
	runner.SetCheckpointStores(cpStore, jobStore)

	err = runner.RunForJob(ctx, agent, job)
	assert.NoError(t, err)

	// Verify idempotency: n1 should NOT have been re-executed (already completed in checkpoint)
	assert.Equal(t, int64(0), adapter.GetCount("n1"), "n1 should not be re-executed after checkpoint recovery")

	// Verify n2 and n3 were executed during recovery
	assert.Equal(t, int64(1), adapter.GetCount("n2"), "n2 should be executed during recovery")
	assert.Equal(t, int64(1), adapter.GetCount("n3"), "n3 should be executed during recovery")

	// Verify job completed successfully
	status := jobStore.GetStatus(job.ID)
	assert.Equal(t, 2, status, "job should be completed (status=2)")
}

// TestCheckpointRecovery_Idempotency verifies that no step is re-executed after recovery
func TestCheckpointRecovery_Idempotency(t *testing.T) {
	ctx := context.Background()

	// Create counting adapter
	adapter := newCountingNodeAdapter()

	// Build task graph: step1 -> step2 -> step3 -> step4
	taskGraph := &planner.TaskGraph{
		Nodes: []planner.TaskNode{
			{ID: "step1", Type: planner.NodeTool, ToolName: "tool1"},
			{ID: "step2", Type: planner.NodeTool, ToolName: "tool2"},
			{ID: "step3", Type: planner.NodeTool, ToolName: "tool3"},
			{ID: "step4", Type: planner.NodeTool, ToolName: "tool4"},
		},
		Edges: []planner.TaskEdge{
			{From: "step1", To: "step2"},
			{From: "step2", To: "step3"},
			{From: "step3", To: "step4"},
		},
	}

	// Create in-memory checkpoint and job stores
	cpStore := runtime.NewCheckpointStoreMem()
	jobStore := newFakeJobStoreWithCursor()

	// Create compiler with our counting adapter
	compiler := NewCompiler(map[string]NodeAdapter{
		planner.NodeTool: adapter,
	})

	agent := &runtime.Agent{ID: "agent-2"}
	job := &JobForRunner{ID: "job-idempotency-test", AgentID: "agent-2", Goal: "test goal", Cursor: ""}

	graphBytes, err := taskGraph.Marshal()
	assert.NoError(t, err)

	// Scenario: Job was interrupted after completing step2
	// Simulate checkpoint after step2: CursorNode="step2"
	cp := runtime.NewNodeCheckpoint("agent-2", "", job.ID, "step2", graphBytes, []byte(`{"step1":"result-step1","step2":"result-step2"}`), nil)
	cpID, err := cpStore.Save(ctx, cp)
	assert.NoError(t, err)

	job.Cursor = cpID

	// Resume from checkpoint
	runner := NewRunner(compiler)
	runner.SetCheckpointStores(cpStore, jobStore)

	err = runner.RunForJob(ctx, agent, job)
	assert.NoError(t, err)

	// Idempotency check: step1 and step2 should NOT have been re-executed
	assert.Equal(t, int64(0), adapter.GetCount("step1"), "step1 should not be re-executed")
	assert.Equal(t, int64(0), adapter.GetCount("step2"), "step2 should not be re-executed")

	// step3 and step4 should have been executed
	assert.Equal(t, int64(1), adapter.GetCount("step3"), "step3 should be executed during recovery")
	assert.Equal(t, int64(1), adapter.GetCount("step4"), "step4 should be executed during recovery")

	// Verify job completed successfully
	status := jobStore.GetStatus(job.ID)
	assert.Equal(t, 2, status, "job should be completed (status=2)")

	// Verify the cursor was updated after recovery
	newCursor := jobStore.GetCursor(job.ID)
	assert.NotEmpty(t, newCursor, "cursor should be updated after recovery")
}

// TestCheckpointRecovery_ResultsPreserved verifies that payload results from checkpoint are preserved
func TestCheckpointRecovery_ResultsPreserved(t *testing.T) {
	ctx := context.Background()

	// Create counting adapter
	adapter := newCountingNodeAdapter()

	// Build task graph: n1 -> n2
	taskGraph := &planner.TaskGraph{
		Nodes: []planner.TaskNode{
			{ID: "n1", Type: planner.NodeLLM},
			{ID: "n2", Type: planner.NodeLLM},
		},
		Edges: []planner.TaskEdge{
			{From: "n1", To: "n2"},
		},
	}

	// Create in-memory checkpoint and job stores
	cpStore := runtime.NewCheckpointStoreMem()
	jobStore := newFakeJobStoreWithCursor()

	// Create compiler with our counting adapter
	compiler := NewCompiler(map[string]NodeAdapter{
		planner.NodeLLM: adapter,
	})

	agent := &runtime.Agent{ID: "agent-3"}
	job := &JobForRunner{ID: "job-results-test", AgentID: "agent-3", Goal: "test goal", Cursor: ""}

	graphBytes, err := taskGraph.Marshal()
	assert.NoError(t, err)

	// Simulate checkpoint after n1 with n1's result
	preExistingResults := map[string]any{"n1": "pre-existing-n1-result"}
	preExistingResultsBytes, err := json.Marshal(preExistingResults)
	assert.NoError(t, err)

	cp := runtime.NewNodeCheckpoint("agent-3", "", job.ID, "n1", graphBytes, preExistingResultsBytes, nil)
	cpID, err := cpStore.Save(ctx, cp)
	assert.NoError(t, err)

	job.Cursor = cpID

	// Resume from checkpoint
	runner := NewRunner(compiler)
	runner.SetCheckpointStores(cpStore, jobStore)

	err = runner.RunForJob(ctx, agent, job)
	assert.NoError(t, err)

	// n1 should not be re-executed (results come from checkpoint)
	assert.Equal(t, int64(0), adapter.GetCount("n1"), "n1 should not be re-executed")

	// n2 should be executed
	assert.Equal(t, int64(1), adapter.GetCount("n2"), "n2 should be executed")

	// Verify job completed successfully
	status := jobStore.GetStatus(job.ID)
	assert.Equal(t, 2, status, "job should be completed (status=2)")
}
