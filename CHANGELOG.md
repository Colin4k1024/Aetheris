# Changelog

本文档记录 Aetheris 项目的版本与重要变更。格式基于 [Keep a Changelog](https://keepachangelog.com/zh-CN/1.0.0/)。

## [Unreleased]

### Added

- **AgentFactory — 配置驱动的 Agent 创建**:
  - 新增 `internal/runtime/eino/agent_factory.go`：`AgentFactory` + `AgentBuildConfig`
  - 从 `configs/agents.yaml` 自动加载并批量创建 Agent（`GetOrCreateFromConfig`）
  - 支持编程式创建（`CreateAgent`）和带 Checkpoint 创建（`CreateAgentWithCheckpoint`）
  - Runner 缓存机制，同名 Agent 复用已创建实例
  - 工具过滤：`agents.yaml` 中 `tools` 字段限制每个 Agent 可用工具子集

- **Tool Bridge — 工具桥接层**:
  - 新增 `internal/runtime/eino/tool_bridge.go`：`RegistryToolBridge` + `registryToolAdapter`
  - `RuntimeTool` / `RuntimeToolRegistry` 接口抽象（解决 `agent/tools` ↔ `runtime/eino` 循环引用）
  - 自动将 Aetheris 工具（Native + MCP）转为 Eino `InvokableTool`
  - Schema 映射：JSON Schema / 简单 key-value → Eino `ParameterInfo`
  - Context-based Session 传递（`WithSession` / `sessionFromContext`）

- **Engine 集成**:
  - `Engine` 新增 `agentFactory` 字段及 `SetAgentFactory()` / `GetAgentFactory()` 方法
  - `ensureRunner()` 优先查询 AgentFactory
  - `configs/agents.yaml` 新增 `tools` 字段支持

- **测试**:
  - 新增 `tool_bridge_test.go`（13 个测试）和 `agent_factory_test.go`（6 个测试）

### Changed

- `internal/app/api/app.go`：主 ADK Runner 优先通过 `AgentFactory.CreateAgentWithCheckpoint()` 创建，回退到旧路径
- `internal/api/http/handler.go`：新增 `agentFactory` 字段暂露到 HTTP 层
- `pkg/config/config.go`：`AgentDefConfig` 新增 `Tools []string` 字段

### Deprecated

- `internal/agent/agent.go`：`Agent` struct、`New()`、`RunWithSession()`、`Run()` 均标记为 Deprecated，应使用 `eino.AgentFactory` 替代

### Documentation

- AGENTS.md：更新 Important Files、Project Structure，新增 Config-Driven Agent 和 Tool Bridge 使用指南
- CLAUDE.md：更新 Core Components 表和 Execution Flow
- docs/concepts/adk.md：重写，新增 AgentFactory + Tool Bridge 架构说明
- docs/guides/getting-started-agents.md：新增「快速开始：配置驱动 Agent」推荐路径
- docs/guides/sdk.md：新增 AgentFactory 集成说明
- docs/reference/config.md：新增 agents.yaml 配置参考
- design/core.md：新增 4.2 AgentFactory 和 4.3 Tool Bridge Layer 小节

---

## [2.2.0] - 2026-03-04

### Added

- **可观测性增强**:
  - 添加 Jaeger 到 docker-compose 用于分布式追踪
  - 默认启用 OpenTelemetry 追踪
  - 新增 Grafana dashboard 面板 (Plan/Compile duration, Node execution, Run control)
- **多适配器支持**:
  - 添加 LlamaIndex adapter (NodeLlamaIndex)
  - 添加 Vertex AI Agent Engine adapter (NodeVertex)
  - 添加 AWS Bedrock Agents adapter (NodeBedrock)
- **企业级功能**:
  - Postgres RoleStore 用于 RBAC
  - RBAC API 端点 (GET/POST /api/rbac/role, POST /api/rbac/check)
  - Region 配置和 region-aware scheduler
  - SLA Quota manager 和 SLO monitor

### Changed

- docker-compose 默认配置优化

### Documentation

- 新增 3 个 adapter 示例代码

### Verification

- Release gates 测试通过

---

## [2.1.0] - 2026-03-02

### Added

- **运行时执行监控 API** (`/api/runs`):
  - GET/POST `/api/runs` - Run 管理
  - GET `/api/runs/:id/events` - Run 事件流
  - POST `/api/runs/:id/pause|resume` - 执行控制
  - POST `/api/runs/:id/tool-calls` - 幂等工具调用追踪
  - POST `/api/runs/:id/human-decisions` - Human-in-the-Loop 支持
- **RuntimeRunStore 接口** - 内存实现用于运行时状态追踪

### Changed

- NodeEventSink 增强，支持同步到运行视图

### Documentation

- CLAUDE.md - Claude Code 开发指南

---

## [2.0.0] - 2026-02-13

### Added

- **M1 可验证证明链**：
  - 事件哈希链（proof chain）
  - 证据包导出：`POST /api/jobs/:id/export`
  - 离线验证：`aetheris verify <evidence.zip>`
  - Replay/Trace 能力用于确定性复盘
- **M2 合规能力**：
  - 多租户 RBAC（角色与权限控制）
  - 脱敏策略（Redaction）
  - 留存与 Tombstone
  - 审计日志能力
- **M3 取证能力**：
  - Forensics Query（按时间/tool/event 查询）
  - 批量导出与状态轮询
  - Evidence Graph、Audit Log、Consistency Check

### Fixed

- 修复部分历史事件链在导出取证包时的 500 错误：
  - `internal/api/http/forensics.go`
  - 在原链校验失败时执行链归一化重建，确保导出包可验证
- 补充对应单测：
  - `internal/api/http/forensics_test.go`

### Changed

- 发布门禁脚本稳定性增强（`set -u` 场景下 trap 清理安全）：
  - `scripts/release-p0-perf.sh`
  - `scripts/release-p0-drill.sh`
- 故障演练脚本增强：
  - API 重启后 agent 丢失场景支持重建与重试
  - Drill 结果附带更明确的 HTTP code
- 本地 compose 默认 Planner 调整（便于本地 release/perf gate 稳定）：
  - `deployments/compose/docker-compose.yml` 新增 `PLANNER_TYPE=${PLANNER_TYPE:-rule}`

### Verification

- `RUN_P0_PERF=1 RUN_P0_DRILLS=1 PERF_SAMPLES=3 PERF_POLL_MAX=45 ./scripts/release-2.0.sh` 通过
- 性能基线：`artifacts/release/perf-baseline-2.0-20260213-172044.md`
- 故障演练：`artifacts/release/failure-drill-2.0-20260213-172051.md`（passed=4 failed=0 skipped=1）

---

## 历史版本（摘要）

以下为早期提交对应的功能摘要，未按语义化版本打 tag 时可按提交顺序参考。

- **refactor: update planner integration for v1 Agent API** — planGoaler 接口、RulePlanner、PLANNER_TYPE 环境变量
- **feat: implement v1 Agent API and enhance session management** — v1 Agent 端点、Manager/Scheduler/Creator、Session 管理
- **feat: refactor agent execution to support session management and enhance planning** — Session 感知执行、Planner 单步决策、SchemaProvider
- **feat: add agent execution endpoint and integrate agent runner** — `/api/agent/run`、AgentRunner、Session 管理
- **feat: implement gRPC support and JWT authentication** — gRPC 服务、JWT 中间件、文档/查询 gRPC 方法
- **feat: integrate OpenTelemetry for tracing** — 链路追踪与文档处理增强
- **feat: enhance API configuration and workflow execution** — API 配置与工作流执行
- **refactor: migrate from Gin to Hertz** — HTTP 框架由 Gin 迁移至 Hertz
- **feat: 初始化RAG/Agent平台核心组件和架构** — 初始 RAG/Agent 平台骨架
