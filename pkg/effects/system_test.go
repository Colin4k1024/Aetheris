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
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultSystem(t *testing.T) {
	// Test that DefaultSystem is set
	assert.NotNil(t, DefaultSystem)
}

func TestExecute_DefaultSystem(t *testing.T) {
	// Clear the default system history first
	Clear()

	ctx := context.Background()
	effect := NewEffect(KindTool, "test").WithIdempotencyKey("test-execute-key")

	result, err := Execute(ctx, effect)
	require.NoError(t, err)
	assert.NotEmpty(t, result.ID)
}

func TestReplay_DefaultSystem(t *testing.T) {
	// Clear first
	Clear()

	ctx := context.Background()
	effect := NewEffect(KindTool, "test-replay").WithIdempotencyKey("test-replay-key")

	result, err := Execute(ctx, effect)
	require.NoError(t, err)

	// Replay from default system
	replayed, ok := Replay(ctx, result.ID)
	assert.True(t, ok)
	assert.Equal(t, result.ID, replayed.ID)
}

func TestHistory_DefaultSystem(t *testing.T) {
	Clear()

	// Execute some effects
	Execute(context.Background(), NewEffect(KindLLM, "test").WithIdempotencyKey("hist-1"))
	Execute(context.Background(), NewEffect(KindLLM, "test2").WithIdempotencyKey("hist-2"))

	hist := History()
	assert.Len(t, hist, 2)
}

func TestClear_DefaultSystem(t *testing.T) {
	// Add some effects
	Execute(context.Background(), NewEffect(KindLLM, "test").WithIdempotencyKey("clear-1"))

	// Clear
	Clear()

	hist := History()
	assert.Len(t, hist, 0)
}

func TestRegisterSystem(t *testing.T) {
	// Save original
	original := DefaultSystem

	// Create new system
	newSys := NewMemorySystem()

	// Register new system
	RegisterSystem(newSys)
	assert.Equal(t, newSys, DefaultSystem)

	// Restore original
	RegisterSystem(original)
}

func TestNewMemorySystem(t *testing.T) {
	sys := NewMemorySystem()
	assert.NotNil(t, sys)

	// Test basic operations
	ctx := context.Background()
	effect := NewEffect(KindTool, "test").WithIdempotencyKey("mem-sys-key")

	result, err := sys.Execute(ctx, effect)
	require.NoError(t, err)
	assert.NotEmpty(t, result.ID)

	// Test History
	hist := sys.History()
	assert.Len(t, hist, 1)

	// Test Clear
	sys.Clear()
	assert.Len(t, sys.History(), 0)
}

func TestMemorySystem_Complete(t *testing.T) {
	sys := NewMemorySystem()
	ctx := context.Background()

	effect := NewEffect(KindTool, "test").WithIdempotencyKey("complete-key")
	result, err := sys.Execute(ctx, effect)
	require.NoError(t, err)

	// Complete with data
	err = sys.Complete(result.ID, map[string]any{"status": "done"})
	require.NoError(t, err)

	// Verify data is stored
	replayed, ok := sys.Replay(ctx, result.ID)
	assert.True(t, ok)
	assert.Equal(t, map[string]any{"status": "done"}, replayed.Data)
}
