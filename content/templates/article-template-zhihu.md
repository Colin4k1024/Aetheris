# 知乎文章模板

用于在知乎发布技术文章的模板和指南。

## 模板结构

```
标题：[类型] 主题 - 简洁有力
副标题：（可选）补充说明

---

## 引言
- 吸引注意力的hook
- 为什么这个话题重要
- 文章目标

## 目录
（可选，便于长文阅读）

## 正文
### 第一部分：背景与问题
### 第二部分：核心概念/原理
### 第三部分：实战/案例
### 第四部分：总结与展望

## 互动号召
- 评论区问题
- 相关资源链接
```

## 示例文章

---

# 【实战】用 Aetheris 5分钟构建一个可靠的 AI 客服 Agent

> 本文详细介绍如何使用 Aetheris 构建一个具有崩溃恢复能力的 AI 客服系统，包括代码实现和部署方案。

---

## 引言

做客服系统，最怕什么？

我见过太多团队搭了一套 AI 客服，上线第一天：

- ❌ 服务器重启 → 用户会话全部丢失
- ❌ 机器人卡死 → 用户等了 5 分钟没响应
- ❌ 重复回复 → 用户收到两条一模一样的道歉

这些问题听起来很小，但每一个都在伤害用户体验。

今天我来介绍一个不同的方案 —— 基于 Aetheris 构建的客服 Agent，它天然具备：

- ✅ **崩溃恢复**：服务重启？自动从上次checkpoint继续
- ✅ **幂等执行**：同样的请求，永远只处理一次
- ✅ **完整审计**：每次对话、每个决策，全部记录在案

---

## 什么是 Aetheris？

Aetheris 是一个为 AI Agent 设计的**持久化执行运行时**。

你可以把它理解为 "Agent 的 Kubernetes"——它不负责编写 Agent 的逻辑，而是负责**可靠地运行** Agent。

核心特性：

| 特性 | 说明 |
|------|------|
| 持久化执行 | 崩溃后从 checkpoint 恢复 |
| At-Most-Once | 工具调用永不重复 |
| 事件溯源 | 完整执行历史，可追溯 |
| 人机协作 | 支持暂停等待人工审批 |

---

## 开始实战：构建客服 Agent

### 环境准备

```bash
# 安装 Aetheris CLI
go install github.com/Colin4k1024/Aetheris/cmd/cli@latest

# 初始化项目
aetheris init customer-service
cd customer-service
```

### 定义 Agent 逻辑

```go
package main

import (
    "context"
    "fmt"
    "github.com/cloudwego/eino/components/model"
    "github.com/cloudwego/eino/components/prompt"
    "rag-platform/internal/runtime/eino"
)

type CustomerQuery struct {
    CustomerID string
    Question   string
}

func main() {
    ctx := context.Background()
    
    // 1. 创建 Agent
    agent, err := eino.NewAgentFactory(ctx, &eino.AgentConfig{
        Model: model.OpenAI("gpt-4"),
        Tools: []eino.Tool{
            // 客服需要的工具：查订单、查知识库、发消息
        },
    })
    
    // 2. 包装为持久化 Agent
    durableAgent := aetheris.Wrap(agent)
    
    // 3. 提交任务
    job, err := aetheris.Submit(ctx, durableAgent, CustomerQuery{
        CustomerID: "CUST-12345",
        Question:   "我的订单什么时候发货？",
    })
    
    fmt.Printf("Job ID: %s\n", job.ID())
}
```

### 实现暂停与审批

关键场景：用户要求退款超过 500 元，需要人工审批。

```go
func shouldApproveRefund(amount float64) bool {
    return amount > 500
}

func handleRefund(ctx context.Context, job *aetheris.Job, order Order) error {
    if shouldApproveRefund(order.Amount) {
        // 暂停任务，等待审批
        job.Pause(ctx, "refund_approval_required")
        
        // 发送审批请求
        notifyManager(ctx, order)
        
        return nil // 稍后会 resume
    }
    
    // 小额退款，直接处理
    return processRefund(ctx, order)
}
```

---

## 部署方案

### Docker Compose 快速部署

```yaml
version: '3.8'
services:
  api:
    image: aetheris/api:latest
    ports:
      - "8080:8080"
    environment:
      - DATABASE_URL=postgres://...
      - REDIS_URL=redis://...
    depends_on:
      - postgres
      - redis

  worker:
    image: aetheris/worker:latest
    environment:
      - DATABASE_URL=postgres://...
      - REDIS_URL=redis://...
    deploy:
      replicas: 3  # 多副本，高可用
```

---

## 效果对比

| 场景 | 传统方案 | Aetheris 方案 |
|------|----------|---------------|
| 服务器重启 | 会话丢失，从头开始 | 自动从 checkpoint 恢复 |
| 重复点击 | 可能重复下单 | 幂等保证，不会重复 |
| 问题追溯 | 很难 | 完整事件日志 |
| 审批流程 | 需要额外开发 | 内置 Pause/Resume |

---

## 总结

今天我们：

1. 介绍了 Aetheris 的核心概念
2. 实现了一个带审批流程的客服 Agent
3. 给出了 Docker 部署方案

如果你也在做需要**可靠运行**的 AI Agent，欢迎评论区聊聊你的场景。

**相关资源**：

- [Aetheris 官方文档](https://docs.aetheris.ai)
- [GitHub 仓库](https://github.com/Colin4k1024/Aetheris)
- [Discord 社区](https://discord.gg/PrrK2Mua)

---

**讨论**：

> 你的团队目前是怎么处理 Agent 故障恢复的？有什么痛点？欢迎在评论区分享！

---

## SEO 优化

### 标题优化
- 前置关键词：[实战]、[教程]、[对比]、[深入]
- 包含长尾词：构建、教程、实战
- 避免：标题党、夸张用语

### 关键词布局
文章中自然出现：
- 主关键词：Aetheris、AI Agent、持久化
- 长尾词：崩溃恢复、幂等执行、Agent开发
- 关联词：LangChain、LangGraph、Go

### 链接策略
- 内部链接：指向其他相关文章
- 外部链接：官方文档、GitHub

---

## 写作规范

1. **段落长度**：每段不超过3-4句话
2. **代码块**：使用语法高亮，添加行号
3. **表格**：复杂对比使用表格展示
4. **图片**：添加流程图、架构图
5. **引用**：使用知乎的引用块功能
6. **emoji**：适度使用，增加可读性
