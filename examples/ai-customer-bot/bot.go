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
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"regexp"
	"strings"
	"time"
)

// CustomerServiceBot handles customer service interactions
type CustomerServiceBot struct {
	store         *ConversationStore
	reviewer      *HumanReviewer
	chatModel     *OpenAIChatModel
	maxHistoryLen int
	traceEnabled  bool
}

// TraceEvent represents a step in the execution trace
type TraceEvent struct {
	Step       int       `json:"step"`
	Timestamp  time.Time `json:"timestamp"`
	Action     string    `json:"action"`
	Details    string    `json:"details,omitempty"`
	Input      string    `json:"input,omitempty"`
	Output     string    `json:"output,omitempty"`
	DurationMs int64     `json:"duration_ms,omitempty"`
}

// NewCustomerServiceBot creates a new customer service bot
func NewCustomerServiceBot(store *ConversationStore, reviewer *HumanReviewer, chatModel *OpenAIChatModel) *CustomerServiceBot {
	return &CustomerServiceBot{
		store:         store,
		reviewer:      reviewer,
		chatModel:     chatModel,
		maxHistoryLen: 20,
		traceEnabled:  true,
	}
}

// ProcessMessage handles a user message and returns the bot's response
func (bot *CustomerServiceBot) ProcessMessage(ctx context.Context, convID, userInput string) (string, error) {
	startTime := time.Now()
	step := 0

	// Step 1: Log user input
	step++
	bot.trace(step, "receive_input", "Received user message", userInput)

	// Add user message to conversation
	if bot.store != nil {
		_, err := bot.store.AddMessage(convID, "user", userInput)
		if err != nil {
			log.Printf("Warning: failed to store user message: %v", err)
		}
	}

	// Step 2: Classify intent
	step++
	bot.trace(step, "classify_intent", "Classifying user intent", userInput)
	intent, entities := bot.classifyIntent(userInput)
	bot.trace(step, "intent_classified", fmt.Sprintf("Intent: %s, Entities: %v", intent, entities), userInput)

	var response string
	var err error

	// Step 3: Route based on intent
	step++
	bot.trace(step, "route_request", "Routing request based on intent", userInput)

	switch intent {
	case "greeting":
		response = bot.handleGreeting()
	case "order_inquiry":
		response = bot.handleOrderInquiry(entities)
	case "product_inquiry":
		response = bot.handleProductInquiry(entities)
	case "refund_request":
		response, err = bot.handleRefundRequest(ctx, convID, entities)
	case "cancel_order":
		response, err = bot.handleCancelOrder(ctx, convID, entities)
	case "complaint":
		response = bot.handleComplaint(entities)
	case "general_question":
		response = bot.handleGeneralQuestion(ctx, userInput, convID)
	case "goodbye":
		response = bot.handleGoodbye()
	default:
		response = bot.handleUnknown()
	}

	if err != nil {
		return "", err
	}

	// Step 4: Log bot response
	step++
	bot.trace(step, "send_response", "Sending bot response", response)

	// Add bot response to conversation
	if bot.store != nil {
		_, err := bot.store.AddMessage(convID, "assistant", response)
		if err != nil {
			log.Printf("Warning: failed to store bot response: %v", err)
		}
	}

	// Step 5: Log execution time
	duration := time.Since(startTime)
	bot.trace(step, "complete", fmt.Sprintf("Request completed in %v", duration.Round(time.Millisecond)), "")

	return response, nil
}

// trace logs an execution trace event
func (bot *CustomerServiceBot) trace(step int, action, details, output string) {
	if !bot.traceEnabled {
		return
	}

	event := TraceEvent{
		Step:      step,
		Timestamp: time.Now(),
		Action:    action,
		Details:   details,
		Output:    output,
	}

	eventJSON, _ := json.Marshal(event)
	fmt.Printf("[TRACE] %s\n", string(eventJSON))
}

