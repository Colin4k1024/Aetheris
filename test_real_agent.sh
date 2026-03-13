#!/bin/bash

# Real Agent Test with Tools - Using actual LLM
# This test creates a real agent with TaskGraph that uses tools

API_URL="http://localhost:8080"

get_token() {
  curl -s -X POST "$API_URL/api/login" \
    -H "Content-Type: application/json" \
    -d '{"username":"admin","password":"admin"}' | grep -o '"token":"[^"]*"' | cut -d'"' -f4
}

echo "=========================================="
echo "   Real Agent with Tools Test"
echo "   Using Ollama LLM + Real Tool Execution"
echo "=========================================="
echo ""

TOKEN=$(get_token)
echo "🔑 Token: ${TOKEN:0:50}..."
echo ""

# ============================================
# Test: Agent with Tool Node in TaskGraph
# ============================================
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "📋 Creating Agent with Tool Execution"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

# Create an agent with a workflow that includes a tool node
CREATE=$(curl -s -X POST "$API_URL/api/agents" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{
    "name": "calculator-agent",
    "model": "openai.gpt_35_turbo",
    "description": "Agent with calculator tool",
    "workflow": {
      "nodes": [
        {
          "id": "calculate",
          "type": "tool",
          "config": {
            "tool_name": "calculator",
            "input": {"operation": "add", "value1": 100, "value2": 200}
          }
        }
      ],
      "edges": []
    }
  }')

echo "Create Response: $CREATE"
AGENT_ID=$(echo $CREATE | grep -o '"id":"[^"]*"' | cut -d'"' -f4)
echo "Agent ID: $AGENT_ID"

MSG=$(curl -s -X POST "$API_URL/api/agents/$AGENT_ID/message" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"message": "Calculate 100 + 200"}')

JOB_ID=$(echo $MSG | grep -o '"job_id":"[^"]*"' | cut -d'"' -f4)
echo "Job ID: $JOB_ID"

echo ""
echo "⏳ Waiting for job to complete..."
sleep 5

# Check job status
STATUS=$(curl -s -X GET "$API_URL/api/jobs/$JOB_ID" \
  -H "Authorization: Bearer $TOKEN")
echo "Job Status: $(echo $STATUS | grep -o '"status":"[^"]*"' | cut -d'"' -f4)"

# Get events to see tool execution
echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "📊 Job Events (Tool Execution)"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

EVENTS=$(curl -s -X GET "$API_URL/api/jobs/$JOB_ID/events" \
  -H "Authorization: Bearer $TOKEN")

echo "$EVENTS" | jq -r '.events[] | select(.type | contains("tool") or contains("command") or contains("node")) | "[\(.type)] \(.created_at): \(.payload | tojson)"'

echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "📊 Full Trace"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

TRACE=$(curl -s -X GET "$API_URL/api/jobs/$JOB_ID/trace" \
  -H "Authorization: Bearer $TOKEN")

echo "$TRACE" | jq '{job_id, steps: .steps | length, timeline: .timeline | length}'

echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "📋 Test 2: Multi-step with LLM + Tool"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

# Create agent with LLM node + Tool node
CREATE2=$(curl -s -X POST "$API_URL/api/agents" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{
    "name": "llm-with-tool-agent",
    "model": "openai.gpt_35_turbo",
    "description": "LLM then tool execution",
    "workflow": {
      "nodes": [
        {
          "id": "analyze",
          "type": "llm",
          "config": {"prompt": "What mathematical operation is needed for 50 * 4?"}
        },
        {
          "id": "calculate",
          "type": "tool",
          "config": {"tool_name": "calculator"}
        }
      ],
      "edges": [
        {"from": "analyze", "to": "calculate"}
      ]
    }
  }')

AGENT2_ID=$(echo $CREATE2 | grep -o '"id":"[^"]*"' | cut -d'"' -f4)
echo "Agent ID: $AGENT2_ID"

MSG2=$(curl -s -X POST "$API_URL/api/agents/$AGENT2_ID/message" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"message": "Calculate 50 * 4"}')

JOB2_ID=$(echo $MSG2 | grep -o '"job_id":"[^"]*"' | cut -d'"' -f4)
echo "Job ID: $JOB2_ID"

echo ""
echo "⏳ Waiting for job to complete..."
sleep 8

STATUS2=$(curl -s -X GET "$API_URL/api/jobs/$JOB2_ID" \
  -H "Authorization: Bearer $TOKEN")
echo "Job Status: $(echo $STATUS2 | grep -o '"status":"[^"]*"' | cut -d'"' -f4)"

# Get trace
TRACE2=$(curl -s -X GET "$API_URL/api/jobs/$JOB2_ID/trace" \
  -H "Authorization: Bearer $TOKEN")

echo "Steps: $(echo $TRACE2 | jq '.steps | length')"
echo "Nodes: $(echo $TRACE2 | jq '.node_durations | length')"

echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "📋 Test 3: ReAct Agent (with tool calling)"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

# Create a more complex agent
CREATE3=$(curl -s -X POST "$API_URL/api/agents" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{
    "name": "react-agent",
    "model": "openai.gpt_35_turbo",
    "description": "ReAct agent with tools"
  }')

AGENT3_ID=$(echo $CREATE3 | grep -o '"id":"[^"]*"' | cut -d'"' -f4)
echo "Agent ID: $AGENT3_ID"

MSG3=$(curl -s -X POST "$API_URL/api/agents/$AGENT3_ID/message" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"message": "What is 25 + 17? Use the calculator."}')

JOB3_ID=$(echo $MSG3 | grep -o '"job_id":"[^"]*"' | cut -d'"' -f4)
echo "Job ID: $JOB3_ID"

echo ""
echo "⏳ Waiting for job to complete..."
sleep 8

STATUS3=$(curl -s -X GET "$API_URL/api/jobs/$JOB3_ID" \
  -H "Authorization: Bearer $TOKEN")
echo "Job Status: $(echo $STATUS3 | grep -o '"status":"[^"]*"' | cut -d'"' -f4)"

# Get events
EVENTS3=$(curl -s -X GET "$API_URL/api/jobs/$JOB3_ID/events" \
  -H "Authorization: Bearer $TOKEN")

echo ""
echo "Tool/Command Events:"
echo "$EVENTS3" | jq -r '.events[] | select(.type | contains("tool") or contains("command") or contains("node") or contains("llm")) | "[\(.type)] \(.payload | tojson)"'

echo ""
echo "=========================================="
echo "   Summary"
echo "=========================================="
echo ""
echo "✅ Test 1 (Tool Node): $(echo $STATUS | grep -o '"status":"[^"]*"' | cut -d'"' -f4)"
echo "✅ Test 2 (LLM + Tool): $(echo $STATUS2 | grep -o '"status":"[^"]*"' | cut -d'"' -f4)"
echo "✅ Test 3 (ReAct): $(echo $STATUS3 | grep -o '"status":"[^"]*"' | cut -d'"' -f4)"

echo ""
echo "📊 All Jobs:"
curl -s -X GET "$API_URL/api/agents" \
  -H "Authorization: Bearer $TOKEN" | jq '.agents | length' | xargs -I {} echo "   Total Agents: {}"

# Observability
echo ""
echo "📈 Observability:"
curl -s -X GET "$API_URL/api/observability/summary" \
  -H "Authorization: Bearer $TOKEN" | jq .

echo ""
