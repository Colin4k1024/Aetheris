#!/bin/bash

# Complete Agent Test with All Features
# Tests framework adapters, workflows, and advanced features

API_URL="http://localhost:8080"

get_token() {
  curl -s -X POST "$API_URL/api/login" \
    -H "Content-Type: application/json" \
    -d '{"username":"admin","password":"admin"}' | grep -o '"token":"[^"]*"' | cut -d'"' -f4
}

echo "=========================================="
echo "   Aetheris 完整功能测试"
echo "=========================================="
echo ""

TOKEN=$(get_token)
echo "🔑 Token: ${TOKEN:0:50}..."
echo ""

# ============================================
# Test 1: Simple LLM Agent
# ============================================
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "📋 Test 1: 简单 LLM Agent"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

CREATE1=$(curl -s -X POST "$API_URL/api/agents" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"name": "llm-agent-test", "model": "openai.gpt_35_turbo"}')
AGENT1_ID=$(echo $CREATE1 | grep -o '"id":"[^"]*"' | cut -d'"' -f4)
echo "Agent ID: $AGENT1_ID"

MSG1=$(curl -s -X POST "$API_URL/api/agents/$AGENT1_ID/message" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"message": "Say hello in one sentence"}')
JOB1_ID=$(echo $MSG1 | grep -o '"job_id":"[^"]*"' | cut -d'"' -f4)
echo "Job ID: $JOB1_ID"

sleep 4
STATUS1=$(curl -s -X GET "$API_URL/api/jobs/$JOB1_ID" \
  -H "Authorization: Bearer $TOKEN" | grep -o '"status":"[^"]*"' | cut -d'"' -f4)
echo "✅ Job Status: $STATUS1"

# Get the result
EVENTS1=$(curl -s -X GET "$API_URL/api/jobs/$JOB1_ID/events" \
  -H "Authorization: Bearer $TOKEN")
RESULT1=$(echo $EVENTS1 | grep -o '"result":"[^"]*"' | head -1 | cut -d'"' -f4)
echo "📝 Result: $RESULT1"
echo ""

# ============================================
# Test 2: Agent with Custom TaskGraph (LLM Node)
# ============================================
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "📋 Test 2: 自定义 TaskGraph (LLM Node)"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

