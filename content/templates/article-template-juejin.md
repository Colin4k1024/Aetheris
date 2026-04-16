# 掘金文章模板

用于在掘金平台发布技术文章的模板和指南。

## 掘金文章特点

- **代码展示**：掘金对代码块支持很好，语法高亮清晰
- **系列文章**：可以组织成系列，适合深度内容
- **标签系统**：合理使用标签增加曝光
- **推荐算法**：完读率、互动率影响推荐

## 模板结构

```
---
title: 【类型】文章标题
tags: [标签1, 标签2, 标签3]
publish: true
description: 文章描述（显示在列表页）
---

## 前言
- 背景介绍
- 文章目标
- 面向读者

## 正式开始

### 第一节
内容...

### 第二节
内容...

## 总结
- 回顾要点
- 下一步
- 相关资源
```

## 示例文章：入门系列

---

title: 【入门】5分钟上手 Aetheris：打造不间断的 AI Agent
tags: ['Aetheris', 'AI Agent', 'Go', '大语言模型']
publish: true
description: 本文手把手教你使用 Aetheris，构建一个具有崩溃恢复能力的 AI Agent，包含完整代码示例。

---

## 前言

想象一下这个场景：

> 你的 AI 客服正在处理一个复杂订单，突然服务器断电了。等电力恢复，你以为它会从断点继续处理——但实际上，它从头开始了。用户等了5分钟，又从头描述问题。

这不是科幻，是很多 AI 应用的现实。

今天我们来解决这个问题。

### 本文目标

- 理解 Aetheris 是什么、解决什么问题
- 动手实现一个持久化的 Agent
- 掌握核心概念：Checkpoint、事件溯源、At-Most-Once

### 面向读者

- 有 Go 基础
- 了解 AI Agent 基本概念
- 想做生产级 Agent 应用

---

## 一、Aetheris 是什么？

Aetheris 是一个 **Durable Execution Runtime**（持久化执行运行时）。

类比理解：

| 概念 | 传统 Web | Aetheris |
|------|----------|----------|
| 框架 | Gin/Echo | LangChain/LangGraph |
| 运行时 | K8s/Gunicorn | Aetheris |
| 关键特性 | 自动扩缩容 | 崩溃恢复、幂等执行 |

核心价值：**让你的 Agent 可靠地运行在生产环境**。

---

## 二、快速开始

### 2.1 安装

```bash
# 安装 CLI
go install github.com/Colin4k1024/Aetheris/cmd/cli@latest

# 验证安装
aetheris version
# aetheris v1.5.0
```

### 2.2 初始化项目

```bash
aetheris init hello-aetheris
cd hello-aetheris
```

### 2.3 编写你的第一个 Agent

创建 `main.go`：

```go
package main

import (
    "context"
    "fmt"
    
    "github.com/cloudwego/eino/components/model"
    "github.com/cloudwego/eino/components/prompt"
    "rag-platform/internal/runtime/eino"
)

func main() {
    ctx := context.Background()
    
    // 1. 创建 Agent
    agent, err := eino.NewAgentFactory(ctx, &eino.AgentConfig{
        Model: model.OpenAI("gpt-4"),
    })
    if err != nil {
        panic(err)
    }
    
    // 2. 使用 Aetheris 包装
    durable := aetheris.Wrap(agent)
    
    // 3. 提交任务
    job, err := durable.Submit(ctx, "你好，请介绍一下 Aetheris")
    if err != nil {
        panic(err)
    }
    
    // 4. 获取结果
    result, err := job.Result(ctx)
    fmt.Println(result)
}
```

### 2.4 运行

```bash
go run main.go

# 输出
# Aetheris 是一个持久化执行运行时...
```

---

## 三、核心概念

### 3.1 Checkpoint（检查点）

每次工具调用完成后，Aetheris 会自动保存状态：

```
[User Input] → [Agent] → [Tool Call] → [Checkpoint] → [Next Step]
                                    ↓
                            状态已保存
```

### 3.2 事件溯源（Event Sourcing）

每一步操作都被记录为事件：

```go
// 事件类型
type EventType string

const (
    EventToolCall    EventType = "tool_call"
    EventCheckpoint  EventType = "checkpoint" 
    EventLLMResponse EventType = "llm_response"
    EventError       EventType = "error"
)
```

这些事件可以用于：
- **调试**：重放任意执行
- **审计**：完整的操作历史
- **分析**：性能瓶颈分析

### 3.3 At-Most-Once 执行

防止工具被重复调用：

```
传统方式：请求超时 → 重试 → 可能重复执行 ❌
Aetheris：   请求超时 → 重试 → 检测到已执行 → 跳过 ✅
```

---

## 四、实战：构建订单处理 Agent

### 4.1 需求

处理用户订单，包含以下步骤：
1. 验证用户身份
2. 检查库存
3. 创建订单
4. 发送确认邮件

关键是：**任何步骤失败都要能重试，但不能重复执行**。

### 4.2 实现

```go
func createOrderAgent() *eino.Agent {
    return eino.NewAgentFactory(nil, &eino.AgentConfig{
        Model: model.OpenAI("gpt-4"),
        Tools: []eino.Tool{
            verifyUser,
            checkInventory,
            createOrder,
            sendConfirmation,
        },
        // 关键配置：启用 At-Most-Once
        Idempotent: true,
    })
}
```

### 4.3 测试崩溃恢复

```bash
# 启动服务
aetheris run

# 在另一个终端，手动终止进程
kill -9 $(pgrep aetheris)

# 重新启动
aetheris run

# 查看任务恢复
aetheris jobs list
# JOB ID        STATUS      PROGRESS
# abc-123       resumed     step 3/4
```

---

## 五、总结

今天我们学到：

1. **Aetheris 是什么**：持久化执行运行时
2. **核心概念**：Checkpoint、事件溯源、At-Most-Once
3. **快速开始**：5分钟跑通第一个 Agent
4. **实战**：订单处理 Agent

### 下一步

- 📖 阅读 [官方文档](https://docs.aetheris.ai)
- 💬 加入 [Discord 社区](https://discord.gg/PrrK2Mua)
- ⭐ 给 [GitHub](https://github.com/Colin4k1024/Aetheris) 点个 Star

---

## 系列文章

- 【入门】5分钟上手 Aetheris ✅
- 【实战】构建生产级 AI 客服（coming soon）
- 【深入】Aetheris 执行模型源码解析（coming soon）

---

## 标签说明

掘金的推荐算法依赖标签，正确使用很重要：

| 标签类型 | 示例 | 说明 |
|----------|------|------|
| 平台/框架 | `Aetheris` | 核心主题 |
| 语言 | `Go` | 技术栈 |
| 领域 | `AI Agent` `大语言模型` | 应用领域 |
| 进阶 | `RAG` `LangChain` | 相关技术 |

建议使用 **5-8 个标签**。

---

## 发布检查清单

- [ ] 标题包含 `[入门]` `[实战]` 等类型标识
- [ ] description 简洁明了，不超过100字
- [ ] 代码块有语法高亮
- [ ] 有目录结构（文内添加）
- [ ] 配图清晰
- [ ] 结尾有 CTA
- [ ] 检查错别字和格式
