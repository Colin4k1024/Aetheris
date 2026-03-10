# 从原型到生产：部署踩坑与最佳实践

> 把 Agent 从「能跑」变成「能打」——生产环境部署的完整指南。

## 0. 部署的差距

```
原型环境                           生产环境
─────────────────────────────    ─────────────────────────────
单机器                             多机器集群
内存 8GB                         内存 64GB+
无认证                           需要认证
HTTP 开发服务器                   高性能服务器
手动部署                         自动化部署
无监控                           全链路监控
单副本                           多副本高可用
```

每个「箭头」背后都是一堆坑。

## 1. 部署架构

### 1.1 推荐架构

```
┌─────────────────────────────────────────────────────────────┐
│                      生产部署架构                             │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│  ┌─────────────┐                                           │
│  │   CDN/WAF   │  ← 流量入口、DDoS 防护                      │
│  └──────┬──────┘                                           │
│         │                                                    │
│  ┌──────▼──────┐                                           │
│  │ Load Balancer│  ← 负载均衡                               │
│  └──────┬──────┘                                           │
│         │                                                    │
│  ┌──────▼──────┐     ┌──────┐     ┌──────┐                │
│  │ API Pod N   │     │API Pod│     │API Pod│  ← API 层      │
│  └──────┬──────┘     └──────┘     └──────┘                │
│         │                                                    │
│  ┌──────▼──────────────────────────────────────┐            │
│  │           Worker Pool                        │            │
│  │  ┌────────┐  ┌────────┐  ┌────────┐       │            │
│  │  │Worker 1│  │Worker 2│  │Worker N│       │  ← 执行层   │
│  │  └────────┘  └────────┘  └────────┘       │            │
│  └─────────────────────────────────────────────┘            │
│         │                                                    │
│  ┌──────▼──────┐     ┌──────┐                            │
│  │ PostgreSQL  │     │Redis │   ← 数据层（高可用）         │
│  │  (主从)     │     │(集群)│                              │
│  └─────────────┘     └──────┘                            │
│                                                             │
└─────────────────────────────────────────────────────────────┘
```

### 1.2 组件说明

| 组件 | 数量 | 作用 |
|------|------|------|
| API | 2-3+ | 处理请求、任务管理 |
| Worker | N | 执行 Agent 任务 |
| PostgreSQL | 1主+1从 | 持久化存储 |
| Redis | 3+ | 缓存、消息队列 |
| Load Balancer | 2+ | 负载均衡 |

## 2. 配置清单

### 2.1 环境变量

```bash
# .env.production

# API 配置
API_PORT=8080
API_REPLICAS=3

# 数据库
DATABASE_URL=postgres://user:pass@postgres:5432/aetheris
DATABASE_MAX_CONNECTIONS=100
DATABASE_SSL_MODE=require

# Redis
REDIS_URL=redis://redis:6379
REDIS_POOL_SIZE=50

# 认证
JWT_SECRET=your-production-secret-key
JWT_EXPIRY=24h

# 安全
CORS_ORIGINS=https://your-domain.com
ALLOWED_HOSTS=api.your-domain.com

# 日志
LOG_LEVEL=info
LOG_FORMAT=json

# 监控
OTEL_EXPORTER_OTLP_ENDPOINT=http://otel-collector:4317
```

### 2.2 配置文件

```yaml
# config.yaml
server:
  port: 8080
  read_timeout: 30s
  write_timeout: 30s
  idle_timeout: 120s

database:
  host: postgres
  port: 5432
  name: aetheris
  max_open_conns: 25
  max_idle_conns: 10
  conn_max_lifetime: 5m

redis:
  host: redis
  port: 6379
  db: 0
  pool_size: 50

worker:
  id: ${HOSTNAME}
  max_concurrent_jobs: 10
  lease_duration: 30s
  heartbeat_interval: 10s

llm:
  default_model: gpt-4
  timeout: 60s
  max_retries: 3
  retry_delay: 1s

security:
  production_mode: true
  require_auth: true
  jwt_required: true
  cors_strict: true
```

## 3. 数据库准备

### 3.1 初始化

```bash
# 运行数据库迁移
aetheris migrate up

# 或手动执行 SQL
psql -h postgres -U user -d aetheris -f migrations/001_initial.sql
```

### 3.2 关键表

