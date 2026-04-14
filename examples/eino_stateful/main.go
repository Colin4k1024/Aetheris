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

// Package main 展示带状态的 eino Agent，证明可以持久化运行
//
// 这个示例展示了：
// 1. 模拟会话状态的保存和恢复
// 2. 多轮对话中的上下文保持
// 3. Checkpoint 机制的实现
// 4. 与 CoRag 框架的持久化集成
//
// 运行方式：
//
//	go run ./examples/eino_stateful/main.go
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"sync"

	"github.com/cloudwego/eino/schema"

	eino_examples "github.com/Colin4k1024/Aetheris/v2/internal/agent/runtime/executor/eino_examples"
	"github.com/Colin4k1024/Aetheris/v2/internal/model/llm"
)

// SessionState 会话状态
type SessionState struct {
	mu           sync.RWMutex
	Messages     []Message      `json:"messages"`      // 对话历史
	Variables    map[string]any `json:"variables"`     // 变量
	CheckpointID string         `json:"checkpoint_id"` // 检查点ID
}

// Message 消息
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// SessionStore 会话存储（模拟持久化）
type SessionStore struct {
	mu       sync.RWMutex
	sessions map[string]*SessionState
}

// NewSessionStore 创建会话存储
func NewSessionStore() *SessionStore {
	return &SessionStore{
		sessions: make(map[string]*SessionState),
	}
}

// Get 获取会话
func (s *SessionStore) Get(sessionID string) *SessionState {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.sessions[sessionID]
}

// Save 保存会话
func (s *SessionStore) Save(sessionID string, state *SessionState) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sessions[sessionID] = state
}

// Delete 删除会话
func (s *SessionStore) Delete(sessionID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.sessions, sessionID)
}

// CheckpointStore 检查点存储（模拟持久化）
type CheckpointStore struct {
	mu          sync.RWMutex
	checkpoints map[string][]byte
}

// NewCheckpointStore 创建检查点存储
func NewCheckpointStore() *CheckpointStore {
	return &CheckpointStore{
		checkpoints: make(map[string][]byte),
	}
}

// Get 获取检查点
func (c *CheckpointStore) Get(checkpointID string) ([]byte, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	data, ok := c.checkpoints[checkpointID]
	return data, ok
}

// Save 保存检查点
func (c *CheckpointStore) Save(checkpointID string, data []byte) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.checkpoints[checkpointID] = data
	return nil
}

// StatefulAgent 状态管理 Agent
type StatefulAgent struct {
	model      eino_examples.ChatModel
	store      *SessionStore
	checkpoint *CheckpointStore
	sessionID  string
}

// NewStatefulAgent 创建状态管理 Agent
func NewStatefulAgent(model eino_examples.ChatModel, store *SessionStore, checkpoint *CheckpointStore, sessionID string) *StatefulAgent {
	return &StatefulAgent{
		model:      model,
		store:      store,
		checkpoint: checkpoint,
		sessionID:  sessionID,
	}
}

// Query 执行查询并保存状态
func (a *StatefulAgent) Query(ctx context.Context, prompt string) (string, error) {
	// 获取当前会话状态
	state := a.store.Get(a.sessionID)
	if state == nil {
		state = &SessionState{
			Variables: make(map[string]any),
		}
	}

	// 构建消息历史
	messages := make([]*schema.Message, 0, len(state.Messages)+1)
	for _, m := range state.Messages {
		role := schema.User
		if m.Role == "assistant" {
			role = schema.Assistant
		}
		messages = append(messages, &schema.Message{Role: role, Content: m.Content})
	}
	messages = append(messages, &schema.Message{Role: schema.User, Content: prompt})

	// 调用模型
	result, err := a.model.Generate(ctx, messages)
	if err != nil {
		return "", err
	}

	// 保存用户消息
	state.Messages = append(state.Messages, Message{
		Role:    "user",
		Content: prompt,
	})

	// 保存助手回复
	state.Messages = append(state.Messages, Message{
		Role:    "assistant",
		Content: result.Content,
	})

	// 持久化会话状态
	a.store.Save(a.sessionID, state)

	return result.Content, nil
}

// SaveCheckpoint 保存检查点
func (a *StatefulAgent) SaveCheckpoint(checkpointID string) error {
	state := a.store.Get(a.sessionID)
	if state == nil {
		return fmt.Errorf("no state to checkpoint")
	}

	data, err := json.Marshal(state)
	if err != nil {
		return err
	}

	state.CheckpointID = checkpointID
	return a.checkpoint.Save(checkpointID, data)
}

