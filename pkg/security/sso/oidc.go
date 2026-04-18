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

// Package sso provides OIDC/SAML SSO integration.
package sso

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/coreos/go-oidc"
	"github.com/google/uuid"
	"golang.org/x/oauth2"
)

// OIDCConfig OIDC 配置
type OIDCConfig struct {
	IssuerURL      string   // OIDC Issuer URL (如 https://accounts.google.com, https://your-idp.okta.com)
	ClientID       string   // OAuth2 Client ID
	ClientSecret   string   // OAuth2 Client Secret
	RedirectURL    string   // 回调 URL
	Scopes         []string // 请求的 Scope
	AllowedDomains []string // 允许的邮箱域名
	LoginURL       string   // 自定义登录页面 URL（可选）
	LogoutURL      string   // 登出 URL（可选）
}

// UserInfo 用户信息
type UserInfo struct {
	Sub           string                 `json:"sub"`
	Name          string                 `json:"name"`
	Email         string                 `json:"email"`
	EmailVerified bool                   `json:"email_verified"`
	Picture       string                 `json:"picture"`
	Claims        map[string]interface{} `json:"claims"`
}

// OIDCClient OIDC 客户端
type OIDCClient struct {
	config       *OIDCConfig
	provider     *oidc.Provider
	oauth2Config oauth2.Config
	verifier     *oidc.IDTokenVerifier
	stateStore   map[string]*State
}

// State OIDC 认证状态
type State struct {
	Nonce       string
	RedirectURL string
	CreatedAt   time.Time
}

// NewOIDCClient 创建 OIDC 客户端
func NewOIDCClient(ctx context.Context, config OIDCConfig) (*OIDCClient, error) {
	if config.IssuerURL == "" || config.ClientID == "" {
		return nil, fmt.Errorf("OIDC issuer URL and client ID are required")
	}

	// 设置默认 scopes
	if len(config.Scopes) == 0 {
		config.Scopes = []string{oidc.ScopeOpenID, "profile", "email"}
	}

	// 创建 provider
	provider, err := oidc.NewProvider(ctx, config.IssuerURL)
	if err != nil {
		return nil, fmt.Errorf("failed to create OIDC provider: %w", err)
	}

	// 创建 oauth2 配置
	oauth2Config := oauth2.Config{
		ClientID:     config.ClientID,
		ClientSecret: config.ClientSecret,
		RedirectURL:  config.RedirectURL,
		Endpoint:     provider.Endpoint(),
		Scopes:       config.Scopes,
	}

	// 创建 verifier
	verifier := provider.Verifier(&oidc.Config{
		ClientID: config.ClientID,
	})

	return &OIDCClient{
		config:       &config,
		provider:     provider,
		oauth2Config: oauth2Config,
		verifier:     verifier,
		stateStore:   make(map[string]*State),
	}, nil
}

// LoginURL 生成登录 URL
func (c *OIDCClient) LoginURL(state string, redirectURL string) (string, error) {
	// 生成 nonce
	nonce, err := generateNonce()
	if err != nil {
		return "", err
	}

	// 保存 state
	c.stateStore[state] = &State{
		Nonce:       nonce,
		RedirectURL: redirectURL,
		CreatedAt:   time.Now(),
	}

	// 清理过期的 state
	c.cleanupStates()

	// 生成授权 URL
	authURL := c.oauth2Config.AuthCodeURL(state, oauth2.AccessTypeOnline, oidc.Nonce(nonce))
	return authURL, nil
}

// Exchange 交换 code 获取 token
func (c *OIDCClient) Exchange(ctx context.Context, code string) (*oauth2.Token, error) {
	token, err := c.oauth2Config.Exchange(ctx, code)
	if err != nil {
		return nil, fmt.Errorf("failed to exchange code: %w", err)
	}
	return token, nil
}

// VerifyIDToken 验证 ID Token
func (c *OIDCClient) VerifyIDToken(ctx context.Context, rawIDToken string) (*oidc.IDToken, error) {
	idToken, err := c.verifier.Verify(ctx, rawIDToken)
	if err != nil {
		return nil, fmt.Errorf("failed to verify ID token: %w", err)
	}
	return idToken, nil
}

