#!/bin/bash

# Eino ReAct Agent Tool Calling Test
# Tests that the agent actually calls tools and executes them

API_URL="http://localhost:8080"

get_token() {
  curl -s -X POST "$API_URL/api/login" \
    -H "Content-Type: application/json" \
    -d '{"username":"admin","password":"admin"}' | grep -o '"token":"[^"]*"' | cut -d'"' -f4
}

echo "=========================================="
echo "   Eino ReAct Tool Calling Test"
echo "   Testing: Real Tool Execution"
echo "=========================================="
echo ""

TOKEN=$(get_token)
echo "Token: ${TOKEN:0:50}..."
echo ""

# ============================================
# Test 1: Calculator Tool
# ============================================
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "📋 Test 1: Calculator Tool"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

CREATE1=$(curl -s -X POST "$API_URL/api/agents" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{
    "name": "calc-agent",
    "model": "qwen.qwen3_max",
    "workflow": {
      "nodes": [
        {
          "id": "agent",
          "type": "eino_react",
          "config": {
            "system_prompt": "You are a helpful assistant. Use tools when needed."
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
  -d '{"message": "What is 123 + 456? Use the calculator tool."}')

JOB1_ID=$(echo $MSG1 | grep -o '"job_id":"[^"]*"' | cut -d'"' -f4)
echo "Job ID: $JOB1_ID"

echo "⏳ Running (this should call the calculator tool)..."
sleep 10

STATUS1=$(curl -s -X GET "$API_URL/api/jobs/$JOB1_ID" \
  -H "Authorization: Bearer $TOKEN")
echo "Status: $(echo $STATUS1 | jq -r '.status')"

# Get events to check for tool calls
EVENTS1=$(curl -s -X GET "$API_URL/api/jobs/$JOB1_ID/events" \
  -H "Authorization: Bearer $TOKEN")

echo ""
echo "📊 Events:"
echo "$EVENTS1" | jq -r '.events[] | "  [\(.type)] \(.created_at)"'

# Check if tool was called
TOOL_CALLS=$(echo "$EVENTS1" | jq '[.events[] | select(.type == "tool_called" or .payload.tool_name != null)] | length')
echo ""
echo "🔧 Tool calls detected: $TOOL_CALLS"

RESULT1=$(echo $EVENTS1 | jq -r '.events[] | select(.type == "command_committed") | .payload.result.result')
echo ""
echo "📝 Result: $RESULT1"
echo ""

# ============================================
# Test 2: Weather Tool
# ============================================
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "📋 Test 2: Weather Tool"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

CREATE2=$(curl -s -X POST "$API_URL/api/agents" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{
    "name": "weather-agent",
    "model": "qwen.qwen3_max",
    "workflow": {
      "nodes": [
        {
          "id": "agent",
          "type": "eino_react"
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
  -d '{"message": "What is the weather in Beijing today?"}')

JOB2_ID=$(echo $MSG2 | grep -o '"job_id":"[^"]*"' | cut -d'"' -f4)
echo "Job ID: $JOB2_ID"

echo "⏳ Running..."
sleep 10

STATUS2=$(curl -s -X GET "$API_URL/api/jobs/$JOB2_ID" \
  -H "Authorization: Bearer $TOKEN")
echo "Status: $(echo $STATUS2 | jq -r '.status')"

EVENTS2=$(curl -s -X GET "$API_URL/api/jobs/$JOB2_ID/events" \
  -H "Authorization: Bearer $TOKEN")

echo ""
echo "📊 Events:"
echo "$EVENTS2" | jq -r '.events[] | "  [\(.type)] \(.created_at)"'

TOOL_CALLS2=$(echo "$EVENTS2" | jq '[.events[] | select(.type == "tool_called" or .payload.tool_name != null)] | length')
echo ""
echo "🔧 Tool calls detected: $TOOL_CALLS2"

RESULT2=$(echo $EVENTS2 | jq -r '.events[] | select(.type == "command_committed") | .payload.result.result')
echo ""
echo "📝 Result: $RESULT2"
echo ""

# ============================================
# Summary
# ============================================
echo "=========================================="
echo "   TEST SUMMARY"
echo "=========================================="
echo ""
echo "✅ Test 1 (Calculator): Status=$(echo $STATUS1 | jq -r '.status'), ToolCalls=$TOOL_CALLS"
echo "   Result: $RESULT1"
echo ""
echo "✅ Test 2 (Weather): Status=$(echo $STATUS2 | jq -r '.status'), ToolCalls=$TOOL_CALLS2"
echo "   Result: $RESULT2"
echo ""

# Check if tool calling worked
if [ "$TOOL_CALLS" -gt 0 ] || [ "$TOOL_CALLS2" -gt 0 ]; then
  echo "🎉 SUCCESS: Tool calling is working!"
else
  echo "❌ FAILED: No tool calls detected. The agent is not using tools."
fi