// RestoreCheckpoint 恢复检查点
func (a *StatefulAgent) RestoreCheckpoint(checkpointID string) error {
	data, ok := a.checkpoint.Get(checkpointID)
	if !ok {
		return fmt.Errorf("checkpoint not found: %s", checkpointID)
	}

	var state SessionState
	if err := json.Unmarshal(data, &state); err != nil {
		return err
	}

	a.store.Save(a.sessionID, &state)
	return nil
}

// GetHistory 获取对话历史
func (a *StatefulAgent) GetHistory() []Message {
	state := a.store.Get(a.sessionID)
	if state == nil {
		return nil
	}
	return state.Messages
}

func main() {
	ctx := context.Background()

	// ============ 1. 创建 Ollama LLM 客户端 ============
	modelName := os.Getenv("OLLAMA_MODEL")
	if modelName == "" {
		modelName = "llama3"
	}

	baseURL := os.Getenv("OLLAMA_BASE_URL")
	if baseURL == "" {
		baseURL = "http://localhost:11434"
	}

	ollamaClient, err := llm.NewOllamaClient(modelName, baseURL)
	if err != nil {
		log.Fatalf("创建 Ollama 客户端失败: %v", err)
	}

	chatModel := eino_examples.NewOllamaChatModel(ollamaClient)

	// ============ 2. 创建存储 ============
	sessionStore := NewSessionStore()
	checkpointStore := NewCheckpointStore()

	// ============ 3. 创建状态 Agent ============
	sessionID := "user-123"
	agent := NewStatefulAgent(chatModel, sessionStore, checkpointStore, sessionID)

	fmt.Println("=== 测试 1: 多轮对话 ===")

	// 第一轮
	resp1, err := agent.Query(ctx, "我的名字是张三")
	if err != nil {
		log.Printf("查询失败: %v", err)
	} else {
		fmt.Printf("助手: %s\n", resp1)
	}

	// 第二轮 - 应该记住名字
	resp2, err := agent.Query(ctx, "我叫什么名字?")
	if err != nil {
		log.Printf("查询失败: %v", err)
	} else {
		fmt.Printf("助手: %s\n", resp2)
	}

	// 第三轮
	resp3, err := agent.Query(ctx, "今天天气怎么样?")
	if err != nil {
		log.Printf("查询失败: %v", err)
	} else {
		fmt.Printf("助手: %s\n", resp3)
	}

	// 第四轮 - 再次询问名字
	resp4, err := agent.Query(ctx, "还记得我叫什么吗?")
	if err != nil {
		log.Printf("查询失败: %v", err)
	} else {
		fmt.Printf("助手: %s\n", resp4)
	}

	fmt.Println("\n=== 测试 2: 检查点保存 ===")

	// 保存检查点
	checkpointID := "checkpoint-001"
	err = agent.SaveCheckpoint(checkpointID)
	if err != nil {
		log.Printf("保存检查点失败: %v", err)
	} else {
		fmt.Printf("检查点已保存: %s\n", checkpointID)
	}

	// 删除会话模拟重启
	sessionStore.Delete(sessionID)
	fmt.Println("会话已清除")

	// 恢复检查点
	err = agent.RestoreCheckpoint(checkpointID)
	if err != nil {
		log.Printf("恢复检查点失败: %v", err)
	} else {
		fmt.Printf("检查点已恢复: %s\n", checkpointID)
	}

	// 继续对话 - 应该恢复上下文
	resp5, err := agent.Query(ctx, "我们之前聊了什么?")
	if err != nil {
		log.Printf("查询失败: %v", err)
	} else {
		fmt.Printf("助手: %s\n", resp5)
	}

	fmt.Println("\n=== 测试 3: 对话历史 ===")
	history := agent.GetHistory()
	fmt.Printf("对话历史 (%d 条):\n", len(history))
	for i, m := range history {
		role := "用户"
		if m.Role == "assistant" {
			role = "助手"
		}
		content := m.Content
		if len(content) > 50 {
			content = content[:50] + "..."
		}
		fmt.Printf("  %d. [%s]: %s\n", i+1, role, content)
	}

	fmt.Println("\n=== 测试完成 ===")
}
