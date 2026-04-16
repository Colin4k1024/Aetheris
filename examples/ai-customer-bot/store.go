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
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/google/uuid"
)

// Message represents a single message in the conversation
type Message struct {
	ID        string    `json:"id"`
	Role      string    `json:"role"` // user, assistant, system
	Content   string    `json:"content"`
	Timestamp time.Time `json:"timestamp"`
}

// Conversation represents a complete conversation session
type Conversation struct {
	ID        string         `json:"id"`
	UserID    string         `json:"user_id"`
	Messages  []Message      `json:"messages"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	Metadata  map[string]any `json:"metadata,omitempty"`
}

// ConversationStore handles persistence of conversation history
type ConversationStore struct {
	mu            sync.RWMutex
	filePath      string
	conversations map[string]*Conversation
}

// NewConversationStore creates a new conversation store with JSON file persistence
func NewConversationStore(filePath string) (*ConversationStore, error) {
	store := &ConversationStore{
		filePath:      filePath,
		conversations: make(map[string]*Conversation),
	}

	// Load existing conversations from file
	if err := store.loadFromFile(); err != nil {
		// If file doesn't exist, that's okay - we'll create it
		if !os.IsNotExist(err) {
			return nil, fmt.Errorf("failed to load conversation store: %w", err)
		}
	}

	return store, nil
}

// loadFromFile loads conversations from the JSON file
func (s *ConversationStore) loadFromFile() error {
	data, err := os.ReadFile(s.filePath)
	if err != nil {
		return err
	}

	if len(data) == 0 {
		return nil
	}

	var conversations map[string]*Conversation
	if err := json.Unmarshal(data, &conversations); err != nil {
		return fmt.Errorf("failed to unmarshal conversations: %w", err)
	}

	s.conversations = conversations
	return nil
}

// saveToFile saves conversations to the JSON file
func (s *ConversationStore) saveToFile() error {
	data, err := json.MarshalIndent(s.conversations, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal conversations: %w", err)
	}

	if err := os.WriteFile(s.filePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write conversations file: %w", err)
	}

	return nil
}

// CreateConversation creates a new conversation
func (s *ConversationStore) CreateConversation(userID string) (*Conversation, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	conv := &Conversation{
		ID:        "conv-" + uuid.New().String()[:8],
		UserID:    userID,
		Messages:  make([]Message, 0),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Metadata:  make(map[string]any),
	}

	s.conversations[conv.ID] = conv
	if err := s.saveToFile(); err != nil {
		delete(s.conversations, conv.ID)
		return nil, fmt.Errorf("failed to persist new conversation: %w", err)
	}

	return conv, nil
}

// GetConversation retrieves a conversation by ID
func (s *ConversationStore) GetConversation(id string) (*Conversation, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	conv, ok := s.conversations[id]
	if !ok {
		return nil, fmt.Errorf("conversation not found: %s", id)
	}

	// Return a copy to avoid race conditions.
	// Metadata values are copied shallowly; callers must treat reference-type
	// values (slices, maps, pointers) as read-only to avoid unintended mutation.
	metaCopy := make(map[string]any, len(conv.Metadata))
	for k, v := range conv.Metadata {
		metaCopy[k] = v
	}
	convCopy := &Conversation{
		ID:        conv.ID,
		UserID:    conv.UserID,
		Messages:  make([]Message, len(conv.Messages)),
		CreatedAt: conv.CreatedAt,
		UpdatedAt: conv.UpdatedAt,
		Metadata:  metaCopy,
	}
	copy(convCopy.Messages, conv.Messages)

	return convCopy, nil
}

// AddMessage adds a message to a conversation
func (s *ConversationStore) AddMessage(convID, role, content string) (*Message, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	conv, ok := s.conversations[convID]
	if !ok {
		return nil, fmt.Errorf("conversation not found: %s", convID)
	}

	msg := Message{
		ID:        "msg-" + uuid.New().String()[:8],
		Role:      role,
		Content:   content,
		Timestamp: time.Now(),
	}

	origLen := len(conv.Messages)
	origUpdatedAt := conv.UpdatedAt

	conv.Messages = append(conv.Messages, msg)
	conv.UpdatedAt = time.Now()

	if err := s.saveToFile(); err != nil {
		conv.Messages = conv.Messages[:origLen]
		conv.UpdatedAt = origUpdatedAt
		return nil, fmt.Errorf("failed to persist message: %w", err)
	}

	return &msg, nil
}

// GetMessages returns all messages for a conversation
func (s *ConversationStore) GetMessages(convID string) ([]Message, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	conv, ok := s.conversations[convID]
	if !ok {
		return nil, fmt.Errorf("conversation not found: %s", convID)
	}

	// Return a copy
	messages := make([]Message, len(conv.Messages))
	copy(messages, conv.Messages)

	return messages, nil
}

// GetConversationHistory returns formatted history for LLM context
func (s *ConversationStore) GetConversationHistory(convID string) ([]map[string]string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	conv, ok := s.conversations[convID]
	if !ok {
		return nil, fmt.Errorf("conversation not found: %s", convID)
	}

	history := make([]map[string]string, 0, len(conv.Messages))
	for _, msg := range conv.Messages {
		history = append(history, map[string]string{
			"role":    msg.Role,
			"content": msg.Content,
		})
	}

	return history, nil
}

// ListConversations returns all conversation IDs for a user
func (s *ConversationStore) ListConversations(userID string) []string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var ids []string
	for id, conv := range s.conversations {
		if conv.UserID == userID {
			ids = append(ids, id)
		}
	}

	return ids
}

// UpdateMetadata updates conversation metadata
func (s *ConversationStore) UpdateMetadata(convID string, key string, value any) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	conv, ok := s.conversations[convID]
	if !ok {
		return fmt.Errorf("conversation not found: %s", convID)
	}

	conv.Metadata[key] = value
	conv.UpdatedAt = time.Now()

	return s.saveToFile()
}

// DeleteConversation removes a conversation
func (s *ConversationStore) DeleteConversation(convID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.conversations[convID]; !ok {
		return fmt.Errorf("conversation not found: %s", convID)
	}

	delete(s.conversations, convID)
	return s.saveToFile()
}
