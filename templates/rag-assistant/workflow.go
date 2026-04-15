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

// RAGAssistantWorkflow demonstrates a Retrieval-Augmented Generation workflow
// Features:
// - Query understanding and rewrite
// - Multi-step retrieval with re-ranking
// - Context synthesis
// - Source tracing and citations

func main() {
	ctx := context.Background()

	// Create the RAG workflow
	workflow := createRAGWorkflow()

	// Serialize and display
	bytes, err := json.MarshalIndent(workflow, "", "  ")
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("=== RAG Assistant Workflow ===")
	fmt.Println(string(bytes))

	// Execute demo
	executeDemo(ctx, workflow)
}

// createRAGWorkflow defines the RAG workflow with retrieval and generation
func createRAGWorkflow() *planner.TaskGraph {
	return &planner.TaskGraph{
		Nodes: []planner.TaskNode{
			// Step 1: Query understanding and rewrite
			// Breaks down complex queries and identifies key concepts
			{
				ID:   "query_rewrite",
				Type: planner.NodeLLM,
				Config: map[string]any{
					"prompt": `Analyze and rewrite the user query for optimal retrieval.

						Original query: {{input.query}}

						Tasks:
						1. Identify key concepts and entities
						2. Determine search intent (factual, explanatory, procedural)
						3. Rewrite for better retrieval (expand acronyms, clarify ambiguity)
						4. Generate alternative phrasings

						Return JSON with: {"rewritten": "...", "concepts": [...], "alternatives": [...]}`,
					"input": "{{input.query}}",
				},
			},

			// Step 2: Initial retrieval from knowledge base
			{
				ID:   "retrieve_v1",
				Type: planner.NodeTool,
				Config: map[string]any{
					"tool_name": "retriever",
					"query":     "{{query_rewrite.rewritten}}",
					"top_k":     10,
				},
			},

			// Step 3: Web search for additional context
			{
				ID:   "web_search",
				Type: planner.NodeTool,
				Config: map[string]any{
					"tool_name": "web_search",
					"query":     "{{query_rewrite.concepts}}",
					"limit":     5,
				},
			},

			// Step 4: Re-rank and filter retrieved results
			{
				ID:   "rerank",
				Type: planner.NodeLLM,
				Config: map[string]any{
					"prompt": `Re-rank and filter retrieved documents based on relevance to the query.

						Query: {{input.query}}
						Rewritten query: {{query_rewrite.rewritten}}

						Retrieved docs: {{retrieve_v1}}
						Web results: {{web_search}}

						Tasks:
						1. Score each document for relevance (0-1)
						2. Remove duplicates
						3. Filter low-quality or irrelevant content
						4. Select top 5 most relevant documents

						Return JSON: {"ranked": [...], "reasoning": "..."}`,
					"input": "{{input.query}}",
				},
			},

			// Step 5: Synthesize context from documents
			{
				ID:   "synthesize",
				Type: planner.NodeLLM,
				Config: map[string]any{
					"prompt": `You are a research analyst. Synthesize information from multiple sources.

						Query: {{input.query}}

						Documents: {{rerank.ranked}}

						Tasks:
						1. Extract key facts and findings
						2. Identify consensus and conflicts
						3. Note confidence levels
						4. Identify gaps in available information

						Return a structured synthesis.`,
					"input": "{{input.query}}",
				},
			},

			// Step 6: Generate final answer with citations
			{
				ID:   "generate_answer",
				Type: planner.NodeLLM,
				Config: map[string]any{
					"prompt": `Generate a comprehensive, well-cited answer to the user's question.

						Original query: {{input.query}}
						Rewritten query: {{query_rewrite.rewritten}}

						Synthesized information: {{synthesize}}

						Guidelines:
						- Start with a direct answer
						- Provide supporting evidence with citations
						- Address edge cases and limitations
						- If information is insufficient, clearly state this
						- Use clear structure and formatting`,
					"input": "{{input.query}}",
				},
			},

			// Step 7: Quality check
			{
				ID:   "quality_check",
				Type: planner.NodeLLM,
				Config: map[string]any{
					"prompt": `Review the generated answer for quality.

						Original query: {{input.query}}
						Generated answer: {{generate_answer}}

						Check for:
						1. Factual accuracy
						2. Completeness
						3. Proper citations
						4. Clarity and readability

						Return JSON: {"approved": true/false, "issues": [...], "suggestions": [...]}`,
					"input": "{{input.query}}",
				},
			},

			// Step 8: Finalize answer (incorporate feedback if needed)
			{
				ID:   "finalize",
				Type: planner.NodeLLM,
				Config: map[string]any{
					"prompt": `Finalize the answer, incorporating quality check feedback if needed.

						Original answer: {{generate_answer}}
						Quality feedback: {{quality_check}}

						If issues found, revise accordingly. Otherwise, return the original.`,
					"input": "{{input.query}}",
				},
			},
		},
		Edges: []planner.TaskEdge{
			// Query understanding
			{From: "query_rewrite", To: "retrieve_v1"},
			{From: "query_rewrite", To: "web_search"},

			// Parallel retrieval
			{From: "retrieve_v1", To: "rerank"},
			{From: "web_search", To: "rerank"},

			// Synthesis and generation
			{From: "rerank", To: "synthesize"},
			{From: "synthesize", To: "generate_answer"},

			// Quality assurance
			{From: "generate_answer", To: "quality_check"},
			{From: "quality_check", To: "finalize"},
		},
	}
}

