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

//go:build integration

package executor

import (
	"context"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupInvocationPgPool 从 TEST_POSTGRES_DSN 创建连接池并初始化 schema。
func setupInvocationPgPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	dsn := os.Getenv("TEST_POSTGRES_DSN")
	if dsn == "" {
		t.Skip("TEST_POSTGRES_DSN not set; skipping ToolInvocationStorePg integration tests")
	}
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dsn)
	require.NoError(t, err)
	t.Cleanup(func() { pool.Close() })

	_, err = pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS tool_invocations (
			job_id          TEXT NOT NULL,
			idempotency_key TEXT NOT NULL,
			invocation_id   TEXT NOT NULL,
			step_id         TEXT NOT NULL,
			tool_name       TEXT NOT NULL,
			args_hash       TEXT NOT NULL,
			status          TEXT NOT NULL,
			result          BYTEA,
			committed       BOOLEAN NOT NULL DEFAULT false,
			created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
			updated_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
			confirmed_at    TIMESTAMPTZ,
			external_id     TEXT,
			PRIMARY KEY (job_id, idempotency_key)
		)`)
	require.NoError(t, err, "create tool_invocations table")

	_, err = pool.Exec(ctx, `
		CREATE INDEX IF NOT EXISTS idx_tool_invocations_job_id
		    ON tool_invocations (job_id)`)
	require.NoError(t, err)

	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `TRUNCATE TABLE tool_invocations`)
	})

	return pool
}

func TestToolInvocationStorePg_SetStartedAndGet(t *testing.T) {
	pool := setupInvocationPgPool(t)
	ctx := context.Background()
	store := NewToolInvocationStorePg(pool)

	rec := &ToolInvocationRecord{
		JobID:          "job-inv-1",
		IdempotencyKey: "idem-inv-1",
		InvocationID:   "inv-001",
		StepID:         "step-A",
		ToolName:       "web_search",
		ArgsHash:       "abc123",
		Status:         ToolInvocationStatusStarted,
	}
	require.NoError(t, store.SetStarted(ctx, rec))

	got, err := store.GetByJobAndIdempotencyKey(ctx, "job-inv-1", "idem-inv-1")
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, "inv-001", got.InvocationID)
	assert.Equal(t, "web_search", got.ToolName)
	assert.Equal(t, ToolInvocationStatusStarted, got.Status)
	assert.False(t, got.Committed)
}

func TestToolInvocationStorePg_SetFinished_Committed(t *testing.T) {
	pool := setupInvocationPgPool(t)
	ctx := context.Background()
	store := NewToolInvocationStorePg(pool)

	rec := &ToolInvocationRecord{
		JobID:          "job-inv-2",
		IdempotencyKey: "idem-inv-2",
		InvocationID:   "inv-002",
		StepID:         "step-B",
		ToolName:       "calculator",
		ArgsHash:       "def456",
		Status:         ToolInvocationStatusStarted,
	}
	require.NoError(t, store.SetStarted(ctx, rec))

	resultBytes := []byte(`{"answer":42}`)
	require.NoError(t, store.SetFinished(ctx, "idem-inv-2", ToolInvocationStatusSuccess, resultBytes, true, ""))

	got, err := store.GetByJobAndIdempotencyKey(ctx, "job-inv-2", "idem-inv-2")
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, ToolInvocationStatusSuccess, got.Status)
	assert.True(t, got.Committed)
	assert.Equal(t, resultBytes, got.Result)
}

func TestToolInvocationStorePg_SetFinished_Confirmed(t *testing.T) {
	pool := setupInvocationPgPool(t)
	ctx := context.Background()
	store := NewToolInvocationStorePg(pool)

	rec := &ToolInvocationRecord{
		JobID:          "job-inv-3",
		IdempotencyKey: "idem-inv-3",
		InvocationID:   "inv-003",
		StepID:         "step-C",
		ToolName:       "http_call",
		ArgsHash:       "ghi789",
		Status:         ToolInvocationStatusStarted,
	}
	require.NoError(t, store.SetStarted(ctx, rec))

	require.NoError(t, store.SetFinished(ctx, "idem-inv-3", ToolInvocationStatusConfirmed, []byte(`"ok"`), true, "ext-ref-123"))

	got, err := store.GetByJobAndIdempotencyKey(ctx, "job-inv-3", "idem-inv-3")
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, ToolInvocationStatusConfirmed, got.Status)
	assert.True(t, got.Committed)
}

func TestToolInvocationStorePg_Get_NotFound(t *testing.T) {
	pool := setupInvocationPgPool(t)
	ctx := context.Background()
	store := NewToolInvocationStorePg(pool)

	got, err := store.GetByJobAndIdempotencyKey(ctx, "no-job", "no-key")
	require.NoError(t, err)
	assert.Nil(t, got)
}

func TestToolInvocationStorePg_ListByJobID(t *testing.T) {
	pool := setupInvocationPgPool(t)
	ctx := context.Background()
	store := NewToolInvocationStorePg(pool)

	jobID := "job-list-inv"
	for i := range 3 {
		rec := &ToolInvocationRecord{
			JobID:          jobID,
			IdempotencyKey: "idem-list-" + string(rune('a'+i)),
			InvocationID:   "inv-list-" + string(rune('a'+i)),
			StepID:         "step",
			ToolName:       "tool",
			ArgsHash:       "hash",
			Status:         ToolInvocationStatusStarted,
		}
		require.NoError(t, store.SetStarted(ctx, rec))
	}

	list, err := store.ListByJobID(ctx, jobID)
	require.NoError(t, err)
	assert.Len(t, list, 3)
}

func TestToolInvocationStorePg_SetStarted_Nil(t *testing.T) {
	pool := setupInvocationPgPool(t)
	ctx := context.Background()
	store := NewToolInvocationStorePg(pool)

	// nil 或 empty idempotency_key 应静默跳过
	require.NoError(t, store.SetStarted(ctx, nil))
	require.NoError(t, store.SetStarted(ctx, &ToolInvocationRecord{JobID: "j"}))
}
