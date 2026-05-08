package api

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

	agentexec "github.com/Colin4k1024/Aetheris/v2/internal/agent/runtime/executor"
	"github.com/Colin4k1024/Aetheris/v2/internal/agent/tools"
	"github.com/Colin4k1024/Aetheris/v2/internal/runtime/session"
	"github.com/Colin4k1024/Aetheris/v2/pkg/config"
)

const ExternalAgentCallToolName = "external_agent_call"

// maxExternalResponseBytes caps the external agent response body to prevent memory exhaustion.
const maxExternalResponseBytes = 10 * 1024 * 1024 // 10 MiB

type externalAgentRequest struct {
	Message   string         `json:"message"`
	SessionID string         `json:"session_id,omitempty"`
	Metadata  map[string]any `json:"metadata,omitempty"`
}

type externalAgentResponse struct {
	Answer   string         `json:"answer"`
	Final    bool           `json:"final"`
	Metadata map[string]any `json:"metadata,omitempty"`
}

type externalAgentCallTool struct {
	agents map[string]config.AgentExternalConfig
	client *http.Client
}

func NewExternalAgentCallTool(agents map[string]config.AgentExternalConfig) tools.Tool {
	return &externalAgentCallTool{
		agents: agents,
		client: &http.Client{Timeout: 120 * time.Second},
	}
}

func collectExternalAgentConfigs(cfg *config.AgentsConfig) map[string]config.AgentExternalConfig {
	if cfg == nil {
		return nil
	}
	out := make(map[string]config.AgentExternalConfig)
	for name, agent := range cfg.Agents {
		if agent.Type == "external_http" {
			out[name] = agent.External
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func (t *externalAgentCallTool) Name() string {
	return ExternalAgentCallToolName
}

func (t *externalAgentCallTool) Description() string {
	return "Calls an existing HTTP agent through the Aetheris black-box integration protocol."
}

func (t *externalAgentCallTool) Schema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"agent_id": map[string]any{"type": "string", "description": "Configured external_http agent id."},
			"message":  map[string]any{"type": "string", "description": "User message or task goal."},
			"idempotency_key": map[string]any{
				"type":        "string",
				"description": "Optional upstream idempotency key; defaults to the runtime tool execution key.",
			},
			"metadata": map[string]any{"type": "object", "description": "Optional metadata forwarded to the external agent."},
		},
		"required": []any{"agent_id", "message"},
	}
}

func (t *externalAgentCallTool) RequiredCapability() string {
	return ExternalAgentCallToolName
}

func (t *externalAgentCallTool) Protocol() string {
	return "native"
}

func (t *externalAgentCallTool) Source() string {
	return "external_http"
}

func (t *externalAgentCallTool) Execute(ctx context.Context, sess *session.Session, input map[string]any, state interface{}) (any, error) {
	_ = state
	agentID := strings.TrimSpace(stringFromAny(input["agent_id"]))
	if agentID == "" {
		return nil, fmt.Errorf("external_agent_call: agent_id is required")
	}
	agentCfg, ok := t.agents[agentID]
	if !ok {
		return nil, fmt.Errorf("external_agent_call: unknown external agent %q", agentID)
	}
	message := stringFromAny(input["message"])
	if strings.TrimSpace(message) == "" {
		return nil, fmt.Errorf("external_agent_call: message is required")
	}

	idempotencyKey := strings.TrimSpace(stringFromAny(input["idempotency_key"]))
	if idempotencyKey == "" {
		idempotencyKey = agentexec.ExecutionKeyFromContext(ctx)
	}
	jobID := agentexec.JobIDFromContext(ctx)
	sessionID := ""
	if sess != nil {
		sessionID = sess.ID
	}
	metadata := mapFromAny(input["metadata"])
	if metadata == nil {
		metadata = make(map[string]any)
	}
	metadata["agent_id"] = agentID
	if jobID != "" {
		metadata["job_id"] = jobID
	}
	if idempotencyKey != "" {
		metadata["idempotency_key"] = idempotencyKey
	}

	reqBody, err := json.Marshal(externalAgentRequest{
		Message:   message,
		SessionID: sessionID,
		Metadata:  metadata,
	})
	if err != nil {
		return nil, fmt.Errorf("external_agent_call: encode request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, agentCfg.URL, bytes.NewReader(reqBody))
	if err != nil {
		return nil, fmt.Errorf("external_agent_call: create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if idempotencyKey != "" {
		req.Header.Set("Idempotency-Key", idempotencyKey)
	}
	if jobID != "" {
		req.Header.Set("X-Aetheris-Job-ID", jobID)
	}
	req.Header.Set("X-Aetheris-Agent-ID", agentID)
	if agentCfg.TokenEnv != "" {
		token := os.Getenv(agentCfg.TokenEnv)
		if token == "" {
			return nil, fmt.Errorf("external_agent_call: token env %q is not set", agentCfg.TokenEnv)
		}
		req.Header.Set("Authorization", "Bearer "+token)
	}

	client := t.client
	if client == nil {
		client = http.DefaultClient
	}
	if agentCfg.Timeout != "" {
		timeout, err := time.ParseDuration(agentCfg.Timeout)
		if err != nil {
			return nil, fmt.Errorf("external_agent_call: invalid timeout for %s: %w", agentID, err)
		}
		cloned := *client
		cloned.Timeout = timeout
		client = &cloned
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("external_agent_call: request failed: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("external_agent_call: upstream returned HTTP %d", resp.StatusCode)
	}

	// Read at most maxExternalResponseBytes+1 to detect over-size responses.
	limitedBody, err := io.ReadAll(io.LimitReader(resp.Body, maxExternalResponseBytes+1))
	if err != nil {
		return nil, fmt.Errorf("external_agent_call: read response: %w", err)
	}
	if int64(len(limitedBody)) > maxExternalResponseBytes {
		return nil, fmt.Errorf("external_agent_call: response exceeds %d MiB limit", maxExternalResponseBytes/(1024*1024))
	}
	var out externalAgentResponse
	if err := json.Unmarshal(limitedBody, &out); err != nil {
		return nil, fmt.Errorf("external_agent_call: decode response: %w", err)
	}
	if !out.Final {
		return nil, fmt.Errorf("external_agent_call: upstream returned final=false; streaming is not supported")
	}
	if out.Metadata == nil {
		out.Metadata = make(map[string]any)
	}
	outBytes, err := json.Marshal(out)
	if err != nil {
		return nil, fmt.Errorf("external_agent_call: encode response: %w", err)
	}
	return tools.ToolResult{Done: true, Output: string(outBytes)}, nil
}

func stringFromAny(v any) string {
	switch x := v.(type) {
	case string:
		return x
	case fmt.Stringer:
		return x.String()
	default:
		if x == nil {
			return ""
		}
		return fmt.Sprint(x)
	}
}

func mapFromAny(v any) map[string]any {
	if v == nil {
		return nil
	}
	if m, ok := v.(map[string]any); ok {
		out := make(map[string]any, len(m))
		for k, v := range m {
			out[k] = v
		}
		return out
	}
	return nil
}
