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

package llm

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
)

// OllamaClient Ollama LLM 客户端
type OllamaClient struct {
	baseURL  string
	model    string
	httpClient *http.Client
}

// OllamaRequest Ollama API 请求
type OllamaRequest struct {
	Model    string   `json:"model"`
	Prompt   string   `json:"prompt"`
	Stream   bool    `json:"stream"`
	Options  *OllamaOptions `json:"options,omitempty"`
}

// OllamaOptions Ollama 选项
type OllamaOptions struct {
	Temperature  float64 `json:"temperature,omitempty"`
	TopP         float64 `json:"top_p,omitempty"`
	TopK         int     `json:"top_k,omitempty"`
	NumPredict  int     `json:"num_predict,omitempty"`
	Stop        []string `json:"stop,omitempty"`
}

// OllamaResponse Ollama API 响应
type OllamaResponse struct {
	Model     string `json:"model"`
	Response  string `json:"response"`
	Done      bool   `json:"done"`
	Context   []int  `json:"context,omitempty"`
	TotalDuration int64 `json:"total_duration,omitempty"`
	LoadDuration  int64 `json:"load_duration,omitempty"`
	PromptEvalCount int `json:"prompt_eval_count,omitempty"`
	EvalCount    int   `json:"eval_count,omitempty"`
}

// ChatRequest Ollama Chat 请求
type ChatRequest struct {
	Model    string   `json:"model"`
	Messages []ChatMessage `json:"messages"`
	Stream   bool    `json:"stream"`
	Options  *OllamaOptions `json:"options,omitempty"`
}

// ChatMessage Chat 消息
type ChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// ChatResponse Chat 响应
type ChatResponse struct {
	Model     string      `json:"model"`
	Message   ChatMessage `json:"message"`
	Done      bool        `json:"done"`
	TotalDuration int64   `json:"total_duration,omitempty"`
}

// NewOllamaClient 创建 Ollama 客户端
func NewOllamaClient(model string, baseURL string) (*OllamaClient, error) {
	if model == "" {
		model = "llama3"
	}
	if baseURL == "" {
		baseURL = os.Getenv("OLLAMA_BASE_URL")
		if baseURL == "" {
			baseURL = "http://localhost:11434"
		}
	}

	return &OllamaClient{
		baseURL:  strings.TrimSuffix(baseURL, "/"),
		model:    model,
		httpClient: &http.Client{
			Timeout: 5 * time.Minute, // Ollama 可能需要较长时间
		},
	}, nil
}

// Generate 生成文本 (简化版，使用 /api/generate)
func (c *OllamaClient) Generate(prompt string, options GenerateOptions) (string, error) {
	return c.GenerateWithContext(context.Background(), prompt, options)
}

// GenerateWithContext 使用上下文生成文本
func (c *OllamaClient) GenerateWithContext(ctx context.Context, prompt string, options GenerateOptions) (string, error) {
	req := OllamaRequest{
		Model:  c.model,
		Prompt: prompt,
		Stream: false,
	}

	if options.Temperature > 0 || options.MaxTokens > 0 || len(options.Stop) > 0 {
		req.Options = &OllamaOptions{
			Temperature: options.Temperature,
			NumPredict:  options.MaxTokens,
			Stop:        options.Stop,
		}
	}

	jsonData, err := json.Marshal(req)
	if err != nil {
		return "", fmt.Errorf("marshal request failed: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/api/generate", bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("create request failed: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("ollama returned status %d: %s", resp.StatusCode, string(body))
	}

	var result OllamaResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("decode response failed: %w", err)
	}

	return result.Response, nil
}

// Chat 聊天
func (c *OllamaClient) Chat(messages []Message, options GenerateOptions) (string, error) {
	return c.ChatWithContext(context.Background(), messages, options)
}

// ChatWithContext 使用上下文聊天
func (c *OllamaClient) ChatWithContext(ctx context.Context, messages []Message, options GenerateOptions) (string, error) {
	chatMessages := make([]ChatMessage, len(messages))
	for i, m := range messages {
		chatMessages[i] = ChatMessage{
			Role:    m.Role,
			Content: m.Content,
		}
	}

	req := ChatRequest{
		Model:   c.model,
		Messages: chatMessages,
		Stream:  false,
	}

	if options.Temperature > 0 || options.MaxTokens > 0 || len(options.Stop) > 0 {
		req.Options = &OllamaOptions{
			Temperature: options.Temperature,
			NumPredict:  options.MaxTokens,
			Stop:        options.Stop,
		}
	}

	jsonData, err := json.Marshal(req)
	if err != nil {
		return "", fmt.Errorf("marshal request failed: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/api/chat", bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("create request failed: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("ollama returned status %d: %s", resp.StatusCode, string(body))
	}

	var result ChatResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("decode response failed: %w", err)
	}

	return result.Message.Content, nil
}

// Model 返回模型名称
func (c *OllamaClient) Model() string {
	return c.model
}

// Provider 返回提供商名称
func (c *OllamaClient) Provider() string {
	return "ollama"
}

// SetModel 设置模型
func (c *OllamaClient) SetModel(model string) {
	if model != "" {
		c.model = model
	}
}

// SetAPIKey 设置 API Key (Ollama 不需要，但实现接口)
func (c *OllamaClient) SetAPIKey(apiKey string) {
	// Ollama 不需要 API Key
}

// Ensure OllamaClient 实现 Client 接口
var _ Client = (*OllamaClient)(nil)

// ListModelsResponse 列出模型响应
type ListModelsResponse struct {
	Models []ModelInfo `json:"models"`
}

// ModelInfo 模型信息
type ModelInfo struct {
	Name       string    `json:"name"`
	ModifiedAt time.Time `json:"modified_at"`
	Size       int64     `json:"size"`
	Digest     string    `json:"digest"`
}

// ListModels 列出可用模型
func (c *OllamaClient) ListModels(ctx context.Context) ([]ModelInfo, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", c.baseURL+"/api/tags", nil)
	if err != nil {
		return nil, fmt.Errorf("create request failed: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("ollama returned status %d", resp.StatusCode)
	}

	var result ListModelsResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode response failed: %w", err)
	}

	return result.Models, nil
}
