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
	"os"
	"strings"
	"sync"
	"time"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/common/hlog"
	"github.com/cloudwego/hertz/pkg/protocol/consts"
	"github.com/hertz-contrib/jwt"

	"github.com/Colin4k1024/Aetheris/v2/pkg/auth"
)

// Middleware 中间件管理器
type Middleware struct {
	// CORS 允许的来源列表
	AllowOrigins []string
}

// NewMiddleware 创建新的中间件管理器
func NewMiddleware() *Middleware {
	return &Middleware{}
}

// NewMiddlewareWithCORS 创建带有 CORS 配置的中间件管理器
func NewMiddlewareWithCORS(allowOrigins []string) *Middleware {
	return &Middleware{
		AllowOrigins: allowOrigins,
	}
}

// CORS CORS 中间件
func (m *Middleware) CORS() app.HandlerFunc {
	return func(ctx context.Context, c *app.RequestContext) {
		// 如果配置了允许的来源列表，则使用配置的来源
		// 否则默认允许所有（开发环境兼容）
		allowOrigin := "*"
		if len(m.AllowOrigins) > 0 {
			// 检查请求来源是否在允许列表中
			requestOrigin := string(c.Request.Header.Peek("Origin"))
			if requestOrigin != "" {
				for _, origin := range m.AllowOrigins {
					if origin == "*" || origin == requestOrigin {
						allowOrigin = requestOrigin
						break
					}
				}
			}
		}

		c.Header("Access-Control-Allow-Origin", allowOrigin)
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Origin, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization")
		c.Header("Access-Control-Expose-Headers", "Content-Length")
		c.Header("Access-Control-Allow-Credentials", "true")
		c.Header("Access-Control-Max-Age", "86400")

		if string(c.Method()) == "OPTIONS" {
			c.AbortWithStatus(consts.StatusNoContent)
			return
		}

		c.Next(ctx)
	}
}

// Auth 认证中间件（未启用 JWT 时跳过认证）
func (m *Middleware) Auth() app.HandlerFunc {
	return func(ctx context.Context, c *app.RequestContext) {
		c.Next(ctx)
	}
}

// JWTAuth 持有 JWT 中间件，用于 Login 与保护路由
type JWTAuth struct {
	Middleware *jwt.HertzJWTMiddleware
}

// RoleProvider 角色查询接口（用于 JWT 登录时获取用户角色）
type RoleProvider interface {
	GetUserRoles(ctx context.Context, tenantID, userID string) ([]string, error)
}

// LoginHandler 返回登录接口 Handler
func (j *JWTAuth) LoginHandler() app.HandlerFunc {
	return j.Middleware.LoginHandler
}

// MiddlewareFunc 返回 JWT 校验中间件
func (j *JWTAuth) MiddlewareFunc() app.HandlerFunc {
	return j.Middleware.MiddlewareFunc()
}

