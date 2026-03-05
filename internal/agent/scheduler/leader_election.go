// Copyright 2026 Aetheris
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

package scheduler

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// LeaderElection Redis-based leader election implementation
type LeaderElection struct {
	client     *redis.Client
	lockPrefix string
	ttl        time.Duration
}

// LeaderElectionConfig Leader election configuration
type LeaderElectionConfig struct {
	RedisAddr     string
	RedisPassword string
	RedisDB       int
	LockPrefix    string
	TTL           time.Duration
}

// NewLeaderElection 创建新的 leader election
func NewLeaderElection(cfg LeaderElectionConfig) (*LeaderElection, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     cfg.RedisAddr,
		Password: cfg.RedisPassword,
		DB:       cfg.RedisDB,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to redis: %w", err)
	}

	prefix := cfg.LockPrefix
	if prefix == "" {
		prefix = "aetheris:leader"
	}

	ttl := cfg.TTL
	if ttl <= 0 {
		ttl = 30 * time.Second
	}

	return &LeaderElection{
		client:     client,
		lockPrefix: prefix,
		ttl:        ttl,
	}, nil
}

// Elect 尝试获取 leader 身份
// 返回 true 表示获取成功，false 表示已有其他 leader
func (le *LeaderElection) Elect(ctx context.Context, key, identity string) (bool, error) {
	lockKey := le.lockPrefix + ":" + key

	// 使用 SET NX (set if not exists) 来获取锁
	success, err := le.client.SetNX(ctx, lockKey, identity, le.ttl).Result()
	if err != nil {
		return false, fmt.Errorf("failed to acquire lock: %w", err)
	}

	return success, nil
}

// IsLeader 检查当前 identity 是否为 leader
func (le *LeaderElection) IsLeader(ctx context.Context, key, identity string) (bool, error) {
	lockKey := le.lockPrefix + ":" + key

	currentLeader, err := le.client.Get(ctx, lockKey).Result()
	if err == redis.Nil {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("failed to get leader: %w", err)
	}

	return currentLeader == identity, nil
}

// Renew 续约 leader 身份
func (le *LeaderElection) Renew(ctx context.Context, key, identity string) (bool, error) {
	lockKey := le.lockPrefix + ":" + key

	// 只有当前 leader 才能续约
	script := redis.NewScript(`
		if redis.call("GET", KEYS[1]) == ARGV[1] then
			return redis.call("PEXPIRE", KEYS[1], ARGV[2])
		else
			return 0
		end
	`)

	result, err := script.Run(ctx, le.client, []string{lockKey}, identity, le.ttl.Milliseconds()).Int()
	if err != nil {
		return false, fmt.Errorf("failed to renew lock: %w", err)
	}

	return result == 1, nil
}

// Resign 放弃 leader 身份
func (le *LeaderElection) Resign(ctx context.Context, key, identity string) (bool, error) {
	lockKey := le.lockPrefix + ":" + key

	// 只有当前 leader 才能释放锁
	script := redis.NewScript(`
		if redis.call("GET", KEYS[1]) == ARGV[1] then
			return redis.call("DEL", KEYS[1])
		else
			return 0
		end
	`)

	result, err := script.Run(ctx, le.client, []string{lockKey}, identity).Int()
	if err != nil {
		return false, fmt.Errorf("failed to resign: %w", err)
	}

	return result == 1, nil
}

// Close 关闭连接
func (le *LeaderElection) Close() error {
	return le.client.Close()
}

// DistributedLock Redis-based distributed lock
type DistributedLock struct {
	client   *redis.Client
	lockName string
}

// NewDistributedLock 创建新的分布式锁
func NewDistributedLock(client *redis.Client, lockName string) *DistributedLock {
	return &DistributedLock{
		client:   client,
		lockName: lockName,
	}
}

// Lock 尝试获取锁
func (dl *DistributedLock) Lock(ctx context.Context, identity string, ttl time.Duration) (bool, error) {
	return dl.client.SetNX(ctx, dl.lockName, identity, ttl).Result()
}

// Unlock 释放锁
func (dl *DistributedLock) Unlock(ctx context.Context, identity string) (bool, error) {
	script := redis.NewScript(`
		if redis.call("GET", KEYS[1]) == ARGV[1] then
			return redis.call("DEL", KEYS[1])
		else
			return 0
		end
	`)

	result, err := script.Run(ctx, dl.client, []string{dl.lockName}, identity).Int()
	if err != nil {
		return false, fmt.Errorf("failed to unlock: %w", err)
	}

	return result == 1, nil
}
