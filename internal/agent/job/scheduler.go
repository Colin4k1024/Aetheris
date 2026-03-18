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
	"errors"
	"log/slog"
	"sync"
	"time"

	agentexec "rag-platform/internal/agent/runtime/executor"
	"rag-platform/pkg/metrics"
)

// RunJobFunc 执行单条 Job 的回调（由应用层注入，如 Runner.RunForJob）
type RunJobFunc func(ctx context.Context, j *Job) error

// CompensateFunc 在 CompensatableFailure 时调用（jobID、failed节点 nodeID）；Week 1 可为 stub，Phase B 接真实回滚
type CompensateFunc func(ctx context.Context, jobID, nodeID string) error

// SchedulerConfig 调度器配置：并发上限、重试、backoff、队列优先级与能力派发
type SchedulerConfig struct {
	MaxConcurrency int           // 最大并发执行数，<=0 表示 1
	RetryMax       int           // 最大重试次数（不含首次）
	Backoff        time.Duration // 重试前等待时间
	// Queues 按优先级轮询的队列列表（如 realtime, default, background）；空则使用 ClaimNextPending 不区分队列
	Queues []string
	// Capabilities 调度器（Worker）能力列表；非空时仅认领 Job.RequiredCapabilities 满足的 Job
	Capabilities []string
	// WakeupQueueTimeout WakeupQueue Receive 超时时间，默认 1 秒
	WakeupQueueTimeout time.Duration
}

// Scheduler 在 JobStore 之上提供排队、并发限制与重试；形态为 API→Job Queue→Scheduler→Worker→Executor
type Scheduler struct {
	store      JobStore
	runJob     RunJobFunc
	config     SchedulerConfig
	compensate CompensateFunc // optional; called on CompensatableFailure before marking job failed
	logger     *slog.Logger
	stopCh     chan struct{}
	wg         sync.WaitGroup
	limiter    chan struct{} // 信号量，限制并发
	wakeup     WakeupQueue   // 事件驱动唤醒队列
}

// NewScheduler 创建调度器；config 为并发与重试策略
func NewScheduler(store JobStore, runJob RunJobFunc, config SchedulerConfig) *Scheduler {
	max := config.MaxConcurrency
	if max <= 0 {
		max = 1
	}
	return &Scheduler{
		store:   store,
		runJob:  runJob,
		config:  config,
		stopCh:  make(chan struct{}),
		limiter: make(chan struct{}, max),
		wakeup:  nil,
	}
}

// SetWakeupQueue 设置唤醒队列（可选，用于事件驱动唤醒 Parked/Waiting 的 Job）
func (s *Scheduler) SetWakeupQueue(wakeup WakeupQueue) {
	s.wakeup = wakeup
}

// SetCompensate 设置 CompensatableFailure 时的补偿回调（可选）
func (s *Scheduler) SetCompensate(fn CompensateFunc) {
	s.compensate = fn
}

// SetLogger 设置日志记录器（可选）
func (s *Scheduler) SetLogger(logger *slog.Logger) {
	s.logger = logger
}

