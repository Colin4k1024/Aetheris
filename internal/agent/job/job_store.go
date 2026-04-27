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

package job

import (
	"context"
	"sync"
	"time"

	"github.com/google/uuid"
)

// JobStore 任务存储：创建、查询、更新状态、拉取 Pending、更新恢复游标；多租户时 tenantID 过滤
type JobStore interface {
	Create(ctx context.Context, job *Job) (string, error)
	Get(ctx context.Context, jobID string) (*Job, error)
	// GetByAgentAndIdempotencyKey 按 Agent 与幂等键查已有 Job，用于旧调用方兼容；无则返回 nil, nil
	GetByAgentAndIdempotencyKey(ctx context.Context, agentID, idempotencyKey string) (*Job, error)
	// GetByAgentTenantAndIdempotencyKey 按 Agent、Tenant 与幂等键查已有 Job，用于多租户 Idempotency-Key 去重
	GetByAgentTenantAndIdempotencyKey(ctx context.Context, agentID, tenantID, idempotencyKey string) (*Job, error)
	// ListByAgent 按 Agent 列出 Job；tenantID 非空时仅返回该租户下的 Job
	ListByAgent(ctx context.Context, agentID string, tenantID string) ([]*Job, error)
	UpdateStatus(ctx context.Context, jobID string, status JobStatus) error
	// UpdateCursor 更新 Job 的恢复游标（Checkpoint ID），用于恢复时从 LastCheckpoint 继续
	UpdateCursor(ctx context.Context, jobID string, cursor string) error
	// ClaimNextPending 原子取出一条 Pending 并置为 Running，无则返回 nil, nil；tenantID 非空时仅认领该租户的 Job
	ClaimNextPending(ctx context.Context) (*Job, error)
	// ClaimNextPendingFromQueue 从指定队列取出一条 Pending（同队列内按 Priority 降序）；queueClass 为空时等价 ClaimNextPending
	ClaimNextPendingFromQueue(ctx context.Context, queueClass string) (*Job, error)
	// ClaimNextPendingForWorker 从指定队列取出一条 Pending 且该 Job 的 RequiredCapabilities 被 workerCapabilities 覆盖；tenantID 非空时仅认领该租户
	ClaimNextPendingForWorker(ctx context.Context, queueClass string, workerCapabilities []string, tenantID string) (*Job, error)
	// Requeue 将 Job 重新入队为 Pending（用于重试；会递增 RetryCount）
	Requeue(ctx context.Context, job *Job) error
	// RequestCancel 请求取消执行中的 Job；Worker 轮询 Get 时发现 CancelRequestedAt 非零则取消 runCtx
	RequestCancel(ctx context.Context, jobID string) error
	// ReclaimOrphanedJobs 将 status=Running 且 updated_at 早于 (now - olderThan) 的 Job 置回 Pending，供其他 Worker 认领；返回回收数量（design/job-state-machine.md）
	ReclaimOrphanedJobs(ctx context.Context, olderThan time.Duration) (int, error)

	// --- Parked/Waiting 状态管理 ---

	// SetWaiting 将 Job 设置为 Waiting 状态（短暂等待 <1min）
	// correlationKey 用于后续 signal 匹配唤醒
	SetWaiting(ctx context.Context, jobID, correlationKey, waitType, reason string) error
	// SetParked 将 Job 设置为 Parked 状态（长时间等待 >1min）
	// Parked 的 Job 不参与正常调度轮询，仅通过 WakeupQueue 唤醒
	SetParked(ctx context.Context, jobID, correlationKey, waitType, reason string) error
	// WakeupJob 通过 correlationKey 唤醒 Parked/Waiting 的 Job，转换回 Pending 状态
	// 返回被唤醒的 Job，若无匹配则返回 nil
	WakeupJob(ctx context.Context, correlationKey string) (*Job, error)
	// ClaimParkedJob 认领一个 Parked 的 Job（通过 WakeupQueue 收到 jobID 后调用）
	ClaimParkedJob(ctx context.Context, jobID string) (*Job, error)
}

// jobMatchesCapabilities 判断 Job 的 RequiredCapabilities 是否被 workerCapabilities 覆盖；jobRequired 为空表示任意 Worker 可执行；workerCapabilities 为空表示不按能力过滤
func jobMatchesCapabilities(jobRequired, workerCapabilities []string) bool {
	if len(jobRequired) == 0 {
		return true
	}
	if len(workerCapabilities) == 0 {
		return true
	}
	set := make(map[string]struct{}, len(workerCapabilities))
	for _, c := range workerCapabilities {
		set[c] = struct{}{}
	}
	for _, r := range jobRequired {
		if _, ok := set[r]; !ok {
			return false
		}
	}
	return true
}

// waitingInfo 等待信息，用于 Parked/Waiting 状态
type waitingInfo struct {
	jobID          string
	correlationKey string
	waitType       string // webhook, human, timer, signal, message
	reason         string
	status         JobStatus // StatusWaiting 或 StatusParked
	setAt          time.Time
}

