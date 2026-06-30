# Delivery Plan: Aetheris 战略推进 Epic (Q3 2026)

> **来源**: Issue #220 + Requirement Challenge Session
> **状态**: draft
> **主责**: tech-lead
> **日期**: 2026-06-30
> **关联 PRD**: [prd.md](./prd.md)

---

## 1. Requirement Challenge Session 结论

### 核心质疑与仲裁

| # | 质疑 | 来源 | 仲裁决策 |
|---|------|------|----------|
| Q1 | crash_recovery demo 是 external_http 边界级，非 step-level，GIF 会误导 | product-mgr | **采用"worker crash + restart"方案**：演示 worker 崩溃后 job 自动恢复，这是真实可验证的能力。step-level demo 移至 v2.7.0 |
| Q2 | 100 并发压测无基础设施、无 pass/fail 标准 | project-mgr | **拆为两阶段**：先 Go benchmark 验证单组件上限，再 k6 端到端压测。首次目标 50 并发建立基线 |
| Q3 | Grafana dashboard 引用不存在的 metrics，截图将显示 "No data" | architect | **P1a 必须先修复**：以 `corag-dashboard.json`（provisioning 目录，指标名正确）为权威源，合并 `aetheris-dashboard.json` 的面板设计，修复指标名 |
| Q4 | Worker 进程无 OTel 初始化，Jaeger 看不到执行链路 | architect | **P1a 必须先修复**：为 Worker 添加 OTel 初始化，复用 API Server 的 provider 模式 |
| Q5 | 6+ 独立 pgxpool.Pool + Append() 非原子 CAS | architect | **记录为已知技术债**，压测后根据数据决定是否修复。不阻塞首轮压测 |
| Q6 | hermesx 仓库零基础，4 周不现实 | 全员 | **降级为 ADR 占位**：产出架构愿景文档 + Aetheris 侧 Provider Interface，不做跨团队对齐 |
| Q7 | 博客 90% 完成但渠道未定是决策阻塞 | project-mgr | **Day 1 锁定**：GitHub Pages 主发 + dev.to 交叉发布 |
| Q8 | P1→P2 串行依赖是人为假设 | project-mgr | **P2 与 P1a 并行**：GIF 录制只需本地开发环境 |

### 未决项

| # | 问题 | 默认决策 | 需要确认 |
|---|------|----------|----------|
| U1 | 压测环境：本地 Docker 还是独立服务器？ | 本地 Docker（开发机 16GB+ RAM） | devops-engineer |
| U2 | VHS 还是 terminalizer 录制 GIF？ | VHS（声明式可复现） | 执行者自选 |
| U3 | Go benchmark 是否需要合入 main？ | 放 `benchmarks/` 目录，不进 CI | tech-lead |

---

## 2. 版本目标

**版本**: v2.6.0（暂定）
**范围**: Issue #220 四优先级，经 challenge 后收敛
**放行标准**:
- Grafana dashboard 展示真实指标数据（非 "No data"）
- Jaeger 展示完整 job 执行链路（含 worker 侧 span）
- 50 并发压测报告产出（含 P95/P99 延迟、吞吐量、错误率）
- Crash Recovery GIF 嵌入 README 顶部
- 博客已发布至少 1 篇
- routing-advisor 设计文档完善至可对外展示

---

## 3. 工作拆解

### Phase 1: 观测补齐 + Demo（Week 1）

