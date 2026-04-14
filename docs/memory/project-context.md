# Project Context

## 基本信息

| 字段 | 内容 |
|------|------|
| 项目名 | CoRag (Aetheris) |
| 模块 | rag-platform |
| 当前分支 | main |
| 当前任务 | architecture-analysis |

## 技术栈

| 类别 | 技术选型 |
|------|----------|
| 语言 | Go 1.26.1 |
| Agent 框架 | cloudwego/eino v0.7.29 |
| Web 框架 | cloudwego/hertz v0.10.4 |
| 数据库 | jackc/pgx/v5 (PostgreSQL) |
| 缓存 | redis/go-redis/v9 |
| 认证 | hertz-contrib/jwt |
| 可观测 | OpenTelemetry + Prometheus |

## 当前任务

**Task**: fix-review-issues
**Status**: completed
**Start Date**: 2026-04-05
**Phase**: 已完成 CRITICAL/HIGH 修复，MEDIUM 延后

## 已完成工作

| ID | 工作项 | 状态 |
|----|--------|------|
| SEC-01 | 移除硬编码凭证 | ✅ 完成 |
| SEC-02 | ListAgents/ListDocuments 租户过滤 | ✅ 完成 |
| SEC-03 | Document/Agent TenantID | ✅ 完成 |
| SEC-05 | auth disabled 降级改为 RoleUser | ✅ 完成 |
| SEC-06 | 增加启动时凭证检查 | ✅ 完成 |
| SEC-07 | JWT payload 增加 roles | ✅ 完成 |
| SEC-08 | 审计日志增加字段 (ClientIP/UserAgent/RequestID) | ✅ 完成 |
| RTN-01 | Heartbeat attempt_id 校验 | ✅ 完成 |
| RTN-02 | Worker 失租检测机制 | ✅ 完成 |
| RTN-03 | EffectStore 写入顺序修复 | ✅ 完成 |
| RTN-04 | EffectStore catch-up 逻辑验证 | ✅ 完成 |
| RTN-05 | Agent 并发模型决策 (Take/Release) | ✅ 完成 |
| RTN-06 | Agent/Manager 并发保护 | ✅ 完成 |
| RTN-07 | completedSet 同步问题 | ✅ 完成（无需修复） |
| RTN-08 | Checkpoint/Cursor 原子更新验证 | ✅ 完成 |
| RTN-09 | ListJobIDsWithExpiredClaim 过滤终态 | ✅ 完成 |

## 活跃风险

| 风险 | 影响 | 缓解 |
|------|------|------|
| 硬编码凭证 | ✅ 已移除 | 改为环境变量 |
| 跨租户泄露 | ✅ 已修复 | 添加 tenant 过滤 |
| Heartbeat 缺校验 | ✅ 已修复 | 添加 attempt_id 校验 |
| EffectStore 非原子 | ✅ 已修复 | 事件流优先 |
| Agent 并发竞争 | ✅ 已修复 | Take/Release 原子操作 |
| Worker 失租检测 | ✅ 已验证 | Heartbeat + Lease 机制 |

## 关键依赖

| 依赖 | 说明 |
|------|------|
| eino v0.7.29 | 核心编排引擎 |
| PostgreSQL | JobStore, EventStore |
| Redis | 缓存、分布式锁 |

## 下一步行动

| 优先级 | 任务 |
|--------|------|
| - | 全部任务已完成 |

## 已完成测试

| ID | 工作项 | 备注 |
|----|--------|------|
| TST-01 | Scheduler HA 集成测试 | 后台 agent 完成 |
| TST-02 | DAG E2E 测试 | TestDAGE2E_LLMToolChain, TestDAGE2E_ThreeNodeChain |
| TST-03 | EffectStore catch-up 逻辑测试 | 后台 agent 完成 |
| TST-04 | sleep 同步替换 | 发现问题，建议 channel-based start gate |
| TST-05 | Checkpoint 恢复后继续执行测试 | TestCheckpointRecovery_* |
| TST-06 | 内存 vs Postgres store 并发测试 | 后台 agent 完成 |

## 更新历史

| 日期 | 更新内容 | 更新人 |
|------|----------|--------|
| 2026-04-05 | architecture-analysis 任务启动 | tech-lead |
| 2026-04-05 | Phase 1 安全修复完成 (SEC-01~03, SEC-05~06) | team |
| 2026-04-05 | Phase 2 Runtime 完成 (RTN-01~06, RTN-08~09) | team |
| 2026-04-05 | SEC-07~08, TST-01~06 全部完成 | team |
| 2026-04-05 | project-review 审查完成 (43 issues) | team |
| 2026-04-05 | 新任务 fix-review-issues 启动 | tech-lead |
