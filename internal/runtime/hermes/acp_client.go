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

package hermes

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const (
	// DefaultHermesACPUrl is the default URL for Hermes ACP Server.
	DefaultHermesACPUrl = "http://localhost:9090"

	// DefaultAetherisCallbackURL is the default callback URL for Aetheris.
	DefaultAetherisCallbackURL = "http://localhost:8080"

	// MaxRetries is the maximum number of retries for HTTP requests.
	MaxRetries = 3

	// InitialBackoff is the initial backoff duration.
	InitialBackoff = 1 * time.Second

	// MaxBackoff is the maximum backoff duration.
	MaxBackoff = 8 * time.Second
)

// ACPClient is the Go client for Hermes ACP Server.
type ACPClient struct {
	hermesURL           string       // Hermes ACP Server URL
	aetherisCallbackURL string       // Aetheris callback URL
	httpClient          *http.Client // HTTP client
}

// NewACPClient creates a new ACPClient for Hermes.
func NewACPClient(hermesURL string, aetherisCallbackURL string) *ACPClient {
	if hermesURL == "" {
		hermesURL = DefaultHermesACPUrl
	}
	if aetherisCallbackURL == "" {
		aetherisCallbackURL = DefaultAetherisCallbackURL
	}
	return &ACPClient{
		hermesURL:           hermesURL,
		aetherisCallbackURL: aetherisCallbackURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// DispatchJob sends a job.dispatch message to Hermes ACP Server.
// Returns the session_id on success.
func (c *ACPClient) DispatchJob(ctx context.Context, job *JobDispatch) (sessionID string, err error) {
	if job.Type == "" {
		job.Type = EventTypeJobDispatch
	}
	if job.CallbackURL == "" {
		job.CallbackURL = c.aetherisCallbackURL
	}

	payload, err := json.Marshal(job)
	if err != nil {
		return "", fmt.Errorf("failed to marshal job dispatch: %w", err)
	}

	url := fmt.Sprintf("%s/api/acp/jobs", c.hermesURL)
	resp, err := c.doPostWithRetry(ctx, url, payload)
	if err != nil {
		return "", fmt.Errorf("dispatch job failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("dispatch job failed with status %d: %s", resp.StatusCode, string(body))
	}

	var result struct {
		Status    string `json:"status"`
		SessionID string `json:"session_id"`
		Message   string `json:"message"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	if result.Status != "accepted" {
		return "", fmt.Errorf("job not accepted: %s", result.Message)
	}

	return result.SessionID, nil
}

// SendToolResult sends a tool result event to Aetheris callback.
func (c *ACPClient) SendToolResult(ctx context.Context, callID string, result string, idempotencyKey string) error {
	event := map[string]any{
		"type":            EventTypeToolResult,
		"call_id":         callID,
		"result":          result,
		"idempotency_key": idempotencyKey,
		"timestamp":       time.Now().UTC().Format(time.RFC3339),
	}

	payload, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal tool result: %w", err)
	}

	url := fmt.Sprintf("%s/api/acp/events", c.aetherisCallbackURL)
	resp, err := c.doPostWithRetry(ctx, url, payload)
	if err != nil {
		return fmt.Errorf("send tool result failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("send tool result failed with status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// SendCheckpoint saves a Hermes session checkpoint to Aetheris.
func (c *ACPClient) SendCheckpoint(ctx context.Context, sessionID string, checkpoint string) error {
	event := map[string]any{
		"type":       EventTypeCheckpointSave,
		"session_id": sessionID,
		"checkpoint": checkpoint,
		"timestamp":  time.Now().UTC().Format(time.RFC3339),
	}

	payload, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal checkpoint: %w", err)
	}

	url := fmt.Sprintf("%s/api/acp/checkpoints", c.aetherisCallbackURL)
	resp, err := c.doPostWithRetry(ctx, url, payload)
	if err != nil {
		return fmt.Errorf("send checkpoint failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("send checkpoint failed with status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// GetJobStatus queries the status of a job from Hermes.
func (c *ACPClient) GetJobStatus(ctx context.Context, jobID string) (*JobStatusResponse, error) {
	url := fmt.Sprintf("%s/api/acp/jobs/%s/status", c.hermesURL, jobID)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("get job status failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("get job status failed with status %d: %s", resp.StatusCode, string(body))
	}

	var result JobStatusResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &result, nil
}

// JobStatusResponse represents the response from job status query.
type JobStatusResponse struct {
	JobID      string `json:"job_id"`
	Status     string `json:"status"`
	IsCanceled bool   `json:"is_canceled"`
}

// doPostWithRetry performs an HTTP POST with exponential backoff retry.
func (c *ACPClient) doPostWithRetry(ctx context.Context, url string, payload []byte) (*http.Response, error) {
	backoff := InitialBackoff

	var lastErr error
	for attempt := 0; attempt <= MaxRetries; attempt++ {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(backoff):
			}
			backoff *= 2
			if backoff > MaxBackoff {
				backoff = MaxBackoff
			}
		}

		req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(payload))
		if err != nil {
			return nil, fmt.Errorf("failed to create request: %w", err)
		}
		req.Header.Set("Content-Type", "application/json")

		resp, err := c.httpClient.Do(req)
		if err != nil {
			lastErr = err
			continue
		}

		// Don't retry client errors (4xx)
		if resp.StatusCode >= 400 && resp.StatusCode < 500 {
			return resp, nil
		}

		// Retry server errors (5xx) or success
		if resp.StatusCode >= 500 {
			resp.Body.Close()
			lastErr = fmt.Errorf("server error: %d", resp.StatusCode)
			continue
		}

		return resp, nil
	}

	return nil, fmt.Errorf("max retries exceeded: %w", lastErr)
}
