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
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/common/hlog"
	"rag-platform/pkg/security/sso"
)

// OIDCMiddleware OIDC/SSO 认证中间件
type OIDCMiddleware struct {
	client        *sso.OIDCClient
	loginURL      string
	logoutURL     string
	callbackPath  string
	sessionName   string
	sessionExpiry time.Duration
}

// OIDCMiddlewareConfig OIDC 中间件配置
type OIDCMiddlewareConfig struct {
	Enabled        bool
	IssuerURL      string
	ClientID       string
	ClientSecret   string
	RedirectURL    string
	Scopes         []string
	AllowedDomains []string
	LoginPath      string
	LogoutPath     string
	CallbackPath   string
	SessionName    string
	SessionExpiry  time.Duration
}

// NewOIDCMiddleware 创建 OIDC 中间件
func NewOIDCMiddleware(ctx context.Context, cfg OIDCMiddlewareConfig) (*OIDCMiddleware, error) {
	if !cfg.Enabled {
		return nil, nil
	}

	client, err := sso.NewOIDCClient(ctx, sso.OIDCConfig{
		IssuerURL:      cfg.IssuerURL,
		ClientID:       cfg.ClientID,
		ClientSecret:   cfg.ClientSecret,
		RedirectURL:    cfg.RedirectURL,
		Scopes:         cfg.Scopes,
		AllowedDomains: cfg.AllowedDomains,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create OIDC client: %w", err)
	}

	// 设置默认值
	if cfg.LoginPath == "" {
		cfg.LoginPath = "/auth/login"
	}
	if cfg.LogoutPath == "" {
		cfg.LogoutPath = "/auth/logout"
	}
	if cfg.CallbackPath == "" {
		cfg.CallbackPath = "/auth/callback"
	}
	if cfg.SessionName == "" {
		cfg.SessionName = "aetheris_sso_session"
	}
	if cfg.SessionExpiry == 0 {
		cfg.SessionExpiry = 24 * time.Hour
	}

	return &OIDCMiddleware{
		client:        client,
		loginURL:      cfg.LoginPath,
		logoutURL:     cfg.LogoutPath,
		callbackPath:  cfg.CallbackPath,
		sessionName:   cfg.SessionName,
		sessionExpiry: cfg.SessionExpiry,
	}, nil
}

// Handler 返回处理认证的 Handler
func (m *OIDCMiddleware) Handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path

		switch path {
		case m.loginURL:
			m.handleLogin(w, r)
		case m.logoutURL:
			m.handleLogout(w, r)
		case m.callbackPath:
			m.handleCallback(w, r)
		default:
			http.NotFound(w, r)
		}
	})
}

// handleLogin 处理登录请求
func (m *OIDCMiddleware) handleLogin(w http.ResponseWriter, r *http.Request) {
	// 生成 state
	state, err := sso.GenerateState()
	if err != nil {
		http.Error(w, "failed to generate state", http.StatusInternalServerError)
		return
	}

	// 获取重定向 URL（默认回到首页）
	redirectURL := r.URL.Query().Get("redirect_url")
	if redirectURL == "" {
		redirectURL = "/"
	}

	// 生成登录 URL
	loginURL, err := m.client.LoginURL(state, redirectURL)
	if err != nil {
		http.Error(w, "failed to generate login URL", http.StatusInternalServerError)
		return
	}

	// 302 重定向到 OIDC 提供商
	http.Redirect(w, r, loginURL, http.StatusFound)
}

// handleLogout 处理登出请求
func (m *OIDCMiddleware) handleLogout(w http.ResponseWriter, r *http.Request) {
	// 清除 session
	cookie := &http.Cookie{
		Name:   m.sessionName,
		Value:  "",
		Path:   "/",
		MaxAge: -1,
	}
	http.SetCookie(w, cookie)

	// 获取重定向 URL
	redirectURL := r.URL.Query().Get("redirect_url")
	if redirectURL == "" {
		redirectURL = "/"
	}

	// 登出 URL
	logoutURL := m.client.GetLogoutURL(redirectURL)

	// 302 重定向
	http.Redirect(w, r, logoutURL, http.StatusFound)
}

