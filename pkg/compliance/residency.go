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

package compliance

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/cloudwego/hertz/pkg/common/hlog"
)

// RegionCode 区域代码 (ISO 3166-1 alpha-2)
type RegionCode string

// DataCategory 数据分类
type DataCategory string

const (
	DataCategoryPersonal  DataCategory = "personal"  // 个人数据
	DataCategorySensitive DataCategory = "sensitive" // 敏感数据
	DataCategoryFinancial DataCategory = "financial" // 财务数据
	DataCategoryHealth    DataCategory = "health"    // 健康数据
	DataCategoryGeneral   DataCategory = "general"   // 一般数据
)

// ResidencyPolicy 数据驻留策略
type ResidencyPolicy struct {
	AllowedRegions      []RegionCode                   // 允许的数据存储区域
	BlockedRegions      []RegionCode                   // 禁止的数据存储区域
	TransferEncryption  bool                           // 跨区域传输时强制加密
	DefaultRegion       RegionCode                     // 默认存储区域
	RetentionByCategory map[DataCategory]DataRetention // 按分类的保留策略
	CreatedAt           time.Time
	UpdatedAt           time.Time
}

// DataRetention 数据保留策略
type DataRetention struct {
	RetentionDays      int  // 保留天数
	ArchiveDays        int  // 归档天数
	DeleteAfterArchive bool // 归档后删除
}

// DataResidencyController 数据驻留控制器
type DataResidencyController struct {
	mu            sync.RWMutex
	policies      map[string]*ResidencyPolicy // 按租户 ID
	defaultPolicy *ResidencyPolicy
	regionLookup  RegionLookup
}

// RegionLookup 区域查找接口
type RegionLookup interface {
	GetRegion(ctx context.Context, identifier string) (RegionCode, error)
}

// IPRegionLookup 基于 IP 的区域查找
type IPRegionLookup struct {
	// 可以集成 GeoIP 数据库
}

// GetRegion 获取 IP 对应的区域
func (r *IPRegionLookup) GetRegion(ctx context.Context, identifier string) (RegionCode, error) {
	// 简化实现：基于 IP 地址前缀判断
	// 实际应该使用 GeoIP 数据库
	if strings.HasPrefix(identifier, "10.") ||
		strings.HasPrefix(identifier, "172.16.") ||
		strings.HasPrefix(identifier, "192.168.") {
		return "XX", nil // 内部网络
	}
	return "US", nil // 默认
}

// NewDataResidencyController 创建数据驻留控制器
func NewDataResidencyController(defaultPolicy *ResidencyPolicy) *DataResidencyController {
	return &DataResidencyController{
		policies:      make(map[string]*ResidencyPolicy),
		defaultPolicy: defaultPolicy,
		regionLookup:  &IPRegionLookup{},
	}
}

// SetPolicy 设置租户策略
func (c *DataResidencyController) SetPolicy(tenantID string, policy *ResidencyPolicy) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.policies[tenantID] = policy
}

// GetPolicy 获取租户策略
func (c *DataResidencyController) GetPolicy(tenantID string) *ResidencyPolicy {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if policy, ok := c.policies[tenantID]; ok {
		return policy
	}
	return c.defaultPolicy
}

// CheckStorageCompliance 检查数据存储合规性
func (c *DataResidencyController) CheckStorageCompliance(ctx context.Context, tenantID string, region RegionCode, category DataCategory) error {
	policy := c.GetPolicy(tenantID)
	if policy == nil {
		return nil // 没有策略，允许
	}

	// 检查是否在允许列表中
	if len(policy.AllowedRegions) > 0 {
		allowed := false
		for _, r := range policy.AllowedRegions {
			if r == region {
				allowed = true
				break
			}
		}
		if !allowed {
			return fmt.Errorf("region %s is not allowed for tenant %s", region, tenantID)
		}
	}

	// 检查是否在禁止列表中
	for _, r := range policy.BlockedRegions {
		if r == region {
			return fmt.Errorf("region %s is blocked for tenant %s", region, tenantID)
		}
	}

	return nil
}

