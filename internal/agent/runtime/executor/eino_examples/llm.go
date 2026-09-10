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

package eino_examples

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

	"github.com/cloudwego/eino/schema"

	"github.com/Colin4k1024/Aetheris/v2/internal/model/llm"
	"github.com/Colin4k1024/Aetheris/v2/pkg/metrics"
)

// OllamaChatModel 将 llm.Client 转换为 eino_examples.ChatModel 接口
type OllamaChatModel struct {
	client  *llm.OllamaClient
	options Options
}

// NewOllamaChatModel 创建 Ollama ChatModel
func NewOllamaChatModel(client *llm.OllamaClient, opts ...Option) *OllamaChatModel {
	options := Options{}
	for _, opt := range opts {
		opt(&options)
	}
	return &OllamaChatModel{
		client:  client,
		options: options,
	}
}

// NewOllamaChatModelFromEnv 从环境变量创建 Ollama ChatModel
func NewOllamaChatModelFromEnv() (*OllamaChatModel, error) {
	modelName := os.Getenv("OLLAMA_MODEL")
	if modelName == "" {
		modelName = "llama3"
	}

	baseURL := os.Getenv("OLLAMA_BASE_URL")
	if baseURL == "" {
		baseURL = "http://localhost:11434"
	}

	// 探测 Ollama 服务是否可用，不可用时快速返回 error 以便测试 Skip
	probe := &http.Client{Timeout: 2 * time.Second}
	if _, err := probe.Get(baseURL + "/api/tags"); err != nil {
		return nil, fmt.Errorf("ollama not available at %s: %w", baseURL, err)
	}

	client, err := llm.NewOllamaClient(modelName, baseURL)
	if err != nil {
		return nil, err
	}

	return &OllamaChatModel{
		client:  client,
		options: Options{},
	}, nil
}

// Generate 实现 ChatModel 接口
func (m *OllamaChatModel) Generate(ctx context.Context, input []*schema.Message, opts ...Option) (*schema.Message, error) {
	if m.client == nil {
		return nil, fmt.Errorf("OllamaChatModel: client not configured")
	}

	// 转换消息
	messages := make([]llm.Message, len(input))
	for i, msg := range input {
		messages[i] = llm.Message{
			Role:    string(msg.Role),
			Content: msg.Content,
		}
	}

	// 合并选项
	options := m.options
	for _, opt := range opts {
		opt(&options)
	}

	// 记录 input tokens
	inputTokens := estimateTokenCount(messages)
	metrics.LLMTokensTotal.WithLabelValues("input").Add(float64(inputTokens))

	// 转换为 llm.GenerateOptions
	genOpts := llm.GenerateOptions{
		Temperature: options.Temperature,
		MaxTokens:   options.MaxTokens,
	}

	resp, err := m.client.ChatWithContext(ctx, messages, genOpts)
	if err != nil {
		return nil, err
	}

	// 记录 output tokens
	outputTokens := estimateTokenCount([]llm.Message{{Content: resp}})
	metrics.LLMTokensTotal.WithLabelValues("output").Add(float64(outputTokens))

	return &schema.Message{
		Role:    schema.Assistant,
		Content: resp,
	}, nil
}

// estimateTokenCount 估算 token 数量 (约 4 字符 = 1 token)
func estimateTokenCount(messages []llm.Message) int {
	total := 0
	for _, msg := range messages {
		total += len(msg.Content) / 4
	}
	if total == 0 {
		total = 10 // 默认估算
	}
	return total
}

// Stream 实现 ChatModel 接口：通过 Ollama /api/chat stream=true 获取 NDJSON 流式响应
func (m *OllamaChatModel) Stream(ctx context.Context, input []*schema.Message, opts ...Option) (*schema.StreamReader[*schema.Message], error) {
	if m.client == nil {
		return nil, fmt.Errorf("OllamaChatModel: client not configured")
	}

	// 合并选项
	options := m.options
	for _, opt := range opts {
		opt(&options)
	}

	// 转换消息
	chatMessages := make([]struct {
		Role    string `json:"role"`
		Content string `json:"content"`
	}, len(input))
	for i, msg := range input {
		chatMessages[i].Role = string(msg.Role)
		chatMessages[i].Content = msg.Content
	}

	reqBody := map[string]any{
		"model":    m.client.Model(),
		"messages": chatMessages,
		"stream":   true,
	}
	if options.Temperature > 0 || options.MaxTokens > 0 {
		reqBody["options"] = map[string]any{
			"temperature": options.Temperature,
			"num_predict": options.MaxTokens,
		}
	}

	return streamHTTP(ctx, m.client.BaseURL()+"/api/chat", reqBody, nil, parseOllamaStreamChunk)
}

