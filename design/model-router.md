# Model Router Design - EPIC 1: 动态模型路由

## 1. 概述

本文档描述 Aetheris 动态模型路由功能的设计方案，实现根据节点复杂度、费用需求、延迟目标自动选择最适合的 LLM 模型。

## 2. 模型分级策略

### 2.1 分级定义

| 级别 | 模型示例 | 适用场景 | 延迟目标 | 成本级别 |
|------|----------|----------|----------|----------|
| **T1 - 推理** | o1, o3-mini | 复杂推理、规划、代码生成 | < 60s | $$$ |
| **T2 - 旗舰** | gpt-4o, claude-4, gemini-2.5 | 复杂对话、长上下文理解 | < 30s | $$ |
| **T3 - 均衡** | gpt-4o-mini, claude-3.5-haiku, gemini-1.5-flash | 日常对话、轻量推理 | < 10s | $ |
| **T4 - 经济** | qwen-turbo, gpt-3.5-turbo | 简单任务、批量处理 | < 5s | $ |

### 2.2 模型元数据

```go
type ModelInfo struct {
    Name         string    // 模型名称
    Provider     string    // 提供商 (openai, anthropic, google, qwen)
    Tier         ModelTier // T1-T4
    ContextLimit int       // 上下文窗口大小 (tokens)
    CostPer1KInput  float64  // 每 1K input tokens 成本 (USD)
    CostPer1KOutput float64  // 每 1K output tokens 成本 (USD)
    AvgLatencyMs  int      // 平均延迟 (ms)
    Capabilities []string  // 能力标签: reasoning, vision, function_call, etc.
}
```

## 3. 路由策略

### 3.1 路由决策因素

1. **节点复杂度** (Node Complexity)
   - 简单: 单轮问答、摘要
   - 中等: 多轮对话、意图识别
   - 复杂: 规划、代码生成、复杂推理

2. **成本约束** (Cost Budget)
   - 预算上限 ($/请求 或 $/小时)
   - 成本优先级: 最低/平衡/最优

3. **延迟目标** (Latency SLA)
   - 延迟上限 (ms)
   - 延迟优先级: 最低/平衡/最优

4. **能力要求** (Capability Requirements)
   - 必须支持的能力 (vision, function_call, etc.)

### 3.2 路由策略接口

```go
type Router interface {
    // SelectModel 选择最适合的模型
    SelectModel(ctx context.Context, req *RoutingRequest) (*ModelInfo, error)
    
    // SelectFallback 获取备用模型
    SelectFallback(ctx context.Context, primary *ModelInfo, reason FallbackReason) (*ModelInfo, error)
    
    // RecordOutcome 记录路由结果（用于优化）
    RecordOutcome(ctx context.Context, model *ModelInfo, outcome *RoutingOutcome)
}

type RoutingRequest struct {
    Complexity     NodeComplexity // 节点复杂度
    MaxCost        float64         // 最大成本 ($)
    MaxLatencyMs   int             // 最大延迟 (ms)
    RequiredCaps   []string        // 必需能力
    PreferProvider string          // 首选提供商
    Priority       RoutingPriority // 优先级
}

type NodeComplexity int
const (
    ComplexitySimple NodeComplexity = iota
    ComplexityMedium
    ComplexityHigh
)

type RoutingPriority int
const (
    PriorityCost RoutingPriority = iota
    PriorityBalanced
    PriorityLatency
    PriorityQuality
)
```

### 3.3 内置策略

1. **CostOptimizedRouter** - 成本优先
2. **LatencyOptimizedRouter** - 延迟优先
3. **QualityOptimizedRouter** - 质量优先
4. **BalancedRouter** - 均衡策略（默认）

## 4. 容灾机制

### 4.1 热切换 (Hot Failover)

```go
type FailoverConfig struct {
    MaxRetries        int           // 最大重试次数
    RetryDelayMs      int           // 重试延迟 (ms)
    EnableHotSwitch   bool          // 启用热切换
    FallbackStrategy  FallbackStrategy // 降级策略
}
```

**切换流程:**
1. 主模型调用失败 → 记录错误和延迟
2. 触发限流 (429) 或服务端错误 (5xx) → 立即切换
3. 选择下一级模型 → 保留原上下文继续执行
4. 切换后记录切换事件和原因

### 4.2 限流处理

- **429 Too Many Requests**: 等待 retry-after 后重试，或切换模型
- **5xx Server Error**: 立即切换模型
- **超时**: 记录超时事件，切换模型

### 4.3 事件溯源上下文保留

确保切换模型时保留完整上下文:
- 对话历史
- 已消耗的 tokens
- 中间结果

## 5. 审计与 Metrics

### 5.1 路由 Metrics

| Metric | Type | Labels |
|--------|------|--------|
| `router_selections_total` | Counter | tier, provider, complexity, strategy |
| `router_latency_ms` | Histogram | tier, provider |
| `router_fallbacks_total` | Counter | from_tier, to_tier, reason |
| `router_cost_estimated_total` | Counter | tier, provider |
| `router_errors_total` | Counter | provider, error_type |

### 5.2 审计日志

```go
type RouterAuditLog struct {
    Timestamp     time.Time
    RequestID    string
    Strategy     string
    SelectedTier string
    SelectedModel string
    Complexity   NodeComplexity
    MaxCost      float64
    MaxLatencyMs int
    FallbackFrom string
    FallbackTo   string
    FallbackReason string
    TokensUsed   int
    LatencyMs    int
    Error        string
}
```

## 6. 配置示例

```yaml
router:
  enabled: true
  default_strategy: "balanced"
  
  tiers:
    - name: "t1-reasoning"
      models: ["o1", "o3-mini"]
      max_latency_ms: 60000
      max_cost_per_request: 0.5
      
    - name: "t2-flagship"
      models: ["gpt-4o", "claude-4", "gemini-2.5"]
      max_latency_ms: 30000
      max_cost_per_request: 0.1
      
    - name: "t3-balanced"
      models: ["gpt-4o-mini", "claude-3.5-haiku", "gemini-1.5-flash"]
      max_latency_ms: 10000
      max_cost_per_request: 0.02
      
    - name: "t4-economy"
      models: ["qwen-turbo", "gpt-3.5-turbo"]
      max_latency_ms: 5000
      max_cost_per_request: 0.005

  failover:
    max_retries: 2
    retry_delay_ms: 1000
    enable_hot_switch: true
    
  audit:
    enabled: true
    log_to_file: true
    log_path: "./logs/router-audit.jsonl"
```

## 7. 实现计划

1. **Phase 1**: 核心路由引擎
   - ModelInfo 定义
   - Router 接口
   - 基础路由策略

2. **Phase 2**: 容灾机制
   - 重试逻辑
   - 热切换
   - 上下文保留

3. **Phase 3**: 观测与审计
   - Metrics 集成
   - 审计日志
   - 监控面板

4. **Phase 4**: 测试与优化
   - 单元测试
   - 集成测试
   - 性能测试
