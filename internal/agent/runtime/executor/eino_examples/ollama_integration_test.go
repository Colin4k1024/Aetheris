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
	"os"
	"testing"

	"github.com/cloudwego/eino/schema"

	"github.com/Colin4k1024/Aetheris/v2/internal/agent/runtime/executor"
)

// TestOllamaReactAgent_Invoke 使用真实 Ollama LLM 测试 ReAct Agent
// 运行: go test -v -run TestOllamaReactAgent_Invoke -timeout 5m
func TestOllamaReactAgent_Invoke(t *testing.T) {
	// 检查是否配置了 Ollama
	modelName := os.Getenv("OLLAMA_MODEL")
	if modelName == "" {
		modelName = "llama3"
	}

	// 创建 Ollama 客户端
	client, err := NewOllamaChatModelFromEnv()
	if err != nil {
		t.Skipf("Skipping Ollama test: %v", err)
	}

	// 创建 ReAct Agent
	adapter := NewReactAgentAdapter(client, nil)

	// 测试简单问答
	result, err := adapter.Invoke(context.Background(), map[string]any{
		"prompt": "What is 2+2? Answer in one sentence.",
	})
	if err != nil {
		t.Fatalf("Invoke error = %v", err)
	}

	response, ok := result["response"].(string)
	if !ok {
		t.Fatal("response should be a string")
	}
	if response == "" {
		t.Fatal("response should not be empty")
	}

	t.Logf("Ollama Response: %s", response)
}

// TestOllamaDEERAgent_Invoke 使用真实 Ollama LLM 测试 DEER Agent
// 运行: go test -v -run TestOllamaDEERAgent_Invoke -timeout 5m
func TestOllamaDEERAgent_Invoke(t *testing.T) {
	client, err := NewOllamaChatModelFromEnv()
	if err != nil {
		t.Skipf("Skipping Ollama test: %v", err)
	}

	adapter := NewDEERAgentAdapter(client, nil)

	result, err := adapter.Invoke(context.Background(), map[string]any{
		"prompt": "What is the capital of France?",
	})
	if err != nil {
		t.Fatalf("Invoke error = %v", err)
	}

	response, ok := result["response"].(string)
	if !ok {
		t.Fatal("response should be a string")
	}
	if response == "" {
		t.Fatal("response should not be empty")
	}

	t.Logf("DEER Agent Response: %s", response)
}

// TestOllamaManusAgent_Invoke 使用真实 Ollama LLM 测试 Manus Agent
// 运行: go test -v -run TestOllamaManusAgent_Invoke -timeout 5m
func TestOllamaManusAgent_Invoke(t *testing.T) {
	client, err := NewOllamaChatModelFromEnv()
	if err != nil {
		t.Skipf("Skipping Ollama test: %v", err)
	}

	adapter := NewManusAgentAdapter(client, nil)

	result, err := adapter.Invoke(context.Background(), map[string]any{
		"prompt": "Tell me a short joke.",
	})
	if err != nil {
		t.Fatalf("Invoke error = %v", err)
	}

	response, ok := result["response"].(string)
	if !ok {
		t.Fatal("response should be a string")
	}

	t.Logf("Manus Agent Response: %s", response)
}

// TestOllamaReactAgent_WithOptions 使用选项测试
// 运行: go test -v -run TestOllamaReactAgent_WithOptions -timeout 5m
func TestOllamaReactAgent_WithOptions(t *testing.T) {
	client, err := NewOllamaChatModelFromEnv()
	if err != nil {
		t.Skipf("Skipping Ollama test: %v", err)
	}

	// 创建带温度选项的 ReAct Agent
	adapter := NewReactAgentAdapter(client, nil, WithTemperature(0.9))

	result, err := adapter.Invoke(context.Background(), map[string]any{
		"prompt": "Complete this sentence: The quick brown fox",
	})
	if err != nil {
		t.Fatalf("Invoke error = %v", err)
	}

	response := result["response"].(string)
	t.Logf("Response with temperature 0.9: %s", response)
}