// Ensure OllamaChatModel implements ChatModel
var _ ChatModel = (*OllamaChatModel)(nil)

// OpenAIChatModel 将 OpenAI 兼容客户端转换为 eino_examples.ChatModel 接口
type OpenAIChatModel struct {
	client  llm.Client
	options Options
}

// NewOpenAIChatModel 创建 OpenAI ChatModel
func NewOpenAIChatModel(client llm.Client, opts ...Option) *OpenAIChatModel {
	options := Options{}
	for _, opt := range opts {
		opt(&options)
	}
	return &OpenAIChatModel{
		client:  client,
		options: options,
	}
}

// NewOpenAIChatModelFromEnv 从环境变量创建 OpenAI ChatModel
func NewOpenAIChatModelFromEnv() (*OpenAIChatModel, error) {
	apiKey := os.Getenv("OPENAI_API_KEY")
	model := os.Getenv("OPENAI_MODEL")
	if model == "" {
		model = "gpt-3.5-turbo"
	}

	baseURL := os.Getenv("OPENAI_BASE_URL")
	if baseURL == "" {
		baseURL = "https://api.openai.com/v1"
	}

	client, err := llm.NewClient("openai", model, apiKey, baseURL)
	if err != nil {
		return nil, err
	}

	return &OpenAIChatModel{
		client:  client,
		options: Options{},
	}, nil
}

// Generate 实现 ChatModel 接口
func (m *OpenAIChatModel) Generate(ctx context.Context, input []*schema.Message, opts ...Option) (*schema.Message, error) {
	if m.client == nil {
		return nil, fmt.Errorf("OpenAIChatModel: client not configured")
	}

	// 转换消息
	messages := make([]llm.Message, len(input))
	for i, msg := range input {
		messages[i] = llm.Message{
			Role:    string(msg.Role),
			Content: msg.Content,
		}
	}

	// 记录 input tokens
	inputTokens := estimateTokenCount(messages)
	metrics.LLMTokensTotal.WithLabelValues("input").Add(float64(inputTokens))

	// 合并选项
	options := m.options
	for _, opt := range opts {
		opt(&options)
	}

	// 转换为 llm.GenerateOptions
	genOpts := llm.GenerateOptions{
		Temperature: options.Temperature,
		MaxTokens:   options.MaxTokens,
	}

	resp, err := m.client.ChatWithContext(ctx, messages, genOpts)
	if err != nil {
		return nil, err
	}

	// 记录 output tokens
	outputTokens := estimateTokenCount([]llm.Message{{Content: resp}})
	metrics.LLMTokensTotal.WithLabelValues("output").Add(float64(outputTokens))

	return &schema.Message{
		Role:    schema.Assistant,
		Content: resp,
	}, nil
}

// Stream 实现 ChatModel 接口：通过 OpenAI /chat/completions stream=true 获取 SSE 流式响应
func (m *OpenAIChatModel) Stream(ctx context.Context, input []*schema.Message, opts ...Option) (*schema.StreamReader[*schema.Message], error) {
	if m.client == nil {
		return nil, fmt.Errorf("OpenAIChatModel: client not configured")
	}

	options := m.options
	for _, opt := range opts {
		opt(&options)
	}

	messages := make([]map[string]string, len(input))
	for i, msg := range input {
		messages[i] = map[string]string{
			"role":    string(msg.Role),
			"content": msg.Content,
		}
	}

	oc, ok := m.client.(*llm.OpenAIClient)
	if !ok {
		return nil, fmt.Errorf("OpenAIChatModel.Stream: client is not *llm.OpenAIClient (got %T)", m.client)
	}

	reqBody := map[string]any{
		"model":       m.client.Model(),
		"messages":    messages,
		"stream":      true,
		"temperature": options.Temperature,
	}
	if options.MaxTokens > 0 {
		reqBody["max_tokens"] = options.MaxTokens
	}

	headers := map[string]string{
		"Authorization": "Bearer " + oc.APIKey(),
	}

	return streamHTTP(ctx, oc.BaseURL()+"/chat/completions", reqBody, headers, parseOpenAIStreamChunk)
}

