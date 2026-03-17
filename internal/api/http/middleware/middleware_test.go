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

func TestParseIPList(t *testing.T) {
	tests := []struct {
		name      string
		ips      []string
		wantCount int
		wantErr   bool
	}{
		{
			name:      "empty list",
			ips:      []string{},
			wantCount: 0,
			wantErr:   false,
		},
		{
			name:      "single IPv4",
			ips:      []string{"192.168.1.1"},
			wantCount: 1,
			wantErr:   false,
		},
		{
			name:      "single IPv6",
			ips:      []string{"::1"},
			wantCount: 1,
			wantErr:   false,
		},
		{
			name:      "IPv4 CIDR",
			ips:      []string{"10.0.0.0/8"},
			wantCount: 1,
			wantErr:   false,
		},
		{
			name:      "IPv6 CIDR",
			ips:      []string{"2001:db8::/32"},
			wantCount: 1,
			wantErr:   false,
		},
		{
			name:      "invalid IP",
			ips:      []string{"invalid"},
			wantCount: 0,
			wantErr:   true,
		},
		{
			name:      "mixed valid",
			ips:      []string{"192.168.1.1", "10.0.0.0/8"},
			wantCount: 2,
			wantErr:   false,
		},
		{
			name:      "empty string",
			ips:      []string{""},
			wantCount: 0,
			wantErr:   false,
		},
		{
			name:      "spaces trimmed",
			ips:      []string{"  192.168.1.1  "},
			wantCount: 1,
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parseIPList(tt.ips)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseIPList() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if len(result) != tt.wantCount {
				t.Errorf("parseIPList() got %d, want %d", len(result), tt.wantCount)
			}
		})
	}
}

func TestNewIPAllowList(t *testing.T) {
	tests := []struct {
		name          string
		allowIPs     []string
		blockIPs     []string
		trustedProxies []string
		wantErr      bool
	}{
		{
			name:          "valid empty",
			allowIPs:     nil,
			blockIPs:     nil,
			trustedProxies: nil,
			wantErr:      false,
		},
		{
			name:          "valid with allow list",
			allowIPs:     []string{"192.168.1.0/24"},
			blockIPs:     nil,
			trustedProxies: nil,
			wantErr:      false,
		},
		{
			name:          "valid with block list",
			allowIPs:     nil,
			blockIPs:     []string{"10.0.0.1"},
			trustedProxies: nil,
			wantErr:      false,
		},
		{
			name:          "invalid IP in allow",
			allowIPs:     []string{"invalid"},
			blockIPs:     nil,
			trustedProxies: nil,
			wantErr:      true,
		},
		{
			name:          "invalid IP in block",
			allowIPs:     nil,
			blockIPs:     []string{"invalid"},
			trustedProxies: nil,
			wantErr:      true,
		},
		{
			name:          "all lists",
			allowIPs:     []string{"192.168.1.0/24"},
			blockIPs:     []string{"192.168.1.100"},
			trustedProxies: []string{"10.0.0.1"},
			wantErr:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewIPAllowList(tt.allowIPs, tt.blockIPs, tt.trustedProxies)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewIPAllowList() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestIPAllowList_isIPAllowed(t *testing.T) {
	// Create IPAllowList with allow list 192.168.1.0/24 and block list 192.168.1.100
	allowList, err := NewIPAllowList([]string{"192.168.1.0/24"}, []string{"192.168.1.100"}, nil)
	if err != nil {
		t.Fatalf("failed to create IPAllowList: %v", err)
	}

	tests := []struct {
		name      string
		ip        string
		wantAllow bool
	}{
		{
			name:      "allowed IP in range",
			ip:        "192.168.1.50",
			wantAllow: true,
		},
		{
			name:      "blocked IP",
			ip:        "192.168.1.100",
			wantAllow: false,
		},
		{
			name:      "outside allow range",
			ip:        "192.168.2.1",
			wantAllow: false,
		},
		{
			name:      "invalid IP",
			ip:        "invalid",
			wantAllow: false,
		},
		{
			name:      "empty IP",
			ip:        "",
			wantAllow: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := allowList.isIPAllowed(tt.ip)
			if got != tt.wantAllow {
				t.Errorf("isIPAllowed(%s) = %v, want %v", tt.ip, got, tt.wantAllow)
			}
		})
	}
}

func TestIPAllowList_EmptyAllowList(t *testing.T) {
	// Empty allow list means allow all (except blocked)
	allowList, err := NewIPAllowList(nil, []string{"10.0.0.1"}, nil)
	if err != nil {
		t.Fatalf("failed to create IPAllowList: %v", err)
	}

	// Should allow 192.168.1.1 since allow list is empty
	if !allowList.isIPAllowed("192.168.1.1") {
		t.Error("expected to allow IP not in block list")
	}

	// Should block 10.0.0.1 since it's in block list
	if allowList.isIPAllowed("10.0.0.1") {
		t.Error("expected to block IP in block list")
	}
}

func TestIPAllowList_getClientIP(t *testing.T) {
	// With trusted proxy
	allowList, err := NewIPAllowList(nil, nil, []string{"10.0.0.0/24"})
	if err != nil {
		t.Fatalf("failed to create IPAllowList: %v", err)
	}

	tests := []struct {
		name           string
		xForwardedFor string
		wantIP        string
	}{
		{
			name:           "no forwarded header",
			xForwardedFor: "",
			wantIP:        "",
		},
		{
			name:           "client behind trusted proxy",
			xForwardedFor: "192.168.1.1, 10.0.0.1",
			wantIP:        "192.168.1.1",
		},
		{
			name:           "only trusted proxy",
			xForwardedFor: "10.0.0.1",
			wantIP:        "",
		},
		{
			name:           "multiple trusted proxies",
			xForwardedFor: "192.168.1.1, 10.0.0.1, 10.0.0.2",
			wantIP:        "192.168.1.1",
		},
		{
			name:           "invalid IP in header",
			xForwardedFor: "invalid, 10.0.0.1",
			wantIP:        "",
		},
		{
			name:           "empty entries",
			xForwardedFor: ", 192.168.1.1, , 10.0.0.1",
			wantIP:        "192.168.1.1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := allowList.getClientIP(tt.xForwardedFor)
			if got != tt.wantIP {
				t.Errorf("getClientIP() = %v, want %v", got, tt.wantIP)
			}
		})
	}
}

func TestNewIPAllowListFromConfig(t *testing.T) {
	tests := []struct {
		name string
		cfg  IPAllowListConfig
		want bool
	}{
		{
			name: "disabled",
			cfg: IPAllowListConfig{
				Enabled: false,
			},
			want: false,
		},
		{
			name: "enabled with valid IPs",
			cfg: IPAllowListConfig{
				Enabled:  true,
				AllowIPs: []string{"192.168.1.0/24"},
			},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := NewIPAllowListFromConfig(tt.cfg)
			if err != nil {
				t.Errorf("NewIPAllowListFromConfig() error = %v", err)
				return
			}
			if (result != nil) != tt.want {
				t.Errorf("NewIPAllowListFromConfig() = %v, want nil = %v", result, !tt.want)
			}
		})
	}
}