// Start 启动调度循环：最多 MaxConcurrency 个 worker 拉取 Pending、执行、成功则 UpdateStatus(Completed)，failed则按 RetryMax/Backoff 重试或 UpdateStatus(Failed)
// 若配置了 WakeupQueue，则优先从 WakeupQueue 接收唤醒事件，实现事件驱动唤醒 Parked/Waiting Job
func (s *Scheduler) Start(ctx context.Context) {
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		for {
			select {
			case <-s.stopCh:
				return
			case <-ctx.Done():
				return
			case s.limiter <- struct{}{}:
				tickStart := time.Now()
				var j *Job
				var err error

				// 优先尝试从 WakeupQueue 获取唤醒的 Job（事件驱动）
				if s.wakeup != nil {
					timeout := s.config.WakeupQueueTimeout
					if timeout <= 0 {
						timeout = time.Second
					}
					jobID, ok := s.wakeup.Receive(ctx, timeout)
					if ok && jobID != "" {
						// 收到唤醒事件，认领该 Parked/Waiting Job
						j, err = s.store.ClaimParkedJob(ctx, jobID)
						if err != nil && s.logger != nil {
							s.logger.Error("failed to claim parked job from wakeup", "job_id", jobID, "error", err)
						}
					}
				}

				// 如果没有从 WakeupQueue 获取到 Job，则从普通队列拉取
				// 注意：Parked 状态的 Job 不会出现在 pending 队列中，只能通过 WakeupQueue 唤醒
				if j == nil {
					if len(s.config.Queues) > 0 {
						for _, q := range s.config.Queues {
							if len(s.config.Capabilities) > 0 {
								j, err = s.store.ClaimNextPendingForWorker(ctx, q, s.config.Capabilities, "")
							} else {
								j, err = s.store.ClaimNextPendingFromQueue(ctx, q)
							}
							if err != nil && s.logger != nil {
								s.logger.Error("failed to claim job from queue", "queue", q, "error", err)
							}
							if j != nil {
								break
							}
						}
					} else {
						if len(s.config.Capabilities) > 0 {
							j, err = s.store.ClaimNextPendingForWorker(ctx, "", s.config.Capabilities, "")
						} else {
							j, err = s.store.ClaimNextPending(ctx)
						}
						if err != nil && s.logger != nil {
							s.logger.Error("failed to claim job", "error", err)
						}
					}
				}

				metrics.SchedulerTickDurationSeconds.Observe(time.Since(tickStart).Seconds())
				if j == nil {
					metrics.LeaseAcquireTotal.WithLabelValues("default", "false").Inc()
					<-s.limiter
					time.Sleep(200 * time.Millisecond)
					continue
				}
				tenant := j.TenantID
				if tenant == "" {
					tenant = "default"
				}
				metrics.LeaseAcquireTotal.WithLabelValues(tenant, "true").Inc()
				go func(job *Job) {
					defer func() { <-s.limiter }()
					// 使用 detached context，避免外层 ctx 取消影响已认领的 job
					runCtx := context.WithoutCancel(ctx)
					err := s.runJob(runCtx, job)
					if err != nil {
						var sf *agentexec.StepFailure
						if errors.As(err, &sf) {
							switch sf.Type {
							case agentexec.StepResultRetryableFailure:
								if job.RetryCount < s.config.RetryMax {
									time.Sleep(s.config.Backoff)
									if err := s.store.Requeue(runCtx, job); err != nil && s.logger != nil {
										s.logger.Error("failed to requeue job", "job_id", job.ID, "error", err)
									}
								} else {
									if err := s.store.UpdateStatus(runCtx, job.ID, StatusFailed); err != nil && s.logger != nil {
										s.logger.Error("failed to update job status to failed", "job_id", job.ID, "error", err)
									}
								}
							case agentexec.StepResultPermanentFailure:
								if err := s.store.UpdateStatus(runCtx, job.ID, StatusFailed); err != nil && s.logger != nil {
									s.logger.Error("failed to update job status to failed", "job_id", job.ID, "error", err)
								}
							case agentexec.StepResultCompensatableFailure:
								if s.compensate != nil {
									if err := s.compensate(runCtx, job.ID, sf.FailedNodeID()); err != nil && s.logger != nil {
										s.logger.Error("failed to compensate job", "job_id", job.ID, "error", err)
									}
								}
								if err := s.store.UpdateStatus(runCtx, job.ID, StatusFailed); err != nil && s.logger != nil {
									s.logger.Error("failed to update job status to failed", "job_id", job.ID, "error", err)
								}
							case agentexec.StepResultSideEffectCommitted, agentexec.StepResultCompensated:
								// 不应以错误返回；若出现则不再重试，直接failed
								if err := s.store.UpdateStatus(runCtx, job.ID, StatusFailed); err != nil && s.logger != nil {
									s.logger.Error("failed to update job status to failed", "job_id", job.ID, "error", err)
								}
							default:
								if err := s.store.UpdateStatus(runCtx, job.ID, StatusFailed); err != nil && s.logger != nil {
									s.logger.Error("failed to update job status to failed", "job_id", job.ID, "error", err)
								}
							}
						} else {
							// No step outcome: backward compat, retry up to RetryMax
							if job.RetryCount < s.config.RetryMax {
								time.Sleep(s.config.Backoff)
								if err := s.store.Requeue(runCtx, job); err != nil && s.logger != nil {
									s.logger.Error("failed to requeue job", "job_id", job.ID, "error", err)
								}
							} else {
								if err := s.store.UpdateStatus(runCtx, job.ID, StatusFailed); err != nil && s.logger != nil {
									s.logger.Error("failed to update job status to failed", "job_id", job.ID, "error", err)
								}
							}
						}
					} else {
						if err := s.store.UpdateStatus(runCtx, job.ID, StatusCompleted); err != nil && s.logger != nil {
							s.logger.Error("failed to update job status to completed", "job_id", job.ID, "error", err)
						}
					}
				}(j)
			}
		}
	}()
}

// Stop 优雅退出：关闭 stopCh，等待当前循环结束（不等待已在执行的 job 完成）
func (s *Scheduler) Stop() {
	close(s.stopCh)
	s.wg.Wait()
}
