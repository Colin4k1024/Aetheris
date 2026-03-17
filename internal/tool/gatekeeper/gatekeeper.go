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
	"regexp"
	"strings"
	"sync"
	"time"

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
	// enableCommandValidation 启用命令注入检查
	enableCommandValidation bool
	// enableRateLimiting 启用速率限制
	enableRateLimiting bool
	// rateLimiter 速率限制器
	rateLimiter *RateLimiter
	// maxRequestSize 最大请求大小
	maxRequestSize int64
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

// WithCommandValidation 启用命令注入检查
func WithCommandValidation(enable bool) Option {
	return func(g *Gatekeeper) {
		g.enableCommandValidation = enable
	}
}

// WithRateLimiting 启用速率限制
func WithRateLimiting(requestsPerMinute int) Option {
	return func(g *Gatekeeper) {
		g.enableRateLimiting = true
		g.rateLimiter = NewRateLimiter(requestsPerMinute)
	}
}

// WithMaxRequestSize 设置最大请求大小
func WithMaxRequestSize(size int64) Option {
	return func(g *Gatekeeper) {
		g.maxRequestSize = size
	}
}

// RateLimiter 简单的速率限制器
type RateLimiter struct {
	mu           sync.Mutex
	requests     []time.Time
	maxRequests  int
	windowSize   time.Duration
}

// NewRateLimiter 创建速率限制器
func NewRateLimiter(requestsPerMinute int) *RateLimiter {
	return &RateLimiter{
		maxRequests: requestsPerMinute,
		windowSize:  time.Minute,
	}
}

// Allow 检查是否允许请求
func (rl *RateLimiter) Allow() bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	cutoff := now.Add(-rl.windowSize)

	// 清理过期的请求
	var validRequests []time.Time
	for _, t := range rl.requests {
		if t.After(cutoff) {
			validRequests = append(validRequests, t)
		}
	}

	if len(validRequests) >= rl.maxRequests {
		rl.requests = validRequests
		return false
	}

	rl.requests = append(validRequests, now)
	return true
}

// Reset 重置速率限制器
func (rl *RateLimiter) Reset() {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	rl.requests = nil
}

// New 创建 Gatekeeper
func New(opts ...Option) *Gatekeeper {
	g := &Gatekeeper{
		enableNetworkValidation:  true,
		enableTypeValidation:    true,
		enableCommandValidation: true,
		enableRateLimiting:      false,
		maxRequestSize:          10 * 1024 * 1024, // 10MB default
	}
	for _, opt := range opts {
		opt(g)
	}
	return g
}

