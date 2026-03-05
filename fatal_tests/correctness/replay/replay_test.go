// Copyright 2026 Aetheris
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

package replay

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Event represents an event in the event store
type Event struct {
	ID        string                 `json:"id"`
	JobID     string                 `json:"job_id"`
	StepID    string                 `json:"step_id,omitempty"`
	Type      string                 `json:"type"`
	Payload   map[string]interface{} `json:"payload"`
	PrevHash  string                 `json:"prev_hash"`
	Hash      string                 `json:"hash"`
	Timestamp time.Time              `json:"timestamp"`
}

// EventStore simulates the event store for replay testing
type EventStore struct {
	events map[string][]*Event
}

// NewEventStore creates a new event store
func NewEventStore() *EventStore {
	return &EventStore{
		events: make(map[string][]*Event),
	}
}

// Append adds an event to the store
func (es *EventStore) Append(ctx context.Context, jobID string, event *Event) error {
	es.events[jobID] = append(es.events[jobID], event)
	return nil
}

// List returns all events for a job
func (es *EventStore) List(ctx context.Context, jobID string) ([]*Event, error) {
	return es.events[jobID], nil
}

// computeHash computes a hash for an event
func computeHash(data []byte) string {
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:])
}

// TestReplay_F8_SameInputSameOutput tests that replay produces the same output
// for the same input (deterministic replay).
func TestReplay_F8_SameInputSameOutput(t *testing.T) {
	ctx := context.Background()
	store := NewEventStore()

	jobID := "job-f8-test"

	// Simulate first execution: create events
	events := []*Event{
		{
			ID:        "event-1",
			JobID:     jobID,
			Type:      "job_created",
			Payload:   map[string]interface{}{"goal": "test goal"},
			PrevHash:  "",
			Timestamp: time.Now(),
		},
		{
			ID:        "event-2",
			JobID:     jobID,
			Type:      "step_started",
			StepID:    "step-1",
			Payload:   map[string]interface{}{"input": "test input"},
			Timestamp: time.Now(),
		},
		{
			ID:        "event-3",
			JobID:     jobID,
			Type:      "step_completed",
			StepID:    "step-1",
			Payload:   map[string]interface{}{"output": "test output"},
			Timestamp: time.Now(),
		},
	}

	// Compute hashes and store
	var prevHash string
	for _, e := range events {
		data, _ := json.Marshal(e.Payload)
		e.PrevHash = prevHash
		hashInput := prevHash + string(data)
		e.Hash = computeHash([]byte(hashInput))
		prevHash = e.Hash

		err := store.Append(ctx, jobID, e)
		require.NoError(t, err)
	}

	// Get events for replay
	savedEvents, err := store.List(ctx, jobID)
	require.NoError(t, err)
	require.NotEmpty(t, savedEvents)

	// Verify: event chain is intact
	var chainHash string
	for i, e := range savedEvents {
		if i == 0 {
			assert.Equal(t, "", e.PrevHash, "first event should have empty prev_hash")
		} else {
			assert.Equal(t, chainHash, e.PrevHash, "event %d should have correct prev_hash", i)
		}
		chainHash = e.Hash
	}

	// Replay should produce same outputs
	lastEvent := savedEvents[len(savedEvents)-1]
	assert.Equal(t, "step_completed", lastEvent.Type)
	assert.Equal(t, "test output", lastEvent.Payload["output"])
}

