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

package sla

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// QuotaType Quota 类型
type QuotaType string

const (
	QuotaTypeJobs       QuotaType = "jobs"       // Job 数量
	QuotaTypeTokens     QuotaType = "tokens"     // Token 数量
	QuotaTypeRequests   QuotaType = "requests"   // 请求次数
	QuotaTypeConcurrent QuotaType = "concurrent" // 并发数
)

// Quota 租户配额
type Quota struct {
	TenantID string        `yaml:"tenant_id"`
	Type     QuotaType     `yaml:"type"`
	Limit    int64         `yaml:"limit"`
	Window   time.Duration `yaml:"window"` // 时间窗口
	Used     int64         `yaml:"-"`      // 当前使用量
	ResetAt  time.Time     `yaml:"-"`      // 重置时间
}

// QuotaManager Quota 管理器
type QuotaManager struct {
	mu           sync.RWMutex
	quotas       map[string]*TenantQuota // key: tenantID
	defaultQuota *TenantQuota
}

// TenantQuota 租户配额配置
type TenantQuota struct {
	TenantID     string
	JobsPerHour  int64
	TokensPerMin int64
	RequestsPerS int64
	Concurrent   int64
}

// NewQuotaManager 创建 Quota 管理器
func NewQuotaManager() *QuotaManager {
	return &QuotaManager{
		quotas: make(map[string]*TenantQuota),
		defaultQuota: &TenantQuota{
			TenantID:     "default",
			JobsPerHour:  1000,
			TokensPerMin: 100000,
			RequestsPerS: 100,
			Concurrent:   10,
		},
	}
}

// SetQuota 设置租户配额
func (m *QuotaManager) SetQuota(quota *TenantQuota) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.quotas[quota.TenantID] = quota
}

// GetQuota 获取租户配额
func (m *QuotaManager) GetQuota(tenantID string) *TenantQuota {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if q, ok := m.quotas[tenantID]; ok {
		return q
	}
	return m.defaultQuota
}

// CheckQuota 检查配额是否允许
func (m *QuotaManager) CheckQuota(tenantID string, quotaType QuotaType) (allowed bool, remaining int64, retryAfter time.Duration) {
	quota := m.GetQuota(tenantID)
	var limit int64

	switch quotaType {
	case QuotaTypeJobs:
		limit = quota.JobsPerHour
	case QuotaTypeTokens:
		limit = quota.TokensPerMin
	case QuotaTypeRequests:
		limit = quota.RequestsPerS
	case QuotaTypeConcurrent:
		limit = quota.Concurrent
	default:
		return true, 0, 0
	}

	// 简化实现：返回允许
	// 实际实现需要跟踪使用量
	return true, limit, 0
}

// QuotaExceededError Quota 超限错误
type QuotaExceededError struct {
	TenantID  string
	QuotaType QuotaType
	Limit     int64
	Used      int64
	ResetAt   time.Time
}

func (e *QuotaExceededError) Error() string {
	return fmt.Sprintf("quota exceeded for tenant %s: type=%s, limit=%d, used=%d",
		e.TenantID, e.QuotaType, e.Limit, e.Used)
}

// CheckAndConsume 检查并消费配额
func (m *QuotaManager) CheckAndConsume(ctx context.Context, tenantID string, quotaType QuotaType, amount int64) error {
	allowed, remaining, _ := m.CheckQuota(tenantID, quotaType)
	if !allowed {
		return &QuotaExceededError{
			TenantID:  tenantID,
			QuotaType: quotaType,
		}
	}
	if remaining < amount {
		return &QuotaExceededError{
			TenantID:  tenantID,
			QuotaType: quotaType,
			Used:      amount,
		}
	}
	return nil
}
