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

func TestExecuteTime(t *testing.T) {
	sys := NewMemorySystem()
	ctx := context.Background()

	result, err := ExecuteTime(ctx, sys)
	require.NoError(t, err)
	assert.NotZero(t, result.UnixNano)
}

func TestTimeEffect(t *testing.T) {
	now := time.Now()
	eff := TimeEffect(now)
	assert.Equal(t, KindTime, eff.Kind)
	assert.Contains(t, eff.Description, "time")
}

func TestRecordTimeToRecorder(t *testing.T) {
	recorder := &nopRecorder{}
	ctx := context.Background()
	now := time.Now()

	err := RecordTimeToRecorder(ctx, recorder, "effect-1", now)
	require.NoError(t, err)
}

func TestRecordTimeToRecorder_NilRecorder(t *testing.T) {
	ctx := context.Background()
	now := time.Now()

	err := RecordTimeToRecorder(ctx, nil, "effect-1", now)
	assert.ErrorIs(t, err, ErrNoRecorder)
}
