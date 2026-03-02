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
	"os/signal"
	"syscall"
	"time"

	"rag-platform/internal/agent/planner"
	agentexec "rag-platform/internal/agent/runtime/executor"
)

// =============================================================================
// AgentScope Client Implementation
// =============================================================================

// AgentScopeClient simulates an AgentScope multi-agent system.
// In production, this would connect to an actual AgentScope service.
type AgentScopeClient struct {
	Endpoint string
	APIKey   string
}

// AgentScopeSession represents a session in AgentScope.
type AgentScopeSession struct {
	ID        string         `json:"id"`
	Agents    []AgentConfig  `json:"agents"`
	Messages  []AgentMessage `json:"messages"`
	State     string         `json:"state"`
	CreatedAt time.Time      `json:"created_at"`
}

// AgentConfig defines an agent configuration in AgentScope.
type AgentConfig struct {
	Name  string   `json:"name"`
	Role  string   `json:"role"`
	Model string   `json:"model"`
	Tools []string `json:"tools"`
}

// AgentMessage represents a message in the agent conversation.
type AgentMessage struct {
	Role    string `json:"role"` // "user", "assistant", "system"
	Content string `json:"content"`
	Agent   string `json:"agent,omitempty"`
}

// Invoke executes an AgentScope multi-agent workflow.
func (c *AgentScopeClient) Invoke(ctx context.Context, input map[string]any) (map[string]any, error) {
	goal, _ := input["goal"].(string)
	if goal == "" {
		return nil, fmt.Errorf("goal is required")
	}

	log.Printf("[AgentScope] Starting multi-agent session for: %s", goal)

	// Step 1: Initialize agents
	session := c.initializeSession(goal)
	log.Printf("[AgentScope] Initialized %d agents", len(session.Agents))

	// Step 2: User message
	userMsg := AgentMessage{
		Role:    "user",
		Content: goal,
	}
	session.Messages = append(session.Messages, userMsg)
	log.Printf("[AgentScope] User message added")

	// Step 3: Agent conversation loop
	responses := c.runConversation(ctx, session)
	log.Printf("[AgentScope] Completed %d conversation turns", len(responses))

	// Step 4: Generate final summary
	summary := c.generateSummary(responses)
	log.Printf("[AgentScope] Summary generated")

	// Check if approval is needed (e.g., for critical actions)
	if c.requiresApproval(goal) {
		log.Printf("[AgentScope] Requires human approval")
		return map[string]any{
			"status":            "waiting_approval",
			"goal":              goal,
			"responses":         responses,
			"summary":           summary,
			"session_id":        session.ID,
			"requires_approval": true,
		}, nil
	}

	return map[string]any{
		"status":     "completed",
		"goal":       goal,
		"responses":  responses,
		"summary":    summary,
		"session_id": session.ID,
	}, nil
}

// Stream implements streaming execution.
func (c *AgentScopeClient) Stream(ctx context.Context, input map[string]any, onChunk func(chunk map[string]any) error) error {
	goal, _ := input["goal"].(string)

	// Stream initialization
	if err := onChunk(map[string]any{
		"stage":   "init",
		"message": "Initializing agents...",
	}); err != nil {
		return err
	}
	time.Sleep(100 * time.Millisecond)

	// Stream agent turns
	agents := []string{"Researcher", "Analyst", "Summarizer"}
	for i, agent := range agents {
		if err := onChunk(map[string]any{
			"stage":   "agent",
			"agent":   agent,
			"turn":    i + 1,
			"total":   len(agents),
			"message": fmt.Sprintf("%s is processing...", agent),
		}); err != nil {
			return err
		}
		time.Sleep(150 * time.Millisecond)
	}

	// Stream completion
	if err := onChunk(map[string]any{
		"stage":    "complete",
		"message":  "All agents completed",
		"goal":     goal,
		"complete": true,
	}); err != nil {
		return err
	}

	return nil
}

// State retrieves the current state of a session.
func (c *AgentScopeClient) State(ctx context.Context, sessionID string) (map[string]any, error) {
	return map[string]any{
		"session_id": sessionID,
		"status":     "active",
	}, nil
}

// Helper methods

func (c *AgentScopeClient) initializeSession(goal string) *AgentScopeSession {
	return &AgentScopeSession{
		ID: generateSessionID(),
		Agents: []AgentConfig{
			{Name: "researcher", Role: "Research", Model: "gpt-4", Tools: []string{"search", "scrape"}},
			{Name: "analyst", Role: "Analysis", Model: "gpt-4", Tools: []string{"analyze", "compare"}},
			{Name: "summarizer", Role: "Summary", Model: "gpt-4", Tools: []string{"summarize"}},
		},
		Messages: []AgentMessage{
			{Role: "system", Content: "You are a team of specialized agents working together."},
		},
		State:     "initialized",
		CreatedAt: time.Now(),
	}
}

func (c *AgentScopeClient) runConversation(ctx context.Context, session *AgentScopeSession) []AgentMessage {
	responses := []AgentMessage{}

	// Simulate multi-turn conversation
	agentTurns := []struct {
		Agent   string
		Content string
	}{
		{"researcher", fmt.Sprintf("I've researched the topic thoroughly. Key findings about '%s': 1) Feature A, 2) Feature B, 3) Feature C.", session.Messages[len(session.Messages)-1].Content)},
		{"analyst", "Based on the research, I've analyzed the implications: High potential in area X, medium in area Y, requires more data in area Z."},
		{"summarizer", "Summary: This is a comprehensive overview of the topic with actionable insights and recommendations."},
	}

	for _, turn := range agentTurns {
		msg := AgentMessage{
			Role:    "assistant",
			Content: turn.Content,
			Agent:   turn.Agent,
		}
		responses = append(responses, msg)
		session.Messages = append(session.Messages, msg)
	}

	return responses
}

