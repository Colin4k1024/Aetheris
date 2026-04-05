---
artifact: prd
task: architecture-analysis
date: 2026-04-05
role: tech-lead
status: active
---

# 需求简报：系统实现与架构合理性分析

## 1. 背景与目标

**需求背景：**
CoRag (Aetheris) 是一个面向 AI Agent 的持久化执行运行时，定位为"Temporal for Agents"。项目经历从 1.0 到 2.0 的演进，当前 main 分支已实现核心执行引擎、事件溯源、RAG Pipeline 和 MCP 工具桥接等能力。

**本次分析目标：**
对当前系统实现和架构设计进行系统性评估，识别架构合理性、潜在风险、改进机会和未完成的设计承诺。

**成功标准：**
- 输出结构化的架构评估报告
- 识别关键架构问题（如果有）
- 明确当前架构与设计文档的一致性
- 提出改进建议和后续行动

---

## 2. 参与角色

| 角色 | 主责 |
|------|------|
| `tech-lead` | 架构评估主责，输出评估报告 |
| `architect` | 深度架构审查（按需） |

---

## 3. 当前系统概述

### 3.1 项目定位
- **模块**：`rag-platform`
- **定位**：Agent Workflow Runtime（类比 Temporal）
- **核心能力**：任务编排、事件溯源、恢复与可观测
- **RAG 定位**：Pipeline/工具形式接入，默认可选

### 3.2 技术栈

| 类别 | 技术选型 | 评估 |
|------|----------|------|
| 语言 | Go 1.26.1 | ✅ 合理，Go 适合高并发和微服务 |
| Agent 框架 | cloudwego/eino v0.7.29 | ✅ 合理，作为核心编排引擎 |
| Web 框架 | cloudwego/hertz v0.10.4 | ✅ 合理，与 eino 同源生态 |
| 数据库 | jackc/pgx/v5 (PostgreSQL) | ✅ 合理，pgx 是 Go Postgres 最佳驱动 |
| 缓存 | redis/go-redis/v9 | ✅ 合理 |
| 认证 | hertz-contrib/jwt | ✅ 合理 |
| 可观测 | OpenTelemetry + Prometheus | ✅ 合理，标准可观测方案 |

### 3.3 包结构（internal/）

| 包 | 行数 | 职责 |
|----|------|------|
| `agent/` | ~18,716 | 核心执行引擎 |
| `agent/runtime/` | ~9,333 | DAG 编译器、Runner、状态管理 |
| `agent/job/` | ~2,222 | 事件溯源 Job 管理 |
| `agent/tools/` | ~1,496 | 工具注册、MCP 集成 |
| `agent/planner/` | ~527 | TaskGraph 生成 |
| `agent/memory/` | ~940 | Agent 记忆 |
| `api/` | ~9,610 | HTTP (Hertz) 和 gRPC API |
| `runtime/` | ~4,778 | Eino 集成、Job Store、Sessions |
| `app/` | ~4,336 | 应用层编排 |
| `pipeline/` | ~3,165 | RAG Pipeline（ingest/query） |
| `model/` | ~1,970 | LLM、Embedding、Vision 模型客户端 |
| `storage/` | ~1,532 | Vector、元数据、对象、缓存存储 |
| `tool/` | ~1,045 | 内置工具（LLM、RAG、HTTP、workflow） |

### 3.4 核心架构分层

```
┌─────────────────────────────────────────┐
│       Authoring Layer (Eino-first)      │
│   Eino Agent Construction                │
└─────────────────┬─────────────────────────┘
                  │
┌─────────────────▼─────────────────────────┐
│      Aetheris Control Plane              │
│   API / CLI / SDK Facade + Auth/RBAC   │
└─────────────────┬─────────────────────────┘
                  │
┌─────────────────▼─────────────────────────┐
│      Aetheris Data Plane (Runtime Core)  │
│  Scheduler │ Runner │ Tool Plane │ Replay │
└─────────────────┬─────────────────────────┘
                  │
┌─────────────────▼─────────────────────────┐
│           Durable Stores                  │
│  Event Store │ Checkpoint │ Effect │ Job   │
└───────────────────────────────────────────┘
```

---

## 4. 架构评估维度

### 4.1 设计文档 vs 实现一致性

