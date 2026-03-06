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
	"sync"
	"time"
)

// SLOType SLO 类型
type SLOType string

const (
	SLOTypeLatency      SLOType = "latency"      // 延迟
	SLOTypeThroughput   SLOType = "throughput"   // 吞吐量
	SLOTypeAvailability SLOType = "availability" // 可用性
	SLOTypeErrorRate    SLOType = "error_rate"   // 错误率
)

// SLO Service Level Objective
type SLO struct {
	Name      string        `yaml:"name"`
	Type      SLOType       `yaml:"type"`
	Target    float64       `yaml:"target"`    // 目标值 (如 99.9)
	Window    time.Duration `yaml:"window"`    // 统计窗口
	Threshold float64       `yaml:"threshold"` // 告警阈值
}

// SLAContract SLA 合约
type SLAContract struct {
	TenantID    string        `yaml:"tenant_id"`
	SLOs        []SLO         `yaml:"slos"`
	GracePeriod time.Duration `yaml:"grace_period"` // 宽限期
	Enforcement bool          `yaml:"enforcement"`  // 是否强制执行
}

// Monitor SLA 监控器
type Monitor struct {
	mu           sync.RWMutex
	contracts    map[string]*SLAContract // key: tenantID
	measurements map[string]*SLOmeasurements
}

// SLOmeasurements SLO 测量数据
type SLOmeasurements struct {
	mu          sync.RWMutex
	latencyP50  []float64
	latencyP99  []float64
	throughput  []int64
	errors      int64
	requests    int64
	windowStart time.Time
}

// NewMonitor 创建 SLA 监控器
func NewMonitor() *Monitor {
	return &Monitor{
		contracts:    make(map[string]*SLAContract),
		measurements: make(map[string]*SLOmeasurements),
	}
}

// RegisterContract 注册 SLA 合约
func (m *Monitor) RegisterContract(contract *SLAContract) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.contracts[contract.TenantID] = contract
	m.measurements[contract.TenantID] = &SLOmeasurements{
		windowStart: time.Now(),
	}
}

// RecordLatency 记录延迟
func (m *Monitor) RecordLatency(tenantID string, p50, p99 float64) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m, ok := m.measurements[tenantID]; ok {
		m.mu.Lock()
		defer m.mu.Unlock()
		m.latencyP50 = append(m.latencyP50, p50)
		m.latencyP99 = append(m.latencyP99, p99)
	}
}

// RecordRequest 记录请求
func (m *Monitor) RecordRequest(tenantID string, success bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m, ok := m.measurements[tenantID]; ok {
		m.mu.Lock()
		defer m.mu.Unlock()
		m.requests++
		if !success {
			m.errors++
		}
	}
}

// GetStatus 获取 SLA 状态
func (m *Monitor) GetStatus(tenantID string) *SLAStatus {
	m.mu.RLock()
	defer m.mu.RUnlock()

	contract, ok := m.contracts[tenantID]
	if !ok {
		return &SLAStatus{TenantID: tenantID, Valid: false}
	}

	measurements, ok := m.measurements[tenantID]
	if !ok {
		return &SLAStatus{TenantID: tenantID, Valid: true, Met: true}
	}

	measurements.mu.RLock()
	defer measurements.mu.RUnlock()

	status := &SLAStatus{
		TenantID: tenantID,
		Valid:    true,
		Met:      true,
		SLOs:     make([]SLOStatus, len(contract.SLOs)),
	}

	for i, slo := range contract.SLOs {
		sloStatus := SLOStatus{
			Name:   slo.Name,
			Type:   slo.Type,
			Target: slo.Target,
			Actual: 100.0,
			Met:    true,
		}

		switch slo.Type {
		case SLOTypeAvailability:
			if measurements.requests > 0 {
				sloStatus.Actual = float64(measurements.requests-measurements.errors) / float64(measurements.requests) * 100
				sloStatus.Met = sloStatus.Actual >= slo.Target
			}
		case SLOTypeErrorRate:
			if measurements.requests > 0 {
				sloStatus.Actual = float64(measurements.errors) / float64(measurements.requests) * 100
				sloStatus.Met = sloStatus.Actual <= slo.Target
			}
		}

		status.SLOs[i] = sloStatus
		if !sloStatus.Met {
			status.Met = false
		}
	}

	return status
}

// SLAStatus SLA 状态
type SLAStatus struct {
	TenantID string
	Valid    bool
	Met      bool
	SLOs     []SLOStatus
}

// SLOStatus 单个 SLO 状态
type SLOStatus struct {
	Name   string
	Type   SLOType
	Target float64
	Actual float64
	Met    bool
}

// CheckSLOs 检查所有 SLO
func (m *Monitor) CheckSLOs(ctx context.Context) map[string]*SLAStatus {
	m.mu.RLock()
	defer m.mu.RUnlock()

	statuses := make(map[string]*SLAStatus)
	for tenantID := range m.contracts {
		statuses[tenantID] = m.GetStatus(tenantID)
	}
	return statuses
}
