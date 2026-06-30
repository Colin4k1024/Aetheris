# Aetheris Benchmarks

## Go Benchmark（单组件基准测试）

### 前置条件

- PostgreSQL 运行中（`make docker-run` 或手动启动）
- schema 已初始化（compose 自动执行 `schema.sql`）

### 运行

```bash
# 默认连接 localhost:5432
go test -tags benchmark -bench=. -benchtime=10s -count=3 ./internal/runtime/jobstore/

# 自定义 DSN
BENCHMARK_DSN="postgres://corag:corag@localhost:5432/corag?sslmode=disable" \
  go test -tags benchmark -bench=. -benchtime=10s ./internal/runtime/jobstore/

# 只运行 Append benchmark
go test -tags benchmark -bench=BenchmarkAppend -benchtime=10s ./internal/runtime/jobstore/

# 只运行 Claim benchmark
go test -tags benchmark -bench=BenchmarkClaim -benchtime=10s ./internal/runtime/jobstore/
```

### Benchmark 列表

| Benchmark | 测试目标 | 关注指标 |
|-----------|----------|----------|
| `BenchmarkAppend` | 单 goroutine Append 吞吐 | ops/sec, ns/op |
| `BenchmarkAppendParallel` | 多 goroutine 并发 Append（多 Job） | ops/sec, CAS 冲突率 |
| `BenchmarkClaim` | 空轮询 Claim（无 Job 可 Claim） | ops/sec |
| `BenchmarkClaimWithJobs` | 有 Job 时的 Claim 吞吐 | claimed_jobs |
| `BenchmarkClaimConcurrent` | 10 Worker 并发 Claim | claimed_jobs |
| `BenchmarkListEvents` | 50 事件的 ListEvents 查询 | ops/sec |

### 预期发现

- **Append CAS 冲突率**：并发 Append 时 `ErrVersionMismatch` 的频率
- **Claim FOR UPDATE SKIP LOCKED 效果**：多 Worker 并发 Claim 的成功率
- **连接池压力**：高并发下的连接等待时间

---

## k6 端到端压测

### 前置条件

- k6 已安装（`brew install k6`）
- Aetheris API 运行中（`make run` 或 `make docker-run`）

### 运行

```bash
# 默认 10 VUs, 1 分钟
k6 run benchmarks/k6/load-test.js

# 自定义参数
k6 run --env BASE_URL=http://localhost:8080 --env VUS=50 --env DURATION=2m benchmarks/k6/load-test.js

# 50 并发基线测试
k6 run --env VUS=50 --env DURATION=1m benchmarks/k6/load-test.js

# 100 并发压力测试
k6 run --env VUS=100 --env DURATION=2m benchmarks/k6/load-test.js
```

### 测试场景

每个 VU（虚拟用户）循环执行：
1. 10% 概率：健康检查 `GET /api/health`
2. 90% 概率：
   - 创建 Job `POST /api/agents/{id}/message`
   - 等待 500ms
   - 轮询 Job 状态 `GET /api/jobs/{id}`

### Pass/Fail 标准

| 指标 | 50 并发标准 | 100 并发标准 |
|------|------------|-------------|
| 错误率 | < 5% | < 5% |
| Job 创建 P95 | < 500ms | < 1000ms |
| Job 轮询 P95 | < 200ms | < 500ms |
| HTTP P95 | < 500ms | < 1000ms |

### 输出

- 终端：实时指标 + 汇总
- 文件：`benchmarks/reports/k6-summary.json`

---

## 报告格式

每次压测后在 `benchmarks/reports/` 下创建报告：

```
benchmarks/reports/
├── baseline-50.md          # 50 并发基线报告
├── load-100.md             # 100 并发压测报告
├── k6-summary.json         # k6 最近一次运行结果
└── go-benchmark.txt        # Go benchmark 最近一次结果
```
