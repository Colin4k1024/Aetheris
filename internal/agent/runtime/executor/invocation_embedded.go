package executor

import (
	"context"
	"sync"

	"github.com/Colin4k1024/Aetheris/v2/internal/storage/embedded"
)

type toolInvocationStoreEmbedded struct {
	*ToolInvocationStoreMem
	path      string
	persistMu sync.Mutex
}

// NewToolInvocationStoreEmbedded creates a local durable tool invocation store.
func NewToolInvocationStoreEmbedded(path string) (ToolInvocationStore, error) {
	s := &toolInvocationStoreEmbedded{
		ToolInvocationStoreMem: NewToolInvocationStoreMem(),
		path:                   path,
	}
	if err := s.load(); err != nil {
		return nil, err
	}
	return s, nil
}

func (s *toolInvocationStoreEmbedded) load() error {
	s.persistMu.Lock()
	defer s.persistMu.Unlock()
	var data map[string]*ToolInvocationRecord
	if err := embedded.LoadJSON(s.path, &data); err != nil {
		return err
	}
	if data != nil {
		s.mu.Lock()
		s.byKey = data
		s.mu.Unlock()
	}
	return nil
}

func (s *toolInvocationStoreEmbedded) persist() error {
	s.persistMu.Lock()
	defer s.persistMu.Unlock()
	s.mu.RLock()
	data := s.byKey
	s.mu.RUnlock()
	return embedded.SaveJSON(s.path, data)
}

func (s *toolInvocationStoreEmbedded) SetStarted(ctx context.Context, r *ToolInvocationRecord) error {
	if err := s.ToolInvocationStoreMem.SetStarted(ctx, r); err != nil {
		return err
	}
	return s.persist()
}

func (s *toolInvocationStoreEmbedded) SetFinished(ctx context.Context, idempotencyKey string, status string, result []byte, committed bool, externalID string) error {
	if err := s.ToolInvocationStoreMem.SetFinished(ctx, idempotencyKey, status, result, committed, externalID); err != nil {
		return err
	}
	return s.persist()
}
