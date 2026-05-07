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

package runtime

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupCheckpointPgPool 从 TEST_POSTGRES_DSN 创建连接池并初始化 schema。
// 若环境变量未设置，则跳过测试。
func setupCheckpointPgPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	dsn := os.Getenv("TEST_POSTGRES_DSN")
	if dsn == "" {
		t.Skip("TEST_POSTGRES_DSN not set; skipping CheckpointStorePg integration tests")
	}
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dsn)
	require.NoError(t, err, "connect to postgres")
	t.Cleanup(func() { pool.Close() })

	// 建表（幂等）
	_, err = pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS checkpoints (
			id               TEXT PRIMARY KEY,
			agent_id         TEXT NOT NULL,
			session_id       TEXT NOT NULL,
			job_id           TEXT,
			task_graph_state BYTEA,
			memory_state     BYTEA,
			cursor_node      TEXT,
			payload_results  BYTEA,
			created_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
			updated_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
			expires_at       TIMESTAMPTZ
		)`)
	require.NoError(t, err, "create checkpoints table")

	// 每个测试后清理
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `TRUNCATE TABLE checkpoints`)
	})

	return pool
}

func TestCheckpointStorePg_SaveAndLoad(t *testing.T) {
	pool := setupCheckpointPgPool(t)
	ctx := context.Background()
	store := NewCheckpointStorePg(pool)

	cp := NewNodeCheckpoint("agent-pg-1", "sess-1", "job-1", "node-A", []byte("graph"), []byte("results"), nil)
	id, err := store.Save(ctx, cp)
	require.NoError(t, err)
	assert.NotEmpty(t, id)

	loaded, err := store.Load(ctx, id)
	require.NoError(t, err)
	require.NotNil(t, loaded)
	assert.Equal(t, "agent-pg-1", loaded.AgentID)
	assert.Equal(t, "sess-1", loaded.SessionID)
	assert.Equal(t, "job-1", loaded.JobID)
	assert.Equal(t, "node-A", loaded.CursorNode)
	assert.Equal(t, []byte("graph"), loaded.TaskGraphState)
	assert.Equal(t, []byte("results"), loaded.PayloadResults)
}

func TestCheckpointStorePg_Load_NotFound(t *testing.T) {
	pool := setupCheckpointPgPool(t)
	ctx := context.Background()
	store := NewCheckpointStorePg(pool)

	loaded, err := store.Load(ctx, "nonexistent-id")
	require.NoError(t, err)
	assert.Nil(t, loaded)
}

func TestCheckpointStorePg_Save_Upsert(t *testing.T) {
	pool := setupCheckpointPgPool(t)
	ctx := context.Background()
	store := NewCheckpointStorePg(pool)

	cp := NewNodeCheckpoint("agent-2", "sess-2", "job-2", "node-B", []byte("g1"), []byte("r1"), nil)
	id, err := store.Save(ctx, cp)
	require.NoError(t, err)

	// 用相同 ID upsert 更新数据
	cp.CursorNode = "node-C"
	cp.TaskGraphState = []byte("g2")
	_, err = store.Save(ctx, cp)
	require.NoError(t, err)

	loaded, err := store.Load(ctx, id)
	require.NoError(t, err)
	require.NotNil(t, loaded)
	assert.Equal(t, "node-C", loaded.CursorNode)
	assert.Equal(t, []byte("g2"), loaded.TaskGraphState)
}

func TestCheckpointStorePg_ListByAgent(t *testing.T) {
	pool := setupCheckpointPgPool(t)
	ctx := context.Background()
	store := NewCheckpointStorePg(pool)

	agentID := "agent-list-test"
	for i := range 3 {
		cp := NewNodeCheckpoint(agentID, "sess", "job", "node", []byte("g"), []byte("r"), nil)
		_ = cp
		cp2 := NewNodeCheckpoint(agentID, "sess", "job", "node", nil, nil, nil)
		cp2.AgentID = agentID
		cp2.SessionID = "sess"
		cp2.TaskGraphState = []byte{byte(i)}
		_, err := store.Save(ctx, cp2)
		require.NoError(t, err)
	}

	list, err := store.ListByAgent(ctx, agentID)
	require.NoError(t, err)
	assert.Len(t, list, 3)
	for _, item := range list {
		assert.Equal(t, agentID, item.AgentID)
	}
}

func TestCheckpointStorePg_Cleanup(t *testing.T) {
	pool := setupCheckpointPgPool(t)
	ctx := context.Background()
	store := NewCheckpointStorePg(pool)

	cp := NewNodeCheckpoint("agent-cleanup", "sess", "job", "node", []byte("g"), []byte("r"), nil)
	id, err := store.Save(ctx, cp)
	require.NoError(t, err)

	// created_at 应该是 ~now；用未来时间 cleanup，应该删掉
	count, err := store.Cleanup(ctx, time.Now().Add(time.Minute))
	require.NoError(t, err)
	assert.Greater(t, count, 0)

	// 再 Load 应该 nil
	loaded, err := store.Load(ctx, id)
	require.NoError(t, err)
	assert.Nil(t, loaded)
}
