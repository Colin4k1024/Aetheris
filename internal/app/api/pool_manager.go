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

package api

import (
	"context"
	"fmt"
	"sync"

	"github.com/jackc/pgx/v5/pgxpool"
)

// PoolManager PostgreSQL 连接池管理器
// 优化：共享相同 DSN 的连接池，减少数据库连接数
type PoolManager struct {
	pools map[string]*pgxpool.Pool
	mu    sync.RWMutex
}

// NewPoolManager 创建新的连接池管理器
func NewPoolManager() *PoolManager {
	return &PoolManager{
		pools: make(map[string]*pgxpool.Pool),
	}
}

// GetPool 获取或创建连接池
// 如果 DSN 相同，则复用现有连接池
func (m *PoolManager) GetPool(ctx context.Context, dsn string) (*pgxpool.Pool, error) {
	if dsn == "" {
		return nil, fmt.Errorf("empty DSN")
	}

	m.mu.RLock()
	if pool, exists := m.pools[dsn]; exists {
		m.mu.RUnlock()
		return pool, nil
	}
	m.mu.RUnlock()

	// 需要创建新池
	m.mu.Lock()
	defer m.mu.Unlock()

	// 双重检查
	if pool, exists := m.pools[dsn]; exists {
		return pool, nil
	}

	config, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to parse DSN: %w", err)
	}

	newPool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("failed to create pool: %w", err)
	}

	if err := newPool.Ping(ctx); err != nil {
		newPool.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	m.pools[dsn] = newPool
	return newPool, nil
}

// Close 关闭所有连接池
func (m *PoolManager) Close() {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, pool := range m.pools {
		pool.Close()
	}
	m.pools = make(map[string]*pgxpool.Pool)
}

// PoolStats 返回所有连接池的统计信息
func (m *PoolManager) PoolStats() map[string]PoolStat {
	m.mu.RLock()
	defer m.mu.RUnlock()

	stats := make(map[string]PoolStat)
	for dsn, pool := range m.pools {
		stat := pool.Stat()
		stats[dsn] = PoolStat{
			TotalConns:    int(stat.TotalConns()),
			IdleConns:     int(stat.IdleConns()),
			AcquiredConns: int(stat.AcquiredConns()),
			MaxConns:      int(stat.MaxConns()),
		}
	}
	return stats
}

// PoolStat 连接池统计信息
type PoolStat struct {
	TotalConns    int
	IdleConns     int
	AcquiredConns int
	MaxConns      int
}
