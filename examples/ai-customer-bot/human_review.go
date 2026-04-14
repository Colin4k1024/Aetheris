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
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

// Decision represents the approval decision
type Decision string

const (
	DecisionPending  Decision = "pending"
	DecisionApproved Decision = "approved"
	DecisionRejected Decision = "rejected"
	DecisionExpired  Decision = "expired"
)

// ApprovalRequest represents a request for human approval
type ApprovalRequest struct {
	ID          string                 `json:"id"`
	JobID       string                 `json:"job_id"`
	NodeID      string                 `json:"node_id"`
	Type        string                 `json:"type"` // refund, account_change, etc.
	Title       string                 `json:"title"`
	Description string                 `json:"description"`
	Payload     map[string]interface{} `json:"payload,omitempty"`
	Status      Decision               `json:"status"`
	Response    *ApprovalResponse      `json:"response,omitempty"`
	Timeout     time.Duration          `json:"timeout"`
	ExpiresAt   time.Time              `json:"expires_at"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
}

// ApprovalResponse represents the human's response to an approval request
type ApprovalResponse struct {
	Decision    Decision  `json:"decision"`
	ApproverID  string    `json:"approver_id"`
	Comment     string    `json:"comment,omitempty"`
	RespondedAt time.Time `json:"responded_at"`
}

// HumanReviewer handles human-in-the-loop approval workflows
type HumanReviewer struct {
	mu       sync.RWMutex
	requests map[string]*ApprovalRequest
	results  map[string]chan *ApprovalResponse
}

// NewHumanReviewer creates a new human reviewer
func NewHumanReviewer() *HumanReviewer {
	return &HumanReviewer{
		requests: make(map[string]*ApprovalRequest),
		results:  make(map[string]chan *ApprovalResponse),
	}
}

// RequestApproval requests human approval for a sensitive operation
// This blocks until human approves or rejects (or timeout)
func (hr *HumanReviewer) RequestApproval(ctx context.Context, req *ApprovalRequest) (*ApprovalResponse, error) {
	// Set defaults
	if req.ID == "" {
		req.ID = "approval-" + uuid.New().String()[:8]
	}
	if req.Timeout == 0 {
		req.Timeout = 5 * time.Minute // Default 5 minute timeout
	}
	if req.Status == "" {
		req.Status = DecisionPending
	}

	now := time.Now()
	req.CreatedAt = now
	req.UpdatedAt = now
	req.ExpiresAt = now.Add(req.Timeout)

	// Create response channel
	respCh := make(chan *ApprovalResponse, 1)
	hr.mu.Lock()
	hr.requests[req.ID] = req
	hr.results[req.ID] = respCh
	hr.mu.Unlock()

	// Log the approval request
	fmt.Printf("\n=== Human Approval Required ===\n")
	fmt.Printf("Request ID: %s\n", req.ID)
	fmt.Printf("Type: %s\n", req.Type)
	fmt.Printf("Details: %s\n", req.Description)
	if req.Payload != nil {
		for k, v := range req.Payload {
			fmt.Printf("%s: %v\n", k, v)
		}
	}
	fmt.Printf("Timeout: %v\n", req.Timeout.Round(time.Second))
	fmt.Println()

	// Start timeout goroutine
	go func() {
		select {
		case <-time.After(req.Timeout):
			hr.mu.Lock()
			if hr.requests[req.ID] != nil && hr.requests[req.ID].Status == DecisionPending {
				hr.requests[req.ID].Status = DecisionExpired
				hr.requests[req.ID].UpdatedAt = time.Now()
				respCh <- &ApprovalResponse{
					Decision:    DecisionExpired,
					ApproverID:  "system",
					Comment:     "Request timed out",
					RespondedAt: time.Now(),
				}
			}
			hr.mu.Unlock()
		case <-ctx.Done():
			hr.mu.Lock()
			if hr.requests[req.ID] != nil && hr.requests[req.ID].Status == DecisionPending {
				hr.requests[req.ID].Status = DecisionExpired
				hr.requests[req.ID].UpdatedAt = time.Now()
				respCh <- &ApprovalResponse{
					Decision:    DecisionExpired,
					ApproverID:  "system",
					Comment:     "Context cancelled",
					RespondedAt: time.Now(),
				}
			}
			hr.mu.Unlock()
		}
	}()

	// Wait for response
	select {
	case resp := <-respCh:
		return resp, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// SubmitDecision submits a human's decision for an approval request
func (hr *HumanReviewer) SubmitDecision(reqID string, decision Decision, comment string) error {
	hr.mu.Lock()
	defer hr.mu.Unlock()

	req, ok := hr.requests[reqID]
	if !ok {
		return fmt.Errorf("approval request not found: %s", reqID)
	}

	if req.Status != DecisionPending {
		return fmt.Errorf("approval request already processed: %s (status=%s)", reqID, req.Status)
	}

	if time.Now().After(req.ExpiresAt) {
		req.Status = DecisionExpired
		return fmt.Errorf("approval request has expired: %s", reqID)
	}

	resp := &ApprovalResponse{
		Decision:    decision,
		ApproverID:  "human-operator",
		Comment:     comment,
		RespondedAt: time.Now(),
	}

	req.Status = decision
	req.Response = resp
	req.UpdatedAt = time.Now()

	// Send response if channel is waiting
	if ch, ok := hr.results[reqID]; ok {
		select {
		case ch <- resp:
		default:
		}
	}

	return nil
}

// GetPendingRequests returns all pending approval requests
func (hr *HumanReviewer) GetPendingRequests() []*ApprovalRequest {
	hr.mu.RLock()
	defer hr.mu.RUnlock()

	var pending []*ApprovalRequest
	now := time.Now()
	for _, req := range hr.requests {
		if req.Status == DecisionPending && now.Before(req.ExpiresAt) {
			pending = append(pending, req)
		}
	}

	return pending
}

// InteractiveApprovalPrompt prompts the user for approval in CLI mode
func InteractiveApprovalPrompt(req *ApprovalRequest) (*ApprovalResponse, error) {
	reader := bufio.NewReader(os.Stdin)

	fmt.Printf("Title: %s\n", req.Title)
	fmt.Printf("Description: %s\n", req.Description)
	if req.Payload != nil {
		for k, v := range req.Payload {
			fmt.Printf("  %s: %v\n", k, v)
		}
	}
	fmt.Println()

	// Simple yes/no prompt
	fmt.Print("Approve this request? (yes/no): ")

	input, err := reader.ReadString('\n')
	if err != nil {
		return nil, fmt.Errorf("failed to read input: %w", err)
	}

	input = strings.TrimSpace(strings.ToLower(input))

	switch input {
	case "yes", "y", "是", "同意", "approve":
		return &ApprovalResponse{
			Decision:    DecisionApproved,
			ApproverID:  "cli-user",
			Comment:     "Approved via CLI",
			RespondedAt: time.Now(),
		}, nil
	case "no", "n", "否", "拒绝", "reject":
		fmt.Print("Please provide a reason for rejection: ")
		reason, _ := reader.ReadString('\n')
		return &ApprovalResponse{
			Decision:    DecisionRejected,
			ApproverID:  "cli-user",
			Comment:     strings.TrimSpace(reason),
			RespondedAt: time.Now(),
		}, nil
	default:
		return &ApprovalResponse{
			Decision:    DecisionRejected,
			ApproverID:  "cli-user",
			Comment:     fmt.Sprintf("Invalid input: %s", input),
			RespondedAt: time.Now(),
		}, nil
	}
}

// NeedsApproval determines if a customer service action requires human approval
func NeedsApproval(actionType string) bool {
	// Sensitive actions that require human approval
	approvalRequired := map[string]bool{
		"refund":           true,
		"account_delete":   true,
		"password_reset":   true,
		"address_change":   true,
		"large_refund":     true, // Refunds over certain amount
		"order_cancel":     true,
		"promocode_create": true,
		"admin_action":     true,
	}

	return approvalRequired[actionType]
}

// ExtractActionType determines the action type from user input
func ExtractActionType(userInput string) string {
	input := strings.ToLower(userInput)

	if strings.Contains(input, "refund") || strings.Contains(input, "退款") {
		if strings.Contains(input, "large") || strings.Contains(input, "大") {
			return "large_refund"
		}
		return "refund"
	}

	if strings.Contains(input, "cancel order") || strings.Contains(input, "取消订单") {
		return "order_cancel"
	}

	if strings.Contains(input, "delete account") || strings.Contains(input, "删除账户") {
		return "account_delete"
	}

	if strings.Contains(input, "reset password") || strings.Contains(input, "重置密码") {
		return "password_reset"
	}

	if strings.Contains(input, "change address") || strings.Contains(input, "修改地址") {
		return "address_change"
	}

	return "" // No approval needed
}
