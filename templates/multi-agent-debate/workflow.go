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

// MultiAgentDebate demonstrates a multi-agent debate workflow
// Features:
// - Multiple agents with different perspectives
// - Structured rounds of argument
// - Real-time synthesis and moderation
// - Final judgment with scoring

func main() {
	ctx := context.Background()

	// Create the debate workflow
	workflow := createDebateWorkflow()

	// Serialize and display
	bytes, err := json.MarshalIndent(workflow, "", "  ")
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("=== Multi-Agent Debate Workflow ===")
	fmt.Println(string(bytes))

	// Run demo
	executeDemo(ctx, workflow)
}

// DebateInput represents input to the debate workflow
type DebateInput struct {
	Topic     string `json:"topic"`              // The debate topic/question
	Position  string `json:"position,omitempty"` // Optional: specific position to argue
	Rounds    int    `json:"rounds"`             // Number of debate rounds
	TimeLimit int    `json:"time_limit"`         // Time limit per round (minutes)
}

// DebateOutput represents output from the debate workflow
type DebateOutput struct {
	Summary       string             `json:"summary"`
	Verdict       string             `json:"verdict"`
	Scores        map[string]float64 `json:"scores"`        // Agent scores
	KeyPoints     []string           `json:"key_points"`    // Agreed key points
	Disagreements []string           `json:"disagreements"` // Disputed points
}