// JobStoreMem 内存实现：map + Pending 队列，Create 时入队，ClaimNextPending 取队首并置 Running
type JobStoreMem struct {
	mu      sync.Mutex
	byID    map[string]*Job
	pending []string
	cond    *sync.Cond

	// waiting 和 parked 状态的 Job 跟踪
	waiting      map[string]*waitingInfo // jobID -> waitingInfo
	waitingByKey map[string]*waitingInfo // correlationKey -> waitingInfo
}

// NewJobStoreMem 创建内存 JobStore
func NewJobStoreMem() *JobStoreMem {
	j := &JobStoreMem{
		byID:         make(map[string]*Job),
		pending:      nil,
		waiting:      make(map[string]*waitingInfo),
		waitingByKey: make(map[string]*waitingInfo),
	}
	j.cond = sync.NewCond(&j.mu)
	return j
}

func (s *JobStoreMem) Create(ctx context.Context, job *Job) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if job.ID == "" {
		job.ID = "job-" + uuid.New().String()
	}
	if job.TenantID == "" {
		job.TenantID = "default"
	}
	job.Status = StatusPending
	job.CreatedAt = time.Now()
	job.UpdatedAt = job.CreatedAt
	cp := *job
	s.byID[job.ID] = &cp
	s.pending = append(s.pending, job.ID)
	s.cond.Signal()
	return job.ID, nil
}

func (s *JobStoreMem) Get(ctx context.Context, jobID string) (*Job, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	j, ok := s.byID[jobID]
	if !ok {
		return nil, nil
	}
	cp := *j
	return &cp, nil
}

