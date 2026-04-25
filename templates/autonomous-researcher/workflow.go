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

// AutonomousResearcher demonstrates an autonomous research workflow
// Features:
// - Autonomous topic exploration
// - Multi-source information gathering
// - Verification and fact-checking
// - Structured report generation

func main() {
	ctx := context.Background()

	// Create the research workflow
	workflow := createResearchWorkflow()

	// Serialize and display
	bytes, err := json.MarshalIndent(workflow, "", "  ")
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("=== Autonomous Researcher Workflow ===")
	fmt.Println(string(bytes))

	// Run demo
	executeDemo(ctx, workflow)
}

// ResearchInput represents input to the research workflow
type ResearchInput struct {
	Topic        string   `json:"topic"`        // Research topic/question
	Depth        string   `json:"depth"`        // shallow, medium, deep
	Audience     string   `json:"audience"`     // Target audience
	FocusAreas   []string `json:"focus_areas"`  // Specific areas to emphasize
	SourceTypes  []string `json:"source_types"` // web, documents, knowledge_base
	MaxSources   int      `json:"max_sources"`  // Maximum sources to consult
	Deliverables []string `json:"deliverables"` // report, summary, presentation
}

// ResearchOutput represents output from the research workflow
type ResearchOutput struct {
	Title            string   `json:"title"`
	ExecutiveSummary string   `json:"executive_summary"`
	KeyFindings      []string `json:"key_findings"`
	Report           string   `json:"report"`
	Sources          []Source `json:"sources"`
	Confidence       float64  `json:"confidence"`
	Gaps             []string `json:"gaps"`       // Areas that need more research
	NextSteps        []string `json:"next_steps"` // Recommended follow-up research
}

// Source represents a cited source
type Source struct {
	Title     string `json:"title"`
	URL       string `json:"url,omitempty"`
	Type      string `json:"type"` // web, document, database
	Relevance string `json:"relevance"`
}

