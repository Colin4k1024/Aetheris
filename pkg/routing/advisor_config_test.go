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

package routing_test

import (
	"testing"

	"github.com/Colin4k1024/Aetheris/v2/pkg/config"
	"github.com/Colin4k1024/Aetheris/v2/pkg/routing"
)

// ---------------------------------------------------------------------------
// NewAdvisorFromConfig
// ---------------------------------------------------------------------------

func TestNewAdvisorFromConfig_Disabled_ReturnsNoOp(t *testing.T) {
	cfg := config.RoutingAdvisorConfig{Enabled: false}
	adv, err := routing.NewAdvisorFromConfig(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if adv == nil {
		t.Fatal("expected non-nil advisor")
	}
}

func TestNewAdvisorFromConfig_EmptyMode_ReturnsNoOp(t *testing.T) {
	cfg := config.RoutingAdvisorConfig{Enabled: true, Mode: ""}
	adv, err := routing.NewAdvisorFromConfig(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if adv == nil {
		t.Fatal("expected non-nil advisor")
	}
}

func TestNewAdvisorFromConfig_NoopMode_ReturnsNoOp(t *testing.T) {
	cfg := config.RoutingAdvisorConfig{
		Enabled:        true,
		Mode:           "noop",
		FallbackPolicy: string(routing.FallbackClosed),
	}
	adv, err := routing.NewAdvisorFromConfig(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if adv == nil {
		t.Fatal("expected non-nil advisor")
	}
}

func TestNewAdvisorFromConfig_NoopOverrides_Applied(t *testing.T) {
	cfg := config.RoutingAdvisorConfig{
		Enabled: true,
		Mode:    "noop",
		Noop: config.RoutingAdvisorNoOpConfig{
			Name:    "custom-advisor",
			Version: "v2",
		},
	}
	adv, err := routing.NewAdvisorFromConfig(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Decide a trivial request to inspect the returned decision's AdvisorInfo.
	req := routing.RouteDecisionRequest{
		SchemaVersion: routing.RouteDecisionRequestSchemaV1,
		JobID:         "j1",
		TenantID:      "t1",
		NodeID:        "n1",
		DecisionKey:   "j1:n1:s1",
		Candidates: []routing.RouteCandidate{
			{CapabilityID: "tool:x", Kind: routing.KindTool, Provider: "builtin", Rank: 1},
		},
	}
	decision, err := adv.Decide(t.Context(), req)
	if err != nil {
		t.Fatalf("Decide error: %v", err)
	}
	if decision.Advisor.Name != "custom-advisor" {
		t.Errorf("Advisor.Name = %q, want %q", decision.Advisor.Name, "custom-advisor")
	}
	if decision.Advisor.Version != "v2" {
		t.Errorf("Advisor.Version = %q, want %q", decision.Advisor.Version, "v2")
	}
}

func TestNewAdvisorFromConfig_RemoteMode_Error(t *testing.T) {
	cfg := config.RoutingAdvisorConfig{Enabled: true, Mode: "remote"}
	_, err := routing.NewAdvisorFromConfig(cfg)
	if err == nil {
		t.Fatal("expected error for remote mode, got nil")
	}
}

func TestNewAdvisorFromConfig_UnknownMode_Error(t *testing.T) {
	cfg := config.RoutingAdvisorConfig{Enabled: true, Mode: "wisepick"}
	_, err := routing.NewAdvisorFromConfig(cfg)
	if err == nil {
		t.Fatal("expected error for unknown mode, got nil")
	}
}

func TestNewAdvisorFromConfig_InvalidFallbackPolicy_Error(t *testing.T) {
	cfg := config.RoutingAdvisorConfig{
		Enabled:        true,
		FallbackPolicy: "totally_wrong",
	}
	_, err := routing.NewAdvisorFromConfig(cfg)
	if err == nil {
		t.Fatal("expected error for invalid fallback_policy, got nil")
	}
}

// ---------------------------------------------------------------------------
// ValidateRoutingAdvisorConfig
// ---------------------------------------------------------------------------

func TestValidateRoutingAdvisorConfig_Zero_OK(t *testing.T) {
	if err := routing.ValidateRoutingAdvisorConfig(config.RoutingAdvisorConfig{}); err != nil {
		t.Errorf("unexpected error on zero value: %v", err)
	}
}

func TestValidateRoutingAdvisorConfig_ValidPolicies(t *testing.T) {
	for _, p := range []string{"fail_open", "fail_closed", "cached_decision"} {
		cfg := config.RoutingAdvisorConfig{FallbackPolicy: p}
		if err := routing.ValidateRoutingAdvisorConfig(cfg); err != nil {
			t.Errorf("policy %q: unexpected error: %v", p, err)
		}
	}
}

func TestValidateRoutingAdvisorConfig_InvalidPolicy(t *testing.T) {
	cfg := config.RoutingAdvisorConfig{FallbackPolicy: "bad_value"}
	if err := routing.ValidateRoutingAdvisorConfig(cfg); err == nil {
		t.Error("expected error for bad fallback_policy")
	}
}

func TestValidateRoutingAdvisorConfig_RemoteMissingEndpoint(t *testing.T) {
	cfg := config.RoutingAdvisorConfig{Mode: "remote"}
	if err := routing.ValidateRoutingAdvisorConfig(cfg); err == nil {
		t.Error("expected error when remote endpoint is missing")
	}
}

func TestValidateRoutingAdvisorConfig_RemoteValidEndpoint(t *testing.T) {
	cfg := config.RoutingAdvisorConfig{
		Mode: "remote",
		Remote: config.RoutingAdvisorRemoteConfig{
			Endpoint: "https://advisor.example.com",
			Timeout:  "10s",
		},
	}
	if err := routing.ValidateRoutingAdvisorConfig(cfg); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestValidateRoutingAdvisorConfig_RemoteBadURL(t *testing.T) {
	cases := []string{"not-a-url", "ftp://example.com", ":bad"}
	for _, endpoint := range cases {
		cfg := config.RoutingAdvisorConfig{
			Mode:   "remote",
			Remote: config.RoutingAdvisorRemoteConfig{Endpoint: endpoint},
		}
		if err := routing.ValidateRoutingAdvisorConfig(cfg); err == nil {
			t.Errorf("expected error for endpoint %q, got nil", endpoint)
		}
	}
}

func TestValidateRoutingAdvisorConfig_RemoteBadTimeout(t *testing.T) {
	cfg := config.RoutingAdvisorConfig{
		Mode: "remote",
		Remote: config.RoutingAdvisorRemoteConfig{
			Endpoint: "https://advisor.example.com",
			Timeout:  "notaduration",
		},
	}
	if err := routing.ValidateRoutingAdvisorConfig(cfg); err == nil {
		t.Error("expected error for invalid timeout")
	}
}

func TestValidateRoutingAdvisorConfig_RemoteNegativeTimeout(t *testing.T) {
	cfg := config.RoutingAdvisorConfig{
		Mode: "remote",
		Remote: config.RoutingAdvisorRemoteConfig{
			Endpoint: "https://advisor.example.com",
			Timeout:  "-5s",
		},
	}
	if err := routing.ValidateRoutingAdvisorConfig(cfg); err == nil {
		t.Error("expected error for negative timeout")
	}
}

func TestValidateRoutingAdvisorConfig_UnknownMode(t *testing.T) {
	cfg := config.RoutingAdvisorConfig{Mode: "wisepick"}
	if err := routing.ValidateRoutingAdvisorConfig(cfg); err == nil {
		t.Error("expected error for unknown mode")
	}
}
