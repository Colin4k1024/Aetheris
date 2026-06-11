"""
External HTTP Agent 示例

这是一个通用的 Python HTTP Agent 示例，展示如何将现有 Agent 集成到 Aetheris。
支持标准 JSON 协议和幂等性。

运行方式：
    pip install -r requirements.txt
    python app.py

测试：
    curl -X POST http://localhost:9001/invoke \
      -H "Content-Type: application/json" \
      -d '{"message": "Hello!"}'
"""

import os
import logging
import uuid
from datetime import datetime
from typing import Optional

from fastapi import FastAPI, HTTPException, Header, Request
from pydantic import BaseModel, Field

# 配置日志
logging.basicConfig(
    level=logging.INFO,
    format="%(asctime)s - %(name)s - %(levelname)s - %(message)s"
)
logger = logging.getLogger("external-agent")

# FastAPI 应用
app = FastAPI(
    title="External HTTP Agent",
    description="Aetheris External Agent Integration Example",
    version="1.0.0"
)

# ==================== 数据模型 ====================

class AgentRequest(BaseModel):
    """Aetheris 发送的请求格式"""
    message: str = Field(..., description="用户消息或任务描述")
    session_id: Optional[str] = Field(None, description="会话ID，用于多轮对话")
    metadata: Optional[dict] = Field(default_factory=dict, description="元数据")

class AgentResponse(BaseModel):
    """返回给 Aetheris 的响应格式"""
    answer: str = Field(..., description="Agent 的回答")
    final: bool = Field(True, description="是否为最终结果")
    metadata: Optional[dict] = Field(default_factory=dict, description="响应元数据")

# ==================== 幂等性存储 ====================

class IdempotencyStore:
    """
    简单的内存幂等性存储
    
    生产环境应使用 Redis 或数据库
    """
    def __init__(self):
        self._store: dict[str, AgentResponse] = {}
    
    def get(self, key: str) -> Optional[AgentResponse]:
        return self._store.get(key)
    
    def set(self, key: str, response: AgentResponse):
        self._store[key] = response
    
    def has(self, key: str) -> bool:
        return key in self._store

idempotency_store = IdempotencyStore()

# ==================== Agent 逻辑 ====================

def process_message(message: str, metadata: dict) -> str:
    """
    你的 Agent 逻辑
    
    这里是示例实现，替换为你自己的 Agent 代码。
    可以是：
    - 调用 LLM API
    - 运行 LangChain/AutoGen Agent
    - 执行自定义逻辑
    - 调用其他微服务
    """
    logger.info(f"Processing message: {message[:100]}...")
    
    # 示例：简单的回显 + 处理
    if "天气" in message or "weather" in message.lower():
        return f"今天天气晴朗，温度 25°C。您的问题：{message}"
    
    if "翻译" in message or "translate" in message.lower():
        return f"Translation result: {message}"
    
    if "总结" in message or "summarize" in message.lower():
        return f"Summary: {message[:50]}..."
    
    # 默认响应
    return f"Processed: {message}"

# ==================== API 端点 ====================

@app.post("/invoke", response_model=AgentResponse)
async def invoke(
    request: AgentRequest,
    idempotency_key: Optional[str] = Header(None, alias="Idempotency-Key"),
    job_id: Optional[str] = Header(None, alias="X-Aetheris-Job-ID"),
    agent_id: Optional[str] = Header(None, alias="X-Aetheris-Agent-ID"),
):
    """
    Aetheris 调用的主端点
    
    Headers:
        Idempotency-Key: 幂等键，用于去重
        X-Aetheris-Job-ID: Aetheris 任务 ID
        X-Aetheris-Agent-ID: Agent 配置 ID
    """
    logger.info(f"Received request: job_id={job_id}, agent_id={agent_id}")
    logger.info(f"Message: {request.message[:100]}...")
    
    # 1. 检查幂等性
    if idempotency_key and idempotency_store.has(idempotency_key):
        logger.info(f"Returning cached response for key: {idempotency_key}")
        return idempotency_store.get(idempotency_key)
    
    # 2. 处理消息
    try:
        answer = process_message(request.message, request.metadata or {})
    except Exception as e:
        logger.error(f"Error processing message: {e}")
        raise HTTPException(status_code=500, detail=str(e))
    
    # 3. 构建响应
    response = AgentResponse(
        answer=answer,
        final=True,
        metadata={
            "agent_id": agent_id or "unknown",
            "job_id": job_id,
            "idempotency_key": idempotency_key,
            "processed_at": datetime.utcnow().isoformat(),
            "session_id": request.session_id,
        }
    )
    
    # 4. 缓存响应（用于幂等性）
    if idempotency_key:
        idempotency_store.set(idempotency_key, response)
    
    logger.info(f"Response generated for job_id={job_id}")
    return response

@app.get("/health")
async def health():
    """健康检查端点"""
    return {
        "status": "healthy",
        "timestamp": datetime.utcnow().isoformat(),
        "agent_type": "external_http"
    }

@app.get("/")
async def root():
    """根路径，返回 Agent 信息"""
    return {
        "name": "External HTTP Agent Example",
        "description": "Aetheris External Agent Integration Example",
        "endpoints": {
            "invoke": "/invoke",
            "health": "/health"
        },
        "version": "1.0.0"
    }

# ==================== 启动 ====================

if __name__ == "__main__":
    import uvicorn
    
    port = int(os.getenv("AGENT_PORT", "9001"))
    logger.info(f"Starting External HTTP Agent on port {port}")
    
    uvicorn.run(
        app,
        host="0.0.0.0",
        port=port,
        log_level="info"
    )
