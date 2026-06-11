"""
LangChain Agent 示例

展示如何将 LangChain Agent 集成到 Aetheris。
使用框架类型别名 `langchain` 自动获得 Aetheris 的持久化能力。

运行方式：
    export OPENAI_API_KEY=your_key_here
    pip install -r requirements.txt
    python app.py

测试：
    curl -X POST http://localhost:9002/invoke \
      -H "Content-Type: application/json" \
      -d '{"message": "What is the weather today?"}'
"""

import os
import logging
from typing import Optional, List
from datetime import datetime

from fastapi import FastAPI, HTTPException, Header
from pydantic import BaseModel, Field
from langchain_openai import ChatOpenAI
from langchain.agents import AgentExecutor, create_react_agent
from langchain_core.prompts import PromptTemplate
from langchain_core.tools import tool

# 配置日志
logging.basicConfig(
    level=logging.INFO,
    format="%(asctime)s - %(name)s - %(levelname)s - %(message)s"
)
logger = logging.getLogger("langchain-agent")

# FastAPI 应用
app = FastAPI(
    title="LangChain Agent",
    description="Aetheris LangChain Integration Example",
    version="1.0.0"
)

# ==================== 数据模型 ====================

class AgentRequest(BaseModel):
    """Aetheris 发送的请求格式"""
    message: str = Field(..., description="用户消息")
    session_id: Optional[str] = Field(None, description="会话ID")
    metadata: Optional[dict] = Field(default_factory=dict, description="元数据")

class AgentResponse(BaseModel):
    """返回给 Aetheris 的响应格式"""
    answer: str = Field(..., description="Agent 的回答")
    final: bool = Field(True, description="是否为最终结果")
    metadata: Optional[dict] = Field(default_factory=dict, description="响应元数据")

# ==================== LangChain 工具 ====================

@tool
def search_weather(city: str) -> str:
    """Search for current weather in a city."""
    # 模拟天气查询
    weather_data = {
        "北京": "北京今天晴，温度 25°C，湿度 40%",
        "上海": "上海今天多云，温度 28°C，湿度 60%",
        "深圳": "深圳今天小雨，温度 30°C，湿度 80%",
    }
    return weather_data.get(city, f"{city}天气未知")

@tool
def calculate(expression: str) -> str:
    """Calculate a mathematical expression."""
    try:
        result = eval(expression)
        return f"计算结果: {expression} = {result}"
    except Exception as e:
        return f"计算错误: {str(e)}"

@tool
def get_current_time() -> str:
    """Get current date and time."""
    return datetime.now().strftime("%Y-%m-%d %H:%M:%S")

# ==================== LangChain Agent ====================

def create_agent() -> AgentExecutor:
    """创建 LangChain Agent"""
    
    # 检查 API Key
    api_key = os.getenv("OPENAI_API_KEY")
    if not api_key:
        logger.warning("OPENAI_API_KEY not set, using mock mode")
        return None
    
    # 创建 LLM
    llm = ChatOpenAI(
        model="gpt-4o-mini",
        temperature=0,
        api_key=api_key
    )
    
    # 定义工具
    tools = [search_weather, calculate, get_current_time]
    
    # 定义 Prompt
    prompt = PromptTemplate.from_template("""
You are a helpful assistant. You can use tools to answer questions.

Tools available:
{tools}

Tool names: {tool_names}

Use the following format:
Question: the input question
Thought: think about what to do
Action: the action to take, should be one of [{tool_names}]
Action Input: the input to the action
Observation: the result of the action
... (this Thought/Action/Action Input/Observation can repeat N times)
Thought: I now know the final answer
Final Answer: the final answer to the original input question

Begin!

Question: {input}
Thought:{agent_scratchpad}
""")
    
    # 创建 Agent
    agent = create_react_agent(llm, tools, prompt)
    
    # 创建 Executor
    return AgentExecutor(
        agent=agent,
        tools=tools,
        verbose=True,
        handle_parsing_errors=True,
        max_iterations=5
    )

# 全局 Agent 实例
agent_executor = create_agent()

# ==================== Mock 模式 ====================

def mock_process(message: str) -> str:
    """当没有 OPENAI_API_KEY 时使用 mock 模式"""
    logger.info("Using mock mode (no OPENAI_API_KEY)")
    
    if "天气" in message or "weather" in message.lower():
        return "今天天气晴朗，温度 25°C（Mock 模式）"
    
    if "计算" in message or "calculate" in message.lower():
        return "计算结果: 42（Mock 模式）"
    
    if "时间" in message or "time" in message.lower():
        return f"当前时间: {datetime.now().strftime('%Y-%m-%d %H:%M:%S')}（Mock 模式）"
    
    return f"收到消息: {message}（Mock 模式 - 请设置 OPENAI_API_KEY 启用完整功能）"

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
    
    使用 langchain 类型时，Aetheris 会自动：
    1. 传递 Idempotency-Key 用于去重
    2. 传递 Job-ID 用于追踪
    3. 处理超时和重试
    """
    logger.info(f"Received request: job_id={job_id}, agent_id={agent_id}")
    logger.info(f"Message: {request.message[:100]}...")
    
    try:
        # 使用 LangChain Agent 或 Mock 模式
        if agent_executor:
            result = agent_executor.invoke({
                "input": request.message
            })
            answer = result.get("output", "No output")
        else:
            answer = mock_process(request.message)
        
        # 构建响应
        response = AgentResponse(
            answer=answer,
            final=True,
            metadata={
                "agent_id": agent_id or "langchain_agent",
                "job_id": job_id,
                "idempotency_key": idempotency_key,
                "processed_at": datetime.utcnow().isoformat(),
                "framework": "langchain",
                "session_id": request.session_id,
            }
        )
        
        logger.info(f"Response generated for job_id={job_id}")
        return response
        
    except Exception as e:
        logger.error(f"Error processing message: {e}")
        raise HTTPException(status_code=500, detail=str(e))

@app.get("/health")
async def health():
    """健康检查端点"""
    return {
        "status": "healthy",
        "timestamp": datetime.utcnow().isoformat(),
        "agent_type": "langchain",
        "has_api_key": bool(os.getenv("OPENAI_API_KEY"))
    }

@app.get("/")
async def root():
    """根路径，返回 Agent 信息"""
    return {
        "name": "LangChain Agent Example",
        "description": "Aetheris LangChain Integration Example",
        "endpoints": {
            "invoke": "/invoke",
            "health": "/health"
        },
        "version": "1.0.0",
        "tools": ["search_weather", "calculate", "get_current_time"]
    }

# ==================== 启动 ====================

if __name__ == "__main__":
    import uvicorn
    
    port = int(os.getenv("AGENT_PORT", "9002"))
    logger.info(f"Starting LangChain Agent on port {port}")
    
    if not os.getenv("OPENAI_API_KEY"):
        logger.warning("=" * 60)
        logger.warning("OPENAI_API_KEY not set!")
        logger.warning("Running in mock mode.")
        logger.warning("Set OPENAI_API_KEY to enable full functionality.")
        logger.warning("=" * 60)
    
    uvicorn.run(
        app,
        host="0.0.0.0",
        port=port,
        log_level="info"
    )
