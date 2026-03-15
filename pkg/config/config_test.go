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

func TestMTLSConfig(t *testing.T) {
	cfg := MTLSConfig{
		Enabled:            true,
		CertFile:           "/path/to/cert",
		KeyFile:            "/path/to/key",
		CAFile:             "/path/to/ca",
		ClientCertFile:     "/path/to/client cert",
		ClientKeyFile:      "/path/to/client key",
		InsecureSkipVerify: true,
	}
	if !cfg.Enabled {
		t.Error("Expected Enabled to be true")
	}
	if cfg.CertFile != "/path/to/cert" {
		t.Errorf("Expected cert file path")
	}
}

func TestAPISigningConfig(t *testing.T) {
	cfg := APISigningConfig{
		Enabled:       true,
		Algorithm:     "HMAC-SHA256",
		ClockSkew:     "5m",
		RequiredPaths: []string{"/api/v1/agents"},
	}
	if !cfg.Enabled {
		t.Error("Expected Enabled to be true")
	}
	if cfg.Algorithm != "HMAC-SHA256" {
		t.Errorf("Expected HMAC-SHA256, got %s", cfg.Algorithm)
	}
}

func TestIPAllowListConfig(t *testing.T) {
	cfg := IPAllowListConfig{
		Enabled:        true,
		AllowIPs:       []string{"192.168.1.0/24"},
		BlockIPs:       []string{"10.0.0.0/8"},
		TrustedProxies: []string{"proxy.example.com"},
	}
	if !cfg.Enabled {
		t.Error("Expected Enabled to be true")
	}
	if len(cfg.AllowIPs) != 1 {
		t.Errorf("Expected 1 allow IP, got %d", len(cfg.AllowIPs))
	}
}

func TestSecretsConfig(t *testing.T) {
	cfg := SecretsConfig{
		Provider: "vault",
		Config:   map[string]string{"addr": "https://vault.example.com"},
	}
	if cfg.Provider != "vault" {
		t.Errorf("Expected vault, got %s", cfg.Provider)
	}
	if cfg.Config["addr"] != "https://vault.example.com" {
		t.Errorf("Expected vault address")
	}
}

func TestJobStoreConfig(t *testing.T) {
	cfg := JobStoreConfig{
		Type:          "postgres",
		DSN:           "postgres://user:pass@localhost/db",
		LeaseDuration: "30s",
	}
	if cfg.Type != "postgres" {
		t.Errorf("Expected postgres, got %s", cfg.Type)
	}
}

func TestEffectStoreConfig(t *testing.T) {
	cfg := EffectStoreConfig{
		Type: "postgres",
		DSN:  "postgres://user:pass@localhost/effects",
	}
	if cfg.Type != "postgres" {
		t.Errorf("Expected postgres, got %s", cfg.Type)
	}
}

func TestCheckpointStoreConfig(t *testing.T) {
	cfg := CheckpointStoreConfig{
		Type: "postgres",
		DSN:  "postgres://user:pass@localhost/checkpoints",
		TTL:  30,
	}
	if cfg.Type != "postgres" {
		t.Errorf("Expected postgres, got %s", cfg.Type)
	}
	if cfg.TTL != 30 {
		t.Errorf("Expected TTL 30, got %d", cfg.TTL)
	}
}

func TestCheckpointStoreConfig_ZeroTTL(t *testing.T) {
	cfg := CheckpointStoreConfig{
		Type: "memory",
		TTL:  0,
	}
	if cfg.TTL != 0 {
		t.Errorf("Expected 0, got %d", cfg.TTL)
	}
}