// Ensure OpenAIChatModel implements ChatModel
var _ ChatModel = (*OpenAIChatModel)(nil)

// ClaudeChatModel 将 Claude 客户端转换为 eino_examples.ChatModel 接口
type ClaudeChatModel struct {
	client  llm.Client
	options Options
}

// NewClaudeChatModel 创建 Claude ChatModel
func NewClaudeChatModel(client llm.Client, opts ...Option) *ClaudeChatModel {
	options := Options{}
	for _, opt := range opts {
		opt(&options)
	}
	return &ClaudeChatModel{
		client:  client,
		options: options,
	}
}

// NewClaudeChatModelFromEnv 从环境变量创建 Claude ChatModel
func NewClaudeChatModelFromEnv() (*ClaudeChatModel, error) {
	apiKey := os.Getenv("ANTHROPIC_API_KEY")
	model := os.Getenv("CLAUDE_MODEL")
	if model == "" {
		model = "claude-3-haiku-20240307"
	}

	client, err := llm.NewClient("claude", model, apiKey, "")
	if err != nil {
		return nil, err
	}

	return &ClaudeChatModel{
		client:  client,
		options: Options{},
	}, nil
}

// Generate 实现 ChatModel 接口
func (m *ClaudeChatModel) Generate(ctx context.Context, input []*schema.Message, opts ...Option) (*schema.Message, error) {
	if m.client == nil {
		return nil, fmt.Errorf("ClaudeChatModel: client not configured")
	}

	// 转换消息
	messages := make([]llm.Message, len(input))
	for i, msg := range input {
		messages[i] = llm.Message{
			Role:    string(msg.Role),
			Content: msg.Content,
		}
	}

	// 记录 input tokens
	inputTokens := estimateTokenCount(messages)
	metrics.LLMTokensTotal.WithLabelValues("input").Add(float64(inputTokens))

	// 合并选项
	options := m.options
	for _, opt := range opts {
		opt(&options)
	}

	genOpts := llm.GenerateOptions{
		Temperature: options.Temperature,
		MaxTokens:   options.MaxTokens,
	}

	resp, err := m.client.ChatWithContext(ctx, messages, genOpts)
	if err != nil {
		return nil, err
	}

	// 记录 output tokens
	outputTokens := estimateTokenCount([]llm.Message{{Content: resp}})
	metrics.LLMTokensTotal.WithLabelValues("output").Add(float64(outputTokens))

	return &schema.Message{
		Role:    schema.Assistant,
		Content: resp,
	}, nil
}

// Stream 实现 ChatModel 接口：Claude 流式通过 OpenAI 兼容层（如 OpenRouter），否则返回错误
func (m *ClaudeChatModel) Stream(ctx context.Context, input []*schema.Message, opts ...Option) (*schema.StreamReader[*schema.Message], error) {
	if m.client == nil {
		return nil, fmt.Errorf("ClaudeChatModel: client not configured")
	}

	options := m.options
	for _, opt := range opts {
		opt(&options)
	}

	messages := make([]map[string]string, len(input))
	for i, msg := range input {
		messages[i] = map[string]string{
			"role":    string(msg.Role),
			"content": msg.Content,
		}
	}

	// 尝试 OpenAI 兼容层（Claude via OpenRouter/proxy）
	oc, ok := m.client.(*llm.OpenAIClient)
	if !ok {
		return nil, fmt.Errorf("ClaudeChatModel.Stream: streaming requires OpenAI-compatible client (got %T)", m.client)
	}

	reqBody := map[string]any{
		"model":       m.client.Model(),
		"messages":    messages,
		"stream":      true,
		"temperature": options.Temperature,
	}
	if options.MaxTokens > 0 {
		reqBody["max_tokens"] = options.MaxTokens
	}

	headers := map[string]string{
		"Authorization": "Bearer " + oc.APIKey(),
	}

	return streamHTTP(ctx, oc.BaseURL()+"/chat/completions", reqBody, headers, parseOpenAIStreamChunk)
}

