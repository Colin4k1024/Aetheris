# At-Most-Once 语义：如何保证工具调用不重复

> 这是 Agent Runtime 最核心的能力——如果这都保证不了，就别谈生产级可靠性。

## 0. 那个噩梦般的场景

```
Agent 执行退款操作：
1. 调用 Stripe API 退款 $100
2. Stripe 返回成功
3. Agent 准备记录日志...
4. [崩溃]

重启后：
A. 全部重头来过 → Stripe API 被调用第二次 → 用户被扣 $200
B. 不重试 → 不知道第 1 步是否真的成功 → 数据不一致
```

这是所有 Agent 开发者都必须面对的**副作用问题**。

## 1. 三种执行语义

在分布式系统中，任务执行有三种语义：

| 语义 | 含义 | 副作用 | 适用场景 |
|------|------|--------|----------|
| **At-Least-Once** | 至少执行一次，可能重复 | 可能有重复 | 日志、监控 |
| **At-Most-Once** | 至多执行一次，不重复 | 无重复 | 支付、发邮件 |
| **Exactly-Once** | 恰好执行一次 | 无重复 | 理想状态（很难实现） |

**Agent 的工具调用必须保证 At-Most-Once**——没有人希望自己的钱被扣两次。

## 2. Aetheris 的 At-Most-Once 方案

### 2.1 核心组件：Tool Ledger

```
┌─────────────────────────────────────────────────────────────┐
│                     Tool Ledger                             │
│  ┌─────────────────────────────────────────────────────┐   │
│  │ idempotency_key          │ result                  │   │
│  ├─────────────────────────────────────────────────────┤   │
│  │ job:123:step:2:attempt:1 │ {status: success, ...}   │   │
│  │ job:123:step:3:attempt:1 │ {status: success, ...}   │   │
│  └─────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────┘
```

**工作原理**：
1. 每次工具调用生成唯一的 `idempotency_key`
2. 执行前检查 Ledger：key 是否已存在？
3. 如果存在 → 直接返回上次结果（不调用工具）
4. 如果不存在 → 调用工具，保存结果到 Ledger

### 2.2 Idempotency Key 的设计

```go
// 生成规则
IdempotencyKey = {job_id}:{step_id}:{attempt}:{tool_name}:{input_hash}

// 示例
// job:123:step:2:attempt:1:stripe.refund:a1b2c3d4
```

这个 key 足够唯一，包含了：
- Job ID（哪个任务）
- Step ID（哪一步）
- Attempt（重试次数）
- Tool Name（什么工具）
- Input Hash（输入哈希）

### 2.3 执行流程图

```
┌─────────────────────────────────────────────────────────────┐
│                    Runner 执行 Tool                         │
└────────────────────────────┬────────────────────────────────┘
                             │
                             ▼
              ┌──────────────────────────────┐
              │ 1. 生成 Idempotency Key       │
              │ key = "job:123:step:2:..."    │
              └──────────────┬───────────────┘
                             │
                             ▼
              ┌──────────────────────────────┐
              │ 2. 查询 Tool Ledger           │
              │ SELECT * FROM tool_ledger    │
              │ WHERE idempotency_key = ?    │
              └──────────────┬───────────────┘
                             │
                    ┌────────┴────────┐
                    │                 │
                已存在             不存在
                    │                 │
                    ▼                 ▼
        ┌───────────────────┐  ┌───────────────────┐
        │ 3a. 返回缓存结果   │  │ 3b. 调用工具 API  │
        │ (不调用外部)       │  │    (Stripe 等)    │
        └───────────────────┘  └──────────────┬──────┘
                                                │
                                                ▼
                                     ┌───────────────────┐
                                     │ 4. 写入 Ledger    │
                                     │ (result + status) │
                                     └───────────────────┘
```

## 3. 完整的调用协议

### 3.1 Two-Phase Commit

为了处理「调用成功但写入失败」的边缘情况，Aetheris 使用**两阶段提交**：

