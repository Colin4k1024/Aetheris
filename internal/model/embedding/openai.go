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

package embedding

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/go-resty/resty/v2"
)

// openaiEmbeddingRequest OpenAI embeddings API 请求体
type openaiEmbeddingRequest struct {
	Model string   `json:"model"`
	Input []string `json:"input"`
}

// openaiEmbeddingResponse OpenAI embeddings API 响应体
type openaiEmbeddingResponse struct {
	Data []struct {
		Embedding []float64 `json:"embedding"`
	} `json:"data"`
	Usage struct {
		PromptTokens int `json:"prompt_tokens"`
		TotalTokens  int `json:"total_tokens"`
	} `json:"usage"`
}

// OpenAIEmbedder 调用 OpenAI (或兼容) embeddings API 的真实 Embedder
type OpenAIEmbedder struct {
	embedderBase
	apiKey  string
	baseURL string
	client  *resty.Client
}

// NewOpenAIEmbedder 创建 OpenAI Embedding 客户端。
// baseURL 为空时使用 https://api.openai.com/v1 或 OPENAI_BASE_URL 环境变量。
// apiKey 为空时返回错误，禁止静默回退全零向量。
func NewOpenAIEmbedder(apiKey, model, baseURL string, dimension int) (*OpenAIEmbedder, error) {
	if apiKey == "" {
		return nil, fmt.Errorf("OpenAI embedding api_key not configured")
	}
	if model == "" {
		model = "text-embedding-3-small"
	}
	if dimension <= 0 {
		dimension = 1536
	}
	if baseURL == "" {
		baseURL = "https://api.openai.com/v1"
		if envURL := os.Getenv("OPENAI_BASE_URL"); envURL != "" {
			baseURL = envURL
		}
	}

	client := resty.New()
	client.SetTimeout(60 * time.Second)
	client.SetRetryCount(2)
	client.SetRetryWaitTime(1 * time.Second)
	client.SetRetryMaxWaitTime(5 * time.Second)

	return &OpenAIEmbedder{
		embedderBase: embedderBase{model: model, dimension: dimension},
		apiKey:       apiKey,
		baseURL:      baseURL,
		client:       client,
	}, nil
}

// Embed 调用 OpenAI embeddings API 返回向量
func (e *OpenAIEmbedder) Embed(ctx context.Context, texts []string) ([][]float64, error) {
	if e == nil || len(texts) == 0 {
		return nil, nil
	}

	reqBody := openaiEmbeddingRequest{
		Model: e.model,
		Input: texts,
	}

	var resp openaiEmbeddingResponse
	resp_, err := e.client.R().
		SetContext(ctx).
		SetHeader("Authorization", "Bearer "+e.apiKey).
		SetHeader("Content-Type", "application/json").
		SetBody(reqBody).
		SetResult(&resp).
		Post(e.baseURL + "/embeddings")

	if err != nil {
		return nil, fmt.Errorf("embedding API request failed: %w", err)
	}

	if resp_.StatusCode() != http.StatusOK {
		return nil, fmt.Errorf("embedding API returned status %d: %s", resp_.StatusCode(), resp_.String())
	}

	if len(resp.Data) != len(texts) {
		return nil, fmt.Errorf("embedding API returned %d vectors, expected %d", len(resp.Data), len(texts))
	}

	out := make([][]float64, len(resp.Data))
	for i, d := range resp.Data {
		if len(d.Embedding) == 0 {
			return nil, fmt.Errorf("embedding API returned empty vector at index %d", i)
		}
		out[i] = d.Embedding
	}

	return out, nil
}

// Ensure *OpenAIEmbedder implements Embedder
var _ Embedder = (*OpenAIEmbedder)(nil)

// --- OpenAIAdapter (legacy compat, delegates to OpenAIEmbedder) ---

// OpenAIAdapter 向后兼容的 OpenAI Embedding 适配器。
// 新代码应直接使用 NewOpenAIEmbedder。
type OpenAIAdapter struct {
	*OpenAIEmbedder
}

// NewOpenAIAdapter 创建 OpenAI Embedding 适配器（向后兼容）
func NewOpenAIAdapter(apiKey, model string, dimension int) (*OpenAIAdapter, error) {
	inner, err := NewOpenAIEmbedder(apiKey, model, "", dimension)
	if err != nil {
		return nil, err
	}
	return &OpenAIAdapter{OpenAIEmbedder: inner}, nil
}

// Ensure *OpenAIAdapter implements Embedder
var _ Embedder = (*OpenAIAdapter)(nil)
