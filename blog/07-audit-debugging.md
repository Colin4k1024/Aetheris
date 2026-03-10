# 审计与调试：事件流回放与证据链

> 当 Agent 出了问题，你能不能回答「为什么」？

## 0. 一个审计需求

银行使用 Agent 自动审批贷款：

```
Agent 执行记录：
1. Step 1: 调用 RAG 获取用户信用记录
2. Step 2: 调用风控模型评估风险
3. Step 3: 调用贷款 API 批准贷款 $50,000
4. Step 4: 发送通知

监管机构要求：
- "为什么 Agent 批准了这笔贷款？"
- "Agent 使用了什么数据做出决策？"
- "整个决策过程可以重现吗？"
```

**普通 Agent 做不到。但 Aetheris 可以。**

## 1. Aetheris 的审计架构

### 1.1 三层审计体系

```
┌─────────────────────────────────────────────────────────────┐
│                     审计层                                   │
├─────────────────────────────────────────────────────────────┤
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐           │
│  │  决策时间线  │  │  证据图      │  │  执行回放   │           │
│  │  (Timeline) │  │(Evidence Grph)│  │  (Replay)  │           │
│  └─────────────┘  └─────────────┘  └─────────────┘           │
└─────────────────────────────────────────────────────────────┘
                              │
┌─────────────────────────────▼───────────────────────────────┐
│                     事件流层                                  │
│         Every event is recorded. Everything is traceable.   │
└─────────────────────────────────────────────────────────────┘
```

### 1.2 核心组件

| 组件 | 职责 |
|------|------|
| **事件流** | 记录所有执行事件（原始数据） |
| **决策时间线** | 可视化展示执行过程 |
| **证据图** | 记录每步决策的证据（输入、输出、依据） |
| **Replay** | 完整重现执行过程 |

## 2. 事件流：原始记录

### 2.1 事件类型

```go
// 生命周期事件
JobCreated, JobScheduled, JobParked, JobResumed, JobCompleted, JobFailed

// 执行事件
StepStarted, StepCompleted, StepFailed, StepRetrying

// LLM 事件
LLMRequest, LLMResponse

// 工具事件
ToolInvocated, ToolCompleted, ToolFailed

// 人工介入事件
SignalReceived, MessageReceived, HumanIntervention

// 证据事件
EvidenceRecorded, ReasoningSnapshot
```

### 2.2 完整事件示例

```json
{
  "event_id": "evt_1234567890",
  "job_id": "job_abc123",
  "sequence_id": 15,
  "event_type": "StepCompleted",
  "timestamp": 1709999999,
  "payload": {
    "step_id": "step_approve_loan",
    "step_type": "llm",
    "input": {
      "goal": "评估贷款申请 #12345",
      "state_before": {
        "credit_score": 750,
        "debt_ratio": 0.3,
        "loan_amount": 50000
      }
    },
    "output": {
      "decision": "approve",
      "reason": "信用评分良好，负债率适中",
      "risk_level": "low"
    },
    "duration_ms": 1500
  },
  "evidence": {
    "rag_doc_ids": ["doc_001", "doc_002", "doc_003"],
    "model_version": "gpt-4-2024-02",
    "temperature": 0.2,
    "token_usage": {
      "prompt": 1200,
      "completion": 300
    }
  }
}
```

## 3. 证据图（Evidence Graph）

### 3.1 什么是证据图？

**证据图 = 决策的完整推理链**

```
┌─────────────────────────────────────────────────────────────┐
│                    证据图示例：贷款审批                        │
└─────────────────────────────────────────────────────────────┘

                    ┌──────────────┐
                    │  贷款申请    │
                    │  #12345      │
                    └──────┬───────┘
                           │
           ┌───────────────┼───────────────┐
           │               │               │
           ▼               ▼               ▼
    ┌────────────┐  ┌────────────┐  ┌────────────┐
    │  信用记录   │  │  收入证明  │  │  资产证明   │
    │  (RAG)     │  │  (API)     │  │  (API)     │
    └─────┬──────┘  └─────┬──────┘  └─────┬──────┘
          │               │               │
          └───────────────┼───────────────┘
                          │
                          ▼
               ┌─────────────────────┐
               │  Agent 决策分析      │
               │  - 信用分: 750      │
               │  - 负债率: 30%      │
               │  - 风险等级: 低     │
               └──────────┬──────────┘
                          │
                          ▼
               ┌─────────────────────┐
               │  批准贷款 $50,000   │
               │  [Step #3]          │
               └─────────────────────┘
```

### 3.2 证据类型

