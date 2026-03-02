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

import "fmt"

// Region 区域配置
type Region struct {
	ID        string `yaml:"id"`         // 区域 ID，如 "us-east-1"
	Name      string `yaml:"name"`       // 区域名称
	Endpoint  string `yaml:"endpoint"`   // API 端点
	Priority  int    `yaml:"priority"`   // 优先级（越小越高）
	IsPrimary bool   `yaml:"is_primary"` // 是否为主区域
}

// Config 多区域配置
type Config struct {
	// CurrentRegion 当前区域 ID
	CurrentRegion string `yaml:"current_region"`

	// Regions 所有可用区域
	Regions []Region `yaml:"regions"`

	// EnableCrossRegionReplication 是否启用跨区域复制
	EnableCrossRegionReplication bool `yaml:"enable_cross_region_replication"`

	// ReplicationMode 复制模式: sync | async
	ReplicationMode string `yaml:"replication_mode"`
}

// GetCurrentRegion 获取当前区域配置
func (c *Config) GetCurrentRegion() (*Region, error) {
	for _, r := range c.Regions {
		if r.ID == c.CurrentRegion {
			return &r, nil
		}
	}
	return nil, fmt.Errorf("region not found: %s", c.CurrentRegion)
}

// GetPrimaryRegion 获取主区域
func (c *Config) GetPrimaryRegion() *Region {
	for _, r := range c.Regions {
		if r.IsPrimary {
			return &r
		}
	}
	return nil
}

// IsLocalRegion 检查目标区域是否为本地区域
func (c *Config) IsLocalRegion(regionID string) bool {
	return regionID == c.CurrentRegion
}

// GetRegionByID 根据 ID 获取区域
func (c *Config) GetRegionByID(id string) *Region {
	for i := range c.Regions {
		if c.Regions[i].ID == id {
			return &c.Regions[i]
		}
	}
	return nil
}
