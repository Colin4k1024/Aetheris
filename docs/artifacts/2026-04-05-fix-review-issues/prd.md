---
artifact: prd
task: fix-review-issues
date: 2026-04-05
role: tech-lead
status: draft
---

# 需求简报：修复项目审查发现的问题

## 背景

`project-review` 任务完成了系统性代码审查，发现 43 个问题（1 CRITICAL, 24 HIGH, 18 MEDIUM）。本任务旨在修复所有已识别的问题，提升代码质量和安全性。

## 目标与成功标准

| 目标 | 成功标准 |
|------|----------|
| 修复 CRITICAL 问题 | effect_adapter.go 调试语句移除 |
| 修复 HIGH 问题 | 所有 HIGH 问题已修复或有意忽略（有记录） |
| 修复 MEDIUM 问题 | 所有 MEDIUM 问题已修复或延后（有记录） |
| 通过 lint 检查 | golangci-lint 无 CRITICAL/HIGH 警告 |
| 测试通过 | 所有现有测试继续通过 |

## 范围

### In Scope

**CRITICAL (1):**
- 移除 effect_adapter.go 中的 fmt.Printf 调试语句

**HIGH (24):**
- 修复 Authorizator RBAC 绕过 (middleware.go:189-190)
- 添加 NewJWTAuth 空 key 验证
- 修复 context.Background() 热路径 (engine.go:134, job_runner.go:72)
- 添加 runner.go 错误日志
- 修复 effectsCtx 锁粒度 (effects/runtime.go:31-68)
- 修复 ToolRateLimiter 竞态 (rate_limiter.go:93-103)
- 修复其他被忽略的错误

**MEDIUM (18):**
- Watch goroutine 泄漏修复
- OIDC state 并发安全
- 登录速率限制
- 路径参数验证

### Out of Scope

- 重构（除非 CRITICAL/HIGH 需要）
- 新功能开发
- 架构变更

## 用户故事

- 作为维护者，我需要问题被修复，以便提升代码质量
- 作为安全负责人，我需要安全问题被修复
- 作为开发者，我需要热路径性能问题被修复

## 风险与依赖

| 风险 | 影响 | 缓解 |
|------|------|------|
| 修复引入新 bug | 测试回归 | 保持测试覆盖 |
| 修复工作量大 | 超时 | 按优先级分批 |
| RBAC 修复复杂 | 可能破坏现有功能 | 充分测试 |

## 待确认项

1. HIGH 问题是否可以分批修复？
2. 是否有已知风险需要回退方案？
3. 测试覆盖率是否足够？

## 参与角色

| 角色 | 输入 |
|------|------|
| backend-engineer | 代码修复、自测 |
| security-reviewer | 安全修复验证 |
| tech-lead | 验收、决策 |

## 企业治理

- 应用等级：内部工具
- 技术架构等级：标准
- 数据/合规风险：低

## 问题清单摘要

| 严重程度 | 数量 | 位置 |
|---------|------|------|
| CRITICAL | 1 | effect_adapter.go |
| HIGH | 24 | middleware, runner, engine, effects |
| MEDIUM | 18 | handler, oidc, memory_store |
| **总计** | **43** | |