func (s *JobStoreMem) GetByAgentAndIdempotencyKey(ctx context.Context, agentID, idempotencyKey string) (*Job, error) {
	if idempotencyKey == "" {
		return nil, nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, j := range s.byID {
		if j.AgentID == agentID && j.IdempotencyKey == idempotencyKey {
			cp := *j
			return &cp, nil
		}
	}
	return nil, nil
}

func (s *JobStoreMem) GetByAgentTenantAndIdempotencyKey(ctx context.Context, agentID, tenantID, idempotencyKey string) (*Job, error) {
	if idempotencyKey == "" {
		return nil, nil
	}
	if tenantID == "" {
		tenantID = "default"
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, j := range s.byID {
		if j.AgentID == agentID && j.TenantID == tenantID && j.IdempotencyKey == idempotencyKey {
			cp := *j
			return &cp, nil
		}
	}
	return nil, nil
}

func (s *JobStoreMem) ListByAgent(ctx context.Context, agentID string, tenantID string) ([]*Job, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	var list []*Job
	for _, j := range s.byID {
		if j.AgentID != agentID {
			continue
		}
		if tenantID != "" && j.TenantID != tenantID {
			continue
		}
		cp := *j
		list = append(list, &cp)
	}
	return list, nil
}

func (s *JobStoreMem) UpdateStatus(ctx context.Context, jobID string, status JobStatus) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	j, ok := s.byID[jobID]
	if !ok {
		return nil
	}
	j.Status = status
	j.UpdatedAt = time.Now()
	return nil
}

func (s *JobStoreMem) UpdateCursor(ctx context.Context, jobID string, cursor string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	j, ok := s.byID[jobID]
	if !ok {
		return nil
	}
	j.Cursor = cursor
	j.UpdatedAt = time.Now()
	return nil
}

func (s *JobStoreMem) ClaimNextPending(ctx context.Context) (*Job, error) {
	return s.ClaimNextPendingFromQueue(ctx, "")
}

func (s *JobStoreMem) ClaimNextPendingFromQueue(ctx context.Context, queueClass string) (*Job, error) {
	return s.ClaimNextPendingForWorker(ctx, queueClass, nil, "")
}

func (s *JobStoreMem) ClaimNextPendingForWorker(ctx context.Context, queueClass string, workerCapabilities []string, tenantID string) (*Job, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	var bestID string
	var bestPriority int
	var bestIdx int = -1
	for idx, id := range s.pending {
		j, ok := s.byID[id]
		if !ok || j.Status != StatusPending {
			continue
		}
		if tenantID != "" && j.TenantID != tenantID {
			continue
		}
		if queueClass != "" && j.QueueClass != "" && j.QueueClass != queueClass {
			continue
		}
		if !jobMatchesCapabilities(j.RequiredCapabilities, workerCapabilities) {
			continue
		}
		if bestIdx < 0 || j.Priority > bestPriority {
			bestID = id
			bestPriority = j.Priority
			bestIdx = idx
		}
	}
	if bestIdx < 0 {
		return nil, nil
	}
	// 从 pending 中移除 bestID（保持顺序）
	s.pending = append(s.pending[:bestIdx], s.pending[bestIdx+1:]...)
	j := s.byID[bestID]
	j.Status = StatusRunning
	j.UpdatedAt = time.Now()
	cp := *j
	return &cp, nil
}

func (s *JobStoreMem) Requeue(ctx context.Context, job *Job) error {
	if job == nil {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	j, ok := s.byID[job.ID]
	if !ok {
		return nil
	}
	j.RetryCount = job.RetryCount + 1
	j.Status = StatusPending
	j.UpdatedAt = time.Now()
	s.pending = append(s.pending, job.ID)
	s.cond.Signal()
	return nil
}

func (s *JobStoreMem) RequestCancel(ctx context.Context, jobID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	j, ok := s.byID[jobID]
	if !ok {
		return nil
	}
	j.CancelRequestedAt = time.Now()
	j.UpdatedAt = j.CancelRequestedAt
	return nil
}

// ReclaimOrphanedJobs 内存实现：单进程无租约过期语义，返回 0
func (s *JobStoreMem) ReclaimOrphanedJobs(ctx context.Context, olderThan time.Duration) (int, error) {
	_ = olderThan
	return 0, nil
}

// WaitNextPending 阻塞直到有 Pending 或 ctx 取消，然后尝试 Claim；无则返回 nil, nil
func (s *JobStoreMem) WaitNextPending(ctx context.Context) (*Job, error) {
	done := ctx.Done()
	for {
		s.mu.Lock()
		for len(s.pending) == 0 {
			s.cond.Wait()
			select {
			case <-done:
				s.mu.Unlock()
				return nil, ctx.Err()
			default:
			}
		}
		id := s.pending[0]
		s.pending = s.pending[1:]
		j, ok := s.byID[id]
		if !ok {
			s.mu.Unlock()
			continue
		}
		if j.Status != StatusPending {
			s.mu.Unlock()
			continue
		}
		j.Status = StatusRunning
		j.UpdatedAt = time.Now()
		cp := *j
		s.mu.Unlock()
		return &cp, nil
	}
}

// SetWaiting 将 Job 设置为 Waiting 状态（短暂等待 <1min）
func (s *JobStoreMem) SetWaiting(ctx context.Context, jobID, correlationKey, waitType, reason string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	j, ok := s.byID[jobID]
	if !ok {
		return nil
	}
	j.Status = StatusWaiting
	j.UpdatedAt = time.Now()

	info := &waitingInfo{
		jobID:          jobID,
		correlationKey: correlationKey,
		waitType:       waitType,
		reason:         reason,
		status:         StatusWaiting,
		setAt:          time.Now(),
	}
	s.waiting[jobID] = info
	if correlationKey != "" {
		s.waitingByKey[correlationKey] = info
	}
	return nil
}

// SetParked 将 Job 设置为 Parked 状态（长时间等待 >1min）
// Parked 的 Job 不参与正常调度轮询，仅通过 WakeupQueue 唤醒
func (s *JobStoreMem) SetParked(ctx context.Context, jobID, correlationKey, waitType, reason string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	j, ok := s.byID[jobID]
	if !ok {
		return nil
	}
	j.Status = StatusParked
	j.UpdatedAt = time.Now()

	info := &waitingInfo{
		jobID:          jobID,
		correlationKey: correlationKey,
		waitType:       waitType,
		reason:         reason,
		status:         StatusParked,
		setAt:          time.Now(),
	}
	s.waiting[jobID] = info
	if correlationKey != "" {
		s.waitingByKey[correlationKey] = info
	}
	return nil
}

// WakeupJob 通过 correlationKey 唤醒 Parked/Waiting 的 Job，转换回 Pending 状态
func (s *JobStoreMem) WakeupJob(ctx context.Context, correlationKey string) (*Job, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	info, ok := s.waitingByKey[correlationKey]
	if !ok {
		return nil, nil
	}

	j, ok := s.byID[info.jobID]
	if !ok {
		delete(s.waitingByKey, correlationKey)
		delete(s.waiting, info.jobID)
		return nil, nil
	}

	// 从 waiting 跟踪中移除
	delete(s.waitingByKey, correlationKey)
	delete(s.waiting, info.jobID)

	// 转换回 Pending 状态
	j.Status = StatusPending
	j.UpdatedAt = time.Now()

	// 加入 pending 队列
	s.pending = append(s.pending, j.ID)
	s.cond.Signal()

	cp := *j
	return &cp, nil
}

// ClaimParkedJob 认领一个 Parked 的 Job（通过 WakeupQueue 收到 jobID 后调用）
func (s *JobStoreMem) ClaimParkedJob(ctx context.Context, jobID string) (*Job, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	j, ok := s.byID[jobID]
	if !ok {
		return nil, nil
	}

	// 只能认领 Parked 或 Waiting 状态的 Job
	if j.Status != StatusParked && j.Status != StatusWaiting {
		return nil, nil
	}

	// 从 waiting 跟踪中移除
	if info, ok := s.waiting[jobID]; ok {
		delete(s.waiting, jobID)
		if info.correlationKey != "" {
			delete(s.waitingByKey, info.correlationKey)
		}
	}

	// 设置为 Running 状态
	j.Status = StatusRunning
	j.UpdatedAt = time.Now()

	cp := *j
	return &cp, nil
}
