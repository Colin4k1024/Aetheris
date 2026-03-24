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

package approval

import (
	"context"
	"sync"
	"time"
)

// MemStore 内存审批存储（仅用于测试或单实例部署）
type MemStore struct {
	mu    sync.RWMutex
	items map[string]*ApprovalRequest
}

// NewMemStore 创建内存存储
func NewMemStore() *MemStore {
	return &MemStore{items: make(map[string]*ApprovalRequest)}
}

func (s *MemStore) Create(ctx context.Context, req *ApprovalRequest) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	req.CreatedAt = time.Now()
	req.UpdatedAt = time.Now()
	if req.Status == "" {
		req.Status = DecisionPending
	}
	s.items[req.ID] = req
	return nil
}

func (s *MemStore) GetByID(ctx context.Context, id string) (*ApprovalRequest, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	req, ok := s.items[id]
	if !ok {
		return nil, nil
	}
	return req, nil
}

func (s *MemStore) GetByJobID(ctx context.Context, jobID string) ([]*ApprovalRequest, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var results []*ApprovalRequest
	for _, req := range s.items {
		if req.JobID == jobID {
			results = append(results, req)
		}
	}
	return results, nil
}

func (s *MemStore) GetPending(ctx context.Context) ([]*ApprovalRequest, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var results []*ApprovalRequest
	now := time.Now()
	for _, req := range s.items {
		if req.Status == DecisionPending && (req.ExpiresAt == nil || req.ExpiresAt.After(now)) {
			results = append(results, req)
		}
	}
	return results, nil
}

func (s *MemStore) GetPendingByApprover(ctx context.Context, approverID string) ([]*ApprovalRequest, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var results []*ApprovalRequest
	now := time.Now()
	for _, req := range s.items {
		if req.Status != DecisionPending {
			continue
		}
		if req.ExpiresAt != nil && req.ExpiresAt.Before(now) {
			continue
		}
		switch req.ApproverType {
		case ApproverTypeAnyone:
			results = append(results, req)
		case ApproverTypeSpecific:
			if req.ApproverID == approverID {
				results = append(results, req)
			}
		case ApproverTypeRole:
			// Role-based filtering would require a role service; for now return empty
		}
	}
	return results, nil
}

func (s *MemStore) Complete(ctx context.Context, id string, resp *ApprovalResponse) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	req, ok := s.items[id]
	if !ok {
		return nil
	}
	req.Status = resp.Decision
	req.ApproverResp = resp
	req.UpdatedAt = time.Now()
	return nil
}

func (s *MemStore) Expire(ctx context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	req, ok := s.items[id]
	if !ok {
		return nil
	}
	req.Status = DecisionExpired
	req.UpdatedAt = time.Now()
	return nil
}
