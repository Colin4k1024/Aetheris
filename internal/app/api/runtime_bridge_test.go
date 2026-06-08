package api

import (
	"context"
	"testing"

	agentexec "github.com/Colin4k1024/Aetheris/v2/internal/agent/runtime/executor"
	"github.com/Colin4k1024/Aetheris/v2/internal/agent/tools"
	apihttp "github.com/Colin4k1024/Aetheris/v2/internal/api/http"
	"github.com/Colin4k1024/Aetheris/v2/internal/runtime/session"
)

type countingRuntimeBridgeTool struct {
	count int
}

func (t *countingRuntimeBridgeTool) Name() string        { return "counting_tool" }
func (t *countingRuntimeBridgeTool) Description() string { return "counting tool" }
func (t *countingRuntimeBridgeTool) Schema() map[string]any {
	return map[string]any{"type": "object"}
}
func (t *countingRuntimeBridgeTool) Execute(ctx context.Context, sess *session.Session, input map[string]any, state interface{}) (any, error) {
	t.count++
	return tools.ToolResult{Done: true, Output: `{"ok":true}`}, nil
}

func TestRuntimeBridgeInvokeToolUsesLedger(t *testing.T) {
	reg := tools.NewRegistry()
	tool := &countingRuntimeBridgeTool{}
	reg.Register(tool)
	invoker := NewRuntimeBridgeInvoker(nil, reg, nil, agentexec.NewToolInvocationStoreMem(), nil, nil, nil)

	req := apihttp.RuntimeBridgeToolRequest{
		JobID:    "job-1",
		NodeID:   "node-1",
		ToolName: "counting_tool",
		Input:    map[string]any{"query": "same"},
	}
	if _, err := invoker.InvokeTool(context.Background(), req); err != nil {
		t.Fatalf("first InvokeTool returned error: %v", err)
	}
	if _, err := invoker.InvokeTool(context.Background(), req); err != nil {
		t.Fatalf("second InvokeTool returned error: %v", err)
	}
	if tool.count != 1 {
		t.Fatalf("expected tool to execute once, got %d", tool.count)
	}
}