// createResearchWorkflow defines the autonomous research workflow
func createResearchWorkflow() *planner.TaskGraph {
	return &planner.TaskGraph{
		Nodes: []planner.TaskNode{
			// Phase 1: Topic Analysis and Planning
			{
				ID:   "analyze_topic",
				Type: planner.NodeLLM,
				Config: map[string]any{
					"prompt": `Analyze the research topic and create a research plan.

						Topic: {{input.topic}}
						Depth: {{input.depth}}
						Audience: {{input.audience}}
						Focus areas: {{input.focus_areas}}

						Tasks:
						1. Define key questions to answer
						2. Identify required knowledge areas
						3. Determine information sources needed
						4. Estimate time/resources required
						5. Identify potential challenges

						Return JSON: {
						  "key_questions": [...],
						  "knowledge_areas": [...],
						  "source_plan": {...},
						  "estimated_sources_needed": number
						}`,
					"input": "{{input}}",
				},
			},

			// Phase 2: Background Research (parallel searches)
			{
				ID:   "search_web",
				Type: planner.NodeTool,
				Config: map[string]any{
					"tool_name": "web_search",
					"query":     "{{analyze_topic.key_questions}}",
					"limit":     "{{input.max_sources}}",
				},
			},
			{
				ID:   "search_knowledge",
				Type: planner.NodeTool,
				Config: map[string]any{
					"tool_name": "retriever",
					"query":     "{{input.topic}}",
					"top_k":     "{{input.max_sources}}",
				},
			},
			{
				ID:   "search_documents",
				Type: planner.NodeTool,
				Config: map[string]any{
					"tool_name": "document_loader",
					"paths":     "{{input.source_types}}",
				},
			},

			// Phase 3: Deep Dive Research
			// Multiple specialized searches based on knowledge areas
			{
				ID:   "deep_dive_1",
				Type: planner.NodeLLM,
				Config: map[string]any{
					"prompt": `Conduct deep research on the first knowledge area.

						Topic: {{input.topic}}
						Knowledge areas: {{analyze_topic.knowledge_areas}}
						Web results: {{search_web}}
						Knowledge base: {{search_knowledge}}

						Focus on the first knowledge area from the list.
						Find specific facts, statistics, and expert opinions.

						Return detailed findings with sources.`,
					"input": "{{input}}",
				},
			},
			{
				ID:   "deep_dive_2",
				Type: planner.NodeLLM,
				Config: map[string]any{
					"prompt": `Conduct deep research on the second knowledge area.

						Topic: {{input.topic}}
						Knowledge areas: {{analyze_topic.knowledge_areas}}
						Web results: {{search_web}}
						Knowledge base: {{search_knowledge}}

						Focus on the second knowledge area.
						Find specific facts, statistics, and expert opinions.

						Return detailed findings with sources.`,
					"input": "{{input}}",
				},
			},
			{
				ID:   "deep_dive_3",
				Type: planner.NodeLLM,
				Config: map[string]any{
					"prompt": `Conduct deep research on the third knowledge area.

						Topic: {{input.topic}}
						Knowledge areas: {{analyze_topic.knowledge_areas}}
						Web results: {{search_web}}
						Knowledge base: {{search_knowledge}}

						Focus on the third knowledge area.
						Find specific facts, statistics, and expert opinions.

						Return detailed findings with sources.`,
					"input": "{{input}}",
				},
			},

			// Phase 4: Verification
			{
				ID:   "verify_findings",
				Type: planner.NodeLLM,
				Config: map[string]any{
					"prompt": `Verify research findings for accuracy.

						Topic: {{input.topic}}
						Deep dive 1: {{deep_dive_1}}
						Deep dive 2: {{deep_dive_2}}
						Deep dive 3: {{deep_dive_3}}

						Tasks:
						1. Check facts against multiple sources
						2. Identify unsupported claims
						3. Assess source reliability
						4. Note any contradictions
						5. Rate confidence level (0-1)

						Return JSON: {
						  "verified": [...],
						  "unverified": [...],
						  "contradictions": [...],
						  "confidence": 0.0-1.0,
						  "gaps": [...]
						}`,
					"input": "{{input}}",
				},
			},

			// Phase 5: Analysis and Synthesis
			{
				ID:   "analyze_patterns",
				Type: planner.NodeLLM,
				Config: map[string]any{
					"prompt": `Analyze research findings for patterns and insights.

						Topic: {{input.topic}}
						Verified findings: {{verify_findings.verified}}
						Web search: {{search_web}}

						Tasks:
						1. Identify common themes
						2. Note trends and patterns
						3. Draw logical conclusions
						4. Identify cause-effect relationships
						5. Note expert consensus

						Return structured analysis.`,
					"input": "{{input}}",
				},
			},

			// Phase 6: Draft Report
			{
				ID:   "draft_executive_summary",
				Type: planner.NodeLLM,
				Config: map[string]any{
					"prompt": `Write an executive summary for the research report.

						Topic: {{input.topic}}
						Audience: {{input.audience}}
						Key findings: {{analyze_patterns}}
						Verification: {{verify_findings}}

						Write a concise (1-2 paragraph) executive summary.
						Highlight the most important findings and conclusions.`,
					"input": "{{input}}",
				},
			},
			{
				ID:   "draft_main_report",
				Type: planner.NodeLLM,
				Config: map[string]any{
					"prompt": `Write the main research report.

						Topic: {{input.topic}}
						Audience: {{input.audience}}
						Key findings: {{analyze_patterns}}
						Deep dives: {{deep_dive_1}}, {{deep_dive_2}}, {{deep_dive_3}}
						Verification: {{verify_findings}}

						Structure:
						1. Introduction
						2. Background
						3. Key Findings (with subsections)
						4. Analysis
						5. Conclusions
						6. Recommendations

						Include citations and be thorough.`,
					"input": "{{input}}",
				},
			},

			// Phase 7: Final Review
			{
				ID:   "review_report",
				Type: planner.NodeLLM,
				Config: map[string]any{
					"prompt": `Review and refine the research report.

						Original draft: {{draft_main_report}}
						Executive summary: {{draft_executive_summary}}
						Verification: {{verify_findings}}

						Tasks:
						1. Check for completeness
						2. Verify all citations
						3. Improve clarity and flow
						4. Ensure balanced presentation
						5. Add any missing critical information

						Return the finalized report.`,
					"input": "{{input}}",
				},
			},

			// Phase 8: Generate Deliverables
			{
				ID:   "generate_summary",
				Type: planner.NodeLLM,
				Config: map[string]any{
					"prompt": `Create a condensed summary of the research.

						Topic: {{input.topic}}
						Final report: {{review_report}}

						Create a 1-page summary suitable for busy executives.
						Include: Key findings, main conclusions, top 3 recommendations.`,
					"input": "{{input}}",
				},
			},
			{
				ID:   "generate_presentation",
				Type: planner.NodeLLM,
				Config: map[string]any{
					"prompt": `Create presentation outline.

						Topic: {{input.topic}}
						Final report: {{review_report}}

						Create a slide-by-slide outline for a presentation.
						Format: Slide title, key points (3-5 per slide), notes`,
					"input": "{{input}}",
				},
			},

			// Phase 9: Compile Final Output
			{
				ID:   "compile_output",
				Type: planner.NodeLLM,
				Config: map[string]any{
					"prompt": `Compile the final research package.

						Executive summary: {{draft_executive_summary}}
						Main report: {{review_report}}
						Summary: {{generate_summary}}
						Presentation: {{generate_presentation}}
						Verification: {{verify_findings}}

						Organize all deliverables into the final output format.
						Include source list and confidence rating.`,
					"input": "{{input}}",
				},
			},
		},
		Edges: []planner.TaskEdge{
			// Topic analysis
			{From: "analyze_topic", To: "search_web"},
			{From: "analyze_topic", To: "search_knowledge"},
			{From: "analyze_topic", To: "search_documents"},

			// Background searches complete -> deep dives
			{From: "search_web", To: "deep_dive_1"},
			{From: "search_web", To: "deep_dive_2"},
			{From: "search_web", To: "deep_dive_3"},
			{From: "search_knowledge", To: "deep_dive_1"},
			{From: "search_knowledge", To: "deep_dive_2"},
			{From: "search_knowledge", To: "deep_dive_3"},

			// Deep dives -> verification
			{From: "deep_dive_1", To: "verify_findings"},
			{From: "deep_dive_2", To: "verify_findings"},
			{From: "deep_dive_3", To: "verify_findings"},
			{From: "search_web", To: "verify_findings"},

			// Verification -> analysis
			{From: "verify_findings", To: "analyze_patterns"},

			// Analysis -> drafts
			{From: "analyze_patterns", To: "draft_executive_summary"},
			{From: "analyze_patterns", To: "draft_main_report"},

			// Drafts -> review
			{From: "draft_main_report", To: "review_report"},
			{From: "draft_executive_summary", To: "review_report"},

			// Review -> deliverables
			{From: "review_report", To: "generate_summary"},
			{From: "review_report", To: "generate_presentation"},

			// Deliverables -> final output
			{From: "generate_summary", To: "compile_output"},
			{From: "generate_presentation", To: "compile_output"},
			{From: "review_report", To: "compile_output"},
		},
	}
}

