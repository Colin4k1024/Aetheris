---
artifact: launch-acceptance
task: fix-review-issues
date: 2026-04-05
role: qa-engineer
status: draft
---

# 上线验收：修复项目审查发现的问题

## 验收概览

| 字段 | 内容 |
|------|------|
| 任务 | fix-review-issues |
| 验收日期 | 2026-04-05 |
| 验收角色 | qa-engineer |
| 验收方式 | 代码审查 + 测试验证 |

## 验收范围

### 已修复的问题

| 严重程度 | 问题 | 位置 | 修复方式 |
|---------|------|------|----------|
| CRITICAL | debug 语句规范化 | effect_adapter.go | slog.DebugContext |
| HIGH | RBAC 授权绕过 | middleware.go | 检查 Roles 非空 |
| HIGH | 空 key 验证 | middleware.go | ErrMissingSecretKey |
| HIGH | effectsCtx 锁优化 | effects/runtime.go | I/O 移到锁外 |
| HIGH | rate_limiter 竞态 | rate_limiter.go | 双检锁定 |
| HIGH | context 传递 | engine.go | 传递 ctx 参数 |
| HIGH | checkpoint 错误日志 | runner.go | slog.Error |

### 未修复的问题 (延后)

| 严重程度 | 问题 | 风险评估 |
|---------|------|----------|
| MEDIUM | Watch goroutine 泄漏 | 可接受，当前实现有 defer close |
| MEDIUM | OIDC state 并发 | 可接受，非关键路径 |
| MEDIUM | 登录速率限制 | 可接受，内部使用 |
| MEDIUM | 路径参数验证 | 可接受，上层有验证 |

## Go / No-Go 检查项

### Go 检查项 ✅

- [x] go build ./... 通过
- [x] go test ./... 全部通过 (40+ packages)
- [x] go test -race ./... 通过
- [x] RBAC 逻辑变更有测试覆盖
- [x] 并发修复有竞态测试
- [x] 错误日志有日志输出

### No-Go 检查项

- [ ] 无阻塞项

## 已接受风险

1. MEDIUM 问题延后处理（见上表）
2. agent_dag.go 的 context.Background() 未修复（不在热路径）

## 上线结论

| 结论 | 说明 |
|------|------|
| **允许上线** | 所有 CRITICAL/HIGH 问题已修复，测试通过 |

## 风险判断

| 风险项 | 状态 | 说明 |
|--------|------|------|
| RBAC 修复 | ✅ 已验证 | 检查 Roles 非空 |
| 并发修复 | ✅ 已验证 | race 测试通过 |
| 安全修复 | ✅ 已验证 | 空 key 校验 |

## 后续行动

| 优先级 | 行动 | Owner |
|--------|------|-------|
| LOW | 后续 sprint 处理 MEDIUM 问题 | backend |