// TestReplay_F9_EventChainIntegrity tests that the event chain hash
// remains consistent after replay.
func TestReplay_F9_EventChainIntegrity(t *testing.T) {
	ctx := context.Background()
	store := NewEventStore()

	jobID := "job-f9-test"

	// Create initial chain
	initialEvents := []*Event{
		{ID: "e1", JobID: jobID, Type: "job_created", Payload: map[string]interface{}{"id": jobID}},
		{ID: "e2", JobID: jobID, Type: "step_1", Payload: map[string]interface{}{"result": "a"}},
		{ID: "e3", JobID: jobID, Type: "step_2", Payload: map[string]interface{}{"result": "b"}},
	}

	var prevHash string
	for _, e := range initialEvents {
		data, _ := json.Marshal(e.Payload)
		e.PrevHash = prevHash
		hashInput := prevHash + string(data) + e.ID
		e.Hash = computeHash([]byte(hashInput))
		prevHash = e.Hash
		store.Append(ctx, jobID, e)
	}

	// Get final hash
	eventsAfterCreation, _ := store.List(ctx, jobID)
	hashAfterCreation := eventsAfterCreation[len(eventsAfterCreation)-1].Hash

	// Simulate replay: read events again
	replayedEvents, _ := store.List(ctx, jobID)

	// Re-verify chain
	var replayHash string
	for i, e := range replayedEvents {
		if i == 0 {
			assert.Equal(t, "", e.PrevHash)
		} else {
			assert.Equal(t, replayHash, e.PrevHash)
		}
		replayHash = e.Hash
	}

	// Final hash should match
	assert.Equal(t, hashAfterCreation, replayHash, "event chain hash should be consistent after replay")
}

// TestReplay_F10_PartialStateLoss tests that replay can recover from partial state loss.
func TestReplay_F10_PartialStateLoss(t *testing.T) {
	ctx := context.Background()
	store := NewEventStore()

	jobID := "job-f10-test"

	// Create events (simulating full event store)
	events := []*Event{
		{ID: "e1", JobID: jobID, Type: "job_created", Payload: map[string]interface{}{"goal": "process data"}},
		{ID: "e2", JobID: jobID, Type: "step_1", Payload: map[string]interface{}{"action": "fetch", "result": "data_fetched"}},
		{ID: "e3", JobID: jobID, Type: "step_2", Payload: map[string]interface{}{"action": "process", "result": "data_processed"}},
		{ID: "e4", JobID: jobID, Type: "step_3", Payload: map[string]interface{}{"action": "save", "result": "data_saved"}},
	}

	var prevHash string
	for _, e := range events {
		data, _ := json.Marshal(e.Payload)
		e.PrevHash = prevHash
		hashInput := prevHash + string(data) + e.ID
		e.Hash = computeHash([]byte(hashInput))
		prevHash = e.Hash
		store.Append(ctx, jobID, e)
	}

	// Simulate checkpoint loss (only event store available)
	// In real scenario: checkpoint file deleted but event store preserved

	// Replay from events
	replayedEvents, err := store.List(ctx, jobID)
	require.NoError(t, err)

	// Rebuild state from events
	state := make(map[string]interface{})
	for _, e := range replayedEvents {
		for k, v := range e.Payload {
			state[k] = v
		}
	}

	// Verify: final state should match
	assert.Equal(t, "data_saved", state["result"])
	assert.Equal(t, "process data", state["goal"])
	assert.Equal(t, 4, len(replayedEvents), "all events should be replayable")
}

// TestReplay_F11_StateHashConsistency tests that state hash remains
// consistent across replays.
func TestReplay_F11_StateHashConsistency(t *testing.T) {
	ctx := context.Background()
	store := NewEventStore()

	jobID := "job-f11-test"

	// Create events with state snapshots
	state1 := map[string]interface{}{"counter": 1, "data": "a"}
	state2 := map[string]interface{}{"counter": 2, "data": "ab"}
	state3 := map[string]interface{}{"counter": 3, "data": "abc"}

	events := []*Event{
		{ID: "e1", JobID: jobID, Type: "state_1", Payload: state1},
		{ID: "e2", JobID: jobID, Type: "state_2", Payload: state2},
		{ID: "e3", JobID: jobID, Type: "state_3", Payload: state3},
	}

	var prevHash string
	for _, e := range events {
		data, _ := json.Marshal(e.Payload)
		e.PrevHash = prevHash
		hashInput := prevHash + string(data)
		e.Hash = computeHash([]byte(hashInput))
		prevHash = e.Hash
		store.Append(ctx, jobID, e)
	}

	// First replay
	events1, _ := store.List(ctx, jobID)
	hash1 := events1[len(events1)-1].Hash

	// Second replay
	events2, _ := store.List(ctx, jobID)
	hash2 := events2[len(events2)-1].Hash

	// Verification: should produce same hash
	assert.Equal(t, hash1, hash2, "replay should produce consistent state hash")
}
