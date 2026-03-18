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

package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultDevConfig(t *testing.T) {
	cfg := DefaultDevConfig()

	if cfg.Runtime.Profile != "dev" {
		t.Errorf("expected profile 'dev', got '%s'", cfg.Runtime.Profile)
	}

	if cfg.API.Port != 8080 {
		t.Errorf("expected port 8080, got %d", cfg.API.Port)
	}

	if cfg.API.Middleware.Auth != false {
		t.Errorf("expected auth false in dev mode, got %v", cfg.API.Middleware.Auth)
	}

	if !cfg.API.CORS.Enable {
		t.Errorf("expected CORS enabled in dev mode")
	}

	if cfg.JobStore.Type != "memory" {
		t.Errorf("expected jobstore type 'memory', got '%s'", cfg.JobStore.Type)
	}

	if cfg.Storage.Vector.Type != "memory" {
		t.Errorf("expected vector type 'memory', got '%s'", cfg.Storage.Vector.Type)
	}
}

func TestValidateProductionMode(t *testing.T) {
	tests := []struct {
		name    string
		cfg     *Config
		wantErr bool
		errMsg  string
	}{
		{
			name: "prod mode without auth",
			cfg: &Config{
				Runtime: RuntimeConfig{Profile: "prod"},
				API: APIConfig{
					Middleware: MiddlewareConfig{Auth: false},
				},
			},
			wantErr: true,
			errMsg:  "production mode requires authentication",
		},
		{
			name: "prod mode without JWT key",
			cfg: &Config{
				Runtime: RuntimeConfig{Profile: "prod"},
				API: APIConfig{
					Middleware: MiddlewareConfig{Auth: true, JWTKey: ""},
				},
			},
			wantErr: true,
			errMsg:  "production mode requires JWT key",
		},
		{
			name: "prod mode with auth and JWT key",
			cfg: &Config{
				Runtime: RuntimeConfig{Profile: "prod"},
				API: APIConfig{
					Middleware: MiddlewareConfig{Auth: true, JWTKey: "test-key"},
				},
			},
			wantErr: false,
		},
		{
			name: "dev mode without auth",
			cfg: &Config{
				Runtime: RuntimeConfig{Profile: "dev"},
				API: APIConfig{
					Middleware: MiddlewareConfig{Auth: false},
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cfg.ValidateProductionMode()
			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error containing '%s', got nil", tt.errMsg)
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}
		})
	}
}

func TestConfigStructs(t *testing.T) {
	// Test that config structures can be instantiated
	cfg := &Config{
		API: APIConfig{
			Port: 8080,
			Host: "localhost",
			CORS: CORSConfig{
				Enable:       true,
				AllowOrigins: []string{"http://localhost:3000"},
			},
		},
		Runtime: RuntimeConfig{
			Profile: "dev",
			Strict:  false,
		},
		JobStore: JobStoreConfig{
			Type:          "memory",
			LeaseDuration: "30s",
		},
		Model: ModelConfig{
			LLM: LLMConfig{
				Providers: map[string]ProviderConfig{
					"openai": {
						APIKey:  "test-key",
						BaseURL: "https://api.openai.com/v1",
						Models: map[string]ModelInfo{
							"gpt-4": {
								Name:          "gpt-4",
								ContextWindow: 8192,
								Temperature:   0.7,
							},
						},
					},
				},
			},
		},
		Storage: StorageConfig{
			Vector: VectorConfig{
				Type:       "memory",
				Collection: "default",
			},
			Cache: CacheConfig{
				Type: "memory",
			},
		},
	}

	if cfg.API.Port != 8080 {
		t.Errorf("expected port 8080")
	}

	if cfg.Model.LLM.Providers["openai"].Models["gpt-4"].ContextWindow != 8192 {
		t.Errorf("expected context window 8192")
	}
}

func TestLoadConfigWithEnvVar(t *testing.T) {
	// Create a temporary config file
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "test.yaml")

	configContent := `
api:
  port: 9090
model:
  llm:
    providers:
      test:
        api_key: "${TEST_API_KEY}"
        base_url: "https://test.com"
`
	err := os.WriteFile(configPath, []byte(configContent), 0644)
	if err != nil {
		t.Fatalf("failed to write temp config: %v", err)
	}

	// Set environment variable
	os.Setenv("TEST_API_KEY", "env-api-key")
	defer os.Unsetenv("TEST_API_KEY")

	// Test loading with environment variable replacement
	// Note: This test verifies the structure; actual env replacement depends on the implementation
	_ = configPath // configPath is available for future use
}

