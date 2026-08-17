# Aetheris Durability SDK

**为任何 Go agent 添加 crash recovery、checkpoint 和幂等性。零框架依赖。**

## 为什么需要这个？

你的 agent 在处理 1000 条客户记录，处理到第 847 条时进程挂了。

- **没有 Aetheris SDK**: 从头开始，重新跑 847 次 LLM 调用，祈祷没有副作用被重复执行
- **有了 Aetheris SDK**: 重启，从第 847 条继续，已执行的副作用不会重复

## 安装

```bash
go get github.com/Colin4k1024/Aetheris/durability
```

## 30 秒上手

```go
package main

import (
    "context"
    "fmt"
    "log"

    "github.com/Colin4k1024/Aetheris/durability/core"
)

func main() {
    ctx := context.Background()

    // 1. 创建 store（开发用内存，生产用 postgres）
    store := core.NewMemoryStore()

    // 2. 创建 runner
    runner := core.NewRunner(store)

    // 3. 启动一个 job
    job, _ := runner.Start(ctx, "process-order", map[string]any{
        "order_id": "ORD-123",
        "amount":   99.99,
    })

    // 4. 定义 steps（每步接收 state，返回更新后的 state）
    steps := []core.Step{
        {
            ID: "validate",
            Fn: func(ctx core.Context, state map[string]any) (map[string]any, error) {
                // 调用验证 API...
                return map[string]any{"validated": true}, nil
            },
        },
        {
            ID:         "charge",
            MaxRetries: 3, // 失败自动重试 3 次
            Fn: func(ctx core.Context, state map[string]any) (map[string]any, error) {
                // 调用支付 API...
                return map[string]any{"charged": true, "txn_id": "TXN-456"}, nil
            },
        },
        {
            ID: "ship",
            Fn: func(ctx core.Context, state map[string]any) (map[string]any, error) {
                // 调用物流 API...
                return map[string]any{"shipped": true}, nil
            },
        },
    }

    // 5. 执行 — 如果中间挂了，再调一次 Execute 就从上次继续
    result, err := runner.Execute(ctx, job.ID, steps)
    if err != nil {
        log.Fatal(err)
    }
    fmt.Printf("Result: %+v\n", result)
}
```

## Crash Recovery 演示

```go
// 假设 process 在 "charge" 步骤挂了
_, err := runner.Execute(ctx, job.ID, steps)
// err: "process killed"

// 重启后再调一次 — "validate" 被跳过（已 checkpoint），从 "charge" 继续
result, err := runner.Execute(ctx, job.ID, steps)
// 成功！validate 没有被重复执行
```

## 幂等工具（防止副作用重复执行）

```go
import "github.com/Colin4k1024/Aetheris/durability/idempotent"

idem := idempotent.New(store)

// 包装一个有副作用的函数
sendEmail := idem.Wrap("send-email", func(ctx context.Context, input map[string]any) (map[string]any, error) {
    err := emailService.Send(input["to"].(string), input["subject"].(string))
    return map[string]any{"sent": true}, err
})

// 同一个 idempotency key 只会执行一次
result1, _ := sendEmail(ctx, "order-123-email", map[string]any{"to": "user@example.com"})
result2, _ := sendEmail(ctx, "order-123-email", map[string]any{"to": "user@example.com"})
// result2 直接返回缓存结果，不会重复发送邮件
```

## 生产环境（PostgreSQL）

```go
import "github.com/Colin4k1024/Aetheris/durability/postgres"

store, err := postgres.NewStore(ctx, "postgres://user:pass@localhost:5432/aetheris")
if err != nil {
    log.Fatal(err)
}
defer store.Close()

runner := core.NewRunner(store)
// ... 其余代码完全一样
```

表结构自动创建，无需手动 migration。

## 架构

```
┌─────────────────────────────────────────────┐
│  你的 Agent 代码                              │
│  (任何框架: eino/langchain/自研/纯 Go)        │
├─────────────────────────────────────────────┤
│  Aetheris Durability SDK                    │
│  ┌──────────┐ ┌──────────┐ ┌──────────────┐ │
│  │  Runner   │ │ Idempot. │ │  Checkpoint  │ │
│  │ (步骤执行) │ │  (幂等)   │ │  (状态快照)  │ │
│  └──────────┘ └──────────┘ └──────────────┘ │
├─────────────────────────────────────────────┤
│  Store 接口                                  │
│  ┌──────────┐ ┌──────────────────────────┐  │
│  │ Memory   │ │ PostgreSQL               │  │
│  │ (开发/测试)│ │ (生产)                    │  │
│  └──────────┘ └──────────────────────────┘  │
└─────────────────────────────────────────────┘
```

## 包结构

| 包 | 用途 |
|---|------|
| `core` | 核心类型、Store 接口、MemoryStore、Runner |
| `postgres` | PostgreSQL Store 实现 |
| `idempotent` | 幂等工具包装器 |

## 与 Aetheris 运行时的关系

这个 SDK 是 Aetheris 项目提取的核心原语。它：

- ✅ 零框架依赖（不依赖 eino、hertz）
- ✅ 零基础设施依赖（内存实现即开即用）
- ✅ 任何 Go 项目都能 import 使用
- ✅ 生产级 PostgreSQL 实现可选

完整的 Aetheris 运行时（多 worker 调度、事件溯源审计、DAG 执行等）仍然在主项目中。

## License

Apache License 2.0
