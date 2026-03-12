#!/bin/bash

# Agent API 测试脚本
# 使用方法: ./test_agent.sh

API_URL="http://localhost:8080"

echo "=== 1. 登录获取 Token ==="
LOGIN_RESPONSE=$(curl -s -X POST "$API_URL/api/login" \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"admin"}')

echo "登录响应: $LOGIN_RESPONSE"

# 提取 token (简单方式)
TOKEN=$(echo $LOGIN_RESPONSE | grep -o '"token":"[^"]*"' | cut -d'"' -f4)
if [ -z "$TOKEN" ]; then
  # 尝试另一种格式
  TOKEN=$(echo $LOGIN_RESPONSE | grep -o 'token[^,}]*' | head -1 | cut -d':' -f2 | tr -d ' "')
fi

echo "获取的 Token: $TOKEN"

if [ -z "$TOKEN" ]; then
  echo "❌ 无法获取 token"
  exit 1
fi

echo ""
echo "=== 2. 创建 Agent ==="
CREATE_AGENT_RESPONSE=$(curl -s -X POST "$API_URL/api/agents" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{
    "name": "test-agent",
    "model": "openai.gpt_35_turbo",
    "description": "Test agent for eino integration"
  }')

echo "创建 Agent 响应: $CREATE_AGENT_RESPONSE"

# 提取 agent ID
AGENT_ID=$(echo $CREATE_AGENT_RESPONSE | grep -o '"id":"[^"]*"' | head -1 | cut -d'"' -f4)
if [ -z "$AGENT_ID" ]; then
  AGENT_ID=$(echo $CREATE_AGENT_RESPONSE | grep -o 'id[^,}]*' | head -1 | cut -d':' -f2 | tr -d ' "}')
fi

echo "Agent ID: $AGENT_ID"

echo ""
echo "=== 3. 发送消息到 Agent ==="
MESSAGE_RESPONSE=$(curl -s -X POST "$API_URL/api/agents/$AGENT_ID/message" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{
    "message": "Hello, what is 2+2?"
  }')

echo "消息响应: $MESSAGE_RESPONSE"

echo ""
echo "=== 4. 获取 Agent 状态 ==="
STATE_RESPONSE=$(curl -s -X GET "$API_URL/api/agents/$AGENT_ID/state" \
  -H "Authorization: Bearer $TOKEN")

echo "状态响应: $STATE_RESPONSE"

echo ""
echo "=== 5. 列出所有 Agents ==="
LIST_RESPONSE=$(curl -s -X GET "$API_URL/api/agents" \
  -H "Authorization: Bearer $TOKEN")

echo "列表响应: $LIST_RESPONSE"

echo ""
echo "=== 测试完成 ==="
