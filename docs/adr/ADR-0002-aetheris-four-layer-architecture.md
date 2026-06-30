# ADR-0002: Aetheris Position in Four-Layer Architecture

> **Status**: proposed
> **Date**: 2026-06-30
> **Owner**: architect
> **关联**: Issue #220 P4, hermesx 治理桥接

## 背景与约束

Aetheris 的长期架构愿景是一个四层 Agent 基础设施栈：

```
L3: hermesx          — 治理层（策略、合规、审计、多租户治理）
L2: superagent-base  — 编排层（Agent 生命周期管理、工作流编排）
L1: Aetheris         — 执行层（持久化执行、crash recovery、event sourcing）
L0: Oris/openhuman   — 能力层（LLM、工具、数据源、外部服务）
```

当前状态：
- **L1 (Aetheris)**: 已实现，v2.5.3 生产就绪
- **L0 (Oris/openhuman)**: 部分实现（MCP 工具、外部 HTTP Agent 适配器）
- **L2 (superagent-base)**: 独立仓库存在，有 SSE 流式协议集成
- **L3 (hermesx)**: 仓库可访问性未确认，无接口定义，无活跃维护者对齐

**核心问题**: 四层架构目前只在文档中存在，没有明确的 integration contract 说明各层如何通信。特别是 L1→L3 的治理上报路径（event stream、audit trail、compliance evidence）缺乏正式接口定义。

## 备选方案

### 方案 A: 完整 hermesx 接口定义

产出 L1↔L3 的完整双向接口文档，包含事件上报、策略下发、审计查询等所有 API。

- **优点**: 架构完整，文档齐全
- **风险**: hermesx 仓库可访问性和团队意愿未确认，可能产出无法验证的空中接口
- **不选原因**: 缺乏外部对齐，投入产出比低

### 方案 B: 单侧 Provider Interface + ADR 占位（采用）

产出 Aetheris 侧的 Provider Interface — 即 Aetheris 期望上层（L2/L3）调用方实现的接口。同时用 ADR 声明架构定位和桥接意图。

- **优点**: 不依赖外部团队，可立即产出；与 routing-advisor 的 "evidence-first" 哲学一致
- **风险**: 单侧接口可能在对方参与后需要调整
- **选择原因**: 先有证据，再有集成；避免在缺乏对齐时产出可能被推翻的设计

### 方案 C: 不产出任何文档

将 hermesx 桥接完全推迟到对方团队主动联系时。

- **优点**: 零投入
- **风险**: Aetheris 在四层架构中的定位不明确，影响吸引企业背景贡献者
- **不选原因**: Issue #220 明确将架构定位作为战略优先级

## 决策结果

**采用方案 B**: 单侧 Provider Interface + ADR 占位。

### Aetheris Provider Interface

Aetheris 作为 L1 执行层，向上层暴露以下能力接口：

```go
// ExecutionProvider is what Aetheris exposes to upper layers (L2/L3).
// Upper layers call these to submit work, query state, and receive events.
type ExecutionProvider interface {
    // SubmitJob submits a job for durable execution.
    // Returns a job ID that can be used to track progress.
    SubmitJob(ctx context.Context, req JobSubmissionRequest) (JobSubmissionResponse, error)

    // GetJobStatus returns the current status of a job.
    GetJobStatus(ctx context.Context, jobID string) (JobStatusResponse, error)

    // GetJobEvents returns the event stream for a job.
    // This is the primary mechanism for audit trail access.
    GetJobEvents(ctx context.Context, jobID string) (JobEventsResponse, error)

    // GetEvidenceExport returns a signed evidence ZIP for a job.
    // Used by L3 for compliance and forensics.
    GetEvidenceExport(ctx context.Context, jobID string) (EvidenceExportResponse, error)

    // SubscribeEvents subscribes to real-time job events.
    // Used by L2 for orchestration and L3 for monitoring.
    SubscribeEvents(ctx context.Context, filter EventFilter) (EventSubscription, error)
}
```

### 上层调用方接口（待对齐）

Aetheris 期望上层实现以下接口（具体定义待 hermesx 团队参与后确定）：

```go
// GovernanceProvider is what Aetheris expects from upper layers (L3).
// L3 implements these to provide policy, compliance, and audit capabilities.
type GovernanceProvider interface {
    // EvaluatePolicy evaluates whether a job/action complies with governance policies.
    EvaluatePolicy(ctx context.Context, req PolicyEvaluationRequest) (PolicyEvaluationResponse, error)

    // ReportEvent reports job events to the governance layer for audit/compliance.
    ReportEvent(ctx context.Context, event GovernanceEvent) error

    // GetComplianceConstraints returns compliance constraints for a tenant/job.
    GetComplianceConstraints(ctx context.Context, tenantID string) (ComplianceConstraints, error)
}
```

### 事件上报路径

```
Aetheris (L1) → Job Events → GovernanceProvider.ReportEvent (L3)
                             → Evidence Export → Compliance Archive
                             → Audit Trail → Forensics Read Model
```

## 企业内控补充

- **应用等级**: 不适用（开源项目）
- **技术架构等级**: 不适用
- **关键组件偏离**: 无
- **资产文档入口**: `docs/artifacts/2026-05-26-routing-advisor-contract/`

## 后续动作

| 动作 | Owner | 完成条件 |
|------|-------|----------|
| 确认 hermesx 仓库可访问性 | tech-lead | 能 clone 或访问 hermesx 仓库 |
| 确认 hermesx 团队对齐意愿 | tech-lead | 对方有活跃维护者愿意 review 接口 |
| 补齐 GovernanceProvider 接口细节 | architect | hermesx 团队参与后 |
| 产出 integration test plan | qa-engineer | 接口定义锁定后 |