// Ensure ClaudeChatModel implements ChatModel
var _ ChatModel = (*ClaudeChatModel)(nil)

// NewChatModelFromEnv 根据环境变量自动选择合适的 ChatModel
// 支持: OLLAMA_, OPENAI_, ANTHROPIC_ 前缀的环境变量
func NewChatModelFromEnv() (ChatModel, error) {
	// 优先检查 Ollama
	if os.Getenv("OLLAMA_MODEL") != "" || os.Getenv("OLLAMA_BASE_URL") != "" {
		return NewOllamaChatModelFromEnv()
	}

	// 检查 OpenAI
	if os.Getenv("OPENAI_API_KEY") != "" {
		return NewOpenAIChatModelFromEnv()
	}

	// 检查 Claude
	if os.Getenv("ANTHROPIC_API_KEY") != "" {
		return NewClaudeChatModelFromEnv()
	}

	// 默认尝试 Ollama
	if isOllamaAvailable() {
		return NewOllamaChatModelFromEnv()
	}

	return nil, fmt.Errorf("no LLM provider configured. Set OLLAMA_MODEL, OPENAI_API_KEY, or ANTHROPIC_API_KEY")
}

// isOllamaAvailable 检查 Ollama 是否可用
func isOllamaAvailable() bool {
	baseURL := os.Getenv("OLLAMA_BASE_URL")
	if baseURL == "" {
		baseURL = "http://localhost:11434"
	}
	probe := &http.Client{Timeout: 2 * time.Second}
	resp, err := probe.Get(baseURL + "/api/tags")
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}

// CreateToolsFromFuncs 将函数映射转换为 ToolExecutor 列表，可直接传入 adapter 的 Tools 字段。
func CreateToolsFromFuncs(funcs map[string]func(ctx context.Context, args map[string]any) (string, error)) []interface{} {
	tools := make([]interface{}, 0, len(funcs))
	for name, fn := range funcs {
		tools = append(tools, NewSimpleTool(name, "Tool: "+name, fn))
	}
	return tools
}

// SimpleTool 简单工具实现（占位符，完整实现需要 eino tool 接口）
type SimpleTool struct {
	name        string
	description string
	fn          func(ctx context.Context, args map[string]any) (string, error)
}

// NewSimpleTool 创建简单工具
func NewSimpleTool(name, description string, fn func(ctx context.Context, args map[string]any) (string, error)) *SimpleTool {
	return &SimpleTool{
		name:        name,
		description: description,
		fn:          fn,
	}
}

// Name returns the tool's unique name.
func (t *SimpleTool) Name() string {
	return t.name
}

// Description returns the tool's description.
func (t *SimpleTool) Description() string {
	return t.description
}

// Execute runs the tool with the parsed JSON arguments.
func (t *SimpleTool) Execute(ctx context.Context, args map[string]any) (string, error) {
	if t.fn == nil {
		return "", fmt.Errorf("tool %q has no handler", t.name)
	}
	return t.fn(ctx, args)
}

// MockChatModelWithResponse 创建带有预设响应的 Mock ChatModel
func MockChatModelWithResponse(response string) *MockChatModel {
	return &MockChatModel{
		response: &schema.Message{Content: response},
	}
}

// MockChatModel  Mock ChatModel for testing
type MockChatModel struct {
	response *schema.Message
	err      error
	Calls    int
}

// Generate 实现 ChatModel 接口
func (m *MockChatModel) Generate(ctx context.Context, input []*schema.Message, opts ...Option) (*schema.Message, error) {
	m.Calls++
	if m.err != nil {
		return nil, m.err
	}
	if m.response != nil {
		return m.response, nil
	}
	return &schema.Message{Content: "mock response"}, nil
}

