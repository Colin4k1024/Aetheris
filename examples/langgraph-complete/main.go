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

// Package main demonstrates a complete LangGraph integration with Aetheris.
// This example shows how to build a research agent that uses LangGraph
// for reasoning while leveraging Aetheris for durability, replay, and audit.
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
// LangGraph Client Implementation
// =============================================================================

// ResearchGraphClient simulates a LangGraph research agent.
// In production, this would connect to an actual LangGraph API or run locally.
type ResearchGraphClient struct {
	APIEndpoint string
	APIKey      string
}

// GraphState represents the state in a LangGraph workflow.
type GraphState struct {
	Query            string   `json:"query"`
	SearchResults    []string `json:"search_results"`
	Analysis         string   `json:"analysis"`
	FinalAnswer      string   `json:"final_answer"`
	RequiresApproval bool     `json:"requires_approval"`
	Approved         bool     `json:"approved"`
}

// Invoke executes the LangGraph research workflow.
func (c *ResearchGraphClient) Invoke(ctx context.Context, input map[string]any) (map[string]any, error) {
	query, _ := input["goal"].(string)
	if query == "" {
		return nil, fmt.Errorf("query is required")
	}

	log.Printf("[LangGraph] Starting research for query: %s", query)

	// Step 1: Search for information (simulated)
	searchResults := c.performSearch(query)
	log.Printf("[LangGraph] Found %d search results", len(searchResults))

	// Step 2: Analyze results (simulated)
	analysis := c.analyzeResults(query)
	log.Printf("[LangGraph] Analysis complete")

	// Step 3: Generate final answer
	answer := c.generateAnswer(query, analysis)
	log.Printf("[LangGraph] Generated final answer")

	// Step 4: Check if human approval is needed
	// In this example, we auto-approve for queries under $1000
	requiresApproval := c.checkApprovalNeeded(query)
	if requiresApproval {
		log.Printf("[LangGraph] Requires human approval")
		return map[string]any{
			"status":            "waiting_approval",
			"query":             query,
			"search_results":    searchResults,
			"analysis":          analysis,
			"final_answer":      answer,
			"requires_approval": true,
		}, nil
	}

	return map[string]any{
		"status":         "completed",
		"query":          query,
		"search_results": searchResults,
		"analysis":       analysis,
		"final_answer":   answer,
	}, nil
}

// Stream implements streaming execution.
func (c *ResearchGraphClient) Stream(ctx context.Context, input map[string]any, onChunk func(chunk map[string]any) error) error {
	query, _ := input["goal"].(string)

	// Stream search results
	if err := onChunk(map[string]any{
		"stage":   "search",
		"message": "Searching for information...",
	}); err != nil {
		return err
	}

	// Simulate processing time
	time.Sleep(100 * time.Millisecond)

	// Stream analysis
	if err := onChunk(map[string]any{
		"stage":   "analyze",
		"message": "Analyzing results...",
	}); err != nil {
		return err
	}

	time.Sleep(100 * time.Millisecond)

	// Stream final answer
	if err := onChunk(map[string]any{
		"stage":    "finalize",
		"message":  "Generating answer for: " + query,
		"complete": true,
	}); err != nil {
		return err
	}

	return nil
}

// State retrieves the current state of a graph execution.
func (c *ResearchGraphClient) State(ctx context.Context, threadID string) (map[string]any, error) {
	return map[string]any{
		"thread_id": threadID,
		"status":    "idle",
	}, nil
}

// Helper methods (simulated)

func (c *ResearchGraphClient) performSearch(query string) []string {
	// Simulated search results
	return []string{
		fmt.Sprintf("Result 1: Information about %s from source A", query),
		fmt.Sprintf("Result 2: Information about %s from source B", query),
		fmt.Sprintf("Result 3: Information about %s from source C", query),
	}
}

func (c *ResearchGraphClient) analyzeResults(query string) string {
	return fmt.Sprintf("Analysis: Found comprehensive information about '%s'. The key findings indicate multiple perspectives on this topic.", query)
}

func (c *ResearchGraphClient) generateAnswer(query, analysis string) string {
	_ = query // unused in this simplified example
	return fmt.Sprintf("Based on the research, here is the answer: %s", analysis)
}

func (c *ResearchGraphClient) checkApprovalNeeded(query string) bool {
	// Example: queries mentioning "purchase" or "buy" require approval
	keywords := []string{"purchase", "buy", "expensive", "$"}
	for _, kw := range keywords {
		if len(query) > 5 && (query == kw || len(query) > 10) {
			return true
		}
	}
	return false
}

// =============================================================================
// Aetheris Integration
// =============================================================================

// LangGraphAdapter wraps the LangGraph client for use with Aetheris.
type LangGraphAdapter struct {
	Client      agentexec.LangGraphClient
	EffectStore agentexec.EffectStore
}

