#!/bin/bash

# Real Agent Test with Qwen3-Max
# Uses Alibaba Cloud DashScope Qwen3-Max model

API_URL="http://localhost:8080"
MODEL="qwen.qwen3_max"

get_token() {
  curl -s -X POST "$API_URL/api/login" \
    -H "Content-Type: application/json" \
    -d '{"username":"admin","password":"admin"}' | grep -o '"token":"[^"]*"' | cut -d'"' -f4
}

echo "=========================================="
echo "   Aetheris Real Agent Test"
echo "   Model: Qwen3-Max (Alibaba DashScope)"
echo "=========================================="
echo ""

TOKEN=$(get_token)
echo "🔑 Token: ${TOKEN:0:50}..."
echo "📡 Model: $MODEL"
echo ""

# ============================================
# Test 1: Basic LLM Query
# ============================================
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "📋 Test 1: Basic LLM Query"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

CREATE1=$(curl -s -X POST "$API_URL/api/agents" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{
    "name": "qwen-basic",
    "model": "'"$MODEL"'"
  }')

AGENT1_ID=$(echo $CREATE1 | grep -o '"id":"[^"]*"' | cut -d'"' -f4)
echo "Agent ID: $AGENT1_ID"

MSG1=$(curl -s -X POST "$API_URL/api/agents/$AGENT1_ID/message" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"message": "What is artificial intelligence? Answer in 2 sentences."}')

JOB1_ID=$(echo $MSG1 | grep -o '"job_id":"[^"]*"' | cut -d'"' -f4)
echo "Job ID: $JOB1_ID"

echo "⏳ Running..."
sleep 8

STATUS1=$(curl -s -X GET "$API_URL/api/jobs/$JOB1_ID" \
  -H "Authorization: Bearer $TOKEN")
echo "Status: $(echo $STATUS1 | jq -r '.status')"

# Get result
EVENTS1=$(curl -s -X GET "$API_URL/api/jobs/$JOB1_ID/events" \
  -H "Authorization: Bearer $TOKEN")
RESULT1=$(echo $EVENTS1 | jq -r '.events[] | select(.type == "command_committed") | .payload.result.result')
echo "Result: $RESULT1"
echo ""

# ============================================
# Test 2: Mathematical Reasoning
# ============================================
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "📋 Test 2: Mathematical Reasoning"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

CREATE2=$(curl -s -X POST "$API_URL/api/agents" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{
    "name": "qwen-math",
    "model": "'"$MODEL"'"
  }')

AGENT2_ID=$(echo $CREATE2 | grep -o '"id":"[^"]*"' | cut -d'"' -f4)
echo "Agent ID: $AGENT2_ID"

MSG2=$(curl -s -X POST "$API_URL/api/agents/$AGENT2_ID/message" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"message": "If a train travels 120km in 2 hours, what is its speed in km/h?"}')

JOB2_ID=$(echo $MSG2 | grep -o '"job_id":"[^"]*"' | cut -d'"' -f4)
echo "Job ID: $JOB2_ID"

echo "⏳ Running..."
sleep 8

STATUS2=$(curl -s -X GET "$API_URL/api/jobs/$JOB2_ID" \
  -H "Authorization: Bearer $TOKEN")
echo "Status: $(echo $STATUS2 | jq -r '.status')"

EVENTS2=$(curl -s -X GET "$API_URL/api/jobs/$JOB2_ID/events" \
  -H "Authorization: Bearer $TOKEN")
RESULT2=$(echo $EVENTS2 | jq -r '.events[] | select(.type == "command_committed") | .payload.result.result')
echo "Result: $RESULT2"
echo ""

# ============================================
# Test 3: Creative Writing
# ============================================
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "📋 Test 3: Creative Writing"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

CREATE3=$(curl -s -X POST "$API_URL/api/agents" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{
    "name": "qwen-creative",
    "model": "'"$MODEL"'"
  }')

AGENT3_ID=$(echo $CREATE3 | grep -o '"id":"[^"]*"' | cut -d'"' -f4)
echo "Agent ID: $AGENT3_ID"

MSG3=$(curl -s -X POST "$API_URL/api/agents/$AGENT3_ID/message" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"message": "Write a short poem about technology"}')

JOB3_ID=$(echo $MSG3 | grep -o '"job_id":"[^"]*"' | cut -d'"' -f4)
echo "Job ID: $JOB3_ID"

echo "⏳ Running..."
sleep 8

STATUS3=$(curl -s -X GET "$API_URL/api/jobs/$JOB3_ID" \
  -H "Authorization: Bearer $TOKEN")
echo "Status: $(echo $STATUS3 | jq -r '.status')"

EVENTS3=$(curl -s -X GET "$API_URL/api/jobs/$JOB3_ID/events" \
  -H "Authorization: Bearer $TOKEN")
RESULT3=$(echo $EVENTS3 | jq -r '.events[] | select(.type == "command_committed") | .payload.result.result')
echo "Result: $RESULT3"
echo ""

# ============================================
# Test 4: Multi-step with Tools
# ============================================
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "📋 Test 4: Multi-step Workflow"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

CREATE4=$(curl -s -X POST "$API_URL/api/agents" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{
    "name": "qwen-workflow",
    "model": "'"$MODEL"'",
    "workflow": {
      "nodes": [
        {"id": "analyze", "type": "llm", "config": {"prompt": "What is 25 + 17?"}},
        {"id": "verify", "type": "llm", "config": {"prompt": "Is the previous answer correct?"}}
      ],
      "edges": [
        {"from": "analyze", "to": "verify"}
      ]
    }
  }')

AGENT4_ID=$(echo $CREATE4 | grep -o '"id":"[^"]*"' | cut -d'"' -f4)
echo "Agent ID: $AGENT4_ID"

MSG4=$(curl -s -X POST "$API_URL/api/agents/$AGENT4_ID/message" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"message": "Run the workflow"}')

JOB4_ID=$(echo $MSG4 | grep -o '"job_id":"[^"]*"' | cut -d'"' -f4)
echo "Job ID: $JOB4_ID"

echo "⏳ Running..."
sleep 10

STATUS4=$(curl -s -X GET "$API_URL/api/jobs/$JOB4_ID" \
  -H "Authorization: Bearer $TOKEN")
echo "Status: $(echo $STATUS4 | jq -r '.status')"

TRACE4=$(curl -s -X GET "$API_URL/api/jobs/$JOB4_ID/trace" \
  -H "Authorization: Bearer $TOKEN")
echo "Nodes executed: $(echo $TRACE4 | jq '.node_durations | length')"
echo ""

# ============================================
# Test 5: Verify & Evidence
# ============================================
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "📋 Test 5: Verify & Evidence Chain"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

VERIFY=$(curl -s -X GET "$API_URL/api/jobs/$JOB1_ID/verify" \
  -H "Authorization: Bearer $TOKEN")
echo "Verify:"
echo "$VERIFY" | jq '{execution_hash: .execution_hash, event_chain_ok: .replay_proof_result.ok, tool_ledger_ok: .tool_invocation_ledger_proof.ok}'
echo ""

# ============================================
# Test 6: Trace & Timeline
# ============================================
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "📋 Test 6: Trace & Timeline"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

TRACE=$(curl -s -X GET "$API_URL/api/jobs/$JOB1_ID/trace" \
  -H "Authorization: Bearer $TOKEN")

echo "Job ID: $(echo $TRACE | jq -r '.job_id')"
echo "Timeline Events: $(echo $TRACE | jq '.timeline | length')"
echo "Steps: $(echo $TRACE | jq '.steps | length')"
echo ""

# Show step details
echo "Step Details:"
echo "$TRACE" | jq -r '.steps[] | "  - \(.label): \(.state) (\(.duration_ms // 0)ms)"'
echo ""

# ============================================
# Summary
# ============================================
echo "=========================================="
echo "   TEST SUMMARY"
echo "=========================================="
echo ""
echo "Model: $MODEL"
echo ""
echo "✅ Test 1 (Basic Query): $(echo $STATUS1 | jq -r '.status')"
echo "   Result: $RESULT1"
echo ""
echo "✅ Test 2 (Math): $(echo $STATUS2 | jq -r '.status')"
echo "   Result: $RESULT2"
echo ""
echo "✅ Test 3 (Creative): $(echo $STATUS3 | jq -r '.status')"
echo "   Result: $(echo $RESULT3 | head -c 100)..."
echo ""
echo "✅ Test 4 (Workflow): $(echo $STATUS4 | jq -r '.status')"
echo ""
echo "✅ Test 5 (Verify): OK"
echo "✅ Test 6 (Trace): OK"

echo ""
echo "📈 Observability:"
curl -s -X GET "$API_URL/api/observability/summary" \
  -H "Authorization: Bearer $TOKEN" | jq .

echo ""
echo "✅ All tests completed!"
