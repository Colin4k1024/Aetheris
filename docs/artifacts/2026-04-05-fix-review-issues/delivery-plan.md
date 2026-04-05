---
artifact: delivery-plan
task: fix-review-issues
date: 2026-04-05
role: tech-lead
status: draft
---

# 交付计划：修复项目审查发现的问题

## 版本目标

| 版本 | 范围 | 放行标准 |
|------|------|----------|
| v1 | CRITICAL + HIGH | golangci-lint 无 CRITICAL/HIGH |
| v2 | MEDIUM | 所有问题已处理 |

## 工作拆解

### Sprint 1: CRITICAL + HIGH (Day 1)

| 工作项 | 主责 | 依赖 | 预估时间 |
|--------|------|------|----------|
| 移除 fmt.Printf 调试语句 | backend | - | 15min |
| 修复 Authorizator RBAC | backend | - | 30min |
| 添加空 key 验证 | backend | - | 15min |
| 修复 context.Background() | backend | - | 30min |
| 添加错误日志 | backend | - | 1h |
| 修复 effectsCtx 锁 | backend | - | 1h |
| 修复 ToolRateLimiter 竞态 | backend | - | 30min |

### Sprint 2: MEDIUM (Day 2)

| 工作项 | 主责 | 依赖 | 预估时间 |
|--------|------|------|----------|
| Watch goroutine 泄漏修复 | backend | Sprint 1 | 30min |
| OIDC state 并发安全 | backend | Sprint 1 | 30min |
| 登录速率限制 | backend | Sprint 1 | 30min |
| 路径参数验证 | backend | Sprint 1 | 1h |

## 需求挑战会结论

1. **假设：问题修复不会引入新 bug**
   - 质疑：effectsCtx 锁重构是否会影响现有逻辑？
   - 替代：保留原有锁结构，仅移动 I/O 操作
   - 结论：充分测试可验证

2. **假设：所有 HIGH 问题必须立即修复**
   - 质疑：是否有可以接受的风险而不修复的问题？
   - 替代：按风险评估决定是否修复
   - 结论：所有 HIGH 必须修复

3. **假设：测试覆盖足以验证修复**
   - 质疑：现有测试是否足够覆盖并发场景？
   - 替代：添加并发压力测试
   - 结论：可先修复，再补充测试

## 风险与缓解

| 风险 | 影响 | 缓解措施 | Owner |
|------|------|----------|-------|
| effectsCtx 重构复杂 | 可能引入 bug | 充分测试 | backend |
| RBAC 修复破坏功能 | 功能回归 | 端到端测试 | backend |
| 并发修复引入死锁 | 系统卡死 | 压力测试 | backend |

## 节点检查

| 阶段 | 完成标准 | 验证方式 |
|------|----------|----------|
| Sprint 1 | 所有 CRITICAL/HIGH 修复 | golangci-lint, go test |
| Sprint 2 | 所有 MEDIUM 修复 | 代码审查 |

## 角色分工

| 角色 | 职责 |
|------|------|
| backend-engineer | 执行所有修复 |
| security-reviewer | 验证 RBAC 修复 |
| tech-lead | 验收 |

## 技能装配

- `golang/coding-style` - 代码修复指导
- `golang/testing` - 测试覆盖
- `security-reviewer` - 安全修复验证

## 无需 ADR

此任务只修复问题，不涉及架构决策。
