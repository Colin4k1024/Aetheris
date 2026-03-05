# Fatal Correctness Test Suite

> **Version**: 1.0
> **Purpose**: 确保 Aetheris 的核心保证（At-Most-Once、Crash Recovery、Deterministic Replay）在生产环境中真正成立
> **Reference**: 基于 CTO 架构分析文档

---

## 1. 测试套件概述

### 1.1 核心保证目标

| 保证 | 描述 | 测试覆盖 |
|------|------|----------|
| **At-Most-Once** | 外部副作用（Tool 调用）不会重复执行 | F1-F4 |
| **Crash Recovery** | Worker/API 崩溃后 Job 能恢复继续执行 | F5-F7 |
| **Deterministic Replay** | 回放能准确恢复执行状态 | F8-F10 |
| **Lease Fencing** | 多 Worker 不会同时执行同一 Job | F11-F12 |

### 1.2 测试分类

```
fatal_tests/
├── correctness/        # 正确性测试 (必须通过)
│   ├── at_most_once/   # At-Most-Once 保证
│   ├── crash_recovery/ # 崩溃恢复
│   └── replay/         # 回放验证
├── performance/         # 性能测试 (可选)
│   └── benchmarks/
└── integration/        # 集成测试
    └── multi_worker/
```

---

## 2. At-Most-Once 测试 (F1-F4)

### F1: Worker Crash Before Tool Execution

**场景**: Worker 在执行 Tool 之前崩溃
**预期**: Job 被其他 Worker 接收，Tool 未被执行

```bash
# 伪代码测试逻辑
1. 创建 Job，开始执行
2. 在 Tool 执行前 kill Worker
3. 重启 Worker
4. 验证:
   - Job 状态为 pending/running (非 completed)
   - Tool Ledger 中无该 tool 调用记录
   - Job 最终成功完成
```

### F2: Worker Crash After Tool, Before Commit

**场景**: Tool 已执行但 Ledger 未提交时 Worker 崩溃
**预期**: 重新执行时 Tool 不重复调用，使用 Ledger 恢复结果

```bash
# 伪代码测试逻辑
1. 创建带 Tool 的 Job
2. 拦截 Tool 执行，在返回后、Ledger 提交前 kill Worker
3. 重启 Worker
4. 验证:
   - Tool 只被执行一次 (通过 Tool 侧日志/metrics)
   - Ledger 中有该调用记录
   - Job 最终成功完成
```

### F3: Two Workers Same Step (竞争测试)

**场景**: 两个 Worker 同时尝试认领同一 Job
**预期**: 只有 一个 Worker 获得执行权，另一个被拒绝

```bash
# 伪代码测试逻辑
1. 启动两个 Worker
2. 创建一个 pending Job
3. 两个 Worker 同时尝试 claim
4. 验证:
   - 只有一个 Worker 获得 lease
   - 另一个收到 "lease conflict" 错误
   - Job 只执行一次
```

### F4: Replay Restores Output Without Re-execution

**场景**: 回放时不重新执行 Tool，只用 Ledger 恢复结果
**预期**: 回放结果与原执行一致，无重复 Tool 调用

```bash
# 伪代码测试逻辑
1. 执行 Job (包含 Tool 调用)
2. 获取 Tool 调用记录
3. 执行 Replay
4. 验证:
   - Tool 调用次数为 0 (或最小化)
   - 结果与原执行一致
   - Ledger 被正确读取
```

---

## 3. Crash Recovery 测试 (F5-F7)

### F5: Worker Crash During Job Execution

**场景**: Job 执行过程中 Worker 崩溃
**预期**: Job 被重新调度并完成

```bash
# 已在 scripts/release-p0-drill.sh Drill A 中覆盖
# 测试命令: ./scripts/release-p0-drill.sh
```

### F6: API Restart During Job Execution

**场景**: API 服务重启，Job 正在执行
**预期**: Worker 继续执行，Job 不受影响

```bash
# 已在 scripts/release-p0-drill.sh Drill B 中覆盖
```

### F7: PostgreSQL Short Outage

**场景**: 数据库短暂不可用
**预期**: 恢复后系统继续正常工作

```bash
# 已在 scripts/release-p0-drill.sh Drill C 中覆盖
# 可选测试: RUN_DB_DRILL=1 ./scripts/release-p0-drill.sh
```

---

## 4. Deterministic Replay 测试 (F8-F10)

### F8: Replay Produces Same Output

**场景**: 同一 Job 回放产生相同结果
**预期**: 输入相同则输出相同（确定性）

```bash
# 伪代码测试逻辑
1. 执行 Job A，记录输出 O1
2. 执行 Replay，记录输出 O2
3. 验证: O1 == O2
```

### F9: Replay Preserves Event Chain

**场景**: 回放保持事件链完整性
**预期**: 事件哈希链在回放后仍然一致

```bash
# 伪代码测试逻辑
1. 执行 Job，记录事件链哈希 H1
2. 执行 Replay
3. 验证: 事件链哈希 H2 == H1
4. 验证: GET /api/jobs/:id/verify 返回成功
```

