package job

import (
	"context"
	"sync"
	"time"

	"github.com/Colin4k1024/Aetheris/v2/internal/storage/embedded"
)

type embeddedJobState struct {
	ByID       map[string]*Job         `json:"by_id"`
	Pending    []string                `json:"pending"`
	Waiting    map[string]*waitingInfo `json:"waiting"`
	WaitingKey map[string]*waitingInfo `json:"waiting_key"`
}

// JobStoreEmbedded is a local durable JobStore backed by a JSON file.
type JobStoreEmbedded struct {
	*JobStoreMem
	path      string
	persistMu sync.Mutex
}

func NewJobStoreEmbedded(path string) (*JobStoreEmbedded, error) {
	s := &JobStoreEmbedded{
		JobStoreMem: NewJobStoreMem(),
		path:        path,
	}
	if err := s.load(); err != nil {
		return nil, err
	}
	return s, nil
}

func (s *JobStoreEmbedded) load() error {
	s.persistMu.Lock()
	defer s.persistMu.Unlock()
	var st embeddedJobState
	if err := embedded.LoadJSON(s.path, &st); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if st.ByID != nil {
		s.byID = st.ByID
	}
	if st.Pending != nil {
		s.pending = st.Pending
	}
	if st.Waiting != nil {
		s.waiting = st.Waiting
	}
	if st.WaitingKey != nil {
		s.waitingByKey = st.WaitingKey
	}
	return nil
}

func (s *JobStoreEmbedded) persist() error {
	s.persistMu.Lock()
	defer s.persistMu.Unlock()
	s.mu.Lock()
	st := embeddedJobState{
		ByID:       s.byID,
		Pending:    s.pending,
		Waiting:    s.waiting,
		WaitingKey: s.waitingByKey,
	}
	s.mu.Unlock()
	return embedded.SaveJSON(s.path, st)
}

func (s *JobStoreEmbedded) Create(ctx context.Context, j *Job) (string, error) {
	id, err := s.JobStoreMem.Create(ctx, j)
	if err != nil {
		return id, err
	}
	return id, s.persist()
}

func (s *JobStoreEmbedded) UpdateStatus(ctx context.Context, jobID string, status JobStatus) error {
	if err := s.JobStoreMem.UpdateStatus(ctx, jobID, status); err != nil {
		return err
	}
	return s.persist()
}

func (s *JobStoreEmbedded) UpdateCursor(ctx context.Context, jobID string, cursor string) error {
	if err := s.JobStoreMem.UpdateCursor(ctx, jobID, cursor); err != nil {
		return err
	}
	return s.persist()
}

func (s *JobStoreEmbedded) ClaimNextPending(ctx context.Context) (*Job, error) {
	j, err := s.JobStoreMem.ClaimNextPending(ctx)
	if err != nil || j == nil {
		return j, err
	}
	return j, s.persist()
}

func (s *JobStoreEmbedded) ClaimNextPendingFromQueue(ctx context.Context, queueClass string) (*Job, error) {
	j, err := s.JobStoreMem.ClaimNextPendingFromQueue(ctx, queueClass)
	if err != nil || j == nil {
		return j, err
	}
	return j, s.persist()
}

func (s *JobStoreEmbedded) ClaimNextPendingForWorker(ctx context.Context, queueClass string, workerCapabilities []string, tenantID string) (*Job, error) {
	j, err := s.JobStoreMem.ClaimNextPendingForWorker(ctx, queueClass, workerCapabilities, tenantID)
	if err != nil || j == nil {
		return j, err
	}
	return j, s.persist()
}

func (s *JobStoreEmbedded) Requeue(ctx context.Context, j *Job) error {
	if err := s.JobStoreMem.Requeue(ctx, j); err != nil {
		return err
	}
	return s.persist()
}

func (s *JobStoreEmbedded) RequestCancel(ctx context.Context, jobID string) error {
	if err := s.JobStoreMem.RequestCancel(ctx, jobID); err != nil {
		return err
	}
	return s.persist()
}

func (s *JobStoreEmbedded) ReclaimOrphanedJobs(ctx context.Context, olderThan time.Duration) (int, error) {
	n, err := s.JobStoreMem.ReclaimOrphanedJobs(ctx, olderThan)
	if err != nil {
		return n, err
	}
	return n, s.persist()
}

func (s *JobStoreEmbedded) SetWaiting(ctx context.Context, jobID, correlationKey, waitType, reason string) error {
	if err := s.JobStoreMem.SetWaiting(ctx, jobID, correlationKey, waitType, reason); err != nil {
		return err
	}
	return s.persist()
}

func (s *JobStoreEmbedded) SetParked(ctx context.Context, jobID, correlationKey, waitType, reason string) error {
	if err := s.JobStoreMem.SetParked(ctx, jobID, correlationKey, waitType, reason); err != nil {
		return err
	}
	return s.persist()
}

func (s *JobStoreEmbedded) WakeupJob(ctx context.Context, correlationKey string) (*Job, error) {
	j, err := s.JobStoreMem.WakeupJob(ctx, correlationKey)
	if err != nil || j == nil {
		return j, err
	}
	return j, s.persist()
}

func (s *JobStoreEmbedded) ClaimParkedJob(ctx context.Context, jobID string) (*Job, error) {
	j, err := s.JobStoreMem.ClaimParkedJob(ctx, jobID)
	if err != nil || j == nil {
		return j, err
	}
	return j, s.persist()
}
