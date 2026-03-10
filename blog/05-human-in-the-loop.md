# Human-in-the-Loop 实战：审批流与长时间等待

> 当 AI 需要等待人类批准时，Runtime 该如何设计？

## 0. 真实业务场景

### 场景 1：法务审批合同

```
Day 1, 10:00
├─ Agent: 生成采购合同
├─ Agent: 调用法务 API 初步审核
└─ Agent: 提交给法务审批
        [暂停，等待审批]

Day 3, 15:00
├─ 法务: 审批通过
└─ Agent: 继续执行 → 发送合同给供应商
```

### 场景 2：财务审批付款

```
Agent: 处理退款申请
├─ 读取用户申请
├─ 验证订单信息
├─ 计算退款金额
└─ [如果金额 > $1000]
    ├─ 提交财务审批 ← 暂停
    └─ 等待财务确认
```

### 场景 3：客服升级

```
Agent: 处理客户投诉
├─ 分析投诉内容
├─ 尝试自动解决
├─ [如果无法自动解决]
    └─ 转人工处理 ← 暂停
        [人工客服介入]
        [人工点击"继续"]
```

这些场景的共同点：**Agent 需要「等待」某个外部事件**，可能是几分钟、几小时、甚至几天。

## 1. 传统方案的困境

### 1.1 轮询方案

```python
# ❌ 低效的方案
while True:
    result = check_approval_status(approval_id)
    if result.status == "approved":
        break
    time.sleep(60)  # 每分钟检查一次
```

**问题**：
- 浪费资源：Worker 一直被占用
- 延迟高：最长需要等待一个轮询周期
- 不可扩展：1000 个待审批，1000 个 Worker 都不够

### 1.2 超时放弃

```python
# ❌ 不可靠的方案
try:
    result = wait_for_approval(timeout=3600)  # 1小时超时
except TimeoutError:
    # 超时了，怎么办？
    # 放弃？重试？通知管理员？
```

**问题**：
- 法务审批 3 天很正常，1 小时不够
- 超时后的处理逻辑复杂

## 2. Aetheris 的方案：Wait 节点 + Signal

### 2.1 核心概念

**Wait 节点**：Agent 执行到某一步时，主动「暂停」

**Signal（信号）**：外部事件触发，继续执行

```
┌─────────────────────────────────────────────────────────────┐
│                    Wait 节点执行流程                         │
└─────────────────────────────────────────────────────────────┘

Step 1: 分析申请
    │
    ▼
Step 2: 生成合同
    │
    ▼
Step 3: Wait (correlation_key="approval:12345")
    │
    ├── 状态变为 "Parked"（暂停）
    ├── 释放 Worker（可处理其他任务）
    └── 等待 Signal
              │
              │ [3 天后...]
              │
              ▼
Signal 到达: correlation_key="approval:12345", payload={approved: true}
    │
    ▼
Step 4: 发送合同
```

### 2.2 Parked 状态

当 Agent 执行到 Wait 节点时：

```
Job 状态机：

Created → Scheduled → Running → Parked → Running → Completed
                              ↓
                            Failed
                              ↓
                            Parked
```

**Parked 的特点**：
- 不占用任何 Worker 资源
- 状态保存在 JobStore 中
- 可以持续「等待」任意时间

### 2.3 Signal 的投递

```go
// 外部系统调用 API 发送 Signal
func (api *API) SendSignal(jobID string, signal *Signal) error {
    // 1. 验证 Job 确实在 Parked 状态
    job := jobStore.Get(jobID)
    if job.Status != "Parked" {
        return ErrJobNotParked
    }
    
    // 2. 写入 Signal 事件
    jobstore.AppendEvent(&Event{
        Type:    "SignalReceived",
        JobID:   jobID,
        Payload: signal,
    })
    
    // 3. 更新 Job 状态
    jobStore.UpdateStatus(jobID, "Running")
    
    // 4. 重新入队等待调度
    scheduler.Enqueue(jobID)
    
    return nil
}
```

## 3. 完整示例：退款审批 Agent

### 3.1 Agent 定义

```go
// 定义 Agent
agent := &Agent{
    Name: "RefundApprovalAgent",
    Steps: []Step{
        {
            Name: "analyze_request",
            Node: &LLMNode{
                Prompt: "分析退款申请 {{.request_id}}",
                Output: "analysis",
            },
        },
        {
            Name: "check_amount",
            Node: &DecisionNode{
                Condition: "{{.analysis.amount}} > 1000",
                Then: "require_approval",
                Else: "auto_approve",
            },
        },
        {
            Name: "require_approval",
            Node: &WaitNode{
                CorrelationKey: "refund:approval:{{.request_id}}",
                Timeout:        7 * 24 * time.Hour, // 7 天超时
            },
        },
        {
            Name: "auto_approve",
            Node: &ToolNode{
                Tool: "auto_refund",
                Input: map[string]interface{}{
                    "request_id": "{{.request_id}}",
                    "amount":     "{{.analysis.amount}}",
                },
            },
        },
        {
            Name: "execute_refund",
            Node: &ToolNode{
                Tool: "stripe.refund",
                Input: map[string]interface{}{
                    "charge_id": "{{.analysis.charge_id}}",
                },
            },
        },
        {
            Name: "send_notification",
            Node: &ToolNode{
                Tool: "email.send",
                Input: map[string]interface{}{
                    "to":      "{{.analysis.customer_email}}",
                    "template": "refund_confirmed",
                },
            },
        },
    },
}
```

