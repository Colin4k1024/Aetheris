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
	"fmt"
	"sync"
	"time"
)

// ModelTier 模型级别
type ModelTier int

const (
	TierReasoning ModelTier = iota + 1 // T1: 推理模型
	TierFlagship                       // T2: 旗舰模型
	TierBalanced                       // T3: 均衡模型
	TierEconomy                        // T4: 经济模型
)

// ModelTierString 返回 tier 的字符串表示
func (t ModelTier) String() string {
	switch t {
	case TierReasoning:
		return "t1-reasoning"
	case TierFlagship:
		return "t2-flagship"
	case TierBalanced:
		return "t3-balanced"
	case TierEconomy:
		return "t4-economy"
	default:
		return "unknown"
	}
}

// ParseModelTier 从字符串解析 tier
func ParseModelTier(s string) (ModelTier, error) {
	switch s {
	case "t1-reasoning", "reasoning":
		return TierReasoning, nil
	case "t2-flagship", "flagship":
		return TierFlagship, nil
	case "t3-balanced", "balanced":
		return TierBalanced, nil
	case "t4-economy", "economy":
		return TierEconomy, nil
	default:
		return 0, fmt.Errorf("unknown tier: %s", s)
	}
}

// NodeComplexity 节点复杂度
type NodeComplexity int

const (
	ComplexitySimple NodeComplexity = iota
	ComplexityMedium
	ComplexityHigh
)

// String 返回复杂度字符串
func (c NodeComplexity) String() string {
	switch c {
	case ComplexitySimple:
		return "simple"
	case ComplexityMedium:
		return "medium"
	case ComplexityHigh:
		return "high"
	default:
		return "unknown"
	}
}

// RoutingPriority 路由优先级
type RoutingPriority int

const (
	PriorityCost RoutingPriority = iota
	PriorityBalanced
	PriorityLatency
	PriorityQuality
)

// String 返回优先级字符串
func (p RoutingPriority) String() string {
	switch p {
	case PriorityCost:
		return "cost"
	case PriorityBalanced:
		return "balanced"
	case PriorityLatency:
		return "latency"
	case PriorityQuality:
		return "quality"
	default:
		return "unknown"
	}
}

// FallbackReason 降级原因
type FallbackReason int

const (
	FallbackReasonNone FallbackReason = iota
	FallbackReasonRateLimit     // 限流 (429)
	FallbackReasonServerError   // 服务端错误 (5xx)
	FallbackReasonTimeout       // 超时
	FallbackReasonQuality        // 质量不满足
	FallbackReasonCost           // 成本超限
)

// String 返回降级原因字符串
func (r FallbackReason) String() string {
	switch r {
	case FallbackReasonNone:
		return "none"
	case FallbackReasonRateLimit:
		return "rate_limit"
	case FallbackReasonServerError:
		return "server_error"
	case FallbackReasonTimeout:
		return "timeout"
	case FallbackReasonQuality:
		return "quality"
	case FallbackReasonCost:
		return "cost"
	default:
		return "unknown"
	}
}

// ModelInfo 模型信息
type ModelInfo struct {
	Name             string
	Provider         string
	Tier             ModelTier
	ContextLimit     int           // tokens
	CostPer1KInput   float64        // USD
	CostPer1KOutput  float64        // USD
	AvgLatencyMs     int            // 平均延迟
	Capabilities     []string       // 能力标签
	MaxRetries       int            // 最大重试次数
}

// GetEstimatedCost 估算单次请求成本 (假设 1K input, 500 output)
func (m *ModelInfo) GetEstimatedCost() float64 {
	return m.CostPer1KInput*1 + m.CostPer1KOutput*0.5
}

// HasCapability 检查是否具备指定能力
func (m *ModelInfo) HasCapability(cap string) bool {
	for _, c := range m.Capabilities {
		if c == cap {
			return true
		}
	}
	return false
}

