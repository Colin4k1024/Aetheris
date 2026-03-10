# 集成 LangGraph/AutoGen：在 Aetheris 上运行已有 Agent

> 不需要重写代码，直接把现有的 LangGraph、AutoGen、CrewAI Agent 部署到 Aetheris 上。

## 0. 背景：存量 Agent 的困境

你可能已经有了一个用 LangGraph、AutoGen 或 CrewAI 写的 Agent：

```
LangGraph Agent:
┌─────────────┐     ┌─────────────┐     ┌─────────────┐
│   Agent     │ ──▶ │   Tools     │ ──▶ │   State      │
│  (Define)   │     │  (Actions)  │     │  (Memory)    │
└─────────────┘     └─────────────┘     └─────────────┘

问题：
- 如何持久化？崩溃后怎么办？
- 如何保证工具调用不重复？
- 如何支持人工审批？
- 如何审计？
```

**答案：不需要重写，把现有 Agent 接入 Aetheris。**

## 1. Aetheris 的 Adapter 架构

### 1.1 架构设计

```
┌─────────────────────────────────────────────────────────────┐
│                   Aetheris Adapter Layer                    │
├─────────────────────────────────────────────────────────────┤
│  ┌────────────┐  ┌────────────┐  ┌────────────┐            │
│  │ LangGraph  │  │  AutoGen   │  │   CrewAI   │            │
│  │  Adapter   │  │  Adapter   │  │  Adapter   │            │
│  └─────┬──────┘  └─────┬──────┘  └─────┬──────┘            │
│        │               │               │                    │
│        └───────────────┼───────────────┘                    │
│                        ▼                                     │
│              ┌─────────────────┐                            │
│              │  Node Adapter    │  ← 统一抽象               │
│              │   Interface      │                            │
│              └────────┬────────┘                            │
└───────────────────────┼─────────────────────────────────────┘
                        │
                        ▼
┌─────────────────────────────────────────────────────────────┐
│                   Aetheris Runtime                           │
│  - 事件溯源     - Checkpoint     - At-Most-Once            │
│  - Human-in-loop - 审计回放                                 │
└─────────────────────────────────────────────────────────────┘
```

### 1.2 Adapter 接口

```go
// NodeAdapter 定义了所有框架适配器的统一接口
type NodeAdapter interface {
    // 初始化 Agent
    Initialize(ctx context.Context, config AgentConfig) error
    
    // 将 Agent 编译为可执行的 Node
    Compile() (Node, error)
    
    // 获取初始状态
    GetInitialState() (map[string]interface{}, error)
    
    // 序列化/反序列化状态
    SerializeState(state map[string]interface{}) ([]byte, error)
    DeserializeState(data []byte) (map[string]interface{}, error)
}

// 适配器必须实现的节点类型
type Node interface {
    // 执行节点
    Execute(ctx context.Context, input Input) (Output, error)
    
    // 获取节点类型
    GetType() string
    
    // 获取节点名称
    GetName() string
}
```

## 2. LangGraph Adapter

### 2.1 什么是 LangGraph？

LangGraph 是 LangChain 的扩展，用**图**的方式来定义 Agent：

```python
from langgraph.graph import StateGraph
from langgraph.prebuilt import tool_node

# 定义状态
class AgentState(TypedDict):
    messages: list
    next_action: str

# 定义图
graph = StateGraph(AgentState)

graph.add_node("agent", agent_node)
graph.add_node("tools", tool_node)

graph.add_edge("__start__", "agent")
graph.add_conditional_edges("agent", should_continue)
graph.add_edge("tools", "agent")

app = graph.compile()
```

### 2.2 接入 Aetheris

```go
// 1. 引入 LangGraph Adapter
import (
    "rag-platform/adapters/langgraph"
)

// 2. 创建 Adapter 实例
adapter := langgraph.NewAdapter(&langgraph.Config{
    GraphPath: "./my_agent/graph.json",  // 导出的 LangGraph 图
    StateConfig: langgraph.StateConfig{
        MessageKey: "messages",
    },
})

// 3. 编译为 Aetheris 可执行的 Node
node, err := adapter.Compile()
if err != nil {
    return err
}

// 4. 注册到 Aetheris
runtime.RegisterNode("langgraph_agent", node)
```

### 2.3 导出 LangGraph 图