// classifyIntent determines the user's intent from their message
func (bot *CustomerServiceBot) classifyIntent(input string) (string, map[string]string) {
	input = strings.ToLower(input)
	entities := make(map[string]string)

	// Extract order number
	orderRegex := regexp.MustCompile(`(?i)order\s*#?(\d+)|订单\s*#?(\d+)`)
	if match := orderRegex.FindStringSubmatch(input); len(match) > 0 {
		if match[1] != "" {
			entities["order_id"] = match[1]
		} else if match[2] != "" {
			entities["order_id"] = match[2]
		}
	}

	// Extract amount
	amountRegex := regexp.MustCompile(`\$?\s*(\d+(?:\.\d{2})?)|(\d+(?:\.\d{2})?)\s*美元|(\d+(?:\.\d{2})?)\s*元`)
	if match := amountRegex.FindStringSubmatch(input); len(match) > 0 {
		for _, m := range match[1:] {
			if m != "" {
				entities["amount"] = m
				break
			}
		}
	}

	// Intent classification
	switch {
	case strings.Contains(input, "hello") || strings.Contains(input, "hi") ||
		strings.Contains(input, "你好") || strings.Contains(input, "嗨"):
		return "greeting", entities

	case strings.Contains(input, "order status") || strings.Contains(input, "订单状态") ||
		strings.Contains(input, "where is my order") || strings.Contains(input, "我的订单在哪"):
		return "order_inquiry", entities

	case strings.Contains(input, "refund") || strings.Contains(input, "退款") ||
		strings.Contains(input, "return"):
		return "refund_request", entities

	case strings.Contains(input, "cancel") && (strings.Contains(input, "order") || strings.Contains(input, "订单")):
		return "cancel_order", entities

	case strings.Contains(input, "product") && strings.Contains(input, "info"):
		return "product_inquiry", entities

	case strings.Contains(input, "complaint") || strings.Contains(input, "投诉") ||
		strings.Contains(input, "unhappy") || strings.Contains(input, "不满意"):
		return "complaint", entities

	case strings.Contains(input, "bye") || strings.Contains(input, "goodbye") ||
		strings.Contains(input, "谢谢") || strings.Contains(input, "thanks"):
		return "goodbye", entities

	case strings.Contains(input, "?"), strings.Contains(input, "？"), strings.Contains(input, "how"),
		strings.Contains(input, "what"), strings.Contains(input, "怎么"), strings.Contains(input, "如何"):
		return "general_question", entities

	default:
		return "unknown", entities
	}
}

// handleGreeting returns a greeting response
func (bot *CustomerServiceBot) handleGreeting() string {
	greetings := []string{
		"Hello! Welcome to our customer service. How can I assist you today?",
		"Hi there! I'm here to help with your inquiries. What can I do for you?",
		"Good day! How may I help you today?",
	}
	return greetings[time.Now().UnixNano()%int64(len(greetings))]
}

// handleOrderInquiry handles order status inquiries
func (bot *CustomerServiceBot) handleOrderInquiry(entities map[string]string) string {
	orderID, ok := entities["order_id"]
	if !ok {
		return "I'd be happy to help you check your order status. Could you please provide your order number?"
	}

	// Simulated order lookup
	statuses := []string{
		"Processing - expected to ship within 1-2 business days",
		"Shipped - tracking number: TRK" + orderID + "2024",
		"Delivered - signed by recipient on " + time.Now().AddDate(0, 0, -2).Format("Jan 2, 2006"),
		"Processing - your items are being prepared for shipment",
	}
	status := statuses[time.Now().UnixNano()%int64(len(statuses))]

	return fmt.Sprintf("I found your order #%s. Current status: %s. Is there anything else you'd like to know about this order?", orderID, status)
}

// handleProductInquiry handles product information requests
func (bot *CustomerServiceBot) handleProductInquiry(entities map[string]string) string {
	return "I'd be happy to help with product information. Could you tell me which product you're interested in? Our catalog includes electronics, home appliances, and more."
}

