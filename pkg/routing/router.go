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

package routing

import (
	"context"
	"fmt"
	"sort"
	"sync"
)

// Router 模型路由器接口
type Router interface {
	// SelectModel 选择最适合的模型
	SelectModel(ctx context.Context, req *RoutingRequest) (*ModelInfo, error)

	// SelectFallback 获取备用模型
	SelectFallback(ctx context.Context, primary *ModelInfo, reason FallbackReason) (*ModelInfo, error)

	// RecordOutcome 记录路由结果（用于优化）
	RecordOutcome(ctx context.Context, outcome *RoutingOutcome)
}

// NewRouter 创建路由器
func NewRouter(registry *ModelRegistry, strategy RoutingStrategy, config *Config) Router {
	if registry == nil {
		registry = DefaultModelRegistry()
	}
	if strategy == nil {
		strategy = NewBalancedStrategy(registry)
	}
	if config == nil {
		config = DefaultConfig()
	}

	return &defaultRouter{
		registry: registry,
		strategy: strategy,
		config:   config,
		mu:       sync.RWMutex{},
	}
}

// defaultRouter 默认路由器实现
type defaultRouter struct {
	registry *ModelRegistry
	strategy RoutingStrategy
	config   *Config
	mu       sync.RWMutex
}

// SelectModel 选择模型
func (r *defaultRouter) SelectModel(ctx context.Context, req *RoutingRequest) (*ModelInfo, error) {
	if req == nil {
		return nil, fmt.Errorf("routing request is nil")
	}

	// 使用策略选择模型
	model, err := r.strategy.Select(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to select model: %w", err)
	}

	// Record selection metrics (if metrics package available)
	// Note: In production, integrate with pkg/metrics

	return model, nil
}

// SelectFallback 获取备用模型
func (r *defaultRouter) SelectFallback(ctx context.Context, primary *ModelInfo, reason FallbackReason) (*ModelInfo, error) {
	if primary == nil {
		return nil, fmt.Errorf("primary model is nil")
	}

	// 根据原因确定备用层级
	tier := primary.Tier

	switch reason {
	case FallbackReasonRateLimit, FallbackReasonServerError, FallbackReasonTimeout:
		// 错误情况，尝试低一级模型（更便宜）
		if tier < TierEconomy {
			tier = tier + 1
		}
	case FallbackReasonCost:
		// 成本问题，尝试更便宜的模型
		if tier < TierEconomy {
			tier = tier + 1
		}
	case FallbackReasonQuality:
		// 质量问题，升级到更高级别（更好的模型）
		if tier > TierReasoning {
			tier = tier - 1
		}
	default:
		// 默认降级到均衡模型
		tier = TierBalanced
	}

	// 获取该层级的模型
	models := r.registry.GetModelsByTier(tier)
	if len(models) == 0 {
		// 没有备用模型，返回原模型
		return primary, nil
	}

	// 选择第一个可用的模型（可以改进为根据性能选择）
	fallback := models[0]

	// Record fallback metrics (if metrics package available)
	// Note: In production, integrate with pkg/metrics

	return fallback, nil
}

// RecordOutcome 记录结果
func (r *defaultRouter) RecordOutcome(ctx context.Context, outcome *RoutingOutcome) {
	if outcome == nil {
		return
	}

	// Note: In production, integrate with pkg/metrics
	// Record latency and error metrics here
}

