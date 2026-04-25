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

package eino

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/Colin4k1024/Aetheris/v2/internal/runtime/hermes"
)

// Runner interface for step execution (adk.Runner compatible).
type Runner interface {
	Run(ctx context.Context, input map[string]any) (output map[string]any, err error)
	Signal(ctx context.Context, sig string) error
}

// HermesRunner is a Runner implementation that dispatches jobs to Hermes agent
// via ACP protocol. It implements the Runner interface for workflow step execution.
type HermesRunner struct {
	config     *HermesStepConfig
	acpClient  *hermes.ACPClient
	eventCh    chan *RunnerEvent
	mu         sync.RWMutex
	currentSig string
}

// RunnerEvent represents an event from Hermes runner.
type RunnerEvent struct {
	Type      string                 // tool_call, tool_result, session_end, error
	CallID    string                 `json:"call_id,omitempty"`
	ToolName  string                 `json:"tool_name,omitempty"`
	Arguments map[string]interface{} `json:"arguments,omitempty"`
	Result    interface{}            `json:"result,omitempty"`
	Error     string                 `json:"error,omitempty"`
	Status    string                 `json:"status,omitempty"`
}

// NewHermesRunner creates a new HermesRunner with the given configuration.
func NewHermesRunner(config *HermesStepConfig) *HermesRunner {
	hermesURL := hermes.DefaultHermesACPUrl
	callbackURL := config.CallbackURL
	if callbackURL == "" {
		callbackURL = hermes.DefaultAetherisCallbackURL
	}

	return &HermesRunner{
		config:    config,
		acpClient: hermes.NewACPClient(hermesURL, callbackURL),
		eventCh:   make(chan *RunnerEvent, 100),
	}
}

// Run executes the Hermes step with the given input.
// It dispatches a job to Hermes via ACP protocol and waits for completion events.
func (r *HermesRunner) Run(ctx context.Context, input map[string]any) (output map[string]any, err error) {
	runID := generateRunID()
	stepID := r.config.Name

	// Build instruction from input and config
	instruction := r.config.Instruction
	if userInput, ok := input["query"].(string); ok && userInput != "" {
		instruction = instruction + "\n\nUser request: " + userInput
	}

	// Create job dispatch
	dispatch := &hermes.JobDispatch{
		Type:        hermes.EventTypeJobDispatch,
		JobID:       runID,
		StepID:      stepID,
		Instruction: instruction,
		Tools:       r.config.Tools,
		CallbackURL: hermes.DefaultAetherisCallbackURL,
	}

	// Dispatch job to Hermes
	sessionID, err := r.acpClient.DispatchJob(ctx, dispatch)
	if err != nil {
		return nil, fmt.Errorf("dispatch job to Hermes failed: %w", err)
	}

	// Wait for completion or cancellation
	resultCh := make(chan map[string]any, 1)
	errorCh := make(chan error, 1)

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	go r.waitForCompletion(ctx, runID, sessionID, resultCh, errorCh)

	select {
	case <-ctx.Done():
		// Send stop signal on cancellation
		if err := r.Signal(ctx, "stop"); err != nil {
			slog.WarnContext(ctx, "failed to send stop signal to Hermes", "run_id", runID, "session_id", sessionID, "error", err)
		}
		// Drain channels to unblock goroutine
		select {
		case <-resultCh:
		default:
		}
		select {
		case <-errorCh:
		default:
		}
		return nil, ctx.Err()
	case result := <-resultCh:
		return result, nil
	case err := <-errorCh:
		return nil, err
	}
}

// waitForCompletion waits for Hermes session completion and collects events.
func (r *HermesRunner) waitForCompletion(ctx context.Context, runID, sessionID string, resultCh chan<- map[string]any, errorCh chan<- error) {
	var finalOutput map[string]any
	var finalStatus string
	var finalResponse string

	pollInterval := 500 * time.Millisecond
	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			errorCh <- ctx.Err()
			return
		case event := <-r.eventCh:
			if event == nil {
				continue
			}
			switch event.Type {
			case "session_end":
				if event.Result != nil {
					finalResponse = fmt.Sprintf("%v", event.Result)
				}
			case "tool_result":
				// Collect tool results for audit
			}
		case <-ticker.C:
			// Poll Hermes for status
			status, err := r.acpClient.GetJobStatus(ctx, sessionID)
			if err != nil {
				// Continue polling
				continue
			}
			if status.Status == "completed" || status.Status == "failed" || status.Status == "canceled" {
				finalStatus = status.Status
				finalOutput = map[string]any{
					"session_id":     sessionID,
					"status":         finalStatus,
					"final_response": finalResponse,
				}
				resultCh <- finalOutput
				return
			}
		}
	}
}

// Signal sends a control signal to Hermes (stop/pause).
func (r *HermesRunner) Signal(ctx context.Context, sig string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.currentSig = sig
	// Note: Hermes ACP client doesn't have a Signal method yet
	// This will be implemented in Hermes side
	return nil
}

// Events returns a channel for receiving runner events (tool calls, results, etc.).
func (r *HermesRunner) Events() <-chan *RunnerEvent {
	return r.eventCh
}

// HandleToolCallEvent processes a tool.call event from Hermes.
func (r *HermesRunner) HandleToolCallEvent(callID, toolName string, arguments map[string]interface{}) {
	r.eventCh <- &RunnerEvent{
		Type:      "tool_call",
		CallID:    callID,
		ToolName:  toolName,
		Arguments: arguments,
	}
}

// HandleToolResultEvent processes a tool.result event from Hermes.
func (r *HermesRunner) HandleToolResultEvent(callID, toolName string, result interface{}, errMsg string) {
	r.eventCh <- &RunnerEvent{
		Type:     "tool_result",
		CallID:   callID,
		ToolName: toolName,
		Result:   result,
		Error:    errMsg,
	}
}

// HandleSessionEndEvent processes a session.end event from Hermes.
func (r *HermesRunner) HandleSessionEndEvent(sessionID, status string, finalResponse interface{}) {
	r.eventCh <- &RunnerEvent{
		Type:   "session_end",
		Status: status,
		Result: finalResponse,
	}
}

// GetConfig returns the step configuration.
func (r *HermesRunner) GetConfig() *HermesStepConfig {
	return r.config
}

// generateRunID generates a unique run ID for a Hermes job.
func generateRunID() string {
	return "run_" + uuid.New().String()
}
