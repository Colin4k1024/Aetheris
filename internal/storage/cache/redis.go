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

package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"rag-platform/pkg/metrics"
)

// RedisStore Redis 缓存存储实现
type RedisStore struct {
	client   *redis.Client
	prefix   string
	tenant   string
	poolName string
}

// NewRedisStore 创建新的 Redis 缓存存储
func NewRedisStore(addr string, password string, db int, prefix string) (*RedisStore, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       db,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to redis: %w", err)
	}

	return &RedisStore{
		client:   client,
		prefix:   prefix,
		poolName: "default",
	}, nil
}

// NewRedisStoreWithMetrics 创建带 metrics 的 Redis 缓存存储
func NewRedisStoreWithMetrics(addr string, password string, db int, prefix string, tenant string) (*RedisStore, error) {
	store, err := NewRedisStore(addr, password, db, prefix)
	if err != nil {
		return nil, err
	}
	store.tenant = tenant
	if prefix != "" {
		store.poolName = prefix
	}
	// 启动连接池 metrics 采集
	go store.startPoolMetricsCollection()
	return store, nil
}

// startPoolMetricsCollection 定期采集连接池 metrics
func (s *RedisStore) startPoolMetricsCollection() {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		s.collectPoolMetrics()
	}
}

// collectPoolMetrics 采集连接池 metrics
func (s *RedisStore) collectPoolMetrics() {
	if s.client == nil {
		return
	}

	// 使用 Stats 获取连接池统计信息
	stats := s.client.PoolStats()
	if stats == nil {
		return
	}

	metrics.SetConnectionPoolMetrics(
		s.tenant,
		"redis",
		s.poolName,
		int(stats.TotalConns),
		int(stats.TotalConns), // MaxTotalConns not available, use TotalConns as approximation
		int(stats.IdleConns),
	)
}

// NewRedisStoreFromConfig 从配置创建 Redis 缓存存储
func NewRedisStoreFromConfig(cfg StoreConfig) (*RedisStore, error) {
	return NewRedisStore(cfg.Addr, cfg.Password, cfg.DB, cfg.Prefix)
}

// StoreConfig Redis 存储配置
type StoreConfig struct {
	Addr     string
	Password string
	DB       int
	Prefix   string
}

// buildKey 构建带前缀的 key
func (s *RedisStore) buildKey(key string) string {
	if s.prefix != "" {
		return s.prefix + ":" + key
	}
	return key
}

// Set 设置缓存
func (s *RedisStore) Set(ctx context.Context, key string, value any, expiration time.Duration) error {
	start := time.Now()
	// 序列化值
	data, err := json.Marshal(value)
	if err != nil {
		metrics.ObserveStorageOperation(s.tenant, "redis", "set", "error", time.Since(start))
		return fmt.Errorf("failed to marshal cache value: %w", err)
	}

	err = s.client.Set(ctx, s.buildKey(key), data, expiration).Err()
	metrics.ObserveStorageOperation(s.tenant, "redis", "set", "success", time.Since(start))
	return err
}

// Get 获取缓存
func (s *RedisStore) Get(ctx context.Context, key string, dest any) error {
	start := time.Now()
	data, err := s.client.Get(ctx, s.buildKey(key)).Bytes()
	if err != nil {
		if err == redis.Nil {
			metrics.ObserveStorageOperation(s.tenant, "redis", "get", "miss", time.Since(start))
			return fmt.Errorf("cache item with key %s not found", key)
		}
		metrics.ObserveStorageOperation(s.tenant, "redis", "get", "error", time.Since(start))
		return fmt.Errorf("failed to get cache: %w", err)
	}
	metrics.ObserveStorageOperation(s.tenant, "redis", "get", "hit", time.Since(start))

	if err := json.Unmarshal(data, dest); err != nil {
		return fmt.Errorf("failed to unmarshal cache value: %w", err)
	}

	return nil
}

// Delete 删除缓存
func (s *RedisStore) Delete(ctx context.Context, key string) error {
	start := time.Now()
	err := s.client.Del(ctx, s.buildKey(key)).Err()
	metrics.ObserveStorageOperation(s.tenant, "redis", "delete", "success", time.Since(start))
	return err
}

// Exists 检查缓存是否存在
func (s *RedisStore) Exists(ctx context.Context, key string) (bool, error) {
	start := time.Now()
	count, err := s.client.Exists(ctx, s.buildKey(key)).Result()
	metrics.ObserveStorageOperation(s.tenant, "redis", "exists", "success", time.Since(start))
	if err != nil {
		return false, fmt.Errorf("failed to check exists: %w", err)
	}
	return count > 0, nil
}

// Clear 清除所有缓存（带前缀的 key）
func (s *RedisStore) Clear(ctx context.Context) error {
	if s.prefix == "" {
		// 如果没有前缀，不允许 clear all（危险操作）
		return fmt.Errorf("cannot clear all keys without prefix")
	}

	iter := s.client.Scan(ctx, 0, s.prefix+":*", 0).Iterator()
	var keys []string
	for iter.Next(ctx) {
		keys = append(keys, iter.Val())
	}
	if err := iter.Err(); err != nil {
		return fmt.Errorf("failed to scan keys: %w", err)
	}

	if len(keys) > 0 {
		return s.client.Del(ctx, keys...).Err()
	}
	return nil
}

// Close 关闭缓存连接
func (s *RedisStore) Close() error {
	return s.client.Close()
}

// MGet 批量获取缓存
func (s *RedisStore) MGet(ctx context.Context, keys []string) ([]any, error) {
	if len(keys) == 0 {
		return nil, nil
	}

	start := time.Now()
	fullKeys := make([]string, len(keys))
	for i, key := range keys {
		fullKeys[i] = s.buildKey(key)
	}

	result, err := s.client.MGet(ctx, fullKeys...).Result()
	metrics.ObserveStorageOperation(s.tenant, "redis", "mget", "success", time.Since(start))
	return result, err
}

// MSet 批量设置缓存
func (s *RedisStore) MSet(ctx context.Context, items map[string]any, expiration time.Duration) error {
	if len(items) == 0 {
		return nil
	}

	start := time.Now()
	pipe := s.client.Pipeline()
	for key, value := range items {
		data, err := json.Marshal(value)
		if err != nil {
			return fmt.Errorf("failed to marshal cache value: %w", err)
		}
		pipe.Set(ctx, s.buildKey(key), data, expiration)
	}

	_, err := pipe.Exec(ctx)
	metrics.ObserveStorageOperation(s.tenant, "redis", "mset", "success", time.Since(start))
	return err
}
