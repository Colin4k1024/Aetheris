# 为什么 AI Agent 需要自己的 Runtime

> 传统微服务框架擅长处理请求-响应式的短任务，但 AI Agent 的工作方式截然不同：自主决策、多步推理、长时间运行、调用外部工具。当你的 Agent 执行到一半突然崩溃，是从头重跑还是从断点恢复？当同一个 Tool 被执行两次，是「无所谓」还是「灾难性错误」？本文从 Agent 与微服务的本质差异出发，解析为什么通用工作流引擎不够用，以及 CoRag（Aetheris）如何用事件溯源 + 检查点 + At-Most-Once 为 Agent 提供专属的执行契约。

## Agent 与微服务：两种截然不同的执行模型

传统微服务的执行模型是**请求驱动的同步调用链**：客户端发起请求，服务处理，返回响应。整个过程通常在毫秒到秒级完成，失败时重试即可，最多产生一条错误日志。这种模型在互联网服务中运行良好，但它假设了一个基本前提——**执行单元（函数/服务）是无状态的、幂等的、且不关心「中途去了哪里」**。

AI Agent 打破了这些假设。一个典型 Agent 的执行过程是这样的：

```
用户目标：「帮我分析这家公司并生成一份投资报告」

Agent 执行流程：
1. 调用 Search 工具查询公司背景（Tool Call #1）
2. 调用 LLM 分析财务数据（Tool Call #2）
3. 调用 ReportGenerator 生成报告（Tool Call #3）
4. 用户审批：「数据来源标注不够，补充一下」
5. Agent 继续执行 Tool Call #4、#5
6. 再次等待用户确认
7. ...
```

这带来了几个**传统微服务从未面临的问题**：

**第一，执行是长期且可中断的。** 一个复杂任务可能持续几分钟到几小时，期间用户可能随时介入（审批、修改目标、追加指令）。传统 HTTP 请求-响应模型无法承载这种「暂停-等待-恢复」的生命周期。

**第二，执行路径是动态的。** Agent 根据 LLM 的推理结果决定下一步做什么，不是预先写死的调用图。你无法在部署时确定执行顺序，也无法用静态的 API 契约描述它。

**第三，Tool 调用具有外部副作用。** 调用 `send_email` 意味着真的发了邮件；调用 `deduct_credit` 意味着真的扣了钱。这类操作**不能被执行两次**，但传统框架的重试机制（「失败了就重跑」）恰恰会导致重复执行。

**第四，崩溃不是「偶发异常」，而是「常态」。** 在长时间运行的任务中，Worker 进程被杀掉、容器被重启、Kubernetes Pod 被调度迁移——这些事件几乎是必然发生的。传统框架假设「跑完就结束」，而 Agent Runtime 必须假设「随时可能崩溃，必须能从断点恢复」。

**第五，决策过程需要审计。** 在金融、医疗、政府等合规场景中，你不仅需要知道「Agent 最终做了什么」，还需要知道「Agent 为什么在第 3 步调用了这个 Tool、第 5 步做出了那个决策」。传统微服务日志是分散的、难以回溯的；而 Agent 的决策过程需要作为**证据链**被完整保留。

这些差异意味着：传统微服务框架（FastAPI、Hertz、gRPC）以及通用工作流引擎（Temporal、AWS Step Functions）并不能直接满足 Agent 的需求。Agent 需要一个**专为自身特性设计的执行运行时**。

## 现有方案的不足

让我们具体看看几种常见方案的局限性。

### 方案一：无状态调用（直接调用 LLM API）

最简单的做法是：每次需要 Agent 能力时，直接调用 LLM API，传入历史对话，返回结果。这本质上是一个**无状态的函数调用**。

问题显而易见：没有持久化，进程崩溃后所有状态丢失；没有断点恢复，任务只能从头重跑；没有幂等保证，多次调用可能产生多次副作用；没有审计日志，决策过程不可追溯。

这种方案只适合原型和 Demo，生产级应用无法接受这些限制。

### 方案二：用定时任务 + 数据库模拟状态机

一些团队在 PostgreSQL 中建几张表（Job 表、Step 表、State 表），用定时任务轮询 Job 状态，模拟 Agent 的执行流程。

这种方式的问题是：

