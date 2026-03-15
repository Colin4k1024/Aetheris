# 性能调优指南

本文档提供 Aetheris 运行时的性能调优建议。

## 数据库调优

### PostgreSQL 连接池

配置 `jobstore.postgres.max_open_conns` 和 `jobstore.postgres.max_idle_conns`:

```yaml
jobstore:
  postgres:
    max_open_conns: 25
    max_idle_conns: 10
```

- `max_open_conns`: 建议设置为 CPU 核心数的 2-4 倍
- `max_idle_conns`: 建议设置为 `max_open_conns` 的 25-50%

### 索引优化

关键查询索引已包含在 schema.sql 中:

- `idx_jobs_tenant_status` - 租户 + 状态查询
- `idx_job_events_job_id` - 事件按 job_id 查询
- `idx_checkpoints_agent_id` - 按 agent_id 查询 checkpoint
- `idx_tool_invocations_job_id` - 按 job_id 查询工具调用

### TTL 配置

合理设置数据过期时间减少存储压力:

```yaml
gc:
  enable: true
  ttl_days: 90  # 默认 90 天，可根据业务调整
```

checkpoint 配置:

```yaml
checkpoint_store:
  type: postgres
  ttl: 7  # checkpoint 保留天数
```

## 缓存策略

### Redis 缓存

启用 Redis 加速热点数据:

```yaml
cache:
  redis:
    enabled: true
    addr: localhost:6379
    db: 0
```

### 内存缓存

对于小规模部署，可使用内存缓存:

```yaml
checkpoint_store:
  type: memory
```

## 并发配置

### Worker 并发

调整 worker 数量和并发限制:

```yaml
worker:
  concurrency: 10  # 单 Worker 并发数
  queue_size: 1000 # 任务队列大小
```

### 调度器配置

```yaml
job_scheduler:
  max_retries: 3
  retry_delay: 5s
  lease_ttl: 30s
```

## 监控与诊断

### Prometheus 指标

访问 `/metrics` 端点获取运行时指标:

```bash
curl http://localhost:8080/api/system/metrics
```

关键指标:
- `job_state` - 各状态 Job 数量
- `worker_active` - 活跃 Worker 数量
- `tool_invocation_duration` - 工具调用耗时

### 健康检查

```bash
curl http://localhost:8080/api/health
```

## Benchmark

运行基准测试:

```bash
make bench
```

典型结果 (单节点):
- Job 创建: ~10ms
- 事件追加: ~5ms
- Checkpoint 保存: ~20ms
- 工具调用: 取决于外部服务