// CreateSimpleResearch creates a simplified research workflow
func CreateSimpleResearch(ctx context.Context) (*planner.TaskGraph, error) {
	return &planner.TaskGraph{
		Nodes: []planner.TaskNode{
			{
				ID:   "search",
				Type: planner.NodeTool,
				Config: map[string]any{
					"tool_name": "web_search",
					"query":     "{{input.topic}}",
					"limit":     10,
				},
			},
			{
				ID:   "research",
				Type: planner.NodeLLM,
				Config: map[string]any{
					"prompt": `Research: {{input.topic}}
						Sources: {{search}}

						Provide comprehensive findings with citations.`,
					"input": "{{input}}",
				},
			},
			{
				ID:   "report",
				Type: planner.NodeLLM,
				Config: map[string]any{
					"prompt": `Write a research report on: {{input.topic}}

						Research findings: {{research}}

						Format: Introduction, Key Findings, Analysis, Conclusion.`,
					"input": "{{input}}",
				},
			},
		},
		Edges: []planner.TaskEdge{
			{From: "search", To: "research"},
			{From: "research", To: "report"},
		},
	}, nil
}

// executeDemo runs a demonstration
func executeDemo(ctx context.Context, workflow *planner.TaskGraph) {
	fmt.Println("\n=== Autonomous Research Demo ===")
	fmt.Println("This workflow demonstrates:")
	fmt.Println("1. Autonomous topic analysis and planning")
	fmt.Println("2. Multi-source information gathering")
	fmt.Println("3. Deep dive research on knowledge areas")
	fmt.Println("4. Verification and fact-checking")
	fmt.Println("5. Structured report generation")
	fmt.Println("6. Multiple deliverable formats")

	_ = os.WriteFile("research_workflow.json", mustMarshalJSON(workflow), 0644)
	fmt.Println("\nWorkflow saved to research_workflow.json")
}

