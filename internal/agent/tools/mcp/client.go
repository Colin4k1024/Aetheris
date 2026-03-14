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
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"sync"
	"time"
)

// Transport abstracts the underlying communication channel (stdio / SSE).
type Transport interface {
	Send(msg *JSONRPCRequest) error
	Receive() (*JSONRPCResponse, error)
	Close() error
}

// Client is a high-level MCP protocol client.
// It manages the lifecycle of one MCP server connection,
// including initialization, tool discovery, and tool calling.
type Client struct {
	name      string // server name (for logging / identification)
	transport Transport
	idGen     IDGenerator

	// Initialized server info.
	serverInfo *InitializeResult

	// Discovered tools (populated after Initialize + ListTools).
	mu    sync.RWMutex
	tools map[string]MCPToolDef

	initTimeout time.Duration
	callTimeout time.Duration
}

// ClientConfig configures an MCP client.
type ClientConfig struct {
	// Name identifies this MCP server for logging and tool namespacing.
	Name string
	// Transport is the underlying communication channel.
	Transport Transport
	// InitTimeout is how long to wait for the initialize handshake. Default 30s.
	InitTimeout time.Duration
	// CallTimeout is the default timeout for tool calls. Default 60s.
	CallTimeout time.Duration
}

// NewClient creates a new MCP client wrapping the given transport.
func NewClient(cfg ClientConfig) *Client {
	initTimeout := cfg.InitTimeout
	if initTimeout <= 0 {
		initTimeout = 30 * time.Second
	}
	callTimeout := cfg.CallTimeout
	if callTimeout <= 0 {
		callTimeout = 60 * time.Second
	}
	return &Client{
		name:        cfg.Name,
		transport:   cfg.Transport,
		tools:       make(map[string]MCPToolDef),
		initTimeout: initTimeout,
		callTimeout: callTimeout,
	}
}

// Initialize performs the MCP initialize handshake and tool discovery.
func (c *Client) Initialize(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, c.initTimeout)
	defer cancel()

	// 1. Send initialize request.
	params := InitializeParams{
		ProtocolVersion: ProtocolVersion,
		Capabilities:    ClientCapability{},
		ClientInfo: Implementation{
			Name:    "aetheris",
			Version: "2.0",
		},
	}
	result, err := c.call(ctx, MethodInitialize, params)
	if err != nil {
		return fmt.Errorf("mcp initialize: %w", err)
	}
	var initResult InitializeResult
	if err := json.Unmarshal(result, &initResult); err != nil {
		return fmt.Errorf("mcp initialize: unmarshal result: %w", err)
	}
	c.serverInfo = &initResult

	// 2. Send initialized notification.
	notif := &JSONRPCNotification{
		JSONRPC: "2.0",
		Method:  "notifications/initialized",
	}
	if st, ok := c.transport.(*StdioTransport); ok {
		_ = st.SendNotification(notif)
	}

	// 3. Discover tools.
	if err := c.discoverTools(ctx); err != nil {
		return fmt.Errorf("mcp discover tools: %w", err)
	}

	return nil
}

// discoverTools fetches all available tools from the MCP server via pagination.
func (c *Client) discoverTools(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	var cursor string
	for {
		params := ToolsListParams{Cursor: cursor}
		result, err := c.call(ctx, MethodToolsList, params)
		if err != nil {
			return err
		}
		var listResult ToolsListResult
		if err := json.Unmarshal(result, &listResult); err != nil {
			return fmt.Errorf("unmarshal tools/list: %w", err)
		}
		for _, tool := range listResult.Tools {
			c.tools[tool.Name] = tool
		}
		if listResult.NextCursor == "" {
			break
		}
		cursor = listResult.NextCursor
	}
	return nil
}

// Tools returns all discovered tools.
func (c *Client) Tools() []MCPToolDef {
	c.mu.RLock()
	defer c.mu.RUnlock()
	list := make([]MCPToolDef, 0, len(c.tools))
	for _, t := range c.tools {
		list = append(list, t)
	}
	return list
}

// CallTool invokes a tool on the MCP server and returns the result.
func (c *Client) CallTool(ctx context.Context, toolName string, arguments map[string]any) (*ToolsCallResult, error) {
	ctx, cancel := context.WithTimeout(ctx, c.callTimeout)
	defer cancel()

	params := ToolsCallParams{
		Name:      toolName,
		Arguments: arguments,
	}
	result, err := c.call(ctx, MethodToolsCall, params)
	if err != nil {
		return nil, err
	}
	var callResult ToolsCallResult
	if err := json.Unmarshal(result, &callResult); err != nil {
		return nil, fmt.Errorf("unmarshal tools/call result: %w", err)
	}
	if callResult.IsError {
		// Extract error text from content blocks.
		var errTexts []string
		for _, block := range callResult.Content {
			if block.Type == "text" && block.Text != "" {
				errTexts = append(errTexts, block.Text)
			}
		}
		return &callResult, fmt.Errorf("mcp tool %q returned error: %s", toolName, strings.Join(errTexts, "; "))
	}
	return &callResult, nil
}

// ServerInfo returns the server's initialize response (nil if not yet initialized).
func (c *Client) ServerInfo() *InitializeResult {
	return c.serverInfo
}

// Name returns the server name.
func (c *Client) Name() string {
	return c.name
}

// Close shuts down the client and underlying transport.
func (c *Client) Close() error {
	return c.transport.Close()
}

// call performs a synchronous JSON-RPC call: send request, then receive response.
func (c *Client) call(ctx context.Context, method string, params any) (json.RawMessage, error) {
	id := c.idGen.Next()
	req, err := newRequest(id, method, params)
	if err != nil {
		return nil, err
	}

	if err := c.transport.Send(req); err != nil {
		return nil, fmt.Errorf("send %s: %w", method, err)
	}

	// Read responses until we find one matching our ID.
	// NOTE: for stdio this is simple single-threaded read; for SSE, responses
	// come via channel. Non-matching responses are discarded.
	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		resp, err := c.transport.Receive()
		if err != nil {
			if err == io.EOF {
				return nil, fmt.Errorf("mcp server closed connection")
			}
			return nil, fmt.Errorf("receive %s: %w", method, err)
		}

		if resp.ID != id {
			// Not our response; could be a notification or out-of-order response.
			continue
		}

		if resp.Error != nil {
			return nil, resp.Error
		}
		return resp.Result, nil
	}
}
