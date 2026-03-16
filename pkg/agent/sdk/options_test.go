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

package sdk

import (
	"testing"
	"time"
)

func TestWithWaitTimeout(t *testing.T) {
	// Test default config
	cfg := &AgentConfig{}
	if cfg.WaitTimeout != 0 {
		t.Errorf("expected default WaitTimeout to be 0, got %v", cfg.WaitTimeout)
	}

	// Apply option
	timeout := 10 * time.Second
	opt := WithWaitTimeout(timeout)
	opt(cfg)

	if cfg.WaitTimeout != timeout {
		t.Errorf("expected WaitTimeout to be %v, got %v", timeout, cfg.WaitTimeout)
	}
}

func TestAgentConfig_Default(t *testing.T) {
	cfg := AgentConfig{}
	if cfg.WaitTimeout != 0 {
		t.Errorf("expected default WaitTimeout to be 0, got %v", cfg.WaitTimeout)
	}
}

func TestWithWaitTimeout_Multiple(t *testing.T) {
	cfg := &AgentConfig{}

	// Apply multiple options
	WithWaitTimeout(5 * time.Second)(cfg)
	if cfg.WaitTimeout != 5*time.Second {
		t.Errorf("expected 5s, got %v", cfg.WaitTimeout)
	}

	// Second option should override
	WithWaitTimeout(10 * time.Second)(cfg)
	if cfg.WaitTimeout != 10*time.Second {
		t.Errorf("expected 10s, got %v", cfg.WaitTimeout)
	}
}
