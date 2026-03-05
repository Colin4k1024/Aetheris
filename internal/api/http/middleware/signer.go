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

package middleware

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/common/hlog"
	"github.com/cloudwego/hertz/pkg/protocol/consts"
	"rag-platform/pkg/security/signer"
)

// SignerMiddleware API 请求签名验证中间件
type SignerMiddleware struct {
	signer        *signer.Signer
	clockSkew     time.Duration
	requiredPaths []string // 需要签名验证的路径模式
	excludePaths  []string // 排除签名验证的路径
}

// NewSignerMiddleware 创建签名中间件
// secretKey: 用于签名的密钥
// requiredPaths: 需要验证签名的路径模式（如 "/api/admin/*", "/api/jobs/*"）
func NewSignerMiddleware(secretKey string, requiredPaths []string) (*SignerMiddleware, error) {
	if secretKey == "" {
		return nil, nil // 未配置时返回 nil，禁用中间件
	}

	s := signer.New(secretKey)
	return &SignerMiddleware{
		signer:        s,
		clockSkew:     5 * time.Minute,
		requiredPaths: requiredPaths,
		excludePaths:  []string{"/api/login", "/metrics", "/health"},
	}, nil
}

// NewSignerMiddlewareWithConfig 从配置创建签名中间件
func NewSignerMiddlewareWithConfig(secretKey string, clockSkew time.Duration, requiredPaths []string) (*SignerMiddleware, error) {
	if secretKey == "" {
		return nil, nil
	}

	s := signer.New(secretKey, signer.WithClockSkew(clockSkew))
	return &SignerMiddleware{
		signer:        s,
		clockSkew:     clockSkew,
		requiredPaths: requiredPaths,
		excludePaths:  []string{"/api/login", "/metrics", "/health"},
	}, nil
}

// Middleware 返回 Hertz 中间件
func (m *SignerMiddleware) Middleware() app.HandlerFunc {
	if m == nil {
		return func(c context.Context, ctx *app.RequestContext) {
			ctx.Next(c)
		}
	}

	return func(c context.Context, ctx *app.RequestContext) {
		// 检查是否需要验证签名
		if !m.requiresSignature(ctx) {
			ctx.Next(c)
			return
		}

		// 获取签名相关 Header
		signature := string(ctx.Request.Header.Peek(signer.HeaderSignature))
		timestamp := string(ctx.Request.Header.Peek(signer.HeaderTimestamp))
		accessKey := string(ctx.Request.Header.Peek(signer.HeaderAccessKey))

		// 检查必要的 Header
		if signature == "" || timestamp == "" || accessKey == "" {
			hlog.CtxWarnf(c, "missing signature headers: signature=%q, timestamp=%q, accessKey=%q",
				signature, timestamp, accessKey)
			ctx.JSON(consts.StatusUnauthorized, map[string]interface{}{
				"code":    consts.StatusUnauthorized,
				"message": "missing required signature headers",
			})
			ctx.Abort()
			return
		}

		// 获取请求体
		body := string(ctx.Request.Body())

		// 构建 Header 映射（用于签名）
		headers := make(map[string]string)
		ctx.Request.Header.VisitAll(func(key, value []byte) {
			headers[strings.ToLower(string(key))] = string(value)
		})

		// 验证签名
		err := m.signer.VerifySignatureWithHeaders(
			string(ctx.Method()),
			string(ctx.Path()),
			timestamp,
			body,
			headers,
			signature,
		)

		if err != nil {
			hlog.CtxWarnf(c, "signature verification failed: %v, path=%s, accessKey=%s",
				err, ctx.Path(), accessKey)
			ctx.JSON(consts.StatusUnauthorized, map[string]interface{}{
				"code":    consts.StatusUnauthorized,
				"message": "invalid signature: " + err.Error(),
			})
			ctx.Abort()
			return
		}

		// 签名验证通过，将 accessKey 注入 context
		ctx.Set("access_key", accessKey)

		ctx.Next(c)
	}
}

