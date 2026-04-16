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
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// TraceEvent represents a single step in agent execution
type TraceEvent struct {
	Timestamp time.Time              `json:"timestamp"`
	Type      string                 `json:"type"` // "thought", "action", "observation", "result", "error"
	Content   string                 `json:"content"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// RunRequest represents a request to run an agent
type RunRequest struct {
	AgentConfig string                 `json:"agent_config"` // YAML or JSON
	Query       string                 `json:"query"`
	APIKey      string                 `json:"api_key,omitempty"`
	ConfigType  string                 `json:"config_type"` // "yaml" or "json"
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// RunResponse represents the response after starting an agent run
type RunResponse struct {
	RunID     string    `json:"run_id"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
}

// TraceResponse represents the trace of an agent run
type TraceResponse struct {
	RunID  string       `json:"run_id"`
	Status string       `json:"status"` // "running", "completed", "failed"
	Events []TraceEvent `json:"events"`
	Result string       `json:"result,omitempty"`
	Error  string       `json:"error,omitempty"`
}

// PlaygroundServer is the main server for the playground
type PlaygroundServer struct {
	mu   sync.RWMutex
	runs map[string]*RunState
	gin  *gin.Engine
}

type RunState struct {
	ID        string
	Status    string
	CreatedAt time.Time
	Events    []TraceEvent
	Result    string
	Error     string
	query     string
	config    string
	apiKey    string
}

// NewPlaygroundServer creates a new playground server
func NewPlaygroundServer() *PlaygroundServer {
	s := &PlaygroundServer{
		runs: make(map[string]*RunState),
	}
	s.setupRoutes()
	return s
}

func (s *PlaygroundServer) setupRoutes() {
	s.gin = gin.Default()

	// CORS middleware
	s.gin.Use(func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization, X-API-Key")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	})

	// Health check
	s.gin.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	// Run an agent
	s.gin.POST("/api/run", s.handleRun)

	// Get trace
	s.gin.GET("/api/trace/:run_id", s.handleTrace)

	// List runs
	s.gin.GET("/api/runs", s.handleListRuns)

	// Get run status
	s.gin.GET("/api/run/:run_id", s.handleGetRun)

	// Serve static files (frontend)
	s.gin.Static("/static", "./static")
	s.gin.GET("/", func(c *gin.Context) {
		c.File("./static/index.html")
	})
}

func (s *PlaygroundServer) handleRun(c *gin.Context) {
	var req RunRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.Query == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "query is required"})
		return
	}

	if req.ConfigType == "" {
		req.ConfigType = "yaml"
	}

	runID := fmt.Sprintf("run_%d", time.Now().UnixNano())

	runState := &RunState{
		ID:        runID,
		Status:    "running",
		CreatedAt: time.Now(),
		Events:    []TraceEvent{},
		query:     req.Query,
		config:    req.AgentConfig,
		apiKey:    req.APIKey,
	}

	s.mu.Lock()
	s.runs[runID] = runState
	s.mu.Unlock()

	// Run agent asynchronously
	go s.executeAgent(runID, runState)

	c.JSON(http.StatusAccepted, RunResponse{
		RunID:     runID,
		Status:    "running",
		CreatedAt: runState.CreatedAt,
	})
}

func (s *PlaygroundServer) executeAgent(runID string, state *RunState) {
	_, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	// Simulate agent execution with realistic trace events
	events := []TraceEvent{
		{
			Timestamp: time.Now(),
			Type:      "thought",
			Content:   "Received query: " + state.query,
		},
		{
			Timestamp: time.Now(),
			Type:      "thought",
			Content:   "Parsing agent configuration...",
		},
		{
			Timestamp: time.Now(),
			Type:      "action",
			Content:   "Initializing LLM client",
			Metadata: map[string]interface{}{
				"model": "gpt-4o-mini",
			},
		},
	}

	s.updateEvents(runID, events)
	time.Sleep(500 * time.Millisecond)

	// Simulate thinking
	thoughts := []string{
		"Analyzing the query to determine the best approach...",
		"Breaking down the problem into smaller steps...",
		"Consulting available tools for information gathering...",
		"Processing retrieved information...",
		"Formulating response based on gathered data...",
	}

	for _, thought := range thoughts {
		events = []TraceEvent{
			{
				Timestamp: time.Now(),
				Type:      "thought",
				Content:   thought,
			},
		}
		s.updateEvents(runID, events)
		time.Sleep(300 * time.Millisecond)
	}

	// Simulate tool usage if config mentions tools
	if len(state.config) > 0 {
		events = []TraceEvent{
			{
				Timestamp: time.Now(),
				Type:      "action",
				Content:   "Using tool: retriever",
				Metadata: map[string]interface{}{
					"tool":  "retriever",
					"input": state.query,
				},
			},
			{
				Timestamp: time.Now(),
				Type:      "observation",
				Content:   "Retrieved 3 relevant documents from knowledge base",
				Metadata: map[string]interface{}{
					"doc_count": 3,
				},
			},
		}
		s.updateEvents(runID, events)
		time.Sleep(400 * time.Millisecond)
	}

	// Final result
	s.mu.Lock()
	if state, ok := s.runs[runID]; ok {
		state.Status = "completed"
		state.Result = fmt.Sprintf("Based on the query '%s', I can help you with that. This is a simulated response from the CoRag Playground. In a real deployment, this would execute the configured agent with actual LLM calls.", state.query)
		state.Events = append(state.Events, TraceEvent{
			Timestamp: time.Now(),
			Type:      "result",
			Content:   state.Result,
		})
	}
	s.mu.Unlock()
}

func (s *PlaygroundServer) updateEvents(runID string, events []TraceEvent) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if state, ok := s.runs[runID]; ok {
		state.Events = append(state.Events, events...)
	}
}

func (s *PlaygroundServer) handleTrace(c *gin.Context) {
	runID := c.Param("run_id")

	s.mu.RLock()
	state, ok := s.runs[runID]
	s.mu.RUnlock()

	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "run not found"})
		return
	}

	s.mu.RLock()
	events := make([]TraceEvent, len(state.Events))
	copy(events, state.Events)
	status := state.Status
	result := state.Result
	err := state.Error
	s.mu.RUnlock()

	c.JSON(http.StatusOK, TraceResponse{
		RunID:  runID,
		Status: status,
		Events: events,
		Result: result,
		Error:  err,
	})
}

func (s *PlaygroundServer) handleListRuns(c *gin.Context) {
	s.mu.RLock()
	runs := make([]RunResponse, 0, len(s.runs))
	for _, r := range s.runs {
		runs = append(runs, RunResponse{
			RunID:     r.ID,
			Status:    r.Status,
			CreatedAt: r.CreatedAt,
		})
	}
	s.mu.RUnlock()
	c.JSON(http.StatusOK, runs)
}

func (s *PlaygroundServer) handleGetRun(c *gin.Context) {
	runID := c.Param("run_id")

	s.mu.RLock()
	state, ok := s.runs[runID]
	s.mu.RUnlock()

	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "run not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"run_id":     state.ID,
		"status":     state.Status,
		"created_at": state.CreatedAt,
		"result":     state.Result,
		"error":      state.Error,
	})
}

func (s *PlaygroundServer) Run(addr string) error {
	return s.gin.Run(addr)
}

func main() {
	addr := ":8081"
	if p := os.Getenv("PORT"); p != "" {
		addr = ":" + p
	}

	log.Printf("Starting CoRag Playground server on %s", addr)
	server := NewPlaygroundServer()
	if err := server.Run(addr); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