func containsAny(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// RoutingStrategy 路由策略接口
type RoutingStrategy interface {
	// Name 返回策略名称
	Name() string

	// Select 选择模型
	Select(ctx context.Context, req *RoutingRequest) (*ModelInfo, error)
}

// Config 路由器配置
type Config struct {
	EnableHotSwitch bool   // 启用热切换
	MaxRetries      int    // 最大重试次数
	RetryDelayMs    int    // 重试延迟
	DefaultStrategy string // 默认策略
}

// DefaultConfig 返回默认配置
func DefaultConfig() *Config {
	return &Config{
		EnableHotSwitch: true,
		MaxRetries:      2,
		RetryDelayMs:    1000,
		DefaultStrategy: "balanced",
	}
}

// costStrategy 成本优先策略
type costStrategy struct {
	registry *ModelRegistry
}

func NewCostStrategy(registry *ModelRegistry) RoutingStrategy {
	return &costStrategy{registry: registry}
}

func (s *costStrategy) Name() string {
	return "cost"
}

func (s *costStrategy) Select(ctx context.Context, req *RoutingRequest) (*ModelInfo, error) {
	return s.selectByTier(req, false)
}

// latencyStrategy 延迟优先策略
type latencyStrategy struct {
	registry *ModelRegistry
}

func NewLatencyStrategy(registry *ModelRegistry) RoutingStrategy {
	return &latencyStrategy{registry: registry}
}

func (s *latencyStrategy) Name() string {
	return "latency"
}

func (s *latencyStrategy) Select(ctx context.Context, req *RoutingRequest) (*ModelInfo, error) {
	// 优先选择延迟低的模型
	models := s.registry.GetAllModels()
	if len(models) == 0 {
		return nil, fmt.Errorf("no models available")
	}

	// 按延迟排序
	sort.Slice(models, func(i, j int) bool {
		return models[i].Model().AvgLatencyMs < models[j].Model().AvgLatencyMs
	})

	// 根据复杂度选择合适的延迟级别
	targetLatency := req.MaxLatencyMs
	if targetLatency <= 0 {
		targetLatency = 10000 // 默认 10s
	}

	for _, iter := range models {
		model := iter.Model()
		if model.AvgLatencyMs <= targetLatency && meetsRequirements(model, req) {
			return model, nil
		}
	}

	// 没有满足延迟要求的，返回最快的
	return models[0].Model(), nil
}

// qualityStrategy 质量优先策略
type qualityStrategy struct {
	registry *ModelRegistry
}

func NewQualityStrategy(registry *ModelRegistry) RoutingStrategy {
	return &qualityStrategy{registry: registry}
}

func (s *qualityStrategy) Name() string {
	return "quality"
}

func (s *qualityStrategy) Select(ctx context.Context, req *RoutingRequest) (*ModelInfo, error) {
	// 优先选择质量最高的模型
	tier := TierReasoning
	if req.Complexity == ComplexitySimple {
		tier = TierBalanced
	} else if req.Complexity == ComplexityMedium {
		tier = TierFlagship
	}

	models := s.registry.GetModelsByTier(tier)
	if len(models) == 0 {
		// 降级获取
		models = s.registry.GetModelsByTier(tier - 1)
	}

	if len(models) == 0 {
		return nil, fmt.Errorf("no models available")
	}

	// 返回第一个满足要求的模型
	for _, model := range models {
		if meetsRequirements(model, req) {
			return model, nil
		}
	}

	return models[0], nil
}

// balancedStrategy 均衡策略
type balancedStrategy struct {
	registry *ModelRegistry
}

func NewBalancedStrategy(registry *ModelRegistry) RoutingStrategy {
	return &balancedStrategy{registry: registry}
}

func (s *balancedStrategy) Name() string {
	return "balanced"
}

func (s *balancedStrategy) Select(ctx context.Context, req *RoutingRequest) (*ModelInfo, error) {
	// 根据复杂度确定目标层级
	targetTier := s.getTargetTier(req)
	models := s.registry.GetModelsByTier(targetTier)

	// 如果没有找到，尝试其他层级
	if len(models) == 0 {
		for tier := targetTier - 1; tier >= TierEconomy; tier-- {
			models = s.registry.GetModelsByTier(tier)
			if len(models) > 0 {
				break
			}
		}
	}

	if len(models) == 0 {
		return nil, fmt.Errorf("no models available")
	}

	// 选择成本效益最好的模型
	best := s.selectBestCostPerformance(models, req)
	return best, nil
}

func (s *balancedStrategy) getTargetTier(req *RoutingRequest) ModelTier {
	switch req.Complexity {
	case ComplexitySimple:
		if req.Priority == PriorityCost {
			return TierEconomy
		}
		return TierBalanced
	case ComplexityMedium:
		return TierBalanced
	case ComplexityHigh:
		if req.Priority == PriorityCost {
			return TierBalanced
		}
		return TierFlagship
	default:
		return TierBalanced
	}
}

func (s *balancedStrategy) selectBestCostPerformance(models []*ModelInfo, req *RoutingRequest) *ModelInfo {
	var best *ModelInfo
	bestScore := -1.0

	for _, model := range models {
		if !meetsRequirements(model, req) {
			continue
		}

		// 计算成本效益分数 (越低延迟越好, 越低成本越好)
		cost := model.GetEstimatedCost()
		latency := float64(model.AvgLatencyMs)

		// 分数 = 1 / (成本 * 延迟)，越高越好
		score := 1.0 / (cost*0.5 + latency*0.0001)

		if best == nil || score > bestScore {
			best = model
			bestScore = score
		}
	}

	if best == nil {
		return models[0]
	}

	return best
}

func (s *costStrategy) selectByTier(req *RoutingRequest, preferLower bool) (*ModelInfo, error) {
	targetTier := s.getTargetTier(req)
	models := s.registry.GetModelsByTier(targetTier)

	if len(models) == 0 {
		// 降级获取
		for tier := targetTier - 1; tier >= TierEconomy; tier-- {
			models = s.registry.GetModelsByTier(tier)
			if len(models) > 0 {
				break
			}
		}
	}

	if len(models) == 0 {
		return nil, fmt.Errorf("no models available")
	}

	// 按成本排序
	sort.Slice(models, func(i, j int) bool {
		return models[i].GetEstimatedCost() < models[j].GetEstimatedCost()
	})

	// 返回最便宜的
	for _, model := range models {
		if meetsRequirements(model, req) {
			return model, nil
		}
	}

	return models[0], nil
}

func (s *costStrategy) getTargetTier(req *RoutingRequest) ModelTier {
	switch req.Complexity {
	case ComplexitySimple:
		return TierEconomy
	case ComplexityMedium:
		return TierEconomy
	case ComplexityHigh:
		return TierBalanced
	default:
		return TierEconomy
	}
}

// meetsRequirements 检查模型是否满足要求
func meetsRequirements(model *ModelInfo, req *RoutingRequest) bool {
	// 检查成本约束
	if req.MaxCost > 0 && model.GetEstimatedCost() > req.MaxCost {
		return false
	}

	// 检查延迟约束
	if req.MaxLatencyMs > 0 && model.AvgLatencyMs > req.MaxLatencyMs {
		return false
	}

	// 检查必需能力
	for _, cap := range req.RequiredCaps {
		if !model.HasCapability(cap) {
			return false
		}
	}

	// 检查首选提供商
	if req.PreferProvider != "" && model.Provider != req.PreferProvider {
		return false
	}

	return true
}