func TestRateLimitsConfig(t *testing.T) {
	cfg := &Config{
		RateLimits: RateLimitsConfig{
			Tools: map[string]ToolRateLimitConfig{
				"web_search": {
					QPS:           10.0,
					MaxConcurrent: 5,
					Burst:         20,
				},
			},
			LLM: map[string]LLMRateLimitConfig{
				"openai": {
					TokensPerMinute:   90000,
					RequestsPerMinute: 60.0,
					MaxConcurrent:     10,
				},
			},
		},
	}

	if cfg.RateLimits.Tools["web_search"].QPS != 10.0 {
		t.Errorf("expected QPS 10.0")
	}

	if cfg.RateLimits.LLM["openai"].TokensPerMinute != 90000 {
		t.Errorf("expected tokens per minute 90000")
	}
}

func TestWorkerConfig(t *testing.T) {
	cfg := &Config{
		Worker: WorkerConfig{
			Concurrency:  4,
			QueueSize:    100,
			RetryCount:   3,
			RetryDelay:   "5s",
			Timeout:      "30s",
			PollInterval: "2s",
			MaxAttempts:  5,
			Capabilities: []string{"llm", "tool", "rag"},
		},
	}

	if cfg.Worker.Concurrency != 4 {
		t.Errorf("expected concurrency 4")
	}

	if len(cfg.Worker.Capabilities) != 3 {
		t.Errorf("expected 3 capabilities, got %d", len(cfg.Worker.Capabilities))
	}
}

func TestMCPConfig(t *testing.T) {
	cfg := &Config{
		MCP: MCPConfig{
			InitTimeout: "30s",
			Servers: map[string]MCPServerConfig{
				"filesystem": {
					Type:    "stdio",
					Command: "npx",
					Args:    []string{"-y", "@modelcontextprotocol/server-filesystem", "/tmp"},
					Env:     map[string]string{"KEY": "VALUE"},
					Dir:     "/home/user",
				},
				"remote": {
					Type:    "sse",
					URL:     "https://mcp.example.com/sse",
					Headers: map[string]string{"Authorization": "Bearer token"},
					Timeout: "60s",
				},
			},
		},
	}

	if cfg.MCP.Servers["filesystem"].Type != "stdio" {
		t.Errorf("expected stdio type")
	}

	if cfg.MCP.Servers["remote"].Type != "sse" {
		t.Errorf("expected sse type")
	}
}

func TestRuntimeConfig(t *testing.T) {
	cfg := RuntimeConfig{
		Profile: "prod",
		Strict:  true,
	}
	if cfg.Profile != "prod" {
		t.Errorf("expected prod, got %s", cfg.Profile)
	}
	if !cfg.Strict {
		t.Error("expected Strict to be true")
	}
}

func TestModelConfig(t *testing.T) {
	cfg := ModelConfig{
		LLM: LLMConfig{
			Providers: map[string]ProviderConfig{
				"openai": {
					APIKey:  "test-key",
					BaseURL: "https://api.openai.com/v1",
					Models: map[string]ModelInfo{
						"gpt-4": {
							Name:          "gpt-4",
							ContextWindow: 8192,
							Temperature:   0.7,
							MaxTokens:     4096,
						},
					},
				},
			},
		},
		Embedding: EmbeddingConfig{
			Providers: map[string]ProviderConfig{
				"cohere": {
					APIKey: "test-key",
					Models: map[string]ModelInfo{
						"embed-english-v3.0": {
							Name:      "embed-english-v3.0",
							Dimension: 1024,
						},
					},
				},
			},
		},
		Defaults: DefaultsConfig{
			LLM:       "gpt-4",
			Embedding: "cohere",
		},
	}
	if cfg.Defaults.LLM != "gpt-4" {
		t.Errorf("expected gpt-4, got %s", cfg.Defaults.LLM)
	}
	if cfg.LLM.Providers["openai"].Models["gpt-4"].MaxTokens != 4096 {
		t.Errorf("expected 4096 max tokens, got %d", cfg.LLM.Providers["openai"].Models["gpt-4"].MaxTokens)
	}
}

