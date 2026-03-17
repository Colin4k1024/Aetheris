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
	"sync"

	"rag-platform/internal/tool"
	"rag-platform/internal/tool/gatekeeper"
	"rag-platform/internal/tool/types"
)

// MCPProtocolVersion MCP 协议版本
const MCPProtocolVersion = "2024-11-05"

// JSONRPCVersion JSON-RPC 版本
const JSONRPCVersion = "2.0"

// Method types for MCP protocol
const (
	MethodInitialize       = "initialize"
	MethodToolsList        = "tools/list"
	MethodToolsCall        = "tools/call"
	MethodResourcesList    = "resources/list"
	MethodResourcesRead    = "resources/read"
	MethodResourcesSubscribe = "resources/subscribe"
	MethodPromptsList      = "prompts/list"
	MethodPromptsGet       = "prompts/get"
)

// ServerCapabilities MCP 服务器能力
type ServerCapabilities struct {
	Tools    *ToolsCapability    `json:"tools,omitempty"`
	Resources *ResourcesCapability `json:"resources,omitempty"`
	Prompts  *PromptsCapability   `json:"prompts,omitempty"`
}

// ToolsCapability 工具能力
type ToolsCapability struct {
	ListChanged bool `json:"listChanged,omitempty"`
}

// ResourcesCapability 资源能力
type ResourcesCapability struct {
	Subscribe bool `json:"subscribe,omitempty"`
	List      bool `json:"list,omitempty"`
}

// PromptsCapability 提示能力
type PromptsCapability struct {
	List bool `json:"list,omitempty"`
}

// InitializeResult 初始化结果
type InitializeResult struct {
	ProtocolVersion string             `json:"protocolVersion"`
	Capabilities   ServerCapabilities `json:"capabilities"`
	ServerInfo     ServerInfo         `json:"serverInfo"`
}

// ServerInfo 服务器信息
type ServerInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

// ToolListResult 工具列表结果
type ToolListResult struct {
	Tools []ToolDefinition `json:"tools"`
}

// ToolDefinition MCP 工具定义（转换为我们的工具系统）
type ToolDefinition struct {
	Name        string      `json:"name"`
	Description string      `json:"description"`
	InputSchema JSONSchema  `json:"inputSchema"`
}

// JSONSchema MCP JSON Schema
type JSONSchema struct {
	Type        string               `json:"type"`
	Properties  map[string]Property  `json:"properties,omitempty"`
	Required    []string             `json:"required,omitempty"`
	Description string               `json:"description,omitempty"`
}

// Property 属性定义
type Property struct {
	Type        string      `json:"type"`
	Description string      `json:"description,omitempty"`
	Enum        []any       `json:"enum,omitempty"`
	Items       *JSONSchema `json:"items,omitempty"`
	Properties  map[string]Property `json:"properties,omitempty"`
}

// ToolCallResult 工具调用结果
type ToolCallResult struct {
	Content []ContentBlock `json:"content"`
	IsError bool           `json:"isError,omitempty"`
}

// ContentBlock 内容块
type ContentBlock struct {
	Type     string `json:"type"`
	Text     string `json:"text,omitempty"`
	Resource string `json:"resource,omitempty"`
	MimeType string `json:"mimeType,omitempty"`
}

// MCPServer MCP 服务器实现
type MCPServer struct {
	mu           sync.RWMutex
	registry     *tool.Registry
	gatekeeper   *gatekeeper.Gatekeeper
	capabilities ServerCapabilities
	serverInfo   ServerInfo
}

// NewMCPServer 创建 MCP 服务器
func NewMCPServer(reg *tool.Registry, gk *gatekeeper.Gatekeeper) *MCPServer {
	return &MCPServer{
		registry: reg,
		gatekeeper: gk,
		capabilities: ServerCapabilities{
			Tools: &ToolsCapability{ListChanged: true},
			Resources: &ResourcesCapability{List: true, Subscribe: true},
			Prompts: &PromptsCapability{List: true},
		},
		serverInfo: ServerInfo{
			Name:    "CoRag MCP Server",
			Version: "1.0.0",
		},
	}
}

