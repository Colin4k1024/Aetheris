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

package runtime

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

// fakeCheckpointStore implements CheckpointStore for testing
type fakeCheckpointStore struct {
	checkpoints map[string][]byte
}

func (f *fakeCheckpointStore) Save(ctx context.Context, stepKey string, data []byte) error {
	if f.checkpoints == nil {
		f.checkpoints = make(map[string][]byte)
	}
	f.checkpoints[stepKey] = data
	return nil
}

func (f *fakeCheckpointStore) Load(ctx context.Context, stepKey string) ([]byte, error) {
	if f.checkpoints == nil {
		return nil, nil
	}
	data, ok := f.checkpoints[stepKey]
	if !ok {
		return nil, nil
	}
	return data, nil
}

func (f *fakeCheckpointStore) Delete(ctx context.Context, stepKey string) error {
	if f.checkpoints != nil {
		delete(f.checkpoints, stepKey)
	}
	return nil
}

// TestCheckpointSaveAndLoad tests saving and loading checkpoints
func TestCheckpointSaveAndLoad(t *testing.T) {
	ctx := context.Background()
	store := &fakeCheckpointStore{}

	// Save a checkpoint
	testData := []byte(`{"step": 1, "state": "test_state"}`)
	err := store.Save(ctx, "job-1/step-1", testData)
	assert.NoError(t, err)

	// Load the checkpoint
	loadedData, err := store.Load(ctx, "job-1/step-1")
	assert.NoError(t, err)
	assert.Equal(t, testData, loadedData)
}

// TestCheckpointLoadNonExistent tests loading a non-existent checkpoint
func TestCheckpointLoadNonExistent(t *testing.T) {
	ctx := context.Background()
	store := &fakeCheckpointStore{}

	// Load non-existent checkpoint
	loadedData, err := store.Load(ctx, "non-existent-key")
	assert.NoError(t, err)
	assert.Nil(t, loadedData)
}

// TestCheckpointOverwrite tests overwriting an existing checkpoint
func TestCheckpointOverwrite(t *testing.T) {
	ctx := context.Background()
	store := &fakeCheckpointStore{}

	// Save initial checkpoint
	initialData := []byte(`{"step": 1, "state": "initial"}`)
	err := store.Save(ctx, "job-1/step-1", initialData)
	assert.NoError(t, err)

	// Overwrite with new data
	newData := []byte(`{"step": 2, "state": "updated"}`)
	err = store.Save(ctx, "job-1/step-1", newData)
	assert.NoError(t, err)

	// Verify new data is loaded
	loadedData, err := store.Load(ctx, "job-1/step-1")
	assert.NoError(t, err)
	assert.Equal(t, newData, loadedData)
}

// TestCheckpointDelete tests deleting a checkpoint
func TestCheckpointDelete(t *testing.T) {
	ctx := context.Background()
	store := &fakeCheckpointStore{}

	// Save a checkpoint
	testData := []byte(`{"step": 1, "state": "test"}`)
	err := store.Save(ctx, "job-1/step-1", testData)
	assert.NoError(t, err)

	// Delete the checkpoint
	err = store.Delete(ctx, "job-1/step-1")
	assert.NoError(t, err)

	// Verify it's deleted
	loadedData, err := store.Load(ctx, "job-1/step-1")
	assert.NoError(t, err)
	assert.Nil(t, loadedData)
}

// TestMultipleJobsCheckpoints tests checkpoints for multiple jobs
func TestMultipleJobsCheckpoints(t *testing.T) {
	ctx := context.Background()
	store := &fakeCheckpointStore{}

	// Save checkpoints for multiple jobs
	jobs := []struct {
		key  string
		data []byte
	}{
		{"job-1/step-1", []byte(`{"job": 1, "step": 1}`)},
		{"job-1/step-2", []byte(`{"job": 1, "step": 2}`)},
		{"job-2/step-1", []byte(`{"job": 2, "step": 1}`)},
	}

	for _, j := range jobs {
		err := store.Save(ctx, j.key, j.data)
		assert.NoError(t, err)
	}

	// Verify all checkpoints exist
	for _, j := range jobs {
		loadedData, err := store.Load(ctx, j.key)
		assert.NoError(t, err)
		assert.Equal(t, j.data, loadedData)
	}
}
