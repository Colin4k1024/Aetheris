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

package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/cloudwego/eino/schema"

	eino_examples "github.com/Colin4k1024/Aetheris/v2/internal/agent/runtime/executor/eino_examples"
	"github.com/Colin4k1024/Aetheris/v2/internal/model/llm"
)

// OllamaChatModel 将 ll eino_examples.Chatm.Client 转换为Model
type OllamaChatModel struct {
	client *llm.OllamaClient
}

func NewOllamaChatModel(client *llm.OllamaClient) *OllamaChatModel {
	return &OllamaChatModel{client: client}
}

func (m *OllamaChatModel) Generate(ctx context.Context, input []*schema.Message, opts ...eino_examples.Option) (*schema.Message, error) {
	// 转换消息
	messages := make([]llm.Message, len(input))
	for i, msg := range input {
		messages[i] = llm.Message{
			Role:    string(msg.Role),
			Content: msg.Content,
		}
	}

	// 转换选项 (简化版)
	options := llm.GenerateOptions{}

	resp, err := m.client.ChatWithContext(ctx, messages, options)
	if err != nil {
		return nil, err
	}

	return &schema.Message{
		Role:    schema.Assistant,
		Content: resp,
	}, nil
}

func (m *OllamaChatModel) Stream(ctx context.Context, input []*schema.Message, opts ...eino_examples.Option) (*schema.StreamReader[*schema.Message], error) {
	// TODO: 实现流式
	return nil, fmt.Errorf("streaming not implemented")
}

func main() {
	ctx := context.Background()

	// 从环境变量或默认值获取配置
	modelName := os.Getenv("OLLAMA_MODEL")
	if modelName == "" {
		modelName = "llama3"
	}

	baseURL := os.Getenv("OLLAMA_BASE_URL")
	if baseURL == "" {
		baseURL = "http://localhost:11434"
	}

	// 创建 Ollama 客户端
	client, err := llm.NewOllamaClient(modelName, baseURL)
	if err != nil {
		log.Fatalf("Failed to create Ollama client: %v", err)
	}

	// 列出可用模型
	models, err := client.ListModels(ctx)
	if err != nil {
		log.Printf("Warning: Failed to list models: %v", err)
	} else {
		fmt.Println("Available models:")
		for _, m := range models {
			fmt.Printf("  - %s\n", m.Name)
		}
	}

	// 测试聊天
	fmt.Println("\nTesting chat...")
	resp, err := client.Chat(
		[]llm.Message{
			{Role: "user", Content: "Hello, how are you?"},
		},
		llm.GenerateOptions{
			Temperature: 0.7,
		},
	)
	if err != nil {
		log.Fatalf("Chat failed: %v", err)
	}
	fmt.Printf("Response: %s\n", resp)

	// 测试生成
	fmt.Println("\nTesting generate...")
	genResp, err := client.Generate(
		"Write a short poem about AI:",
		llm.GenerateOptions{
			Temperature: 0.8,
			MaxTokens:   100,
		},
	)
	if err != nil {
		log.Fatalf("Generate failed: %v", err)
	}
	fmt.Printf("Generated: %s\n", genResp)

	// 测试 ReAct Agent
	fmt.Println("\nTesting ReAct Agent...")
	reactAdapter := eino_examples.NewReactAgentAdapter(
		NewOllamaChatModel(client),
		nil, // 无工具
	)

	result, err := reactAdapter.Invoke(ctx, map[string]any{
		"prompt": "What is 2+2?",
	})
	if err != nil {
		log.Fatalf("ReAct invoke failed: %v", err)
	}
	fmt.Printf("ReAct Result: %v\n", result)
}