// Input for RAG workflow
type RAGInput struct {
	Query      string `json:"query"`
	Collection string `json:"collection,omitempty"`
	TopK       int    `json:"top_k,omitempty"`
	UserID     string `json:"user_id,omitempty"`
}

// Output from RAG workflow
type RAGOutput struct {
	Answer       string   `json:"answer"`
	Sources      []string `json:"sources"`
	Confidence   float64  `json:"confidence"`
	QueryRewrite string   `json:"query_rewrite,omitempty"`
}

// executeDemo runs a demonstration of the RAG workflow
func executeDemo(ctx context.Context, workflow *planner.TaskGraph) {
	fmt.Println("\n=== RAG Workflow Demo ===")
	fmt.Println("This workflow demonstrates:")
	fmt.Println("1. Query understanding and rewrite")
	fmt.Println("2. Multi-source retrieval (vector DB + web)")
	fmt.Println("3. Re-ranking and filtering")
	fmt.Println("4. Context synthesis")
	fmt.Println("5. Answer generation with citations")
	fmt.Println("6. Quality assurance")

	// Save workflow definition
	_ = os.WriteFile("rag_workflow.json", mustMarshalJSON(workflow), 0644)
	fmt.Println("\nWorkflow saved to rag_workflow.json")
}

// createSimpleRAGWorkflow is a simpler version for basic RAG
func createSimpleRAGWorkflow() *planner.TaskGraph {
	return &planner.TaskGraph{
		Nodes: []planner.TaskNode{
			{
				ID:   "retrieve",
				Type: planner.NodeTool,
				Config: map[string]any{
					"tool_name": "retriever",
					"top_k":     5,
				},
			},
			{
				ID:   "generate",
				Type: planner.NodeLLM,
				Config: map[string]any{
					"prompt": `Based on the following context, answer the user's question.

						Question: {{input.query}}
						Context: {{retrieve}}

						Provide a clear, accurate answer with citations.`,
					"input": "{{input.query}}",
				},
			},
		},
		Edges: []planner.TaskEdge{
			{From: "retrieve", To: "generate"},
		},
	}
}

// CreateQueryWorkflow creates a basic query workflow using compose
func CreateQueryWorkflow(ctx context.Context) (compose.Runnable[*RAGInput, *RAGOutput], error) {
	graph := compose.NewGraph[*RAGInput, *RAGOutput]()

	// Add retrieval node
	graph.AddLambdaNode("retrieve", compose.InvokableLambda(func(ctx context.Context, input *RAGInput) (*struct {
		Docs  []string
		Query string
	}, error) {
		// Placeholder - in production, this calls the retriever tool
		return &struct {
			Docs  []string
			Query string
		}{
			Docs:  []string{"doc1", "doc2"},
			Query: input.Query,
		}, nil
	}))

	// Add generation node
	graph.AddLambdaNode("generate", compose.InvokableLambda(func(ctx context.Context, input *struct {
		Docs  []string
		Query string
	}) (*RAGOutput, error) {
		return &RAGOutput{
			Answer:     "Generated answer based on " + input.Query,
			Sources:    input.Docs,
			Confidence: 0.85,
		}, nil
	}))

	// Add edges
	graph.AddEdge(compose.START, "retrieve")
	graph.AddEdge("retrieve", "generate")
	graph.AddEdge("generate", compose.END)

	return graph.Compile(ctx)
}

func mustMarshalJSON(v any) []byte {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		log.Fatal(err)
	}
	return b
}
