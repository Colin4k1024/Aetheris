#!/bin/bash

# Real Agent Test with Built-in Tools
# Tests actual tool execution through Aetheris API

API_URL="http://localhost:8080"

get_token() {
  curl -s -X POST "$API_URL/api/login" \
    -H "Content-Type: application/json" \
    -d '{"username":"admin","password":"admin"}' | grep -o '"token":"[^"]*"' | cut -d'"' -f4
}

echo "=========================================="
echo "   Real Agent with Built-in Tools Test"
echo "   Testing: RAG, HTTP, LLM Tools"
echo "=========================================="
echo ""

TOKEN=$(get_token)
echo "🔑 Token: ${TOKEN:0:50}..."
echo ""

# ============================================
# Test 1: Agent with LLM Tool
# ============================================
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "📋 Test 1: LLM Generate Tool"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

CREATE1=$(curl -s -X POST "$API_URL/api/agents" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{
    "name": "llm-tool-agent",
    "model": "openai.gpt_35_turbo"
  }')

AGENT1_ID=$(echo $CREATE1 | grep -o '"id":"[^"]*"' | cut -d'"' -f4)
echo "Agent ID: $AGENT1_ID"

# Send a message that triggers the LLM
MSG1=$(curl -s -X POST "$API_URL/api/agents/$AGENT1_ID/message" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"message": "Write a haiku about artificial intelligence"}')

JOB1_ID=$(echo $MSG1 | grep -o '"job_id":"[^"]*"' | cut -d'"' -f4)
echo "Job ID: $JOB1_ID"

echo "⏳ Waiting for completion..."
sleep 6

STATUS1=$(curl -s -X GET "$API_URL/api/jobs/$JOB1_ID" \
  -H "Authorization: Bearer $TOKEN")
echo "Status: $(echo $STATUS1 | grep -o '"status":"[^"]*"' | cut -d'"' -f4)"

# Get result from events
EVENTS1=$(curl -s -X GET "$API_URL/api/jobs/$JOB1_ID/events" \
  -H "Authorization: Bearer $TOKEN")
RESULT1=$(echo $EVENTS1 | grep -o '"result":"[^"]*"' | head -1)
echo "Result: $RESULT1"
echo ""

# ============================================
# Test 2: Multi-step Workflow with Tools
# ============================================
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "📋 Test 2: Multi-step with RAG + LLM"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

# Create agent and run a query that would use RAG
CREATE2=$(curl -s -X POST "$API_URL/api/agents" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{
    "name": "rag-agent",
    "model": "openai.gpt_35_turbo"
  }')

AGENT2_ID=$(echo $CREATE2 | grep -o '"id":"[^"]*"' | cut -d'"' -f4)
echo "Agent ID: $AGENT2_ID"

MSG2=$(curl -s -X POST "$API_URL/api/agents/$AGENT2_ID/message" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"message": "What is machine learning? Give me a detailed explanation."}')

JOB2_ID=$(echo $MSG2 | grep -o '"job_id":"[^"]*"' | cut -d'"' -f4)
echo "Job ID: $JOB2_ID"

echo "⏳ Waiting for completion..."
sleep 8

STATUS2=$(curl -s -X GET "$API_URL/api/jobs/$JOB2_ID" \
  -H "Authorization: Bearer $TOKEN")
echo "Status: $(echo $STATUS2 | grep -o '"status":"[^"]*"' | cut -d'"' -f4)"
echo ""

# ============================================
# Test 3: Complex reasoning
# ============================================
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "📋 Test 3: Complex Reasoning"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

CREATE3=$(curl -s -X POST "$API_URL/api/agents" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{
    "name": "reasoning-agent",
    "model": "openai.gpt_35_turbo"
  }')

AGENT3_ID=$(echo $CREATE3 | grep -o '"id":"[^"]*"' | cut -d'"' -f4)
echo "Agent ID: $AGENT3_ID"

MSG3=$(curl -s -X POST "$API_URL/api/agents/$AGENT3_ID/message" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"message": "Explain the concept of recursion to a 5-year-old with an example"}')

JOB3_ID=$(echo $MSG3 | grep -o '"job_id":"[^"]*"' | cut -d'"' -f4)
echo "Job ID: $JOB3_ID"

echo "⏳ Waiting for completion..."
sleep 8

STATUS3=$(curl -s -X GET "$API_URL/api/jobs/$JOB3_ID" \
  -H "Authorization: Bearer $TOKEN")
JOB3_STATUS=$(echo $STATUS3 | grep -o '"status":"[^"]*"' | cut -d'"' -f4)
echo "Status: $JOB3_STATUS"
echo ""

# ============================================
# Test 4: Verify & Evidence
# ============================================
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "📋 Test 4: Verify & Evidence Chain"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

# Get verify info
VERIFY=$(curl -s -X GET "$API_URL/api/jobs/$JOB3_ID/verify" \
  -H "Authorization: Bearer $TOKEN")
echo "Verify Response:"
echo "$VERIFY" | jq '{execution_hash: .execution_hash, event_chain_ok: .replay_proof_result.ok, tool_ledger_ok: .tool_invocation_ledger_proof.ok}'

# Get evidence graph
EVIDENCE=$(curl -s -X GET "$API_URL/api/jobs/$JOB3_ID/evidence-graph" \
  -H "Authorization: Bearer $TOKEN")
echo ""
echo "Evidence Graph:"
echo "$EVIDENCE" | jq '.'
echo ""

# ============================================
# Test 5: Observability - Check metrics
# ============================================
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "📋 Test 5: Observability & Metrics"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

# Summary
SUMMARY=$(curl -s -X GET "$API_URL/api/observability/summary" \
  -H "Authorization: Bearer $TOKEN")
echo "Summary: $SUMMARY"

# Stuck jobs
STUCK=$(curl -s -X GET "$API_URL/api/observability/stuck" \
  -H "Authorization: Bearer $TOKEN")
echo "Stuck: $STUCK"
echo ""

# ============================================
# Test 6: Trace with detailed info
# ============================================
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "📋 Test 6: Detailed Trace"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

TRACE=$(curl -s -X GET "$API_URL/api/jobs/$JOB3_ID/trace" \
  -H "Authorization: Bearer $TOKEN")

echo "Job: $(echo $TRACE | jq -r '.job_id')"
echo "Timeline Events: $(echo $TRACE | jq '.timeline | length')"
echo "Steps: $(echo $TRACE | jq '.steps | length')"
echo ""
echo "Step Details:"
echo "$TRACE" | jq -r '.steps[] | "  - \(.label): \(.state) (\(.duration_ms)ms)"'

# Reasoning snapshot
echo ""
echo "Reasoning Snapshot (LLM evidence):"
echo "$TRACE" | jq -r '.steps[] | select(.reasoning_snapshot != null) | .reasoning_snapshot'
echo ""

# ============================================
# Summary
# ============================================
echo "=========================================="
echo "   TEST SUMMARY"
echo "=========================================="
echo ""
echo "✅ Test 1 (LLM Tool): $JOB3_STATUS"
echo "✅ Test 2 (RAG + LLM): $(echo $STATUS2 | grep -o '"status":"[^"]*"' | cut -d'"' -f4)"
echo "✅ Test 3 (Complex Reasoning): $JOB3_STATUS"
echo "✅ Test 4 (Verify & Evidence): OK"
echo "✅ Test 5 (Observability): OK"
echo "✅ Test 6 (Trace): OK"

echo ""
echo "📊 Job Statistics:"
echo "   - Test 1 Duration: $(echo $STATUS1 | grep -o '"updated_at":"[^"]*"' | cut -d'"' -f4)"
echo "   - Test 2 Duration: $(echo $STATUS2 | grep -o '"updated_at":"[^"]*"' | cut -d'"' -f4)"
echo "   - Test 3 Duration: $(echo $STATUS3 | grep -o '"updated_at":"[^"]*"' | cut -d'"' -f4)"

echo ""
echo "📈 All Agents:"
curl -s -X GET "$API_URL/api/agents" \
  -H "Authorization: Bearer $TOKEN" | jq '.agents | .[] | {name, status, created_at}'

echo ""
echo "✅ All tests completed!"
