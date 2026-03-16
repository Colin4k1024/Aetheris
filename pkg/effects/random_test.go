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

func TestExecuteRandomInt63(t *testing.T) {
	sys := NewMemorySystem()
	ctx := context.Background()

	val, err := ExecuteRandomInt63(ctx, sys, "test-source")
	require.NoError(t, err)
	assert.GreaterOrEqual(t, val, int64(0))
}

func TestExecuteRandomBytes(t *testing.T) {
	sys := NewMemorySystem()
	ctx := context.Background()

	bytes, err := ExecuteRandomBytes(ctx, sys, "test-source", 16)
	require.NoError(t, err)
	assert.Len(t, bytes, 16)
}

func TestExecuteRandomIntn(t *testing.T) {
	sys := NewMemorySystem()
	ctx := context.Background()

	val, err := ExecuteRandomIntn(ctx, sys, "test-source", 100)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, val, 0)
	assert.Less(t, val, 100)
}

func TestExecuteRandomIntn_Zero(t *testing.T) {
	sys := NewMemorySystem()
	ctx := context.Background()

	val, err := ExecuteRandomIntn(ctx, sys, "test-source", 0)
	require.NoError(t, err)
	assert.Equal(t, 0, val)
}

func TestExecuteRandomIntn_Negative(t *testing.T) {
	sys := NewMemorySystem()
	ctx := context.Background()

	val, err := ExecuteRandomIntn(ctx, sys, "test-source", -1)
	require.NoError(t, err)
	assert.Equal(t, 0, val)
}

func TestRandomEffect(t *testing.T) {
	values := []byte{1, 2, 3, 4}
	eff := RandomEffect("test-source", values)
	assert.Equal(t, KindRandom, eff.Kind)
	assert.NotEmpty(t, eff.IdempotencyKey)
	assert.NotEmpty(t, eff.Description)
}

func TestRecordRandomToRecorder(t *testing.T) {
	recorder := &nopRecorder{}
	ctx := context.Background()

	err := RecordRandomToRecorder(ctx, recorder, "effect-1", "test-source", []byte{1, 2, 3})
	require.NoError(t, err)
}

func TestRecordRandomToRecorder_NilRecorder(t *testing.T) {
	ctx := context.Background()

	err := RecordRandomToRecorder(ctx, nil, "effect-1", "test-source", []byte{1, 2, 3})
	assert.ErrorIs(t, err, ErrNoRecorder)
}
