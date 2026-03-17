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
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// TestEffectStoreMem_PutAndGetByIdempotencyKey tests saving and retrieving effects by idempotency key
func TestEffectStoreMem_PutAndGetByIdempotencyKey(t *testing.T) {
	ctx := context.Background()
	store := NewEffectStoreMem()

	// Put an effect
	effect := &EffectRecord{
		JobID:          "job-1",
		CommandID:      "cmd-1",
		IdempotencyKey: "idem-1",
		Kind:            EffectKindTool,
		Input:           []byte(`{"query": "test"}`),
		Output:          []byte(`{"result": "test result"}`),
		CreatedAt:      time.Now(),
	}
	err := store.PutEffect(ctx, effect)
	assert.NoError(t, err)

	// Get effect by idempotency key
	retrieved, err := store.GetEffectByJobAndIdempotencyKey(ctx, "job-1", "idem-1")
	assert.NoError(t, err)
	assert.NotNil(t, retrieved)
	assert.Equal(t, "job-1", retrieved.JobID)
	assert.Equal(t, "idem-1", retrieved.IdempotencyKey)
	assert.Equal(t, EffectKindTool, retrieved.Kind)
}

// TestEffectStoreMem_PutAndGetByCommandID tests saving and retrieving effects by command ID
func TestEffectStoreMem_PutAndGetByCommandID(t *testing.T) {
	ctx := context.Background()
	store := NewEffectStoreMem()

	// Put an effect
	effect := &EffectRecord{
		JobID:          "job-1",
		CommandID:      "cmd-1",
		IdempotencyKey: "idem-1",
		Kind:           EffectKindLLM,
		Input:          []byte(`{"prompt": "test prompt"}`),
		Output:         []byte(`{"response": "test response"}`),
		CreatedAt:      time.Now(),
	}
	err := store.PutEffect(ctx, effect)
	assert.NoError(t, err)

	// Get effect by command ID
	retrieved, err := store.GetEffectByJobAndCommandID(ctx, "job-1", "cmd-1")
	assert.NoError(t, err)
	assert.NotNil(t, retrieved)
	assert.Equal(t, "job-1", retrieved.JobID)
	assert.Equal(t, "cmd-1", retrieved.CommandID)
	assert.Equal(t, EffectKindLLM, retrieved.Kind)
}

// TestEffectStoreMem_GetNonExistent tests getting a non-existent effect
func TestEffectStoreMem_GetNonExistent(t *testing.T) {
	ctx := context.Background()
	store := NewEffectStoreMem()

	// Try to get non-existent effect by idempotency key
	retrieved, err := store.GetEffectByJobAndIdempotencyKey(ctx, "job-1", "non-existent")
	assert.NoError(t, err)
	assert.Nil(t, retrieved)

	// Try to get non-existent effect by command ID
	retrieved, err = store.GetEffectByJobAndCommandID(ctx, "job-1", "non-existent")
	assert.NoError(t, err)
	assert.Nil(t, retrieved)
}

// TestEffectStoreMem_MultipleEffects tests multiple effects for the same job
func TestEffectStoreMem_MultipleEffects(t *testing.T) {
	ctx := context.Background()
	store := NewEffectStoreMem()

	// Put multiple effects
	effects := []*EffectRecord{
		{
			JobID:          "job-1",
			CommandID:      "cmd-1",
			IdempotencyKey: "idem-1",
			Kind:           EffectKindTool,
			Input:          []byte(`{"step": 1}`),
			CreatedAt:      time.Now(),
		},
		{
			JobID:          "job-1",
			CommandID:      "cmd-2",
			IdempotencyKey: "idem-2",
			Kind:           EffectKindLLM,
			Input:          []byte(`{"step": 2}`),
			CreatedAt:      time.Now(),
		},
		{
			JobID:          "job-1",
			CommandID:      "cmd-3",
			IdempotencyKey: "idem-3",
			Kind:           EffectKindTool,
			Input:          []byte(`{"step": 3}`),
			CreatedAt:      time.Now(),
		},
	}

	for _, e := range effects {
		err := store.PutEffect(ctx, e)
		assert.NoError(t, err)
	}

	// Verify each effect can be retrieved
	retrieved, err := store.GetEffectByJobAndIdempotencyKey(ctx, "job-1", "idem-1")
	assert.NoError(t, err)
	assert.NotNil(t, retrieved)

	retrieved, err = store.GetEffectByJobAndIdempotencyKey(ctx, "job-1", "idem-2")
	assert.NoError(t, err)
	assert.NotNil(t, retrieved)

	retrieved, err = store.GetEffectByJobAndIdempotencyKey(ctx, "job-1", "idem-3")
	assert.NoError(t, err)
	assert.NotNil(t, retrieved)
}

// TestEffectStoreMem_DifferentJobs tests effects across different jobs
func TestEffectStoreMem_DifferentJobs(t *testing.T) {
	ctx := context.Background()
	store := NewEffectStoreMem()

	// Put effects for different jobs
	effects := []*EffectRecord{
		{
			JobID:          "job-1",
			CommandID:      "cmd-1",
			IdempotencyKey: "idem-1",
			Kind:           EffectKindTool,
			CreatedAt:      time.Now(),
		},
		{
			JobID:          "job-2",
			CommandID:      "cmd-1",
			IdempotencyKey: "idem-1",
			Kind:           EffectKindTool,
			CreatedAt:      time.Now(),
		},
	}

	for _, e := range effects {
		err := store.PutEffect(ctx, e)
		assert.NoError(t, err)
	}

	// Verify effects are isolated by job
	retrieved, err := store.GetEffectByJobAndIdempotencyKey(ctx, "job-1", "idem-1")
	assert.NoError(t, err)
	assert.NotNil(t, retrieved)
	assert.Equal(t, "job-1", retrieved.JobID)

	retrieved, err = store.GetEffectByJobAndIdempotencyKey(ctx, "job-2", "idem-1")
	assert.NoError(t, err)
	assert.NotNil(t, retrieved)
	assert.Equal(t, "job-2", retrieved.JobID)
}
