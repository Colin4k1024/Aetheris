---
artifact: handoff
task: project-review
date: 2026-04-05
role: tech-lead
status: draft
---

# 交接：tech-lead → backend-engineer

## 背景

`project-review` 任务已通过 `/team-intake` 和 `/team-plan`，现交接给 backend-engineer 执行代码审查。

## 输入依据

- `docs/artifacts/2026-04-05-project-review/prd.md` - 需求简报
- `docs/artifacts/2026-04-05-project-review/delivery-plan.md` - 交付计划
- `docs/memory/project-context.md` - 项目上下文
- 近期完成的 `architecture-analysis` 任务（SEC-01~08, RTN-01~09, TST-01~06）

## 结论

已完成：
- ✅ Intake 和 Plan 阶段
- ✅ 审查范围定义（代码质量、安全、并发、错误处理、可观测性）
- ✅ 优先级排序（Phase 1 代码质量 → Phase 2 安全 → Phase 3 运行时）

待执行：
- ⏳ Phase 1: Go 最佳实践检查
- ⏳ Phase 1: 错误处理一致性检查
- ⏳ Phase 1: 日志和可观测性检查
- ⏳ Phase 3: 并发模型审查
- ⏳ Phase 3: 性能瓶颈识别

## 风险

| 风险 | 影响 | 缓解 |
|------|------|------|
| 审查范围过大 | 超时 | 按优先级分批处理 |
| 发现过多问题 | 影响发布 | 分类管理，严重优先 |

## 待确认项

1. 审查优先级排序是否合理？
2. 是否有特定代码区域需要重点审查？
3. 是否需要 security-reviewer 参与安全审查？

## 下一跳角色

**backend-engineer**

### 期望产出

1. 代码质量问题清单（HIGH/CRITICAL 优先）
2. 并发模型审查报告
3. 错误处理一致性报告
4. 日志和可观测性评估报告

### 执行步骤

1. 使用 `golang/coding-style` 规则检查代码
2. 检查 `internal/agent/runtime/` 并发实现
3. 检查 `internal/api/` 错误处理模式
4. 检查日志和 metrics 埋点
5. 输出结构化问题清单

## 当前阶段

- **当前阶段:** plan
- **目标阶段:** execute
- **就绪状态:** ready-for-review

## 下游质疑记录

> 接收方尚未填写，须先提出至少 1 条对上游输入的质疑。

---

## 执行记录（由 backend-engineer 填写）

### 下游质疑

- 质疑内容：
- 质疑目标：
- 结论：
- 处理说明：

### 实际执行

- 开始时间：
- 结束时间：
- 执行的检查项：

### 发现的问题

| 严重程度 | 问题描述 | 位置 | 建议修复 |
|----------|----------|------|----------|
| | | | |

### 影响面

- 涉及的模块：
- 涉及的接口：
- 涉及的数据：

### 未完成项

- |
