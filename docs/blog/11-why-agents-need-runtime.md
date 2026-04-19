# 为什么 AI Agent 需要自己的 Runtime——对比 Temporal、Aetheris 和 LangChain

> 2026-04-19 | Aetheris v2.5.3

## TL;DR

如果你在用 LangChain/LangGraph 写 Agent，那是在写**业务逻辑**。但生产级别的 Agent 还需要**执行可靠性**——崩溃恢复、幂等保证、审计追溯。Temporal 是工作流引擎，Aetheris 是为 Agent 场景重新设计的执行 Runtime。

---

## 背景：为什么 AI Agent 不同于普通微服务

一个 AI Agent 微服务，需要：
- **执行过程可追溯**：LLM 每一步决策、每个 Tool 调用，都要记录
- **Tool 幂等性**：同一个退款请求，Worker 崩溃重启后不能执行两次
- **Human-in-the-Loop**：Agent 在关键节点等待人工审批，不占用 Worker 资源
- **多 Worker 并行**：多个 Worker 同时处理不同 Job，不能相互干扰

这些需求，微服务框架不解决，LangChain 也不解决。它们解决的是"怎么写 Agent"，而不是"怎么可靠地跑 Agent"。

---

## Temporal：工作流引擎的正确打开方式

Temporal 是一个强大的工作流编排引擎，核心解决**长时间运行工作流的可靠性**。

**优点：**
- 活动（Activity）级别的重试和超时
- Workflow 状态持久化，崩溃可恢复
- 丰富的查询和信号机制
- 多语言 SDK 成熟

**局限性：**
- Workflow 代码有严格约束（幂等、无副作用、不依赖外部时间）
- LLM 调用作为 Activity 时，每次重试都会重新调用 LLM（Token 浪费 + 结果不稳定）
- 没有内置的"AI 执行轨迹"概念，LLM 决策历史需要自己存储
- 信号机制适合工作流控制，但不适合 LLM 的流式输出处理

**Temporal 的适用场景：**
- 订单处理、支付流程
- 保险理赔、工作流审批
- 数据 ETL 管道

---

## Aetheris：专为 Agent Runtime 设计

Aetheris 从一开始就是为 AI Agent 设计的执行层，解决的是 Temporal 没有覆盖的 Agent 特有需求。

### 核心区别 1：LLM 调用的 Effect Store

Temporal 里，Activity 重试 = 重新执行，包括重新调用 LLM。

Aetheris 有 Effect Store，专门存储 LLM 调用结果：

```
Step 1: LLM Decide  →  Effect Store 记录: {input: "...", output: "..."}
Step 2: Worker 崩溃
Step 3: 新 Worker 接管
Step 4: Replay 时，LLM 调用被拦截，直接从 Effect Store 返回缓存结果
```

这意味着：
- **崩溃恢复后 LLM 不会重新调用** — 节省 Token，保证确定性
- **结果一致性** — 同一个 Job 无论重试多少次，LLM 输出不变

### 核心区别 2：Tool 幂等性 + Ledger

Aetheris 的 Invocation Ledger 记录每个 Tool 的执行状态：

```
send_refund(idempotency_key="aetheris:job-123:send_refund:1")  // 第一次执行
→ Ledger 记录: EXECUTED
→ Effect Store 记录: {idempotency_key, result}

Worker 崩溃
新 Worker 接管，读取 Ledger
→ 发现 send_refund 已执行，直接跳到下一步
```

退款 API 不会被调用两次。这是 Temporal Activity 做不到的。

### 核心区别 3：Human-in-the-Loop 的 Parked 状态

Agent 在关键节点等待人工审批（或者外部 webhook）：

```
Job 执行到 wait_approval 节点
→ Status = Parked（不占用 Worker）
→ 人工审批后 Signal 唤醒
→ 继续执行 send_refund
```

Temporal 的信号机制可以做到类似的事，但：
- Temporal 信号是给 Workflow 发消息，LLM 决策不能被打断
- Aetheris 的 Wait 节点可以暂停在 LLM 决策点，审批后继续从该节点执行

### 核心区别 4：Agent 原生的 Trace 和 Evidence

Aetheris 的执行记录专为 AI Agent 设计：

```json
{
  "steps": [
    {
      "node_id": "llm_decide",
      "evidence": {
        "llm_invocation_id": "aetheris:job-123:llm:1",
        "model": "gpt-4o-2024-11-20",
        "temperature": 0.7,
        "input_tokens": 1234,
        "output_tokens": 567
      }
    },
    {
      "node_id": "send_refund",
      "evidence": {
        "tool_invocation_ids": ["aetheris:job-123:send_refund:1"],
        "idempotency_key": "aetheris:job-123:send_refund:1",
        "external_ref": "stripe_charge_xxx"
      }
    }
  ]
}
```

每个 LLM 调用、每个 Tool 调用，都可以用 Evidence 证明。

---

## 对比表

| 维度 | Temporal | Aetheris |
|------|----------|----------|
| 定位 | 工作流引擎 | Agent 执行 Runtime |
| LLM 重试 | 重新调用（浪费 Token） | Effect Store 拦截（零重试开销） |
| Tool 幂等 | Activity 实现（业务负责） | Ledger + Idempotency Key（内置） |
| Human-in-the-Loop | Signal 机制 | Wait Node + Parked 状态 |
| 执行 Trace | 通用工作流历史 | AI 原生 Trace + Evidence |
| 适用场景 | 订单/支付/ETL | AI Agent + RAG + 对话系统 |
| Agent 框架集成 | 需自己包装 | MCP 协议原生支持 |

---

## 什么时候选什么

**选 Temporal 如果：**
- 核心是业务流程，LLM 只是其中一步
- 已有 Temporal 基础设施
- 需要强一致的工作流编排

**选 Aetheris 如果：**
- 核心是 AI Agent，需要可靠地跑 LLM
- 需要 LLM 决策的确定性（不重复调用）
- 需要 AI 决策的审计追溯
- 需要 MCP 协议的工具生态

---

## 结论

Temporal 解决的是"长时间运行工作流的可靠性"。

Aetheris 解决的是"AI Agent 执行的可靠性"——LLM 不重复调用、Tool 不重复执行、决策过程完整可审计。

两者不是非此即彼。对于需要强工作流编排 + AI Agent 的系统，可以：
- 用 Temporal 做顶层业务流程编排
- 用 Aetheris 作为 Temporal Activity 中的"AI Agent 执行单元"

这就是 Aetheris v2.5.3 的设计思路：作为 MCP Server 被宿主框架调用，而不是替代所有工作流引擎。

---

**相关链接：**
- [Aetheris GitHub](https://github.com/Colin4k1024/Aetheris)
- [v2.5.3 Release Notes](https://github.com/Colin4k1024/Aetheris/releases/tag/v2.5.3)
- [MCP Gateway 集成指南](../mcp/integration.md)
- [At-Most-Once 执行原理](../blog/06-at-most-once-ledger.md)
