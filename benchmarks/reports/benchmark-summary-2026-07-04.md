# Benchmark Report — 2026-07-04

> **Operator**: DevOps Engineer (automated)
> **Environment**: Local workstation (macOS, no Docker stack running)
> **Commit**: `main` branch, working tree has uncommitted changes

---

## Go Benchmark Results

**Package**: `internal/runtime/jobstore/`
**Build Tag**: `benchmark`
**File**: `internal/runtime/jobstore/pgstore_bench_test.go`

### Run Output

```
PASS
ok  	github.com/Colin4k1024/Aetheris/v2/internal/runtime/jobstore	0.721s
```

**Status**: SKIPPED — All 6 benchmark functions require a live PostgreSQL connection. The `setupBenchStore` helper calls `b.Skipf` when `BENCHMARK_DSN` is unset or the database is unreachable. No benchmark timings were recorded.

### Available Benchmark Functions (not executed)

| Function | Purpose | Notes |
|----------|---------|-------|
| `BenchmarkAppend` | Single-goroutine Append throughput | Sequential event append to one job |
| `BenchmarkAppendParallel` | Concurrent Append (multi-job) | 100 pre-seeded jobs, parallel goroutines |
| `BenchmarkClaim` | Claim throughput (empty poll) | Measures scheduler poll latency when no jobs available |
| `BenchmarkClaimWithJobs` | Claim throughput (with jobs) | Pre-seeds up to 1000 jobs, measures claim rate |
| `BenchmarkClaimConcurrent` | Multi-worker concurrent Claim | 10 workers, 500 pre-seeded jobs |
| `BenchmarkListEvents` | Event query throughput | 50-event job, repeated ListEvents calls |

### How to Run with Database

```bash
# Start the Docker stack first
make docker-run

# Run benchmarks
go test -tags benchmark -bench=. -benchtime=5s -count=1 ./internal/runtime/jobstore/
```

---

## k6 Load Test Configuration

**Script**: `benchmarks/k6/load-test.js`
**Syntax Check**: PASS (202 lines, brace-balanced, all required exports present)

### Configuration Summary

| Parameter | Default | Override (env) |
|-----------|---------|----------------|
| Target URL | `http://localhost:8080` | `BASE_URL` |
| Virtual Users | 10 | `VUS` |
| Duration (steady state) | 1m | `DURATION` |
| Agent ID | `conversation` | `AGENT_ID` |

### Load Profile

| Phase | Duration | VUs |
|-------|----------|-----|
| Ramp up | 10s | 0 -> VUS |
| Steady state | DURATION | VUS |
| Ramp down | 10s | VUS -> 0 |

### Thresholds

| Metric | Threshold |
|--------|-----------|
| Error rate | < 5% |
| Job creation P95 | < 500ms |
| Job poll P95 | < 200ms |

### Scenarios

1. **Job creation + poll** (90% of iterations): POST to `/api/agents/{id}/message`, then GET `/api/jobs/{id}`
2. **Health check** (10% of iterations): GET `/api/health`

### Custom Metrics

| Metric | Type | Description |
|--------|------|-------------|
| `errors` | Rate | Error rate across all checks |
| `job_creation_duration` | Trend | Job creation latency |
| `job_poll_duration` | Trend | Job poll latency |
| `jobs_created` | Counter | Total jobs successfully created |
| `jobs_completed` | Counter | Jobs reaching `completed` status |
| `jobs_failed` | Counter | Jobs reaching `failed` status |

### Previous Run (2026-06-30, 50 VUs)

Source: `benchmarks/reports/k6-summary.json`

| Metric | Value | Threshold | Status |
|--------|-------|-----------|--------|
| Jobs Created | 4,578 | — | OK |
| Error Rate | 0.00% | < 5% | PASS |
| Job Creation P95 | 19ms | < 500ms | PASS |
| Job Poll P95 | 2ms | < 200ms | PASS |
| HTTP P95 | 16ms | — | OK |
| Throughput | ~76 jobs/s | — | OK |

---

## Grafana Dashboard Panels

**Dashboard**: `Aetheris Agent Runtime`
**UID**: `aetheris-runtime`
**Datasource**: Prometheus
**Auto-refresh**: 10s
**Default time range**: Last 15 minutes

### Panel Inventory (21 panels across 8 row sections)

#### Row 1: Job Throughput & Status

| # | ID | Title | Type | PromQL |
|---|-----|-------|------|--------|
| 1 | 1 | Job Creation / Terminal State Rate | timeseries | `sum by (status) (rate(aetheris_jobs_total[5m]))` |
| 2 | 2 | Current Job State Distribution (stacked) | timeseries | `aetheris_job_state` |
| 3 | 3 | Completed Jobs Count | stat | `sum(aetheris_job_state{state="completed"})` |
| 4 | 4 | Running Jobs Count | stat | `sum(aetheris_job_state{state="running"})` |
| 5 | 5 | Failed Jobs Count | stat | `sum(aetheris_job_state{state="failed"})` |
| 6 | 6 | Stuck Jobs Count | stat | `aetheris_stuck_job_count` |
| 7 | 7 | Job P95 Duration | stat | `histogram_quantile(0.95, rate(aetheris_job_duration_seconds_bucket[5m]))` |