**文档覆盖：**
- ✅ `design/core.md` — 总体架构（Eino-first + Runtime-first）
- ✅ `design/execution-guarantees.md` — 运行时保证契约
- ✅ `design/aetheris-2.0-overview.md` — 2.0 Roadmap 和模块结构
- ✅ `design/runtime-core-diagrams.md` — Runtime 流程图
- ✅ `design/internal/` — 大量内部设计文档（20+ 篇）

**关键设计承诺 vs 实现状态：**

| 设计承诺 | 实现状态 | 评估 |
|----------|----------|------|
| Eino 作为唯一核心编排引擎 | `AgentFactory` 基于 eino | ✅ 一致 |
| 事件溯源 JobStore | `internal/agent/jobstore/` | ✅ 一致 |
| Checkpoint + Replay | `internal/agent/replay/` | ✅ 一致 |
| Tool Bridge 抽象 | `internal/runtime/eino/tool_bridge.go` | ✅ 一致 |
| 配置驱动 Agent 构建 | `configs/agents.yaml` + `AgentFactory` | ✅ 一致 |
| 4 阶段 Roadmap | Phase 1-4 在 design docs 中定义 | ⚠️ 部分实现 |

### 4.2 架构合理性分析

**优点：**

1. **清晰的分离关注点**
   - Runtime 与 Agent 逻辑分离
   - Pipeline 作为可插拔节点
   - Tool Registry 通过 Bridge 抽象解耦

2. **配置驱动的 Agent 构建**
   - `AgentFactory` 提供统一入口
   - 避免硬编码，支持动态加载

3. **完善的可证明语义**
   - `execution-guarantees.md` 明确定义保证条件
   - Step 至少/至多执行一次、Signal 投递、Replay 确定性

4. **MCP 标准化接入**
   - Tool Bridge 支持 MCP 协议
   - 符合 industry trend

**潜在问题：**

1. **Legacy Agent 代码**
   - `internal/agent/agent.go` 标记为 Deprecated，但可能仍有引用
   - 需要确认旧路径是否已完全迁移

2. **双写模式**
   - Job 创建时"双写：事件流 + 状态型 Job"
   - 需要确认一致性保证和事务边界

3. **Effect Store 依赖**
   - "崩溃后不重复副作用"依赖 Effect Store 配置
   - 未配置时不保证，存在认知负担

4. **多 goroutine 安全性**
   - `internal/agent/runtime/` 中大量并发操作
   - 需要检查锁和竞态条件

### 4.3 安全性初步评估

| 方面 | 现状 | 评估 |
|------|------|------|
| 认证 | JWT 实现 | ✅ 基础可用 |
| RBAC | `design/enterprise/rbac.md` 定义 | ⚠️ 设计存在，实现待确认 |
| 审计 | `design/enterprise/audit.md` 定义 | ⚠️ 设计存在，实现待确认 |
| 多租户 | `design/enterprise/iam.md` 定义 | ⚠️ 设计存在，实现待确认 |

### 4.4 可观测性评估

| 方面 | 现状 | 评估 |
|------|------|------|
| Tracing | OpenTelemetry 集成 | ✅ 已实现 |
| Metrics | Prometheus 集成 | ✅ 已实现 |
| Logging | slog 集成 | ✅ 已实现 |
| Dashboard | - | ⚠️ 未实现（2.0 Phase 3） |

---

## 5. 待确认项

### 5.1 架构层面
- [ ] Legacy Agent (`internal/agent/agent.go`) 是否已完全弃用？
- [ ] 双写模式的一致性边界是否清晰？
- [ ] Effect Store 配置是否为必需项？
- [ ] 多 Worker 场景下的租约回收逻辑是否正确？

### 5.2 实现层面
- [ ] `AgentFactory` 的 Runner 缓存机制是否存在内存泄漏风险？
- [ ] Checkpoint 与 Event Store 的写入顺序是否正确？
- [ ] Session 传递通过 context 是否一致？

### 5.3 安全层面
- [ ] RBAC/审计/多租户的实现完成度？
- [ ] API 鉴权是否有完整的权限检查？
- [ ] 敏感数据是否正确处理？