```go
// Phase 1: 预提交
func CallToolWithTwopc(ctx context.Context, tool Tool, input Input) (Output, error) {
    key := GenerateIdempotencyKey(tool, input)
    
    // 1. 尝试获取分布式锁
    lock := redis.Lock("tool:" + key)
    if !lock.Acquire(ctx, 10*time.Second) {
        return nil, ErrConcurrentExecution
    }
    defer lock.Release()
    
    // 2. 检查 Ledger
    cached, err := ledger.Get(key)
    if err == nil && cached != nil {
        return cached.Output, nil  // 已执行过
    }
    
    // 3. 调用工具
    output, err := tool.Execute(ctx, input)
    if err != nil {
        // 工具调用失败，直接返回错误
        return nil, err
    }
    
    // 4. 预写入 Ledger（状态：pending）
    ledger.Put(&LedgerEntry{
        Key:      key,
        Status:   "pending",
        Output:   output,
    })
    
    // 5. 确认提交（状态：committed）
    ledger.UpdateStatus(key, "committed")
    
    return output, nil
}
```

### 3.2 边缘情况处理

| 场景 | 处理方式 |
|------|----------|
| 调用前崩溃 | 不会写入 Ledger，下次执行会重新调用 |
| 调用成功，Ledger 写入前崩溃 | 工具已执行，但 Ledger 无记录 → **风险** |
| 调用成功，Ledger pending 状态崩溃 | 下次执行检测到 pending，需要人工/自动确认 |
| 调用失败，Ledger 无记录 | 正常重试 |

### 3.3 Crash Recovery 场景

```
时间线：

T1: Worker A 调用 Stripe API 退款 $100
    │
    │ Stripe 返回成功
    │
T2: Worker A 准备写入 Ledger...
    │
    │ [崩溃！]
    │
T3: Worker B 接收任务（Scheduler 重新分配）
    │
    │ 恢复 Checkpoint（Step 2 执行中）
    │
T4: Worker B 尝试重新执行 Step 2
    │
    │ 检查 Ledger：
    │ - key = "job:123:step:2:..."
    │ - 记录不存在
    │
T5: [关键决策！]
    │
    │ 选项 A：重新调用 Stripe
    │    ❌ 问题：用户被扣 $200！
    │
    │ 选项 B：向 Stripe 查询该退款是否已存在
    │    ✅ 正确：查询后确认已退款，直接使用之前的结果
```

### 3.4 Effect Store + 外部验证

Aetheris 的解决方案是 **Effect Store（副作用存储）** + **外部验证**：

```go
type Effect struct {
    EffectID      string    `json:"effect_id"`
    JobID         string    `json:"job_id"`
    StepID        string    `json:"step_id"`
    ToolName      string    `json:"tool_name"`
    Input         any       `json:"input"`
    ExternalID    string    `json:"external_id,omitempty"` // 外部系统的 ID
    Status        string    `json:"status"`               // pending/committed/verified
}

// 恢复时的验证逻辑
func (r *Runner) verifyAndRecover(ctx context.Context, effect *Effect) (any, error) {
    // 1. 检查外部系统
    switch effect.ToolName {
    case "stripe.refund":
        // 查询 Stripe：该 refund_id 是否已存在？
        refund, err := stripe.GetRefund(effect.ExternalID)
        if err == nil && refund.Status == "succeeded" {
            // 外部已确认，重新构建结果
            return &RefundResult{
                ID:     refund.ID,
                Amount: refund.Amount,
                Status: "succeeded",
            }, nil
        }
        
    case "email.send":
        // 查询邮件服务：该 message_id 是否已发送？
        status, err := email.GetStatus(effect.ExternalID)
        if err == nil && status == "sent" {
            return &EmailResult{Status: "sent"}, nil
        }
    }
    
    // 2. 外部验证失败：需要重新执行
    return nil, ErrEffectVerificationFailed
}
```

## 4. Replay 时的 At-Most-Once

### 4.1 什么是 Replay？

Replay 是 Aetheris 的调试功能：**重新运行历史执行**。