// createDebateWorkflow defines the multi-agent debate workflow
func createDebateWorkflow() *planner.TaskGraph {
	return &planner.TaskGraph{
		Nodes: []planner.TaskNode{
			// Phase 1: Setup and context gathering
			{
				ID:   "setup",
				Type: planner.NodeLLM,
				Config: map[string]any{
					"prompt": `Setup the debate framework.

						Topic: {{input.topic}}
						Position: {{input.position}}

						Tasks:
						1. Define key terms and scope
						2. Identify stakeholders affected
						3. Set rules for the debate
						4. Create a balanced motion statement

						Return JSON: {"motion": "...", "definitions": {...}, "scope": "..."}`,
					"input": "{{input}}",
				},
			},

			// Phase 2: Research gathering (parallel)
			{
				ID:   "research_for",
				Type: planner.NodeTool,
				Config: map[string]any{
					"tool_name": "retriever",
					"query":     "{{setup.motion}} pro arguments",
					"top_k":     10,
				},
			},
			{
				ID:   "research_against",
				Type: planner.NodeTool,
				Config: map[string]any{
					"tool_name": "retriever",
					"query":     "{{setup.motion}} con arguments",
					"top_k":     10,
				},
			},
			{
				ID:   "research_facts",
				Type: planner.NodeTool,
				Config: map[string]any{
					"tool_name": "web_search",
					"query":     "{{setup.motion}} facts statistics",
					"limit":     10,
				},
			},

			// Phase 3: Opening arguments
			{
				ID:   "opening_for",
				Type: planner.NodeLLM,
				Config: map[string]any{
					"prompt": `You are the ADVOCATE. Present your opening argument FOR the motion.

						Motion: {{setup.motion}}
						Research: {{research_for}}

						Tasks:
						1. State your main thesis clearly
						2. Provide 3-5 supporting arguments
						3. Cite evidence from research
						4. Anticipate counterarguments

						Be persuasive, logical, and factual.`,
					"input": "{{input}}",
				},
			},
			{
				ID:   "opening_against",
				Type: planner.NodeLLM,
				Config: map[string]any{
					"prompt": `You are the OPPONENT. Present your opening argument AGAINST the motion.

						Motion: {{setup.motion}}
						Research: {{research_against}}

						Tasks:
						1. State your main thesis clearly
						2. Provide 3-5 objections
						3. Cite evidence from research
						4. Highlight risks and downsides

						Be persuasive, logical, and factual.`,
					"input": "{{input}}",
				},
			},

			// Phase 4: Rebuttals (can be multiple rounds)
			{
				ID:   "rebuttal_for",
				Type: planner.NodeLLM,
				Config: map[string]any{
					"prompt": `You are the ADVOCATE. Respond to the opponent's arguments.

						Your opening: {{opening_for}}
						Opponent's opening: {{opening_against}}
						Facts: {{research_facts}}

						Tasks:
						1. Address key objections
						2. Refute weak arguments
						3. Reinforce strong points
						4. Provide new evidence if needed

						Be direct and logical.`,
					"input": "{{input}}",
				},
			},
			{
				ID:   "rebuttal_against",
				Type: planner.NodeLLM,
				Config: map[string]any{
					"prompt": `You are the OPPONENT. Respond to the advocate's arguments.

						Your opening: {{opening_against}}
						Opponent's opening: {{opening_for}}
						Facts: {{research_facts}}

						Tasks:
						1. Address key objections
						2. Refute weak arguments
						3. Reinforce strong points
						4. Provide new evidence if needed

						Be direct and logical.`,
					"input": "{{input}}",
				},
			},

			// Phase 5: Cross-examination
			{
				ID:   "question_for",
				Type: planner.NodeLLM,
				Config: map[string]any{
					"prompt": `You are the ADVOCATE. Ask the opponent a critical question.

						Your arguments: {{rebuttal_for}}
						Opponent's arguments: {{rebuttal_against}}

						Ask ONE pointed question that challenges their position.
						Format: "My question is: [question]"`,
					"input": "{{input}}",
				},
			},
			{
				ID:   "answer_against",
				Type: planner.NodeLLM,
				Config: map[string]any{
					"prompt": `You are the OPPONENT. Answer the advocate's question.

						Question: {{question_for}}
						Your arguments: {{rebuttal_against}}

						Provide a direct, honest answer. If you can't fully answer, explain why.`,
					"input": "{{input}}",
				},
			},
			{
				ID:   "question_against",
				Type: planner.NodeLLM,
				Config: map[string]any{
					"prompt": `You are the OPPONENT. Ask the advocate a critical question.

						Your arguments: {{rebuttal_against}}
						Opponent's arguments: {{rebuttal_for}}

						Ask ONE pointed question that challenges their position.
						Format: "My question is: [question]"`,
					"input": "{{input}}",
				},
			},
			{
				ID:   "answer_for",
				Type: planner.NodeLLM,
				Config: map[string]any{
					"prompt": `You are the ADVOCATE. Answer the opponent's question.

						Question: {{question_against}}
						Your arguments: {{rebuttal_for}}

						Provide a direct, honest answer. If you can't fully answer, explain why.`,
					"input": "{{input}}",
				},
			},

			// Phase 6: Moderator synthesis
			{
				ID:   "synthesize",
				Type: planner.NodeLLM,
				Config: map[string]any{
					"prompt": `You are the MODERATOR. Synthesize the debate.

						Advocate's opening: {{opening_for}}
						Opponent's opening: {{opening_against}}
						Advocate's rebuttal: {{rebuttal_for}}
						Opponent's rebuttal: {{rebuttal_against}}
						Cross-examination:
						  Q: {{question_for}} A: {{answer_against}}
						  Q: {{question_against}} A: {{answer_for}}

						Tasks:
						1. Identify points of agreement
						2. Identify key disagreements
						3. Evaluate strength of arguments (1-10)
						4. Provide fair summary

						Return JSON: {
						  "key_agreements": [...],
						  "key_disagreements": [...],
						  "advocate_score": 0-10,
						  "opponent_score": 0-10,
						  "summary": "..."
						}`,
					"input": "{{input}}",
				},
			},

			// Phase 7: Final statements
			{
				ID:   "final_for",
				Type: planner.NodeLLM,
				Config: map[string]any{
					"prompt": `You are the ADVOCATE. Give your closing statement.

						Your arguments: {{rebuttal_for}}
						Moderator's synthesis: {{synthesize}}

						Make a compelling final case. Be concise (2-3 paragraphs).`,
					"input": "{{input}}",
				},
			},
			{
				ID:   "final_against",
				Type: planner.NodeLLM,
				Config: map[string]any{
					"prompt": `You are the OPPONENT. Give your closing statement.

						Your arguments: {{rebuttal_against}}
						Moderator's synthesis: {{synthesize}}

						Make a compelling final case. Be concise (2-3 paragraphs).`,
					"input": "{{input}}",
				},
			},

			// Phase 8: Final judgment
			{
				ID:   "judgment",
				Type: planner.NodeLLM,
				Config: map[string]any{
					"prompt": `You are the MODERATOR. Deliver the final judgment.

						Debate topic: {{setup.motion}}
						Moderator synthesis: {{synthesize}}
						Advocate closing: {{final_for}}
						Opponent closing: {{final_against}}

						Tasks:
						1. Declare a winner (or tie)
						2. Explain the reasoning
						3. Highlight what each side got right
						4. Provide final verdict

						Return JSON: {
						  "verdict": "Advocate/Opponent/Tie",
						  "reasoning": "...",
						  "highlights_for": [...],
						  "highlights_against": [...]
						}`,
					"input": "{{input}}",
				},
			},

			// Phase 9: Generate final report
			{
				ID:   "report",
				Type: planner.NodeLLM,
				Config: map[string]any{
					"prompt": `Generate a comprehensive debate report.

						Topic: {{setup.motion}}
						Synthesis: {{synthesize}}
						Judgment: {{judgment}}

						Format as a structured report with:
						- Executive summary
						- Arguments for
						- Arguments against
						- Key findings
						- Final verdict

						Be objective and comprehensive.`,
					"input": "{{input}}",
				},
			},
		},
		Edges: []planner.TaskEdge{
			// Setup
			{From: "setup", To: "research_for"},
			{From: "setup", To: "research_against"},
			{From: "setup", To: "research_facts"},

			// Opening arguments (parallel)
			{From: "research_for", To: "opening_for"},
			{From: "research_against", To: "opening_against"},

			// Opening complete -> rebuttals
			{From: "opening_for", To: "rebuttal_for"},
			{From: "opening_against", To: "rebuttal_against"},

			// Cross-examination
			{From: "rebuttal_for", To: "question_for"},
			{From: "rebuttal_against", To: "question_against"},
			{From: "question_for", To: "answer_against"},
			{From: "question_against", To: "answer_for"},

			// All cross-examination complete -> synthesis
			{From: "answer_for", To: "synthesize"},
			{From: "answer_against", To: "synthesize"},

			// Closing statements
			{From: "synthesize", To: "final_for"},
			{From: "synthesize", To: "final_against"},

			// Final judgment
			{From: "final_for", To: "judgment"},
			{From: "final_against", To: "judgment"},

			// Report
			{From: "judgment", To: "report"},
		},
	}
}

