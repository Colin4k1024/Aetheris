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

func TestExecuteTool(t *testing.T) {
	sys := NewMemorySystem()
	ctx := context.Background()

	// Create a mock caller
	caller := func(ctx context.Context, name string, args map[string]interface{}) (map[string]interface{}, error) {
		return map[string]interface{}{"result": "success"}, nil
	}

	result, err := ExecuteTool(ctx, sys, "test_tool", map[string]interface{}{"arg": "value"}, caller)
	require.NoError(t, err)
	assert.Equal(t, map[string]interface{}{"result": "success"}, result)
}

func TestExecuteTool_WithCacher(t *testing.T) {
	sys := NewMemorySystem()
	ctx := context.Background()

	// First execution
	caller := func(ctx context.Context, name string, args map[string]interface{}) (map[string]interface{}, error) {
		return map[string]interface{}{"result": "first"}, nil
	}

	result1, err := ExecuteTool(ctx, sys, "test_tool", map[string]interface{}{"arg": "value"}, caller)
	require.NoError(t, err)
	assert.Equal(t, map[string]interface{}{"result": "first"}, result1)

	// Second execution with same args should use cached result
	caller2 := func(ctx context.Context, name string, args map[string]interface{}) (map[string]interface{}, error) {
		return map[string]interface{}{"result": "second"}, nil
	}

	result2, err := ExecuteTool(ctx, sys, "test_tool", map[string]interface{}{"arg": "value"}, caller2)
	require.NoError(t, err)
	// Should return cached result, not call caller2
	assert.Equal(t, map[string]interface{}{"result": "first"}, result2)
}

func TestExecuteToolWithTimeout(t *testing.T) {
	sys := NewMemorySystem()
	ctx := context.Background()

	timeout := 5 * time.Second
	caller := func(ctx context.Context, name string, args map[string]interface{}) (map[string]interface{}, error) {
		return map[string]interface{}{"result": "timeout-test"}, nil
	}

	result, err := ExecuteToolWithTimeout(ctx, sys, "test_tool", map[string]interface{}{}, timeout, caller)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestComputeToolIdempotencyKey(t *testing.T) {
	key := computeToolIdempotencyKey("test_tool", map[string]interface{}{"arg": "value"})
	assert.NotEmpty(t, key)
	assert.Contains(t, key, "tool:")

	// Same args should produce same key
	key2 := computeToolIdempotencyKey("test_tool", map[string]interface{}{"arg": "value"})
	assert.Equal(t, key, key2)

	// Different args should produce different key
	key3 := computeToolIdempotencyKey("test_tool", map[string]interface{}{"arg": "different"})
	assert.NotEqual(t, key, key3)
}

func TestNewToolRequest(t *testing.T) {
	req := NewToolRequest("my_tool", map[string]interface{}{"x": 1})
	assert.Equal(t, "my_tool", req.Name)
	assert.Equal(t, map[string]interface{}{"x": 1}, req.Arguments)
}

func TestToolRequest_WithTimeout(t *testing.T) {
	req := NewToolRequest("tool", nil)
	reqWithTimeout := req.WithTimeout(5 * time.Second)

	assert.NotNil(t, reqWithTimeout.Timeout)
	assert.Equal(t, 5*time.Second, *reqWithTimeout.Timeout)
}

func TestToolRequest_WithMetadata(t *testing.T) {
	req := NewToolRequest("tool", nil)
	reqWithMeta := req.WithMetadata("key", "value")

	assert.NotNil(t, reqWithMeta.Metadata)
	assert.Equal(t, "value", reqWithMeta.Metadata["key"])
}

func TestToolEffect(t *testing.T) {
	eff := ToolEffect("my_tool", map[string]interface{}{"arg": "test"})
	assert.Equal(t, KindTool, eff.Kind)
	assert.NotEmpty(t, eff.IdempotencyKey)
	assert.Contains(t, eff.Description, "my_tool")
}

func TestRecordToolToRecorder(t *testing.T) {
	recorder := &nopRecorder{}
	ctx := context.Background()

	req := NewToolRequest("tool", map[string]interface{}{"x": 1})
	response := map[string]interface{}{"result": "ok"}

	err := RecordToolToRecorder(ctx, recorder, "effect-1", "idem-key", req, response, 100*time.Millisecond)
	require.NoError(t, err)
}

func TestRecordToolToRecorder_NilRecorder(t *testing.T) {
	ctx := context.Background()
	req := NewToolRequest("tool", nil)

	err := RecordToolToRecorder(ctx, nil, "effect-1", "idem-key", req, nil, 0)
	assert.ErrorIs(t, err, ErrNoRecorder)
}

func TestNewToolError(t *testing.T) {
	err := NewToolError("execution", "tool failed", 500)
	assert.Equal(t, "execution", err.Type)
	assert.Equal(t, "tool failed", err.Message)
	assert.Equal(t, 500, err.Code)
}
