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

package api

import (
	"testing"
	"time"

	"rag-platform/pkg/config"
)

func TestParseDuration(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		defaultVal time.Duration
		expected   time.Duration
	}{
		{
			name:       "valid duration",
			input:      "5s",
			defaultVal: time.Second,
			expected:   5 * time.Second,
		},
		{
			name:       "valid minutes",
			input:      "2m",
			defaultVal: time.Second,
			expected:   2 * time.Minute,
		},
		{
			name:       "valid hours",
			input:      "1h",
			defaultVal: time.Second,
			expected:   time.Hour,
		},
		{
			name:       "empty string returns default",
			input:      "",
			defaultVal: 10 * time.Second,
			expected:   10 * time.Second,
		},
		{
			name:       "invalid format returns default",
			input:      "invalid",
			defaultVal: 5 * time.Second,
			expected:   5 * time.Second,
		},
		{
			name:       "negative duration is valid in Go",
			input:      "-1s",
			defaultVal: time.Second,
			expected:   -time.Second,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseDuration(tt.input, tt.defaultVal)
			if result != tt.expected {
				t.Errorf("parseDuration(%q, %v) = %v, want %v", tt.input, tt.defaultVal, result, tt.expected)
			}
		})
	}
}

func TestEmbeddedDataDir(t *testing.T) {
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

func TestContainsDefaultPassword(t *testing.T) {
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
			name: "DSN without default password (user format 1)",
			dsn:  "postgres://user:secret@localhost/db",
			want: false,
		},
		{
			name: "DSN with default password (user format 1)",
			dsn:  "postgres://aetheris:aetheris@localhost/db",
			want: true,
		},
		{
			name: "DSN with default password (user format 2)",
			dsn:  "postgres://aetheris:aetheris@localhost:5432/db?sslmode=disable",
			want: true,
		},
		{
			name: "DSN with default password (password= format)",
			dsn:  "host=localhost dbname=test user=aetheris password=aetheris",
			want: true,
		},
		{
			name: "DSN with different password",
			dsn:  "host=localhost dbname=test user=aetheris password=mysecret",
			want: false,
		},
		{
			name: "DSN with sslmode require",
			dsn:  "postgres://aetheris:aetheris@localhost/db?sslmode=require",
			want: true,
		},
		{
			name: "DSN with sslmode disable but still has default",
			dsn:  "postgres://aetheris:aetheris@localhost/db?sslmode=disable",
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

func TestIsSSLEnabled(t *testing.T) {
	tests := []struct {
		name string
		dsn  string
		want bool
	}{
		{
			name: "empty DSN",
			dsn:  "",
			want: true, // empty uses secure default
		},
		{
			name: "DSN without sslmode specified",
			dsn:  "postgres://user:pass@localhost/db",
			want: true, // defaults to secure
		},
		{
			name: "DSN with sslmode=disable",
			dsn:  "postgres://user:pass@localhost/db?sslmode=disable",
			want: false,
		},
		{
			name: "DSN with sslmode=require",
			dsn:  "postgres://user:pass@localhost/db?sslmode=require",
			want: true,
		},
		{
			name: "DSN with sslmode=verify-full",
			dsn:  "postgres://user:pass@localhost/db?sslmode=verify-full",
			want: true,
		},
		{
			name: "DSN with sslmode=allow",
			dsn:  "postgres://user:pass@localhost/db?sslmode=allow",
			want: true,
		},
		{
			name: "DSN with sslmode=prefer",
			dsn:  "postgres://user:pass@localhost/db?sslmode=prefer",
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

func TestValidateProductionRuntimeConfig(t *testing.T) {
	tests := []struct {
		name    string
		cfg     *config.Config
		wantErr bool
		errMsg  string
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
			name: "strict mode without production requirements",
			cfg: &config.Config{
				Runtime: config.RuntimeConfig{
					Strict: true,
				},
				JobStore: config.JobStoreConfig{
					Type: "memory",
				},
			},
			wantErr: true,
			errMsg:  "production requires jobstore.type=postgres",
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
			errMsg:  "production requires jobstore.type=postgres",
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
			errMsg:  "production requires effect_store.type=postgres",
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
			errMsg:  "production requires checkpoint_store.type=postgres",
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
			errMsg:  "production requires changing default passwords",
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
			errMsg:  "production requires SSL",
		},
		{
			name: "prod profile with CORS *",
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
				API: config.APIConfig{
					CORS: config.CORSConfig{
						AllowOrigins: []string{"*"},
					},
					Middleware: config.MiddlewareConfig{
						Auth: true,
					},
				},
			},
			wantErr: true,
			errMsg:  "production requires specific CORS origins",
		},
		{
			name: "prod profile with auth disabled",
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
				API: config.APIConfig{
					CORS: config.CORSConfig{
						AllowOrigins: []string{"https://example.com"},
					},
					Middleware: config.MiddlewareConfig{
						Auth: false,
					},
				},
			},
			wantErr: true,
			errMsg:  "production requires authentication",
		},
		{
			name: "prod profile with missing JWT key",
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
				API: config.APIConfig{
					CORS: config.CORSConfig{
						AllowOrigins: []string{"https://example.com"},
					},
					Middleware: config.MiddlewareConfig{
						Auth:   true,
						JWTKey: "",
					},
				},
			},
			wantErr: true,
			errMsg:  "production requires JWT key",
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
				API: config.APIConfig{
					CORS: config.CORSConfig{
						AllowOrigins: []string{"https://example.com"},
					},
					Middleware: config.MiddlewareConfig{
						Auth:   true,
						JWTKey: "my-secret-key",
					},
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
				return
			}
			if tt.wantErr && tt.errMsg != "" {
				if err == nil || !containsString(err.Error(), tt.errMsg) {
					t.Errorf("validateProductionRuntimeConfig() error = %v, want error containing %q", err, tt.errMsg)
				}
			}
		})
	}
}

func containsString(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsSubstring(s, substr))
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func TestGrpcRunGracefulStop(t *testing.T) {
	// Test that GracefulStop doesn't panic with nil fields
	g := &grpcRun{}
	g.GracefulStop() // Should not panic

	// Test with nil listener
	g = &grpcRun{srv: nil, lis: nil}
	g.GracefulStop() // Should not panic
}

func TestJobStoreForRunnerAdapter(t *testing.T) {
	// Test that adapter compiles correctly and implements interface
	var _ interface{} = &jobStoreForRunnerAdapter{}
}
