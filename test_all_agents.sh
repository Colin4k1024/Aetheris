#!/bin/bash

# Comprehensive Agent Test Script
# Tests all supported agent functionalities

API_URL="http://localhost:8080"

get_token() {
  curl -s -X POST "$API_URL/api/login" \
    -H "Content-Type: application/json" \
    -d '{"username":"admin","password":"admin"}' | grep -o '"token":"[^"]*"' | cut -d'"' -f4
}

echo "=== Aetheris Agent 功能测试 ==="
echo ""

TOKEN=$(get_token)
echo "Token: ${TOKEN:0:50}..."
echo ""

# Test 1: Basic Agent
echo "=== Test 1: 基础 Agent (Basic Agent) ==="
CREATE_RESPONSE=$(curl -s -X POST "$API_URL/api/agents" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{
    "name": "test-basic-agent",
    "model": "openai.gpt_35_turbo",
    "description": "Basic test agent"
  }')
AGENT_ID=$(echo $CREATE_RESPONSE | grep -o '"id":"[^"]*"' | cut -d'"' -f4)
echo "Agent ID: $AGENT_ID"

MESSAGE_RESPONSE=$(curl -s -X POST "$API_URL/api/agents/$AGENT_ID/message" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"message": "What is 1+1?"}')

JOB_ID=$(echo $MESSAGE_RESPONSE | grep -o '"job_id":"[^"]*"' | cut -d'"' -f4)
echo "Job ID: $JOB_ID"

# Wait for completion
sleep 5
JOB_STATUS=$(curl -s -X GET "$API_URL/api/jobs/$JOB_ID" \
  -H "Authorization: Bearer $TOKEN" | grep -o '"status":"[^"]*"' | cut -d'"' -f4)
echo "Job Status: $JOB_STATUS"
echo "✓ 基础 Agent 测试完成"
echo ""

# Test 2: Agent with Tools
echo "=== Test 2: 带工具的 Agent (Tool Agent) ==="
CREATE_RESPONSE2=$(curl -s -X POST "$API_URL/api/agents" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{
    "name": "test-tool-agent",
    "model": "openai.gpt_35_turbo",
    "description": "Agent with tools"
  }')
AGENT_ID2=$(echo $CREATE_RESPONSE2 | grep -o '"id":"[^"]*"' | cut -d'"' -f4)
echo "Agent ID: $AGENT_ID2"

# Send message that should trigger tool use
MESSAGE_RESPONSE2=$(curl -s -X POST "$API_URL/api/agents/$AGENT_ID2/message" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"message": "Search for information about Go programming"}')

JOB_ID2=$(echo $MESSAGE_RESPONSE2 | grep -o '"job_id":"[^"]*"' | cut -d'"' -f4)
echo "Job ID: $JOB_ID2"

sleep 5
JOB_STATUS2=$(curl -s -X GET "$API_URL/api/jobs/$JOB_ID2" \
  -H "Authorization: Bearer $TOKEN" | grep -o '"status":"[^"]*"' | cut -d'"' -f4)
echo "Job Status: $JOB_STATUS2"
echo "✓ 带工具 Agent 测试完成"
echo ""

# Test 3: Human-in-the-loop (Wait/Signal)
echo "=== Test 3: 人机协作 (Human-in-the-Loop) ==="
CREATE_RESPONSE3=$(curl -s -X POST "$API_URL/api/agents" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{
    "name": "test-approval-agent",
    "model": "openai.gpt_35_turbo",
    "description": "Approval workflow agent"
  }')
AGENT_ID3=$(echo $CREATE_RESPONSE3 | grep -o '"id":"[^"]*"' | cut -d'"' -f4)
echo "Agent ID: $AGENT_ID3"

# Create a job that will wait for human approval
MESSAGE_RESPONSE3=$(curl -s -X POST "$API_URL/api/agents/$AGENT_ID3/message" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"message": "Please approve refund of \$100 for order #12345"}')

JOB_ID3=$(echo $MESSAGE_RESPONSE3 | grep -o '"job_id":"[^"]*"' | cut -d'"' -f4)
echo "Job ID: $JOB_ID3"

sleep 3
JOB_STATUS3=$(curl -s -X GET "$API_URL/api/jobs/$JOB_ID3" \
  -H "Authorization: Bearer $TOKEN" | grep -o '"status":"[^"]*"' | cut -d'"' -f4)
echo "Job Status: $JOB_STATUS3"

# Send signal to approve
if [ "$JOB_STATUS3" == "waiting" ] || [ "$JOB_STATUS3" == "parked" ]; then
  echo "Sending approval signal..."
  SIGNAL_RESPONSE=$(curl -s -X POST "$API_URL/api/jobs/$JOB_ID3/signal" \
    -H "Content-Type: application/json" \
    -H "Authorization: Bearer $TOKEN" \
    -d '{
      "correlation_key": "approval",
      "signal": "approved"
    }')
  echo "Signal Response: $SIGNAL_RESPONSE"

  sleep 3
  JOB_STATUS3_FINAL=$(curl -s -X GET "$API_URL/api/jobs/$JOB_ID3" \
    -H "Authorization: Bearer $TOKEN" | grep -o '"status":"[^"]*"' | cut -d'"' -f4)
  echo "Final Job Status: $JOB_STATUS3_FINAL"
fi
echo "✓ 人机协作测试完成"
echo ""

# Test 4: Job Events and Trace
echo "=== Test 4: 事件追踪 (Events & Trace) ==="
EVENTS=$(curl -s -X GET "$API_URL/api/jobs/$JOB_ID/events" \
  -H "Authorization: Bearer $TOKEN")
EVENT_COUNT=$(echo $EVENTS | grep -o '"type":"' | wc -l)
echo "Event Count: $EVENT_COUNT"

TRACE=$(curl -s -X GET "$API_URL/api/jobs/$JOB_ID/trace" \
  -H "Authorization: Bearer $TOKEN")
echo "Trace: $TRACE"
echo "✓ 事件追踪测试完成"
echo ""

# Test 5: Observability
echo "=== Test 5: 可观测性 (Observability) ==="
SUMMARY=$(curl -s -X GET "$API_URL/api/observability/summary" \
  -H "Authorization: Bearer $TOKEN")
echo "Summary: $SUMMARY"

STUCK=$(curl -s -X GET "$API_URL/api/observability/stuck" \
  -H "Authorization: Bearer $TOKEN")
echo "Stuck Jobs: $STUCK"
echo "✓ 可观测性测试完成"
echo ""

# Test 6: List all jobs
echo "=== Test 6: 任务列表 ==="
AGENTS_LIST=$(curl -s -X GET "$API_URL/api/agents" \
  -H "Authorization: Bearer $TOKEN")
echo "Agents: $AGENTS_LIST"
echo ""

echo "=== 所有测试完成 ==="
echo ""
echo "Docker 容器状态:"
docker ps --format "table {{.Names}}\t{{.Status}}\t{{.Ports}}"
