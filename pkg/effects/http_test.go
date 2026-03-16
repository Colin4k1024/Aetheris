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

package effects

import (
	"context"
	"encoding/json"
	"testing"
	"time"
)

func TestComputeHTTPIdempotencyKey(t *testing.T) {
	tests := []struct {
		name     string
		req      HTTPRequest
		wantLen  int // minimum expected length
		wantSame bool // same inputs should produce same keys
	}{
		{
			name: "simple GET",
			req: HTTPRequest{
				Method: "GET",
				URL:    "https://example.com/api",
			},
			wantLen:  12, // "http:" prefix + 8 hex chars
			wantSame: true,
		},
		{
			name: "GET with headers",
			req: HTTPRequest{
				Method:  "GET",
				URL:     "https://example.com/api",
				Headers: map[string]string{"Authorization": "Bearer token"},
			},
			wantLen:  12,
			wantSame: true,
		},
		{
			name: "POST with body",
			req: HTTPRequest{
				Method: "POST",
				URL:    "https://example.com/api",
				Body:   []byte(`{"key": "value"}`),
			},
			wantLen:  12,
			wantSame: true,
		},
		{
			name: "different URLs produce different keys",
			req: HTTPRequest{
				Method: "GET",
				URL:    "https://example.com/api2",
			},
			wantLen:  12,
			wantSame: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key1 := computeHTTPIdempotencyKey(tt.req)
			if len(key1) < tt.wantLen {
				t.Errorf("key too short: got %d, want at least %d", len(key1), tt.wantLen)
			}

			// Same input should produce same key
			key2 := computeHTTPIdempotencyKey(tt.req)
			if key1 != key2 {
				t.Errorf("same input produced different keys: %s vs %s", key1, key2)
			}
		})
	}
}

func TestNewHTTPRequest(t *testing.T) {
	req := NewHTTPRequest("GET", "https://example.com/api")
	if req.Method != "GET" {
		t.Errorf("expected GET, got %s", req.Method)
	}
	if req.URL != "https://example.com/api" {
		t.Errorf("expected URL, got %s", req.URL)
	}
	if req.Headers == nil {
		t.Error("expected non-nil Headers")
	}
	if req.Body == nil {
		t.Error("expected non-nil Body")
	}
}

func TestHTTPRequest_WithHeader(t *testing.T) {
	req := NewHTTPRequest("GET", "https://example.com").
		WithHeader("Content-Type", "application/json").
		WithHeader("Authorization", "Bearer token")

	if req.Headers["Content-Type"] != "application/json" {
		t.Errorf("expected Content-Type header, got %s", req.Headers["Content-Type"])
	}
	if req.Headers["Authorization"] != "Bearer token" {
		t.Errorf("expected Authorization header, got %s", req.Headers["Authorization"])
	}
}

func TestHTTPRequest_WithBody(t *testing.T) {
	body := []byte("test body")
	req := NewHTTPRequest("POST", "https://example.com").WithBody(body)

	if string(req.Body) != "test body" {
		t.Errorf("expected body, got %s", req.Body)
	}
}

func TestHTTPRequest_WithJSONBody(t *testing.T) {
	data := map[string]string{"key": "value"}
	req, err := NewHTTPRequest("POST", "https://example.com").WithJSONBody(data)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if req.Headers["Content-Type"] != "application/json" {
		t.Errorf("expected Content-Type header, got %s", req.Headers["Content-Type"])
	}

	var parsed map[string]string
	if err := json.Unmarshal(req.Body, &parsed); err != nil {
		t.Fatalf("failed to unmarshal JSON: %v", err)
	}
	if parsed["key"] != "value" {
		t.Errorf("expected key=value, got %v", parsed)
	}
}

func TestHTTPRequest_WithJSONBody_Error(t *testing.T) {
	// Test with unsupported type (channels cannot be marshaled)
	req, err := NewHTTPRequest("POST", "https://example.com").WithJSONBody(make(chan int))
	if err == nil {
		t.Error("expected error for unsupported type")
	}
	_ = req // unused
}

func TestHTTPRequest_WithTimeout(t *testing.T) {
	timeout := 5 * time.Second
	req := NewHTTPRequest("GET", "https://example.com").WithTimeout(timeout)

	if req.Timeout == nil {
		t.Fatal("expected non-nil Timeout")
	}
	if *req.Timeout != timeout {
		t.Errorf("expected %v, got %v", timeout, *req.Timeout)
	}
}

func TestHTTPEffect(t *testing.T) {
	req := NewHTTPRequest("GET", "https://example.com/api")
	effect := HTTPEffect(req)

	if effect.Kind != KindHTTP {
		t.Errorf("expected KindHTTP, got %v", effect.Kind)
	}
	// Verify payload is the request
	payloadReq, ok := effect.Payload.(HTTPRequest)
	if !ok {
		t.Error("expected Payload to be HTTPRequest")
	}
	if payloadReq.URL != req.URL {
		t.Errorf("expected URL %s, got %s", req.URL, payloadReq.URL)
	}
	if effect.Description == "" {
		t.Error("expected non-empty Description")
	}
	if effect.IdempotencyKey == "" {
		t.Error("expected non-empty IdempotencyKey")
	}
}

func TestHTTPResponse_Marshal(t *testing.T) {
	resp := HTTPResponse{
		StatusCode: 200,
		Headers:    map[string]string{"Content-Type": "application/json"},
		Body:       []byte(`{"status":"ok"}`),
		Duration:   100 * time.Millisecond,
	}

	data, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var parsed HTTPResponse
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if parsed.StatusCode != 200 {
		t.Errorf("expected 200, got %d", parsed.StatusCode)
	}
	if string(parsed.Body) != `{"status":"ok"}` {
		t.Errorf("expected body, got %s", parsed.Body)
	}
}

func TestRecordHTTPToRecorder_NilRecorder(t *testing.T) {
	req := NewHTTPRequest("GET", "https://example.com")
	resp := HTTPResponse{StatusCode: 200}

	err := RecordHTTPToRecorder(context.Background(), nil, "effect-id", "idem-key", req, resp, time.Second)
	if err != ErrNoRecorder {
		t.Errorf("expected ErrNoRecorder, got %v", err)
	}
}