#### Row 2: Scheduler & Lease

| # | ID | Title | Type | PromQL |
|---|-----|-------|------|--------|
| 8 | 8 | Scheduler Tick Duration P50/P95 | timeseries | `histogram_quantile(0.50/0.95, rate(aetheris_scheduler_tick_duration_seconds_bucket[5m]))` |
| 9 | 9 | Lease Acquire / Conflict Rate | timeseries | `rate(aetheris_lease_acquire_total[5m])`, `rate(aetheris_lease_conflict_total[5m])` |

#### Row 3: Rate Limit Wait

| # | ID | Title | Type | PromQL |
|---|-----|-------|------|--------|
| 10 | 10 | Rate Limit Wait Time P99 (LLM & Tool) | timeseries | `histogram_quantile(0.99, rate(aetheris_rate_limit_wait_seconds_bucket{...}[5m]))` |
| 11 | 11 | Rate Limit Rejection Rate | timeseries | `rate(aetheris_rate_limit_rejections_total[5m])` |

#### Row 4: Worker Load

| # | ID | Title | Type | PromQL |
|---|-----|-------|------|--------|
| 12 | 12 | Worker Concurrency | timeseries | `aetheris_worker_busy` |
| 13 | 13 | Step P95 Duration | timeseries | `histogram_quantile(0.95, rate(aetheris_step_duration_seconds_bucket[5m]))` |

#### Row 5: LLM Token Usage

| # | ID | Title | Type | PromQL |
|---|-----|-------|------|--------|
| 14 | 14 | LLM Token Consumption Rate | timeseries | `rate(aetheris_llm_tokens_total{direction="input/output"}[5m])` |
| 15 | 15 | Tool Invocation Rate | timeseries | `rate(aetheris_tool_invocations_total[5m])` |

#### Row 6: Plan & Compile Latency

| # | ID | Title | Type | PromQL |
|---|-----|-------|------|--------|
| 16 | 16 | Plan Generation Duration P50/P95 | timeseries | `histogram_quantile(0.50/0.95, rate(aetheris_plan_duration_seconds_bucket[5m]))` |
| 17 | 17 | TaskGraph Compile Duration P50/P95 | timeseries | `histogram_quantile(0.50/0.95, rate(aetheris_compile_duration_seconds_bucket[5m]))` |

#### Row 7: Node Execution

| # | ID | Title | Type | PromQL |
|---|-----|-------|------|--------|
| 18 | 18 | Node Execution Rate (by type & status) | timeseries | `rate(aetheris_node_execution_total[5m])` |

#### Row 8: Run Control

| # | ID | Title | Type | PromQL |
|---|-----|-------|------|--------|
| 19 | 19 | Run Pause Rate | stat | `rate(aetheris_run_pause_total[5m])` |
| 20 | 20 | Run Resume Rate | stat | `rate(aetheris_run_resume_total[5m])` |
| 21 | 21 | Human Decision Injection Rate | stat | `rate(aetheris_human_decision_total[5m])` |

### Key Metrics Coverage

| Required Metric | Covered | Panel IDs |
|-----------------|---------|-----------|
| Job throughput | Yes | #1, #2, #3, #4, #5 |
| Step latency | Yes | #13 |
| Worker utilization | Yes | #12 |
| Scheduler health | Yes | #8, #9 |
| LLM token usage | Yes | #14, #15 |
| Rate limiting | Yes | #10, #11 |
| Plan/compile latency | Yes | #16, #17 |
| Node execution | Yes | #18 |
| Human-in-the-loop | Yes | #19, #20, #21 |

---

## Production Readiness Evidence

### Summary

| Dimension | Status | Evidence |
|-----------|--------|----------|
| Go benchmark infrastructure | READY | 6 benchmark functions defined, gracefully skip when DB unavailable |
| k6 load test infrastructure | READY | Script syntax valid, 50-VU baseline PASS (2026-06-30) |
| Grafana observability | READY | 21 panels covering all critical runtime dimensions |
| Prometheus metrics | READY | All PromQL queries reference `aetheris_*` namespace |
| Previous load test baseline | PASS | 50 VUs: 0% error rate, 19ms P95 job creation, 76 jobs/s |

### Gaps

1. **Go benchmarks not executed today**: PostgreSQL is not running. To get fresh numbers, run `make docker-run` then re-execute the benchmark command.
2. **100-VU baseline not yet recorded**: The 2026-06-30 baseline was at 50 VUs. A 100-VU run is recommended to validate scaling behavior.
3. **No automated CI integration**: Benchmarks and k6 tests are not yet wired into CI pipeline. Recommend adding a GitHub Actions workflow with threshold gates.

### Recommendations

1. Run the full Docker stack and execute both Go benchmarks and k6 load test to produce a complete baseline with database.
2. Record the 100-VU baseline and compare against the 50-VU results to identify scaling bottlenecks (connection pool, scheduler contention).
3. Add benchmark regression detection to CI (e.g., `benchstat` comparison on PR).