```python
# 在 LangGraph 端导出图结构
import json

# 获取图的节点和边
graph_data = {
    "nodes": [
        {"id": "agent", "type": "agent", "data": {...}},
        {"id": "tools", "type": "tool", "data": {...}}
    ],
    "edges": [
        {"source": "__start__", "target": "agent"},
        {"source": "agent", "target": "tools"},
        {"source": "tools", "target": "agent"}
    ],
    "state_schema": AgentState.schema()
}

with open("graph.json", "w") as f:
    json.dump(graph_data, f)
```

### 2.4 状态映射

```
LangGraph State          Aetheris Checkpoint
────────────────────────────────────────────────
messages: [...]      ↔    checkpoint.state["messages"]
next_action: "tool"  ↔    checkpoint.state["next_action"]
```

```go
// LangGraph Adapter 内部
func (a *LangGraphAdapter) DeserializeState(data []byte) (map[string]interface{}, error) {
    var state map[string]interface{}
    json.Unmarshal(data, &state)
    
    // 转换为 LangGraph 格式
    messages := state["messages"].([]interface{})
    
    // 构建 LangChain 消息对象
    langgraphMessages := make([]BaseMessage, len(messages))
    for i, msg := range messages {
        langgraphMessages[i] = convertToLangChainMessage(msg)
    }
    
    return map[string]interface{}{
        "messages": langgraphMessages,
    }, nil
}
```

## 3. AutoGen Adapter

### 3.1 什么是 AutoGen？

Microsoft AutoGen 是一个多 Agent 框架：

```python
from autogen import ConversableAgent, AssistantAgent

# 定义 Agent
assistant = AssistantAgent(
    name="assistant",
    llm_config={"model": "gpt-4"}
)

user_proxy = ConversableAgent(
    name="user_proxy",
    is_termination_msg=lambda msg: "TERMINATE" in msg.get("content", ""),
)

# 启动对话
result = user_proxy.initiate_chat(
    assistant,
    message="写一个 Python 函数计算斐波那契数列"
)
```

### 3.2 接入 Aetheris

```go
// 1. 创建 AutoGen Adapter
adapter := autogen.NewAdapter(&autogen.Config{
    AgentConfigPath: "./agents.yaml",
    GroupChatConfig: "./groupchat.yaml",
})

// 2. 编译
node, err := adapter.Compile()
if err != nil {
    return err
}

// 3. 注册
runtime.RegisterNode("autogen_team", node)
```

### 3.3 Agent 配置

```yaml
# agents.yaml
agents:
  - name: assistant
    type: ConversableAgent
    llm_config:
      model: gpt-4
      temperature: 0.7
    system_message: |
      你是一个专业的 Python 开发者。
      
  - name: code_reviewer
    type: ConversableAgent
    llm_config:
      model: gpt-4
    system_message: |
      你是一个代码审查专家。检查代码的正确性和性能。

# groupchat.yaml
groupchat:
  agents:
    - assistant
    - code_reviewer
  speaker_selection_method: round_robin
  max_round: 10
```

### 3.4 状态持久化

AutoGen 的会话状态包含：

```go
type AutoGenState struct {
    ChatHistory    []Message     `json:"chat_history"`
    AgentStates    map[string]AgentState `json:"agent_states"`
    GroupChatState GroupChatState `json:"group_chat_state"`
}
```

Aetheris 会自动保存和恢复这些状态。

## 4. CrewAI Adapter

### 4.1 什么是 CrewAI？

CrewAI 是一个多 Agent 编排框架，强调 Agent 之间的协作：

```python
from crewai import Agent, Task, Crew

# 定义 Agent
researcher = Agent(
    role="Researcher",
    goal="Find information about AI",
    backstory="Expert researcher"
)

writer = Agent(
    role="Writer",
    goal="Write a report",
    backstory="Professional writer"
)

# 定义 Task
task1 = Task(description="Research AI", agent=researcher)
task2 = Task(description="Write report", agent=writer)

# 创建 Crew
crew = Crew(agents=[researcher, writer], tasks=[task1, task2])
result = crew.kickoff()
```

### 4.2 接入 Aetheris

```go
adapter := crewai.NewAdapter(&crewai.Config{
    CrewDefinitionPath: "./crew.yaml",
})

node, err := adapter.Compile()
```

### 4.3 Crew 定义

```yaml
# crew.yaml
crew:
  name: "Research and Write"
  agents:
    - id: researcher
      role: Researcher
      goal: Find information about {{topic}}
      backstory: Expert researcher with 10 years experience
      
    - id: writer
      role: Writer
      goal: Write a comprehensive report
      backstory: Professional technical writer
      
  tasks:
    - id: research_task
      description: Research {{topic}}
      agent: researcher
      expected_output: Detailed research findings
      
    - id: write_task
      description: Write report based on research
      agent: writer
      expected_output: Professional report document
      dependencies: [research_task]
```

