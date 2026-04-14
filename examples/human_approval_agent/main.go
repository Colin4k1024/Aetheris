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

// HumanApprovalAgent demonstrates human-in-the-loop workflow
// This example shows how to build approval flows with Aetheris

func main() {
	// Example: Refund Approval Workflow
	// 1. Agent analyzes refund request
	// 2. Waits for human approval
	// 3. Executes based on approval/rejection

	refundWorkflow := &planner.TaskGraph{
		Nodes: []planner.TaskNode{
			{
				ID:   "analyze_request",
				Type: planner.NodeLLM,
				Config: map[string]any{
					"prompt": `分析退款请求:
					- 订单金额
					- 退款原因
					- 客户历史
					返回: 批准/拒绝/需要更多信息`,
				},
			},
			{
				ID:   "approve_refund",
				Type: planner.NodeWait,
				Config: map[string]any{
					"wait_kind":       "signal",
					"correlation_key": "refund-approval",
					"timeout":         "24h", // 24小时超时
				},
			},
			{
				ID:   "execute_refund",
				Type: planner.NodeTool,
				Config: map[string]any{
					"tool_name": "process_refund",
				},
			},
			{
				ID:   "notify_customer",
				Type: planner.NodeTool,
				Config: map[string]any{
					"tool_name": "send_notification",
				},
			},
		},
		Edges: []planner.TaskEdge{
			{From: "analyze_request", To: "approve_refund"},
			{From: "approve_refund", To: "execute_refund"},
			{From: "execute_refund", To: "notify_customer"},
		},
	}

	// Serialize to JSON
	bytes, err := json.MarshalIndent(refundWorkflow, "", "  ")
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Human Approval Workflow:")
	fmt.Println(string(bytes))
	fmt.Println("\n=== Usage ===")
	fmt.Print(`
# 1. Create agent with workflow
curl -X POST http://localhost:8080/api/agents \
  -H "Content-Type: application/json" \
  -d '{
    "name": "refund-approval",
    "workflow": ${WORKFLOW_JSON}
  }'

# 2. Submit refund request (job created, waits at approval node)
curl -X POST http://localhost:8080/api/agents/{agent_id}/message \
  -d '{"message": "客户张三申请订单 #12345 退款 $99.99，原因：商品损坏"}'

# 3. Check job status - will be "waiting"
curl http://localhost:8080/api/jobs/{job_id}

# 4. Human approves via signal
curl -X POST http://localhost:8080/api/jobs/{job_id}/signal \
  -H "Content-Type: application/json" \
  -d '{
    "correlation_key": "refund-approval",
    "signal": "approved",
    "comment": "已核实，同意退款"
  }'

# 5. Or reject
curl -X POST http://localhost:8080/api/jobs/{job_id}/signal \
  -H "Content-Type: application/json" \
  -d '{
    "correlation_key": "refund-approval",
    "signal": "rejected",
    "comment": "退款原因不充分"
  }'
`)
}

// Another example: Document Review Workflow
func documentReviewWorkflow() *planner.TaskGraph {
	return &planner.TaskGraph{
		Nodes: []planner.TaskNode{
			{
				ID:   "generate_draft",
				Type: planner.NodeLLM,
				Config: map[string]any{
					"prompt": "根据需求生成文档草稿",
				},
			},
			{
				ID:   "legal_review",
				Type: planner.NodeWait,
				Config: map[string]any{
					"wait_kind":       "signal",
					"correlation_key": "legal-approval",
				},
			},
			{
				ID:   "manager_review",
				Type: planner.NodeWait,
				Config: map[string]any{
					"wait_kind":       "signal",
					"correlation_key": "manager-approval",
				},
			},
			{
				ID:   "finalize",
				Type: planner.NodeTool,
				Config: map[string]any{
					"tool_name": "publish_document",
				},
			},
		},
		Edges: []planner.TaskEdge{
			{From: "generate_draft", To: "legal_review"},
			{From: "legal_review", To: "manager_review"},
			{From: "manager_review", To: "finalize"},
		},
	}
}

// Payment Approval Workflow
func paymentApprovalWorkflow() *planner.TaskGraph {
	return &planner.TaskGraph{
		Nodes: []planner.TaskNode{
			{
				ID:   "validate_payment",
				Type: planner.NodeTool,
				Config: map[string]any{
					"tool_name": "validate_payment_request",
				},
			},
			{
				ID:   "finance_approval",
				Type: planner.NodeWait,
				Config: map[string]any{
					"wait_kind":       "message", // 也可以用消息队列
					"correlation_key": "payment-finance",
				},
			},
			{
				ID:   "ceo_approval",
				Type: planner.NodeWait,
				Config: map[string]any{
					"wait_kind":       "signal",
					"correlation_key": "payment-ceo",
					"threshold":       10000, // 金额大于10000需要CEO审批
				},
			},
			{
				ID:   "execute_payment",
				Type: planner.NodeTool,
				Config: map[string]any{
					"tool_name": "process_payment",
				},
			},
		},
		Edges: []planner.TaskEdge{
			{From: "validate_payment", To: "finance_approval"},
			{From: "finance_approval", To: "ceo_approval"},
			{From: "ceo_approval", To: "execute_payment"},
		},
	}
}