- **崩溃恢复靠自己写**。如果在执行 Tool 的过程中数据库写入之前崩溃，定时任务不知道当前 Job 处于什么状态，要么重跑（可能重复执行 Tool），要么死等（Job 永久卡住）。
- **重复执行风险高**。没有幂等性保障，如果同一个 Job 被两个 Worker 同时认领，两个 Worker 都去执行同一个 Tool，外部副作用会被触发两次。
- **无法审计推理过程**。State 表只记录「当前状态」，不记录「为什么到达这个状态」。当合规审计要求你解释「第 3 步的决策依据是什么」，数据库里的状态字段给不出答案。
- **水平扩展困难**。自建的状态机在多 Worker 场景下需要复杂的分布式锁或乐观并发控制，容易出现竞态条件。

### 方案三：通用工作流引擎（Temporal / AWS Step Functions）

Temporal 是目前最成熟的通用工作流引擎，提供了持久化执行、活动（Activity）幂等性、工作流状态快照等能力。AWS Step Functions 则是云原生的流程自动化方案。

这些方案对于**通用业务工作流**（审批流、数据处理流水线、订单履约等）确实强大，但它们在面对 AI Agent 时有几个关键局限：

**第一，活动（Activity）模型与 Tool 调用语义不匹配。** Temporal 的 Activity 是「可重试的业务动作」，默认语义是「失败了就重试，直到成功」。这对扣款、发短信这类**不能重试**的操作是致命的——Temporal 建议通过「向外部系统查询状态」来绕过，但这需要大量额外开发工作。AI Agent 的 Tool 调用恰恰是这种「要么不执行，要么执行一次，不能执行多次」的操作，而通用引擎的默认语义与此相悖。

**第二，工作流状态快照无法捕获 LLM 推理过程。** Temporal 的工作流状态是代码执行状态的序列化，但 LLM 的推理过程（prompt、context、思考链）是存在于工作流之外的。要完整记录 Agent 的决策过程，需要在应用层做大量定制。

**第三，人机协同（Human-in-the-Loop）支持有限。** Agent 场景中，用户经常需要「审批后再继续」「修改目标后重新执行」「在等待用户输入时暂停任务」。Temporal 确实有 Signal 机制，但其设计初衷是「工作流主动发信号给外部」，而非「外部随时介入修改工作流状态」。在 Agent 场景中，这种能力是不可协商的基础需求。

**第四，事件历史不等于证据链。** 合规场景需要的不仅是「某一步在某个时间执行了」，还要知道「当时的输入是什么、LLM 返回了什么、为什么选择了这个 Tool」。Temporal 的历史记录是低层次的，不保留完整的推理上下文。

这并不是说 Temporal 不好——它在通用工作流场景下是优秀的选择。但如果你的核心场景是**需要持久化、可恢复、有外部副作用、且需要完整审计的 AI Agent**，专用 Runtime 能提供开箱即用的能力，而通用引擎需要大量定制才能达到同等效果。

## CoRag 的解决思路

CoRag（即 Aetheris）是一个专为 AI Agent 设计的执行运行时。它的核心设计哲学是：**把「可信执行」作为第一等公民**，而不是事后补救的功能。

CoRag 的解决思路围绕三个核心能力展开：

### 1. 事件溯源（Event Sourcing）：记录完整执行历史

CoRag 不只为 Job 存储「当前状态」，而是存储**完整的事件流**。每个关键节点都会写入一条不可变的事件记录：

```go
// 事件类型示例
JobCreated        // Job 被创建
PlanGenerated     // Planner 产出任务图
NodeStarted       // 某节点开始执行
NodeFinished      // 某节点执行完成
ToolInvocationStarted   // 工具调用开始
ToolInvocationFinished   // 工具调用完成（含结果）
CommandCommitted  // LLM 调用结果已提交
CheckpointCreated // 检查点已创建
JobCompleted      // Job 成功结束
JobFailed         // Job 执行失败
```

这些事件按时间顺序追加存储，构成一份完整的**执行事实日志**。与传统日志不同，事件溯源中的每条记录都是「已发生的事实」，可用于重建任意时刻的执行状态。

这带来了几个关键优势：

- **完整审计**：任何时候都可以回溯「Job 在第 5 步时处于什么状态、当时的上下文是什么、为什么执行了这个 Tool」。
- **无损恢复**：崩溃后重建状态时，不依赖「当前状态快照」是否最新（快照可能恰好在写入前丢失），而是将事件流视为唯一真相来源。
- **多 Worker 共享**：所有 Worker 共享同一份事件流，新 Worker 可以从事件中完整重建前任的执行上下文。

### 2. 检查点（Checkpoint）：从断点恢复，而不是从头重跑

CoRag 在每个节点（Step）执行完成后写入检查点：

```go
// 检查点的核心字段
type Checkpoint struct {
    JobID       string    // Job 唯一标识
    Cursor      string    // 恢复点：下一个待执行的节点 ID
    CreatedAt   time.Time // 创建时间
}
```

