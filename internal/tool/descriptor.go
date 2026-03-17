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

package tool

// ParameterConstraint 参数约束
type ParameterConstraint struct {
	// Type 参数类型（string, number, integer, boolean, array, object）
	Type string `json:"type"`
	// Description 参数描述
	Description string `json:"description,omitempty"`
	// Enum 允许的枚举值
	Enum []any `json:"enum,omitempty"`
	// MinLength 字符串最小长度
	MinLength *int `json:"minLength,omitempty"`
	// MaxLength 字符串最大长度
	MaxLength *int `json:"maxLength,omitempty"`
	// Minimum 数值最小值
	Minimum *float64 `json:"minimum,omitempty"`
	// Maximum 数值最大值
	Maximum *float64 `json:"maximum,omitempty"`
	// Pattern 正则表达式模式
	Pattern string `json:"pattern,omitempty"`
	// Items 数组元素类型
	Items *ParameterConstraint `json:"items,omitempty"`
	// Properties 嵌套对象属性
	Properties map[string]ParameterConstraint `json:"properties,omitempty"`
	// Required 必需的属性
	Required []string `json:"required,omitempty"`
	// Default 默认值
	Default any `json:"default,omitempty"`
	// Deprecated 是否已废弃
	Deprecated bool `json:"deprecated,omitempty"`
}

// ToolDescriptor 工具描述符（强类型）
type ToolDescriptor struct {
	// Name 工具名称（唯一标识符）
	Name string `json:"name"`
	// Version 工具版本
	Version string `json:"version"`
	// Description 工具描述
	Description string `json:"description"`
	// Category 工具分类
	Category string `json:"category,omitempty"`
	// Tags 工具标签
	Tags []string `json:"tags,omitempty"`
	// Parameters 参数定义
	Parameters ParameterConstraint `json:"parameters"`
	// Output 输出定义
	Output *ParameterConstraint `json:"output,omitempty"`
	// Examples 使用示例
	Examples []ToolExample `json:"examples,omitempty"`
	// Security 安全相关配置
	Security SecurityConfig `json:"security,omitempty"`
	// RateLimit 速率限制
	RateLimit *RateLimitConfig `json:"rateLimit,omitempty"`
}

// ToolExample 工具使用示例
type ToolExample struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Input       map[string]any `json:"input"`
	Output      string         `json:"output,omitempty"`
}

// SecurityConfig 安全配置
type SecurityConfig struct {
	// RequireAuth 是否需要认证
	RequireAuth bool `json:"requireAuth"`
	// AllowedHosts 允许访问的主机（用于 HTTP 工具）
	AllowedHosts []string `json:"allowedHosts,omitempty"`
	// BlockedHosts 禁止访问的主机
	BlockedHosts []string `json:"blockedHosts,omitempty"`
	// MaxRequestSize 最大请求大小（字节）
	MaxRequestSize int64 `json:"maxRequestSize,omitempty"`
	// Timeout 超时时间（毫秒）
	Timeout int64 `json:"timeout,omitempty"`
}

// RateLimitConfig 速率限制配置
type RateLimitConfig struct {
	// RequestsPerMinute 每分钟请求数
	RequestsPerMinute int `json:"requestsPerMinute"`
	// Burst 突发请求数
	Burst int `json:"burst,omitempty"`
}

// Validate 验证工具描述符的完整性
func (d *ToolDescriptor) Validate() []ValidationError {
	var errs []ValidationError

	if d.Name == "" {
		errs = append(errs, ValidationError{Field: "name", Message: "name is required"})
	}
	if d.Description == "" {
		errs = append(errs, ValidationError{Field: "description", Message: "description is required"})
	}
	if d.Parameters.Type == "" {
		errs = append(errs, ValidationError{Field: "parameters.type", Message: "parameters.type is required"})
	}

	// 验证参数约束
	if err := validateConstraint("", d.Parameters); err != nil {
		errs = append(errs, err...)
	}

	return errs
}

// validateConstraint 验证参数约束
func validateConstraint(path string, c ParameterConstraint) []ValidationError {
	var errs []ValidationError

	validTypes := map[string]bool{
		"string":  true,
		"number":  true,
		"integer": true,
		"boolean": true,
		"array":   true,
		"object":  true,
	}

	if !validTypes[c.Type] {
		errs = append(errs, ValidationError{
			Field:   path + ".type",
			Message: "invalid type: " + c.Type,
		})
	}

	if c.MinLength != nil && c.MaxLength != nil && *c.MinLength > *c.MaxLength {
		errs = append(errs, ValidationError{
			Field:   path + ".minLength/maxLength",
			Message: "minLength cannot be greater than maxLength",
		})
	}

	if c.Minimum != nil && c.Maximum != nil && *c.Minimum > *c.Maximum {
		errs = append(errs, ValidationError{
			Field:   path + ".minimum/maximum",
			Message: "minimum cannot be greater than maximum",
		})
	}

	// 递归验证嵌套类型
	if c.Items != nil {
		if err := validateConstraint(path+".items", *c.Items); err != nil {
			errs = append(errs, err...)
		}
	}

	for name, prop := range c.Properties {
		if err := validateConstraint(path+".properties."+name, prop); err != nil {
			errs = append(errs, err...)
		}
	}

	return errs
}

// ValidationError 验证错误
type ValidationError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

func (e ValidationError) Error() string {
	return e.Field + ": " + e.Message
}

// ToSchema 转换为工具接口使用的 Schema
func (d *ToolDescriptor) ToSchema() Schema {
	schema := Schema{
		Type:        d.Parameters.Type,
		Description: d.Description,
		Properties:  make(map[string]SchemaProperty),
		Required:    d.Parameters.Required,
	}

	for name, prop := range d.Parameters.Properties {
		schema.Properties[name] = SchemaProperty{
			Type:        prop.Type,
			Description: prop.Description,
		}
	}

	return schema
}
