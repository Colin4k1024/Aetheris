---
artifact: delivery-plan
task: architecture-analysis
date: 2026-04-05
role: tech-lead
status: ready-for-execute
---

# 交付计划：架构分析与修复

## 1. 版本目标

| 项目 | 说明 |
|------|------|
| 版本 | v0.2.0 (补丁版本) |
| 范围 | 安全修复 + Runtime 正确性修复 + 测试增强 |
| 放行标准 | 所有 CRITICAL/P0 问题已修复，回归测试通过 |

---

## 2. 工作拆解

### Phase 1: 安全修复（阻断发布）

| ID | 工作项 | 主责 | 依赖 | 优先级 |
|----|--------|------|------|--------|
| SEC-01 | 移除硬编码凭证，改为环境变量读取 | backend-engineer | 无 | P0 |
| SEC-02 | 修复 ListAgents/ListDocuments 租户过滤 | backend-engineer | SEC-01 | P0 |
| SEC-03 | Document/Agent 数据模型增加 TenantID 字段 | backend-engineer | SEC-02 | P1 |
| SEC-04 | RoleStore 配置与 JobStore 解耦 | backend-engineer | 无 | P1 |
| SEC-05 | auth disabled 降级改为 RoleUser | backend-engineer | SEC-01 | P1 |
| SEC-06 | 增加启动时凭证存在性检查 | backend-engineer | SEC-01 | P1 |
| SEC-07 | JWT payload 增加 roles/permissions 声明 | backend-engineer | SEC-04 | P2 |
| SEC-08 | 审计日志增加 ClientIP/UserAgent/RequestID | backend-engineer | 无 | P2 |

### Phase 2: Runtime 正确性修复

| ID | 工作项 | 主责 | 依赖 | 优先级 |
|----|--------|------|------|--------|
| RTN-01 | Heartbeat SQL 增加 attempt_id 校验 | backend-engineer | 无 | P0 |
| RTN-02 | 明确 Worker 失租检测机制 | architect | RTN-01 | P0 |
| RTN-03 | EffectStore 写入顺序修复（事件流优先） | backend-engineer | 无 | P0 |
| RTN-04 | 确认/实现 EffectStore catch-up 逻辑 | backend-engineer | RTN-03 | P0 |
| RTN-05 | Agent 并发模型明确（Take/Release 语义） | architect | 无 | P0 |
| RTN-06 | Agent/Manager 字段并发保护 | backend-engineer | RTN-05 | P1 |
| RTN-07 | completedSet 同步问题修复 | backend-engineer | 无 | P1 |
| RTN-08 | Checkpoint/Cursor 原子更新 | backend-engineer | 无 | P1 |
| RTN-09 | ListJobIDsWithExpiredClaim 过滤终态 | backend-engineer | 无 | P2 |

### Phase 3: 测试增强

| ID | 工作项 | 主责 | 依赖 | 优先级 |
|----|--------|------|------|--------|
| TST-01 | Scheduler HA 集成测试（多 worker 竞争） | qa-engineer | RTN-02 | P1 |
| TST-02 | DAG E2E 测试（PlanGenerated→JobCompleted） | qa-engineer | RTN-03 | P1 |
| TST-03 | EffectStore catch-up 逻辑测试 | qa-engineer | RTN-04 | P1 |
| TST-04 | 用 channel 替换 sleep 同步 | backend-engineer | 无 | P2 |
| TST-05 | Checkpoint 恢复后继续执行测试 | qa-engineer | RTN-08 | P2 |
| TST-06 | 内存 store vs Postgres store 并发差异测试 | qa-engineer | 无 | P2 |

---

## 3. 风险与缓解

| 风险 | 影响 | 缓解措施 | Owner |
|------|------|----------|-------|
| Schema 迁移破坏现有数据 | 高 | 先在测试环境验证，提供回滚脚本 | backend-engineer |
| EffectStore 顺序修改影响现有流程 | 高 | 充分测试，确保 backward compatible | backend-engineer |
| 测试环境与生产环境行为差异 | 中 | 使用 testcontainers 进行隔离测试 | qa-engineer |
| 时间窗口不足 | 中 | 按优先级排序，必要时拆分版本 | tech-lead |

---

## 4. 节点检查

| 节点 | 标准 | 预期时间 |
|------|------|----------|
| 方案评审 | Phase 1+2 所有 P0 问题有明确修复方案 | Day 1 |
| Phase 1 完成 | SEC-01~06 代码审查通过 | Day 3 |
| Phase 2 完成 | RTN-01~06 代码审查通过 | Day 5 |
| Phase 3 完成 | 核心测试覆盖率提升至 70%+ | Day 7 |
| 发布准备 | 回归测试通过，安全扫描通过 | Day 10 |

---

## 5. 需求挑战会结论

### 安全修复关键发现

1. **SEC-01 不足**：移除硬编码凭证必须配套启动时凭证检查，否则开发便利性丧失但安全性未真正提升
2. **SEC-02 范围**：ListAgents/ListDocuments 只是租户泄露的冰山一角，Document/Agent 数据模型本身缺少 TenantID 字段
3. **SEC-04 架构缺陷**：RoleStore 配置与 JobStore 类型耦合，导致生产可能悄悄使用 MemoryRoleStore

### Runtime 修复关键发现

1. **RTN-01 不充分**：Heartbeat 加 attempt_id 校验是必要条件，但不充分。需要明确 Worker 失租检测机制
2. **RTN-03 顺序错误**：当前 Phase 1 写 EffectStore、Phase 2 写事件流的顺序与 design doc "事件流优先" 原则矛盾
3. **RTN-05 并发模型缺失**：Agent 是否允许多 goroutine 并发调用未明确，需要 architect 决策

### 测试增强关键发现

1. **TST-01~03 依赖架构修复**：测试增强依赖 Phase 1+2 的架构修复完成
2. **TST-04 不只是替换**：channel 方案在所有场景是否最优需要验证

---

## 6. 角色分工

| 角色 | 主责任务 |
|------|----------|
| `tech-lead` | 整体协调，方案评审，发布决策 |
| `architect` | RTN-02, RTN-05 架构决策 |
| `backend-engineer` | SEC-01~08, RTN-01, RTN-03~04, RTN-06~09, TST-04 |
| `qa-engineer` | TST-01~03, TST-05~06 |

---

## 7. 后续行动

1. **Day 1**: SEC-01, RTN-01, RTN-05 并行启动
2. **Day 3**: Phase 1 code review + RTN-02 决策
3. **Day 5**: Phase 2 code review + RTN-03~04 验证
4. **Day 7**: Phase 3 测试完成
5. **Day 10**: 回归测试 + 发布

---

## 8. ADR 需求

建议新增以下 ADR：

| ADR | 标题 | 决策点 |
|-----|------|--------|
| ADR-XXX | Agent 并发模型决策 | Agent 是否允许多 goroutine 并发 Run()？ |
| ADR-XXX | EffectStore 与事件流写入顺序 | 事件流优先 vs EffectStore 优先？ |
| ADR-XXX | RoleStore 配置解耦 | RoleStore 类型是否应与 JobStore 解耦？ |
