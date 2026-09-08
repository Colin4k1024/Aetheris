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
- `idx_tool_invocations_archive_archived_at` - 按归档时间清理归档副本

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

### tool_invocations 归档

`gc.ttl_days` 到期后，GC 会清理 `tool_invocations` 账本。若需在清理前保留完整副本，开启归档:

```yaml
runtime:
  gc:
    enabled: true
    ttl_days: 90
    batch_size: 1000
    archive_enabled: true   # 删除前先归档完整副本
    archive_ttl_days: 0     # 0 = 归档副本永久保留
```

行为契约（`internal/runtime/jobstore/archive.go`）:

- **先归档、后校验、再删除**：GC 仅在归档副本写入并回读校验通过后才删除源记录。
- **失败即中止**：归档目标不可用、写入失败或副本校验不一致时，GC 返回错误且**不删除任何源记录**。
- **未配置即报错**：`archive_enabled: true` 但 JobStore 没有归档目标时返回 `ErrArchiveNotConfigured`，不会静默成功。
- **幂等可重入**：归档表主键与源表一致（`job_id` + `idempotency_key`），崩溃后重跑 GC 不产生重复副本、不丢数据。
- **归档表**：`tool_invocations_archive`（`schema.sql` 中以 `CREATE TABLE IF NOT EXISTS` 提供，升级既有库时重跑 schema 即可）。
- **自定义归档目标**：通过 `jobstore.NewPostgresStoreWithOptions(ctx, dsn, lease, jobstore.WithToolInvocationArchiveSink(sink))`
  接入独立冷存储；sink 必须保证 `Write` 返回 nil 时副本已可被 `CountPersisted` 读到。

> ⚠️ `archive_ttl_days > 0` 会按 `archived_at` 删除超期归档副本，属于**不可逆**操作。除非有明确留存期限要求，建议保持 0。

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