// CreateResearchGraph creates a simple research graph using compose
func CreateResearchGraph(ctx context.Context) (compose.Runnable[*ResearchInput, *ResearchOutput], error) {
	graph := compose.NewGraph[*ResearchInput, *ResearchOutput]()

	// Search node
	if err := graph.AddLambdaNode("search", compose.InvokableLambda(func(ctx context.Context, input *ResearchInput) (*struct {
		Results []string
		Topic   string
	}, error) {
		return &struct {
			Results []string
			Topic   string
		}{
			Results: []string{"source1", "source2", "source3"},
			Topic:   input.Topic,
		}, nil
	})); err != nil {
		return nil, fmt.Errorf("add search node: %w", err)
	}

	// Research node
	if err := graph.AddLambdaNode("research", compose.InvokableLambda(func(ctx context.Context, input *struct {
		Results []string
		Topic   string
	}) (*struct {
		Findings []string
	}, error) {
		return &struct {
			Findings []string
		}{
			Findings: []string{"Finding 1", "Finding 2", "Finding 3"},
		}, nil
	})); err != nil {
		return nil, fmt.Errorf("add research node: %w", err)
	}

	// Report node
	if err := graph.AddLambdaNode("report", compose.InvokableLambda(func(ctx context.Context, input *struct {
		Findings []string
	}) (*ResearchOutput, error) {
		return &ResearchOutput{
			Title:            "Research Report",
			ExecutiveSummary: "Summary of findings",
			KeyFindings:      input.Findings,
			Report:           "Full report content...",
			Confidence:       0.85,
		}, nil
	})); err != nil {
		return nil, fmt.Errorf("add report node: %w", err)
	}

	if err := graph.AddEdge(compose.START, "search"); err != nil {
		return nil, fmt.Errorf("add start->search edge: %w", err)
	}
	if err := graph.AddEdge("search", "research"); err != nil {
		return nil, fmt.Errorf("add search->research edge: %w", err)
	}
	if err := graph.AddEdge("research", "report"); err != nil {
		return nil, fmt.Errorf("add research->report edge: %w", err)
	}
	if err := graph.AddEdge("report", compose.END); err != nil {
		return nil, fmt.Errorf("add report->end edge: %w", err)
	}

	return graph.Compile(ctx)
}

func mustMarshalJSON(v any) []byte {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		log.Fatal(err)
	}
	return b
}