// CreateSimpleDebate creates a simpler two-agent debate
func CreateSimpleDebate(ctx context.Context) (*planner.TaskGraph, error) {
	return &planner.TaskGraph{
		Nodes: []planner.TaskNode{
			{
				ID:   "research",
				Type: planner.NodeTool,
				Config: map[string]any{
					"tool_name": "web_search",
					"query":     "{{input.topic}}",
					"limit":     5,
				},
			},
			{
				ID:   "argue_for",
				Type: planner.NodeLLM,
				Config: map[string]any{
					"prompt": `Argue FOR: {{input.topic}}. Use research: {{research}}`,
					"input":  "{{input}}",
				},
			},
			{
				ID:   "argue_against",
				Type: planner.NodeLLM,
				Config: map[string]any{
					"prompt": `Argue AGAINST: {{input.topic}}. Use research: {{research}}`,
					"input":  "{{input}}",
				},
			},
			{
				ID:   "synthesize",
				Type: planner.NodeLLM,
				Config: map[string]any{
					"prompt": `Debate on "{{input.topic}}":
						For: {{argue_for}}
						Against: {{argue_against}}

						Provide a balanced synthesis and identify who made stronger points.`,
					"input": "{{input}}",
				},
			},
		},
		Edges: []planner.TaskEdge{
			{From: "research", To: "argue_for"},
			{From: "research", To: "argue_against"},
			{From: "argue_for", To: "synthesize"},
			{From: "argue_against", To: "synthesize"},
		},
	}, nil
}

// executeDemo runs a demonstration
func executeDemo(ctx context.Context, workflow *planner.TaskGraph) {
	fmt.Println("\n=== Multi-Agent Debate Demo ===")
	fmt.Println("This workflow demonstrates:")
	fmt.Println("1. Structured multi-round debates")
	fmt.Println("2. Multiple agents with different roles")
	fmt.Println("3. Real-time synthesis and moderation")
	fmt.Println("4. Final judgment with scoring")

	_ = os.WriteFile("debate_workflow.json", mustMarshalJSON(workflow), 0644)
	fmt.Println("\nWorkflow saved to debate_workflow.json")
}

// CreateDebateGraph creates a simple debate graph using compose
func CreateDebateGraph(ctx context.Context) (compose.Runnable[*DebateInput, *DebateOutput], error) {
	graph := compose.NewGraph[*DebateInput, *DebateOutput]()

	// Simplified debate flow
	graph.AddLambdaNode("setup", compose.InvokableLambda(func(ctx context.Context, input *DebateInput) (*struct {
		Motion string
		Topic  string
	}, error) {
		return &struct {
			Motion string
			Topic  string
		}{
			Motion: "Should AI be regulated?",
			Topic:  input.Topic,
		}, nil
	}))

	graph.AddLambdaNode("debate", compose.InvokableLambda(func(ctx context.Context, input *struct {
		Motion string
		Topic  string
	}) (*struct {
		For     string
		Against string
	}, error) {
		return &struct {
			For     string
			Against string
		}{
			For:     "AI regulation is necessary for safety and accountability",
			Against: "AI regulation would stifle innovation and be premature",
		}, nil
	}))

	graph.AddLambdaNode("synthesize", compose.InvokableLambda(func(ctx context.Context, input *struct {
		For     string
		Against string
	}) (*DebateOutput, error) {
		return &DebateOutput{
			Summary:   "Balanced debate with valid points on both sides",
			Verdict:   "Inconclusive - further discussion needed",
			Scores:    map[string]float64{"advocate": 7.5, "opponent": 7.0},
			KeyPoints: []string{"AI safety is important", "Innovation matters"},
		}, nil
	}))

	graph.AddEdge(compose.START, "setup")
	graph.AddEdge("setup", "debate")
	graph.AddEdge("debate", "synthesize")
	graph.AddEdge("synthesize", compose.END)

	return graph.Compile(ctx)
}

func mustMarshalJSON(v any) []byte {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		log.Fatal(err)
	}
	return b
}
