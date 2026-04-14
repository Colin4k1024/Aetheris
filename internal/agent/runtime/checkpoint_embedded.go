package runtime

import (
	"context"
	"sync"

	"github.com/Colin4k1024/Aetheris/v2/internal/storage/embedded"
)

// checkpointStoreEmbedded is a local durable checkpoint store.
type checkpointStoreEmbedded struct {
	*checkpointStoreMem
	path      string
	persistMu sync.Mutex
}

// NewCheckpointStoreEmbedded creates a file-backed checkpoint store.
func NewCheckpointStoreEmbedded(path string) (CheckpointStore, error) {
	s := &checkpointStoreEmbedded{
		checkpointStoreMem: NewCheckpointStoreMem().(*checkpointStoreMem),
		path:               path,
	}
	if err := s.load(); err != nil {
		return nil, err
	}
	return s, nil
}

func (s *checkpointStoreEmbedded) load() error {
	s.persistMu.Lock()
	defer s.persistMu.Unlock()
	var data map[string]*Checkpoint
	if err := embedded.LoadJSON(s.path, &data); err != nil {
		return err
	}
	if data != nil {
		s.mu.Lock()
		s.byID = data
		s.mu.Unlock()
	}
	return nil
}

func (s *checkpointStoreEmbedded) persist() error {
	s.persistMu.Lock()
	defer s.persistMu.Unlock()
	s.mu.RLock()
	data := s.byID
	s.mu.RUnlock()
	return embedded.SaveJSON(s.path, data)
}

func (s *checkpointStoreEmbedded) Save(ctx context.Context, cp *Checkpoint) (string, error) {
	id, err := s.checkpointStoreMem.Save(ctx, cp)
	if err != nil {
		return id, err
	}
	return id, s.persist()
}