// Validate 验证工具参数
func (g *Gatekeeper) Validate(toolName string, params map[string]any, schema tool.Schema) error {
	// 0. 速率限制检查
	if g.enableRateLimiting && !g.rateLimiter.Allow() {
		return &types.ValidationError{
			Field:   "rate_limit",
			Message: "rate limit exceeded, please try again later",
		}
	}

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
	case "file.delete":
		return g.validateFileDelete(params)
	case "db.query":
		return g.validateDBQuery(params)
	case "shell.execute":
		return g.validateShellExecute(params)
	case "browser.navigate":
		return g.validateBrowserNavigate(params)
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

	// SSRF 防护：检查内网 IP
	if err := g.validateSSRF(url); err != nil {
		return err
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

// validateSSRF 检查 SSRF 攻击
func (g *Gatekeeper) validateSSRF(urlStr string) error {
	host := extractHost(urlStr)

	// 检查 localhost 变体
	localhostPatterns := []string{
		"localhost", "127.0.0.1", "::1", "0.0.0.0",
		"127.0.0.2", "127.0.0.3", "127.0.0.4", "127.0.0.5",
		"127.0.0.6", "127.0.0.7", "127.0.0.8", "127.0.0.9",
	}

	for _, pattern := range localhostPatterns {
		if strings.HasPrefix(host, pattern) || host == pattern {
			return &types.ValidationError{
				Field:   "url",
				Message: "localhost access not allowed (SSRF protection)",
			}
		}
	}

	// 检查云元数据端点
	metadataEndpoints := []string{
		"169.254.169.254",  // AWS, GCP, Azure
		"metadata.google.internal", // GCP
		"metadata.google",  // GCP
	}

	for _, endpoint := range metadataEndpoints {
		if strings.HasPrefix(host, endpoint) {
			return &types.ValidationError{
				Field:   "url",
				Message: "cloud metadata endpoint access not allowed (SSRF protection)",
			}
		}
	}

	// 检查私有 IP 范围
	privateIPPatterns := []string{
		"10.", "172.16.", "172.17.", "172.18.", "172.19.",
		"172.20.", "172.21.", "172.22.", "172.23.", "172.24.",
		"172.25.", "172.26.", "172.27.", "172.28.", "172.29.",
		"172.30.", "172.31.", "192.168.",
	}

	for _, pattern := range privateIPPatterns {
		if strings.HasPrefix(host, pattern) {
			return &types.ValidationError{
				Field:   "url",
				Message: "private IP access not allowed (SSRF protection)",
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
	dangerousPaths := []string{"/etc/", "/usr/bin/", "/usr/sbin/", "C:\\Windows\\", "/proc/", "/sys/"}
	for _, dp := range dangerousPaths {
		if strings.HasPrefix(path, dp) {
			return &types.ValidationError{Field: "path", Message: "cannot write to system directory: " + dp}
		}
	}

	return nil
}

// validateFileDelete 验证文件删除参数
func (g *Gatekeeper) validateFileDelete(params map[string]any) error {
	path, ok := params["path"].(string)
	if !ok || path == "" {
		return &types.ValidationError{Field: "path", Message: "path is required and must be a string"}
	}

	// 检查路径遍历攻击
	if strings.Contains(path, "..") {
		return &types.ValidationError{Field: "path", Message: "path traversal not allowed"}
	}

	// 更严格的路径检查
	dangerousPaths := []string{"/etc/", "/usr/bin/", "/usr/sbin/", "C:\\Windows\\", "C:\\Program Files\\", "/proc/", "/sys/"}
	for _, dp := range dangerousPaths {
		if strings.HasPrefix(path, dp) {
			return &types.ValidationError{Field: "path", Message: "cannot delete system directory: " + dp}
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
	dangerous := []string{"DROP ", "DELETE ", "TRUNCATE ", "ALTER ", "CREATE ", "GRANT ", "REVOKE "}
	upperQuery := strings.ToUpper(query)
	for _, d := range dangerous {
		if strings.Contains(upperQuery, d) {
			return &types.ValidationError{Field: "query", Message: "potentially dangerous SQL command: " + d}
		}
	}

	return nil
}

// validateShellExecute 验证 shell 执行参数
func (g *Gatekeeper) validateShellExecute(params map[string]any) error {
	if !g.enableCommandValidation {
		return nil
	}

	command, ok := params["command"].(string)
	if !ok || command == "" {
		return &types.ValidationError{Field: "command", Message: "command is required and must be a string"}
	}

	// 命令注入检查
	dangerousPatterns := []string{
		";", "&&", "||", "|", "`", "$(", "${",
		"\n", "\r", "\x00",
		">>", ">",
		"<",
		"rm -rf", "del /", "format",
	}

	lowerCommand := strings.ToLower(command)
	for _, pattern := range dangerousPatterns {
		if strings.Contains(lowerCommand, strings.ToLower(pattern)) {
			return &types.ValidationError{
				Field:   "command",
				Message: fmt.Sprintf("potentially dangerous pattern in command: %s", pattern),
			}
		}
	}

	// 常见危险命令黑名单
	blacklist := []string{
		"curl", "wget", "nc", "netcat", "socat",
		"chmod", "chown", "sudo", "su",
		"ssh", "scp", "rsync",
		"nmap", "ping", "traceroute",
	}

	for _, cmd := range blacklist {
		if strings.HasPrefix(strings.TrimSpace(command), cmd+" ") || strings.TrimSpace(command) == cmd {
			return &types.ValidationError{
				Field:   "command",
				Message: fmt.Sprintf("command not allowed: %s", cmd),
			}
		}
	}

	return nil
}

// validateBrowserNavigate 验证浏览器导航参数
func (g *Gatekeeper) validateBrowserNavigate(params map[string]any) error {
	url, ok := params["url"].(string)
	if !ok || url == "" {
		return &types.ValidationError{Field: "url", Message: "url is required and must be a string"}
	}

	// 检查 javascript: 协议
	if strings.HasPrefix(strings.ToLower(url), "javascript:") {
		return &types.ValidationError{
			Field:   "url",
			Message: "javascript: protocol not allowed",
		}
	}

	// 检查 data: 协议
	if strings.HasPrefix(strings.ToLower(url), "data:") {
		return &types.ValidationError{
			Field:   "url",
			Message: "data: protocol not allowed",
		}
	}

	// SSRF 检查
	if err := g.validateSSRF(url); err != nil {
		return err
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
	parts = strings.Split(strings.TrimPrefix(urlStr, "ftp://"), "/")
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

// ValidateSchema 验证 JSON Schema (简化实现)
func ValidateSchema(data string, schema string) error {
	// 实际实现应使用 JSON Schema 验证库
	// 这里提供简化版本
	if data == "" {
		return &types.ValidationError{Field: "data", Message: "data cannot be empty"}
	}

	// 编译正则表达式模式
	pattern := regexp.MustCompile(`^\{.*\}$`)
	if !pattern.MatchString(data) {
		return &types.ValidationError{Field: "data", Message: "invalid JSON format"}
	}

	return nil
}
