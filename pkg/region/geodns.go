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

package region

import (
	"context"
	"math"
	"sort"
	"sync"
	"time"
)

// GeoDNSResolver GeoDNS 解析器 - 基于延迟的路由
type GeoDNSResolver struct {
	mu           sync.RWMutex
	regions      map[string]*Region
	latencyMap   map[string]time.Duration // regionID -> latency
	lastUpdate   time.Time
	updatePeriod time.Duration
}

// NewGeoDNSResolver 创建新的 GeoDNS 解析器
func NewGeoDNSResolver(regions []Region) *GeoDNSResolver {
	regionMap := make(map[string]*Region)
	for i := range regions {
		regionMap[regions[i].ID] = &regions[i]
	}

	return &GeoDNSResolver{
		regions:      regionMap,
		latencyMap:   make(map[string]time.Duration),
		updatePeriod: 30 * time.Second,
	}
}

// Resolve 根据延迟选择最佳区域
func (g *GeoDNSResolver) Resolve(ctx context.Context) string {
	g.mu.RLock()
	defer g.mu.RUnlock()

	// 如果没有延迟数据，返回主区域
	var primary *Region

	for _, r := range g.regions {
		if r.IsPrimary {
			primary = r
			break
		}
	}

	if primary == nil {
		// 没有主区域，返回第一个
		for _, r := range g.regions {
			return r.ID
		}
	}

	// 找到最低延迟的区域
	var bestRegion *Region
	var bestLatency time.Duration = math.MaxInt64

	for id, lat := range g.latencyMap {
		if lat < bestLatency {
			bestLatency = lat
			bestRegion = g.regions[id]
		}
	}

	// 如果没有延迟数据或最佳区域不可用，使用主区域
	if bestRegion == nil {
		return primary.ID
	}

	// 如果最佳区域延迟超过阈值，使用主区域
	if bestLatency > 5*time.Second {
		return primary.ID
	}

	return bestRegion.ID
}

// ResolveWithFallback 返回带回退的最佳区域
func (g *GeoDNSResolver) ResolveWithFallback(ctx context.Context, fallbackRegions []string) string {
	// 首先尝试找到最低延迟的区域
	g.mu.RLock()
	defer g.mu.RUnlock()

	type regionLatency struct {
		regionID string
		latency  time.Duration
	}

	var latencies []regionLatency
	for id, lat := range g.latencyMap {
		latencies = append(latencies, regionLatency{regionID: id, latency: lat})
	}

	if len(latencies) == 0 {
		// 没有延迟数据，使用回退列表
		if len(fallbackRegions) > 0 {
			return fallbackRegions[0]
		}
		// 返回第一个可用区域
		for id := range g.regions {
			return id
		}
	}

	// 按延迟排序
	sort.Slice(latencies, func(i, j int) bool {
		return latencies[i].latency < latencies[j].latency
	})

	// 返回最低延迟的区域
	return latencies[0].regionID
}

// UpdateLatency 更新区域延迟
func (g *GeoDNSResolver) UpdateLatency(regionID string, latency time.Duration) {
	g.mu.Lock()
	defer g.mu.Unlock()

	g.latencyMap[regionID] = latency
	g.lastUpdate = time.Now()
}

// GetLatency 获取区域延迟
func (g *GeoDNSResolver) GetLatency(regionID string) time.Duration {
	g.mu.RLock()
	defer g.mu.RUnlock()

	return g.latencyMap[regionID]
}

// GetAllLatencies 获取所有区域延迟
func (g *GeoDNSResolver) GetAllLatencies() map[string]time.Duration {
	g.mu.RLock()
	defer g.mu.RUnlock()

	result := make(map[string]time.Duration)
	for k, v := range g.latencyMap {
		result[k] = v
	}
	return result
}

// RegionFailoverManager 区域故障转移管理器
type RegionFailoverManager struct {
	mu           sync.RWMutex
	regions      map[string]*Region
	failoverChan chan FailoverEvent
}

// FailoverEvent 故障转移事件
type FailoverEvent struct {
	FromRegion string
	ToRegion   string
	JobID      string
	Timestamp  time.Time
	Reason     string
}

// NewRegionFailoverManager 创建故障转移管理器
func NewRegionFailoverManager(regions []Region) *RegionFailoverManager {
	regionMap := make(map[string]*Region)
	for i := range regions {
		regionMap[regions[i].ID] = &regions[i]
	}

	return &RegionFailoverManager{
		regions:      regionMap,
		failoverChan: make(chan FailoverEvent, 100),
	}
}

// IsRegionHealthy 检查区域是否健康
func (m *RegionFailoverManager) IsRegionHealthy(regionID string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	region, exists := m.regions[regionID]
	if !exists {
		return false
	}

	return !region.Disabled
}

// GetFallbackRegion 获取回退区域
func (m *RegionFailoverManager) GetFallbackRegion(ctx context.Context, failedRegion string) string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// 查找同一数据中心或最近区域的回退
	var fallback *Region
	minDistance := math.MaxInt64

	for _, r := range m.regions {
		if r.ID == failedRegion || r.Disabled {
			continue
		}

		// 简单距离计算：同一 continent 优先
		distance := 1
		if r.Continent != m.regions[failedRegion].Continent {
			distance = 10
		}

		if distance < minDistance {
			minDistance = distance
			fallback = r
		}
	}

	if fallback != nil {
		return fallback.ID
	}

	// 返回第一个可用区域
	for id, r := range m.regions {
		if !r.Disabled {
			return id
		}
	}

	return ""
}

// TriggerFailover 触发故障转移
func (m *RegionFailoverManager) TriggerFailover(ctx context.Context, fromRegion, toRegion, jobID, reason string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	event := FailoverEvent{
		FromRegion: fromRegion,
		ToRegion:   toRegion,
		JobID:      jobID,
		Timestamp:  time.Now(),
		Reason:     reason,
	}

	select {
	case m.failoverChan <- event:
	default:
		// channel full, drop event
	}
}

// Subscribe 订阅故障转移事件
func (m *RegionFailoverManager) Subscribe() <-chan FailoverEvent {
	return m.failoverChan
}