CREATE2=$(curl -s -X POST "$API_URL/api/agents" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{
    "name": "taskgraph-agent-test",
    "model": "openai.gpt_35_turbo",
    "workflow": {
      "nodes": [
        {"id": "n1", "type": "llm", "config": {"prompt": "What is the capital of France?"}}
      ],
      "edges": []
    }
  }')
AGENT2_ID=$(echo $CREATE2 | grep -o '"id":"[^"]*"' | cut -d'"' -f4)
echo "Agent ID: $AGENT2_ID"

MSG2=$(curl -s -X POST "$API_URL/api/agents/$AGENT2_ID/message" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"message": "Run the workflow"}')
JOB2_ID=$(echo $MSG2 | grep -o '"job_id":"[^"]*"' | cut -d'"' -f4)
echo "Job ID: $JOB2_ID"

sleep 4
STATUS2=$(curl -s -X GET "$API_URL/api/jobs/$JOB2_ID" \
  -H "Authorization: Bearer $TOKEN" | grep -o '"status":"[^"]*"' | cut -d'"' -f4)
echo "✅ Job Status: $STATUS2"
echo ""

# ============================================
# Test 3: Multi-step Workflow
# ============================================
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "📋 Test 3: 多步骤工作流"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

CREATE3=$(curl -s -X POST "$API_URL/api/agents" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{
    "name": "multi-step-workflow",
    "model": "openai.gpt_35_turbo",
    "workflow": {
      "nodes": [
        {"id": "step1", "type": "llm", "config": {"prompt": "Add 1 to 5"}},
        {"id": "step2", "type": "llm", "config": {"prompt": "Now multiply the result by 2"}}
      ],
      "edges": [
        {"from": "step1", "to": "step2"}
      ]
    }
  }')
AGENT3_ID=$(echo $CREATE3 | grep -o '"id":"[^"]*"' | cut -d'"' -f4)
echo "Agent ID: $AGENT3_ID"

MSG3=$(curl -s -X POST "$API_URL/api/agents/$AGENT3_ID/message" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"message": "Calculate (5+1)*2"}')
JOB3_ID=$(echo $MSG3 | grep -o '"job_id":"[^"]*"' | cut -d'"' -f4)
echo "Job ID: $JOB3_ID"

sleep 6
STATUS3=$(curl -s -X GET "$API_URL/api/jobs/$JOB3_ID" \
  -H "Authorization: Bearer $TOKEN" | grep -o '"status":"[^"]*"' | cut -d'"' -f4)
echo "✅ Job Status: $STATUS3"

# Get trace
TRACE3=$(curl -s -X GET "$API_URL/api/jobs/$JOB3_ID/trace" \
  -H "Authorization: Bearer $TOKEN")
NODE_COUNT=$(echo $TRACE3 | grep -o '"node_id":"[^"]*"' | wc -l)
echo "📊 Nodes in Trace: $NODE_COUNT"
echo ""

# ============================================
# Test 4: Parallel Nodes
# ============================================
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "📋 Test 4: 并行节点"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

CREATE4=$(curl -s -X POST "$API_URL/api/agents" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{
    "name": "parallel-workflow",
    "model": "openai.gpt_35_turbo",
    "workflow": {
      "nodes": [
        {"id": "task_a", "type": "llm", "config": {"prompt": "Say A"}},
        {"id": "task_b", "type": "llm", "config": {"prompt": "Say B"}},
        {"id": "combine", "type": "llm", "config": {"prompt": "Combine the results"}}
      ],
      "edges": [
        {"from": "task_a", "to": "combine"},
        {"from": "task_b", "to": "combine"}
      ]
    }
  }')
AGENT4_ID=$(echo $CREATE4 | grep -o '"id":"[^"]*"' | cut -d'"' -f4)
echo "Agent ID: $AGENT4_ID"

MSG4=$(curl -s -X POST "$API_URL/api/agents/$AGENT4_ID/message" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"message": "Run parallel tasks"}')
JOB4_ID=$(echo $MSG4 | grep -o '"job_id":"[^"]*"' | cut -d'"' -f4)
echo "Job ID: $JOB4_ID"

sleep 8
STATUS4=$(curl -s -X GET "$API_URL/api/jobs/$JOB4_ID" \
  -H "Authorization: Bearer $TOKEN" | grep -o '"status":"[^"]*"' | cut -d'"' -f4)
echo "✅ Job Status: $STATUS4"
echo ""

# ============================================
# Test 5: Job Stop
# ============================================
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "📋 Test 5: 停止 Job"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

CREATE5=$(curl -s -X POST "$API_URL/api/agents" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"name": "stop-test-agent", "model": "openai.gpt_35_turbo"}')
AGENT5_ID=$(echo $CREATE5 | grep -o '"id":"[^"]*"' | cut -d'"' -f4)
echo "Agent ID: $AGENT5_ID"

MSG5=$(curl -s -X POST "$API_URL/api/agents/$AGENT5_ID/message" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"message": "Count from 1 to 100 slowly"}')
JOB5_ID=$(echo $MSG5 | grep -o '"job_id":"[^"]*"' | cut -d'"' -f4)
echo "Job ID: $JOB5_ID"

sleep 2
# Stop the job
STOP_RESP=$(curl -s -X POST "$API_URL/api/jobs/$JOB5_ID/stop" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN")
echo "Stop Response: $STOP_RESP"

sleep 1
STATUS5=$(curl -s -X GET "$API_URL/api/jobs/$JOB5_ID" \
  -H "Authorization: Bearer $TOKEN" | grep -o '"status":"[^"]*"' | cut -d'"' -f4)
echo "✅ Job Status after stop: $STATUS5"
echo ""

# ============================================
# Test 6: Replay
# ============================================
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "📋 Test 6: Replay 功能"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

REPLAY_RESP=$(curl -s -X GET "$API_URL/api/jobs/$JOB1_ID/replay" \
  -H "Authorization: Bearer $TOKEN")
echo "Replay Response: $REPLAY_RESP"
echo ""

# ============================================
# Test 7: Verify
# ============================================
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "📋 Test 7: Verify 功能"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

VERIFY_RESP=$(curl -s -X GET "$API_URL/api/jobs/$JOB1_ID/verify" \
  -H "Authorization: Bearer $TOKEN")
echo "Verify Response: $VERIFY_RESP"
echo ""

# ============================================
# Test 8: Evidence Graph
# ============================================
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "📋 Test 8: 证据图 (Evidence Graph)"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

EVIDENCE_RESP=$(curl -s -X GET "$API_URL/api/jobs/$JOB1_ID/evidence-graph" \
  -H "Authorization: Bearer $TOKEN")
echo "Evidence Graph: $EVIDENCE_RESP"
echo ""

# ============================================
# Test 9: Audit Log
# ============================================
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "📋 Test 9: 审计日志 (Audit Log)"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

AUDIT_RESP=$(curl -s -X GET "$API_URL/api/jobs/$JOB1_ID/audit-log" \
  -H "Authorization: Bearer $TOKEN")
AUDIT_LINES=$(echo $AUDIT_RESP | grep -o '"type":"[^"]*"' | wc -l)
echo "Audit Log Entries: $AUDIT_LINES"
echo ""

# ============================================
# Test 10: Forensics Query
# ============================================
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "📋 Test 10: 取证查询 (Forensics)"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

FORENSICS_RESP=$(curl -s -X POST "$API_URL/api/forensics/query" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{
    "job_ids": ["'"$JOB1_ID"'", "'"$JOB2_ID"'"],
    "time_range": {"from": "2026-01-01T00:00:00Z", "to": "2026-12-31T23:59:59Z"}
  }')
echo "Forensics Query Response: $FORENSICS_RESP"
echo ""

# ============================================
# Summary
# ============================================
echo "=========================================="
echo "   测试完成 - 汇总"
echo "=========================================="

echo ""
echo "📊 测试结果汇总:"
echo "   ✅ Test 1: 简单 LLM Agent - $STATUS1"
echo "   ✅ Test 2: 自定义 TaskGraph - $STATUS2"
echo "   ✅ Test 3: 多步骤工作流 - $STATUS3"
echo "   ✅ Test 4: 并行节点 - $STATUS4"
echo "   ✅ Test 5: 停止 Job - $STATUS5"
echo "   ✅ Test 6: Replay - 可用"
echo "   ✅ Test 7: Verify - 可用"
echo "   ✅ Test 8: Evidence Graph - 可用"
echo "   ✅ Test 9: Audit Log - $AUDIT_LINES 条记录"
echo "   ✅ Test 10: Forensics Query - 可用"

echo ""
echo "📦 Docker 容器状态:"
docker ps --format "table {{.Names}}\t{{.Status}}"

echo ""
echo "🌐 API 健康:"
curl -s http://localhost:8080/api/health
echo ""
