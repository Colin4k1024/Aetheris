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

//go:build distributed
// +build distributed

package runtime

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

// SchedulerDistributed 使用 Redis 分布式锁的调度器
type SchedulerDistributed struct {
	manager   *Manager
	run       RunFunc
	redis     *redis.Client
	lockTTL   time.Duration
	keyPrefix string
}

// NewSchedulerDistributed 创建分布式调度器（需要 Redis）
func NewSchedulerDistributed(manager *Manager, run RunFunc, redisClient *redis.Client, opts ...SchedulerDistributedOption) *SchedulerDistributed {
	s := &SchedulerDistributed{
		manager:   manager,
		run:       run,
		redis:     redisClient,
		lockTTL:   30 * time.Second,
		keyPrefix: "aetheris:scheduler:",
	}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

// SchedulerDistributedOption 分布式调度器选项
type SchedulerDistributedOption func(*SchedulerDistributed)

// WithLockTTL 设置锁超时时间
func WithLockTTL(ttl time.Duration) SchedulerDistributedOption {
	return func(s *SchedulerDistributed) {
		s.lockTTL = ttl
	}
}

// WithKeyPrefix 设置 Redis 键前缀
func WithKeyPrefix(prefix string) SchedulerDistributedOption {
	return func(s *SchedulerDistributed) {
		s.keyPrefix = prefix
	}
}

// acquireLock 尝试获取分布式锁
func (s *SchedulerDistributed) acquireLock(ctx context.Context, agentID string) (bool, string, error) {
	lockKey := s.keyPrefix + "lock:" + agentID
	lockValue := uuid.New().String()
	result, err := s.redis.SetNX(ctx, lockKey, lockValue, s.lockTTL).Result()
	if err != nil {
		return false, "", err
	}
	return result, lockValue, nil
}

// releaseLock 释放分布式锁（只释放自己持有的锁）
func (s *SchedulerDistributed) releaseLock(ctx context.Context, agentID, lockValue string) error {
	lockKey := s.keyPrefix + "lock:" + agentID
	// 使用 Lua 脚本确保原子性：只有值匹配时才删除
	script := redis.NewScript(`
		if redis.call("get", KEYS[1]) == ARGV[1] then
			return redis.call("del", KEYS[1])
		else
			return 0
		end
	`)
	_, err := script.Run(ctx, s.redis, []string{lockKey}, lockValue).Result()
	return err
}

// WakeAgent 唤醒 Agent（分布式锁保护）
func (s *SchedulerDistributed) WakeAgent(ctx context.Context, agentID string) error {
	acquired, lockValue, err := s.acquireLock(ctx, agentID)
	if err != nil {
		return err
	}
	if !acquired {
		// 未能获取锁，说明其他 Worker 正在处理
		return nil
	}
	defer s.releaseLock(ctx, agentID, lockValue)

	agent, err := s.manager.Get(ctx, agentID)
	if err != nil || agent == nil {
		return nil
	}
	status := agent.GetStatus()
	if status == StatusRunning || status == StatusWaitingTool {
		return nil
	}
	agent.SetStatus(StatusRunning)
	if s.run != nil {
		s.run(ctx, agentID)
	}
	return nil
}

// Suspend 挂起 Agent
func (s *SchedulerDistributed) Suspend(ctx context.Context, agentID string) error {
	agent, err := s.manager.Get(ctx, agentID)
	if err != nil || agent == nil {
		return nil
	}
	agent.SetStatus(StatusSuspended)
	return nil
}

// Resume 恢复 Agent（分布式锁保护）
func (s *SchedulerDistributed) Resume(ctx context.Context, agentID string) error {
	acquired, lockValue, err := s.acquireLock(ctx, agentID)
	if err != nil {
		return err
	}
	if !acquired {
		return nil
	}
	defer s.releaseLock(ctx, agentID, lockValue)

	agent, err := s.manager.Get(ctx, agentID)
	if err != nil || agent == nil {
		return nil
	}
	agent.SetStatus(StatusIdle)
	agent.SetStatus(StatusRunning)
	if s.run != nil {
		s.run(ctx, agentID)
	}
	return nil
}

// Stop 停止 Agent
func (s *SchedulerDistributed) Stop(ctx context.Context, agentID string) error {
	agent, err := s.manager.Get(ctx, agentID)
	if err != nil || agent == nil {
		return nil
	}
	agent.SetStatus(StatusIdle)
	return nil
}
