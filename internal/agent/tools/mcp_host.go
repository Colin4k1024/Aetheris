package tools

import (
	"context"
	"fmt"

	"rag-platform/internal/runtime/session"
)

// MCPInvoker bridges runtime tool calls to MCP servers.
type MCPInvoker interface {
	CallTool(ctx context.Context, server string, tool string, input map[string]any) (any, error)
}

// MCPHost manages MCP server registrations and tool adapters.
type MCPHost struct {
	invoker MCPInvoker
	servers map[string]struct{}
}

func NewMCPHost(invoker MCPInvoker) *MCPHost {
	return &MCPHost{
		invoker: invoker,
		servers: make(map[string]struct{}),
	}
}

func (h *MCPHost) RegisterServer(name string) {
	if name == "" {
		return
	}
	h.servers[name] = struct{}{}
}

func (h *MCPHost) HasServer(name string) bool {
	_, ok := h.servers[name]
	return ok
}

// RegisterToolAdapter exposes an MCP tool as a runtime Tool.
func (h *MCPHost) RegisterToolAdapter(reg *Registry, server string, manifest ToolManifest) error {
	if h.invoker == nil {
		return fmt.Errorf("mcp host invoker not configured")
	}
	if reg == nil {
		return fmt.Errorf("tool registry is nil")
	}
	if manifest.Name == "" {
		return fmt.Errorf("tool manifest name is required")
	}
	if !h.HasServer(server) {
		return fmt.Errorf("mcp server not registered: %s", server)
	}
	reg.Register(&mcpToolAdapter{
		server:   server,
		manifest: manifest,
		invoker:  h.invoker,
	})
	return nil
}

type mcpToolAdapter struct {
	server   string
	manifest ToolManifest
	invoker  MCPInvoker
}

func (a *mcpToolAdapter) Name() string           { return a.manifest.Name }
func (a *mcpToolAdapter) Description() string    { return a.manifest.Description }
func (a *mcpToolAdapter) Schema() map[string]any { return a.manifest.InputSchema }
func (a *mcpToolAdapter) RequiredCapability() string {
	return "mcp." + a.server + "." + a.manifest.Name
}
func (a *mcpToolAdapter) Protocol() string { return "mcp" }
func (a *mcpToolAdapter) Source() string   { return a.server }

func (a *mcpToolAdapter) Execute(ctx context.Context, sess *session.Session, input map[string]any, state interface{}) (any, error) {
	if a.invoker == nil {
		return nil, fmt.Errorf("mcp invoker is nil")
	}
	output, err := a.invoker.CallTool(ctx, a.server, a.manifest.Name, input)
	if err != nil {
		return nil, err
	}
	return map[string]any{
		"done":   true,
		"state":  state,
		"output": output,
		"mcp": map[string]any{
			"server": a.server,
			"tool":   a.manifest.Name,
		},
	}, nil
}