```sql
-- Jobs 表
CREATE TABLE jobs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    agent_id UUID NOT NULL,
    status VARCHAR(20) NOT NULL,
    input JSONB NOT NULL,
    output JSONB,
    error TEXT,
    priority INTEGER DEFAULT 0,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    completed_at TIMESTAMP
);

-- Events 表（核心）
CREATE TABLE events (
    id BIGSERIAL PRIMARY KEY,
    job_id UUID NOT NULL,
    sequence_id BIGINT NOT NULL,
    event_type VARCHAR(30) NOT NULL,
    payload JSONB NOT NULL,
    created_at TIMESTAMP DEFAULT NOW(),
    UNIQUE(job_id, sequence_id)
);

CREATE INDEX idx_events_job_id ON events(job_id);
CREATE INDEX idx_events_sequence ON events(job_id, sequence_id);

-- Tool Ledger
CREATE TABLE tool_ledger (
    idempotency_key VARCHAR(255) PRIMARY KEY,
    job_id UUID NOT NULL,
    tool_name VARCHAR(100) NOT NULL,
    status VARCHAR(20) DEFAULT 'committed',
    output JSONB,
    created_at TIMESTAMP DEFAULT NOW()
);

-- Checkpoints
CREATE TABLE checkpoints (
    job_id UUID PRIMARY KEY,
    step_id VARCHAR(50) NOT NULL,
    state JSONB NOT NULL,
    cursor BIGINT NOT NULL,
    created_at TIMESTAMP DEFAULT NOW()
);
```

## 4. Kubernetes 部署

### 4.1 Namespace

```yaml
# namespace.yaml
apiVersion: v1
kind: Namespace
metadata:
  name: aetheris
  labels:
    name: aetheris
```

### 4.2 ConfigMap

```yaml
# configmap.yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: aetheris-config
  namespace: aetheris
data:
  config.yaml: |
    server:
      port: 8080
    database:
      host: postgres
      port: 5432
      name: aetheris
    redis:
      host: redis
      port: 6379
    worker:
      max_concurrent_jobs: 10
      lease_duration: 30s
```

### 4.3 API Deployment

```yaml
# api-deployment.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: aetheris-api
  namespace: aetheris
spec:
  replicas: 3
  selector:
    matchLabels:
      app: aetheris-api
  template:
    metadata:
      labels:
        app: aetheris-api
    spec:
      containers:
      - name: api
        image: aetheris/api:latest
        ports:
        - containerPort: 8080
        env:
        - name: CONFIG_PATH
          value: /config/config.yaml
        - name: DATABASE_URL
          valueFrom:
            secretKeyRef:
              name: aetheris-secrets
              key: database-url
        - name: JWT_SECRET
          valueFrom:
            secretKeyRef:
              name: aetheris-secrets
              key: jwt-secret
        volumeMounts:
        - name: config
          mountPath: /config
        resources:
          requests:
            memory: "256Mi"
            cpu: "250m"
          limits:
            memory: "512Mi"
            cpu: "500m"
        livenessProbe:
          httpGet:
            path: /api/health
            port: 8080
          initialDelaySeconds: 10
          periodSeconds: 10
        readinessProbe:
          httpGet:
            path: /api/health
            port: 8080
          initialDelaySeconds: 5
          periodSeconds: 5
      volumes:
      - name: config
        configMap:
          name: aetheris-config
```

### 4.4 Worker Deployment

```yaml
# worker-deployment.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: aetheris-worker
  namespace: aetheris
spec:
  replicas: 5
  selector:
    matchLabels:
      app: aetheris-worker
  template:
    metadata:
      labels:
        app: aetheris-worker
    spec:
      containers:
      - name: worker
        image: aetheris/worker:latest
        env:
        - name: CONFIG_PATH
          value: /config/config.yaml
        - name: DATABASE_URL
          valueFrom:
            secretKeyRef:
              name: aetheris-secrets
              key: database-url
        - name: REDIS_URL
          valueFrom:
            secretKeyRef:
              name: aetheris-secrets
              key: redis-url
        volumeMounts:
        - name: config
          mountPath: /config
        resources:
          requests:
            memory: "512Mi"
            cpu: "500m"
          limits:
            memory: "1Gi"
            cpu: "1000m"
      volumes:
      - name: config
        configMap:
          name: aetheris-config
```

### 4.5 Service

```yaml
# api-service.yaml
apiVersion: v1
kind: Service
metadata:
  name: aetheris-api
  namespace: aetheris
spec:
  selector:
    app: aetheris-api
  ports:
  - port: 80
    targetPort: 8080
  type: ClusterIP
```

### 4.6 Ingress

```yaml
# ingress.yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: aetheris-ingress
  namespace: aetheris
  annotations:
    kubernetes.io/ingress.class: nginx
    cert-manager.io/cluster-issuer: letsencrypt-prod
spec:
  tls:
  - hosts:
    - api.your-domain.com
    secretName: aetheris-tls
  rules:
  - host: api.your-domain.com
    http:
      paths:
      - path: /
        pathType: Prefix
        backend:
          service:
            name: aetheris-api
            port:
              number: 80
```

## 5. 安全配置

### 5.1 认证

```go
// 启用 JWT 认证
config := &Config{
    Security: SecurityConfig{
        ProductionMode: true,
        JWT: JWTConfig{
            Enabled:   true,
            Secret:    os.Getenv("JWT_SECRET"),
            Expiry:    24 * time.Hour,
            Issuer:    "aetheris",
        },
    },
}
```

### 5.2 API Key 管理

```bash
# 创建 API Key
aetheris api-key create --name production-key --expiry 90d

# 查看 API Keys
aetheris api-key list

# 撤销 API Key
aetheris api-key revoke key_id_123
```

