package jobstore

import (
	"context"
	"sync"

	"github.com/Colin4k1024/Aetheris/v2/internal/storage/embedded"
)

type embeddedState struct {
	ByJob     map[string][]JobEvent    `json:"by_job"`
	Claims    map[string]claimRecord   `json:"claims"`
	Snapshots map[string][]JobSnapshot `json:"snapshots"`
}

// embeddedStore is a local durable JobStore backed by a JSON file.
type embeddedStore struct {
	*memoryStore
	path      string
	persistMu sync.Mutex
	snapshots map[string][]JobSnapshot
}

// NewEmbeddedStore creates an embedded local durable job event store.
func NewEmbeddedStore(path string) (JobStore, error) {
	s := &embeddedStore{
		memoryStore: &memoryStore{
			byJob:    make(map[string][]JobEvent),
			claims:   make(map[string]claimRecord),
			watchers: make(map[string][]chan JobEvent),
		},
		path:      path,
		snapshots: make(map[string][]JobSnapshot),
	}
	if err := s.load(); err != nil {
		return nil, err
	}
	return s, nil
}

func (s *embeddedStore) load() error {
	s.persistMu.Lock()
	defer s.persistMu.Unlock()
	var st embeddedState
	if err := embedded.LoadJSON(s.path, &st); err != nil {
		return err
	}
	if st.ByJob != nil {
		s.byJob = st.ByJob
	}
	if st.Claims != nil {
		s.claims = st.Claims
	}
	if st.Snapshots != nil {
		s.snapshots = st.Snapshots
	}
	return nil
}

func (s *embeddedStore) persist() error {
	s.persistMu.Lock()
	defer s.persistMu.Unlock()
	s.mu.RLock()
	st := embeddedState{
		ByJob:     s.byJob,
		Claims:    s.claims,
		Snapshots: s.snapshots,
	}
	s.mu.RUnlock()
	return embedded.SaveJSON(s.path, st)
}

func (s *embeddedStore) Append(ctx context.Context, jobID string, expectedVersion int, event JobEvent) (int, error) {
	v, err := s.memoryStore.Append(ctx, jobID, expectedVersion, event)
	if err != nil {
		return v, err
	}
	return v, s.persist()
}

func (s *embeddedStore) Claim(ctx context.Context, workerID string) (string, int, string, error) {
	jobID, version, attemptID, err := s.memoryStore.Claim(ctx, workerID)
	if err != nil {
		return jobID, version, attemptID, err
	}
	return jobID, version, attemptID, s.persist()
}

func (s *embeddedStore) ClaimJob(ctx context.Context, workerID string, jobID string) (int, string, error) {
	version, attemptID, err := s.memoryStore.ClaimJob(ctx, workerID, jobID)
	if err != nil {
		return version, attemptID, err
	}
	return version, attemptID, s.persist()
}

func (s *embeddedStore) Heartbeat(ctx context.Context, workerID string, jobID string) error {
	if err := s.memoryStore.Heartbeat(ctx, workerID, jobID); err != nil {
		return err
	}
	return s.persist()
}

func (s *embeddedStore) CreateSnapshot(ctx context.Context, jobID string, upToVersion int, snapshot []byte) error {
	s.mu.Lock()
	cp := make([]byte, len(snapshot))
	copy(cp, snapshot)
	s.snapshots[jobID] = append(s.snapshots[jobID], JobSnapshot{
		JobID:    jobID,
		Version:  upToVersion,
		Snapshot: cp,
	})
	s.mu.Unlock()
	return s.persist()
}

func (s *embeddedStore) GetLatestSnapshot(ctx context.Context, jobID string) (*JobSnapshot, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	arr := s.snapshots[jobID]
	if len(arr) == 0 {
		return nil, nil
	}
	out := arr[len(arr)-1]
	cp := make([]byte, len(out.Snapshot))
	copy(cp, out.Snapshot)
	out.Snapshot = cp
	return &out, nil
}

func (s *embeddedStore) DeleteSnapshotsBefore(ctx context.Context, jobID string, beforeVersion int) error {
	s.mu.Lock()
	arr := s.snapshots[jobID]
	if len(arr) == 0 {
		s.mu.Unlock()
		return nil
	}
	next := arr[:0]
	for _, snap := range arr {
		if snap.Version >= beforeVersion {
			next = append(next, snap)
		}
	}
	s.snapshots[jobID] = next
	s.mu.Unlock()
	return s.persist()
}
