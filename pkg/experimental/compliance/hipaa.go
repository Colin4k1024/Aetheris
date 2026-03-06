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

package compliance

import (
	"context"
	"fmt"
	"time"
)

// HIPAAConfig HIPAA 合规配置
type HIPAAConfig struct {
	EnablePHIEncryption        bool          // 加密 PHI 数据
	EnableAuditLog             bool          // 启用审计日志
	EnableAccessControl        bool          // 启用访问控制
	EnableDataMinimization     bool          // 数据最小化
	PHIRetentionPeriod         time.Duration // PHI 保留期限
	RequireEncryptionAtRest    bool          // 静态数据加密
	RequireEncryptionInTransit bool          // 传输中加密
}

// HIPAACompliance HIPAA 合规检查器
type HIPAACompliance struct {
	config *HIPAAConfig
}

// NewHIPAACompliance 创建 HIPAA 合规检查器
func NewHIPAACompliance(cfg *HIPAAConfig) *HIPAACompliance {
	if cfg == nil {
		cfg = &HIPAAConfig{
			EnablePHIEncryption:    true,
			EnableAuditLog:         true,
			EnableAccessControl:    true,
			EnableDataMinimization: true,
			PHIRetentionPeriod:     6 * 30 * 24 * time.Hour, // 6 years as required
		}
	}
	return &HIPAACompliance{config: cfg}
}

// PHIData PHI 数据
type PHIData struct {
	Type      string    `json:"type"`
	Value     string    `json:"value"`
	Encrypted bool      `json:"encrypted"`
	CreatedAt time.Time `json:"created_at"`
	ExpiredAt time.Time `json:"expired_at"`
}

// ValidatePHIData 验证 PHI 数据是否符合 HIPAA 要求
func (h *HIPAACompliance) ValidatePHIData(ctx context.Context, phi PHIData) error {
	// 检查加密
	if h.config.EnablePHIEncryption && !phi.Encrypted {
		return fmt.Errorf("PHI data must be encrypted")
	}

	// 检查保留期限
	if !phi.ExpiredAt.IsZero() && time.Now().After(phi.ExpiredAt) {
		return fmt.Errorf("PHI data has exceeded retention period")
	}

	// 检查最小化原则
	if len(phi.Value) > 10000 {
		return fmt.Errorf("PHI data exceeds minimum necessary principle")
	}

	return nil
}

// CheckEncryptionAtRest 检查静态数据加密
func (h *HIPAACompliance) CheckEncryptionAtRest(ctx context.Context) error {
	if h.config.RequireEncryptionAtRest {
		// TODO: 检查数据库存储加密状态
	}
	return nil
}

// CheckEncryptionInTransit 检查传输中加密
func (h *HIPAACompliance) CheckEncryptionInTransit(ctx context.Context) error {
	if h.config.RequireEncryptionInTransit {
		// TODO: 检查 TLS 配置
	}
	return nil
}

// HIPAA controls mapping
var HIPAAControls = map[string]string{
	"164.308(a)(1)(i)": "Security Management Process",
	"164.308(a)(5)":    "Security Awareness and Training",
	"164.310(a)(1)":    "Facility Access Controls",
	"164.312(a)(1)":    "Access Control",
	"164.312(b)":       "Audit Controls",
	"164.312(c)(1)":    "Integrity Controls",
	"164.312(e)(1)":    "Transmission Security",
}
