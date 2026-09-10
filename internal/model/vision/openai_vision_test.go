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

package vision

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// --- OpenAIVisionClient Tests ---

func TestNewOpenAIVisionClient_Valid(t *testing.T) {
	client, err := NewOpenAIVisionClient("gpt-4o", "test-key", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if client.Name() != "gpt-4o" {
		t.Errorf("expected model 'gpt-4o', got %s", client.Name())
	}
}

func TestNewOpenAIVisionClient_DefaultModel(t *testing.T) {
	client, err := NewOpenAIVisionClient("", "test-key", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if client.Name() != "gpt-4o" {
		t.Errorf("expected default model 'gpt-4o', got %s", client.Name())
	}
}

func TestNewOpenAIVisionClient_MissingAPIKey(t *testing.T) {
	_, err := NewOpenAIVisionClient("gpt-4o", "", "")
	if err == nil {
		t.Error("expected error for missing API key")
	}
}

func TestOpenAIVisionClient_DescribeWithURL(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{
			"choices": [{"message": {"content": "A cat sitting on a table"}}]
		}`)
	}))
	defer server.Close()

	client, _ := NewOpenAIVisionClient("gpt-4o", "test-key", server.URL)
	result, err := client.Describe(context.Background(), "https://example.com/cat.jpg")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "A cat sitting on a table" {
		t.Errorf("expected description, got: %s", result)
	}
}

func TestOpenAIVisionClient_DescribeWithBase64(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{
			"choices": [{"message": {"content": "A landscape painting"}}]
		}`)
	}))
	defer server.Close()

	client, _ := NewOpenAIVisionClient("gpt-4o", "test-key", server.URL)

	// Create a minimal JPEG header to pass type detection
	jpegData := []byte{0xFF, 0xD8, 0xFF, 0xE0, 0x00, 0x10, 0x4A, 0x46, 0x49, 0x46}
	b64 := base64.StdEncoding.EncodeToString(jpegData)

	result, err := client.Describe(context.Background(), b64)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "A landscape painting" {
		t.Errorf("expected description, got: %s", result)
	}
}

func TestOpenAIVisionClient_DescribeWithDataURI(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, `{"choices": [{"message": {"content": "desc"}}]}`)
	}))
	defer server.Close()

	client, _ := NewOpenAIVisionClient("gpt-4o", "test-key", server.URL)
	result, err := client.Describe(context.Background(), "data:image/png;base64,iVBORwKG==")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "desc" {
		t.Errorf("expected 'desc', got: %s", result)
	}
}

func TestOpenAIVisionClient_EmptyInput(t *testing.T) {
	client, _ := NewOpenAIVisionClient("gpt-4o", "test-key", "")
	_, err := client.Describe(context.Background(), "")
	if err == nil {
		t.Error("expected error for empty input")
	}
}

func TestOpenAIVisionClient_InvalidBase64(t *testing.T) {
	client, _ := NewOpenAIVisionClient("gpt-4o", "test-key", "")
	_, err := client.Describe(context.Background(), "!!!not-valid-base64!!!")
	if err == nil {
		t.Error("expected error for invalid base64")
	}
}

func TestOpenAIVisionClient_UnsupportedImageType(t *testing.T) {
	client, _ := NewOpenAIVisionClient("gpt-4o", "test-key", "")
	// Data URI with unsupported type
	_, err := client.Describe(context.Background(), "data:image/bmp;base64,Qk0=")
	if err == nil {
		t.Error("expected error for unsupported image type")
	}
}

func TestOpenAIVisionClient_APIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		fmt.Fprintf(w, `{"error": "invalid api key"}`)
	}))
	defer server.Close()

	client, _ := NewOpenAIVisionClient("gpt-4o", "bad-key", server.URL)
	_, err := client.Describe(context.Background(), "https://example.com/img.jpg")
	if err == nil {
		t.Error("expected error for API failure")
	}
	if !strings.Contains(err.Error(), "401") {
		t.Errorf("expected status code in error, got: %v", err)
	}
}

func TestOpenAIVisionClient_NoChoices(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, `{"choices": []}`)
	}))
	defer server.Close()

	client, _ := NewOpenAIVisionClient("gpt-4o", "test-key", server.URL)
	_, err := client.Describe(context.Background(), "https://example.com/img.jpg")
	if err == nil {
		t.Error("expected error for no choices in response")
	}
}

func TestOpenAIVisionClient_CancelledContext(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Slow response
		select {
		case <-r.Context().Done():
			return
		}
	}))
	defer server.Close()

	client, _ := NewOpenAIVisionClient("gpt-4o", "test-key", server.URL)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := client.Describe(ctx, "https://example.com/img.jpg")
	if err == nil {
		t.Error("expected error for cancelled context")
	}
}

func TestDetectImageType_JPEG(t *testing.T) {
	data := []byte{0xFF, 0xD8, 0xFF, 0xE0}
	if mt := detectImageType(data); mt != "image/jpeg" {
		t.Errorf("expected image/jpeg, got %s", mt)
	}
}

func TestDetectImageType_PNG(t *testing.T) {
	data := []byte{0x89, 0x50, 0x4E, 0x47}
	if mt := detectImageType(data); mt != "image/png" {
		t.Errorf("expected image/png, got %s", mt)
	}
}

func TestDetectImageType_GIF(t *testing.T) {
	data := []byte{0x47, 0x49, 0x46, 0x38}
	if mt := detectImageType(data); mt != "image/gif" {
		t.Errorf("expected image/gif, got %s", mt)
	}
}

func TestDetectImageType_WebP(t *testing.T) {
	data := []byte{0x52, 0x49, 0x46, 0x46, 0x00, 0x00, 0x00, 0x00, 0x57, 0x45, 0x42, 0x50}
	if mt := detectImageType(data); mt != "image/webp" {
		t.Errorf("expected image/webp, got %s", mt)
	}
}

func TestDetectImageType_Unknown(t *testing.T) {
	data := []byte{0x00, 0x01, 0x02, 0x03}
	if mt := detectImageType(data); mt != "" {
		t.Errorf("expected empty for unknown, got %s", mt)
	}
}

func TestDetectImageType_TooSmall(t *testing.T) {
	data := []byte{0x00}
	if mt := detectImageType(data); mt != "" {
		t.Errorf("expected empty for too-small data, got %s", mt)
	}
}

// --- Factory Tests ---

func TestNewClientFromConfig_OpenAI(t *testing.T) {
	client, err := NewClientFromConfig(Config{
		Provider: "openai",
		APIKey:   "test-key",
		Model:    "gpt-4o",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := client.(*OpenAIVisionClient); !ok {
		t.Errorf("expected *OpenAIVisionClient, got %T", client)
	}
}

func TestNewClientFromConfig_Stub(t *testing.T) {
	client, err := NewClientFromConfig(Config{
		Provider: "stub",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := client.(*StubClient); !ok {
		t.Errorf("expected *StubClient, got %T", client)
	}
}

func TestNewClientFromConfig_Unknown(t *testing.T) {
	_, err := NewClientFromConfig(Config{
		Provider: "unknown",
	})
	if err == nil {
		t.Error("expected error for unknown provider")
	}
}

func TestNewClientFromConfig_OpenAI_MissingKey(t *testing.T) {
	_, err := NewClientFromConfig(Config{
		Provider: "openai",
		APIKey:   "",
	})
	if err == nil {
		t.Error("expected error for missing API key")
	}
}
