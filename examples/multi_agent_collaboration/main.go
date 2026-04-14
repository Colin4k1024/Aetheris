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
	"encoding/json"
	"fmt"
	"log"

	"github.com/Colin4k1024/Aetheris/v2/internal/agent/planner"
)

// MultiAgentCollaboration demonstrates multi-agent workflows
// Agents work together to complete complex tasks

func main() {
	// Example: Research and Report Generation
	// - Researcher: Gathers information
	// - Analyzer: Processes and analyzes
	// - Writer: Creates final report
	// - Editor: Reviews and approves

	researchWorkflow := &planner.TaskGraph{
		Nodes: []planner.TaskNode{
			{
				ID:   "researcher_gather",
				Type: planner.NodeTool,
				Config: map[string]any{
					"tool_name": "web_search",
					"query":     "{{input.topic}}",
				},
			},
			{
				ID:   "researcher_analyze_sources",
				Type: planner.NodeLLM,
				Config: map[string]any{
					"prompt": "分析收集到的信息，提取关键要点",
					"input":  "{{researcher_gather.results}}",
				},
			},
			{
				ID:   "writer_create_draft",
				Type: planner.NodeLLM,
				Config: map[string]any{
					"prompt": "根据分析结果创建报告草稿",
					"input":  "{{researcher_analyze_sources.key_points}}",
				},
			},
			{
				ID:   "editor_review",
				Type: planner.NodeWait,
				Config: map[string]any{
					"wait_kind":       "signal",
					"correlation_key": "editor-review",
				},
			},
			{
				ID:   "writer_finalize",
				Type: planner.NodeLLM,
				Config: map[string]any{
					"prompt": "根据编辑反馈修改报告",
					"input":  "{{writer_create_draft}} + {{editor_review.feedback}}",
				},
			},
			{
				ID:   "publish",
				Type: planner.NodeTool,
				Config: map[string]any{
					"tool_name": "publish_report",
				},
			},
		},
		Edges: []planner.TaskEdge{
			{From: "researcher_gather", To: "researcher_analyze_sources"},
			{From: "researcher_analyze_sources", To: "writer_create_draft"},
			{From: "writer_create_draft", To: "editor_review"},
			{From: "editor_review", To: "writer_finalize"},
			{From: "writer_finalize", To: "publish"},
		},
	}

	bytes, err := json.MarshalIndent(researchWorkflow, "", "  ")
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Multi-Agent Research Workflow:")
	fmt.Println(string(bytes))
	fmt.Println("\n=== Parallel Agents Example ===")

	// Example: Parallel Execution with Aggregation
	parallelWorkflow := parallelAgentWorkflow()
	parallelBytes, _ := json.MarshalIndent(parallelWorkflow, "", "  ")
	fmt.Println(string(parallelBytes))
}

// parallelAgentWorkflow demonstrates parallel execution
func parallelAgentWorkflow() *planner.TaskGraph {
	return &planner.TaskGraph{
		Nodes: []planner.TaskNode{
			// Parallel research tasks
			{
				ID:   "search_web",
				Type: planner.NodeTool,
				Config: map[string]any{
					"tool_name": "web_search",
				},
			},
			{
				ID:   "search_database",
				Type: planner.NodeTool,
				Config: map[string]any{
					"tool_name": "db_query",
				},
			},
			{
				ID:   "search_documents",
				Type: planner.NodeTool,
				Config: map[string]any{
					"tool_name": "document_search",
				},
			},
			// Aggregate results
			{
				ID:   "aggregate",
				Type: planner.NodeLLM,
				Config: map[string]any{
					"prompt": "整合以下来源的结果:\nWeb: {{search_web}}\nDB: {{search_database}}\nDocs: {{search_documents}}",
				},
			},
			{
				ID:   "finalize",
				Type: planner.NodeTool,
				Config: map[string]any{
					"tool_name": "process_results",
				},
			},
		},
		Edges: []planner.TaskEdge{
			// All search tasks run in parallel, then aggregate
			{From: "search_web", To: "aggregate"},
			{From: "search_database", To: "aggregate"},
			{From: "search_documents", To: "aggregate"},
			{From: "aggregate", To: "finalize"},
		},
	}
}

// Customer Support Multi-Agent
func customerSupportWorkflow() *planner.TaskGraph {
	return &planner.TaskGraph{
		Nodes: []planner.TaskNode{
			{
				ID:   "triage",
				Type: planner.NodeLLM,
				Config: map[string]any{
					"prompt": "分类客户请求: 技术支持/账单/投诉/其他",
				},
			},
			// Conditional routing based on triage result
			{
				ID:   "technical_support",
				Type: planner.NodeTool,
				Config: map[string]any{
					"tool_name": "diagnose_issue",
				},
			},
			{
				ID:   "billing",
				Type: planner.NodeTool,
				Config: map[string]any{
					"tool_name": "check_billing",
				},
			},
			{
				ID:   "escalate",
				Type: planner.NodeWait,
				Config: map[string]any{
					"wait_kind":       "signal",
					"correlation_key": "manager-escalation",
				},
			},
			{
				ID:   "respond",
				Type: planner.NodeTool,
				Config: map[string]any{
					"tool_name": "send_response",
				},
			},
		},
		Edges: []planner.TaskEdge{
			{From: "triage", To: "technical_support"},
			{From: "triage", To: "billing"},
			{From: "triage", To: "escalate"},
			{From: "technical_support", To: "respond"},
			{From: "billing", To: "respond"},
			{From: "escalate", To: "respond"},
		},
	}
}

// Code Review Multi-Agent
func codeReviewWorkflow() *planner.TaskGraph {
	return &planner.TaskGraph{
		Nodes: []planner.TaskNode{
			{
				ID:   "security_scan",
				Type: planner.NodeTool,
				Config: map[string]any{
					"tool_name": "security_check",
				},
			},
			{
				ID:   "style_check",
				Type: planner.NodeTool,
				Config: map[string]any{
					"tool_name": "lint_check",
				},
			},
			{
				ID:   "test_coverage",
				Type: planner.NodeTool,
				Config: map[string]any{
					"tool_name": "coverage_check",
				},
			},
			{
				ID:   "aggregate_feedback",
				Type: planner.NodeLLM,
				Config: map[string]any{
					"prompt": "整合代码审查结果:\n安全: {{security_scan}}\n风格: {{style_check}}\n测试: {{test_coverage}}\n\n给出最终审查意见",
				},
			},
			{
				ID:   "human_review",
				Type: planner.NodeWait,
				Config: map[string]any{
					"wait_kind":       "signal",
					"correlation_key": "code-review",
				},
			},
		},
		Edges: []planner.TaskEdge{
			{From: "security_scan", To: "aggregate_feedback"},
			{From: "style_check", To: "aggregate_feedback"},
			{From: "test_coverage", To: "aggregate_feedback"},
			{From: "aggregate_feedback", To: "human_review"},
		},
	}
}
