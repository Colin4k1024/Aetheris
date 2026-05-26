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

package routing

import (
	"fmt"
	"net/url"
	"time"

	"github.com/Colin4k1024/Aetheris/v2/pkg/config"
)

// NewAdvisorFromConfig constructs a RoutingAdvisor from cfg.
//
// Rules:
//   - If cfg.Enabled is false or cfg.Mode is empty/"noop", a NoOpAdvisor is
//     returned (configured with any noop overrides and the resolved policy).
//   - If cfg.Mode is "remote", an error is returned — not yet implemented.
//   - An unrecognised Mode or invalid FallbackPolicy also returns an error.
//
// The returned advisor is always replay-safe: NoOpAdvisor is deterministic and
// produces evidence events exactly like any remote implementation would.
func NewAdvisorFromConfig(cfg config.RoutingAdvisorConfig) (RoutingAdvisor, error) {
	policy, err := resolvePolicy(cfg.FallbackPolicy)
	if err != nil {
		return nil, err
	}

	if !cfg.Enabled {
		return buildNoOp(cfg.Noop, policy), nil
	}

	switch cfg.Mode {
	case "", "noop":
		return buildNoOp(cfg.Noop, policy), nil
	case "remote":
		return nil, fmt.Errorf("routing: mode %q is not yet implemented; use mode \"noop\" or omit mode", cfg.Mode)
	default:
		return nil, fmt.Errorf("routing: unknown mode %q; accepted values: noop, remote", cfg.Mode)
	}
}

// ValidateRoutingAdvisorConfig returns an error if any field combination in cfg
// is invalid. It performs structural and format checks only — no network calls.
func ValidateRoutingAdvisorConfig(cfg config.RoutingAdvisorConfig) error {
	if cfg.FallbackPolicy != "" {
		if _, err := resolvePolicy(cfg.FallbackPolicy); err != nil {
			return fmt.Errorf("routing_advisor.fallback_policy: %w", err)
		}
	}

	if cfg.Mode != "" && cfg.Mode != "noop" && cfg.Mode != "remote" {
		return fmt.Errorf("routing_advisor.mode: unknown value %q; accepted: noop, remote", cfg.Mode)
	}

	if cfg.Mode == "remote" {
		if err := validateRemoteAdvisorConfig(cfg.Remote); err != nil {
			return fmt.Errorf("routing_advisor.remote: %w", err)
		}
	}

	return nil
}

// resolvePolicy converts the config string to a FallbackPolicy, defaulting to
// FallbackOpen when the string is empty.
func resolvePolicy(s string) (FallbackPolicy, error) {
	if s == "" {
		return FallbackOpen, nil
	}
	p := FallbackPolicy(s)
	if err := ValidateFallbackPolicy(p); err != nil {
		return "", fmt.Errorf("routing: invalid fallback_policy %q: %w", s, err)
	}
	return p, nil
}

// validateRemoteAdvisorConfig checks that the remote endpoint is well-formed
// and that the timeout, if present, is a positive duration.
func validateRemoteAdvisorConfig(r config.RoutingAdvisorRemoteConfig) error {
	if r.Endpoint == "" {
		return fmt.Errorf("endpoint is required when mode is \"remote\"")
	}
	u, err := url.Parse(r.Endpoint)
	if err != nil || u.Scheme == "" || u.Host == "" {
		return fmt.Errorf("endpoint %q is not a valid URL", r.Endpoint)
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return fmt.Errorf("endpoint %q must use http or https scheme", r.Endpoint)
	}
	if r.Timeout != "" {
		d, err := time.ParseDuration(r.Timeout)
		if err != nil {
			return fmt.Errorf("timeout %q is not a valid duration: %w", r.Timeout, err)
		}
		if d <= 0 {
			return fmt.Errorf("timeout must be a positive duration, got %q", r.Timeout)
		}
	}
	return nil
}

// buildNoOp constructs a NoOpAdvisor with optional name/version overrides.
func buildNoOp(noopCfg config.RoutingAdvisorNoOpConfig, policy FallbackPolicy) *NoOpAdvisor {
	a := NewNoOpAdvisor()
	a.FallbackPolicy = policy
	if noopCfg.Name != "" {
		a.Name = noopCfg.Name
	}
	if noopCfg.Version != "" {
		a.Version = noopCfg.Version
	}
	return a
}