### 5.4 测试层面
- [ ] 核心路径（Scheduler、Runner、Replay）的测试覆盖率？
- [ ] 竞态条件和并发场景是否有充分测试？
- [ ] Effect Store 的 catch-up 逻辑是否有测试？

---

## 6. 风险与约束

| 风险 | 影响 | 缓解建议 |
|------|------|----------|
| Legacy 代码路径未完全清理 | 维护负担，认知复杂性 | 完成迁移后删除旧代码 |
| Effect Store 依赖认知负担 | 配置错误导致生产问题 | 明确必需配置，简化默认值 |
| 缺乏 Dashboard | 可观测性不足 | 2.0 Phase 3 优先级提升 |

---

## 7. 后续行动建议

### 7.1 短期（1-2 周）
1. 完成 Legacy Agent 代码迁移确认
2. 检查 Effect Store 配置流程
3. 补充核心路径测试覆盖率
4. 确认 RBAC/审计实现状态

### 7.2 中期（1 个月）
1. 完成安全审计
2. 实现 Dashboard（如果 2.0 Phase 3 未开始）
3. 优化 Runner 缓存机制

### 7.3 长期
1. 按 2.0 Roadmap 完成剩余阶段
2. 完善多租户隔离
3. 性能优化和基准测试

---

## 8. 候选分组（需求挑战会）

建议以下分组参与架构评审：

| 分组 | 参与角色 | 议题 |
|------|----------|------|
| **核心 Runtime 组** | architect, backend-engineer | Scheduler、Runner、Replay 正确性 |
| **安全治理组** | security-reviewer, qa-engineer | RBAC、审计、多租户实现状态 |
| **测试覆盖组** | qa-engineer, backend-engineer | 覆盖率、并发测试、集成测试 |
| **文档对齐组** | tech-lead, architect | 设计 vs 实现一致性 |

---

## 9. 领域技能包启用建议

| 技能 | 触发原因 | 用途 |
|------|----------|------|
| `doc-architecture` | 架构文档补齐 | 收集 Project Profile Card |
| `security-reviewer` | 安全审计 | 鉴权、RBAC、审计实现评估 |
| `code-reviewer` | 代码质量审查 | Legacy 代码、并发安全 |

---

## 10. 分组讨论结论

### 10.1 核心 Runtime 组 — 评分 6.5/10

| 优先级 | 问题 | 位置 | 风险类型 |
|--------|------|------|----------|
| P0 | Heartbeat 不校验 attempt_id，导致租约混淆 | `pgstore.go:241-253` | 分布式正确性 |
| P0 | Effect Store 写入与事件流非原子，crash recovery 后 catch-up 不生效 | `node_adapter.go:711-746` | 数据一致性 |
| P0 | Agent/Manager 并发访问无锁保护 | `runtime/agent.go, manager.go` | 并发安全 |
| P1 | runParallelLevel 中 completedSet 并发写入无同步 | `runner.go:498-500` | 并发安全 |
| P1 | Checkpoint 与 Cursor 更新非原子 | `runner.go:509-520` | 状态一致性 |
| P2 | ListJobIDsWithExpiredClaim 不过滤终态 job | `pgstore.go:269-284` | 资源浪费 |
| P2 | 事件写入错误被静默忽略 | `runner.go:1437` 等 | 可观测性 |
| P3 | EffectStore 无 TTL/清理机制 | `effect_store_pg.go` | 存储膨胀 |

### 10.2 安全治理组 — 评分 4/10

| ID | 问题 | 严重度 | 分类 |
|----|------|--------|------|
| 1 | 硬编码凭证 (admin/admin, test/test) | CRITICAL | 认证 |
| 2 | ListAgents 无租户过滤，跨租户泄露 | CRITICAL | 多租户 |
| 3 | ListDocuments 无租户过滤 | CRITICAL | 多租户 |
| 4 | JWT 不含 roles/permissions | HIGH | RBAC |
| 5 | 审计日志缺少关键字段 (ClientIP, UserAgent, RequestID 等) | HIGH | 审计 |
| 6 | 无 Session 管理/吊销 | HIGH | 认证 |
| 7 | 租户配额未实现 | MEDIUM | 多租户 |
| 8 | 内存 RoleStore 不可用于生产 | MEDIUM | RBAC |

### 10.3 测试覆盖组 — 评分 5.5/10