func TestDetermineAction(t *testing.T) {
	tests := []struct {
		name   string
		method string
		path   string
		want   string
	}{
		{
			name:   "export path GET",
			method: "GET",
			path:   "/api/export",
			want:   "export_evidence",
		},
		{
			name:   "trace path GET",
			method: "GET",
			path:   "/api/trace",
			want:   "view_trace",
		},
		{
			name:   "jobs GET",
			method: "GET",
			path:   "/api/jobs/123",
			want:   "view_job",
		},
		{
			name:   "jobs POST",
			method: "POST",
			path:   "/api/jobs/123",
			want:   "create_job",
		},
		{
			name:   "jobs DELETE",
			method: "DELETE",
			path:   "/api/jobs/123",
			want:   "delete_job",
		},
		{
			name:   "jobs POST stop",
			method: "POST",
			path:   "/api/jobs/123/stop",
			want:   "stop_job",
		},
		{
			name:   "jobs POST signal",
			method: "POST",
			path:   "/api/jobs/123/signal",
			want:   "signal_job",
		},
		{
			name:   "unknown path",
			method: "GET",
			path:   "/api/unknown",
			want:   "unknown",
		},
		{
			name:   "root path",
			method: "GET",
			path:   "/",
			want:   "unknown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := determineAction(tt.method, tt.path)
			if got != tt.want {
				t.Errorf("determineAction() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestExtractResource(t *testing.T) {
	tests := []struct {
		name         string
		path         string
		wantType     string
		wantID       string
	}{
		{
			name:         "jobs path",
			path:         "/api/jobs/123",
			wantType:     "job",
			wantID:       "123",
		},
		{
			name:         "agents path",
			path:         "/api/agents/abc",
			wantType:     "agent",
			wantID:       "abc",
		},
		{
			name:         "short path",
			path:         "/api/jobs",
			wantType:     "unknown",
			wantID:       "",
		},
		{
			name:         "root path",
			path:         "/",
			wantType:     "unknown",
			wantID:       "",
		},
		{
			name:         "empty path",
			path:         "",
			wantType:     "unknown",
			wantID:       "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotType, gotID := extractResource(tt.path)
			if gotType != tt.wantType || gotID != tt.wantID {
				t.Errorf("extractResource() = (%v, %v), want (%v, %v)", gotType, gotID, tt.wantType, tt.wantID)
			}
		})
	}
}