## 5. 自定义 Adapter

### 5.1 为什么要自定义？

- 现有框架不满足需求
- 内部自研的 Agent 框架
- 特殊的执行语义

### 5.2 实现接口

```go
// 实现 NodeAdapter 接口
type MyAgentAdapter struct {
    config *Config
    agent  *MyAgent
}

func (a *MyAgentAdapter) Initialize(ctx context.Context, config AgentConfig) error {
    a.agent = NewMyAgent(config)
    return nil
}

func (a *MyAgentAdapter) Compile() (Node, error) {
    return &MyAgentNode{
        agent: a.agent,
    }, nil
}

func (a *MyAgentAdapter) GetInitialState() (map[string]interface{}, error) {
    return a.agent.GetInitialState()
}

// 实现 Node 接口
type MyAgentNode struct {
    agent *MyAgent
}

func (n *MyAgentNode) Execute(ctx context.Context, input Input) (Output, error) {
    result, err := n.agent.Run(ctx, input)
    return Output{
        State: result.State,
        Data:  result.Data,
    }, err
}

func (n *MyAgentNode) GetType() string    { return "my_agent" }
func (n *MyAgentNode) GetName() string    { return "my_agent_node" }
```

### 5.3 注册 Adapter

```go
// 注册到 Aetheris
runtime.RegisterAdapter("my_agent", &MyAgentAdapter{})
```

## 6. 迁移路径

### 6.1 渐进式迁移

```
Phase 1: 直接部署
         现有 Agent → Aetheris（不做代码修改）
         
Phase 2: 添加能力
         + Checkpoint（崩溃恢复）
         + Tool Ledger（At-Most-Once）
         
Phase 3: 增强功能
         + Human-in-loop（审批流）
         + 审计回放
```

### 6.2 Adapter 选择指南

| 场景 | 推荐 Adapter |
|------|--------------|
| 已有 LangGraph Agent | LangGraph Adapter |
| 已有 AutoGen 多 Agent 系统 | AutoGen Adapter |
| 已有 CrewAI Crew | CrewAI Adapter |
| 自研框架 | 自定义 Adapter |

## 7. 示例代码

### 7.1 完整示例：运行 LangGraph Agent

```go
package main

import (
    "context"
    "log"
    
    "rag-platform/adapters/langgraph"
    "rag-platform/runtime"
)

func main() {
    // 1. 初始化 Aetheris
    rt, err := runtime.New(runtime.Config{
        JobStoreURL: "postgres://localhost/aetheris",
        RedisURL:    "redis://localhost",
    })
    if err != nil {
        log.Fatal(err)
    }
    
    // 2. 创建 LangGraph Adapter
    adapter := langgraph.NewAdapter(&langgraph.Config{
        GraphPath: "./examples/langgraph-agent/graph.json",
    })
    
    // 3. 编译
    node, err := adapter.Compile()
    if err != nil {
        log.Fatal(err)
    }
    
    // 4. 注册
    rt.RegisterNode("langgraph_agent", node)
    
    // 5. 创建 Job
    job, err := rt.CreateJob(context.Background(), &runtime.JobRequest{
        AgentID: "langgraph_agent",
        Input: map[string]interface{}{
            "messages": []map[string]string{
                {"role": "user", "content": "帮我写一个排序算法"},
            },
        },
    })
    if err != nil {
        log.Fatal(err)
    }
    
    log.Printf("Job created: %s", job.ID)
    
    // 6. 等待完成
    result, err := rt.WaitForCompletion(context.Background(), job.ID)
    if err != nil {
        log.Fatal(err)
    }
    
    log.Printf("Result: %v", result)
}
```

## 8. 小结

Aetheris 的 **Adapter 架构** 让存量 Agent 可以无缝迁移：

1. **LangGraph Adapter** — 一键部署现有 LangGraph 图
2. **AutoGen Adapter** — 支持 Microsoft AutoGen 多 Agent
3. **CrewAI Adapter** — 支持 CrewAI Crew 编排
4. **自定义 Adapter** — 灵活适配任意框架

**不需要重写代码**，就能获得：
- 持久化与崩溃恢复
- At-Most-Once 工具调用
- Human-in-the-Loop 支持
- 完整的审计与回放

---

*下篇预告：审计与调试——事件流回放与证据链*