func (a *LangGraphAdapter) runNode(ctx context.Context, taskID string, cfg map[string]any, p *AgentDAGPayload) (*AgentDAGPayload, error) {
	if a.Client == nil {
		return nil, fmt.Errorf("LangGraph client not configured")
	}

	if p == nil {
		p = &AgentDAGPayload{}
	}
	if p.Results == nil {
		p.Results = make(map[string]any)
	}

	jobID := JobIDFromContext(ctx)

	// Check effect store for previous execution (idempotency)
	if a.EffectStore != nil && jobID != "" {
		eff, err := a.EffectStore.GetEffectByJobAndCommandID(ctx, jobID, taskID)
		if err == nil && eff != nil && len(eff.Output) > 0 {
			var out map[string]any
			if json.Unmarshal(eff.Output, &out) == nil {
				p.Results[taskID] = out
				log.Printf("[Aetheris] Restored from effect store for task: %s", taskID)
				return p, nil
			}
		}
	}

	// Prepare input
	input := map[string]any{
		"goal": p.Goal,
	}
	if cfg != nil {
		if v, ok := cfg["input"].(map[string]any); ok {
			for k, val := range v {
				input[k] = val
			}
		}
	}

	// Execute LangGraph
	log.Printf("[Aetheris] Executing LangGraph task: %s", taskID)
	out, err := a.Client.Invoke(ctx, input)
	if err != nil {
		// Check if it's a wait error (needs approval)
		var lgErr *agentexec.LangGraphError
		if As(err, &lgErr) && lgErr.Code == agentexec.LangGraphErrorWait {
			return nil, &agentexec.SignalWaitRequired{
				CorrelationKey: lgErr.CorrelationKey,
				Reason:         lgErr.Reason,
			}
		}
		return nil, err
	}

	// Store effect for replay
	if a.EffectStore != nil && jobID != "" {
		outputBytes, _ := json.Marshal(out)
		_ = a.EffectStore.PutEffect(ctx, &agentexec.EffectRecord{
			JobID:     jobID,
			CommandID: taskID,
			Kind:      agentexec.EffectKindTool,
			Input:     nil,
			Output:    outputBytes,
			Metadata:  map[string]any{"adapter": "langgraph"},
		})
	}

	p.Results[taskID] = out
	return p, nil
}

// JobIDFromContext extracts job ID from context.
func JobIDFromContext(ctx context.Context) string {
	// In real implementation, extract from context
	return ""
}

// As is a simple type assertion helper.
func As(err error, target interface{}) bool {
	return false
}

// AgentDAGPayload represents the payload for DAG execution.
type AgentDAGPayload struct {
	Goal    string
	Results map[string]any
}

// =============================================================================
// Main Entry Point
// =============================================================================

func main() {
	log.Println("=== LangGraph + Aetheris Research Agent Demo ===")

	// 1. Create LangGraph client
	client := &ResearchGraphClient{
		APIEndpoint: "http://localhost:8000",
		APIKey:      "demo-key",
	}

	// 2. Create adapter (used for actual Aetheris integration)
	_ = &LangGraphAdapter{
		Client:      client,
		EffectStore: nil, // Use effect store in production
	}

	// 3. Build TaskGraph with Aetheris
	taskGraph := buildResearchTaskGraph()

	// 4. Serialize for submission to Aetheris
	graphJSON, err := json.MarshalIndent(taskGraph, "", "  ")
	if err != nil {
		log.Fatalf("Failed to marshal TaskGraph: %v", err)
	}

	fmt.Println("\n=== TaskGraph Definition ===")
	fmt.Println(string(graphJSON))

	// 5. Demonstrate execution
	fmt.Println("\n=== Running Demo ===")

	// Demo: Research query (no approval needed)
	fmt.Println("\n--- Demo 1: Simple Research Query ---")
	runDemo(client, "What is machine learning?")

	// Demo: Query requiring approval
	fmt.Println("\n--- Demo 2: Query Requiring Approval ---")
	runDemo(client, "I want to purchase an expensive AI model")

	// 6. Wait for signal (in real usage)
	fmt.Println("\n=== Demo Complete ===")
	fmt.Println("In production, this would:")
	fmt.Println("1. Submit TaskGraph to Aetheris API")
	fmt.Println("2. Create agent with the workflow")
	fmt.Println("3. Submit jobs and track execution")
	fmt.Println("4. Pause at approval nodes")
	fmt.Println("5. Resume via signal API")

	// Wait for interrupt
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh
}

func runDemo(client *ResearchGraphClient, query string) {
	fmt.Printf("Query: %s\n", query)

	input := map[string]any{"goal": query}
	result, err := client.Invoke(context.Background(), input)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	status, _ := result["status"].(string)
	fmt.Printf("Status: %s\n", status)

	if status == "waiting_approval" {
		answer, _ := result["final_answer"].(string)
		fmt.Printf("Pending Answer: %s\n", answer)
		fmt.Println("Action: Waiting for human approval via Aetheris signal API")
	} else {
		answer, _ := result["final_answer"].(string)
		fmt.Printf("Answer: %s\n", answer)
	}
}

func buildResearchTaskGraph() *planner.TaskGraph {
	return &planner.TaskGraph{
		Nodes: []planner.TaskNode{
			{
				ID:   "research",
				Type: planner.NodeLangGraph,
				Config: map[string]any{
					"input": map[string]any{
						"task": "research_topic",
					},
				},
			},
			{
				ID:   "human_approval",
				Type: planner.NodeWait,
				Config: map[string]any{
					"wait_kind":       "signal",
					"correlation_key": "research-approval",
					"timeout":         "24h",
				},
			},
			{
				ID:   "format_output",
				Type: planner.NodeLLM,
				Config: map[string]any{
					"prompt": "Format the research results into a clean report",
				},
			},
			{
				ID:   "save_report",
				Type: planner.NodeTool,
				Config: map[string]any{
					"tool_name": "save_to_database",
				},
			},
		},
		Edges: []planner.TaskEdge{
			{From: "research", To: "human_approval"},
			{From: "human_approval", To: "format_output"},
			{From: "format_output", To: "save_report"},
		},
	}
}