// NewJWTAuth 创建 JWT 认证（key 签名密钥）
// roleProvider 用于在登录时查询用户角色；若为 nil 则使用 ADMIN_ROLE 环境变量指定角色
func NewJWTAuth(key []byte, timeout, maxRefresh time.Duration, roleProvider RoleProvider) (*JWTAuth, error) {
	// SECURITY: Empty key allows trivial JWT forgery - reject early
	if len(key) == 0 {
		return nil, jwt.ErrMissingSecretKey
	}
	identityKey := "id"
	// 从环境变量读取凭证
	adminUsername := os.Getenv("ADMIN_USERNAME")
	adminPassword := os.Getenv("ADMIN_PASSWORD")
	adminRole := os.Getenv("ADMIN_ROLE") // 可选，默认 "admin"
	if adminRole == "" {
		adminRole = "admin"
	}
	authMiddleware, err := jwt.New(&jwt.HertzJWTMiddleware{
		Realm:       "rag-api",
		Key:         key,
		Timeout:     timeout,
		MaxRefresh:  maxRefresh,
		IdentityKey: identityKey,
		PayloadFunc: func(data interface{}) jwt.MapClaims {
			if u, ok := data.(*AuthUser); ok {
				claims := jwt.MapClaims{
					identityKey: u.Username,
					"tenant_id": u.TenantID,
					"user_id":   u.UserID,
					"roles":     u.Roles,
				}
				if claims["tenant_id"] == "" {
					claims["tenant_id"] = "default"
				}
				if claims["user_id"] == "" {
					claims["user_id"] = u.Username
				}
				if claims["roles"] == nil {
					claims["roles"] = []string{}
				}
				return claims
			}
			return jwt.MapClaims{}
		},
		IdentityHandler: func(ctx context.Context, c *app.RequestContext) interface{} {
			claims := jwt.ExtractClaims(ctx, c)
			u := &AuthUser{Username: getClaimString(claims, identityKey)}
			u.TenantID = getClaimString(claims, "tenant_id")
			u.UserID = getClaimString(claims, "user_id")
			u.Roles = getClaimRoles(claims, "roles")
			if u.TenantID == "" {
				u.TenantID = "default"
			}
			if u.UserID == "" {
				u.UserID = u.Username
			}
			return u
		},
		Authenticator: func(ctx context.Context, c *app.RequestContext) (interface{}, error) {
			var loginVals struct {
				Username string `form:"username" json:"username"`
				Password string `form:"password" json:"password"`
			}
			if err := c.Bind(&loginVals); err != nil {
				return nil, jwt.ErrMissingLoginValues
			}
			// 凭证必须从环境变量配置
			if adminUsername == "" || adminPassword == "" {
				hlog.Warnf("SECURITY: ADMIN_USERNAME or ADMIN_PASSWORD environment variable is not set. Authentication will fail.")
				return nil, jwt.ErrFailedAuthentication
			}
			if loginVals.Username == adminUsername && loginVals.Password == adminPassword {
				roles := []string{adminRole}
				// 如果提供了 RoleProvider，从 store 查询用户角色
				if roleProvider != nil {
					if userRoles, err := roleProvider.GetUserRoles(ctx, "default", loginVals.Username); err == nil && len(userRoles) > 0 {
						roles = userRoles
					}
				}
				return &AuthUser{Username: loginVals.Username, TenantID: "default", UserID: loginVals.Username, Roles: roles}, nil
			}
			return nil, jwt.ErrFailedAuthentication
		},
		Authorizator: func(data interface{}, ctx context.Context, c *app.RequestContext) bool {
			// SECURITY: Require user to have at least one role for authorization
			if u, ok := data.(*AuthUser); ok && u != nil && len(u.Roles) > 0 {
				return true
			}
			return false
		},
		Unauthorized: func(ctx context.Context, c *app.RequestContext, code int, message string) {
			c.JSON(code, map[string]interface{}{"code": code, "message": message})
		},
	})
	if err != nil {
		return nil, err
	}
	if errInit := authMiddleware.MiddlewareInit(); errInit != nil {
		return nil, errInit
	}
	return &JWTAuth{Middleware: authMiddleware}, nil
}

// AuthUser 登录用户；JWT claims 含 tenant_id、user_id、roles 供 RBAC
type AuthUser struct {
	Username string
	TenantID string
	UserID   string
	Roles    []string
}

