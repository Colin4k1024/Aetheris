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

package mcp

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"
)

// ServerConfig describes how to connect to a single MCP server.
type ServerConfig struct {
	// Type is the transport type: "stdio" or "sse".
	Type string `mapstructure:"type"`
	// Command is the executable for stdio transport.
	Command string `mapstructure:"command"`
	// Args are the command arguments for stdio transport.
	Args []string `mapstructure:"args"`
	// Env are additional environment variables (KEY=VALUE) for stdio transport.
	Env map[string]string `mapstructure:"env"`
	// Dir is the working directory for stdio transport.
	Dir string `mapstructure:"dir"`
	// URL is the SSE endpoint URL for SSE transport.
	URL string `mapstructure:"url"`
	// Headers are optional HTTP headers for SSE transport (e.g. authorization).
	Headers map[string]string `mapstructure:"headers"`
	// Timeout is the per-call timeout. Default 60s.
	Timeout string `mapstructure:"timeout"`
}

// ManagerConfig configures the MCP Manager.
type ManagerConfig struct {
	// Servers maps server names to their configurations.
	Servers map[string]ServerConfig `mapstructure:"servers"`
	// InitTimeout is how long to wait for each server's initialize handshake.
	InitTimeout string `mapstructure:"init_timeout"`
}

// Manager manages the lifecycle of multiple MCP server connections.
// It implements the MCPInvoker interface expected by MCPHost.
type Manager struct {
	mu      sync.RWMutex
	clients map[string]*Client
	logger  *slog.Logger
}

// NewManager creates an empty MCP Manager.
func NewManager(logger *slog.Logger) *Manager {
	if logger == nil {
		logger = slog.Default()
	}
	return &Manager{
		clients: make(map[string]*Client),
		logger:  logger,
	}
}

// ConnectAll initializes connections to all configured MCP servers.
// Servers that fail to connect are logged and skipped, not fatal.
func (m *Manager) ConnectAll(ctx context.Context, cfg ManagerConfig) error {
	initTimeout := 30 * time.Second
	if cfg.InitTimeout != "" {
		if d, err := time.ParseDuration(cfg.InitTimeout); err == nil && d > 0 {
			initTimeout = d
		}
	}

	for name, scfg := range cfg.Servers {
		if err := m.connectOne(ctx, name, scfg, initTimeout); err != nil {
			m.logger.Warn("mcp server connection failed, skipping",
				"server", name, "error", err)
			continue
		}
		m.logger.Info("mcp server connected",
			"server", name,
			"transport", scfg.Type,
			"tools", len(m.clients[name].Tools()))
	}
	return nil
}

func (m *Manager) connectOne(ctx context.Context, name string, cfg ServerConfig, initTimeout time.Duration) error {
	callTimeout := 60 * time.Second
	if cfg.Timeout != "" {
		if d, err := time.ParseDuration(cfg.Timeout); err == nil && d > 0 {
			callTimeout = d
		}
	}

	var transport Transport
	switch strings.ToLower(cfg.Type) {
	case "stdio":
		envSlice := make([]string, 0, len(cfg.Env))
		for k, v := range cfg.Env {
			envSlice = append(envSlice, k+"="+v)
		}
		t, err := NewStdioTransport(ctx, StdioConfig{
			Command: cfg.Command,
			Args:    cfg.Args,
			Env:     envSlice,
			Dir:     cfg.Dir,
		})
		if err != nil {
			return fmt.Errorf("stdio transport: %w", err)
		}
		transport = t

	case "sse":
		t, err := NewSSETransport(ctx, SSEConfig{
			URL:        cfg.URL,
			HTTPClient: http.DefaultClient,
			Headers:    cfg.Headers,
		})
		if err != nil {
			return fmt.Errorf("sse transport: %w", err)
		}
		transport = t

	default:
		return fmt.Errorf("unknown transport type: %q (expected stdio or sse)", cfg.Type)
	}

	client := NewClient(ClientConfig{
		Name:        name,
		Transport:   transport,
		InitTimeout: initTimeout,
		CallTimeout: callTimeout,
	})

	if err := client.Initialize(ctx); err != nil {
		_ = transport.Close()
		return fmt.Errorf("initialize: %w", err)
	}

	m.mu.Lock()
	m.clients[name] = client
	m.mu.Unlock()
	return nil
}

// CallTool implements the MCPInvoker interface.
func (m *Manager) CallTool(ctx context.Context, server string, tool string, input map[string]any) (any, error) {
	m.mu.RLock()
	client, ok := m.clients[server]
	m.mu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("mcp server %q not connected", server)
	}

	result, err := client.CallTool(ctx, tool, input)
	if err != nil {
		return nil, err
	}

	// Convert MCP content blocks to a simple output map.
	return contentToOutput(result), nil
}

// ServerNames returns the names of all connected servers.
func (m *Manager) ServerNames() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	names := make([]string, 0, len(m.clients))
	for name := range m.clients {
		names = append(names, name)
	}
	return names
}

// ServerTools returns the discovered tools for a specific server.
func (m *Manager) ServerTools(name string) []MCPToolDef {
	m.mu.RLock()
	client, ok := m.clients[name]
	m.mu.RUnlock()
	if !ok {
		return nil
	}
	return client.Tools()
}

// AllTools returns all discovered tools across all connected servers,
// keyed by server name.
func (m *Manager) AllTools() map[string][]MCPToolDef {
	m.mu.RLock()
	defer m.mu.RUnlock()
	result := make(map[string][]MCPToolDef, len(m.clients))
	for name, client := range m.clients {
		result[name] = client.Tools()
	}
	return result
}

// Close shuts down all MCP server connections.
func (m *Manager) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for name, client := range m.clients {
		if err := client.Close(); err != nil {
			m.logger.Warn("mcp server close error", "server", name, "error", err)
		}
	}
	m.clients = make(map[string]*Client)
	return nil
}

// contentToOutput converts MCP content blocks to a simple output value.
func contentToOutput(result *ToolsCallResult) any {
	if result == nil || len(result.Content) == 0 {
		return nil
	}
	// Single text block: return as string.
	if len(result.Content) == 1 && result.Content[0].Type == "text" {
		return result.Content[0].Text
	}
	// Multiple blocks: return as list of maps.
	blocks := make([]map[string]any, len(result.Content))
	for i, b := range result.Content {
		block := map[string]any{"type": b.Type}
		if b.Text != "" {
			block["text"] = b.Text
		}
		if b.MimeType != "" {
			block["mimeType"] = b.MimeType
		}
		if b.Data != "" {
			block["data"] = b.Data
		}
		blocks[i] = block
	}
	return blocks
}
