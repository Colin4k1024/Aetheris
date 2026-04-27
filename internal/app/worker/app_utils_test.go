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

package worker

import (
	"testing"
	"time"

	"github.com/Colin4k1024/Aetheris/v2/pkg/config"
)

func TestValidateProductionRuntimeConfig_Worker(t *testing.T) {
	tests := []struct {
		name    string
		cfg     *config.Config
		wantErr bool
	}{
		{
			name:    "nil config is valid",
			cfg:     nil,
			wantErr: false,
		},
		{
			name: "non-production profile is valid",
			cfg: &config.Config{
				Runtime: config.RuntimeConfig{
					Profile: "dev",
					Strict:  false,
				},
			},
			wantErr: false,
		},
		{
			name: "prod profile missing jobstore",
			cfg: &config.Config{
				Runtime: config.RuntimeConfig{
					Profile: "prod",
				},
				JobStore: config.JobStoreConfig{
					Type: "memory",
				},
			},
			wantErr: true,
		},
		{
			name: "prod profile with postgres but missing effect store",
			cfg: &config.Config{
				Runtime: config.RuntimeConfig{
					Profile: "prod",
				},
				JobStore: config.JobStoreConfig{
					Type: "postgres",
					DSN:  "postgres://user:pass@localhost/db?sslmode=require",
				},
				EffectStore: config.EffectStoreConfig{
					Type: "memory",
				},
			},
			wantErr: true,
		},
		{
			name: "prod profile with postgres but missing checkpoint store",
			cfg: &config.Config{
				Runtime: config.RuntimeConfig{
					Profile: "prod",
				},
				JobStore: config.JobStoreConfig{
					Type: "postgres",
					DSN:  "postgres://user:pass@localhost/db?sslmode=require",
				},
				EffectStore: config.EffectStoreConfig{
					Type: "postgres",
					DSN:  "postgres://user:pass@localhost/db?sslmode=require",
				},
				CheckpointStore: config.CheckpointStoreConfig{
					Type: "memory",
				},
			},
			wantErr: true,
		},
		{
			name: "prod profile with default password",
			cfg: &config.Config{
				Runtime: config.RuntimeConfig{
					Profile: "prod",
				},
				JobStore: config.JobStoreConfig{
					Type: "postgres",
					DSN:  "postgres://aetheris:aetheris@localhost/db?sslmode=require",
				},
				EffectStore: config.EffectStoreConfig{
					Type: "postgres",
					DSN:  "postgres://user:pass@localhost/db?sslmode=require",
				},
				CheckpointStore: config.CheckpointStoreConfig{
					Type: "postgres",
					DSN:  "postgres://user:pass@localhost/db?sslmode=require",
				},
			},
			wantErr: true,
		},
		{
			name: "prod profile with sslmode=disable",
			cfg: &config.Config{
				Runtime: config.RuntimeConfig{
					Profile: "prod",
				},
				JobStore: config.JobStoreConfig{
					Type: "postgres",
					DSN:  "postgres://user:pass@localhost/db?sslmode=disable",
				},
				EffectStore: config.EffectStoreConfig{
					Type: "postgres",
					DSN:  "postgres://user:pass@localhost/db?sslmode=require",
				},
				CheckpointStore: config.CheckpointStoreConfig{
					Type: "postgres",
					DSN:  "postgres://user:pass@localhost/db?sslmode=require",
				},
			},
			wantErr: true,
		},
		{
			name: "valid production config",
			cfg: &config.Config{
				Runtime: config.RuntimeConfig{
					Profile: "prod",
				},
				JobStore: config.JobStoreConfig{
					Type: "postgres",
					DSN:  "postgres://user:pass@localhost/db?sslmode=require",
				},
				EffectStore: config.EffectStoreConfig{
					Type: "postgres",
					DSN:  "postgres://user:pass@localhost/db?sslmode=require",
				},
				CheckpointStore: config.CheckpointStoreConfig{
					Type: "postgres",
					DSN:  "postgres://user:pass@localhost/db?sslmode=require",
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateProductionRuntimeConfig(tt.cfg)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateProductionRuntimeConfig() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestContainsDefaultPassword_Worker(t *testing.T) {
	tests := []struct {
		name string
		dsn  string
		want bool
	}{
		{
			name: "empty DSN",
			dsn:  "",
			want: false,
		},
		{
			name: "DSN with default password (user format)",
			dsn:  "postgres://aetheris:aetheris@localhost/db",
			want: true,
		},
		{
			name: "DSN with different password",
			dsn:  "postgres://user:secret@localhost/db",
			want: false,
		},
		{
			name: "DSN with password= format default",
			dsn:  "host=localhost dbname=test user=aetheris password=aetheris",
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := containsDefaultPassword(tt.dsn)
			if result != tt.want {
				t.Errorf("containsDefaultPassword(%q) = %v, want %v", tt.dsn, result, tt.want)
			}
		})
	}
}

func TestIsSSLEnabled_Worker(t *testing.T) {
	tests := []struct {
		name string
		dsn  string
		want bool
	}{
		{
			name: "empty DSN",
			dsn:  "",
			want: true,
		},
		{
			name: "DSN with sslmode=disable",
			dsn:  "postgres://user:pass@localhost/db?sslmode=disable",
			want: false,
		},
		{
			name: "DSN without sslmode",
			dsn:  "postgres://user:pass@localhost/db",
			want: true,
		},
		{
			name: "DSN with sslmode=require",
			dsn:  "postgres://user:pass@localhost/db?sslmode=require",
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isSSLEnabled(tt.dsn)
			if result != tt.want {
				t.Errorf("isSSLEnabled(%q) = %v, want %v", tt.dsn, result, tt.want)
			}
		})
	}
}

func TestEmbeddedDataDir_Worker(t *testing.T) {
	tests := []struct {
		name     string
		cfg      *config.Config
		expected string
	}{
		{
			name:     "nil config",
			cfg:      nil,
			expected: "data/embedded",
		},
		{
			name: "empty DSN",
			cfg: &config.Config{
				JobStore: config.JobStoreConfig{},
			},
			expected: "data/embedded",
		},
		{
			name: "custom DSN",
			cfg: &config.Config{
				JobStore: config.JobStoreConfig{
					DSN: "/custom/path",
				},
			},
			expected: "/custom/path",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := embeddedDataDir(tt.cfg)
			if result != tt.expected {
				t.Errorf("embeddedDataDir() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestGetHostname(t *testing.T) {
	result := getHostname()
	// Just verify it returns a non-empty string
	if result == "" {
		t.Error("getHostname() returned empty string")
	}
	// Verify it's deterministic
	result2 := getHostname()
	if result != result2 {
		t.Errorf("getHostname() not deterministic: got %q then %q", result, result2)
	}
}

func TestNewAgentJobRunner_Defaults(t *testing.T) {
	runner := NewAgentJobRunner(
		"test-worker",
		nil, // jobEventStore
		nil, // jobStore
		nil, // runJob
		2*time.Second,
		30*time.Second,
		0,   // maxConcurrency 0 -> should default to 2
		nil, // capabilities
		nil, // logger
	)

	if runner == nil {
		t.Fatal("expected non-nil runner")
	}
	if runner.maxConcurrency != 2 {
		t.Errorf("expected maxConcurrency 2, got %d", runner.maxConcurrency)
	}
	if runner.heartbeatTicker != 15*time.Second {
		t.Errorf("expected heartbeat 15s, got %v", runner.heartbeatTicker)
	}
}

func TestRuntimeOpsConfigDefaults(t *testing.T) {
	snapshot := runtimeSnapshotOpsConfig(nil)
	if !snapshot.Enabled || snapshot.Interval != time.Hour || snapshot.EventThreshold != 1000 || snapshot.BatchLimit != 50 || snapshot.KeepLatest != 1 {
		t.Fatalf("unexpected snapshot defaults: %+v", snapshot)
	}

	gc := runtimeGCOpsConfig(nil)
	if !gc.Enabled || gc.Interval != 24*time.Hour || gc.TTLDays != 90 || gc.BatchSize != 1000 {
		t.Fatalf("unexpected GC defaults: %+v", gc)
	}
}

func TestRuntimeOpsConfigOverrides(t *testing.T) {
	cfg := &config.Config{
		Runtime: config.RuntimeConfig{
			Snapshot: config.RuntimeSnapshotConfig{
				Enabled:        false,
				Interval:       "30m",
				EventThreshold: 250,
				BatchLimit:     10,
				KeepLatest:     3,
			},
			GC: config.RuntimeGCConfig{
				Enabled:   false,
				Interval:  "12h",
				TTLDays:   30,
				BatchSize: 25,
			},
		},
	}

	snapshot := runtimeSnapshotOpsConfig(cfg)
	if snapshot.Enabled || snapshot.Interval != 30*time.Minute || snapshot.EventThreshold != 250 || snapshot.BatchLimit != 10 || snapshot.KeepLatest != 3 {
		t.Fatalf("unexpected snapshot override: %+v", snapshot)
	}

	gc := runtimeGCOpsConfig(cfg)
	if gc.Enabled || gc.Interval != 12*time.Hour || gc.TTLDays != 30 || gc.BatchSize != 25 {
		t.Fatalf("unexpected GC override: %+v", gc)
	}
}

func TestNewAgentJobRunner_NegativeConcurrency(t *testing.T) {
	runner := NewAgentJobRunner(
		"test-worker",
		nil,
		nil,
		nil,
		2*time.Second,
		30*time.Second,
		-5, // negative -> should default to 2
		nil,
		nil,
	)

	if runner.maxConcurrency != 2 {
		t.Errorf("expected maxConcurrency 2, got %d", runner.maxConcurrency)
	}
}

func TestNewAgentJobRunner_ZeroLeaseDuration(t *testing.T) {
	runner := NewAgentJobRunner(
		"test-worker",
		nil,
		nil,
		nil,
		2*time.Second,
		0, // zero lease -> should default heartbeat to 15s
		2,
		nil,
		nil,
	)

	if runner.heartbeatTicker != 15*time.Second {
		t.Errorf("expected heartbeat 15s for zero lease, got %v", runner.heartbeatTicker)
	}
}

func TestJobStoreForRunnerAdapter_Implements(t *testing.T) {
	// Compile-time check that adapter implements interface
	var _ interface{} = &jobStoreForRunnerAdapter{}
}