检查点记录了「执行到哪了」。当 Worker 崩溃后，新 Worker 认领该 Job 时：

1. 从 JobStore 读取完整事件流
2. 通过 `ReplayContextBuilder` 从事件重建执行状态（已完成节点、已提交工具调用、状态变更记录等）
3. 读取 Job 的 `Cursor`，从对应节点之后继续执行
4. 已完成的节点从事件/Ledger 注入结果，**不会重新执行**

```
[Worker A 执行 Step1 → 写 Checkpoint(Cursor=Step2) → 写事件]
         ↓
[Worker A 崩溃]
         ↓
[Worker B Claim Job → ListEvents → 构建 ReplayContext]
         ↓
[从 Cursor=Step2 恢复：Step1 从事件注入结果，不重执行]
         ↓
[执行 Step2, Step3, ... → 继续写 Checkpoint 与事件]
```

这就是「从断点恢复」的核心逻辑：只重跑未完成的部分，已完成的不动。

### 3. At-Most-Once：Invocation Ledger 保证不重复执行

这是 CoRag 最核心的创新之一。事件溯源解决了「状态恢复」的问题，但还有一个关键问题：**Replay 时，已完成节点的 Tool 调用会不会被执行两次？**

答案是：**在 CoRag 中，不会。**

CoRag 引入了 **Invocation Ledger（调用账本）**，为每个 Tool/LLM 调用提供执行许可：

```go
// Ledger 的裁决结果
type LedgerResult int

const (
    AllowExecute           LedgerResult = iota  // 尚无记录，允许执行
    ReturnRecordedResult                         // 已有记录，返回已记录结果，禁止重复执行
)
```

执行流程如下：

```
[Runner 要执行 Tool_A]
         ↓
[向 Ledger 请求许可：Ledger.Acquire(tool_id)]
         ↓
   ┌─────┴─────┐
   │           │
Ledger 有记录？  Ledger 无记录？
   │           │
   ↓           ↓
[返回          [返回 AllowExecute]
 ReturnRecordedResult]          ↓
   ↑                    [执行 Tool_A]
   │                    [向 Ledger Commit 结果]
   │                           ↓
   └────── 注入已记录结果 ←──────┘
```

关键是：**唯一会真正调用 Tool 的代码路径是 Ledger 返回 `AllowExecute` → 执行 → `Commit`**。Replay 或重试时，Ledger 发现已有记录即返回 `ReturnRecordedResult`，Runner 只做结果注入，不调用 Tool。

这意味着：

- **崩溃恢复不重执行**：Worker 在 Tool 执行后、结果写回前崩溃，新 Worker 从 Checkpoint 恢复时，Ledger 已有该 Tool 的提交记录，直接注入结果。
- **多 Worker 不冲突**：两个 Worker 同时认领同一 Job，Ledger 裁决只允许其中一个执行，另一个收到 `ReturnRecordedResult`。
- **外部副作用安全**：扣款、发短信、发邮件这类「不能执行两次」的操作，在 CoRag 的语义下是安全的。

## 实际代码示例

下面通过一个简化示例展示 CoRag 的核心用法（完整示例见 [examples/basic_agent](../../examples/basic_agent)）：

### 定义 Agent 与 Tool

```go
// 定义一个搜索 Tool
searchTool := &tool.Definition{
    Name:        "search",
    Description: "搜索互联网获取相关信息",
    Parameters:  schema,
}

// 定义 Agent
agent := &agent.Config{
    Name:        "research_agent",
    Description: "研究助手：搜索 → 分析 → 生成报告",
    Model:       "gpt-4o",
    Tools:       []tool.Definition{searchTool},
    MaxSteps:    20,
}
```

### 提交 Job 并等待结果

```go
// 创建一个长时任务
job, err := runtime.CreateJob(ctx, &CreateJobRequest{
    AgentConfig: agent,
    UserGoal:    "分析 Apple 公司的竞争优势并生成一份投资研究报告",
    Priority:    5,
})
if err != nil {
    log.Fatalf("创建 Job 失败: %v", err)
}

// 提交后立即返回，任务在后台异步执行
fmt.Printf("Job 已创建: %s\n", job.ID)
fmt.Printf("初始状态: %s\n", job.Status)
```

### 暂停等待人工审批（Human-in-the-Loop）

```go
// 在 Agent 内部，当需要用户确认时，会自动进入 Parked 状态
// 外部可以通过以下方式注入审批结果：

err = runtime.InjectDecision(ctx, job.ID, &DecisionRequest{
    Approved:  true,
    Comment:   "数据来源符合要求，继续执行",
})
if err != nil {
    log.Fatalf("审批注入失败: %v", err)
}
```