// Stream 实现 ChatModel 接口：将预设响应分片发送
func (m *MockChatModel) Stream(ctx context.Context, input []*schema.Message, opts ...Option) (*schema.StreamReader[*schema.Message], error) {
	sr, sw := schema.Pipe[*schema.Message](10)
	go func() {
		defer sw.Close()
		if m.err != nil {
			sw.Send(nil, m.err)
			return
		}
		content := "mock response"
		if m.response != nil && m.response.Content != "" {
			content = m.response.Content
		}
		// Split into word-sized chunks
		words := strings.Fields(content)
		if len(words) == 0 {
			words = []string{content}
		}
		for _, word := range words {
			select {
			case <-ctx.Done():
				sw.Send(nil, ctx.Err())
				return
			default:
			}
			sw.Send(&schema.Message{
				Role:    schema.Assistant,
				Content: word + " ",
			}, nil)
		}
	}()
	return sr, nil
}

// Ensure MockChatModel implements ChatModel
var _ ChatModel = (*MockChatModel)(nil)

// ParseToolArguments 解析工具参数 JSON 字符串为 map
func ParseToolArguments(jsonStr string) (map[string]any, error) {
	if jsonStr == "" {
		return make(map[string]any), nil
	}
	var args map[string]any
	if err := json.Unmarshal([]byte(jsonStr), &args); err != nil {
		return nil, fmt.Errorf("parse tool arguments: %w", err)
	}
	if args == nil {
		args = make(map[string]any)
	}
	return args, nil
}

// streamChunkParser 解析单条流式响应数据，返回消息内容和是否结束
type streamChunkParser func(line string) (content string, done bool, err error)

// streamHTTP 发起 HTTP 请求并将流式响应通过 schema.Pipe 转换为 StreamReader
func streamHTTP(ctx context.Context, url string, reqBody map[string]any, headers map[string]string, parser streamChunkParser) (*schema.StreamReader[*schema.Message], error) {
	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	for k, v := range headers {
		httpReq.Header.Set(k, v)
	}

	resp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("stream request: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 64*1024))
		resp.Body.Close()
		return nil, fmt.Errorf("stream request failed (status %d): %s", resp.StatusCode, string(body))
	}

	sr, sw := schema.Pipe[*schema.Message](50)
	go func() {
		defer sw.Close()
		defer resp.Body.Close()

		scanner := bufio.NewScanner(resp.Body)
		scanner.Buffer(make([]byte, 0, 64*1024), 10*1024*1024)
		for scanner.Scan() {
			select {
			case <-ctx.Done():
				sw.Send(nil, ctx.Err())
				return
			default:
			}
			line := scanner.Text()
			if line == "" {
				continue
			}
			content, done, perr := parser(line)
			if perr != nil {
				sw.Send(nil, perr)
				return
			}
			if done {
				return
			}
			if content != "" {
				sw.Send(&schema.Message{
					Role:    schema.Assistant,
					Content: content,
				}, nil)
			}
		}
		if err := scanner.Err(); err != nil {
			sw.Send(nil, err)
		}
	}()

	return sr, nil
}

// parseOllamaStreamChunk 解析 Ollama NDJSON 流式响应行
func parseOllamaStreamChunk(line string) (content string, done bool, err error) {
	var chunk struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
		Done bool `json:"done"`
	}
	if err := json.Unmarshal([]byte(line), &chunk); err != nil {
		return "", false, nil // skip unparseable lines
	}
	if chunk.Done {
		return "", true, nil
	}
	return chunk.Message.Content, false, nil
}

// parseOpenAIStreamChunk 解析 OpenAI SSE 流式响应行
func parseOpenAIStreamChunk(line string) (content string, done bool, err error) {
	if !strings.HasPrefix(line, "data: ") {
		return "", false, nil // skip non-data lines
	}
	data := strings.TrimPrefix(line, "data: ")
	if data == "[DONE]" {
		return "", true, nil
	}
	var chunk struct {
		Choices []struct {
			Delta struct {
				Content string `json:"content"`
			} `json:"delta"`
			FinishReason string `json:"finish_reason"`
		} `json:"choices"`
	}
	if err := json.Unmarshal([]byte(data), &chunk); err != nil {
		return "", false, nil // skip unparseable
	}
	if len(chunk.Choices) == 0 {
		return "", false, nil
	}
	if chunk.Choices[0].FinishReason != "" {
		return "", true, nil
	}
	return chunk.Choices[0].Delta.Content, false, nil
}