// Initialize 处理初始化请求
func (s *MCPServer) Initialize(ctx context.Context, params json.RawMessage) (interface{}, error) {
	var initParams struct {
		ProtocolVersion string `json:"protocolVersion"`
		Capabilities    struct {
			Tools    bool `json:"tools"`
			Resources bool `json:"resources"`
			Prompts  bool `json:"prompts"`
		} `json:"capabilities"`
	}

	if err := json.Unmarshal(params, &initParams); err != nil {
		return nil, fmt.Errorf("invalid initialize params: %w", err)
	}

	return InitializeResult{
		ProtocolVersion: MCPProtocolVersion,
		Capabilities:    s.capabilities,
		ServerInfo:      s.serverInfo,
	}, nil
}

// ListTools 返回所有工具列表
func (s *MCPServer) ListTools(ctx context.Context) (ToolListResult, error) {
	tools := s.registry.List()
	definitions := make([]ToolDefinition, 0, len(tools))

	for _, t := range tools {
		schema := t.Schema()
		definitions = append(definitions, ToolDefinition{
			Name:        t.Name(),
			Description: t.Description(),
			InputSchema: convertSchemaToMCP(schema),
		})
	}

	return ToolListResult{Tools: definitions}, nil
}

// CallTool 调用工具
func (s *MCPServer) CallTool(ctx context.Context, name string, arguments map[string]any) (ToolCallResult, error) {
	// 1. 获取工具
	t, ok := s.registry.Get(name)
	if !ok {
		return ToolCallResult{
			Content: []ContentBlock{{Type: "text", Text: fmt.Sprintf("tool not found: %s", name)}},
			IsError:  true,
		}, nil
	}

	// 2. Gatekeeper 参数验证
	if s.gatekeeper != nil {
		schema := t.Schema()
		if err := s.gatekeeper.Validate(name, arguments, schema); err != nil {
			return ToolCallResult{
				Content: []ContentBlock{{Type: "text", Text: fmt.Sprintf("validation error: %s", err.Error())}},
				IsError:  true,
			}, nil
		}
	}

	// 3. 执行工具
	result, err := t.Execute(ctx, arguments)
	if err != nil {
		return ToolCallResult{
			Content: []ContentBlock{{Type: "text", Text: fmt.Sprintf("execution error: %s", err.Error())}},
			IsError:  true,
		}, nil
	}

	// 4. 返回结果
	if result.Err != "" {
		return ToolCallResult{
			Content: []ContentBlock{{Type: "text", Text: result.Err}},
			IsError:  true,
		}, nil
	}

	return ToolCallResult{
		Content: []ContentBlock{{Type: "text", Text: result.Content}},
		IsError:  false,
	}, nil
}

// convertSchemaToMCP 将内部 Schema 转换为 MCP JSON Schema
func convertSchemaToMCP(schema tool.Schema) JSONSchema {
	mcpSchema := JSONSchema{
		Type:        schema.Type,
		Description: schema.Description,
		Required:    schema.Required,
		Properties:  make(map[string]Property),
	}

	for name, prop := range schema.Properties {
		mcpSchema.Properties[name] = Property{
			Type:        prop.Type,
			Description: prop.Description,
		}
	}

	return mcpSchema
}

// ConvertToolToMCP 将工具转换为 MCP 工具定义
func ConvertToolToMCP(t tool.Tool) ToolDefinition {
	schema := t.Schema()
	return ToolDefinition{
		Name:        t.Name(),
		Description: t.Description(),
		InputSchema: convertSchemaToMCP(schema),
	}
}

// RegisterMCPEndpoints 注册 MCP 协议端点（供 HTTP/WebSocket 适配器使用）
func (s *MCPServer) RegisterMCPEndpoints() []types.Endpoint {
	return []types.Endpoint{
		{Path: "/mcp/initialize", Method: "POST", Handler: s.handleInitialize},
		{Path: "/mcp/tools/list", Method: "POST", Handler: s.handleToolsList},
		{Path: "/mcp/tools/call", Method: "POST", Handler: s.handleToolsCall},
	}
}

func (s *MCPServer) handleInitialize(ctx context.Context, params json.RawMessage) (interface{}, error) {
	return s.Initialize(ctx, params)
}

func (s *MCPServer) handleToolsList(ctx context.Context, _ json.RawMessage) (interface{}, error) {
	return s.ListTools(ctx)
}

func (s *MCPServer) handleToolsCall(ctx context.Context, params json.RawMessage) (interface{}, error) {
	var req struct {
		Name      string         `json:"name"`
		Arguments map[string]any `json:"arguments"`
	}
	if err := json.Unmarshal(params, &req); err != nil {
		return nil, fmt.Errorf("invalid tool call params: %w", err)
	}
	return s.CallTool(ctx, req.Name, req.Arguments)
}