func TestAgentDefConfig(t *testing.T) {
	cfg := AgentDefConfig{
		Type:          "react",
		Description:   "A reactive agent",
		LLM:           "gpt-4",
		MaxIterations: 10,
		SystemPrompt:  "You are a helpful assistant",
		Tools:         []string{"web_search", "calculator"},
		ChainType:     "sequential",
		GraphType:     "dag",
		WorkflowType:  "linear",
	}
	if cfg.Type != "react" {
		t.Errorf("Expected react, got %s", cfg.Type)
	}
	if cfg.MaxIterations != 10 {
		t.Errorf("Expected 10, got %d", cfg.MaxIterations)
	}
	if len(cfg.Tools) != 2 {
		t.Errorf("Expected 2 tools, got %d", len(cfg.Tools))
	}
}

func TestAgentLLMConfig(t *testing.T) {
	cfg := AgentLLMConfig{
		Provider: "openai",
		Model:    "gpt-4",
		APIKey:   "sk-test",
	}
	if cfg.Provider != "openai" {
		t.Errorf("Expected openai, got %s", cfg.Provider)
	}
	if cfg.Model != "gpt-4" {
		t.Errorf("Expected gpt-4, got %s", cfg.Model)
	}
}

func TestToolsConfig(t *testing.T) {
	cfg := ToolsConfig{
		Enabled:     []string{"web_search", "calculator"},
		WebSearch:   WebSearchToolConfig{APIKey: "test-key", Engine: "google"},
		Calculator:  CalculatorToolConfig{Precision: 2},
		FileReader:  FileReaderToolConfig{AllowedPaths: []string{"/tmp"}},
		HTTPRequest: HTTPRequestToolConfig{Timeout: 30, MaxRetries: 3},
	}
	if len(cfg.Enabled) != 2 {
		t.Errorf("Expected 2 enabled tools, got %d", len(cfg.Enabled))
	}
	if cfg.WebSearch.APIKey != "test-key" {
		t.Errorf("Expected test-key, got %s", cfg.WebSearch.APIKey)
	}
}

func TestMCPConfig(t *testing.T) {
	cfg := MCPConfig{
		Servers: map[string]MCPServerConfig{
			"filesystem": {
				Type:    "stdio",
				Command: "npx",
				Args:    []string{"-y", "@modelcontextprotocol/server-filesystem", "/tmp"},
				Env:     map[string]string{"KEY": "value"},
				Dir:     "/home/user",
			},
		},
		InitTimeout: "30s",
	}
	if len(cfg.Servers) != 1 {
		t.Errorf("Expected 1 server, got %d", len(cfg.Servers))
	}
	if cfg.Servers["filesystem"].Type != "stdio" {
		t.Errorf("Expected stdio, got %s", cfg.Servers["filesystem"].Type)
	}
}

func TestMCPServerConfig(t *testing.T) {
	cfg := MCPServerConfig{
		Type:    "sse",
		URL:     "https://mcp.example.com/sse",
		Headers: map[string]string{"Authorization": "Bearer token"},
		Timeout: "60s",
	}
	if cfg.Type != "sse" {
		t.Errorf("Expected sse, got %s", cfg.Type)
	}
	if cfg.URL != "https://mcp.example.com/sse" {
		t.Errorf("Expected URL")
	}
}

func TestJobSchedulerConfig(t *testing.T) {
	enabled := true
	cfg := JobSchedulerConfig{
		Enabled:        &enabled,
		MaxConcurrency: 10,
		RetryMax:       5,
		Backoff:        "2s",
		Queues:         []string{"realtime", "default"},
	}
	if *cfg.Enabled != true {
		t.Error("Expected Enabled to be true")
	}
	if cfg.MaxConcurrency != 10 {
		t.Errorf("Expected 10, got %d", cfg.MaxConcurrency)
	}
}

func TestAPIConfig(t *testing.T) {
	cfg := APIConfig{
		Port:    8080,
		Host:    "0.0.0.0",
		Timeout: "30s",
	}
	if cfg.Port != 8080 {
		t.Errorf("Expected 8080, got %d", cfg.Port)
	}
	if cfg.Host != "0.0.0.0" {
		t.Errorf("Expected 0.0.0.0, got %s", cfg.Host)
	}
}