### 崩溃恢复验证

```bash
# 启动 Worker 处理任务
go run ./cmd/worker

# 在任务执行过程中，手动 kill Worker 进程（模拟崩溃）
# 然后重新启动 Worker

# 观察日志：新 Worker 会从 Checkpoint 恢复
# 日志中应出现类似：
# [Recovery] Job xxx claimed, cursor=Step3, replaying completed nodes...
# [Recovery] Step1 injected from event store, skipping execution
# [Recovery] Step2 injected from event store, skipping execution
# [Resume] Executing from Step3...
```

### 查看执行历史与审计

```go
// 获取 Job 的完整事件流
events, err := runtime.ListEvents(ctx, job.ID, nil)
if err != nil {
    log.Fatalf("查询事件失败: %v", err)
}

for _, e := range events {
    fmt.Printf("[%s] %s: %s\n", e.Timestamp, e.Type, e.Summary)
}

// 输出示例：
// [2026-04-14 10:00:01] JobCreated: research_agent 创建
// [2026-04-14 10:00:02] PlanGenerated: 生成 5 步任务图
// [2026-04-14 10:00:03] NodeStarted: Step1(search)
// [2026-04-14 10:00:05] ToolInvocationFinished: search 返回 12 条结果
// [2026-04-14 10:00:06] NodeFinished: Step1 完成
// [2026-04-14 10:00:06] CheckpointCreated: cursor=Step2
// [2026-04-14 10:00:07] StatusParked: 等待人工审批
// [2026-04-14 10:15:00] DecisionInjected: 审批通过
// ...
```

## 与 Temporal 的关系：不是竞争，是分工

需要明确的是，CoRag 并不试图取代 Temporal。事实上，两者解决的问题有重叠，但侧重点不同：

| 维度 | CoRag | Temporal |
|------|-------|----------|
| **核心定位** | AI Agent 可信执行运行时 | 通用工作流引擎 |
| **Tool/LLM 调用** | 一等公民，内置 At-Most-Once | Activity 语义，需额外配置幂等性 |
| **Human-in-the-Loop** | 内置 Wait/Signal/StatusParked | 支持但非核心场景 |
| **事件溯源** | 内置，完整保留推理上下文 | 有历史记录，但不保留完整上下文 |
| **检查点恢复** | 自动，从 Cursor 恢复 | 需要 Workflow 实现快照逻辑 |
| **适用场景** | AI Agent 长任务、合规审计 | 通用业务流程、数据处理 |

如果你已经在使用 Temporal，且核心场景是「非 Agent 的业务工作流」，继续用 Temporal 是合理的。如果你的核心场景是「AI Agent 的可信执行」，CoRag 能提供开箱即用的专属能力，而无需在 Temporal 基础上做大量定制开发。

## 小结

AI Agent 的执行模型与传统微服务有着本质差异：长期运行、可中断、多步推理、Tool 调用带副作用、决策过程需要审计。通用方案（无状态调用、自建状态机、通用工作流引擎）要么无法解决这些问题，要么需要大量定制开发才能勉强应对。

CoRag 的设计思路是**从源头解决**这些问题：

- **事件溯源**：用完整的事件流记录执行历史，支持任意时刻的状态重建和完整审计。
- **检查点**：每个节点执行完成后保存恢复点，崩溃后从断点继续，而非从头重跑。
- **Invocation Ledger**：为每个 Tool/LLM 调用提供执行许可裁决，保证 At-Most-Once，外部副作用不会重复执行。

这三个能力共同构成了 Agent Runtime 的核心契约：**可信执行**——你交给 Agent 的任务，无论执行多久、经历多少次崩溃、换过多少 Worker，最终都会以「恰好一次」的方式完成，并且每一步都有完整的执行记录可供审计。

如果你正在构建需要持久化、可恢复、有外部副作用、且需要完整审计的 AI Agent 应用，CoRag 值得一试。

## 延伸阅读

- [Aetheris 入门 - 5 分钟快速开始](./01-quick-start.md)
- [事件溯源与 Replay 恢复：从崩溃中继续执行](./07-event-sourcing-replay.md)
- [At-Most-Once 执行保证：Invocation Ledger 原理](./06-at-most-once-ledger.md)
- [何时选择 Aetheris：与 LangGraph、Temporal 的对比](./11-when-to-choose-aetheris.md)
- [design/execution-guarantees.md](../../design/execution-guarantees.md) — 正式保证一览与条件
