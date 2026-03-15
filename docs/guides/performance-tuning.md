# Aetheris 性能调优指南

> 优化 Aetheris 生产环境性能的完整指南

## 1. 连接池配置

### PostgreSQL 连接池

```yaml
# configs/api.yaml
database:
  max_conns: 50        # 最大连接数
  min_conns: 10       # 最小连接数
  max_conn_lifetime: 1h
  max_conn_idle_time: 30m
  health_check_period: 1m
```

**建议：**
- API 服务：20-50 连接
- Worker 服务：10-30 连接
- 根据 CPU 核心数调整：`连接数 = CPU核心数 * 2 + 磁盘数`

### Redis 连接池

```yaml
# configs/api.yaml
redis:
  pool_size: 50       # 每个 CPU 核心的连接数
  min_idle_conns: 10  # 最小空闲连接
  dial_timeout: 5s
  read_timeout: 3s
  write_timeout: 3s
```

## 2. 并发配置

### Worker 并发

```yaml
# configs/worker.yaml
worker:
  # 并发执行的任务数
  max_concurrent_jobs: 10
  
  # 每个任务的超时时间
  job_timeout: 30m
  
  # 任务队列长度
  queue_size: 1000
```

**公式：**
```
max_concurrent_jobs = min(CPU核心数 * 2, 内存GB / 2)
```

### 调度器配置

```yaml
# 调度器轮询间隔（越短越快响应，但 CPU 开销越大）
scheduler:
  poll_interval: 100ms  # 建议范围：50ms - 500ms
  
  # 任务就绪通知队列（可选）
  wakeup_queue: redis
```

## 3. 缓存策略

### 内存缓存

```yaml
cache:
  # 检查点缓存
  checkpoint:
    max_entries: 1000
    ttl: 1h
    
  # Agent 状态缓存
  agent_state:
    max_entries: 500
    ttl: 5m
```

### Redis 缓存

```yaml
redis:
  # 会话缓存
  session:
    ttl: 24h
    
  # 工具结果缓存
  tool_result:
    ttl: 1h
    max_size: 10mb
```

## 4. 检查点优化

### 检查点策略

```yaml
checkpoint:
  # 节点执行完成后自动保存
  auto_save: true
  
  # 保存间隔（节点数）
  save_interval: 1
  
  # 检查点 TTL（自动清理过期检查点）
  ttl: 24h
  
  # 大任务图拆分
  chunk_size: 100kb
```

### 检查点清理

```go
// 定期清理过期检查点
store := runtime.NewCheckpointStoreMem()

// 清理 24 小时前的检查点
deleted, err := store.Cleanup(ctx, time.Now().Add(-24*time.Hour))
log.Printf("Cleaned up %d expired checkpoints", deleted)
```

## 5. 监控指标

### 关键指标

| 指标 | 告警阈值 | 优化方向 |
|------|---------|---------|
| `aetheris_job_duration` | > 30min | 优化 Agent 逻辑 |
| `aetheris_checkpoint_save_duration` | > 1s | 减少状态大小 |
| `aetheris_scheduler_lock_wait` | > 100ms | 减少并发冲突 |
| `aetheris_db_pool_wait` | > 500ms | 增加连接池 |
| `aetheris_redis_pool_wait` | > 100ms | 增加连接池 |

### Prometheus 配置

```yaml
# configs/metrics.yaml
metrics:
  enabled: true
  port: 9090
  path: /metrics
  
  # 自定义指标
  custom:
    - job_duration
    - checkpoint_size
    - tool_call_count
```

## 6. 网络优化

### HTTP 超时

```yaml
api:
  read_timeout: 30s
  write_timeout: 30s
  idle_timeout: 60s
  
  # 上传/下载大文件
  max_request_body_size: 100mb
```

### gRPC 优化

```yaml
grpc:
  # 连接池
  max_concurrent_streams: 100
  
  # 超时
  timeout: 30s
  
  # 压缩
  compression: gzip
```

## 7. 容量规划

### 估算公式

```
内存需求 = (检查点大小 * 并发任务数) + (会话状态 * 用户数) + 连接池内存
```

**示例：**
- 检查点平均大小：100KB
- 并发任务：10
- 用户数：100
- 会话状态：1KB

```
内存 = (100KB * 10) + (1KB * 100) + 500MB ≈ 600MB
```

### 存储估算

```
PostgreSQL = (事件大小 * 任务数 * 平均步骤) + 检查点存储
Redis = 会话缓存 + 工具结果缓存 + 队列
```

## 8. 性能测试

### 基准测试

```bash
# 运行基准测试
go test -bench=. -benchmem ./internal/agent/runtime/...

# 内存 profiling
go test -memprofile=mem.out ./...
go tool pprof mem.out

# CPU profiling  
go test -cpuprofile=cpu.out ./...
go tool pprof cpu.out
```

### 负载测试

```bash
# 使用 wrk 或 hey
wrk -t12 -c400 -d30s http://localhost:8080/api/query
```

---

**注意：** 具体数值需要根据实际负载调整。建议先使用默认配置上线，再根据监控数据逐步优化。
