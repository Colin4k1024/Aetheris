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

// HermesStepConfig is the configuration for a Hermes step in a workflow.
// This config is used by HermesRunner to dispatch jobs to Hermes agent.
type HermesStepConfig struct {
	Name        string   // step name
	Instruction string   // system prompt for Hermes
	Tools       []string // allowed tool names
	MaxSteps    int      // max AI reasoning steps
	CallbackURL string   // Aetheris callback URL for events
}

// HermesACPToolCall represents a tool call event for the audit trail.
// This is used to store tool call events in the Event Store.
type HermesACPToolCall struct {
	RunID           string                 `json:"run_id"`
	StepID          string                 `json:"step_id"`
	ToolName        string                 `json:"tool_name"`
	RequestPayload  map[string]interface{} `json:"request_payload,omitempty"`
	ResponsePayload map[string]interface{} `json:"response_payload,omitempty"`
	IdempotencyKey  string                 `json:"idempotency_key,omitempty"`
	StartedAt       int64                  `json:"started_at,omitempty"`
	EndedAt         int64                  `json:"ended_at,omitempty"`
	Error           string                 `json:"error,omitempty"`
}
