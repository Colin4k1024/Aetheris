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

package region

import (
	"context"
	"hash/fnv"

	"rag-platform/internal/runtime/jobstore"
)

// Scheduler 区域感知调度器
type Scheduler struct {
	store    jobstore.JobStore
	regionID string
	config   *Config
}

// NewScheduler 创建区域感知调度器
func NewScheduler(store jobstore.JobStore, regionID string, config *Config) *Scheduler {
	return &Scheduler{
		store:    store,
		regionID: regionID,
		config:   config,
	}
}

// SelectRegionForJob 根据 job 特性选择最佳区域
func (s *Scheduler) SelectRegionForJob(jobID string) string {
	// 如果启用跨区域复制，使用哈希确保同一 job 路由到同一区域
	if s.config.EnableCrossRegionReplication {
		// 基于 jobID 哈希选择区域
		h := fnv.New32a()
		h.Write([]byte(jobID))
		idx := int(h.Sum32()) % len(s.config.Regions)
		return s.config.Regions[idx].ID
	}

	// 默认使用本地区域
	return s.regionID
}

// ShouldExecuteLocal 判断 job 是否应该在本地执行
func (s *Scheduler) ShouldExecuteLocal(jobID string) bool {
	selectedRegion := s.SelectRegionForJob(jobID)
	return selectedRegion == s.regionID
}

// ClaimWithRegion 带有区域感知的 claim
func (s *Scheduler) ClaimWithRegion(ctx context.Context, workerID string) (string, int, string, error) {
	// 本地优先策略：先尝试本地 claim
	store, ok := s.store.(*jobstore.ShardedStore)
	if !ok {
		// 非分片存储，直接 claim
		return s.store.Claim(ctx, workerID)
	}

	// 分片存储：使用本地优先调度
	return store.Claim(ctx, workerID)
}

// GetRegionStats 获取区域统计信息
func (s *Scheduler) GetRegionStats(ctx context.Context) (map[string]int, error) {
	stats := make(map[string]int)

	for _, r := range s.config.Regions {
		stats[r.ID] = 0
	}

	// 如果是分片存储，统计各分片
	store, ok := s.store.(*jobstore.ShardedStore)
	if ok {
		// 返回分片数量作为区域负载指标
		stats[s.regionID] = store.ShardCount()
	}

	return stats, nil
}
