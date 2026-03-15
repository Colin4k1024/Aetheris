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
	"testing"
	"time"

	"github.com/hertz-contrib/jwt"
)

func TestNewMiddleware(t *testing.T) {
	m := NewMiddleware()
	if m == nil {
		t.Fatal("expected non-nil middleware")
	}
	if m.AllowOrigins != nil {
		t.Error("expected nil AllowOrigins")
	}
}

func TestNewMiddlewareWithCORS(t *testing.T) {
	origins := []string{"http://localhost:3000", "*"}
	m := NewMiddlewareWithCORS(origins)
	if m == nil {
		t.Fatal("expected non-nil middleware")
	}
	if len(m.AllowOrigins) != 2 {
		t.Errorf("expected 2 origins, got %d", len(m.AllowOrigins))
	}
}

func TestGetClaimString(t *testing.T) {
	claims := jwt.MapClaims{
		"id":        "user1",
		"tenant_id": "tenant1",
		"user_id":   "user1",
	}

	tests := []struct {
		key      string
		expected string
	}{
		{"id", "user1"},
		{"tenant_id", "tenant1"},
		{"user_id", "user1"},
		{"nonexistent", ""},
	}

	for _, tt := range tests {
		result := getClaimString(claims, tt.key)
		if result != tt.expected {
			t.Errorf("expected %s, got %s", tt.expected, result)
		}
	}
}

func TestGetClaimString_NilClaims(t *testing.T) {
	result := getClaimString(nil, "id")
	if result != "" {
		t.Error("expected empty string for nil claims")
	}
}

func TestGetClaimString_NonStringValue(t *testing.T) {
	claims := jwt.MapClaims{
		"id": 123, // int instead of string
	}
	result := getClaimString(claims, "id")
	if result != "" {
		t.Error("expected empty string for non-string value")
	}
}

func TestTenantRateLimiter_New(t *testing.T) {
	limiter := NewTenantRateLimiter(50)
	if limiter == nil {
		t.Fatal("expected non-nil limiter")
	}
	if limiter.defaultRPS != 50 {
		t.Errorf("expected 50, got %d", limiter.defaultRPS)
	}
}

func TestTenantRateLimiter_New_Zero(t *testing.T) {
	limiter := NewTenantRateLimiter(0)
	if limiter.defaultRPS != 100 {
		t.Errorf("expected 100 for zero input, got %d", limiter.defaultRPS)
	}
}

func TestTenantRateLimiter_SetTenantRate(t *testing.T) {
	limiter := NewTenantRateLimiter(100)
	limiter.SetTenantRate("tenant1", 200)

	limiter.mu.RLock()
	l := limiter.limiters["tenant1"]
	limiter.mu.RUnlock()

	if l == nil {
		t.Fatal("expected limiter for tenant1")
	}
	if l.rps != 200 {
		t.Errorf("expected 200, got %d", l.rps)
	}
}

func TestAuthUser(t *testing.T) {
	user := &AuthUser{
		Username: "admin",
		TenantID: "tenant1",
		UserID:   "user1",
	}

	if user.Username != "admin" {
		t.Errorf("expected admin, got %s", user.Username)
	}
	if user.TenantID != "tenant1" {
		t.Errorf("expected tenant1, got %s", user.TenantID)
	}
	if user.UserID != "user1" {
		t.Errorf("expected user1, got %s", user.UserID)
	}
}

func TestNewJWTAuth(t *testing.T) {
	key := []byte("test-secret-key")
	timeout := time.Hour
	maxRefresh := time.Hour * 24

	jwtAuth, err := NewJWTAuth(key, timeout, maxRefresh)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if jwtAuth == nil {
		t.Fatal("expected non-nil JWTAuth")
	}
	if jwtAuth.Middleware == nil {
		t.Error("expected non-nil Middleware")
	}
}

func TestNewJWTAuth_EmptyKey(t *testing.T) {
	_, err := NewJWTAuth([]byte(""), time.Hour, time.Hour)
	if err != nil {
		t.Logf("empty key error: %v", err)
	}
}