// RoutingRequest 路由请求
type RoutingRequest struct {
	Complexity     NodeComplexity
	MaxCost        float64          // 最大成本 ($)
	MaxLatencyMs   int              // 最大延迟 (ms)
	RequiredCaps   []string         // 必需能力
	PreferProvider string           // 首选提供商
	Priority       RoutingPriority  // 优先级
	RequestID      string           // 请求ID (用于追踪)
}

// RoutingOutcome 路由结果
type RoutingOutcome struct {
	Model          *ModelInfo
	Success        bool
	Error          error
	TokensUsed     int
	LatencyMs      int
	FallbackReason FallbackReason
	Fallbacked     bool
}

// ModelRegistry 模型注册表
type ModelRegistry struct {
	mu       sync.RWMutex
	models   map[string]*ModelInfo // model name -> ModelInfo
	byTier   map[ModelTier][]*ModelInfo
	byProvider map[string][]*ModelInfo
}

// NewModelRegistry 创建模型注册表
func NewModelRegistry() *ModelRegistry {
	return &ModelRegistry{
		models: make(map[string]*ModelInfo),
		byTier: make(map[ModelTier][]*ModelInfo),
		byProvider: make(map[string][]*ModelInfo),
	}
}

// RegisterModel 注册模型
func (r *ModelRegistry) RegisterModel(model *ModelInfo) {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	r.models[model.Name] = model
	r.byTier[model.Tier] = append(r.byTier[model.Tier], model)
	r.byProvider[model.Provider] = append(r.byProvider[model.Provider], model)
}

// GetModel 获取模型信息
func (r *ModelRegistry) GetModel(name string) (*ModelInfo, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	model, ok := r.models[name]
	if !ok {
		return nil, fmt.Errorf("model not found: %s", name)
	}
	return model, nil
}

// GetModelsByTier 获取指定级别的所有模型
func (r *ModelRegistry) GetModelsByTier(tier ModelTier) []*ModelInfo {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	models := make([]*ModelInfo, len(r.byTier[tier]))
	copy(models, r.byTier[tier])
	return models
}

// GetAllModels 获取所有模型
func (r *ModelRegistry) GetAllModels() []*ModelRegistryIterator {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	result := make([]*ModelRegistryIterator, 0, len(r.models))
	for _, model := range r.models {
		result = append(result, &ModelRegistryIterator{model: model})
	}
	return result
}

// ModelRegistryIterator 模型注册表迭代器
type ModelRegistryIterator struct {
	model *ModelInfo
}

// Model 获取模型信息
func (i *ModelRegistryIterator) Model() *ModelInfo {
	return i.model
}