// handleRefundRequest handles refund requests with human approval
func (bot *CustomerServiceBot) handleRefundRequest(ctx context.Context, convID string, entities map[string]string) (string, error) {
	amount := "$99.99" // Default simulated amount
	if amt, ok := entities["amount"]; ok {
		amount = "$" + amt
	}

	// Check if approval is required based on action type
	actionType := ExtractActionType("refund request")
	if NeedsApproval(actionType) {
		// Create approval request
		req := &ApprovalRequest{
			JobID:       convID,
			Type:        "refund",
			Title:       "Refund Request",
			Description: fmt.Sprintf("Customer requesting refund for amount: %s", amount),
			Payload: map[string]interface{}{
				"order_id": entities["order_id"],
				"amount":   amount,
				"reason":   "Customer request",
			},
			Timeout: 5 * time.Minute,
		}

		fmt.Println("\n[Waiting for human approval...]")

		// Request human approval
		resp, err := bot.reviewer.RequestApproval(ctx, req)
		if err != nil {
			return "", fmt.Errorf("approval request failed: %w", err)
		}

		switch resp.Decision {
		case DecisionApproved:
			return fmt.Sprintf("Great! Your refund of %s has been approved and will be processed within 3-5 business days. You'll receive a confirmation email shortly.", amount), nil
		case DecisionRejected:
			return fmt.Sprintf("I'm sorry, but your refund request has been declined. Reason: %s. If you have any questions, please contact our support team.", resp.Comment), nil
		case DecisionExpired:
			return "Your refund request has timed out. Please try again or contact our support team for assistance.", nil
		default:
			return "There was an issue processing your refund request. Please try again later.", nil
		}
	}

	// No approval needed - process directly
	return fmt.Sprintf("I've processed your refund request for %s. The refund will be credited to your original payment method within 3-5 business days.", amount), nil
}

// handleCancelOrder handles order cancellation with human approval
func (bot *CustomerServiceBot) handleCancelOrder(ctx context.Context, convID string, entities map[string]string) (string, error) {
	orderID := entities["order_id"]
	if orderID == "" {
		orderID = "unknown"
	}

	actionType := ExtractActionType("cancel order")
	if NeedsApproval(actionType) {
		req := &ApprovalRequest{
			JobID:       convID,
			Type:        "order_cancel",
			Title:       "Order Cancellation",
			Description: fmt.Sprintf("Request to cancel order #%s", orderID),
			Payload: map[string]interface{}{
				"order_id": orderID,
			},
			Timeout: 5 * time.Minute,
		}

		fmt.Println("\n[Waiting for human approval...]")

		resp, err := bot.reviewer.RequestApproval(ctx, req)
		if err != nil {
			return "", fmt.Errorf("approval request failed: %w", err)
		}

		switch resp.Decision {
		case DecisionApproved:
			return fmt.Sprintf("Your order #%s has been successfully cancelled. You'll receive a full refund within 5-7 business days.", orderID), nil
		case DecisionRejected:
			return fmt.Sprintf("I'm sorry, but your cancellation request for order #%s has been declined. Reason: %s", orderID, resp.Comment), nil
		case DecisionExpired:
			return "Your cancellation request has timed out. Please try again or contact our support team.", nil
		default:
			return "There was an issue processing your cancellation request. Please try again later.", nil
		}
	}

	return fmt.Sprintf("Your order #%s has been cancelled.", orderID), nil
}

// handleComplaint handles customer complaints
func (bot *CustomerServiceBot) handleComplaint(entities map[string]string) string {
	return "I'm truly sorry to hear that you're experiencing issues. I want to help resolve this for you. Could you please provide more details about the problem? Our team will review your concern and get back to you within 24 hours."
}

