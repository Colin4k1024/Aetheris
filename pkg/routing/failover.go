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
	"sync"
	"time"
)

// FailoverConfig 容灾配置
type FailoverConfig struct {
	MaxRetries        int     // 最大重试次数
	RetryDelayMs      int     // 重试延迟 (ms)
	EnableHotSwitch   bool    // 启用热切换
	BackoffMultiplier float64 // 退避倍数
	MaxRetryDelayMs   int     // 最大重试延迟
}

// DefaultFailoverConfig 返回默认容灾配置
func DefaultFailoverConfig() *FailoverConfig {
	return &FailoverConfig{
		MaxRetries:        2,
		RetryDelayMs:      1000,
		EnableHotSwitch:   true,
		BackoffMultiplier: 2.0,
		MaxRetryDelayMs:   10000,
	}
}

// FailoverHandler 容灾处理器
type FailoverHandler struct {
	config   *FailoverConfig
	router   Router
	registry *ModelRegistry
	mu       sync.RWMutex
}

// NewFailoverHandler 创建容灾处理器
func NewFailoverHandler(router Router, registry *ModelRegistry, config *FailoverConfig) *FailoverHandler {
	if config == nil {
		config = DefaultFailoverConfig()
	}
	if registry == nil {
		registry = DefaultModelRegistry()
	}

	return &FailoverHandler{
		config:   config,
		router:   router,
		registry: registry,
	}
}

// ExecuteWithFailover 带容灾的执行
type ExecuteFunc func(ctx context.Context, model *ModelInfo) (interface{}, error)

// ExecuteWithFailover 执行函数，支持容灾
func (h *FailoverHandler) ExecuteWithFailover(
	ctx context.Context,
	req *RoutingRequest,
	fn ExecuteFunc,
) (interface{}, *ModelInfo, error) {
	// 首次选择模型
	model, err := h.router.SelectModel(ctx, req)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to select model: %w", err)
	}

	var lastErr error
	var lastFallbackReason FallbackReason
	attempts := 0

	for attempts <= h.config.MaxRetries {
		// 执行调用
		startTime := time.Now()
		result, err := fn(ctx, model)
		latencyMs := int(time.Since(startTime).Milliseconds())

		// 记录结果
		outcome := &RoutingOutcome{
			Model:     model,
			Success:   err == nil,
			Error:     err,
			LatencyMs: latencyMs,
		}
		h.router.RecordOutcome(ctx, outcome)

		if err == nil {
			return result, model, nil
		}

		lastErr = err
		attempts++

		// 判断是否需要切换模型
		fallbackReason := h.determineFallbackReason(err)
		if fallbackReason == FallbackReasonNone || !h.config.EnableHotSwitch {
			// 不需要切换或禁用了热切换
			if attempts <= h.config.MaxRetries {
				// 重试同一模型
				delay := h.calculateRetryDelay(attempts)
				select {
				case <-ctx.Done():
					return nil, model, ctx.Err()
				case <-time.After(delay):
					continue
				}
			}
			break
		}

		// 需要切换模型
		newModel, fallbackErr := h.router.SelectFallback(ctx, model, fallbackReason)
		if fallbackErr != nil {
			// 无法获取备用模型
			if attempts <= h.config.MaxRetries {
				delay := h.calculateRetryDelay(attempts)
				select {
				case <-ctx.Done():
					return nil, model, ctx.Err()
				case <-time.After(delay):
					continue
				}
			}
			break
		}

		// 记录切换
		lastFallbackReason = fallbackReason
		model = newModel

		// 重试新模型
		if attempts <= h.config.MaxRetries {
			delay := h.calculateRetryDelay(attempts)
			select {
			case <-ctx.Done():
				return nil, model, ctx.Err()
			case <-time.After(delay):
				continue
			}
		}
	}

	// 所有重试都失败
	return nil, model, fmt.Errorf("max retries exceeded (%d), last error: %w, fallback reason: %s",
		attempts, lastErr, lastFallbackReason)
}

// determineFallbackReason 确定是否需要降级
func (h *FailoverHandler) determineFallbackReason(err error) FallbackReason {
	if err == nil {
		return FallbackReasonNone
	}

	errStr := err.Error()

	// 检查限流
	if containsAny(errStr, "rate limit") || containsAny(errStr, "429") || containsAny(errStr, "too many requests") {
		return FallbackReasonRateLimit
	}

	// 检查服务端错误
	if containsAny(errStr, "500") || containsAny(errStr, "502") ||
		containsAny(errStr, "503") || containsAny(errStr, "server error") {
		return FallbackReasonServerError
	}

	// 检查超时
	if containsAny(errStr, "timeout") || containsAny(errStr, "deadline exceeded") {
		return FallbackReasonTimeout
	}

	// 检查配额
	if containsAny(errStr, "quota") || containsAny(errStr, "exceeded") {
		return FallbackReasonCost
	}

	// 其他错误，尝试降级
	return FallbackReasonServerError
}