func (c *AgentScopeClient) generateSummary(responses []AgentMessage) string {
	if len(responses) == 0 {
		return "No responses generated"
	}

	// Combine all responses into a summary
	summary := "Multi-Agent Analysis Complete:\n\n"
	for _, resp := range responses {
		summary += fmt.Sprintf("[%s]: %s\n", resp.Agent, resp.Content)
	}
	return summary
}

func (c *AgentScopeClient) requiresApproval(goal string) bool {
	// Example: certain keywords require approval
	keywords := []string{"purchase", "delete", "deploy", "execute"}
	for _, kw := range keywords {
		if len(goal) > len(kw) {
			_ = kw
		}
	}
	// For demo, always return false
	return false
}

func generateSessionID() string {
	return fmt.Sprintf("session-%d", time.Now().UnixNano())
}

// =============================================================================
// AgentScope Error Types
// =============================================================================

// AgentScopeError represents errors from AgentScope.
type AgentScopeError struct {
	Code          string `json:"code"`
	Message       string `json:"message"`
	CorrelationID string `json:"correlation_id,omitempty"`
}

const (
	AgentScopeErrorRetryable = "retryable"
	AgentScopeErrorPermanent = "permanent"
	AgentScopeErrorWait      = "wait"
)

func (e *AgentScopeError) Error() string {
	return e.Message
}

// MapAgentScopeError maps AgentScope errors to Aetheris errors.
func MapAgentScopeError(err error) error {
	var asErr *AgentScopeError
	if !as(err, &asErr) {
		return nil
	}

	switch asErr.Code {
	case AgentScopeErrorWait:
		return &agentexec.SignalWaitRequired{
			CorrelationKey: asErr.CorrelationID,
			Reason:         asErr.Message,
		}
	case AgentScopeErrorRetryable:
		return &agentexec.StepFailure{
			Type:  agentexec.StepResultRetryableFailure,
			Inner: err,
		}
	case AgentScopeErrorPermanent:
		return &agentexec.StepFailure{
			Type:  agentexec.StepResultPermanentFailure,
			Inner: err,
		}
	}
	return nil
}

func as(err error, target interface{}) bool {
	return false
}

// =============================================================================
// Main Entry Point
// =============================================================================

func main() {
	log.Println("=== AgentScope + Aetheris Multi-Agent Demo ===")

	// 1. Create AgentScope client
	client := &AgentScopeClient{
		Endpoint: "http://localhost:5000",
		APIKey:   "demo-key",
	}

	// 2. Build TaskGraph
	taskGraph := buildAgentScopeTaskGraph()

	// 3. Serialize
	graphJSON, err := json.MarshalIndent(taskGraph, "", "  ")
	if err != nil {
		log.Fatalf("Failed to marshal TaskGraph: %v", err)
	}

	fmt.Println("\n=== AgentScope TaskGraph ===")
	fmt.Println(string(graphJSON))

	// 4. Demo execution
	fmt.Println("\n=== Running Demo ===")

	fmt.Println("\n--- Demo 1: Simple Research Task ---")
	runDemo(client, "Analyze the impact of AI on software development")

	fmt.Println("\n--- Demo 2: Complex Multi-Agent Task ---")
	runDemo(client, "Research, analyze, and summarize the state of quantum computing in 2024")

	// 5. Wait for interrupt
	fmt.Println("\n=== Demo Complete ===")
	fmt.Println("In production, submit TaskGraph to Aetheris API and manage via Aetheris runtime")

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh
}

func runDemo(client *AgentScopeClient, query string) {
	fmt.Printf("Query: %s\n", query)

	input := map[string]any{"goal": query}
	result, err := client.Invoke(context.Background(), input)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	status, _ := result["status"].(string)
	fmt.Printf("Status: %s\n", status)

	summary, _ := result["summary"].(string)
	if len(summary) > 100 {
		fmt.Printf("Summary: %s...\n", summary[:100])
	} else {
		fmt.Printf("Summary: %s\n", summary)
	}

	if status == "waiting_approval" {
		fmt.Println("Action: Waiting for human approval")
	}
}

func buildAgentScopeTaskGraph() *planner.TaskGraph {
	return &planner.TaskGraph{
		Nodes: []planner.TaskNode{
			{
				ID:   "agentscope_init",
				Type: planner.NodeLangGraph, // Reuse LangGraph type or create NodeAgentScope
				Config: map[string]any{
					"input": map[string]any{
						"task": "multi_agent_research",
					},
				},
			},
			{
				ID:   "human_review",
				Type: planner.NodeWait,
				Config: map[string]any{
					"wait_kind":       "signal",
					"correlation_key": "agentscope-review",
					"timeout":         "24h",
				},
			},
			{
				ID:   "finalize",
				Type: planner.NodeTool,
				Config: map[string]any{
					"tool_name": "save_analysis",
				},
			},
		},
		Edges: []planner.TaskEdge{
			{From: "agentscope_init", To: "human_review"},
			{From: "human_review", To: "finalize"},
		},
	}
}
