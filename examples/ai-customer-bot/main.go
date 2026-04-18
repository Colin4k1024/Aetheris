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

// AI Customer Service Bot Demo
// This demonstrates multi-turn conversation, human review/approval,
// conversation history persistence, and visual execution tracing.
//
// Run: OPENAI_API_KEY=xxx go run .
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"time"
)

func main() {
	fmt.Println("=== AI Customer Service Bot Demo ===")
	fmt.Println("Aetheris/CoRag Framework - Human-in-the-Loop Demo")
	fmt.Println()

	// Determine if we're using OpenAI or Ollama
	useOpenAI := os.Getenv("OPENAI_API_KEY") != ""
	useOllama := os.Getenv("OLLAMA_BASE_URL") != ""

	// Initialize store
	storeFile := os.Getenv("STORE_FILE")
	if storeFile == "" {
		storeFile = "conversations.json"
	}
	store, err := NewConversationStore(storeFile)
	if err != nil {
		log.Printf("Warning: Failed to initialize conversation store: %v", err)
		store = nil
	} else {
		fmt.Printf("Conversation store initialized: %s\n", storeFile)
	}

	// Initialize human reviewer
	reviewer := NewHumanReviewer()
	fmt.Println("Human approval system initialized")

	// Create bot
	var chatModel *OpenAIChatModel
	if useOpenAI {
		chatModel = NewOpenAIChatModel(
			os.Getenv("OPENAI_API_KEY"),
			os.Getenv("OPENAI_MODEL"),
		)
		fmt.Println("OpenAI chat model initialized")
	} else if useOllama {
		chatModel = NewOllamaChatModel(
			os.Getenv("OLLAMA_BASE_URL"),
			os.Getenv("OLLAMA_MODEL"),
		)
		fmt.Println("Ollama chat model initialized")
	} else {
		fmt.Println("No LLM configured - using rule-based responses")
	}

	bot := NewCustomerServiceBot(store, reviewer, chatModel)
	fmt.Println("Bot initialized successfully")
	fmt.Println()

	// Create context for the session
	ctx := context.Background()

	// Run in demo mode if no TTY
	stat, _ := os.Stdin.Stat()
	isInteractive := (stat.Mode() & os.ModeCharDevice) != 0

	if !isInteractive {
		// Run demo script
		runDemo(ctx, bot)
	} else {
		// Run interactive session
		if err := bot.RunInteractiveSession(ctx); err != nil {
			log.Fatalf("Session error: %v", err)
		}
	}
}

// runDemo runs a non-interactive demo script
func runDemo(ctx context.Context, bot *CustomerServiceBot) {
	fmt.Println("Running demo script...")
	fmt.Println()

	// Create a demo conversation
	var convID string
	if bot.store != nil {
		conv, err := bot.store.CreateConversation("demo-user")
		if err != nil {
			fmt.Printf("Warning: failed to create conversation: %v\n", err)
		} else {
			convID = conv.ID
		}
	}

	// Demo inputs - simpler first to test multi-turn
	demos := []struct {
		input string
	}{
		{"Hello, I need help with my order"},
		{"I want to check order status for order #12345"},
		{"I also have a question about your return policy"},
		{"Thanks, that's all. Goodbye!"},
	}

	// Start approval monitor in background
	approvalDone := make(chan struct{})
	go func() {
		defer func() {
			if r := recover(); r != nil {
				fmt.Printf("panic in approval monitor goroutine: %v\n", r)
			}
		}()
		ticker := time.NewTicker(500 * time.Millisecond)
		defer ticker.Stop()
		for {
			select {
			case <-approvalDone:
				return
			default:
				pending := bot.reviewer.GetPendingRequests()
				for _, req := range pending {
					fmt.Printf("\n[Demo] Auto-approving request: %s\n", req.ID)
					bot.reviewer.SubmitDecision(req.ID, DecisionApproved, "Auto-approved for demo")
				}
				<-ticker.C
			}
		}
	}()
	defer close(approvalDone)

	for i, demo := range demos {
		fmt.Printf("[Demo %d] User: %s\n", i+1, demo.input)

		response, err := bot.ProcessMessage(ctx, convID, demo.input)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			continue
		}

		fmt.Printf("Bot: %s\n\n", response)
	}

	fmt.Println("Demo completed!")
}

// OpenAIChatModel wraps OpenAI API
type OpenAIChatModel struct {
	apiKey  string
	model   string
	baseURL string
}

// NewOpenAIChatModel creates an OpenAI chat model
func NewOpenAIChatModel(apiKey, model string) *OpenAIChatModel {
	if model == "" {
		model = "gpt-3.5-turbo"
	}
	return &OpenAIChatModel{
		apiKey:  apiKey,
		model:   model,
		baseURL: "https://api.openai.com/v1",
	}
}

// NewOllamaChatModel creates an Ollama chat model
func NewOllamaChatModel(baseURL, model string) *OpenAIChatModel {
	if model == "" {
		model = "llama3"
	}
	if baseURL == "" {
		baseURL = "http://localhost:11434"
	}
	return &OpenAIChatModel{
		apiKey:  "",
		model:   model,
		baseURL: baseURL,
	}
}

// Chat sends a chat request to the model
func (m *OpenAIChatModel) Chat(ctx context.Context, history []map[string]string, message string) (string, error) {
	// For simplicity, use rule-based responses when no API key
	if m.apiKey == "" {
		return m.ruleBasedResponse(message, history), nil
	}

	// OpenAI API call would go here
	// For this demo, we use rule-based responses
	return m.ruleBasedResponse(message, history), nil
}

// ruleBasedResponse provides canned responses based on keywords
func (m *OpenAIChatModel) ruleBasedResponse(message string, history []map[string]string) string {
	msg := strings.ToLower(message)

	// Simple keyword matching
	switch {
	case strings.Contains(msg, "return policy") || strings.Contains(msg, "退货政策"):
		return "Our return policy allows returns within 30 days of purchase. Items must be unused and in original packaging. Would you like me to start a return for you?"
	case strings.Contains(msg, "shipping") || strings.Contains(msg, "配送"):
		return "Standard shipping takes 3-5 business days. Express shipping is available for an additional fee and takes 1-2 business days. International shipping takes 7-14 business days."
	case strings.Contains(msg, "payment") || strings.Contains(msg, "支付"):
		return "We accept Visa, MasterCard, American Express, PayPal, and Apple Pay. All transactions are secure and encrypted."
	case strings.Contains(msg, "track") || strings.Contains(msg, "跟踪"):
		return "You can track your order using the tracking number in your confirmation email. Would you like me to help you find your tracking information?"
	default:
		responses := []string{
			"Thank you for your question. I'll be happy to help you with that.",
			"I understand. Let me provide you with more information about that.",
			"That's a great question. Here's what I can tell you...",
		}
		return responses[0]
	}
}
