#!/bin/bash

# 生成真实测试数据
# 使用方法: ./generate_test_data.sh

API_URL="http://localhost:8080"

echo "=== 1. 登录获取 Token ==="
LOGIN_RESPONSE=$(curl -s -X POST "$API_URL/api/login" \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"admin"}')

TOKEN=$(echo $LOGIN_RESPONSE | grep -o '"token":"[^"]*"' | cut -d'"' -f4)
echo "Token: ${TOKEN:0:50}..."

echo ""
echo "=== 2. 创建测试 Agent ==="
AGENT_RESPONSE=$(curl -s -X POST "$API_URL/api/agents" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{
    "name": "test-agent",
    "model": "openai.gpt_35_turbo",
    "description": "Test agent for generating metrics"
  }')

AGENT_ID=$(echo $AGENT_RESPONSE | grep -o '"id":"[^"]*"' | cut -d'"' -f4)
echo "Agent ID: $AGENT_ID"

echo ""
echo "=== 3. 发送多个测试任务 ==="

# 简单问答
echo "- 发送任务 1: 简单问答"
curl -s -X POST "$API_URL/api/agents/$AGENT_ID/message" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"message": "What is the capital of France?"}' > /dev/null

# 数学计算
echo "- 发送任务 2: 数学计算"
curl -s -X POST "$API_URL/api/agents/$AGENT_ID/message" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"message": "Calculate 123 * 456 + 789"}' > /dev/null

#
echo "- 复杂推理发送任务 3: 复杂推理"
curl -s -X POST "$API_URL/api/agents/$AGENT_ID/message" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"message": "Explain how photosynthesis works in simple terms"}' > /dev/null

# 列表任务
echo "- 发送任务 4: 列表任务"
curl -s -X POST "$API_URL/api/agents/$AGENT_ID/message" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"message": "List 5 programming languages"}' > /dev/null

# 翻译任务
echo "- 发送任务 5: 翻译任务"
curl -s -X POST "$API_URL/api/agents/$AGENT_ID/message" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"message": "Translate hello to Chinese, Japanese, and Korean"}' > /dev/null

echo ""
echo "=== 4. 创建第二个 Agent ==="
AGENT_RESPONSE2=$(curl -s -X POST "$API_URL/api/agents" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{
    "name": "math-agent",
    "model": "openai.gpt_35_turbo",
    "description": "Math helper agent"
  }')

AGENT_ID2=$(echo $AGENT_RESPONSE2 | grep -o '"id":"[^"]*"' | cut -d'"' -f4)
echo "Agent 2 ID: $AGENT_ID2"

echo ""
echo "=== 5. 发送更多数学任务到 Agent 2 ==="
for i in {1..5}; do
  echo "- 发送数学任务 $i"
  curl -s -X POST "$API_URL/api/agents/$AGENT_ID2/message" \
    -H "Content-Type: application/json" \
    -H "Authorization: Bearer $TOKEN" \
    -d "{\"message\": \"What is $i * $((i+1))?\"}" > /dev/null
done

echo ""
echo "=== 6. 等待任务完成 ==="
sleep 30

echo ""
echo "=== 7. 获取所有 Jobs ==="
curl -s -X GET "$API_URL/api/agents/$AGENT_ID/jobs" \
  -H "Authorization: Bearer $TOKEN" | head -c 500

echo ""
echo ""
echo "=== 数据生成完成 ==="
echo "等待 1-2 分钟后在 Grafana 查看监控数据"