| # | 工作项 | 主责 | 依赖 | 产出 | 验收标准 |
|---|--------|------|------|------|----------|
| 1.1 | **Grafana Dashboard 统一** | devops-engineer | 无 | `deployments/compose/grafana/provisioning/dashboards/aetheris-dashboard.json` | 面板引用的 metrics 全部在 `pkg/metrics/metrics.go` 中注册；`docker compose up` 后 Grafana 面板有数据 |
| 1.2 | **Worker OTel 初始化** | backend-engineer | 无 | `internal/app/worker/app.go` 变更 | Worker 启动后 Jaeger 能展示 job→node→tool 三级 span |
| 1.3 | **e2e 可观测性验收** | devops-engineer | 1.1, 1.2 | 验收文档 + Grafana 截图 + Jaeger 截图 | compose 启动→提交 job→Grafana 有数据→Jaeger 有完整 trace |
| 1.4 | **Crash Recovery GIF** | tech-lead | 无 | `docs/assets/crash-recovery.gif` | VHS 录制：submit job → kill worker → restart → "job resumed"；≤30s；README 可渲染 |
| 1.5 | **博客发布** | tech-lead | 无 | GitHub Pages + dev.to | `docs/blog/11-why-agents-need-runtime.md` 正式发布 |

### Phase 2: 压测基础设施（Week 2）

| # | 工作项 | 主责 | 依赖 | 产出 | 验收标准 |
|---|--------|------|------|------|----------|
| 2.1 | **Go Benchmark: pgStore** | backend-engineer | 无 | `internal/runtime/jobstore/pgstore_bench_test.go` | `go test -bench=BenchmarkAppend -benchtime=10s` 产出吞吐数据 |
| 2.2 | **Go Benchmark: Claim/Scheduler** | backend-engineer | 无 | `internal/runtime/jobstore/scheduler_bench_test.go` | 50 并发 Claim 的 ops/sec 和 p99 |
| 2.3 | **k6 压测脚本** | devops-engineer | 无 | `benchmarks/k6/load-test.js` | 覆盖：POST /api/agents/{id}/message + GET /api/jobs/{id} + GET /api/jobs/{id}/events |
| 2.4 | **连接池配置优化** | backend-engineer | 2.1 结果 | 配置文档或代码变更 | 如 benchmark 暴露连接瓶颈，统一连接池或调整 MaxConns |

### Phase 3: 压测执行 + 问题修复（Week 3）

| # | 工作项 | 主责 | 依赖 | 产出 | 验收标准 |
|---|--------|------|------|------|----------|
| 3.1 | **50 并发基线测试** | devops-engineer | 2.3 | 压测报告 `benchmarks/reports/baseline-50.md` | P95 < 500ms, P99 < 2000ms, 成功率 > 99%, 内存增长 < 100MB |
| 3.2 | **性能问题修复** | backend-engineer | 3.1 | 代码变更 | 如 Append() CAS 重试风暴或连接池耗尽，修复后回归 |
| 3.3 | **100 并发压测** | devops-engineer | 3.2 | 压测报告 `benchmarks/reports/load-100.md` | 通过标准同 3.1 |
| 3.4 | **routing-advisor 文档完善** | architect | 无 | 更新 `docs/artifacts/2026-05-26-routing-advisor-contract/` | 设计文档可对外展示，状态从 experimental 更新为 draft |

### Phase 4: 收口 + 发布准备（Week 4）

| # | 工作项 | 主责 | 依赖 | 产出 | 验收标准 |
|---|--------|------|------|------|----------|
| 4.1 | **README 更新** | tech-lead | 1.3, 1.4, 3.3 | README.md | 顶部有 GIF；有 "Verified Benchmarks" 章节（Grafana 截图 + 压测数据链接） |
| 4.2 | **hermesx ADR** | architect | 无 | `docs/adr/ADR-001-hermesx-bridge-intent.md` | 声明四层架构定位 + Aetheris 侧 Provider Interface |
| 4.3 | **awesome-ai-agents PR 跟进** | tech-lead | 无 | PR 状态更新 | 已 ping maintainer；2 周 SLA 计时 |
| 4.4 | **Release Notes** | tech-lead | 4.1 | `docs/releases/v2.6.0.md` | 包含：可观测性验收结果、压测数据、GIF、博客链接 |

---

## 4. Story Slice 列表

每个 slice 为可独立执行的最小交付单元：