// calculateRetryDelay 计算重试延迟（带退避）
func (h *FailoverHandler) calculateRetryDelay(attempt int) time.Duration {
	delay := float64(h.config.RetryDelayMs) * pow(h.config.BackoffMultiplier, float64(attempt-1))
	if delay > float64(h.config.MaxRetryDelayMs) {
		delay = float64(h.config.MaxRetryDelayMs)
	}
	return time.Duration(delay) * time.Millisecond
}

// pow 计算幂
func pow(base, exp float64) float64 {
	result := 1.0
	for i := 0; i < int(exp); i++ {
		result *= base
	}
	return result
}

// RateLimitChecker 限流检查器
type RateLimitChecker struct {
	mu           sync.RWMutex
	providerLast map[string]time.Time     // provider -> last request time
	providerRate map[string]time.Duration // provider -> min interval
}

// NewRateLimitChecker 创建限流检查器
func NewRateLimitChecker() *RateLimitChecker {
	return &RateLimitChecker{
		providerLast: make(map[string]time.Time),
		providerRate: make(map[string]time.Duration),
	}
}

// SetRateLimit 设置提供商限流配置
func (c *RateLimitChecker) SetRateLimit(provider string, interval time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.providerRate[provider] = interval
}

// CheckAndWait 检查并等待（如果需要）
func (c *RateLimitChecker) CheckAndWait(ctx context.Context, provider string) error {
	c.mu.RLock()
	interval, ok := c.providerRate[provider]
	lastTime, hasLast := c.providerLast[provider]
	c.mu.RUnlock()

	if !ok {
		return nil // 没有限流配置
	}

	if hasLast {
		elapsed := time.Since(lastTime)
		if elapsed < interval {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(interval - elapsed):
			}
		}
	}

	// 更新最后请求时间
	c.mu.Lock()
	c.providerLast[provider] = time.Now()
	c.mu.Unlock()

	return nil
}

// Reset 重置限流状态
func (c *RateLimitChecker) Reset(provider string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.providerLast, provider)
}

// EventSourcedContext 事件溯源上下文
// 确保模型切换时保留完整上下文
type EventSourcedContext struct {
	mu            sync.RWMutex
	RequestID     string
	Messages      []MessageSnapshot
	TokensUsed    int
	SwitchHistory []SwitchEvent
	LastModel     string
	LastError     error
}

type MessageSnapshot struct {
	Role    string
	Content string
	Tokens  int
}

type SwitchEvent struct {
	Timestamp  time.Time
	FromModel  string
	ToModel    string
	Reason     FallbackReason
	TokensUsed int
}

// NewEventSourcedContext 创建事件溯源上下文
func NewEventSourcedContext(requestID string) *EventSourcedContext {
	return &EventSourcedContext{
		RequestID:     requestID,
		Messages:      make([]MessageSnapshot, 0),
		SwitchHistory: make([]SwitchEvent, 0),
	}
}

// AddMessage 添加消息
func (c *EventSourcedContext) AddMessage(role, content string, tokens int) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.Messages = append(c.Messages, MessageSnapshot{
		Role:    role,
		Content: content,
		Tokens:  tokens,
	})
	c.TokensUsed += tokens
}

// RecordSwitch 记录模型切换
func (c *EventSourcedContext) RecordSwitch(fromModel, toModel string, reason FallbackReason) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.SwitchHistory = append(c.SwitchHistory, SwitchEvent{
		Timestamp:  time.Now(),
		FromModel:  fromModel,
		ToModel:    toModel,
		Reason:     reason,
		TokensUsed: c.TokensUsed,
	})
	c.LastModel = toModel
}

// GetMessages 获取消息快照
func (c *EventSourcedContext) GetMessages() []MessageSnapshot {
	c.mu.RLock()
	defer c.mu.RUnlock()
	msgs := make([]MessageSnapshot, len(c.Messages))
	copy(msgs, c.Messages)
	return msgs
}

// GetSwitchHistory 获取切换历史
func (c *EventSourcedContext) GetSwitchHistory() []SwitchEvent {
	c.mu.RLock()
	defer c.mu.RUnlock()
	history := make([]SwitchEvent, len(c.SwitchHistory))
	copy(history, c.SwitchHistory)
	return history
}

// HasSwitched 检查是否发生过切换
func (c *EventSourcedContext) HasSwitched() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.SwitchHistory) > 0
}

// GetTotalTokens 获取总消耗 tokens
func (c *EventSourcedContext) GetTotalTokens() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.TokensUsed
}
