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

package tools

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/Colin4k1024/Aetheris/v2/internal/runtime/session"
)

const (
	// HermesDispatchToolName is the registered name of the Hermes ACP dispatch tool.
	HermesDispatchToolName = "hermes_dispatch"

	hermesDefaultACPTimeout = 120 * time.Second
	hermesACPDispatchPath   = "/acp/dispatch"
	// maxRespSize caps the Hermes response body to prevent OOM from large outputs.
	maxRespSize = 10 * 1024 * 1024 // 10 MB
)

// hermesDispatchRequest is the payload sent to the Hermes ACP endpoint.
type hermesDispatchRequest struct {
	// Task is a natural-language description of what Hermes should do.
	Task string `json:"task"`
	// Tools is an optional list of Hermes tool names to make available for this task.
	// Empty means Hermes uses its default tool set.
	Tools []string `json:"tools,omitempty"`
	// JobID is the Aetheris job ID for correlation in the Event Store and Hermes session tags.
	JobID string `json:"job_id,omitempty"`
	// Context carries arbitrary key-value metadata forwarded to Hermes.
	Context map[string]any `json:"context,omitempty"`
}

// hermesDispatchResponse is the response returned by the Hermes ACP endpoint.
type hermesDispatchResponse struct {
	// SessionID is the Hermes session ID created for this dispatch.
	SessionID string `json:"session_id,omitempty"`
	// Output is the final text output produced by Hermes.
	Output string `json:"output,omitempty"`
	// Error is a non-empty error message when Hermes reports a failure.
	Error string `json:"error,omitempty"`
	// Done indicates whether the task completed synchronously.
	Done bool `json:"done"`
}

// HermesDispatchTool dispatches a task to Hermes-Agent via the ACP HTTP endpoint.
//
// Phase 2 of the Hermes-Aetheris integration: Aetheris acts as the orchestration
// layer and Hermes-Agent handles AI coding, terminal execution, and multi-platform
// messaging. Each dispatch is tagged with the calling job ID so Hermes sessions can
// be correlated back to the Aetheris Event Store for auditing and replay.
type HermesDispatchTool struct {
	endpoint   string
	httpClient *http.Client
}

// HermesDispatchConfig configures a HermesDispatchTool.
type HermesDispatchConfig struct {
	// Endpoint is the Hermes ACP HTTP base URL, e.g. "http://localhost:8765".
	Endpoint string
	// Timeout is the HTTP timeout for dispatch calls. Defaults to 120s when zero.
	Timeout time.Duration
}

// NewHermesDispatchTool creates a HermesDispatchTool ready for registration.
// endpoint is the Hermes ACP base URL (e.g. "http://localhost:8765").
// timeout is the per-call HTTP timeout; pass 0 to use the default (120s).
func NewHermesDispatchTool(cfg HermesDispatchConfig) (*HermesDispatchTool, error) {
	if cfg.Endpoint == "" {
		return nil, fmt.Errorf("hermes dispatch tool: endpoint must not be empty")
	}
	timeout := cfg.Timeout
	if timeout <= 0 {
		timeout = hermesDefaultACPTimeout
	}
	return &HermesDispatchTool{
		endpoint: cfg.Endpoint,
		httpClient: &http.Client{
			Timeout: timeout,
		},
	}, nil
}

// Name implements Tool.
func (t *HermesDispatchTool) Name() string { return HermesDispatchToolName }

// Description implements Tool.
func (t *HermesDispatchTool) Description() string {
	return "Dispatch a coding, terminal, or messaging task to Hermes-Agent via the ACP protocol. " +
		"Hermes handles AI-native code execution, file operations, and multi-platform messaging. " +
		"The dispatch is tagged with the current job ID for full audit-trail correlation."
}

// Schema implements Tool.
func (t *HermesDispatchTool) Schema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"task": map[string]any{
				"type":        "string",
				"description": "Natural-language description of the task for Hermes to execute.",
			},
			"tools": map[string]any{
				"type": "array",
				"items": map[string]any{
					"type": "string",
				},
				"description": "Optional list of Hermes tool names to make available (e.g. [\"terminal\", \"file\", \"search_files\"]). Empty means Hermes uses its default tool set.",
			},
			"context": map[string]any{
				"type":                 "object",
				"additionalProperties": true,
				"description":          "Optional key-value metadata forwarded to Hermes (e.g. repo URL, PR number).",
			},
		},
		"required": []any{"task"},
	}
}

// Protocol implements ToolWithMetadata.
func (t *HermesDispatchTool) Protocol() string { return "acp" }

// Source implements ToolWithMetadata.
func (t *HermesDispatchTool) Source() string { return "hermes" }

// RequiredCapability implements ToolWithCapability.
func (t *HermesDispatchTool) RequiredCapability() string { return "hermes.dispatch" }

// Execute dispatches the task to Hermes and returns the result.
// The session's job ID (if available) is included in the request for correlation.
func (t *HermesDispatchTool) Execute(ctx context.Context, sess *session.Session, input map[string]any, state interface{}) (any, error) {
	task, _ := input["task"].(string)
	if task == "" {
		return nil, fmt.Errorf("hermes_dispatch: 'task' input is required")
	}

	req := hermesDispatchRequest{
		Task: task,
	}

	if sess != nil {
		req.JobID = sess.ID
	}

	// Accept both []any (from JSON/wrapped callers) and []string (from in-process callers).
	if tools, ok := input["tools"].([]any); ok {
		for _, tv := range tools {
			if s, ok := tv.(string); ok && s != "" {
				req.Tools = append(req.Tools, s)
			}
		}
	} else if tools, ok := input["tools"].([]string); ok {
		for _, s := range tools {
			if s != "" {
				req.Tools = append(req.Tools, s)
			}
		}
	}

	if ctxMap, ok := input["context"].(map[string]any); ok {
		req.Context = ctxMap
	}

	result, err := t.dispatch(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("hermes_dispatch: %w", err)
	}
	if result.Error != "" {
		return nil, fmt.Errorf("hermes_dispatch: remote error: %s", result.Error)
	}

	return &ToolResult{
		Done:   result.Done,
		Output: result.Output,
		Err:    result.Error,
		State: map[string]any{
			"session_id": result.SessionID,
			"source":     "hermes",
			"job_id":     req.JobID,
		},
	}, nil
}

// dispatch sends the request to the Hermes ACP endpoint and decodes the response.
func (t *HermesDispatchTool) dispatch(ctx context.Context, req hermesDispatchRequest) (*hermesDispatchResponse, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	url := strings.TrimSuffix(t.endpoint, "/") + hermesACPDispatchPath
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "application/json")

	resp, err := t.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("http call: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(&io.LimitedReader{R: resp.Body, N: maxRespSize + 1})
	if err != nil {
		return nil, fmt.Errorf("read response body: %w", err)
	}
	if len(respBody) > maxRespSize {
		return nil, fmt.Errorf("response body exceeds %d bytes (got %d); possible OOM risk", maxRespSize, len(respBody))
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("http %d: %s", resp.StatusCode, string(respBody))
	}

	var result hermesDispatchResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	return &result, nil
}