```go
type Evidence struct {
    // RAG 证据
    RAGDocs []RAGDoc `json:"rag_docs"`
    
    // LLM 证据
    LLM *LLMEvidence `json:"llm"`
    
    // 工具证据
    ToolCalls []ToolCall `json:"tool_calls"`
    
    // 人工输入
    HumanInput *HumanInput `json:"human_input"`
    
    // 推理链
    ReasoningChain []string `json:"reasoning_chain"`
}

type RAGDoc struct {
    DocID   string `json:"doc_id"`
    Content string `json:"content"`
    Score   float64 `json:"score"`
    Source  string `json:"source"`
}

type LLMEvidence struct {
    Model     string `json:"model"`
    Prompt    string `json:"prompt"`
    Response  string `json:"response"`
    Tokens    int    `json:"tokens"`
    LatencyMs int    `json:"latency_ms"`
}
```

### 3.3 证据记录时机

```
执行流程：

Step 1: Agent 分析贷款申请
        │
        ├─ 调用 RAG → 记录 RAG 证据
        ├─ 调用 LLM  → 记录 LLM 证据
        │
        ▼
Step 2: Agent 做出决策
        │
        ├─ 生成推理快照（Reasoning Snapshot）
        │  {
        │    "goal": "评估风险",
        │    "state_before": {...},
        │    "state_after": {...},
        │    "evidence": [...]
        │  }
        │
        ▼
Step 3: Agent 调用工具（批准贷款）
        │
        ├─ 记录工具调用证据
        └─ 记录外部系统交互
```

## 4. 决策时间线（Timeline）

### 4.1 Timeline API

```bash
# 获取 Job 的时间线
curl http://localhost:8080/api/jobs/job_abc123/timeline

# 响应
{
  "job_id": "job_abc123",
  "total_duration": "45.2s",
  "events": [
    {
      "time": "14:30:00",
      "type": "job_created",
      "description": "任务创建"
    },
    {
      "time": "14:30:01", 
      "type": "step_started",
      "step": "analyze_application",
      "description": "开始分析申请"
    },
    {
      "time": "14:30:05",
      "type": "llm_request",
      "step": "analyze_application", 
      "model": "gpt-4",
      "description": "调用 LLM 分析"
    },
    {
      "time": "14:30:08",
      "type": "step_completed",
      "step": "analyze_application",
      "duration": "7s",
      "description": "分析完成"
    },
    ...
  ]
}
```

### 4.2 可视化展示

```
Timeline: 贷款审批 #12345
══════════════════════════════════════════════════════════════

14:30:00  ━━━━ JobCreated
           任务创建，贷款申请 #12345

14:30:01  ━━━━ StepStarted: analyze_application
           开始分析申请材料

14:30:05  ━━━━ LLMRequest (gpt-4)
           │ 分析用户信用风险
           │ 输入: credit_score=750, debt_ratio=0.3

14:30:08  ━━━━ LLMResponse
           │ 响应: 建议批准，风险等级低
           │ 推理: 信用评分良好，负债率适中

14:30:10  ━━━━ StepCompleted: evaluate_risk
           决策: approve
           证据: RAG_docs[3], LLM_call[1]

14:30:11  ━━━━ ToolInvocated: approve_loan
           调用贷款系统 API

14:30:15  ━━━━ ToolCompleted: approve_loan
           结果: 贷款批准，ID=loan_xyz

14:30:16  ━━━━ JobCompleted
           任务完成，总耗时 16s
```

## 5. Replay：完整回放

### 5.1 什么是 Replay？

**Replay = 从头重放整个执行过程**

用于：
- **调试**：复现问题
- **审计**：验证决策
- **演示**：展示执行过程

### 5.2 Replay 命令

```bash
# 回放 Job 执行
aetheris replay job_abc123

# 带详细输出
aetheris replay job_abc123 --verbose

# 从指定步骤开始
aetheris replay job_abc123 --from-step step_3

# 导出事件流
aetheris replay job_abc123 --export events.jsonl
```

### 5.3 Replay 输出示例

```
Replay: job_abc123
══════════════════════════════════════════════════════════════

[1/10] JobCreated
  Input: { request_id: "12345", amount: 50000 }

[2/10] StepStarted: analyze_credit
  ├─ Loading checkpoint...
  └─ Executing...
  
[3/10] LLMRequest
  ├─ Model: gpt-4
  ├─ System: 你是一个贷款风控专家
  ├─ User: 分析用户信用...
  └─ [Cached - Using Ledger result]

[4/10] LLMResponse
  ├─ Decision: approve
  ├─ Confidence: 0.92
  └─ Reasoning: 信用评分良好

[5/10] StepCompleted: analyze_credit
  Output: { decision: "approve", risk_level: "low" }
  
  ...

[10/10] JobCompleted
  Final State: { loan_id: "loan_xyz", status: "approved" }

══════════════════════════════════════════════════════════════
Replay complete. 10 events processed.
Output matches original execution: ✓
```