// GetUserInfo 获取用户信息
func (c *OIDCClient) GetUserInfo(ctx context.Context, token *oauth2.Token) (*UserInfo, error) {
	userInfo := &UserInfo{
		Claims: make(map[string]interface{}),
	}

	// 从 token 中提取用户信息
	rawIDToken, ok := token.Extra("id_token").(string)
	if !ok {
		return nil, fmt.Errorf("no id_token in response")
	}

	idToken, err := c.verifier.Verify(ctx, rawIDToken)
	if err != nil {
		return nil, fmt.Errorf("failed to verify ID token: %w", err)
	}

	if err := idToken.Claims(&userInfo.Claims); err != nil {
		return nil, fmt.Errorf("failed to parse claims: %w", err)
	}

	// 提取常用字段
	if sub, ok := userInfo.Claims["sub"].(string); ok {
		userInfo.Sub = sub
	}
	if name, ok := userInfo.Claims["name"].(string); ok {
		userInfo.Name = name
	}
	if email, ok := userInfo.Claims["email"].(string); ok {
		userInfo.Email = email
	}
	if emailVerified, ok := userInfo.Claims["email_verified"].(bool); ok {
		userInfo.EmailVerified = emailVerified
	}
	if picture, ok := userInfo.Claims["picture"].(string); ok {
		userInfo.Picture = picture
	}

	return userInfo, nil
}

// ValidateUser 验证用户是否允许登录
func (c *OIDCClient) ValidateUser(userInfo *UserInfo) error {
	// 检查邮箱域名
	if len(c.config.AllowedDomains) > 0 {
		if userInfo.Email == "" {
			return fmt.Errorf("email is required for domain validation")
		}

		domain := strings.Split(userInfo.Email, "@")
		if len(domain) < 2 {
			return fmt.Errorf("invalid email format")
		}

		allowed := false
		for _, d := range c.config.AllowedDomains {
			if strings.EqualFold(domain[1], d) {
				allowed = true
				break
			}
		}

		if !allowed {
			return fmt.Errorf("domain %s is not allowed", domain[1])
		}
	}

	return nil
}

// GetLogoutURL 生成登出 URL
func (c *OIDCClient) GetLogoutURL(redirectURL string) string {
	if c.config.LogoutURL != "" {
		logoutURL, _ := url.Parse(c.config.LogoutURL)
		params := url.Values{}
		params.Set("post_logout_redirect_uri", redirectURL)
		logoutURL.RawQuery = params.Encode()
		return logoutURL.String()
	}

	// 默认使用 OIDC end_session_endpoint
	var claims struct {
		EndSessionEndpoint string `json:"end_session_endpoint"`
	}
	if err := c.provider.Claims(&claims); err != nil {
		return redirectURL
	}
	endSessionEndpoint := claims.EndSessionEndpoint

	if endSessionEndpoint != "" {
		logoutURL, _ := url.Parse(endSessionEndpoint)
		params := url.Values{}
		params.Set("post_logout_redirect_uri", redirectURL)
		params.Set("client_id", c.config.ClientID)
		logoutURL.RawQuery = params.Encode()
		return logoutURL.String()
	}

	return redirectURL
}

// ValidateState 验证 state 参数
func (c *OIDCClient) ValidateState(state string) (*State, error) {
	s, ok := c.stateStore[state]
	if !ok {
		return nil, fmt.Errorf("invalid state")
	}

	// 检查是否过期 (10 分钟)
	if time.Since(s.CreatedAt) > 10*time.Minute {
		delete(c.stateStore, state)
		return nil, fmt.Errorf("state expired")
	}

	delete(c.stateStore, state)
	return s, nil
}

// cleanupStates 清理过期的 state
func (c *OIDCClient) cleanupStates() {
	now := time.Now()
	for state, s := range c.stateStore {
		if now.Sub(s.CreatedAt) > 10*time.Minute {
			delete(c.stateStore, state)
		}
	}
}

// generateNonce 生成 nonce
func generateNonce() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b), nil
}

// GenerateState 生成 state
func GenerateState() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b), nil
}

// HashState 生成 state 的哈希（用于存储）
func HashState(state string) string {
	h := sha256.Sum256([]byte(state))
	return base64.URLEncoding.EncodeToString(h[:])
}

// OIDCDiscovery OIDC Discovery 文档
type OIDCDiscovery struct {
	Issuer                string `json:"issuer"`
	AuthorizationEndpoint string `json:"authorization_endpoint"`
	TokenEndpoint         string `json:"token_endpoint"`
	UserInfoEndpoint      string `json:"userinfo_endpoint"`
	EndSessionEndpoint    string `json:"end_session_endpoint"`
	JWKSURI               string `json:"jwks_uri"`
}

