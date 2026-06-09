package executor

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/cloudwego/eino/compose"

	"github.com/Colin4k1024/Aetheris/v2/internal/agent/planner"
	"github.com/Colin4k1024/Aetheris/v2/internal/agent/runtime"
)

const maxFrameworkCallableResponseBytes = 10 * 1024 * 1024

type FrameworkCallableNodeAdapter struct {
	CommandEventSink CommandEventSink
	Client           *http.Client
}

type frameworkCallableRequest struct {
	JobID        string         `json:"job_id,omitempty"`
	NodeID       string         `json:"node_id"`
	SessionID    string         `json:"session_id,omitempty"`
	Input        map[string]any `json:"input,omitempty"`
	PriorResults map[string]any `json:"prior_results,omitempty"`
	Metadata     map[string]any `json:"metadata,omitempty"`
}

type frameworkCallableResponse struct {
	Output   any            `json:"output,omitempty"`
	Final    bool           `json:"final,omitempty"`
	Metadata map[string]any `json:"metadata,omitempty"`
}

func (a *FrameworkCallableNodeAdapter) runNode(ctx context.Context, taskID string, cfg map[string]any, p *AgentDAGPayload) (*AgentDAGPayload, error) {
	baseURL := stringFromMap(cfg, "url")
	if baseURL == "" {
		return nil, fmt.Errorf("FrameworkCallableNodeAdapter: node %s missing url", taskID)
	}
	callableURL, err := frameworkNodeInvokeURL(baseURL, taskID)
	if err != nil {
		return nil, err
	}
	if cfg == nil {
		cfg = make(map[string]any)
	}
	jobID := JobIDFromContext(ctx)
	reqBody := frameworkCallableRequest{
		JobID:        jobID,
		NodeID:       taskID,
		SessionID:    p.SessionID,
		Input:        frameworkCallableInput(cfg, p),
		PriorResults: p.Results,
		Metadata: map[string]any{
			"framework":          stringFromMap(cfg, "framework"),
			"framework_agent_id": stringFromMap(cfg, "framework_agent_id"),
			"callable":           stringFromMap(cfg, "callable"),
		},
	}
	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("framework callable encode request: %w", err)
	}
	if a.CommandEventSink != nil && jobID != "" {
		_ = a.CommandEventSink.AppendCommandEmitted(ctx, jobID, taskID, taskID, "framework_callable", body)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, callableURL, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("framework callable create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if tokenEnv := stringFromMap(cfg, "token_env"); tokenEnv != "" {
		token := os.Getenv(tokenEnv)
		if token == "" {
			return nil, fmt.Errorf("framework callable token env %q is not set", tokenEnv)
		}
		req.Header.Set("Authorization", "Bearer "+token)
	}
	client := a.Client
	if client == nil {
		client = &http.Client{Timeout: 120 * time.Second}
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("framework callable request failed: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("framework callable upstream returned HTTP %d", resp.StatusCode)
	}
	respBody, err := io.ReadAll(io.LimitReader(resp.Body, maxFrameworkCallableResponseBytes+1))
	if err != nil {
		return nil, fmt.Errorf("framework callable read response: %w", err)
	}
	if int64(len(respBody)) > maxFrameworkCallableResponseBytes {
		return nil, fmt.Errorf("framework callable response exceeds %d MiB limit", maxFrameworkCallableResponseBytes/(1024*1024))
	}
	var out frameworkCallableResponse
	if err := json.Unmarshal(respBody, &out); err != nil {
		return nil, fmt.Errorf("framework callable decode response: %w", err)
	}
	result := map[string]any{
		"output":   out.Output,
		"metadata": out.Metadata,
	}
	if a.CommandEventSink != nil && jobID != "" {
		resultBytes, _ := json.Marshal(result)
		_ = a.CommandEventSink.AppendCommandCommitted(ctx, jobID, taskID, taskID, resultBytes, "")
	}
	if p.Results == nil {
		p.Results = make(map[string]any)
	}
	p.Results[taskID] = result
	return p, nil
}

func (a *FrameworkCallableNodeAdapter) ToDAGNode(task *planner.TaskNode, agent *runtime.Agent) (*compose.Lambda, error) {
	taskID, cfg := task.ID, task.Config
	if cfg == nil {
		cfg = make(map[string]any)
	}
	return compose.InvokableLambda[*AgentDAGPayload, *AgentDAGPayload](func(ctx context.Context, p *AgentDAGPayload) (*AgentDAGPayload, error) {
		return a.runNode(ctx, taskID, cfg, p)
	}), nil
}

func (a *FrameworkCallableNodeAdapter) ToNodeRunner(task *planner.TaskNode, agent *runtime.Agent) (NodeRunner, error) {
	taskID, cfg := task.ID, task.Config
	if cfg == nil {
		cfg = make(map[string]any)
	}
	return func(ctx context.Context, p *AgentDAGPayload) (*AgentDAGPayload, error) {
		return a.runNode(ctx, taskID, cfg, p)
	}, nil
}

func frameworkNodeInvokeURL(baseURL, nodeID string) (string, error) {
	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if baseURL == "" {
		return "", fmt.Errorf("framework callable url is required")
	}
	return baseURL + "/aetheris/nodes/" + nodeID + "/invoke", nil
}

func frameworkCallableInput(cfg map[string]any, p *AgentDAGPayload) map[string]any {
	input := make(map[string]any)
	for k, v := range cfg {
		switch k {
		case "url", "token_env", "framework", "framework_agent_id", "framework_node_id", "callable":
			continue
		default:
			input[k] = v
		}
	}
	if p != nil {
		if _, exists := input["goal"]; !exists {
			input["goal"] = p.Goal
		}
		if _, exists := input["message"]; !exists {
			input["message"] = p.Goal
		}
		if _, exists := input["agent_id"]; !exists {
			input["agent_id"] = p.AgentID
		}
		if _, exists := input["session_id"]; !exists {
			input["session_id"] = p.SessionID
		}
	}
	return input
}

func stringFromMap(m map[string]any, key string) string {
	if m == nil {
		return ""
	}
	if s, ok := m[key].(string); ok {
		return strings.TrimSpace(s)
	}
	return ""
}
