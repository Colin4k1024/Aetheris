package runtime

import (
	"context"
	"sync"

	"github.com/Colin4k1024/Aetheris/v2/internal/storage/embedded"
)

type agentStateStoreEmbedded struct {
	*agentStateStoreMem
	path      string
	persistMu sync.Mutex
}

// NewAgentStateStoreEmbedded creates a local durable agent state store.
func NewAgentStateStoreEmbedded(path string) (AgentStateStore, error) {
	s := &agentStateStoreEmbedded{
		agentStateStoreMem: NewAgentStateStoreMem().(*agentStateStoreMem),
		path:               path,
	}
	if err := s.load(); err != nil {
		return nil, err
	}
	return s, nil
}

func (s *agentStateStoreEmbedded) load() error {
	s.persistMu.Lock()
	defer s.persistMu.Unlock()
	var data map[string]*AgentState
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

func (s *agentStateStoreEmbedded) persist() error {
	s.persistMu.Lock()
	defer s.persistMu.Unlock()
	s.mu.RLock()
	data := s.byKey
	s.mu.RUnlock()
	return embedded.SaveJSON(s.path, data)
}

func (s *agentStateStoreEmbedded) SaveAgentState(ctx context.Context, agentID, sessionID string, state *AgentState) error {
	if err := s.agentStateStoreMem.SaveAgentState(ctx, agentID, sessionID, state); err != nil {
		return err
	}
	return s.persist()
}
