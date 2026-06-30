# PRD: Aetheris 战略推进 Epic (Q3 2026)

> **来源**: GitHub Issue #220
> **状态**: intake
> **主责**: tech-lead
> **日期**: 2026-06-30

---

## 1. 背景

Aetheris 技术基础扎实（v2.5.3 已发布、372 commits、完整文档体系），但外部牵引力接近零（0 forks、0 外部贡献者、14 stars）。核心问题是**认知问题**——没有人知道它存在，知道的人没有看到足够的"可运行"证据。

Issue #220 通过完整阅读代码、ROADMAP、PROJECT-BOARD 和 prototype-promotion-backlog 后，给出了 4 个优先级方向。本次 intake 将其作为单一 Epic 拆解。

---

## 2. 目标与成功标准

### 业务目标

将 Aetheris 从"技术完成但无人知晓"推进到"可验证的生产级 AI Agent 运行时，有外部证据和社区信号"。

### 成功标准（Epic 完成时）

| 维度 | 指标 | 当前值 | 目标值 |
|------|------|--------|--------|
| 生产可信度 | Grafana + Jaeger e2e 验收通过 | 未验收 | 验收文档 + 截图就绪 |
| 生产可信度 | 100 并发 jobs 压测报告 | 无 | 报告产出，含延迟/吞吐/错误率 |
| 视觉证明 | Crash Recovery GIF | 无 | README 顶部可展示的 GIF |
| 外部发现 | 博客发布 | 草稿已有 | 至少 1 篇正式发布 |
| 外部发现 | awesome-ai-agents 合并 | PR #781 待审 | 已合并或有明确进展 |
| 架构定位 | routing-advisor 设计文档 | 已有初版 | 完善至可对外展示 |
| 架构定位 | hermesx 治理桥接接口 | 无 | interface 定义文档产出 |

---

## 3. 范围

### In Scope

#### P1 — 生产就绪可核验
- Compose + Grafana + Jaeger 端到端运行验收
- 100 并发 jobs + 长事件流 snapshot 压测
- 验收结果写入文档，Grafana 截图进 README

#### P2 — Crash Recovery Demo 可视化
- 打磨 `examples/crash_recovery/` 为一键可运行 demo
- 终端输出清晰展示：处理中 → 崩溃 → 恢复 → 续跑
- 录制 GIF 放在 README 最顶部

#### P3 — 外部发现面
- `docs/blog/11-why-agents-need-runtime.md` 正式发布
- awesome-ai-agents PR #781 跟进
- GitHub Discussions 持续内容输出

#### P4 — 架构定位
- routing-advisor 设计文档完善（已有 `docs/artifacts/2026-05-26-routing-advisor-contract/`）
- hermesx ↔ Aetheris 治理桥接 interface 定义

### Out of Scope

- v3.0 功能实现（mTLS、SOC 2 readiness、SAML/OIDC）
- 实际的 Hacker News Show HN 发布（仅准备素材）
- Discord 社区激活（属于运营范畴）
- 任何代码功能变更（本 Epic 聚焦文档、demo 和验证）

---

## 4. 用户故事

### US-1: 生产就绪验证
> 作为**考虑引入 Aetheris 的外部工程师**，我希望看到 Grafana dashboard 截图和压测数据，以便判断它是否真的能在生产环境运行。

**验收标准**:
- [ ] `docker-compose up` 一键启动含 Grafana + Jaeger 的完整环境
- [ ] Grafana dashboard 有 Aetheris 核心指标截图（job 吞吐、延迟分布、worker 状态）
- [ ] 100 并发 jobs 压测报告产出，包含 P50/P95/P99 延迟、吞吐量、错误率
- [ ] 结果写入 `docs/guides/production-validation.md`

### US-2: Crash Recovery 视觉证明
> 作为**首次访问 GitHub 的工程师**，我希望在 README 顶部看到一个直观的 GIF，展示进程崩溃后自动恢复的能力，以便在 10 秒内理解核心价值。

**验收标准**:
- [ ] `examples/crash_recovery/` 支持一条命令运行
- [ ] 终端输出展示：step 进度 → 崩溃 → 重启 → "Resumed at step N"
- [ ] GIF 时长 ≤ 30 秒，放在 README 顶部
- [ ] GIF 在 GitHub README 中可正常渲染

### US-3: 外部可见性
> 作为**项目维护者**，我希望通过博客和社区收录让更多人发现 Aetheris。

**验收标准**:
- [ ] 博客 #1 正式发布（至少在 GitHub Pages 或外部平台）
- [ ] awesome-ai-agents PR #781 有明确进展（合并或 maintainer 反馈）
- [ ] GitHub Discussions 有至少 2 篇有实质内容的帖子

### US-4: 架构叙事锚定
> 作为**有企业背景的潜在贡献者**，我希望看到 Aetheris 在更大架构中的定位，以便评估它是否适合我的场景。

**验收标准**:
- [ ] routing-advisor 设计文档可对外展示（已有基础，需完善）
- [ ] hermesx 治理桥接 interface 定义文档产出
- [ ] 文档说明 Aetheris 在四层架构中的位置（hermesx → superagent-base → Aetheris → Oris/openhuman）

---

## 5. 关键假设

| # | 假设 | 验证方式 | 风险等级 |
|---|------|----------|----------|
| A1 | Grafana dashboard JSON 可直接在 compose 环境中运行 | 启动 compose 验证 | 低 |
| A2 | 现有 `examples/crash_recovery/demo.py` 功能基本可用，只需打磨 | 跑一遍 demo | 中 |
| A3 | 100 并发压测不需要代码变更，只需测试脚本 | 编写压测脚本验证 | 中 |
| A4 | hermesx repo 可访问，能定义接口 | 确认 repo 权限 | 低 |
| A5 | 博客内容已完成 90%，只需润色和发布 | 审阅现有草稿 | 低 |

