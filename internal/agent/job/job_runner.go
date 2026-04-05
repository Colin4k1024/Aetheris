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
	"log/slog"
	"sync"
	"time"

	"rag-platform/internal/agent/runtime"
	agentexec "rag-platform/internal/agent/runtime/executor"
)

// JobRunner 后台拉取 Pending Job 并调用 executor.Runner 执行
type JobRunner struct {
	store   JobStore
	manager *runtime.Manager
	runner  *agentexec.Runner
	logger  *slog.Logger

	stopCh chan struct{}
	wg     sync.WaitGroup
}

// NewJobRunner 创建 JobRunner
func NewJobRunner(store JobStore, manager *runtime.Manager, runner *agentexec.Runner) *JobRunner {
	return &JobRunner{
		store:   store,
		manager: manager,
		runner:  runner,
		stopCh:  make(chan struct{}),
	}
}

// SetLogger 设置日志记录器
func (r *JobRunner) SetLogger(logger *slog.Logger) {
	r.logger = logger
}

// Start 启动后台循环：拉取 Pending Job，执行，更新状态；ctx 用于执行时传递，不用于停止
func (r *JobRunner) Start(ctx context.Context) {
	r.wg.Add(1)
	go func() {
		defer r.wg.Done()
		for {
			select {
			case <-r.stopCh:
				return
			default:
			}
			j, err := r.store.ClaimNextPending(ctx)
			if err != nil && r.logger != nil {
				r.logger.Error("failed to claim job", "error", err)
			}
			if j == nil {
				time.Sleep(200 * time.Millisecond)
				continue
			}
			agent, err := r.manager.Get(ctx, j.AgentID)
			if err != nil && r.logger != nil {
				r.logger.Error("failed to get agent", "agentID", j.AgentID, "error", err)
			}
			if agent == nil {
				if err := r.store.UpdateStatus(ctx, j.ID, StatusFailed); err != nil && r.logger != nil {
					r.logger.Error("failed to update job status", "jobID", j.ID, "error", err)
				}
				continue
			}
			tenantID := j.TenantID
			if tenantID == "" {
				tenantID = "default"
			}
			runErr := r.runner.RunForJob(ctx, agent, &agentexec.JobForRunner{
				ID: j.ID, AgentID: j.AgentID, Goal: j.Goal, Cursor: j.Cursor, TenantID: tenantID,
			})
			if runErr != nil {
				if err := r.store.UpdateStatus(ctx, j.ID, StatusFailed); err != nil && r.logger != nil {
					r.logger.Error("failed to update job status to failed", "jobID", j.ID, "error", err)
				}
			} else {
				if err := r.store.UpdateStatus(ctx, j.ID, StatusCompleted); err != nil && r.logger != nil {
					r.logger.Error("failed to update job status to completed", "jobID", j.ID, "error", err)
				}
			}
		}
	}()
}

// Stop 优雅退出：关闭 stopCh，等待后台 goroutine 结束
func (r *JobRunner) Stop() {
	close(r.stopCh)
	r.wg.Wait()
}
