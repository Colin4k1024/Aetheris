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
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/cloudwego/eino/schema"

	"rag-platform/internal/model/llm"
	"rag-platform/pkg/metrics"
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

// Stream 实现 ChatModel 接口
func (m *OllamaChatModel) Stream(ctx context.Context, input []*schema.Message, opts ...Option) (*schema.StreamReader[*schema.Message], error) {
	// TODO: 实现流式支持
	return nil, fmt.Errorf("streaming not implemented yet")
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

// Stream 实现 ChatModel 接口
func (m *OpenAIChatModel) Stream(ctx context.Context, input []*schema.Message, opts ...Option) (*schema.StreamReader[*schema.Message], error) {
	// TODO: 实现流式支持
	return nil, fmt.Errorf("streaming not implemented yet")
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

// Stream 实现 ChatModel 接口
func (m *ClaudeChatModel) Stream(ctx context.Context, input []*schema.Message, opts ...Option) (*schema.StreamReader[*schema.Message], error) {
	return nil, fmt.Errorf("streaming not implemented yet")
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
	// 简单检查，不实际请求
	return true
}

// CreateToolsFromFuncs 将函数转换为 eino 工具
// CreateToolsFromFuncs 占位符
func CreateToolsFromFuncs(funcs map[string]func(ctx context.Context, args map[string]any) (string, error)) {
	// 简化实现，实际项目中需要完整的 tool 定义
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

// Stream 实现 ChatModel 接口
func (m *MockChatModel) Stream(ctx context.Context, input []*schema.Message, opts ...Option) (*schema.StreamReader[*schema.Message], error) {
	return nil, fmt.Errorf("streaming not implemented")
}

// Ensure MockChatModel implements ChatModel
var _ ChatModel = (*MockChatModel)(nil)

// ParseToolArguments 解析工具参数 JSON 字符串
func ParseToolArguments(jsonStr string) (map[string]any, error) {
	// 这是一个占位实现
	// 实际项目中需要完整的 JSON 解析
	if jsonStr == "" {
		return make(map[string]any), nil
	}

	// 简单的 key=value 解析
	args := make(map[string]any)
	pairs := strings.Split(jsonStr, ",")
	for _, pair := range pairs {
		kv := strings.SplitN(pair, "=", 2)
		if len(kv) == 2 {
			args[kv[0]] = kv[1]
		}
	}
	return args, nil
}
