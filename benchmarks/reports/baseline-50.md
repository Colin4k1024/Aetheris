# Aetheris Load Test Baseline — 50 VUs

> **日期**: 2026-06-30
> **环境**: Docker Compose v2 (PostgreSQL + API + 2 Workers + Prometheus + Grafana + Jaeger)
> **工具**: k6 v2.0.0
> **配置**: 50 VUs, 1 分钟稳态

---

## 结果摘要

| 指标 | 值 | 标准 | 状态 |
|------|-----|------|------|
| Jobs Created | 4,578 | — | ✅ |
| Error Rate | 0.00% | < 5% | ✅ PASS |
| Job Creation P95 | 19ms | < 500ms | ✅ PASS |
| Job Creation P99 | < 1ms | — | ✅ |
| Job Poll P95 | 2ms | < 200ms | ✅ PASS |
| HTTP P95 | 16ms | < 500ms | ✅ PASS |
| HTTP P99 | < 1ms | — | ✅ |
| Total HTTP Requests | 9,662 | — | ✅ |
| Throughput | ~76 jobs/s | — | ✅ |
| Jobs Failed | 4 | — | ⚠️ 低失败率 |

## 详细指标

- **吞吐量**: 4,578 jobs / 60s = ~76 jobs/s
- **HTTP 吞吐**: 9,662 req / 60s = ~161 req/s
- **Job 创建延迟**: P50 < 1ms, P95 = 19ms
- **Job 轮询延迟**: P95 = 2ms
- **失败率**: 4 / 4,578 = 0.09%

## Pass/Fail 评估

**结论: PASS** — 所有指标均在 50 并发标准范围内。

## 后续

- 100 并发压测待执行（预期 P95 可能升至 30-50ms）
- Worker OTel 变更部署后可验证 Jaeger 完整 trace
- 连接池压力需在 100 并发时观察
