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
)

// RedisStore Redis 缓存存储实现
type RedisStore struct {
	client *redis.Client
	prefix string
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
		client: client,
		prefix: prefix,
	}, nil
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
	// 序列化值
	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("failed to marshal cache value: %w", err)
	}

	return s.client.Set(ctx, s.buildKey(key), data, expiration).Err()
}

// Get 获取缓存
func (s *RedisStore) Get(ctx context.Context, key string, dest any) error {
	data, err := s.client.Get(ctx, s.buildKey(key)).Bytes()
	if err != nil {
		if err == redis.Nil {
			return fmt.Errorf("cache item with key %s not found", key)
		}
		return fmt.Errorf("failed to get cache: %w", err)
	}

	if err := json.Unmarshal(data, dest); err != nil {
		return fmt.Errorf("failed to unmarshal cache value: %w", err)
	}

	return nil
}

// Delete 删除缓存
func (s *RedisStore) Delete(ctx context.Context, key string) error {
	return s.client.Del(ctx, s.buildKey(key)).Err()
}

// Exists 检查缓存是否存在
func (s *RedisStore) Exists(ctx context.Context, key string) (bool, error) {
	count, err := s.client.Exists(ctx, s.buildKey(key)).Result()
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

	fullKeys := make([]string, len(keys))
	for i, key := range keys {
		fullKeys[i] = s.buildKey(key)
	}

	return s.client.MGet(ctx, fullKeys...).Result()
}

// MSet 批量设置缓存
func (s *RedisStore) MSet(ctx context.Context, items map[string]any, expiration time.Duration) error {
	if len(items) == 0 {
		return nil
	}

	pipe := s.client.Pipeline()
	for key, value := range items {
		data, err := json.Marshal(value)
		if err != nil {
			return fmt.Errorf("failed to marshal cache value: %w", err)
		}
		pipe.Set(ctx, s.buildKey(key), data, expiration)
	}

	_, err := pipe.Exec(ctx)
	return err
}