// TestOllamaChatModel_Generate 直接测试 ChatModel 接口
// 运行: go test -v -run TestOllamaChatModel_Generate -timeout 5m
func TestOllamaChatModel_Generate(t *testing.T) {
	client, err := NewOllamaChatModelFromEnv()
	if err != nil {
		t.Skipf("Skipping Ollama test: %v", err)
	}

	// 使用有效的消息输入
	input := []*schema.Message{
		{Role: schema.User, Content: "Hello"},
	}
	msg, err := client.Generate(context.Background(), input)
	if err != nil {
		t.Fatalf("Generate error = %v", err)
	}

	if msg.Content == "" {
		t.Fatal("content should not be empty")
	}

	t.Logf("Generated: %s", msg.Content)
}

// TestOllamaListModels 测试列出模型
// 运行: go test -v -run TestOllamaListModels -timeout 30s
func TestOllamaListModels(t *testing.T) {
	modelName := os.Getenv("OLLAMA_MODEL")
	if modelName == "" {
		modelName = "llama3"
	}

	baseURL := os.Getenv("OLLAMA_BASE_URL")
	if baseURL == "" {
		baseURL = "http://localhost:11434"
	}

	ollamaClient, err := NewOllamaChatModelFromEnv()
	if err != nil {
		t.Skipf("Skipping Ollama test: %v", err)
	}

	models, err := ollamaClient.client.ListModels(context.Background())
	if err != nil {
		t.Fatalf("ListModels error = %v", err)
	}

	if len(models) == 0 {
		t.Fatal("should have at least one model")
	}

	t.Logf("Available models:")
	for _, m := range models {
		t.Logf("  - %s", m.Name)
	}
}

// TestOllamaToNodeRunner 测试转换为 NodeRunner
// 运行: go test -v -run TestOllamaToNodeRunner -timeout 5m
func TestOllamaToNodeRunner(t *testing.T) {
	client, err := NewOllamaChatModelFromEnv()
	if err != nil {
		t.Skipf("Skipping Ollama test: %v", err)
	}

	adapter := NewReactAgentAdapter(client, nil)
	runner := ToNodeRunner(adapter)

	payload := &executor.AgentDAGPayload{
		Goal:    "What is 1+1?",
		Results: make(map[string]any),
	}

	result, err := runner(context.Background(), payload)
	if err != nil {
		t.Fatalf("runner error = %v", err)
	}

	if result.Results["eino"] == nil {
		t.Fatal("result should have eino key")
	}

	t.Logf("NodeRunner Result: %v", result.Results["eino"])
}

// TestOllamaMultiTurnConversation 测试多轮对话
// 运行: go test -v -run TestOllamaMultiTurnConversation -timeout 5m
func TestOllamaMultiTurnConversation(t *testing.T) {
	client, err := NewOllamaChatModelFromEnv()
	if err != nil {
		t.Skipf("Skipping Ollama test: %v", err)
	}

	adapter := NewReactAgentAdapter(client, nil)

	// 第一轮
	result1, err := adapter.Invoke(context.Background(), map[string]any{
		"prompt": "My name is Alice. What is my name?",
	})
	if err != nil {
		t.Fatalf("First invoke error = %v", err)
	}
	t.Logf("First response: %v", result1["response"])

	// 第二轮 (在真实场景中需要维护对话历史)
	result2, err := adapter.Invoke(context.Background(), map[string]any{
		"prompt": "What is 2+2?",
	})
	if err != nil {
		t.Fatalf("Second invoke error = %v", err)
	}
	t.Logf("Second response: %v", result2["response"])
}

// TestOllamaDifferentModels 测试不同模型
// 运行: go test -v -run TestOllamaDifferentModels -timeout 10m
func TestOllamaDifferentModels(t *testing.T) {
	models := []string{"llama3"}

	for _, model := range models {
		t.Run(model, func(t *testing.T) {
			// 设置特定模型
			os.Setenv("OLLAMA_MODEL", model)
			defer os.Unsetenv("OLLAMA_MODEL")

			client, err := NewOllamaChatModelFromEnv()
			if err != nil {
				t.Skipf("Skipping model %s: %v", model, err)
			}

			adapter := NewReactAgentAdapter(client, nil)
			result, err := adapter.Invoke(context.Background(), map[string]any{
				"prompt": "Say 'hello' and nothing else.",
			})
			if err != nil {
				t.Fatalf("Invoke error = %v", err)
			}

			t.Logf("Model %s response: %v", model, result["response"])
		})
	}
}