### 5.4 Safe Replay

Replay 时**不会真的调用外部 API**：

```go
func (r *Runner) Replay(jobID string) error {
    events := jobstore.GetEvents(jobID)
    
    for _, event := range events {
        switch event.Type {
        case "ToolInvocated":
            // 检查 Tool Ledger
            cached, _ := toolLedger.Get(event.IdempotencyKey)
            if cached != nil {
                // 使用缓存结果，不调用外部
                r.injectResult(cached.Output)
                fmt.Printf("[Replay] Tool %s: using cached result\n", event.ToolName)
                continue
            }
            
            // 没有缓存：需要验证外部状态
            effect, _ := effectStore.Get(event.EffectID)
            if verified := r.verifyExternalState(effect); verified {
                r.injectResult(effect.Output)
                fmt.Printf("[Replay] Tool %s: verified externally\n", event.ToolName)
                continue
            }
            
            // 无法验证
            return fmt.Errorf("cannot replay tool %s: no cached result", event.ToolName)
            
        case "LLMRequest":
            // LLM 可以重新调用（幂等）
            // 或者使用缓存
            r.replayLLM(event)
        }
    }
    
    return nil
}
```

## 6. 审计合规

### 6.1 监管要求

| 要求 | Aetheris 能力 |
|------|---------------|
| 记录决策时间 | 事件时间戳 |
| 记录使用了哪些数据 | 证据图（RAG docs） |
| 记录 LLM 输出 | LLM 事件 |
| 记录外部调用 | Tool 事件 |
| 可重现决策 | Replay |
| 防篡改 | 事件哈希链 |

### 6.2 执行证明链（Execution Proof Chain）

```go
// 事件哈希链：防篡改
type EventChain struct {
    Events []HashedEvent `json:"events"`
}

type HashedEvent struct {
    EventID     string `json:"event_id"`
    Payload     []byte `json:"payload"`
    PreviousHash string `json:"previous_hash"`
    EventHash   string `json:"event_hash"`  // SHA256(payload + previousHash)
}

// 验证链完整性
func (ec *EventChain) Verify() bool {
    var previousHash string
    
    for _, event := range ec.Events {
        expectedHash := SHA256(event.Payload + previousHash)
        if expectedHash != event.EventHash {
            return false  // 篡改检测
        }
        previousHash = event.EventHash
    }
    
    return true
}
```

### 6.3 导出审计报告

```bash
# 导出审计报告
aetheris audit export job_abc123 \
  --format pdf \
  --output audit_report.pdf

# 导出原始数据
aetheris audit export job_abc123 \
  --format jsonl \
  --output events.jsonl
```

## 7. 可观测性集成

### 7.1 OpenTelemetry

```go
// Aetheris 自动导出 Trace
tracer := otel.Tracer("aetheris")

func (r *Runner) executeStep(step *Step) {
    ctx, span := tracer.Start(r.ctx, "step."+step.Name)
    defer span.End()
    
    // 执行...
    
    // 添加属性
    span.SetAttributes(
        attribute.String("job.id", r.job.ID),
        attribute.String("step.id", step.ID),
        attribute.Int("llm.tokens", llmUsage.Total),
    )
}
```

### 7.2 可观测性面板

```
┌─────────────────────────────────────────────────────────────┐
│                   Aetheris Dashboard                        │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│  Job Metrics           Execution Timeline    Evidence      │
│  ───────────           ─────────────────    ────────      │
│  Total: 1,234          [████████░░░░░] 80%    RAG: ✓        │
│  Running: 12           Current: step_5      LLM: ✓        │
│  Parked: 45            ETA: 30s             Tool: ✓         │
│  Failed: 3                                    Human: -      │
│                                                             │
└─────────────────────────────────────────────────────────────┘
```

## 8. 小结

Aetheris 的审计能力让 Agent 真正可用于生产：

1. **事件流** — 完整记录所有执行步骤
2. **证据图** — 记录每步决策的输入、输出、依据
3. **Timeline** — 可视化展示执行过程
4. **Replay** — 完整重现执行（安全地）
5. **执行证明链** — 防篡改、可验证

有了这套系统，**监管审计不再困难，调试问题不再抓瞎**。

---

*下篇预告：多 Worker 部署与调度正确性*