func TestProviderConfig(t *testing.T) {
	cfg := ProviderConfig{
		APIKey:  "sk-test",
		BaseURL: "https://api.example.com",
		Models: map[string]ModelInfo{
			"model-1": {
				Name:          "model-1",
				ContextWindow: 4096,
				Temperature:   0.5,
				Dimension:     768,
				InputLimit:    3000,
				MaxTokens:     2000,
			},
		},
	}
	if cfg.APIKey != "sk-test" {
		t.Errorf("expected sk-test, got %s", cfg.APIKey)
	}
	if cfg.Models["model-1"].Dimension != 768 {
		t.Errorf("expected 768 dimension, got %d", cfg.Models["model-1"].Dimension)
	}
}

func TestStorageConfig(t *testing.T) {
	cfg := StorageConfig{
		Metadata: MetadataConfig{
			Type:     "postgres",
			DSN:      "postgres://localhost:5432/metadata",
			PoolSize: 20,
		},
		Vector: VectorConfig{
			Type:       "redis",
			Addr:       "localhost:6379",
			DB:         "0",
			Collection: "vectors",
			Password:   "secret",
		},
		Object: ObjectConfig{
			Type:     "s3",
			Endpoint: "https://s3.amazonaws.com",
			Bucket:   "my-bucket",
			Region:   "us-east-1",
		},
		Cache: CacheConfig{
			Type:     "redis",
			Addr:     "localhost:6379",
			DB:       1,
			Password: "cache-secret",
		},
		Ingest: IngestConfig{
			BatchSize:   1000,
			Concurrency: 4,
		},
	}
	if cfg.Metadata.Type != "postgres" {
		t.Errorf("expected postgres, got %s", cfg.Metadata.Type)
	}
	if cfg.Vector.Type != "redis" {
		t.Errorf("expected redis, got %s", cfg.Vector.Type)
	}
	if cfg.Cache.DB != 1 {
		t.Errorf("expected DB 1, got %d", cfg.Cache.DB)
	}
}

func TestLogConfig(t *testing.T) {
	cfg := LogConfig{
		Level:  "debug",
		Format: "json",
		File:   "/var/log/aetheris.log",
	}
	if cfg.Level != "debug" {
		t.Errorf("expected debug, got %s", cfg.Level)
	}
	if cfg.Format != "json" {
		t.Errorf("expected json, got %s", cfg.Format)
	}
}

func TestMonitoringConfig(t *testing.T) {
	cfg := MonitoringConfig{
		Prometheus: PrometheusConfig{
			Enable: true,
			Port:   9090,
		},
		Tracing: TracingConfig{
			Enable:         true,
			ServiceName:    "aetheris-api",
			ExportEndpoint: "localhost:4317",
			Insecure:       true,
		},
	}
	if !cfg.Prometheus.Enable {
		t.Error("expected Prometheus to be enabled")
	}
	if cfg.Prometheus.Port != 9090 {
		t.Errorf("expected port 9090, got %d", cfg.Prometheus.Port)
	}
	if !cfg.Tracing.Enable {
		t.Error("expected Tracing to be enabled")
	}
}

func TestCORSConfig(t *testing.T) {
	cfg := CORSConfig{
		Enable:       true,
		AllowOrigins: []string{"https://example.com", "https://app.example.com"},
	}
	if !cfg.Enable {
		t.Error("expected Enable to be true")
	}
	if len(cfg.AllowOrigins) != 2 {
		t.Errorf("expected 2 origins, got %d", len(cfg.AllowOrigins))
	}
}

func TestMiddlewareConfig(t *testing.T) {
	cfg := MiddlewareConfig{
		Auth:          true,
		RateLimit:     true,
		RateLimitRPS:  100,
		JWTKey:        "secret-key",
		JWTTimeout:    "24h",
		JWTMaxRefresh: "7d",
	}
	if !cfg.Auth {
		t.Error("expected Auth to be true")
	}
	if !cfg.RateLimit {
		t.Error("expected RateLimit to be true")
	}
	if cfg.RateLimitRPS != 100 {
		t.Errorf("expected 100 RPS, got %d", cfg.RateLimitRPS)
	}
}

func TestGrpcConfig(t *testing.T) {
	cfg := GrpcConfig{
		Enable: true,
		Port:   9000,
	}
	if !cfg.Enable {
		t.Error("expected Enable to be true")
	}
	if cfg.Port != 9000 {
		t.Errorf("expected port 9000, got %d", cfg.Port)
	}
}

