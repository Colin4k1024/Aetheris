package api

import (
	"bufio"
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

// sseAgentRequest 是 sse_legacy 协议的请求体，兼容 superagent-base /api/v1/chat/stream。
type sseAgentRequest struct {
	AgentID   string `json:"agent_id,omitempty"`
	SessionID string `json:"session_id,omitempty"`
	Message   string `json:"message"`
}

// consumeSSELegacyResponse 消费 SSE 流式响应（Legacy 模式），聚合 token 直到收到 [DONE]。
// 格式：每行 "data: <token>\n\n"，结束标记 "data: [DONE]\n\n"。
// 如果流在未收到 [DONE] 的情况下结束，则返回错误（协议错误）。
// 聚合内容超过 maxExternalResponseBytes 时返回错误。
func consumeSSELegacyResponse(body io.Reader) (string, error) {
	const doneMarker = "[DONE]"
	const maxLineBytes = 64 * 1024 // 64 KiB per SSE line
	var sb strings.Builder
	scanner := bufio.NewScanner(body)
	scanner.Buffer(make([]byte, maxLineBytes), maxLineBytes)
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "data:") {
			continue
		}
		token := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		if token == doneMarker {
			return sb.String(), nil
		}
		sb.WriteString(token)
		if int64(sb.Len()) > maxExternalResponseBytes {
			return "", fmt.Errorf("external_agent_call: SSE response exceeds %d MiB limit", maxExternalResponseBytes/(1024*1024))
		}
	}
	if err := scanner.Err(); err != nil {
		return "", fmt.Errorf("external_agent_call: read SSE stream: %w", err)
	}
	// 流结束但未遇到 [DONE] 标记，视为协议错误。
	return "", fmt.Errorf("external_agent_call: SSE stream ended without [DONE] sentinel")
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
		if config.IsExternalAgentType(agent.Type) {
			external := agent.External
			if external.Framework == "" {
				external.Framework = config.ExternalFramework(agent)
			}
			out[name] = external
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
			"framework": map[string]any{
				"type":        "string",
				"description": "Optional framework label supplied by the runtime plan, e.g. langchain or langgraph.",
			},
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
	protocol := strings.ToLower(strings.TrimSpace(agentCfg.Protocol))
	switch protocol {
	case "", "json", "sse_legacy":
		// valid
	default:
		return nil, fmt.Errorf("external_agent_call: unsupported protocol %q for agent %q; allowed values: \"\", \"json\", \"sse_legacy\"", protocol, agentID)
	}

	metadata := mapFromAny(input["metadata"])
	userMetadata := metadata // preserve user-provided metadata for protocol-specific checks
	if metadata == nil {
		metadata = make(map[string]any)
	}
	metadata["agent_id"] = agentID
	framework := strings.ToLower(strings.TrimSpace(agentCfg.Framework))
	if framework != "" {
		metadata["framework"] = framework
	}
	if jobID != "" {
		metadata["job_id"] = jobID
	}
	if idempotencyKey != "" {
		metadata["idempotency_key"] = idempotencyKey
	}

	// 根据协议类型构建请求体。
	var reqBody []byte
	switch protocol {
	case "sse_legacy":
		// sse_legacy 不转发 metadata；有用户输入的 metadata 时报错避免静默丢失。
		if len(userMetadata) > 0 {
			return nil, fmt.Errorf("external_agent_call: metadata is not forwarded in sse_legacy protocol")
		}
		// superagent-base 兼容格式：{"agent_id","session_id","message"}
		sseReq := sseAgentRequest{
			Message:   message,
			SessionID: sessionID,
		}
		if agentCfg.AgentID != "" {
			sseReq.AgentID = agentCfg.AgentID
		}
		var err error
		reqBody, err = json.Marshal(sseReq)
		if err != nil {
			return nil, fmt.Errorf("external_agent_call: encode sse request: %w", err)
		}
	default:
		// 默认 JSON 协议：{"message","session_id","metadata"}
		var err error
		reqBody, err = json.Marshal(externalAgentRequest{
			Message:   message,
			SessionID: sessionID,
			Metadata:  metadata,
		})
		if err != nil {
			return nil, fmt.Errorf("external_agent_call: encode request: %w", err)
		}
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, agentCfg.URL, bytes.NewReader(reqBody))
	if err != nil {
		return nil, fmt.Errorf("external_agent_call: create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if protocol == "sse_legacy" {
		req.Header.Set("Accept", "text/event-stream")
	}
	if idempotencyKey != "" {
		req.Header.Set("Idempotency-Key", idempotencyKey)
	}
	if jobID != "" {
		req.Header.Set("X-Aetheris-Job-ID", jobID)
	}
	req.Header.Set("X-Aetheris-Agent-ID", agentID)
	if framework != "" {
		req.Header.Set("X-Aetheris-Framework", framework)
	}
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

	// 根据协议类型解析响应。
	var out externalAgentResponse
	switch protocol {
	case "sse_legacy":
		// 消费 SSE 流，聚合 token 为最终答案。
		answer, err := consumeSSELegacyResponse(resp.Body)
		if err != nil {
			return nil, err
		}
		out = externalAgentResponse{Answer: answer, Final: true, Metadata: metadata}
	default:
		// 默认 JSON 响应解析。
		limitedBody, err := io.ReadAll(io.LimitReader(resp.Body, maxExternalResponseBytes+1))
		if err != nil {
			return nil, fmt.Errorf("external_agent_call: read response: %w", err)
		}
		if int64(len(limitedBody)) > maxExternalResponseBytes {
			return nil, fmt.Errorf("external_agent_call: response exceeds %d MiB limit", maxExternalResponseBytes/(1024*1024))
		}
		if err := json.Unmarshal(limitedBody, &out); err != nil {
			return nil, fmt.Errorf("external_agent_call: decode response: %w", err)
		}
		if !out.Final {
			return nil, fmt.Errorf("external_agent_call: upstream returned final=false; streaming is not supported")
		}
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
