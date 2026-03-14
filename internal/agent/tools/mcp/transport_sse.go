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
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
)

// SSETransport communicates with an MCP server via HTTP Server-Sent Events.
// The client POSTs JSON-RPC requests to the server's message endpoint and
// receives responses via the SSE event stream.
type SSETransport struct {
	baseURL    string
	httpClient *http.Client

	// messageEndpoint is discovered from the SSE stream's "endpoint" event.
	messageEndpoint string

	// respCh receives parsed JSON-RPC responses from the SSE reader goroutine.
	respCh chan *JSONRPCResponse

	cancel context.CancelFunc
	wg     sync.WaitGroup

	mu     sync.Mutex
	closed bool
}

// SSEConfig configures connection to an MCP server via SSE.
type SSEConfig struct {
	// URL is the SSE endpoint URL of the MCP server (e.g. http://localhost:3000/sse).
	URL string
	// HTTPClient is an optional custom HTTP client. If nil, http.DefaultClient is used.
	HTTPClient *http.Client
	// Headers are optional HTTP headers appended to all requests (e.g. Authorization).
	Headers map[string]string
}

// NewSSETransport connects to the MCP server's SSE endpoint and starts reading events.
func NewSSETransport(ctx context.Context, cfg SSEConfig) (*SSETransport, error) {
	if cfg.URL == "" {
		return nil, fmt.Errorf("mcp sse: url is required")
	}
	client := cfg.HTTPClient
	if client == nil {
		client = http.DefaultClient
	}

	ctx, cancel := context.WithCancel(ctx)

	t := &SSETransport{
		baseURL:    strings.TrimRight(cfg.URL, "/"),
		httpClient: client,
		respCh:     make(chan *JSONRPCResponse, 64),
		cancel:     cancel,
	}

	// Connect to the SSE stream to discover the message endpoint.
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, cfg.URL, nil)
	if err != nil {
		cancel()
		return nil, fmt.Errorf("mcp sse: create request: %w", err)
	}
	req.Header.Set("Accept", "text/event-stream")
	for k, v := range cfg.Headers {
		req.Header.Set(k, v)
	}

	resp, err := client.Do(req)
	if err != nil {
		cancel()
		return nil, fmt.Errorf("mcp sse: connect: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		cancel()
		return nil, fmt.Errorf("mcp sse: unexpected status %d", resp.StatusCode)
	}

	// Start SSE reader goroutine.
	t.wg.Add(1)
	go t.readSSE(ctx, resp.Body, cfg.Headers)

	// Wait for the endpoint event.
	// The first SSE event should be "endpoint" with the message URL.
	select {
	case <-ctx.Done():
		t.Close()
		return nil, ctx.Err()
	default:
	}

	return t, nil
}

// readSSE reads the SSE event stream and dispatches responses.
func (t *SSETransport) readSSE(ctx context.Context, body io.ReadCloser, headers map[string]string) {
	defer t.wg.Done()
	defer body.Close()

	scanner := bufio.NewScanner(body)
	scanner.Buffer(make([]byte, 0, 64*1024), 10*1024*1024)

	var eventType string
	var dataBuf bytes.Buffer

	for scanner.Scan() {
		select {
		case <-ctx.Done():
			return
		default:
		}

		line := scanner.Text()

		if line == "" {
			// End of event: dispatch.
			if eventType != "" || dataBuf.Len() > 0 {
				t.handleSSEEvent(eventType, dataBuf.String())
			}
			eventType = ""
			dataBuf.Reset()
			continue
		}

		if strings.HasPrefix(line, "event:") {
			eventType = strings.TrimSpace(strings.TrimPrefix(line, "event:"))
		} else if strings.HasPrefix(line, "data:") {
			data := strings.TrimPrefix(line, "data:")
			data = strings.TrimSpace(data)
			if dataBuf.Len() > 0 {
				dataBuf.WriteByte('\n')
			}
			dataBuf.WriteString(data)
		}
		// Ignore id:, retry:, and comment lines.
	}
}

func (t *SSETransport) handleSSEEvent(eventType, data string) {
	switch eventType {
	case "endpoint":
		t.mu.Lock()
		// The endpoint event data is a relative or absolute URL for posting messages.
		if strings.HasPrefix(data, "http://") || strings.HasPrefix(data, "https://") {
			t.messageEndpoint = data
		} else {
			// Relative path: resolve against baseURL.
			base := t.baseURL
			// Strip the path from baseURL to get the origin.
			if idx := strings.Index(base, "//"); idx >= 0 {
				rest := base[idx+2:]
				if slashIdx := strings.Index(rest, "/"); slashIdx >= 0 {
					base = base[:idx+2+slashIdx]
				}
			}
			t.messageEndpoint = base + "/" + strings.TrimPrefix(data, "/")
		}
		t.mu.Unlock()
	case "message":
		var resp JSONRPCResponse
		if err := json.Unmarshal([]byte(data), &resp); err != nil {
			return
		}
		select {
		case t.respCh <- &resp:
		default:
			// Drop if channel full — caller is too slow.
		}
	}
}

// Send posts a JSON-RPC request to the server's message endpoint.
func (t *SSETransport) Send(msg *JSONRPCRequest) error {
	t.mu.Lock()
	endpoint := t.messageEndpoint
	closed := t.closed
	t.mu.Unlock()

	if closed {
		return fmt.Errorf("mcp sse: transport closed")
	}
	if endpoint == "" {
		return fmt.Errorf("mcp sse: message endpoint not yet discovered")
	}

	data, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("mcp sse: marshal: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, endpoint, bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("mcp sse: create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := t.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("mcp sse: post: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return fmt.Errorf("mcp sse: server returned %d: %s", resp.StatusCode, string(body))
	}
	return nil
}

// Receive reads the next JSON-RPC response from the SSE stream.
func (t *SSETransport) Receive() (*JSONRPCResponse, error) {
	resp, ok := <-t.respCh
	if !ok {
		return nil, io.EOF
	}
	return resp, nil
}

// MessageEndpoint returns the discovered message endpoint URL.
func (t *SSETransport) MessageEndpoint() string {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.messageEndpoint
}

// Close shuts down the SSE connection.
func (t *SSETransport) Close() error {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.closed {
		return nil
	}
	t.closed = true
	t.cancel()
	t.wg.Wait()
	close(t.respCh)
	return nil
}
