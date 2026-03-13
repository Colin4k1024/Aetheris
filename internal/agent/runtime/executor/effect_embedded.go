package executor

import (
	"context"
	"sync"

	"rag-platform/internal/storage/embedded"
)

type effectStoreEmbedded struct {
	*EffectStoreMem
	path      string
	persistMu sync.Mutex
}

type persistedEffects struct {
	ByKey map[string]*EffectRecord   `json:"by_key"`
	ByJob map[string][]*EffectRecord `json:"by_job"`
}

// NewEffectStoreEmbedded creates a local durable effect store.
func NewEffectStoreEmbedded(path string) (EffectStore, error) {
	s := &effectStoreEmbedded{
		EffectStoreMem: NewEffectStoreMem(),
		path:           path,
	}
	if err := s.load(); err != nil {
		return nil, err
	}
	return s, nil
}

func (s *effectStoreEmbedded) load() error {
	s.persistMu.Lock()
	defer s.persistMu.Unlock()
	var data persistedEffects
	if err := embedded.LoadJSON(s.path, &data); err != nil {
		return err
	}
	s.mu.Lock()
	if data.ByKey != nil {
		s.byKey = data.ByKey
	}
	if data.ByJob != nil {
		s.byJob = data.ByJob
	}
	s.mu.Unlock()
	return nil
}

func (s *effectStoreEmbedded) persist() error {
	s.persistMu.Lock()
	defer s.persistMu.Unlock()
	s.mu.RLock()
	data := persistedEffects{
		ByKey: s.byKey,
		ByJob: s.byJob,
	}
	s.mu.RUnlock()
	return embedded.SaveJSON(s.path, data)
}

func (s *effectStoreEmbedded) PutEffect(ctx context.Context, r *EffectRecord) error {
	if err := s.EffectStoreMem.PutEffect(ctx, r); err != nil {
		return err
	}
	return s.persist()
}