| Slice | 目标 | Owner | Handoff 终点 | 依赖 |
|-------|------|-------|-------------|------|
| S1: Grafana Dashboard 修复 | 指标名对齐，面板有数据 | devops-engineer | PR 合入 + 截图证据 | 无 |
| S2: Worker OTel 初始化 | Worker 产生有效 trace span | backend-engineer | PR 合入 + Jaeger 截图 | 无 |
| S3: e2e 验收 | 端到端可观测性验证 | devops-engineer | 验收文档 | S1, S2 |
| S4: Crash Recovery GIF | README 顶部可展示的 demo | tech-lead | GIF 文件 + README 嵌入 | 无 |
| S5: 博客发布 | 至少 1 篇博客正式发布 | tech-lead | 发布 URL | 无 |
| S6: pgStore Benchmark | Append/Claim 单组件吞吐基线 | backend-engineer | benchmark 结果 | 无 |
| S7: k6 压测脚本 | 端到端并发压测能力 | devops-engineer | 可运行脚本 | 无 |
| S8: 50 并发基线 | 第一份压测报告 | devops-engineer | 报告文档 | S6, S7 |
| S9: 性能修复 | 修复压测暴露的问题 | backend-engineer | PR 合入 + 回归通过 | S8 |
| S10: 100 并发压测 | 最终压测报告 | devops-engineer | 报告文档 | S9 |
| S11: routing-advisor 完善 | 设计文档可对外展示 | architect | 更新文档 | 无 |
| S12: hermesx ADR | 架构愿景 + 单侧契约 | architect | ADR 文档 | 无 |
| S13: README + Release Notes | 最终发布准备 | tech-lead | 更新文档 | S3, S4, S10 |

---

## 5. 角色分工

| 角色 | 职责范围 | 负责 Slices |
|------|----------|-------------|
| **tech-lead** | intake、仲裁、收口、GIF 录制、博客发布、README | S4, S5, S13 |
| **devops-engineer** | Grafana 修复、e2e 验收、压测脚本和执行 | S1, S3, S7, S8, S10 |
| **backend-engineer** | Worker OTel、Go benchmark、性能修复 | S2, S6, S9 |
| **architect** | routing-advisor 完善、hermesx ADR | S11, S12 |
| **qa-engineer** | 压测结果验证、demo 功能验证（按需） | 验收支持 |

---

## 6. 风险与依赖

| 风险 | 概率 | 影响 | 缓解措施 | Owner |
|------|------|------|----------|-------|
| Grafana dashboard 修复后仍有面板显示异常 | 中 | P1 阻塞 | 逐面板验证，不批量替换 | devops-engineer |
| Worker OTel 初始化引入性能开销 | 低 | 可观测性 vs 性能 | 采样率可配置，默认 100% 仅在 dev 环境 | backend-engineer |
| Go benchmark 暴露 Append() 需要大改 | 中 | P1b 延期 1 周 | 先用事务包装最小修复，不重构整个 pgStore | backend-engineer |
| 100 并发压测暴露系统性问题 | 中 | P1b 延期 1-2 周 | 已在 buffer 中预留；如超限则 100 并发降为 v2.7.0 | tech-lead 仲裁 |
| awesome-ai-agents PR 无响应 | 高 | P3 部分未完成 | 2 周 SLA + fallback 到其他 awesome lists | tech-lead |
| VHS 录制 GIF 渲染效果不佳 | 低 | P2 延期 | 备选：asciinema + agg 转 GIF | tech-lead |

---

## 7. Implementation Readiness 结论

### Pre-flight Gate 检查

| 检查项 | 状态 | 说明 |
|--------|------|------|
| PRD 存在且完整 | ✅ | `prd.md` 已落盘 |
| 需求挑战会完成 | ✅ | 3 个角色共 10 条质疑已仲裁 |
| 核心阻断条件已识别 | ✅ | 3 个 HIGH 阻断条件已纳入 Phase 1 |
| 角色分工已明确 | ✅ | 4 个角色 + 13 个 slices |
| 技术上下文已收集 | ✅ | brownfield snapshot 已完成 |
| 放行标准已定义 | ✅ | 6 条量化标准 |

### 就绪状态: `handoff-ready`

