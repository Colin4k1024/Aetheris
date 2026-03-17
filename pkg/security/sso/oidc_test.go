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

package sso

import (
	"testing"
)

func TestGenerateState(t *testing.T) {
	state, err := GenerateState()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if state == "" {
		t.Error("expected non-empty state")
	}
}

func TestGenerateState_Unique(t *testing.T) {
	state1, _ := GenerateState()
	state2, _ := GenerateState()
	if state1 == state2 {
		t.Error("expected unique states")
	}
}

func TestHashState(t *testing.T) {
	state := "test-state"
	hash := HashState(state)
	if hash == "" {
		t.Error("expected non-empty hash")
	}
}

func TestHashState_Deterministic(t *testing.T) {
	state := "test-state"
	hash1 := HashState(state)
	hash2 := HashState(state)
	if hash1 != hash2 {
		t.Error("expected deterministic hash")
	}
}

func TestHashState_DifferentInputs(t *testing.T) {
	hash1 := HashState("state1")
	hash2 := HashState("state2")
	if hash1 == hash2 {
		t.Error("expected different hashes for different inputs")
	}
}

func TestOIDCConfig(t *testing.T) {
	config := OIDCConfig{
		IssuerURL:      "https://example.com",
		ClientID:        "client-id",
		ClientSecret:    "client-secret",
		RedirectURL:    "https://app.com/callback",
		Scopes:         []string{"openid", "profile", "email"},
		AllowedDomains: []string{"example.com"},
	}

	if config.IssuerURL != "https://example.com" {
		t.Errorf("expected https://example.com, got %s", config.IssuerURL)
	}
	if config.ClientID != "client-id" {
		t.Errorf("expected client-id, got %s", config.ClientID)
	}
	if len(config.Scopes) != 3 {
		t.Errorf("expected 3 scopes, got %d", len(config.Scopes))
	}
	if len(config.AllowedDomains) != 1 {
		t.Errorf("expected 1 domain, got %d", len(config.AllowedDomains))
	}
}

func TestUserInfo(t *testing.T) {
	userInfo := UserInfo{
		Sub:           "user123",
		Name:          "Test User",
		Email:         "test@example.com",
		EmailVerified: true,
		Picture:       "https://example.com/photo.jpg",
		Claims:        map[string]interface{}{"custom": "claim"},
	}

	if userInfo.Sub != "user123" {
		t.Errorf("expected user123, got %s", userInfo.Sub)
	}
	if userInfo.Name != "Test User" {
		t.Errorf("expected Test User, got %s", userInfo.Name)
	}
	if userInfo.Email != "test@example.com" {
		t.Errorf("expected test@example.com, got %s", userInfo.Email)
	}
	if !userInfo.EmailVerified {
		t.Error("expected email verified to be true")
	}
	if userInfo.Picture != "https://example.com/photo.jpg" {
		t.Errorf("expected photo URL, got %s", userInfo.Picture)
	}
	if userInfo.Claims["custom"] != "claim" {
		t.Error("expected custom claim")
	}
}

func TestSAMLConfig(t *testing.T) {
	config := SAMLConfig{
		SSOURL:    "https://idp.example.com/sso",
		Issuer:    "https://sp.example.com",
		Certificate: "-----BEGIN CERTIFICATE-----\ntest\n-----END CERTIFICATE-----",
		SPEntityID: "https://sp.example.com/entity",
		ACSURL:    "https://sp.example.com/acs",
		SLOURL:    "https://idp.example.com/slo",
		AttributeMapping: map[string]string{
			"email": "mail",
			"name":  "cn",
		},
	}

	if config.SSOURL != "https://idp.example.com/sso" {
		t.Errorf("unexpected SSO URL: %s", config.SSOURL)
	}
	if len(config.AttributeMapping) != 2 {
		t.Errorf("expected 2 attribute mappings, got %d", len(config.AttributeMapping))
	}
}

