# Aetheris 可观测性实战

> 本文介绍如何使用 Aetheris v2.2.0 的可观测性功能：Trace UI、OpenTelemetry 配置、Jaeger 集成。

## 概述

Aetheris 提供多层次的可观测性：

| 层级 | 工具 | 用途 |
|------|------|------|
| **Job Trace UI** | 内置 HTML | 单个 Job 的执行时间线、节点详情 |
| **运维指标** | Prometheus | Queue 积压、Stuck Job 监控 |
| **分布式追踪** | OpenTelemetry + Jaeger | 跨服务调用链 |
| **Grafana Dashboard** | Grafana | 聚合视图、Plan/Node 耗时 |

## 1. Job Trace UI

每个 Job 执行完成后，可以查看完整的执行轨迹：

```bash
# 创建 Job
curl -X POST http://localhost:8080/api/agents/agent-xxx/message \
  -d '{"message":"请分析这份文档"}'

# 获取 job_id，访问 Trace 页面
open http://localhost:8080/api/jobs/job-xxx/trace/page
```

### Trace 页面功能

- **时间线条**：展示 plan、node、tool、recovery 等片段
- **节点列表**：左侧显示所有节点，点击查看详情
- **State Diff**：执行前后的状态变化
- **Step Replay**：选中某个 step 可回放查询

### JSON API

```bash
# 获取结构化 Trace 数据
curl http://localhost:8080/api/jobs/job-xxx/trace

# 获取事件流
curl http://localhost:8080/api/jobs/job-xxx/events
```

## 2. 运维可观测性

### Queue 积压与 Stuck Job

```bash
# 获取概要（Queue 积压 + Stuck Job）
curl "http://localhost:8080/api/observability/summary?older_than=1h"

# 返回示例
{
  "queue_backlog": {"default": 5},
  "stuck_job_ids": ["job-abc"],
  "stuck_threshold_seconds": 3600
}
```

### Prometheus 指标

启动 Prometheus 后，可查询：

```promql
# Queue 积压
aetheris_queue_backlog{queue="default"}

# Stuck Job 数量
aetheris_stuck_job_count

# Job 执行耗时
histogram_quantile(0.95, rate(aetheris_job_duration_seconds_bucket[5m]))
```

## 3. OpenTelemetry 配置

Aetheris v2.2.0 默认启用 OpenTelemetry。

### 启动完整栈

```bash
# 一键启动（包含 Jaeger + Grafana）
make docker-run
```

服务端口：
- **API**: http://localhost:8080
- **Jaeger**: http://localhost:16686
- **Grafana**: http://localhost:3000 (admin/admin)
- **Prometheus**: http://localhost:9090

### 手动配置

如需自定义 OTLP 端点：

```yaml
# configs/api.yaml
otel:
  endpoint: "jaeger:4317"
  service_name: "aetheris-api"
```

## 4. Jaeger 集成

### 查看分布式追踪

1. 打开 http://localhost:16686
2. 选择 Service：`aetheris-api` 或 `aetheris-worker`
3. 搜索 Trace，查看调用链

### 追踪内容

Jaeger 追踪包括：
- HTTP 请求延迟
- 数据库查询耗时
- LLM 调用时间
- 工具执行时间

### 自定义 Span

在代码中添加自定义 Span：

```go
ctx, span := otel.Tracer("aetheris").Start(ctx, "my-custom-operation")
defer span.End()

// 业务逻辑
span.SetAttributes("job_id", jobID)
span.SetAttributes("agent_id", agentID)
```

## 5. Grafana Dashboard

### 内置面板

v2.2.0 提供 Grafana Dashboard，包含：

| 面板 | 指标 |
|------|------|
| Plan/Compile Duration | 任务规划耗时 |
| Node Execution | 节点执行时间分布 |
| Run Control | Job 暂停/恢复/取消统计 |
| Queue Backlog | 各队列积压情况 |
| Error Rate | 失败率趋势 |

### 导入 Dashboard

```bash
# Grafana API 导入
curl -X POST http://localhost:3000/api/dashboards/db \
  -H "Content-Type: application/json" \
  -d @grafana-dashboard.json
```

或直接在 Grafana UI 中导入 `deployments/grafana/dashboard.json`。

## 6. 故障排查场景

### 场景 1: Job 一直 pending

```bash
# 1. 检查 Queue 积压
curl http://localhost:8080/api/observability/summary

# 2. 检查 Worker 是否在运行
ps aux | grep worker

# 3. 查看 Worker 日志
docker logs aetheris-worker-1
```

### 场景 2: Job 执行超时

```bash
# 1. 获取 stuck job
curl http://localhost:8080/api/observability/stuck

# 2. 查看 Trace
open http://localhost:8080/api/jobs/job-xxx/trace/page

# 3. 在 Jaeger 查找耗时长的 Span
```

### 场景 3: LLM 调用失败

```bash
# 查看 Trace 中的 LLM 节点
curl http://localhost:8080/api/jobs/job-xxx/trace | jq '.nodes[] | select(.type=="llm")'
```

## 总结

Aetheris v2.2.0 提供完整的可观测性解决方案：

1. **Trace UI** — 单 Job 级别的执行细节
2. **运维 API** — Queue 积压、Stuck Job 监控
3. **OpenTelemetry + Jaeger** — 分布式追踪
4. **Grafana** — 聚合指标与可视化

配合 Prometheus 告警规则，可实现生产环境的全链路监控。

## 延伸阅读

- [运维可观测性详解](./guides/observability.md)
- [Tracing 配置](./guides/tracing.md)
- [Runtime Guarantees](./guides/runtime-guarantees.md)