// handleCallback 处理 OIDC 回调
func (m *OIDCMiddleware) handleCallback(w http.ResponseWriter, r *http.Request) {
	// 获取 code 和 state
	code := r.URL.Query().Get("code")
	state := r.URL.Query().Get("state")

	if code == "" || state == "" {
		http.Error(w, "missing code or state", http.StatusBadRequest)
		return
	}

	// 验证 state
	stateData, err := m.client.ValidateState(state)
	if err != nil {
		http.Error(w, "invalid state: "+err.Error(), http.StatusBadRequest)
		return
	}

	// 交换 code 获取 token
	token, err := m.client.Exchange(r.Context(), code)
	if err != nil {
		hlog.Errorf("OIDC exchange failed: %v", err)
		http.Error(w, "failed to exchange code", http.StatusInternalServerError)
		return
	}

	// 获取用户信息
	userInfo, err := m.client.GetUserInfo(r.Context(), token)
	if err != nil {
		hlog.Errorf("OIDC get user info failed: %v", err)
		http.Error(w, "failed to get user info", http.StatusInternalServerError)
		return
	}

	// 验证用户
	if err := m.client.ValidateUser(userInfo); err != nil {
		hlog.Warnf("OIDC user validation failed: %v", err)
		http.Error(w, "user not allowed: "+err.Error(), http.StatusForbidden)
		return
	}

	// 将用户信息写入 session
	sessionValue := fmt.Sprintf("%s|%s|%s", userInfo.Sub, userInfo.Email, userInfo.Name)
	sessionCookie := &http.Cookie{
		Name:     m.sessionName,
		Value:    sessionValue,
		Path:     "/",
		MaxAge:   int(m.sessionExpiry.Seconds()),
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
	}
	http.SetCookie(w, sessionCookie)

	// 重定向回原始页面
	redirectURL := stateData.RedirectURL
	if redirectURL == "" {
		redirectURL = "/"
	}

	http.Redirect(w, r, redirectURL, http.StatusFound)
}

// Middleware 返回 Hertz 中间件（用于保护其他路由）
func (m *OIDCMiddleware) Middleware() app.HandlerFunc {
	if m == nil {
		return func(c context.Context, ctx *app.RequestContext) {
			ctx.Next(c)
		}
	}

	return func(c context.Context, ctx *app.RequestContext) {
		// 跳过认证的路径
		if m.isExcludedPath(string(ctx.Path())) {
			ctx.Next(c)
			return
		}

		// 获取 session
		sessionValue := ctx.Cookie(m.sessionName)
		if sessionValue == nil || string(sessionValue) == "" {
			// 未登录，重定向到登录页
			loginURL := m.loginURL + "?redirect_url=" + url.QueryEscape(string(ctx.Path()))
			ctx.Redirect(http.StatusFound, []byte(loginURL))
			ctx.Abort()
			return
		}

		// 解析 session
		sessionParts := strings.Split(string(sessionValue), "|")
		if len(sessionParts) < 2 {
			// 无效 session
			ctx.Redirect(http.StatusFound, []byte(m.loginURL))
			ctx.Abort()
			return
		}

		// 将用户信息注入 context
		ctx.Set("user_id", sessionParts[0])
		ctx.Set("user_email", sessionParts[1])
		if len(sessionParts) > 2 {
			ctx.Set("user_name", sessionParts[2])
		}

		ctx.Next(c)
	}
}

// isExcludedPath 检查路径是否跳过认证
func (m *OIDCMiddleware) isExcludedPath(path string) bool {
	excludedPaths := []string{
		m.loginURL,
		m.logoutURL,
		m.callbackPath,
		"/metrics",
		"/health",
		"/api/health",
		"/api/login",
	}

	for _, p := range excludedPaths {
		if strings.HasPrefix(path, p) {
			return true
		}
	}

	return false
}

// urlQueryEscape 是 url.QueryEscape 的别名（避免导入冲突）
func urlQueryEscape(s string) string {
	return strings.ReplaceAll(
		strings.ReplaceAll(s, "+", "%2B"),
		"&", "%26",
	)
}

// OIDCConfigFromConfig 从应用配置创建 OIDC 中间件配置
func OIDCConfigFromConfig(issuerURL, clientID, clientSecret, redirectURL string, allowedDomains []string) OIDCMiddlewareConfig {
	return OIDCMiddlewareConfig{
		Enabled:        issuerURL != "",
		IssuerURL:      issuerURL,
		ClientID:       clientID,
		ClientSecret:   clientSecret,
		RedirectURL:    redirectURL,
		AllowedDomains: allowedDomains,
		Scopes:         []string{"openid", "profile", "email"},
		SessionName:    "aetheris_sso_session",
		SessionExpiry:  24 * time.Hour,
	}
}