---

## 6. 风险与依赖

| 风险 | 影响 | 缓解措施 |
|------|------|----------|
| Grafana dashboard 与当前代码版本不兼容 | P1 阻塞 | 先验证，不通则更新 JSON |
| crash recovery demo 实际运行有 bug | P2 阻塞 | 先跑通再录制 |
| awesome-ai-agents maintainer 不活跃 | P3 进度不可控 | 同步准备其他收录渠道 |
| hermesx 接口定义需要对方团队参与 | P4 可能延期 | 先出单侧接口定义，再对齐 |
| 压测暴露性能问题 | P1 可能需要代码修复 | 压测在独立分支进行 |

### 依赖项

| 依赖 | 类型 | 状态 |
|------|------|------|
| `deployments/compose/` docker-compose 配置 | 内部 | 已存在 |
| `deployments/compose/grafana/aetheris-dashboard.json` | 内部 | 已存在 |
| `examples/crash_recovery/demo.py` | 内部 | 已存在 |
| `docs/blog/11-why-agents-need-runtime.md` | 内部 | 草稿已有 |
| `docs/artifacts/2026-05-26-routing-advisor-contract/` | 内部 | 初版已有 |
| awesome-ai-agents 仓库 | 外部 | PR #781 待审 |

---

## 7. 待确认项

| # | 问题 | 建议默认 | 需要谁确认 |
|---|------|----------|------------|
| Q1 | 压测目标：100 并发是最终目标还是起始目标？ | 100 作为基线，后续可扩展 | tech-lead |
| Q2 | Grafana 截图进 README 是放主 README 还是单独文档？ | 主 README 加链接，详细版在 guides/ | tech-lead |
| Q3 | GIF 录制用什么工具？（asciinema / vhs / 手动） | vhs（声明式，可复现） | devops-engineer |
| Q4 | 博客发布渠道：GitHub Pages / Medium / Dev.to / 自建？ | GitHub Pages 优先（已有） | tech-lead |
| Q5 | hermesx 桥接接口是单侧定义还是需要对方参与？ | 先出单侧，再对齐 | architect |
| Q6 | 本 Epic 的时间窗口？ | 4 周（对齐 v2.6.0） | tech-lead |

---

## 8. 企业治理待确认项

本项目为**开源项目**，非企业内部应用。以下为企业治理 checklist 确认：

- [x] 非企业内部应用，不适用应用等级评定
- [x] 无敏感数据处理（当前版本）
- [x] 无跨境数据传输
- [x] v3.0 规划中的 SOC 2 / HIPAA 为可选增强，不在本 Epic 范围
- [ ] 无集团组件约束
- [ ] 无统一技术栈基线要求

---

## 9. 领域技能包启用建议

| 技能 | 触发原因 | 主责角色 |
|------|----------|----------|
| `doc-architecture` | P4 涉及架构文档补齐 | architect |
| `karpathy-guidelines` | 已在 intake 中使用，收敛假设和范围 | tech-lead |

不涉及前端变更，不启用 `frontend-engineering` / `frontend-ui-ux-system`。

---

## 10. UI 范围、终端假设与质量门禁

- **无前端 UI 变更**
- **终端输出**：crash recovery demo 需要清晰的终端输出格式，但不涉及 TUI 框架
- **GIF 质量**：分辨率 ≥ 800px 宽、帧率 ≥ 15fps、时长 ≤ 30s
- **Grafana 截图**：分辨率 ≥ 1200px 宽、包含关键指标面板

---

## 11. 需求挑战会候选分组

建议拆为 3 个并行分组进入 Requirement Challenge Session：

### Group A: 生产验证与观测
- **范围**: P1（e2e 验收 + 压测）
- **建议参与**: `devops-engineer`（主）、`tech-lead`、`qa-engineer`
- **核心挑战**: 压测指标基线是什么？Grafana dashboard 是否需要更新？
- **前置依赖**: 无

### Group B: 开发者体验与外部可见性
- **范围**: P2（demo GIF）+ P3（博客 + 社区）
- **建议参与**: `tech-lead`（主）、内容/社区负责人
- **核心挑战**: demo 是否需要代码修复？博客发布渠道选择？
- **前置依赖**: Group A 的 e2e 验收通过后，GIF 可复用其环境

### Group C: 架构定位
- **范围**: P4（routing-advisor + hermesx 接口）
- **建议参与**: `architect`（主）、`tech-lead`
- **核心挑战**: hermesx 接口是否需要跨团队对齐？
- **前置依赖**: 无（可与 A/B 并行）

---

## 12. Karpathy Guidelines 收敛

### 关键假设（显式列出）
1. 用户（外部工程师）的决策路径是：看到 README → 看到证据 → 尝试运行 → star/contribute
2. 当前最大的瓶颈是"证据缺失"而非"功能缺失"
3. 4 周时间窗口足以完成文档/demo 工作

### 最小可行范围
- P1 + P2 = 生产证据 + 视觉证明，是 Epic 的 MVP
- P3 + P4 = 增长和定位，可延后但不应跳过

### 非目标
- 不做功能开发
- 不做运营活动（Discord 管理、社交媒体运营）
- 不做 v3.0 实现

### 成功标准
- 一个从未接触过 Aetheris 的工程师，能在 5 分钟内从 README 判断"这东西能跑、有压测、有恢复能力"
