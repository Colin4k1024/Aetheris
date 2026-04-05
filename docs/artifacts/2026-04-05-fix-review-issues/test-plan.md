---
artifact: test-plan
task: fix-review-issues
date: 2026-04-05
role: qa-engineer
status: draft
---

# 测试计划：修复项目审查发现的问题

## 测试范围

| 模块 | 测试类型 | 覆盖内容 |
|------|----------|----------|
| middleware | 安全回归 | RBAC 授权、JWT 认证 |
| effects | 并发测试 | effectsCtx Now/UUID 锁 |
| rate_limiter | 竞态测试 | addToolLimiter 双检锁定 |
| engine | 单元测试 | context 传递 |
| runner | 错误日志 | checkpoint save 失败日志 |

## 测试矩阵

| 场景 | 类型 | 前置条件 | 预期结果 |
|------|------|----------|----------|
| Authorizator 无角色用户 | 安全 | 用户 Roles=[] | 返回 false |
| Authorizator 有角色用户 | 安全 | 用户 Roles=["user"] | 返回 true |
| NewJWTAuth 空 key | 安全 | key=[] | 返回 ErrMissingSecretKey |
| Now() 并发调用 | 并发 | 10 goroutines 同时调用 | 无死锁、无数据竞争 |
| UUID() 并发调用 | 并发 | 10 goroutines 同时调用 | 无死锁、无数据竞争 |
| Wait() 双检锁定 | 并发 | 100 goroutines 竞争 | 仅 3 个获得锁 |
| engine.GetRunner ctx | 单元 | 传入 ctx | ctx 正确传递 |
| checkpoint save 失败 | 错误日志 | save 返回 error | slog.Error 被调用 |

## 风险评估

| 风险 | 影响 | 缓解措施 |
|------|------|----------|
| effectsCtx 锁重构 | 低 | 并发测试覆盖 |
| RBAC 逻辑变更 | 中 | 认证测试覆盖 |
| 双检锁定实现 | 中 | 竞态测试覆盖 |

## 放行建议

- [ ] go build ./... 通过
- [ ] go test ./... 通过
- [ ] go test -race ./... 通过
- [ ] 安全测试通过
- [ ] 并发测试通过
