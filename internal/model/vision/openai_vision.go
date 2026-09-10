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
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

// MaxImageSizeBytes limits the size of image data sent to the vision API.
const MaxImageSizeBytes = 20 * 1024 * 1024 // 20 MB

// AllowedImageTypes lists MIME types accepted by the vision adapter.
var AllowedImageTypes = map[string]bool{
	"image/jpeg": true,
	"image/png":  true,
	"image/gif":  true,
	"image/webp": true,
}

// OpenAIVisionClient implements the Client interface using the OpenAI Chat
// Completions API with vision support (gpt-4o, gpt-4-turbo, etc.).
type OpenAIVisionClient struct {
	model   string
	apiKey  string
	baseURL string
	client  *http.Client
}

// NewOpenAIVisionClient creates a vision client for the OpenAI-compatible API.
// apiKey must be non-empty; baseURL defaults to https://api.openai.com/v1 if empty.
func NewOpenAIVisionClient(model, apiKey, baseURL string) (*OpenAIVisionClient, error) {
	if apiKey == "" {
		return nil, fmt.Errorf("vision API key is required")
	}
	if model == "" {
		model = "gpt-4o"
	}
	if baseURL == "" {
		baseURL = "https://api.openai.com/v1"
		if envURL := os.Getenv("OPENAI_BASE_URL"); envURL != "" {
			baseURL = envURL
		}
	}
	return &OpenAIVisionClient{
		model:   model,
		apiKey:  apiKey,
		baseURL: baseURL,
		client:  &http.Client{Timeout: 60 * time.Second},
	}, nil
}

// Name returns the model name.
func (c *OpenAIVisionClient) Name() string {
	return c.model
}

// Describe sends an image to the vision model and returns a text description.
// The imageURLOrBase64 parameter can be:
//   - A URL (http:// or https://)
//   - A base64-encoded image string (optionally prefixed with data:image/...;base64,)
func (c *OpenAIVisionClient) Describe(ctx context.Context, imageURLOrBase64 string) (string, error) {
	if imageURLOrBase64 == "" {
		return "", fmt.Errorf("image input is required")
	}

	// Build the image content for the API request
	imageURL, err := c.resolveImageInput(imageURLOrBase64)
	if err != nil {
		return "", err
	}

	// Build the chat completion request with vision content
	requestBody := map[string]interface{}{
		"model": c.model,
		"messages": []map[string]interface{}{
			{
				"role": "user",
				"content": []map[string]interface{}{
					{
						"type": "text",
						"text": "Describe this image in detail.",
					},
					{
						"type":      "image_url",
						"image_url": map[string]string{"url": imageURL},
					},
				},
			},
		},
		"max_tokens": 300,
	}

	bodyBytes, err := json.Marshal(requestBody)
	if err != nil {
		return "", fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST",
		c.baseURL+"/chat/completions",
		strings.NewReader(string(bodyBytes)))
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("vision API request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("vision API error (status %d): %s", resp.StatusCode, string(respBody))
	}

	// Parse the response
	var result struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return "", fmt.Errorf("parse response: %w", err)
	}

	if len(result.Choices) == 0 {
		return "", fmt.Errorf("vision API returned no choices")
	}

	return result.Choices[0].Message.Content, nil
}

// resolveImageInput converts the input to a URL or data URI suitable for the API.
func (c *OpenAIVisionClient) resolveImageInput(input string) (string, error) {
	// If it's a URL, use directly
	if strings.HasPrefix(input, "http://") || strings.HasPrefix(input, "https://") {
		return input, nil
	}

	// If it's already a data URI
	if strings.HasPrefix(input, "data:image/") {
		// Validate the image type
		mimeEnd := strings.Index(input, ";")
		if mimeEnd > 5 {
			mimeType := input[5:mimeEnd]
			if !AllowedImageTypes[mimeType] {
				return "", fmt.Errorf("unsupported image type: %s", mimeType)
			}
		}
		// Validate size
		if len(input) > MaxImageSizeBytes {
			return "", fmt.Errorf("image exceeds max size %d bytes", MaxImageSizeBytes)
		}
		return input, nil
	}

	// Treat as raw base64 — validate and construct data URI
	decoded, err := base64.StdEncoding.DecodeString(input)
	if err != nil {
		return "", fmt.Errorf("invalid base64 image data: %w", err)
	}
	if len(decoded) > MaxImageSizeBytes {
		return "", fmt.Errorf("image exceeds max size %d bytes", MaxImageSizeBytes)
	}

	// Detect image type from magic bytes
	mimeType := detectImageType(decoded)
	if mimeType == "" {
		return "", fmt.Errorf("unsupported or unrecognized image format")
	}

	return "data:" + mimeType + ";base64," + input, nil
}

// detectImageType detects the image MIME type from magic bytes.
func detectImageType(data []byte) string {
	if len(data) < 4 {
		return ""
	}
	// JPEG: FF D8 FF
	if data[0] == 0xFF && data[1] == 0xD8 && data[2] == 0xFF {
		return "image/jpeg"
	}
	// PNG: 89 50 4E 47
	if data[0] == 0x89 && data[1] == 0x50 && data[2] == 0x4E && data[3] == 0x47 {
		return "image/png"
	}
	// GIF: 47 49 46 38
	if data[0] == 0x47 && data[1] == 0x49 && data[2] == 0x46 && data[3] == 0x38 {
		return "image/gif"
	}
	// WebP: 52 49 46 46 ... 57 45 42 50
	if len(data) >= 12 && data[0] == 0x52 && data[1] == 0x49 && data[2] == 0x46 && data[3] == 0x46 &&
		data[8] == 0x57 && data[9] == 0x45 && data[10] == 0x42 && data[11] == 0x50 {
		return "image/webp"
	}
	return ""
}