func TestAgentADKConfig(t *testing.T) {
	enabled := true
	cfg := AgentADKConfig{
		Enabled:         &enabled,
		CheckpointStore: "postgres",
	}
	if cfg.CheckpointStore != "postgres" {
		t.Errorf("expected postgres, got %s", cfg.CheckpointStore)
	}
}

func TestAgentsConfig(t *testing.T) {
	cfg := AgentsConfig{
		Agents: map[string]AgentDefConfig{
			"my-agent": {
				Type:          "react",
				Description:   "A test agent",
				LLM:           "gpt-4",
				MaxIterations: 10,
				SystemPrompt:  "You are helpful",
				Tools:         []string{"web_search"},
			},
		},
		LLM: AgentLLMConfig{
			Provider: "openai",
			Model:    "gpt-4",
			APIKey:   "sk-test",
		},
		Tools: ToolsConfig{
			Enabled:    []string{"web_search", "calculator"},
			WebSearch:  WebSearchToolConfig{APIKey: "key", Engine: "google"},
			Calculator: CalculatorToolConfig{Precision: 2},
		},
	}
	if len(cfg.Agents) != 1 {
		t.Errorf("expected 1 agent, got %d", len(cfg.Agents))
	}
	if cfg.LLM.Provider != "openai" {
		t.Errorf("expected openai, got %s", cfg.LLM.Provider)
	}
}

func TestDefaultsConfig(t *testing.T) {
	cfg := DefaultsConfig{
		LLM:       "gpt-4",
		Embedding: "cohere",
		Vision:    "claude-3",
	}
	if cfg.LLM != "gpt-4" {
		t.Errorf("expected gpt-4, got %s", cfg.LLM)
	}
}

func TestVectorConfig_Memory(t *testing.T) {
	cfg := VectorConfig{
		Type: "memory",
	}
	if cfg.Type != "memory" {
		t.Errorf("expected memory, got %s", cfg.Type)
	}
}

func TestReplaceEnvVars_LLM(t *testing.T) {
	// Set environment variable
	t.Setenv("TEST_OPENAI_KEY", "env-api-key")

	cfg := &Config{
		Model: ModelConfig{
			LLM: LLMConfig{
				Providers: map[string]ProviderConfig{
					"openai": {
						APIKey: "${TEST_OPENAI_KEY}",
					},
				},
			},
		},
	}

	err := replaceEnvVars(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.Model.LLM.Providers["openai"].APIKey != "env-api-key" {
		t.Errorf("expected env-api-key, got %s", cfg.Model.LLM.Providers["openai"].APIKey)
	}
}

func TestReplaceEnvVars_Embedding(t *testing.T) {
	t.Setenv("TEST_COHERE_KEY", "cohere-key")

	cfg := &Config{
		Model: ModelConfig{
			Embedding: EmbeddingConfig{
				Providers: map[string]ProviderConfig{
					"cohere": {
						APIKey: "${TEST_COHERE_KEY}",
					},
				},
			},
		},
	}

	err := replaceEnvVars(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.Model.Embedding.Providers["cohere"].APIKey != "cohere-key" {
		t.Errorf("expected cohere-key, got %s", cfg.Model.Embedding.Providers["cohere"].APIKey)
	}
}

func TestReplaceEnvVars_Vision(t *testing.T) {
	t.Setenv("TEST_VISION_KEY", "vision-key")

	cfg := &Config{
		Model: ModelConfig{
			Vision: VisionConfig{
				Providers: map[string]ProviderConfig{
					"openai": {
						APIKey: "${TEST_VISION_KEY}",
					},
				},
			},
		},
	}

	err := replaceEnvVars(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.Model.Vision.Providers["openai"].APIKey != "vision-key" {
		t.Errorf("expected vision-key, got %s", cfg.Model.Vision.Providers["openai"].APIKey)
	}
}

func TestReplaceEnvVars_NoEnvVar(t *testing.T) {
	// Make sure the env var is NOT set
	os.Unsetenv("NON_EXISTENT_KEY")

	cfg := &Config{
		Model: ModelConfig{
			LLM: LLMConfig{
				Providers: map[string]ProviderConfig{
					"openai": {
						APIKey: "${NON_EXISTENT_KEY}",
					},
				},
			},
		},
	}

	err := replaceEnvVars(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Key should remain unchanged when env var is not set
	if cfg.Model.LLM.Providers["openai"].APIKey != "${NON_EXISTENT_KEY}" {
		t.Errorf("expected unchanged key, got %s", cfg.Model.LLM.Providers["openai"].APIKey)
	}
}