// handleGeneralQuestion handles general questions using the LLM
func (bot *CustomerServiceBot) handleGeneralQuestion(ctx context.Context, userInput, convID string) string {
	if bot.chatModel != nil {
		// Get conversation history
		history, _ := bot.store.GetConversationHistory(convID)

		// Get LLM response
		response, err := bot.chatModel.Chat(ctx, history, userInput)
		if err != nil {
			log.Printf("LLM error: %v", err)
			return "I'm having trouble processing that right now. Could you please rephrase your question?"
		}

		return response
	}

	// Fallback responses when no LLM is available
	responses := []string{
		"That's an interesting question. Let me look into that for you.",
		"I understand. Let me provide some information on that topic.",
		"Thank you for asking. Here's what I can tell you...",
		"I appreciate your question. Based on our knowledge base...",
	}
	return responses[time.Now().UnixNano()%int64(len(responses))]
}

// handleGoodbye returns a farewell response
func (bot *CustomerServiceBot) handleGoodbye() string {
	return "Thank you for contacting us! If you have any more questions in the future, don't hesitate to reach out. Have a great day!"
}

// handleUnknown handles unrecognized inputs
func (bot *CustomerServiceBot) handleUnknown() string {
	return "I'm not quite sure I understood that. Could you please rephrase? I can help you with order inquiries, refunds, product information, and more."
}

// RunInteractiveSession starts an interactive CLI session
func (bot *CustomerServiceBot) RunInteractiveSession(ctx context.Context) error {
	reader := bufio.NewReader(os.Stdin)

	// Create a new conversation
	var convID string
	if bot.store != nil {
		conv := bot.store.CreateConversation("default-user")
		convID = conv.ID
		fmt.Printf("Conversation ID: %s\n", convID)
	} else {
		convID = "temp-session"
	}

	fmt.Println("\n=== AI Customer Service Bot ===")
	fmt.Println("Type 'quit' or 'exit' to end the session")
	fmt.Println("Type 'history' to see conversation history")
	fmt.Println()

	// Initial greeting
	bot.ProcessMessage(ctx, convID, "hello")

	for {
		fmt.Print("You: ")
		input, err := reader.ReadString('\n')
		if err != nil {
			return fmt.Errorf("failed to read input: %w", err)
		}

		input = strings.TrimSpace(input)
		if input == "" {
			continue
		}

		// Handle special commands
		switch strings.ToLower(input) {
		case "quit", "exit", "退出":
			fmt.Println("\nThank you for chatting with us! Have a great day!")
			return nil
		case "history", "历史":
			if bot.store != nil {
				msgs, _ := bot.store.GetMessages(convID)
				fmt.Println("\n=== Conversation History ===")
				for _, m := range msgs {
					role := "User"
					if m.Role == "assistant" {
						role = "Bot"
					}
					fmt.Printf("[%s] %s\n", role, m.Content)
				}
				fmt.Println()
			}
			continue
		}

		// Process the message
		response, err := bot.ProcessMessage(ctx, convID, input)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			continue
		}

		fmt.Printf("Bot: %s\n\n", response)
	}
}

// CreateDemoTools creates the tools for the eino agent
func CreateDemoTools(bot *CustomerServiceBot) []map[string]any {
	return []map[string]any{
		{
			"name":        "check_order_status",
			"description": "Check the status of an order by order ID",
			"parameters": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"order_id": map[string]any{
						"type":        "string",
						"description": "The order ID to check",
					},
				},
				"required": []string{"order_id"},
			},
		},
		{
			"name":        "request_refund",
			"description": "Request a refund for an order. May require human approval for large amounts.",
			"parameters": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"order_id": map[string]any{
						"type":        "string",
						"description": "The order ID for the refund",
					},
					"amount": map[string]any{
						"type":        "string",
						"description": "The amount to refund",
					},
					"reason": map[string]any{
						"type":        "string",
						"description": "Reason for the refund",
					},
				},
				"required": []string{"order_id", "reason"},
			},
		},
		{
			"name":        "cancel_order",
			"description": "Cancel an order. May require human approval.",
			"parameters": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"order_id": map[string]any{
						"type":        "string",
						"description": "The order ID to cancel",
					},
					"reason": map[string]any{
						"type":        "string",
						"description": "Reason for cancellation",
					},
				},
				"required": []string{"order_id", "reason"},
			},
		},
	}
}
