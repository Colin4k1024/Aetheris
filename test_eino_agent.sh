#!/bin/bash

# Eino Agent Integration Test
# Tests the eino ReAct/DEER/Manus agent integration with TaskGraph

API_URL="http://localhost:8080"

get_token() {
  curl -s -X POST "$API_URL/api/login" \
    -H "Content-Type: application/json" \
    -d '{"username":"admin","password":"admin"}' | grep -o '"token":"[^"]*"' | cut -d'"' -f4
}

echo "=========================================="
echo "   Eino Agent Integration Test"
echo "   Testing: eino_react, eino_deer, eino_manus"
echo "=========================================="
echo ""

TOKEN=$(get_token)
echo "Token: ${TOKEN:0:50}..."
echo ""

# ============================================
# Test 1: Eino React Agent
# ============================================
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "📋 Test 1: Eino React Agent"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

CREATE1=$(curl -s -X POST "$API_URL/api/agents" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{
    "name": "eino-react-agent",
    "model": "qwen.qwen3_max",
    "workflow": {
      "nodes": [
        {
          "id": "react",
          "type": "eino_react",
          "config": {
            "system_prompt": "You are a helpful assistant."
          }
        }
      ],
      "edges": []
    }
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

EVENTS1=$(curl -s -X GET "$API_URL/api/jobs/$JOB1_ID/events" \
  -H "Authorization: Bearer $TOKEN")
RESULT1=$(echo $EVENTS1 | jq -r '.events[] | select(.type == "command_committed") | .payload.result.result')
echo "Result: $RESULT1"
echo ""

# ============================================
# Test 2: Eino DEER Agent
# ============================================
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "📋 Test 2: Eino DEER Agent"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

CREATE2=$(curl -s -X POST "$API_URL/api/agents" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{
    "name": "eino-deer-agent",
    "model": "qwen.qwen3_max",
    "workflow": {
      "nodes": [
        {
          "id": "deer",
          "type": "eino_deer",
          "config": {
            "system_prompt": "You are a data analysis assistant."
          }
        }
      ],
      "edges": []
    }
  }')

AGENT2_ID=$(echo $CREATE2 | grep -o '"id":"[^"]*"' | cut -d'"' -f4)
echo "Agent ID: $AGENT2_ID"

MSG2=$(curl -s -X POST "$API_URL/api/agents/$AGENT2_ID/message" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"message": "Explain what is machine learning in simple terms."}')

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
# Test 3: Eino Manus Agent
# ============================================
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "📋 Test 3: Eino Manus Agent"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

CREATE3=$(curl -s -X POST "$API_URL/api/agents" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{
    "name": "eino-manus-agent",
    "model": "qwen.qwen3_max",
    "workflow": {
      "nodes": [
        {
          "id": "manus",
          "type": "eino_manus",
          "config": {
            "system_prompt": "You are a creative assistant."
          }
        }
      ],
      "edges": []
    }
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
echo "Result: $(echo $RESULT3 | head -c 100)..."
echo ""

# ============================================
# Test 4: Multi-step with Eino + LLM
# ============================================
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "📋 Test 4: Multi-step Eino + LLM"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

CREATE4=$(curl -s -X POST "$API_URL/api/agents" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{
    "name": "eino-multi-step",
    "model": "qwen.qwen3_max",
    "workflow": {
      "nodes": [
        {
          "id": "analyze",
          "type": "llm",
          "config": {"prompt": "What is 25 + 17?"}
        },
        {
          "id": "eino",
          "type": "eino_react"
        }
      ],
      "edges": [
        {"from": "analyze", "to": "eino"}
      ]
    }
  }')

AGENT4_ID=$(echo $CREATE4 | grep -o '"id":"[^"]*"' | cut -d'"' -f4)
echo "Agent ID: $AGENT4_ID"

MSG4=$(curl -s -X POST "$API_URL/api/agents/$AGENT4_ID/message" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"message": "Calculate and explain"}')

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
# Summary
# ============================================
echo "=========================================="
echo "   TEST SUMMARY"
echo "=========================================="
echo ""
echo "✅ Test 1 (Eino React): $(echo $STATUS1 | jq -r '.status')"
echo "   Result: $RESULT1"
echo ""
echo "✅ Test 2 (Eino DEER): $(echo $STATUS2 | jq -r '.status')"
echo "   Result: $RESULT2"
echo ""
echo "✅ Test 3 (Eino Manus): $(echo $STATUS3 | jq -r '.status')"
echo "   Result: $(echo $RESULT3 | head -c 100)..."
echo ""
echo "✅ Test 4 (Multi-step): $(echo $STATUS4 | jq -r '.status')"

echo ""
echo "📈 Observability:"
curl -s -X GET "$API_URL/api/observability/summary" \
  -H "Authorization: Bearer $TOKEN" | jq .

echo ""
echo "✅ All eino agent tests completed!"