// CheckTransferCompliance 检查数据传输合规性
func (c *DataResidencyController) CheckTransferCompliance(ctx context.Context, tenantID string, sourceRegion, destRegion RegionCode) error {
	policy := c.GetPolicy(tenantID)
	if policy == nil {
		return nil
	}

	// 检查目标区域是否允许
	if len(policy.AllowedRegions) > 0 {
		allowed := false
		for _, r := range policy.AllowedRegions {
			if r == destRegion {
				allowed = true
				break
			}
		}
		if !allowed {
			return fmt.Errorf("transfer to region %s is not allowed for tenant %s", destRegion, tenantID)
		}
	}

	// 跨区域传输是否需要加密
	if sourceRegion != destRegion && policy.TransferEncryption {
		hlog.CtxWarnf(ctx, "cross-region transfer from %s to %s requires encryption", sourceRegion, destRegion)
		// 这里可以返回错误或警告
	}

	return nil
}

// GetDataRetention 获取数据保留策略
func (c *DataResidencyController) GetDataRetention(tenantID string, category DataCategory) *DataRetention {
	policy := c.GetPolicy(tenantID)
	if policy == nil {
		return nil
	}

	if retention, ok := policy.RetentionByCategory[category]; ok {
		return &retention
	}

	return nil
}

// GetDefaultRegion 获取默认区域
func (c *DataResidencyController) GetDefaultRegion(tenantID string) RegionCode {
	policy := c.GetPolicy(tenantID)
	if policy == nil || policy.DefaultRegion == "" {
		return "US"
	}
	return policy.DefaultRegion
}

// ValidateStorageRequest 验证存储请求
type ValidateStorageRequest struct {
	TenantID   string
	Region     RegionCode
	Category   DataCategory
	DataSize   int64
	Encryption bool
}

// ValidateStorage 验证存储请求是否合规
func (c *DataResidencyController) ValidateStorage(ctx context.Context, req *ValidateStorageRequest) error {
	// 检查区域合规
	if err := c.CheckStorageCompliance(ctx, req.TenantID, req.Region, req.Category); err != nil {
		return err
	}

	// 检查传输加密
	policy := c.GetPolicy(req.TenantID)
	if policy != nil && policy.TransferEncryption && !req.Encryption {
		return fmt.Errorf("encryption is required for data storage")
	}

	return nil
}

// ComplianceConfig 数据驻留配置
type ComplianceConfig struct {
	Enabled            bool
	DefaultRegion      string
	AllowedRegions     []string
	TransferEncryption bool
}

// NewResidencyPolicyFromConfig 从配置创建策略
func NewResidencyPolicyFromConfig(cfg ComplianceConfig) *ResidencyPolicy {
	var allowedRegions []RegionCode
	for _, r := range cfg.AllowedRegions {
		allowedRegions = append(allowedRegions, RegionCode(r))
	}

	return &ResidencyPolicy{
		AllowedRegions:     allowedRegions,
		TransferEncryption: cfg.TransferEncryption,
		DefaultRegion:      RegionCode(cfg.DefaultRegion),
		RetentionByCategory: map[DataCategory]DataRetention{
			DataCategoryPersonal:  {RetentionDays: 90, ArchiveDays: 30},
			DataCategorySensitive: {RetentionDays: 365, ArchiveDays: 90},
			DataCategoryHealth:    {RetentionDays: 1825, ArchiveDays: 365}, // HIPAA: 5 years
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
}

// RegionCodeList 区域代码列表
var RegionCodes = map[string]RegionCode{
	"US": "US", // United States
	"EU": "EU", // European Union
	"DE": "DE", // Germany
	"FR": "FR", // France
	"UK": "UK", // United Kingdom
	"JP": "JP", // Japan
	"CN": "CN", // China
	"SG": "SG", // Singapore
	"AU": "AU", // Australia
	"CA": "CA", // Canada
}

// GetRegionName 获取区域名称
func (r RegionCode) GetRegionName() string {
	names := map[RegionCode]string{
		"US": "United States",
		"EU": "European Union",
		"DE": "Germany",
		"FR": "France",
		"UK": "United Kingdom",
		"JP": "Japan",
		"CN": "China",
		"SG": "Singapore",
		"AU": "Australia",
		"CA": "Canada",
	}
	if name, ok := names[r]; ok {
		return name
	}
	return string(r)
}

// IsEURegion 检查是否是欧盟区域
func (r RegionCode) IsEURegion() bool {
	return r == "EU" || r == "DE" || r == "FR" || r == "UK"
}
