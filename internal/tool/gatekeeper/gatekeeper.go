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

package gatekeeper

import (
	"fmt"
	"reflect"
	"strings"

	"rag-platform/internal/tool"
	"rag-platform/internal/tool/types"
)

// Gatekeeper 本地护栏：参数验证、网络安全检查
type Gatekeeper struct {
	// allowedHosts 允许访问的域名/IP 白名单
	allowedHosts []string
	// blockedPatterns 禁止访问的 URL 模式
	blockedPatterns []string
	// enableNetworkValidation 启用网络请求验证
	enableNetworkValidation bool
	// enableTypeValidation 启用类型验证
	enableTypeValidation bool
}

// Option Gatekeeper 配置选项
type Option func(*Gatekeeper)

// WithAllowedHosts 设置允许访问的域名/IP 白名单
func WithAllowedHosts(hosts []string) Option {
	return func(g *Gatekeeper) {
		g.allowedHosts = hosts
	}
}

// WithBlockedPatterns 设置禁止访问的 URL 模式
func WithBlockedPatterns(patterns []string) Option {
	return func(g *Gatekeeper) {
		g.blockedPatterns = patterns
	}
}

// WithNetworkValidation 启用网络请求验证
func WithNetworkValidation(enable bool) Option {
	return func(g *Gatekeeper) {
		g.enableNetworkValidation = enable
	}
}

// WithTypeValidation 启用类型验证
func WithTypeValidation(enable bool) Option {
	return func(g *Gatekeeper) {
		g.enableTypeValidation = enable
	}
}

// New 创建 Gatekeeper
func New(opts ...Option) *Gatekeeper {
	g := &Gatekeeper{
		enableNetworkValidation: true,
		enableTypeValidation:    true,
	}
	for _, opt := range opts {
		opt(g)
	}
	return g
}

// Validate 验证工具参数
func (g *Gatekeeper) Validate(toolName string, params map[string]any, schema tool.Schema) error {
	// 1. 检查必需参数
	if err := g.validateRequired(schema.Required, params); err != nil {
		return err
	}

	// 2. 类型验证
	if g.enableTypeValidation {
		if err := g.validateTypes(schema.Properties, params); err != nil {
			return err
		}
	}

	// 3. 工具特定的验证
	if err := g.validateToolSpecific(toolName, params); err != nil {
		return err
	}

	return nil
}

// validateRequired 检查必需参数
func (g *Gatekeeper) validateRequired(required []string, params map[string]any) error {
	for _, field := range required {
		if _, ok := params[field]; !ok {
			return &types.MissingParameterError{Parameter: field}
		}
	}
	return nil
}

// validateTypes 验证参数类型
func (g *Gatekeeper) validateTypes(properties map[string]tool.SchemaProperty, params map[string]any) error {
	for name, value := range params {
		prop, ok := properties[name]
		if !ok {
			// 未知参数，给出警告但不阻止
			continue
		}

		if err := g.validateType(name, value, prop.Type); err != nil {
			return err
		}
	}
	return nil
}

// validateType 验证单个参数类型
func (g *Gatekeeper) validateType(name string, value any, expectedType string) error {
	if value == nil {
		return nil // 忽略 nil 值（已由 required 检查处理）
	}

	actualType := reflect.TypeOf(value).Kind()
	var expected reflect.Kind

	switch expectedType {
	case "string":
		expected = reflect.String
	case "number", "integer":
		expected = reflect.Float64
	case "boolean":
		expected = reflect.Bool
	case "array":
		expected = reflect.Slice
	case "object":
		expected = reflect.Map
	default:
		// 未知类型，跳过验证
		return nil
	}

	if actualType != expected {
		return &types.InvalidTypeError{
			Parameter: name,
			Expected:  expectedType,
			Actual:    actualType.String(),
		}
	}
	return nil
}

// validateToolSpecific 工具特定的验证
func (g *Gatekeeper) validateToolSpecific(toolName string, params map[string]any) error {
	switch toolName {
	case "http.request":
		return g.validateHTTPRequest(params)
	case "file.read":
		return g.validateFileRead(params)
	case "file.write":
		return g.validateFileWrite(params)
	case "db.query":
		return g.validateDBQuery(params)
	}
	return nil
}