func TestForensicsConfig(t *testing.T) {
	cfg := ForensicsConfig{
		Experimental: true,
	}
	if !cfg.Experimental {
		t.Error("Expected Experimental to be true")
	}
}

func TestLoadConfig_FromFile(t *testing.T) {
	dir := t.TempDir()
	yaml := `
api:
  port: 9000
  host: "127.0.0.1"
log:
  level: "debug"
`
	path := filepath.Join(dir, "test.yaml")
	if err := os.WriteFile(path, []byte(yaml), 0644); err != nil {
		t.Fatalf("write temp config: %v", err)
	}
	cfg, err := LoadConfig(path)
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	if cfg.API.Port != 9000 {
		t.Errorf("API.Port: got %d", cfg.API.Port)
	}
	if cfg.API.Host != "127.0.0.1" {
		t.Errorf("API.Host: got %q", cfg.API.Host)
	}
	if cfg.Log.Level != "debug" {
		t.Errorf("Log.Level: got %q", cfg.Log.Level)
	}
}

func TestLoadConfig_InvalidPath(t *testing.T) {
	_, err := LoadConfig("/nonexistent/path/config.yaml")
	if err == nil {
		t.Error("expected error for invalid path")
	}
}

func TestDefaultDevConfig(t *testing.T) {
	cfg := DefaultDevConfig()
	if cfg == nil {
		t.Fatal("expected non-nil config")
	}
	if cfg.Runtime.Profile != "dev" {
		t.Errorf("expected dev profile, got %s", cfg.Runtime.Profile)
	}
	if cfg.API.Port != 8080 {
		t.Errorf("expected port 8080, got %d", cfg.API.Port)
	}
	if cfg.JobStore.Type != "memory" {
		t.Errorf("expected memory job store, got %s", cfg.JobStore.Type)
	}
	if cfg.EffectStore.Type != "memory" {
		t.Errorf("expected memory effect store, got %s", cfg.EffectStore.Type)
	}
	if cfg.CheckpointStore.Type != "memory" {
		t.Errorf("expected memory checkpoint store, got %s", cfg.CheckpointStore.Type)
	}
	if cfg.Storage.Vector.Type != "memory" {
		t.Errorf("expected memory vector store, got %s", cfg.Storage.Vector.Type)
	}
	if cfg.API.CORS.Enable != true {
		t.Error("expected CORS enabled")
	}
	if len(cfg.API.CORS.AllowOrigins) != 1 || cfg.API.CORS.AllowOrigins[0] != "*" {
		t.Error("expected allow all origins")
	}
}

func TestConfig_ValidateProductionMode(t *testing.T) {
	// Test production config with auth enabled
	cfg := &Config{
		Runtime: RuntimeConfig{
			Profile: "prod",
		},
		API: APIConfig{
			Port: 8080,
			Middleware: MiddlewareConfig{
				Auth:   true,
				JWTKey: "test-secret-key",
			},
		},
	}
	err := cfg.ValidateProductionMode()
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestConfig_ValidateProductionMode_MissingAuth(t *testing.T) {
	cfg := &Config{
		Runtime: RuntimeConfig{
			Profile: "prod",
		},
		API: APIConfig{
			Port: 8080,
			Middleware: MiddlewareConfig{
				Auth: false,
			},
		},
	}
	err := cfg.ValidateProductionMode()
	if err == nil {
		t.Error("expected error for missing auth in production mode")
	}
}

func TestConfig_ValidateProductionMode_MissingJWTKey(t *testing.T) {
	cfg := &Config{
		Runtime: RuntimeConfig{
			Profile: "prod",
		},
		API: APIConfig{
			Port: 8080,
			Middleware: MiddlewareConfig{
				Auth:   true,
				JWTKey: "",
			},
		},
	}
	err := cfg.ValidateProductionMode()
	if err == nil {
		t.Error("expected error for missing JWT key in production mode")
	}
}
