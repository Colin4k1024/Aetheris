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
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupEffectStorePgPool 从 TEST_POSTGRES_DSN 创建连接池并初始化 schema。
// 若环境变量未设置，则跳过测试。
func setupEffectStorePgPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	dsn := os.Getenv("TEST_POSTGRES_DSN")
	if dsn == "" {
		t.Skip("TEST_POSTGRES_DSN not set; skipping EffectStorePg integration tests")
	}
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dsn)
	require.NoError(t, err, "connect to postgres")
	t.Cleanup(func() { pool.Close() })

	_, err = pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS effects (
			id               BIGSERIAL PRIMARY KEY,
			job_id           TEXT NOT NULL,
			command_id       TEXT,
			idempotency_key  TEXT,
			kind             TEXT NOT NULL,
			input            BYTEA,
			output           BYTEA NOT NULL,
			error            TEXT,
			metadata         JSONB,
			created_at       TIMESTAMPTZ NOT NULL DEFAULT now()
		)`)
	require.NoError(t, err, "create effects table")

	_, err = pool.Exec(ctx, `
		CREATE UNIQUE INDEX IF NOT EXISTS idx_effects_job_command
		    ON effects (job_id, command_id) WHERE command_id IS NOT NULL`)
	require.NoError(t, err)
	_, err = pool.Exec(ctx, `
		CREATE UNIQUE INDEX IF NOT EXISTS idx_effects_job_idempotency
		    ON effects (job_id, idempotency_key) WHERE idempotency_key IS NOT NULL`)
	require.NoError(t, err)

	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `TRUNCATE TABLE effects RESTART IDENTITY`)
	})

	return pool
}

func TestEffectStorePg_PutAndGetByIdempotencyKey(t *testing.T) {
	pool := setupEffectStorePgPool(t)
	ctx := context.Background()
	store := NewEffectStorePg(pool)

	rec := &EffectRecord{
		JobID:          "job-pg-1",
		CommandID:      "cmd-1",
		IdempotencyKey: "idem-key-1",
		Kind:           EffectKindTool,
		Input:          []byte(`{"q":"test"}`),
		Output:         []byte(`{"r":"ok"}`),
		CreatedAt:      time.Now().UTC(),
	}
	require.NoError(t, store.PutEffect(ctx, rec))

	got, err := store.GetEffectByJobAndIdempotencyKey(ctx, "job-pg-1", "idem-key-1")
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, "job-pg-1", got.JobID)
	assert.Equal(t, "idem-key-1", got.IdempotencyKey)
	assert.Equal(t, EffectKindTool, got.Kind)
	assert.Equal(t, []byte(`{"q":"test"}`), got.Input)
	assert.Equal(t, []byte(`{"r":"ok"}`), got.Output)
}

func TestEffectStorePg_PutAndGetByCommandID(t *testing.T) {
	pool := setupEffectStorePgPool(t)
	ctx := context.Background()
	store := NewEffectStorePg(pool)

	rec := &EffectRecord{
		JobID:     "job-pg-2",
		CommandID: "cmd-2",
		Kind:      EffectKindTool,
		Output:    []byte(`{"result":"done"}`),
		CreatedAt: time.Now().UTC(),
	}
	require.NoError(t, store.PutEffect(ctx, rec))

	got, err := store.GetEffectByJobAndCommandID(ctx, "job-pg-2", "cmd-2")
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, "cmd-2", got.CommandID)
}

func TestEffectStorePg_Get_NotFound(t *testing.T) {
	pool := setupEffectStorePgPool(t)
	ctx := context.Background()
	store := NewEffectStorePg(pool)

	got, err := store.GetEffectByJobAndIdempotencyKey(ctx, "no-job", "no-key")
	require.NoError(t, err)
	assert.Nil(t, got)

	got, err = store.GetEffectByJobAndCommandID(ctx, "no-job", "no-cmd")
	require.NoError(t, err)
	assert.Nil(t, got)
}

func TestEffectStorePg_PutEffect_Idempotent(t *testing.T) {
	pool := setupEffectStorePgPool(t)
	ctx := context.Background()
	store := NewEffectStorePg(pool)

	rec := &EffectRecord{
		JobID:          "job-pg-3",
		IdempotencyKey: "idem-3",
		Kind:           EffectKindTool,
		Output:         []byte(`{"v":1}`),
		CreatedAt:      time.Now().UTC(),
	}
	require.NoError(t, store.PutEffect(ctx, rec))

	// 重复写入同一 idempotency_key 不报错（upsert 语义）
	rec.Output = []byte(`{"v":2}`)
	require.NoError(t, store.PutEffect(ctx, rec))

	got, err := store.GetEffectByJobAndIdempotencyKey(ctx, "job-pg-3", "idem-3")
	require.NoError(t, err)
	require.NotNil(t, got)
}

func TestEffectStorePg_PutEffect_Nil(t *testing.T) {
	pool := setupEffectStorePgPool(t)
	ctx := context.Background()
	store := NewEffectStorePg(pool)

	// nil record 应静默跳过，不报错
	require.NoError(t, store.PutEffect(ctx, nil))
	// JobID 空也应跳过
	require.NoError(t, store.PutEffect(ctx, &EffectRecord{JobID: ""}))
}
