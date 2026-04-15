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
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/cloudwego/eino/compose"

	"github.com/Colin4k1024/Aetheris/v2/internal/agent/planner"
)

// CustomerServiceWorkflow demonstrates a multi-agent customer service workflow
// Features:
// - Triage: Automatically classifies customer requests
// - Specialized handlers: Route to appropriate department
// - Human approval: Sensitive operations require human confirmation
// - Response aggregation: Combines results from multiple sources

func main() {
	ctx := context.Background()

	// Define the customer service workflow using TaskGraph
	// This workflow handles: order inquiries, refunds, technical support, billing
	workflow := createCustomerServiceWorkflow()

	// Serialize and display the workflow
	bytes, err := json.MarshalIndent(workflow, "", "  ")
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("=== Customer Service Agent Workflow ===")
	fmt.Println(string(bytes))

	// Execute the workflow (requires runtime setup)
	executeWorkflow(ctx, workflow)
}

// createCustomerServiceWorkflow defines the multi-agent customer service workflow
// Flow: triage -> [technical_support | billing | refund | general] -> respond
func createCustomerServiceWorkflow() *planner.TaskGraph {
	return &planner.TaskGraph{
		Nodes: []planner.TaskNode{
			// Triage node: Classify customer request type
			{
				ID:   "triage",
				Type: planner.NodeLLM,
				Config: map[string]any{
					"prompt": `Classify this customer request into one of:
						- technical_support: For product issues, bugs, how-to questions
						- billing: For payment issues, invoices, pricing questions
						- refund: For refund requests (requires human approval)
						- general: For everything else

						Return JSON: {"category": "category_name", "confidence": 0.0-1.0}`,
					"input": "{{input.user_message}}",
				},
			},
			// Technical support branch
			{
				ID:   "technical_support",
				Type: planner.NodeLLM,
				Config: map[string]any{
					"prompt": `You are a technical support specialist. Based on the customer's issue:
						{{input.user_message}}

						Provide helpful troubleshooting steps and solutions.
						Format your response as a structured support reply.`,
					"input": "{{triage.output}} + {{input.user_message}}",
				},
			},
			// Billing branch
			{
				ID:   "billing",
				Type: planner.NodeLLM,
				Config: map[string]any{
					"prompt": `You are a billing specialist. Address the customer's billing concern:
						{{input.user_message}}

						Provide accurate billing information and resolve issues.`,
					"input": "{{triage.output}} + {{input.user_message}}",
				},
			},
			// Refund branch (requires human approval)
			{
				ID:   "refund_request",
				Type: planner.NodeWait,
				Config: map[string]any{
					"wait_kind":       "signal",
					"correlation_key": "refund-approval",
					"reason":          "Refund request requires human approval",
					"expires_at":      "24h",
				},
			},
			// General inquiry handler
			{
				ID:   "general_inquiry",
				Type: planner.NodeLLM,
				Config: map[string]any{
					"prompt": `You are a helpful customer service representative. Answer the customer's question:
						{{input.user_message}}

						Be friendly, helpful, and professional.`,
					"input": "{{input.user_message}}",
				},
			},
			// Human review for sensitive operations
			{
				ID:   "human_review",
				Type: planner.NodeWait,
				Config: map[string]any{
					"wait_kind":       "signal",
					"correlation_key": "manager-escalation",
					"reason":          "Complex issue escalated to human agent",
					"expires_at":      "48h",
				},
			},
			// Response formatter
			{
				ID:   "format_response",
				Type: planner.NodeLLM,
				Config: map[string]any{
					"prompt": `Format the final response for the customer. Make it clear, friendly, and actionable.
						Input from handlers: {{technical_support}} OR {{billing}} OR {{refund_request}} OR {{general_inquiry}}`,
					"input": "Combined response from appropriate handler",
				},
			},
			// Send response to customer
			{
				ID:   "send_response",
				Type: planner.NodeTool,
				Config: map[string]any{
					"tool_name": "send_email",
					"template":  "customer_response",
				},
			},
		},
		Edges: []planner.TaskEdge{
			// All requests start with triage
			{From: "triage", To: "technical_support"},
			{From: "triage", To: "billing"},
			{From: "triage", To: "refund_request"},
			{From: "triage", To: "general_inquiry"},

			// Refunds and escalations need human approval
			{From: "refund_request", To: "human_review"},
			{From: "human_review", To: "format_response"},

			// Direct handlers go to response formatting
			{From: "technical_support", To: "format_response"},
			{From: "billing", To: "format_response"},
			{From: "general_inquiry", To: "format_response"},

			// Final response
			{From: "format_response", To: "send_response"},
		},
	}
}

// executeWorkflow demonstrates how to run the workflow
// In production, this would be handled by the Aetheris runtime
func executeWorkflow(ctx context.Context, workflow *planner.TaskGraph) {
	// This is a placeholder showing how the workflow would be executed
	// The actual execution requires:
	// 1. Setting up the Aetheris runtime engine
	// 2. Loading tools from the registry
	// 3. Creating agents from the workflow configuration
	// 4. Executing with proper context and state management

	fmt.Println("\n=== Workflow Execution ===")
	fmt.Println("To run this workflow with Aetheris:")
	fmt.Println("1. Copy agents.yaml to your configs/ directory")
	fmt.Println("2. Start the Aetheris worker: go run ./cmd/worker")
	fmt.Println("3. Submit jobs via the API or CLI")
	fmt.Println("4. Monitor execution in the dashboard")

	_ = os.WriteFile("customer_service_workflow.json", mustMarshalJSON(workflow), 0644)
	fmt.Println("\nWorkflow saved to customer_service_workflow.json")
}

// Input represents the workflow input
type Input struct {
	UserMessage string `json:"user_message"`
	CustomerID  string `json:"customer_id"`
	Priority    string `json:"priority"`
}

// Output represents the workflow output
type Output struct {
	Response   string `json:"response"`
	Category   string `json:"category"`
	Escalated  bool   `json:"escalated"`
	ApprovedBy string `json:"approved_by,omitempty"`
}

func mustMarshalJSON(v any) []byte {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		log.Fatal(err)
	}
	return b
}
