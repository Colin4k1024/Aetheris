# Aetheris Project Board

> 轻量级看板，用 Markdown 管理。转换为 GitHub Project Board：https://github.com/users/Colin4k1024/projects

## v2.6.0 目标（4-6周）

---

## 🔴 P1 — 必须完成

### Operational Runtime 生产就绪验收
- [x] Store 层 tenant-aware Idempotency-Key 查询，避免跨租户复用 Job
- [x] Worker Snapshot/GC 参数配置化，并打印启动生效配置
- [x] Evidence export handler 增加 tenant 校验和无事件存储失败语义
- [x] HTTP/Store/Config 补充生产就绪回归测试
- [ ] Compose + Grafana + Jaeger 端到端运行验收
- [ ] 100 并发 jobs 与长事件流 snapshot 压测

### MCP Gateway 正式版
- [x] 整理 `tools/mcp-marketplace/` → `tools/mcp-gateway/`，生产就绪
- [x] 补充 `tools/mcp-gateway/openapi.yaml`
- [x] 写 `docs/mcp/integration.md` 集成指南
- [x] `examples/` 添加 MCP 集成示例

### 文档完善
- [ ] 快速开始文档（5 分钟跑起来）
- [ ] `docs/guides/getting-started-agents.md` 补全
- [ ] API Reference 文档

### 社区冷启动
- [ ] awesome-ai-agents PR #781 跟进 merge 状态
- [ ] Discord 社区激活（issue #102）
- [ ] GitHub Discussions 周更（issue #113）

---

## 🟡 P2 — 计划中

### 观测能力
- [ ] 验证 Grafana dashboard 完整可跑
- [ ] OpenTelemetry + Jaeger tracing 示例
- [ ] README 添加 "Observability" 章节

### 安全加固
- [ ] RBAC 完整实现检查
- [ ] API 请求签名设计文档
- [ ] `SECURITY.md` 更新

### 内容营销
- [ ] 技术博客 #1：为什么 AI Agent 需要自己的 Runtime
- [ ] 技术博客 #2：事件溯源在 AI Agent 执行中的应用
- [ ] X(Twitter) 日常开发进度更新

---

## 🟢 P3 — 未来版本

### v3.0 规划（企业级）
- [ ] 多租户 RBAC
- [ ] SLA 监控 Dashboard
- [ ] mTLS 内部通信
- [ ] SAML/OIDC SSO

### 插件生态
- [ ] MCP 工具市场正式上线
- [ ] 预置工作流模板 > 10 个
- [ ] 插件市场（issue #120）

---

## 已完成 ✅

| 版本 | 日期 | 关键内容 |
|---|---|---|
| v2.5.3 | 2026-04-19 | Hermes集成+MCP Gateway+安全修复+发布v2.5.3 |
| v2.5.2 | 2026-04-14 | awesome-go 收录 |
| v2.5.1 | 2026-04-14 | 测试补充 |
| v2.5.0 | 2026-03-24 | At-Most-Once 执行保证 |

## 本次完成内容（Week 2 全部收尾）

- [x] `tools/mcp-marketplace/` → `tools/mcp-gateway/`（rename + 生产化）
- [x] `tools/mcp-gateway/openapi.yaml`
- [x] `examples/mcp-gateway/`
- [x] `docs/mcp/integration.md`（MCP Gateway 集成指南）
- [x] `docs/mcp/integration.md`（新增，~5.5KB）
- [x] `docs/reference/api.md`（API 快速参考 + curl 示例）
- [x] `docs/guides/quickstart.md`（补充 MCP 链接）
- [x] 修复 docs 里所有 `rag-platform` → `github.com/Colin4k1024/Aetheris/v2`
- [x] `docs/blog/11-why-agents-need-runtime.md`（博客 #3: Aetheris vs Temporal）
- [x] `docs/guides/mcp-marketplace.md`（MCP 工具市场展示页）
- [x] `docs/promotion/newsletter.md`（订阅方式说明）
- [x] `docs/promotion/tweet-drafts-v2.5.3.md`（推文草稿 x3）
- [x] README.md 新增 Templates & Ecosystem 板块
- [x] GitHub Discussion #166（Announcements: v2.5.3 发布公告）
- [x] GitHub Discussion #167（Q&A 欢迎帖 + FAQ）
- [x] awesome-ai-agents PR #781 ping maintainer + 更新描述

---

## 指标追踪

| 指标 | 当前 | 4周目标 | 12周目标 |
|---|---|---|---|
| GitHub Stars | ~5 | 20 | 100 |
| 外部贡献者 | 0 | 1 | 5 |
| 文档页面 | 基本空白 | 核心功能完整 | 视频教程完成 |
| Discussions | 0 | 周更 | 日更 |