### F10: Replay Handles Partial State

**场景**: 回放时部分状态丢失
**预期**: 能从事件流重建完整状态

```bash
# 伪代码测试逻辑
1. 执行多步 Job
2. 删除 Checkpoint (保留 Event Store)
3. 执行 Replay
4. 验证: 能恢复到最终状态
```

---

## 5. Lease Fencing 测试 (F11-F12)

### F11: Lease Expiry During Execution

**场景**: Worker 执行超时，lease 过期
**预期**: Job 被重新调度，不产生重复执行

```bash
# 伪代码测试逻辑
1. 设置短 lease_duration
2. 开始执行长时间 Job
3. 等待 lease 过期
4. 验证:
   - 原 Worker 失去执行权
   - 新 Worker 获得执行权
   - Tool 未重复执行
```

### F12: Worker Heartbeat Failure

**场景**: Worker 心跳停止，系统检测到故障
**预期**: Job 被重新调度

```bash
# 伪代码测试逻辑
1. Worker 持有 Job lease
2. 停止 Worker 心跳 (模拟网络分区)
3. 等待 lease 过期
4. 验证: Job 被其他 Worker 接收
```

---

## 6. Step Contract 强制测试 (S1-S4)

### S1: No Side Effects in Step (静态分析)

**场景**: 检测 Step 中是否存在非法副作用
**预期**: 检测到 time.Now(), rand, net/http 等调用

```go
// 建议实现: Step Linter
func TestStepLinter_DetectsForbiddenCalls(t *testing.T) {
    // 扫描常见危险调用
    forbidden := []string{
        "time.Now()",
        "rand.Intn()",
        "net/http.Get()",
        "os.Create()",
        "goroutine",
    }

    for _, call := range forbidden {
        code := fmt.Sprintf(`func Step() { %s }`, call)
        violations := linter.Analyze(code)
        assert.NotEmpty(t, violations, "should detect %s", call)
    }
}
```

### S2: Tool Execution via Tool Path Only

**场景**: 验证所有外部调用都经过 Tool 接口
**预期**: 直接调用外部服务被拒绝

### S3: Deterministic Step Verification

**场景**: 验证 Step 的确定性
**预期**: 相同输入产生相同输出

### S4: Effect Store Commit Order

**场景**: 验证 Effect Store 提交顺序
**预期**: Ledger 先于 Effect Store，或按协议顺序

---

## 7. 事件模型测试 (E1-E4)

### E1: Event Schema Versioning

**场景**: 事件 schema 版本升级
**预期**: 旧事件能被新版本正确解析

### E2: Event Chain Integrity

**场景**: 事件链哈希完整性
**预期**: 每一事件包含前驱哈希

### E3: Fact vs Telemetry Events

**场景**: 区分事实事件和观测事件
**预期: 事实事件影响状态重建，观测事件可丢弃

### E4: Evidence Event Audit

**场景**: 审计事件完整性
**预期**: 关键操作都有审计事件记录

---

## 8. 运行测试

### 8.1 本地运行

```bash
# 运行所有正确性测试
go test -v ./fatal_tests/correctness/...

# 运行 At-Most-Once 测试
go test -v -run "AtMostOnce" ./...

# 运行崩溃恢复测试
go test -v -run "CrashRecovery" ./...

# 运行致命测试 (集成)
go test -v -tags=fatal ./internal/agent/runtime/executor/
```

### 8.2 CI Release Gates

```bash
# 必须在 release 前通过
make test-fatal

# 等价于
go test -race -count=1 ./fatal_tests/correctness/...
```

### 8.3 P0 故障演练

```bash
# 运行 P0 故障演练 (需要完整栈)
./scripts/release-p0-drill.sh

# 带数据库演练
RUN_DB_DRILL=1 ./scripts/release-p0-drill.sh
```

---

## 9. 测试矩阵

| 测试 | 类型 | 复杂度 | 失败影响 | CI Gate |
|------|------|--------|----------|---------|
| F1-F4 | At-Most-Once | 高 | 数据重复 | **必须** |
| F5-F7 | Crash Recovery | 中 | 服务中断 | **必须** |
| F8-F10 | Replay | 高 | 审计失败 | **必须** |
| F11-F12 | Lease Fencing | 中 | 竞争条件 | **必须** |
| S1-S4 | Step Contract | 中 | 违反合约 | 推荐 |
| E1-E4 | Event Model | 低 | 兼容性问题 | 推荐 |

---

## 10. 附录

### A. 测试工具

- `scripts/release-p0-drill.sh` - P0 故障演练脚本
- `scripts/release-p0-perf.sh` - 性能基线测试
- `internal/agent/runtime/executor/ledger_1_0_test.go` - Ledger 单元测试

### B. 相关设计文档

- `design/step-contract.md` - Step 合约规范
- `design/1.0-runtime-semantics.md` - 运行时语义
- `design/scheduler-correctness.md` - 调度器正确性

### C. 相关 Issue

- See: [Release Checklist](./release-checklist-2.0.md)
- See: [Execution Guarantees](./execution-guarantees.md)
