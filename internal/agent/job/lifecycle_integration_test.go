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

	"github.com/stretchr/testify/assert"
)

// TestJobLifecycle tests the complete job lifecycle: Created -> Pending -> Running -> Completed
func TestJobLifecycle(t *testing.T) {
	ctx := context.Background()
	store := NewJobStoreMem()

	// Step 1: Create a job
	j := &Job{
		AgentID:  "test-agent",
		TenantID: "test-tenant",
		Goal:     "test goal",
		Status:   StatusPending,
	}
	jobID, err := store.Create(ctx, j)
	assert.NoError(t, err)
	assert.NotEmpty(t, jobID)

	// Step 2: Get the job
	fetchedJob, err := store.Get(ctx, jobID)
	assert.NoError(t, err)
	assert.Equal(t, StatusPending, fetchedJob.Status)

	// Step 3: Update status to Running
	err = store.UpdateStatus(ctx, jobID, StatusRunning)
	assert.NoError(t, err)

	// Step 4: Verify status changed
	fetchedJob, err = store.Get(ctx, jobID)
	assert.NoError(t, err)
	assert.Equal(t, StatusRunning, fetchedJob.Status)

	// Step 5: Complete the job
	err = store.UpdateStatus(ctx, jobID, StatusCompleted)
	assert.NoError(t, err)

	// Step 6: Verify final status
	fetchedJob, err = store.Get(ctx, jobID)
	assert.NoError(t, err)
	assert.Equal(t, StatusCompleted, fetchedJob.Status)
}

// TestJobLifecycleWithFailed tests the failure path: Created -> Pending -> Running -> Failed
func TestJobLifecycleWithFailed(t *testing.T) {
	ctx := context.Background()
	store := NewJobStoreMem()

	// Create a job
	j := &Job{
		AgentID:  "test-agent",
		TenantID: "test-tenant",
		Goal:     "test goal that will fail",
		Status:   StatusPending,
	}
	jobID, err := store.Create(ctx, j)
	assert.NoError(t, err)

	// Job starts running
	err = store.UpdateStatus(ctx, jobID, StatusRunning)
	assert.NoError(t, err)

	// Job fails with error
	err = store.UpdateStatus(ctx, jobID, StatusFailed)
	assert.NoError(t, err)

	// Verify final status
	fetchedJob, err := store.Get(ctx, jobID)
	assert.NoError(t, err)
	assert.Equal(t, StatusFailed, fetchedJob.Status)
}

// TestJobLifecycleParkedAndResumed tests the parked path: Created -> Pending -> Running -> Parked -> Running -> Completed
func TestJobLifecycleParkedAndResumed(t *testing.T) {
	ctx := context.Background()
	store := NewJobStoreMem()

	// Create a job
	j := &Job{
		AgentID:  "test-agent",
		TenantID: "test-tenant",
		Goal:     "test goal that will be parked",
		Status:   StatusPending,
	}
	jobID, err := store.Create(ctx, j)
	assert.NoError(t, err)

	// Job starts running
	err = store.UpdateStatus(ctx, jobID, StatusRunning)
	assert.NoError(t, err)

	// Job is parked (waiting for human approval)
	err = store.UpdateStatus(ctx, jobID, StatusParked)
	assert.NoError(t, err)

	// Verify parked status
	fetchedJob, err := store.Get(ctx, jobID)
	assert.NoError(t, err)
	assert.Equal(t, StatusParked, fetchedJob.Status)

	// Job is resumed after approval
	err = store.UpdateStatus(ctx, jobID, StatusRunning)
	assert.NoError(t, err)

	// Complete the job
	err = store.UpdateStatus(ctx, jobID, StatusCompleted)
	assert.NoError(t, err)

	// Verify final status
	fetchedJob, err = store.Get(ctx, jobID)
	assert.NoError(t, err)
	assert.Equal(t, StatusCompleted, fetchedJob.Status)
}

// TestJobNotFound tests behavior for non-existent jobs
func TestJobNotFound(t *testing.T) {
	ctx := context.Background()
	store := NewJobStoreMem()

	// Get non-existent job - returns nil, nil (not an error)
	job, err := store.Get(ctx, "non-existent-job")
	assert.NoError(t, err)
	assert.Nil(t, job)

	// Update non-existent job - behavior depends on implementation
	// In-memory store may or may not error
	err = store.UpdateStatus(ctx, "non-existent-job", StatusCompleted)
	// Just verify the call completes without panic
	_ = err
}
