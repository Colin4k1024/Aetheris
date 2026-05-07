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

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestAgentStateStorePg_SaveAndLoad verifies basic save/load semantics against
// a real PostgreSQL instance. Set TEST_POSTGRES_DSN to run.
func TestAgentStateStorePg_SaveAndLoad(t *testing.T) {
	dsn := os.Getenv("TEST_POSTGRES_DSN")
	if dsn == "" {
		t.Skip("TEST_POSTGRES_DSN not set; skipping AgentStateStorePg integration tests")
	}

	ctx := context.Background()
	store, err := NewAgentStateStorePg(ctx, dsn)
	require.NoError(t, err)
	t.Cleanup(store.Close)

	// 建表（幂等）
	_, err = store.pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS agent_states (
			agent_id   TEXT NOT NULL,
			session_id TEXT NOT NULL,
			payload    JSONB NOT NULL,
			updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
			PRIMARY KEY (agent_id, session_id)
		)`)
	require.NoError(t, err)
	t.Cleanup(func() {
		_, _ = store.pool.Exec(context.Background(), `TRUNCATE TABLE agent_states`)
	})

	agentID := "agent-pg-state-1"
	sessionID := "sess-1"
	state := &AgentState{
		AgentID:    agentID,
		SessionID:  sessionID,
		Scratchpad: "working on goal-A",
		Variables:  map[string]any{"step": 1},
	}

	require.NoError(t, store.SaveAgentState(ctx, agentID, sessionID, state))

	loaded, err := store.LoadAgentState(ctx, agentID, sessionID)
	require.NoError(t, err)
	require.NotNil(t, loaded)
	assert.Equal(t, "working on goal-A", loaded.Scratchpad)
	assert.Equal(t, map[string]any{"step": float64(1)}, loaded.Variables) // JSON round-trip converts int→float64
}

func TestAgentStateStorePg_Load_NotFound(t *testing.T) {
	dsn := os.Getenv("TEST_POSTGRES_DSN")
	if dsn == "" {
		t.Skip("TEST_POSTGRES_DSN not set; skipping AgentStateStorePg integration tests")
	}

	ctx := context.Background()
	store, err := NewAgentStateStorePg(ctx, dsn)
	require.NoError(t, err)
	t.Cleanup(store.Close)

	_, err = store.pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS agent_states (
			agent_id   TEXT NOT NULL,
			session_id TEXT NOT NULL,
			payload    JSONB NOT NULL,
			updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
			PRIMARY KEY (agent_id, session_id)
		)`)
	require.NoError(t, err)

	loaded, err := store.LoadAgentState(ctx, "no-agent", "no-session")
	require.NoError(t, err)
	assert.Nil(t, loaded)
}

func TestAgentStateStorePg_Save_Upsert(t *testing.T) {
	dsn := os.Getenv("TEST_POSTGRES_DSN")
	if dsn == "" {
		t.Skip("TEST_POSTGRES_DSN not set; skipping AgentStateStorePg integration tests")
	}

	ctx := context.Background()
	store, err := NewAgentStateStorePg(ctx, dsn)
	require.NoError(t, err)
	t.Cleanup(store.Close)

	_, err = store.pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS agent_states (
			agent_id   TEXT NOT NULL,
			session_id TEXT NOT NULL,
			payload    JSONB NOT NULL,
			updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
			PRIMARY KEY (agent_id, session_id)
		)`)
	require.NoError(t, err)
	t.Cleanup(func() {
		_, _ = store.pool.Exec(context.Background(), `TRUNCATE TABLE agent_states`)
	})

	agentID := "agent-upsert"
	sessionID := "sess-upsert"

	state1 := &AgentState{AgentID: agentID, SessionID: sessionID, Scratchpad: "running"}
	require.NoError(t, store.SaveAgentState(ctx, agentID, sessionID, state1))

	// upsert 更新
	state2 := &AgentState{
		AgentID:    agentID,
		SessionID:  sessionID,
		Scratchpad: "completed",
		Variables:  map[string]any{"step": 2, "done": true},
	}
	require.NoError(t, store.SaveAgentState(ctx, agentID, sessionID, state2))

	loaded, err := store.LoadAgentState(ctx, agentID, sessionID)
	require.NoError(t, err)
	require.NotNil(t, loaded)
	assert.Equal(t, "completed", loaded.Scratchpad)
}

func TestAgentStateStorePg_Save_NilState(t *testing.T) {
	dsn := os.Getenv("TEST_POSTGRES_DSN")
	if dsn == "" {
		t.Skip("TEST_POSTGRES_DSN not set; skipping AgentStateStorePg integration tests")
	}

	ctx := context.Background()
	store, err := NewAgentStateStorePg(ctx, dsn)
	require.NoError(t, err)
	t.Cleanup(store.Close)

	// nil state 应静默跳过，不报错
	require.NoError(t, store.SaveAgentState(ctx, "a", "s", nil))
}

// TestAgentStateStorePg_Payload_Roundtrip 验证 JSON payload 字段完整性
func TestAgentStateStorePg_Payload_Roundtrip(t *testing.T) {
	dsn := os.Getenv("TEST_POSTGRES_DSN")
	if dsn == "" {
		t.Skip("TEST_POSTGRES_DSN not set; skipping AgentStateStorePg integration tests")
	}

	ctx := context.Background()
	store, err := NewAgentStateStorePg(ctx, dsn)
	require.NoError(t, err)
	t.Cleanup(store.Close)

	_, err = store.pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS agent_states (
			agent_id   TEXT NOT NULL,
			session_id TEXT NOT NULL,
			payload    JSONB NOT NULL,
			updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
			PRIMARY KEY (agent_id, session_id)
		)`)
	require.NoError(t, err)
	t.Cleanup(func() {
		_, _ = store.pool.Exec(context.Background(), `TRUNCATE TABLE agent_states`)
	})

	state := &AgentState{
		AgentID:    "agent-rt",
		SessionID:  "sess-rt",
		Scratchpad: "paused state",
		Variables:  map[string]any{"key": "value", "count": 3},
	}
	require.NoError(t, store.SaveAgentState(ctx, "agent-rt", "sess-rt", state))

	loaded, err := store.LoadAgentState(ctx, "agent-rt", "sess-rt")
	require.NoError(t, err)
	require.NotNil(t, loaded)
	assert.Equal(t, state.Scratchpad, loaded.Scratchpad)
	assert.Equal(t, "value", loaded.Variables["key"])
}