// getClaimString 从 JWT claims 安全取字符串
func getClaimString(claims jwt.MapClaims, key string) string {
	if v, ok := claims[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

// getClaimRoles 从 JWT claims 安全取字符串数组（roles）
func getClaimRoles(claims jwt.MapClaims, key string) []string {
	if v, ok := claims[key]; ok {
		if roles, ok := v.([]interface{}); ok {
			result := make([]string, 0, len(roles))
			for _, r := range roles {
				if s, ok := r.(string); ok {
					result = append(result, s)
				}
			}
			return result
		}
	}
	return []string{}
}

// InjectAuthContext 将 tenant_id、user_id 注入 context：
// - Tenant ID: JWT claims > X-Tenant-ID header (if matches JWT) > default
// - User ID: JWT claims > X-User-ID header > anonymous
// Security: X-Tenant-ID header is only trusted if it matches JWT claims (prevents header injection)
func (m *Middleware) InjectAuthContext() app.HandlerFunc {
	return func(ctx context.Context, c *app.RequestContext) {
		// Tenant ID from JWT claims (preferred, cryptographically bound)
		var tenantID string
		if claims := jwt.ExtractClaims(ctx, c); getClaimString(claims, "tenant_id") != "" {
			tenantID = getClaimString(claims, "tenant_id")
		}

		// If X-Tenant-ID header is present and a JWT tenant is set, validate that they match.
		// The header is never used to select a tenant when JWT has no tenant_id, to avoid
		// unauthenticated or tenant-less tokens choosing an arbitrary tenant via header.
		if headerTenantID := string(c.GetHeader("X-Tenant-ID")); headerTenantID != "" {
			if tenantID != "" && headerTenantID != tenantID {
				// Header injection / tenant-spoofing attempt detected - reject
				c.JSON(consts.StatusForbidden, map[string]string{
					"error": "X-Tenant-ID header does not match JWT tenant_id",
				})
				c.Abort()
				return
			}
		}

		// Default tenant if none found
		if tenantID == "" {
			tenantID = "default"
		}
		ctx = auth.WithTenantID(ctx, tenantID)

		// User ID from JWT claims (preferred)
		var userID string
		if claims := jwt.ExtractClaims(ctx, c); getClaimString(claims, "user_id") != "" {
			userID = getClaimString(claims, "user_id")
		}

		// X-User-ID header can override if present (less security critical)
		if headerUserID := string(c.GetHeader("X-User-ID")); headerUserID != "" {
			if userID == "" {
				userID = headerUserID
			}
		}

		// Default user if none found
		if userID == "" {
			userID = "anonymous"
		}
		ctx = auth.WithUserID(ctx, userID)

		c.Next(ctx)
	}
}

// RateLimit 速率限制中间件
func (m *Middleware) RateLimit(rps int) app.HandlerFunc {
	var (
		mu       sync.Mutex
		lastTime time.Time
		count    int
	)
	if rps <= 0 {
		rps = 100 // default
	}
	return func(ctx context.Context, c *app.RequestContext) {
		mu.Lock()
		now := time.Now()
		if now.Sub(lastTime) > time.Second {
			lastTime = now
			count = 0
		}
		count++
		if count > rps {
			mu.Unlock()
			c.JSON(consts.StatusTooManyRequests, map[string]string{
				"error": "请求过于频繁，请稍后再试",
			})
			c.Abort()
			return
		}
		mu.Unlock()
		c.Next(ctx)
	}
}

// TenantRateLimiterPerTenant 租户级别速率限制
type TenantRateLimiterPerTenant struct {
	mu         sync.RWMutex
	limiters   map[string]*tenantLimiter
	defaultRPS int
}

type tenantLimiter struct {
	mu       sync.Mutex
	lastTime time.Time
	count    int
	rps      int
}

// NewTenantRateLimiter 创建租户速率限制器
func NewTenantRateLimiter(defaultRPS int) *TenantRateLimiterPerTenant {
	if defaultRPS <= 0 {
		defaultRPS = 100
	}
	return &TenantRateLimiterPerTenant{
		limiters:   make(map[string]*tenantLimiter),
		defaultRPS: defaultRPS,
	}
}

// SetTenantRate 设置租户速率限制
func (t *TenantRateLimiterPerTenant) SetTenantRate(tenantID string, rps int) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.limiters[tenantID] = &tenantLimiter{rps: rps}
}

// Middleware 返回 Hertz 中间件
func (t *TenantRateLimiterPerTenant) Middleware() app.HandlerFunc {
	return func(ctx context.Context, c *app.RequestContext) {
		tenantID := auth.GetTenantID(ctx)
		if tenantID == "" {
			tenantID = "default"
		}

		t.mu.RLock()
		limiter, exists := t.limiters[tenantID]
		if !exists {
			// 使用默认限制器
			t.mu.RUnlock()
			t.mu.Lock()
			limiter = &tenantLimiter{rps: t.defaultRPS}
			t.limiters[tenantID] = limiter
			t.mu.Unlock()
		} else {
			t.mu.RUnlock()
		}

		limiter.mu.Lock()
		now := time.Now()
		if now.Sub(limiter.lastTime) > time.Second {
			limiter.lastTime = now
			limiter.count = 0
		}
		limiter.count++
		if limiter.count > limiter.rps {
			limiter.mu.Unlock()
			c.JSON(consts.StatusTooManyRequests, map[string]string{
				"error":  "租户请求过于频繁，请稍后再试",
				"tenant": tenantID,
			})
			c.Abort()
			return
		}
		limiter.mu.Unlock()
		c.Next(ctx)
	}
}

// AccessLog 访问日志中间件（使用 hlog）
func (m *Middleware) AccessLog() app.HandlerFunc {
	return func(ctx context.Context, c *app.RequestContext) {
		start := time.Now()
		c.Next(ctx)
		latency := time.Since(start)
		hlog.CtxInfof(ctx, "%s %s %s %d %s",
			c.Method(), c.Path(), c.ClientIP(), c.Response.StatusCode(), latency)
	}
}

// Legacy 为迁移期接口添加标准化 deprecation 响应头。
func (m *Middleware) Legacy(replacement string) app.HandlerFunc {
	return func(ctx context.Context, c *app.RequestContext) {
		c.Header("Deprecation", "true")
		c.Header("X-Aetheris-Deprecated", "true")
		if replacement != "" {
			c.Header("X-Aetheris-Replacement", replacement)
		}
		prevWarning := string(c.Response.Header.Peek("Warning"))
		warning := `299 - "Deprecated API: migrate to runtime-first endpoints"`
		if prevWarning != "" {
			warning = strings.TrimSpace(prevWarning) + ", " + warning
		}
		c.Header("Warning", warning)
		c.Next(ctx)
	}
}
