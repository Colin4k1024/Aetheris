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
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExecuteSleep(t *testing.T) {
	sys := NewMemorySystem()
	ctx := context.Background()

	// Use a very short duration to avoid test delays
	result, err := ExecuteSleep(ctx, sys, 1*time.Millisecond)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, result.DurationMs, int64(1))
}

func TestSleepEffect(t *testing.T) {
	duration := 100 * time.Millisecond
	eff := SleepEffect(duration)
	assert.Equal(t, KindSleep, eff.Kind)
	assert.Contains(t, eff.Description, "sleep")
}

func TestRecordSleepToRecorder(t *testing.T) {
	recorder := &nopRecorder{}
	ctx := context.Background()

	err := RecordSleepToRecorder(ctx, recorder, "effect-1", 100)
	require.NoError(t, err)
}

func TestRecordSleepToRecorder_NilRecorder(t *testing.T) {
	ctx := context.Background()

	err := RecordSleepToRecorder(ctx, nil, "effect-1", 100)
	assert.ErrorIs(t, err, ErrNoRecorder)
}