**执行前提**:
1. Phase 1（观测补齐）必须在 Phase 3（压测执行）之前完成
2. S1 + S2 可并行，完成后 S3 验收
3. S4 + S5 与 Phase 1 完全并行，无依赖
4. S11 + S12 与其他 Phase 并行，无依赖

---

## 8. Brownfield 上下文快照

### 现有基础设施

| 组件 | 状态 | 路径 |
|------|------|------|
| Docker Compose (Postgres + API + Worker) | ✅ 可用 | `deployments/compose/docker-compose.yml` |
| Docker Compose v2 (+ Prometheus + Grafana + Jaeger) | ✅ 可用 | `deployments/compose/docker-compose.v2.yml` |
| Grafana Dashboard (provisioning) | ⚠️ 指标名正确但面板少 | `deployments/compose/grafana/provisioning/dashboards/corag-dashboard.json` |
| Grafana Dashboard (手动导入) | ❌ 指标名错误 | `deployments/grafana/aetheris-dashboard.json` |
| Prometheus 配置 | ✅ 可用 | `deployments/compose/prometheus.yml` |
| Crash Recovery Demo | ⚠️ external_http 边界级 | `examples/crash_recovery/demo.py` |
| Performance Script | ⚠️ 串行，无并发 | `scripts/release-p0-perf.sh` |
| Routing Advisor 设计 | ⚠️ experimental/draft | `docs/artifacts/2026-05-26-routing-advisor-contract/` |
| Blog 草稿 | ✅ 内容就绪 | `docs/blog/11-why-agents-need-runtime.md` |

### 已知技术债

| 项目 | 严重度 | 是否阻塞本 Epic | 建议处理 |
|------|--------|-----------------|----------|
| 6+ 独立 pgxpool.Pool | MEDIUM | 否（压测后评估） | 如压测暴露连接瓶颈，统一为 1-2 个共享池 |
| Append() 非原子 CAS | MEDIUM | 否（压测后评估） | 如重试风暴明显，改为事务内 SELECT + INSERT |
| Worker 无 OTel | HIGH | **是** | Phase 1 修复 |

---

## 9. Karpathy Guidelines 收敛

### 假设（显式列出）
1. Worker crash + restart GIF 足以传达核心价值，不需要 step-level demo
2. 50 并发作为基线能暴露 80% 的问题，100 并发是 stretch goal
3. 博客内容已完成 90%，润色 + 发布 < 1 天
4. routing-advisor 设计文档 70% 完成度，补齐 30% 可在 Week 3 完成

### 更简单备选路径
- 如果 Phase 1 观测补齐超出预期时间：跳过压测，只做 e2e 验收 + GIF + 博客（最小可交付集）
- 如果压测暴露严重问题：发布"已知限制"文档而非假装通过

### 当前不做项
- Step-level crash recovery demo（v2.7.0）
- hermesx 跨团队接口对齐（等外部确认）
- HN Show HN 发布（等证据链更强后）
- 连接池架构重构（等压测数据驱动）

### 为什么本轮范围已经足够
本轮聚焦于"把已有的技术能力变成可核验的外部证据"。不做功能开发、不做架构重构、不做运营活动。这是投入产出比最高的路径：用最少的工程量产出最大的信任信号。

---

## 10. 应用等级 / 技术架构等级

- **应用等级**: 不适用（开源项目，非企业内部应用）
- **技术架构等级**: 不适用
- **关键组件偏离**: 无（不涉及集团组件约束）
- **ADR**: 需要新增 1 份（hermesx 桥接意图，ADR-001）

---

## 11. 技能装配清单

| 技能 | 启用原因 | 主责角色 |
|------|----------|----------|
| `karpathy-guidelines` | intake + plan 收口护栏 | tech-lead |
| `doc-architecture` | P4 架构文档补齐 | architect |

---

## 12. 门禁状态

- **Pre-flight**: ✅ 全部通过
- **Revision**: 0 项待修改
- **Escalation**: 0 项需升级
- **Abort**: ✅ 无阻塞（3 个阻断条件已纳入 Phase 1 处理）