// FetchDiscovery 获取 OIDC Discovery 文档
func FetchDiscovery(ctx context.Context, issuerURL string) (*OIDCDiscovery, error) {
	discoveryURL := strings.TrimSuffix(issuerURL, "/") + "/.well-known/openid-configuration"

	// Validate URL scheme to prevent SSRF
	parsedURL, err := url.Parse(discoveryURL)
	if err != nil {
		return nil, fmt.Errorf("invalid discovery URL: %w", err)
	}
	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		return nil, fmt.Errorf("invalid URL scheme: %s, only http and https are allowed", parsedURL.Scheme)
	}

	// Use HTTP client with timeout to prevent SSRF and resource exhaustion
	client := &http.Client{
		Timeout: 10 * time.Second,
	}
	resp, err := client.Get(discoveryURL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch discovery: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("discovery returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 10*1024*1024))
	if err != nil {
		return nil, fmt.Errorf("failed to read discovery body: %w", err)
	}

	var discovery OIDCDiscovery
	if err := json.Unmarshal(body, &discovery); err != nil {
		return nil, fmt.Errorf("failed to parse discovery: %w", err)
	}

	return &discovery, nil
}

// SAMLConfig SAML 配置
type SAMLConfig struct {
	SSOURL           string            // SSO URL
	Issuer           string            // Entity ID
	Certificate      string            // IdP 证书
	SPEntityID       string            // SP Entity ID
	ACSURL           string            // Assertion Consumer Service URL
	SLOURL           string            // Single Logout URL
	AttributeMapping map[string]string // 属性映射
}

// SAMLClient SAML 客户端（简化实现）
type SAMLClient struct {
	config *SAMLConfig
}

// NewSAMLClient 创建 SAML 客户端
func NewSAMLClient(config SAMLConfig) (*SAMLClient, error) {
	if config.SSOURL == "" || config.Certificate == "" {
		return nil, fmt.Errorf("SAML SSO URL and certificate are required")
	}

	return &SAMLClient{
		config: &config,
	}, nil
}

// Exchange 实现 Provider 接口（暂不支持）
func (c *SAMLClient) Exchange(ctx context.Context, code string) (*oauth2.Token, error) {
	return nil, fmt.Errorf("SAML exchange not implemented, use OIDC")
}

// GetUserInfo 实现 Provider 接口（暂不支持）
func (c *SAMLClient) GetUserInfo(ctx context.Context, token *oauth2.Token) (*UserInfo, error) {
	return nil, fmt.Errorf("SAML userinfo not implemented, use OIDC")
}

// ValidateUser 实现 Provider 接口
func (c *SAMLClient) ValidateUser(userInfo *UserInfo) error {
	return nil
}

// GetLogoutURL 实现 Provider 接口
func (c *SAMLClient) GetLogoutURL(redirectURL string) string {
	return redirectURL
}

// LoginURL 实现 Provider 接口（暂不支持）
func (c *SAMLClient) LoginURL(state string, redirectURL string) (string, error) {
	return "", fmt.Errorf("SAML login not implemented, use OIDC")
}

// GetLoginURL 生成 SAML 登录 URL
func (c *SAMLClient) GetLoginURL(relayState string) string {
	// SAML 简化实现 - 实际需要构建完整的 AuthnRequest
	authnRequest := fmt.Sprintf(`<samlp:AuthnRequest xmlns:samlp="urn:oasis:names:tc:SAML:2.0:protocol" ID="_%s" Version="2.0" IssueInstant="%s" AssertionConsumerServiceURL="%s">
		<saml:Issuer xmlns:saml="urn:oasis:names:tc:SAML:2.0:assertion">%s</saml:Issuer>
	</samlp:AuthnRequest>`,
		uuid.New().String(),
		time.Now().Format(time.RFC3339),
		c.config.ACSURL,
		c.config.SPEntityID,
	)

	encoded := base64.StdEncoding.EncodeToString([]byte(authnRequest))
	return fmt.Sprintf("%s?SAMLRequest=%s", c.config.SSOURL, url.QueryEscape(encoded))
}

// Provider SSO Provider 接口
type Provider interface {
	LoginURL(state string, redirectURL string) (string, error)
	Exchange(ctx context.Context, code string) (*oauth2.Token, error)
	GetUserInfo(ctx context.Context, token *oauth2.Token) (*UserInfo, error)
	ValidateUser(userInfo *UserInfo) error
	GetLogoutURL(redirectURL string) string
}

// NewProvider 根据配置创建 SSO Provider
func NewProvider(ctx context.Context, providerType string, config interface{}) (Provider, error) {
	switch strings.ToLower(providerType) {
	case "oidc":
		cfg, ok := config.(OIDCConfig)
		if !ok {
			return nil, fmt.Errorf("invalid OIDC config")
		}
		return NewOIDCClient(ctx, cfg)
	case "saml":
		cfg, ok := config.(SAMLConfig)
		if !ok {
			return nil, fmt.Errorf("invalid SAML config")
		}
		return NewSAMLClient(cfg)
	default:
		return nil, fmt.Errorf("unsupported SSO provider: %s", providerType)
	}
}