### 5.3 网络策略

```yaml
# network-policy.yaml
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: aetheris-policy
  namespace: aetheris
spec:
  podSelector:
    matchLabels:
      app: aetheris-worker
  policyTypes:
  - Ingress
  - Egress
  ingress:
  - from:
    - podSelector:
        matchLabels:
          app: aetheris-api
  egress:
  - to:
    - podSelector:
        matchLabels:
          app: postgres
    - podSelector:
        matchLabels:
          app: redis
```

## 6. 监控与告警

### 6.1 Prometheus 指标

```yaml
# prometheus-rules.yaml
groups:
- name: aetheris
  rules:
  - alert: HighJobFailureRate
    expr: |
      rate(aetheris_jobs_failed_total[5m]) 
      / rate(aetheris_jobs_completed_total[5m]) > 0.05
    for: 5m
    labels:
      severity: critical
    annotations:
      summary: "Job 失败率过高"
      
  - alert: WorkerHighLoad
    expr: aetheris_worker_load > 0.9
    for: 5m
    labels:
      severity: warning
    annotations:
      summary: "Worker 负载过高"
      
  - alert: DatabaseConnectionExhausted
    expr: |
      db_connections_active / db_connections_max > 0.9
    for: 2m
    labels:
      severity: critical
    annotations:
      summary: "数据库连接即将耗尽"
```

### 6.2 日志收集

```yaml
# fluentd-configmap.yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: fluentd-config
  namespace: aetheris
data:
  fluent.conf: |
    <source>
      @type tail
      path /var/log/aetheris/*.log
      pos_file /var/log/aetheris/aetheris.log.pos
      <parse>
        @type json
      </parse>
    </source>
    
    <filter **>
      @type record_transformer
      <record>
        hostname "#{ENV['HOSTNAME']}"
        app aetheris
      </record>
    </filter>
    
    <match **>
      @type elasticsearch
      hosts elasticsearch:9200
      logstash_format true
      logstash_prefix aetheris
    </match>
```

## 7. 常见问题

### 7.1 数据库连接耗尽

```
症状：
- "pq: sorry, too many clients already"
- API 响应变慢

解决：
1. 增大连接池：db.SetMaxOpenConns(50)
2. 检查连接泄漏：确保使用 defer db.Close()
3. 使用连接池中间件
```

### 7.2 Worker 内存泄漏

```
症状：
- Worker OOM 被 Kill
- 内存使用持续增长

解决：
1. 限制状态大小
2. 定期清理缓存
3. 设置资源限制
```

### 7.3 LLM API 限流

```
症状：
- "Rate limit exceeded"
- LLM 调用失败

解决：
1. 添加重试 + 退避
2. 使用多个 API Key 轮换
3. 实现请求队列
```

### 7.4 Job 堆积

```
症状：
- 队列深度持续增长
- Job 延迟增加

解决：
1. 增加 Worker 数量
2. 优化 LLM 调用速度
3. 识别慢 Job 并优化
```

## 8. 检查清单

### 8.1 上线前检查

```
□ 数据库迁移完成
□ 索引创建完成
□ 配置正确（生产环境）
□ 认证已启用
□ TLS 证书已配置
□ 监控指标已配置
□ 告警规则已配置
□ 日志收集已配置
□ 备份已配置
□ 回滚方案已准备
```

### 8.2 上线后检查

```
□ API 健康检查通过
□ Worker 正常启动
□ Job 可以创建和执行
□ LLM 调用成功
□ 工具调用成功
□ Checkpoint 正常工作
□ 事件正确写入
□ 监控数据正常
□ 日志正常收集
□ 告警正常触发
```

## 9. 小结

生产部署不是结束，而是开始：

1. **架构设计** — 高可用、水平扩展
2. **配置管理** — 环境变量、配置分离
3. **Kubernetes** — 自动化部署、弹性伸缩
4. **安全** — 认证、网络策略
5. **监控** — 指标、日志、告警
6. **运维** — 常见问题处理、上线检查清单

遵循这份指南，你可以在生产环境安全地运行 Aetheris。

---

*10 篇技术博客全部完成！*

## 博客索引

| # | 标题 | 核心内容 |
|---|------|----------|
| 1 | 为什么需要 Agent Runtime？ | 从脚本到进程的范式转变 |
| 2 | Aetheris 核心架构解析 | 事件溯源、检查点、Scheduler |
| 3 | At-Most-Once 语义 | Tool Ledger、幂等性保证 |
| 4 | Checkpoint 与状态恢复 | 崩溃恢复、状态持久化 |
| 5 | Human-in-the-Loop | Wait 节点、Signal 机制 |
| 6 | 集成 LangGraph/AutoGen | Adapter 架构、无缝迁移 |
| 7 | 审计与调试 | 证据图、Timeline、Replay |
| 8 | 多 Worker 部署 | Lease、Fencing Token |
| 9 | 性能优化 | LLM 优化、数据库优化 |
| 10 | 从原型到生产 | K8s 部署、安全、监控 |