// validateHTTPRequest 验证 HTTP 请求参数
func (g *Gatekeeper) validateHTTPRequest(params map[string]any) error {
	if !g.enableNetworkValidation {
		return nil
	}

	url, ok := params["url"].(string)
	if !ok || url == "" {
		return &types.ValidationError{Field: "url", Message: "url is required and must be a string"}
	}

	// 检查 URL 是否在白名单中
	if len(g.allowedHosts) > 0 {
		if !g.isHostAllowed(url) {
			return &types.ValidationError{
				Field:   "url",
				Message: fmt.Sprintf("host not allowed: %s (allowed: %v)", extractHost(url), g.allowedHosts),
			}
		}
	}

	// 检查是否匹配黑名单模式
	for _, pattern := range g.blockedPatterns {
		if matchesPattern(url, pattern) {
			return &types.ValidationError{
				Field:   "url",
				Message: fmt.Sprintf("url matches blocked pattern: %s", pattern),
			}
		}
	}

	return nil
}

// validateFileRead 验证文件读取参数
func (g *Gatekeeper) validateFileRead(params map[string]any) error {
	path, ok := params["path"].(string)
	if !ok || path == "" {
		return &types.ValidationError{Field: "path", Message: "path is required and must be a string"}
	}

	// 检查路径遍历攻击
	if strings.Contains(path, "..") {
		return &types.ValidationError{Field: "path", Message: "path traversal not allowed"}
	}

	return nil
}

// validateFileWrite 验证文件写入参数
func (g *Gatekeeper) validateFileWrite(params map[string]any) error {
	path, ok := params["path"].(string)
	if !ok || path == "" {
		return &types.ValidationError{Field: "path", Message: "path is required and must be a string"}
	}

	// 检查路径遍历攻击
	if strings.Contains(path, "..") {
		return &types.ValidationError{Field: "path", Message: "path traversal not allowed"}
	}

	// 检查危险路径
	dangerousPaths := []string{"/etc/", "/usr/bin/", "/usr/sbin/", "C:\\Windows\\"}
	for _, dp := range dangerousPaths {
		if strings.HasPrefix(path, dp) {
			return &types.ValidationError{Field: "path", Message: "cannot write to system directory: " + dp}
		}
	}

	return nil
}

// validateDBQuery 验证数据库查询参数
func (g *Gatekeeper) validateDBQuery(params map[string]any) error {
	query, ok := params["query"].(string)
	if !ok || query == "" {
		return &types.ValidationError{Field: "query", Message: "query is required and must be a string"}
	}

	// 简单的 SQL 注入检查（实际应使用参数化查询）
	dangerous := []string{"DROP ", "DELETE ", "TRUNCATE ", "ALTER ", "CREATE "}
	upperQuery := strings.ToUpper(query)
	for _, d := range dangerous {
		if strings.Contains(upperQuery, d) {
			return &types.ValidationError{Field: "query", Message: "potentially dangerous SQL command: " + d}
		}
	}

	return nil
}

// isHostAllowed 检查主机是否在白名单中
func (g *Gatekeeper) isHostAllowed(urlStr string) bool {
	host := extractHost(urlStr)
	for _, allowed := range g.allowedHosts {
		if host == allowed || strings.HasSuffix(host, "."+allowed) {
			return true
		}
	}
	return false
}

// extractHost 从 URL 中提取主机
func extractHost(urlStr string) string {
	// 简单的实现，实际应使用 url.Parse
	parts := strings.Split(strings.TrimPrefix(urlStr, "http://"), "/")
	if len(parts) > 0 {
		return parts[0]
	}
	parts = strings.Split(strings.TrimPrefix(urlStr, "https://"), "/")
	if len(parts) > 0 {
		return parts[0]
	}
	return ""
}

// matchesPattern 检查 URL 是否匹配模式
func matchesPattern(url, pattern string) bool {
	// 支持通配符 *
	if strings.HasPrefix(pattern, "*.") {
		suffix := strings.TrimPrefix(pattern, "*.")
		return strings.HasSuffix(url, suffix) || strings.Contains(url, "/"+suffix)
	}
	return strings.Contains(url, pattern)
}