// requiresSignature 检查请求是否需要签名验证
func (m *SignerMiddleware) requiresSignature(ctx *app.RequestContext) bool {
	requestPath := string(ctx.Path())

	// 检查排除路径
	for _, exclude := range m.excludePaths {
		if strings.HasPrefix(requestPath, exclude) {
			return false
		}
	}

	// 检查是否在必需路径中
	for _, required := range m.requiredPaths {
		if matchPath(requestPath, required) {
			return true
		}
	}

	return false
}

// matchPath 检查请求路径是否匹配模式
// 支持通配符：/api/admin/* 匹配 /api/admin/users, /api/admin/settings 等
func matchPath(requestPath, pattern string) bool {
	if strings.HasSuffix(pattern, "*") {
		prefix := strings.TrimSuffix(pattern, "*")
		return strings.HasPrefix(requestPath, prefix)
	}
	return requestPath == pattern
}

// SignRequest 签名请求（供客户端使用）
func SignRequest(secretKey, method, path, body string) (signature, timestamp, accessKey string) {
	s := signer.New(secretKey)
	timestamp = s.GenerateTimestamp()
	signature = s.SignRequest(method, path, timestamp, body)
	accessKey = "default" // 可配置的 access key
	return
}

// SignRequestWithAccessKey 使用指定的 access key 签名请求
func SignRequestWithAccessKey(secretKey, accessKey, method, path, body string) (signature, timestamp string) {
	s := signer.New(secretKey)
	timestamp = s.GenerateTimestamp()
	signature = s.SignRequest(method, path, timestamp, body)
	return
}

// SignerConfig 签名中间件配置
type SignerConfig struct {
	Enabled       bool
	SecretKey     string
	ClockSkew     string
	RequiredPaths []string
}

// NewSignerFromConfig 从配置创建签名中间件
func NewSignerFromConfig(cfg SignerConfig) (*SignerMiddleware, error) {
	if !cfg.Enabled || cfg.SecretKey == "" {
		return nil, nil
	}

	clockSkew := 5 * time.Minute
	if cfg.ClockSkew != "" {
		d, err := time.ParseDuration(cfg.ClockSkew)
		if err == nil {
			clockSkew = d
		}
	}

	return NewSignerMiddlewareWithConfig(cfg.SecretKey, clockSkew, cfg.RequiredPaths)
}

// SignerClient 签名客户端（用于测试或内部服务调用）
type SignerClient struct {
	signer    *signer.Signer
	accessKey string
}

// NewSignerClient 创建签名客户端
func NewSignerClient(secretKey, accessKey string) *SignerClient {
	return &SignerClient{
		signer:    signer.New(secretKey),
		accessKey: accessKey,
	}
}

// Sign 对请求进行签名，返回签名所需的 Header
func (c *SignerClient) Sign(method, path, body string) map[string]string {
	timestamp := c.signer.GenerateTimestamp()
	signature := c.signer.SignRequest(method, path, timestamp, body)

	return map[string]string{
		"X-Access-Key": c.accessKey,
		"X-Timestamp":  timestamp,
		"X-Signature":  signature,
	}
}

// SignWithHeaders 对请求进行签名（包含自定义 Header）
func (c *SignerClient) SignWithHeaders(method, path, body string, customHeaders map[string]string) map[string]string {
	headers := make(map[string]string)
	for k, v := range customHeaders {
		headers[strings.ToLower(k)] = v
	}

	timestamp := c.signer.GenerateTimestamp()
	signature := c.signer.SignRequestWithHeaders(method, path, timestamp, body, headers)

	headers["x-access-key"] = c.accessKey
	headers["x-timestamp"] = timestamp
	headers["x-signature"] = signature

	return headers
}

// GetHTTPClient 返回带有签名头的 HTTP 客户端
// 这是一个简单的包装，实际使用中可能需要更复杂的实现
func (c *SignerClient) GetHTTPClient() *http.Client {
	return &http.Client{}
}