```
场景：Agent 行为异常，需要调试
1. 拉取历史事件流
2. 从头重放每一步
3. 观察每一步的输入输出
4. 定位问题
```

### 4.2 Replay 必须也保证 At-Most-Once

Replay 时绝对不能真的调用外部 API！

```go
func (r *Runner) Replay(jobID string) error {
    events := jobstore.ListEvents(jobID)
    
    for _, event := range events {
        switch event.Type {
        case "ToolInvocated":
            // 不调用工具，直接从 Ledger 获取结果
            entry, _ := ledger.Get(event.IdempotencyKey)
            if entry != nil {
                // 使用缓存结果，不调用外部
                r.injectResult(entry.Output)
                continue
            }
            
            // Ledger 没有记录：可能是 Replay 前发生了清理
            // 需要执行验证逻辑
            effect, _ := effectStore.Get(event.EffectID)
            if effect.ExternalID != "" {
                output, err := r.verifyAndRecover(effect)
                if err == nil {
                    r.injectResult(output)
                    continue
                }
            }
            
            // 无法恢复：Replay 失败
            return ErrReplayFailed
            
        case "LLMRequest":
            // LLM 可以重新调用（幂等）
            // 或者也使用缓存
        }
    }
    
    return nil
}
```

## 5. 与传统方案的对比

### 5.1 数据库事务

```sql
BEGIN TRANSACTION;
  INSERT INTO orders ...;
  UPDATE inventory ...;
  INSERT INTO payments ...;
COMMIT;
```

**问题**：
- 只能保证数据库一致性
- 外部 API（Stripe、邮件）不在事务内
- 崩溃后外部状态未知

### 5.2 幂等 API

很多 API 本身支持幂等：

```bash
# Stripe 的幂等 key
curl -X POST https://api.stripe.com/v1/refunds \
  -u sk_test_xxx: \
  -d charge=ch_xxx \
  -d idempotency-key=job:123:step:2
```

**问题**：
- 不是所有 API 都支持幂等
- 幂等 key 格式不统一
- Agent 需要为每个工具适配

### 5.3 Aetheris 的统一方案

| 维度 | 数据库事务 | 幂等 API | Aetheris |
|------|------------|----------|-----------|
| 外部 API | ❌ | ⚠️ 部分支持 | ✅ 统一抽象 |
| 崩溃恢复 | ❌ | ⚠️ 需要手动 | ✅ 自动 |
| Replay | ❌ | ❌ | ✅ 安全重放 |
| 审计 | ❌ | ❌ | ✅ 完整记录 |

## 6. 实现细节

### 6.1 Ledger 表结构

```sql
CREATE TABLE tool_ledger (
    idempotency_key VARCHAR(255) PRIMARY KEY,
    job_id UUID NOT NULL,
    step_id VARCHAR(50) NOT NULL,
    tool_name VARCHAR(100) NOT NULL,
    input_hash VARCHAR(64) NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'committed',
    output JSONB,
    external_id VARCHAR(255),  -- 外部系统返回的 ID
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    
    INDEX idx_job_id (job_id),
    INDEX idx_status (status)
);
```

### 6.2 代码位置

```
pkg/
├── effects/           # 副作用相关
│   ├── ledger.go      # Tool Ledger 实现
│   ├── store.go       # Effect Store 实现
│   └── twopc.go       # 两阶段提交
```

## 7. 小结

At-Most-Once 是 Agent Runtime 的**核心能力**：

1. **Tool Ledger** — 记录工具调用结果，防止重复执行
2. **Idempotency Key** — 唯一标识每次调用
3. **Two-Phase Commit** — 保证调用和记录的原子性
4. **外部验证** — 崩溃后通过查询外部系统确认状态
5. **安全 Replay** — 调试时不调用外部，只用缓存结果

有了 At-Most-Once，Agent 才能真正用于生产——**不再担心重复扣款，不再担心邮件重发**。

---

*下篇预告：Checkpoint 与状态恢复——Worker 崩溃后发生了什么*