// DefaultModelRegistry 返回默认模型注册表（预置常用模型）
func DefaultModelRegistry() *ModelRegistry {
	registry := NewModelRegistry()
	
	// T1: 推理模型
	registry.RegisterModel(&ModelInfo{
		Name:            "o1",
		Provider:        "openai",
		Tier:            TierReasoning,
		ContextLimit:    200000,
		CostPer1KInput:  0.015,
		CostPer1KOutput: 0.075,
		AvgLatencyMs:    30000,
		Capabilities:    []string{"reasoning", "code", "reasoning_native"},
		MaxRetries:      2,
	})
	registry.RegisterModel(&ModelInfo{
		Name:            "o3-mini",
		Provider:        "openai",
		Tier:            TierReasoning,
		ContextLimit:    200000,
		CostPer1KInput:  0.0011,
		CostPer1KOutput: 0.0044,
		AvgLatencyMs:    20000,
		Capabilities:    []string{"reasoning", "code", "reasoning_native"},
		MaxRetries:      2,
	})
	
	// T2: 旗舰模型
	registry.RegisterModel(&ModelInfo{
		Name:            "gpt-4o",
		Provider:        "openai",
		Tier:            TierFlagship,
		ContextLimit:    128000,
		CostPer1KInput:  0.0025,
		CostPer1KOutput: 0.01,
		AvgLatencyMs:    2000,
		Capabilities:    []string{"vision", "function_call"},
		MaxRetries:      3,
	})
	registry.RegisterModel(&ModelInfo{
		Name:            "claude-4-opus",
		Provider:        "anthropic",
		Tier:            TierFlagship,
		ContextLimit:    200000,
		CostPer1KInput:  0.015,
		CostPer1KOutput: 0.075,
		AvgLatencyMs:    3000,
		Capabilities:    []string{"vision", "function_call"},
		MaxRetries:      3,
	})
	registry.RegisterModel(&ModelInfo{
		Name:            "gemini-2.5-pro",
		Provider:        "google",
		Tier:            TierFlagship,
		ContextLimit:    1000000,
		CostPer1KInput:  0.00125,
		CostPer1KOutput: 0.005,
		AvgLatencyMs:    2500,
		Capabilities:    []string{"vision", "long_context"},
		MaxRetries:      3,
	})
	
	// T3: 均衡模型
	registry.RegisterModel(&ModelInfo{
		Name:            "gpt-4o-mini",
		Provider:        "openai",
		Tier:            TierBalanced,
		ContextLimit:    128000,
		CostPer1KInput:  0.00015,
		CostPer1KOutput: 0.0006,
		AvgLatencyMs:    800,
		Capabilities:    []string{"vision", "function_call"},
		MaxRetries:      3,
	})
	registry.RegisterModel(&ModelInfo{
		Name:            "claude-3.5-sonnet",
		Provider:        "anthropic",
		Tier:            TierBalanced,
		ContextLimit:    200000,
		CostPer1KInput:  0.003,
		CostPer1KOutput: 0.015,
		AvgLatencyMs:    1500,
		Capabilities:    []string{"vision", "function_call"},
		MaxRetries:      3,
	})
	registry.RegisterModel(&ModelInfo{
		Name:            "gemini-1.5-flash",
		Provider:        "google",
		Tier:            TierBalanced,
		ContextLimit:    1000000,
		CostPer1KInput:  0.000075,
		CostPer1KOutput: 0.0003,
		AvgLatencyMs:    500,
		Capabilities:    []string{"vision", "long_context"},
		MaxRetries:      3,
	})
	
	// T4: 经济模型
	registry.RegisterModel(&ModelInfo{
		Name:            "qwen-turbo",
		Provider:        "qwen",
		Tier:            TierEconomy,
		ContextLimit:    100000,
		CostPer1KInput:  0.0002,
		CostPer1KOutput: 0.0006,
		AvgLatencyMs:    300,
		Capabilities:    []string{},
		MaxRetries:      3,
	})
	registry.RegisterModel(&ModelInfo{
		Name:            "gpt-3.5-turbo",
		Provider:        "openai",
		Tier:            TierEconomy,
		ContextLimit:    16385,
		CostPer1KInput:  0.0005,
		CostPer1KOutput: 0.0015,
		AvgLatencyMs:    400,
		Capabilities:    []string{"function_call"},
		MaxRetries:      3,
	})
	registry.RegisterModel(&ModelInfo{
		Name:            "claude-3-haiku",
		Provider:        "anthropic",
		Tier:            TierEconomy,
		ContextLimit:    200000,
		CostPer1KInput:  0.00025,
		CostPer1KOutput: 0.00125,
		AvgLatencyMs:    500,
		Capabilities:    []string{"vision"},
		MaxRetries:      3,
	})
	
	return registry
}

// RouterAuditLog 路由审计日志
type RouterAuditLog struct {
	Timestamp        time.Time
	RequestID        string
	Strategy         string
	SelectedTier     string
	SelectedModel    string
	Complexity       NodeComplexity
	MaxCost          float64
	MaxLatencyMs     int
	FallbackFrom     string
	FallbackTo       string
	FallbackReason   string
	TokensUsed       int
	LatencyMs        int
	Error            string
}