func TestNewSAMLClient(t *testing.T) {
	config := SAMLConfig{
		SSOURL:      "https://idp.example.com/sso",
		Certificate: "test-cert",
	}

	client, err := NewSAMLClient(config)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if client == nil {
		t.Fatal("expected non-nil client")
	}
}

func TestNewSAMLClient_MissingConfig(t *testing.T) {
	config := SAMLConfig{}
	_, err := NewSAMLClient(config)
	if err == nil {
		t.Error("expected error for missing config")
	}
}

func TestSAMLClient_ValidateUser(t *testing.T) {
	config := SAMLConfig{
		SSOURL:      "https://idp.example.com/sso",
		Certificate: "test-cert",
	}
	client, _ := NewSAMLClient(config)

	userInfo := &UserInfo{
		Email: "test@example.com",
	}
	err := client.ValidateUser(userInfo)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestSAMLClient_GetLogoutURL(t *testing.T) {
	config := SAMLConfig{
		SSOURL:      "https://idp.example.com/sso",
		Certificate: "test-cert",
	}
	client, _ := NewSAMLClient(config)

	url := client.GetLogoutURL("https://app.com/logout")
	if url != "https://app.com/logout" {
		t.Errorf("expected redirect URL, got %s", url)
	}
}

func TestSAMLClient_Exchange_NotImplemented(t *testing.T) {
	config := SAMLConfig{
		SSOURL:      "https://idp.example.com/sso",
		Certificate: "test-cert",
	}
	client, _ := NewSAMLClient(config)

	_, err := client.Exchange(nil, "code")
	if err == nil {
		t.Error("expected error for SAML exchange")
	}
}

func TestSAMLClient_GetUserInfo_NotImplemented(t *testing.T) {
	config := SAMLConfig{
		SSOURL:      "https://idp.example.com/sso",
		Certificate: "test-cert",
	}
	client, _ := NewSAMLClient(config)

	_, err := client.GetUserInfo(nil, nil)
	if err == nil {
		t.Error("expected error for SAML userinfo")
	}
}

func TestSAMLClient_LoginURL_NotImplemented(t *testing.T) {
	config := SAMLConfig{
		SSOURL:      "https://idp.example.com/sso",
		Certificate: "test-cert",
	}
	client, _ := NewSAMLClient(config)

	_, err := client.LoginURL("state", "https://app.com/callback")
	if err == nil {
		t.Error("expected error for SAML login")
	}
}

func TestNewProvider_InvalidType(t *testing.T) {
	_, err := NewProvider(nil, "invalid", nil)
	if err == nil {
		t.Error("expected error for invalid provider type")
	}
}

func TestNewProvider_OIDCConfigError(t *testing.T) {
	// OIDC config with missing required fields
	_, err := NewProvider(nil, "oidc", map[string]string{})
	if err == nil {
		t.Error("expected error for invalid OIDC config type")
	}
}

func TestNewProvider_SAMLConfigError(t *testing.T) {
	// SAML config with wrong type
	_, err := NewProvider(nil, "saml", "invalid")
	if err == nil {
		t.Error("expected error for invalid SAML config type")
	}
}

func TestOIDCDiscovery(t *testing.T) {
	discovery := OIDCDiscovery{
		Issuer:                 "https://example.com",
		AuthorizationEndpoint: "https://example.com/auth",
		TokenEndpoint:          "https://example.com/token",
		UserInfoEndpoint:       "https://example.com/userinfo",
		EndSessionEndpoint:    "https://example.com/logout",
		JWKSURI:               "https://example.com/jwks",
	}

	if discovery.Issuer != "https://example.com" {
		t.Errorf("expected issuer, got %s", discovery.Issuer)
	}
	if discovery.AuthorizationEndpoint != "https://example.com/auth" {
		t.Errorf("expected auth endpoint, got %s", discovery.AuthorizationEndpoint)
	}
}
