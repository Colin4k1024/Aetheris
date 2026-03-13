# Aetheris: 让 AI Agent 从"玩具"变成"生产系统"

> Kubernetes 管理容器，Aetheris 管理 AI Agent

## 🔥 你是否遇到过这些场景？

你的 AI Agent 在测试环境运行完美，一上线就各种问题：

- ❌ **Worker 崩溃** → 任务要从头开始，用户等待时间翻倍
- ❌ **工具被调用两次** → 重复支付、重复发送邮件、客户投诉
- ❌ **想回溯 AI 的决策** → 没有任何痕迹，完全无法审计
- ❌ **等待人工审批时** → 持续占用资源，成本飙升
- ❌ **想复现问题** → 根本不可能

**这是大多数 AI Agent 部署的真实写照。**

## 🛠️ Aetheris 是什么？

**Aetheris = AI Agent 的生产运行时**

它不是：
- ❌ 聊天机器人框架
- ❌ Prompt 模板库
- ❌ RAG 检索系统
- ❌ 用来"写"Agent 的工具（这些是 LangChainGo、LangGraphGo、Google ADK 做的事）

它是一个**Agent 执行运行时**——帮你把用 LangChainGo/LangGraphGo 构建的 Agent 跑起来，并且：
- ✅ **持久化** — 崩溃后从检查点恢复
- ✅ **幂等性** — 工具调用绝不重复（At-Most-Once）
- ✅ **可回放** — 任意时刻的运行都能复现
- ✅ **可审计** — 完整决策链，证据链可追溯

## ✨ 核心特性

| 特性 | 意味着什么 |
|------|----------|
| **🛡️ At-Most-Once 执行** | 工具调用绝不重复，崩溃也不重复 |
| **💥 崩溃恢复** | 从检查点恢复，不是从头开始 |
| **🔄 确定性回放** | 任意运行都能复现，用于调试或审计 |
| **👤 人机协作** | 暂停等待审批，不占用资源 |
| **📋 完整审计** | 谁、什么、何时、为什么 — 全部记录 |
| **🔌 多框架支持** | LangChainGo、LangGraphGo、Google ADK 都能接入 |

## 🚀 5 分钟快速开始

```bash
# 安装 CLI
go install github.com/Colin4k1024/Aetheris/cmd/cli@latest

# 初始化项目
aetheris init my-agent

# 运行
cd my-agent
aetheris run

# 监控
aetheris jobs list
```

或者用 Docker：

```bash
./scripts/local-2.0-stack.sh start
curl http://localhost:8080/api/health
```

## 📊 三大核心场景

### 1. 人机协作流程
> 退款审批、合同审批、客服升级 — Agent 暂停等待人工确认后自动继续

### 2. 长时 API 编排
> 数据同步、支付处理、多步骤流水线 — 50+ API 调用，保证不重复

### 3. 可审计决策系统
> 贷款审批、处方开具、合规判断 — 监管机构可追溯每一个决策

## 🔗 接入现有 Agent

已经用 LangChainGo/LangGraphGo 写了 Agent？直接迁移：

```python
from aetheris import AetherisRuntime

runtime = AetherisRuntime()
job = runtime.submit(graph=your_langgraph_agent, input={"query": "..."})
# 现在它有了持久化、可恢复、可审计的能力
```

## 📈 对比

| 问题 | 不用 Aetheris | 用 Aetheris |
|------|--------------|-------------|
| Worker 崩溃 | 从头开始 | 从检查点恢复 |
| 重复调用工具 | 可能（造成损失） | 绝对保证不重复 |
| 调试失败运行 | 靠猜 | 确定性回放 |
| 审计 AI 决策 | 不可能 | 完整证据链 |
| 等待审批时 | 持续占用资源 | 暂停，不浪费 |

## 🌍 社区

- 💬 [Discord](https://discord.gg/PrrK2Mua)
- 💬 [GitHub Discussions](https://github.com/Colin4k1024/Aetheris/discussions)
- 📖 [文档](docs/)

---

**如果 Aetheris 帮助你构建了生产级 Agent，点个 ⭐ 支持一下！**

GitHub: https://github.com/Colin4k1024/Aetheris