| 维度 | 评分 | 说明 |
|------|------|------|
| 核心路径覆盖 | 5/10 | Runner/Executor 有基础测试，但 DAG 执行路径不完整 |
| 并发测试 | 5/10 | 有竞态测试，但 Scheduler 级别并发覆盖不足 |
| 集成测试 | 5/10 | 有 lifecycle/jobstore/session 集成测试，但缺少 E2E |
| 测试质量 | 6/10 | table-driven 和 mock 隔离好，但有 sleep-based 同步 |
| 错误路径 | 6/10 | 主要错误路径有覆盖 |

**关键缺失：**
- 真正的 Scheduler HA 测试（Lease 过期后多 worker 竞争）
- DAG 执行端到端路径（PlanGenerated → NodeFinished → JobCompleted）
- 内存 store 掩盖并发问题
- Checkpoint 恢复后继续执行的完整测试

### 10.4 设计一致性组 — 评分 8.5/10

**设计承诺与实现高度一致 (95%+)**，核心执行保证均已正确实现：

| 承诺 | 状态 |
|------|------|
| Eino 作为唯一编排引擎 | ✅ 已实现 |
| AgentFactory 统一构建入口 | ✅ 已实现 |
| RegistryToolBridge 工具抽象 | ✅ 已实现 |
| 事件溯源 JobStore | ✅ 已实现 |
| Checkpoint + Replay | ✅ 已实现 |
| InvocationLedger (at-most-once) | ✅ 已实现 |
| Signal 持久化收件箱 | ✅ 已实现 |
| GET /api/jobs/:id/verify | ✅ 已实现 |

**问题：**
- `design/v2.md` 和 `design/milestone.md` 为对话文本而非设计文档，建议清理
- `docs/2.0-capability-matrix.md` 被引用但不存在

---

## 11. 综合结论

### 11.1 整体评级

| 评估维度 | 评分 | 趋势 |
|----------|------|------|
| **架构合理性** | 7/10 | 核心分层清晰，技术选型适当 |
| **安全治理** | 4/10 | ⚠️ 存在多个阻断性问题 |
| **测试覆盖** | 5.5/10 | 基础覆盖存在，深度不足 |
| **设计一致性** | 8.5/10 | ✅ 高度一致 |
| **整体** | **6.25/10** | 中等偏上 |

### 11.2 阻断性问题（必须修复）

1. **CRITICAL - 安全**
   - 移除硬编码凭证
   - 修复 ListAgents/ListDocuments 租户过滤

2. **CRITICAL - Runtime**
   - Heartbeat 添加 attempt_id 校验
   - Effect Store 写入纳入事务或改用事件流优先

3. **CRITICAL - 并发**
   - Agent/Manager 添加锁保护

### 11.3 短期行动项（1-2 周）

| 优先级 | 行动 | 主责 |
|--------|------|------|
| P0 | 移除硬编码凭证，强制配置化 | backend-engineer |
| P0 | 修复 ListAgents/ListDocuments 租户过滤 | backend-engineer |
| P0 | Heartbeat 添加 attempt_id 校验 | backend-engineer |
| P1 | Agent/Manager 并发访问加锁 | backend-engineer |
| P1 | Checkpoint 和 Cursor 原子更新 | backend-engineer |
| P2 | 增强审计日志字段 | backend-engineer |
| P2 | 增加 Scheduler HA 集成测试 | qa-engineer |

### 11.4 中期行动项（1 个月）

| 优先级 | 行动 | 主责 |
|--------|------|------|
| P1 | 实现 JWT roles/permissions 声明 | backend-engineer |
| P2 | 实现租户配额控制 | backend-engineer |
| P2 | 清理对话文本设计文档 | tech-lead |
| P2 | 补充 DAG E2E 测试 | qa-engineer |
| P3 | EffectStore TTL 清理机制 | backend-engineer |
| P3 | 用 channel 替换 sleep 同步 | backend-engineer |

### 11.5 后续建议

**建议进入 `/team-plan`** 进行系统性修复，按以下顺序：

1. **Phase 1**: 安全修复（阻断发布）
2. **Phase 2**: Runtime 正确性修复（P0 并发问题）
3. **Phase 3**: 测试覆盖增强
4. **Phase 4**: 文档治理和长期优化