### 3.2 执行流程

```
时间线：

T1: Job #123 创建
    │
T2: Step "analyze_request" 执行
    │
T3: Step "check_amount" 执行
    │ 条件: $1500 > $1000 → 需要审批
    │
T4: Step "require_approval" 执行
    │ Wait(correlation_key="refund:approval:12345")
    │ Job 状态: Running → Parked
    │ Worker 释放
    │
T5-T6: [等待 3 天]
    │
T7: 财务审批通过，调用 API
    POST /api/jobs/123/signal
    {
        "correlation_key": "refund:approval:12345",
        "payload": {"approved": true, "approver": "张三"}
    }
    │
T8: Job 状态: Parked → Running
    │
T9: Step "execute_refund" 执行
    │
T10: Step "send_notification" 执行
    │
T11: Job Completed
```

### 3.3 Signal API

```bash
# 发送 Signal 批准
curl -X POST http://localhost:8080/api/jobs/123/signal \
  -H "Content-Type: application/json" \
  -d '{
    "correlation_key": "refund:approval:12345",
    "payload": {
      "approved": true,
      "approver": "张三",
      "comment": "同意退款"
    }
  }'

# 发送 Signal 拒绝
curl -X POST http://localhost:8080/api/jobs/123/signal \
  -H "Content-Type: application/json" \
  -d '{
    "correlation_key": "refund:approval:12345",
    "payload": {
      "approved": false,
      "approver": "张三",
      "comment": "不符合退款政策"
    }
  }'
```

## 4. Message 机制

除了 Signal，还有 **Message（消息）** 机制：

### 4.1 Signal vs Message

| 特性 | Signal | Message |
|------|--------|---------|
| 触发方式 | 直接唤醒指定 Job | 发送到会话/队列 |
| 关联方式 | correlation_key | session_id / queue |
| 用途 | 审批、回调 | 多轮对话、外部输入 |

### 4.2 Message 用法

```go
// 用户发送消息到 Session
curl -X POST http://localhost:8080/api/sessions/session_123/messages \
  -H "Content-Type: application/json" \
  -d '{
    "content": "我同意这个方案，请继续"
  }'
```

```go
// Agent 内部接收消息
node := &ReceiveNode{
    Queue: "user_input:{{.session_id}}",
}
```

## 5. 超时处理

### 5.1 Wait 节点超时

```go
node := &WaitNode{
    CorrelationKey: "approval:12345",
    Timeout:        7 * 24 * time.Hour,
    OnTimeout:      "notify_admin",  // 超时后执行的步骤
}
```

### 5.2 超时事件流

```
T1: Wait(correlation_key="approval:123", timeout=7d)
    │
T7, 23:59
    │
T8: 超时前 1 分钟
    │ 生成 TimeoutWarning 事件
    │
T8: 等待超时
    │ 触发 OnTimeout 步骤
    │
    Step: "notify_admin"
    ├─ 发送通知给管理员
    │  "退款申请 #12345 审批超时，请处理"
    └─ 进入新的 Wait 或标记失败
```

## 6. 外部系统集成

### 6.1 Webhook 方式

```
┌─────────────┐     Signal      ┌─────────────┐
│  外部系统    │ ─────────────▶ │  Aetheris   │
│  (CRM/审批流) │                │   Runtime   │
└─────────────┘                 └─────────────┘
```

```javascript
// 外部审批系统配置 Webhook
{
  url: "https://aetheris.internal/api/jobs/{job_id}/signal",
  trigger: "approval_completed",
  payload: {
    correlation_key: "approval:{{.approval_id}}",
    payload: "{{.result}}"
  }
}
```

### 6.2 双向集成

```
┌─────────────────────────────────────────────────────────────┐
│                      完整流程                                │
└─────────────────────────────────────────────────────────────┘

  Aetheris                              外部审批系统
     │                                        │
     │──── 提交审批 (Wait) ──────────────────▶│
     │                                        │
     │◀──── Webhook 回调 ─────────────────────│
     │       (Signal)                         │
     │                                        │
     │──── 执行后续步骤 ──────────────────────▶│
     │                                        │
```

## 7. 监控与可观测性

### 7.1 Parked Job 监控

```bash
# 查看所有 Parked 的 Job
aetheris jobs list --status parked

# 查看特定 Wait 的超时时间
aetheris jobs inspect job_123 --show-wait-state
```

### 7.2 告警规则

```yaml
# 告警：审批超时
- name: approval_timeout
  condition: job.status == "parked" && 
             time_since_parked > 24h
  severity: warning
  message: "Job {{.job_id}} 已等待审批超过 24 小时"
```

## 8. 小结

Human-in-the-Loop 是 **生产级 Agent** 的必备能力：

1. **Wait 节点** — 主动暂停，不占资源
2. **Signal 机制** — 外部事件触发继续执行
3. **Parked 状态** — 状态持久化，支持任意时长等待
4. **超时处理** — 防止无限等待
5. **Webhook 集成** — 与外部审批系统无缝对接

有了这套机制，Agent 可以优雅地处理**审批流、多轮对话、外部回调**等场景——**等待时不浪费资源，需要时立即响应**。

---

*下篇预告：集成 LangGraph/AutoGen——在 Aetheris 上运行已有 Agent*
