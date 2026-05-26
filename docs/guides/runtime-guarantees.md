# Aetheris Runtime Guarantees — 你可以依赖什么

This document explains what Aetheris guarantees in normal and failure scenarios. For a mode-by-mode summary, start with the [guarantee matrix](guarantee-matrix.md). For technical details, see [design/execution-guarantees.md](../../design/execution-guarantees.md) and [design/scheduler-correctness.md](../../design/internal/scheduler-correctness.md).

For startup checks that enforce production-safe storage and API settings, see [production runtime gates](production-runtime-gates.md).

---

## Normal Scenario Guarantees

These guarantees are conditional. They hold only for the runtime boundary Aetheris controls and only when the required durable stores are configured.

| Scenario                    | Guarantee                                              | Condition                                                     |
| --------------------------- | ------------------------------------------------------ | ------------------------------------------------------------- |
| **Worker executes step**    | Recorded Runtime Tool side effect is not repeated      | Configured InvocationLedger + Effect Store                    |
| **Tool calls external API** | Pass idempotency key → downstream dedup                | Tool uses `StepIdempotencyKeyForExternal(ctx, jobID, stepID)` |
| **LLM generates result**    | Stored in Effect Store; Replay injects (NOT re-called) | Effect Store configured                                       |
| **Signal sent**             | At-least-once (write wait_completed → Job scheduled)   | WakeupQueue for multi-worker                                  |
| **Checkpoint saved**        | After each step; crash recovery from latest Checkpoint | CheckpointStore configured                                    |

## Guarantee Boundary by Mode

| Mode | Intended use | What holds | What does not hold |
| --- | --- | --- | --- |
| Embedded / memory | Local demos and tests | Basic execution and local trace | Cross-process crash recovery, production at-most-once side-effect boundary |
| `external_http` | Level 1 migration for existing agents | Durable outer Job, trace, timeout, retry, forwarded idempotency key | Internal payment/email/database writes inside the external agent are opaque to Aetheris |
| Native Runtime Tool | Production side-effect boundary | Ledger/Effect Store can prevent repeated committed Runtime Tool side effects | Downstream systems must still honor idempotency for their own write semantics |
| Postgres multi-worker | Production deployment | Lease fencing, recovery, replay, shared ledger/effects | Storage disaster recovery remains infrastructure responsibility |

### DAG parallel execution (2.0)

When **max parallel steps** is configured (> 0), steps in the same topological level may run in parallel on a single worker. See [design/dag-parallel-execution.md](../../design/internal/dag-parallel-execution.md).

| Aspect                   | Behavior                                                                                                                                                                                         |
| ------------------------ | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| **Max concurrency**      | Configurable per Runner (`SetMaxParallelSteps(n)`); 0 = sequential (default).                                                                                                                    |
| **Failure**              | If any step in a level fails, the level is failed; the job is marked failed and no results from that level are committed. Other in-flight steps in the level are effectively canceled (context). |
| **Replay / determinism** | Results are merged by node ID in sorted order; NodeStarted/NodeFinished are written in deterministic order. Replay and checkpoint semantics remain the same.                                     |
| **Wait nodes**           | Levels that contain a Wait node are run sequentially (one step at a time) so Wait semantics are unchanged.                                                                                       |

---

## Failure Scenarios

### 1. Worker Crash (Medium Frequency)

**Scenario**: Worker terminates mid-execution (process killed, machine power loss, network disconnect)

**What happens**:

1. Worker holds Job lease with `attempt_id` and heartbeat (30s TTL)
2. **Crash** → heartbeat stops → lease expires after 30s
3. **Scheduler Reclaim**: Scans expired leases, sets Job back to Pending
4. **Another Worker claims**: Loads from latest Checkpoint
5. **Replay**: Already-completed steps injected from event stream (no re-execution)
6. **Tool side effect**: If Tool executed but crash before `command_committed`:
   - Effect Store has record → catch-up: append event without re-executing Tool
   - Ledger/InvocationStore recovery flow ensures Tool NOT re-executed

**Guarantees**:

- ✅ Job not lost (Reclaim ensures eventual progress)
- ✅ Step not duplicated (Ledger + Effect Store at-most-once)
- ✅ Maximum loss: Progress of last step (redo from Checkpoint, but Replay injects recorded results)

**Example**:

```
Worker A: query_order (success) → llm_decide (success) → send_refund (executing...) → CRASH
Worker B: Reclaim → Replay: query_order, llm_decide injected → send_refund: check Ledger → injected (NOT re-executed)
Result: Refund sent once
```

**Configuration requirements**:

- JobStore: Postgres (or shared store)
- Event Store: Postgres (lease management)
- InvocationLedger: Enabled
- Effect Store: Enabled

---

### 2. Step Timeout (High Frequency)

**Scenario**: Step execution exceeds configured timeout (e.g., 5 minutes)

**What happens**:

1. Runner wraps step execution in `context.WithTimeout(stepTimeout)`
2. **Timeout** → context canceled → step returns `context.DeadlineExceeded`
3. **Classification**: `StepResultRetryableFailure`
4. **Job status**: Set to Failed (or Requeue based on retry policy)
5. **Tool partial execution**: If Tool started but not committed:
   - Next replay: Activity Log Barrier (event stream has tool_invocation_started, no finished)
   - **Block re-execution**: Recover from Ledger or fail permanently

**Guarantees**:

- ✅ Timeout does NOT cause "step half-done, state inconsistent"
- ✅ Tool already called but uncommitted → NOT re-executed (Activity Log Barrier)

**Configuration**:

```yaml
# configs/worker.yaml
executor:
  step_timeout: "5m" # Per-step timeout
```

**Example**:

```
Step: call_slow_api (timeout 5m)
3 min: API responding...
5 min: Timeout → context canceled
→ Step classified as retryable_failure
→ Job Failed (or Requeue if retry_max > 0)
→ Tool invocation recorded as "started" but not "finished"
→ Replay: Block re-execution (wait for recovery or fail)
```

---

### 3. Signal Lost (Low Risk)

**Scenario**: POST `/api/jobs/:id/signal` request fails (network down, API crash)

**What happens**:

1. **If `wait_completed` NOT written**: Job remains StatusWaiting/StatusParked
2. **Retry signal**: Idempotent (see [design/runtime-contract.md](../../design/internal/runtime-contract.md) § External Event Guarantee)
   - If last event already `wait_completed` with same `correlation_key` → return 200 (already delivered)
3. **WakeupQueue**: If configured, signal → NotifyReady → Worker immediately claims (no poll delay)

**Guarantees**:

- ✅ Signal at-least-once (once wait_completed written, Job WILL be scheduled)
- ✅ Duplicate signal idempotent (no double-unblock)

**Recovery**:

```bash
# Check if signal delivered
curl -s http://localhost:8080/api/jobs/job-xxx/replay | jq '.events[] | select(.type=="wait_completed")'

# If empty → signal not delivered, re-send:
curl -X POST http://localhost:8080/api/jobs/job-xxx/signal \
  -d '{"correlation_key": "approval-xxx", "payload": {"approved": true}}'

# If returns 200 → delivered (even if duplicate)
```

---

### 4. Two Workers Execute Same Step (Very Low Risk)

**Scenario**: Lease just expired, two Workers simultaneously claim same Job

**What happens**:

1. **Event Store Append**: Validates `attempt_id` (see [design/runtime-contract.md](../../design/internal/runtime-contract.md) § Execution Epoch)
   - Only Worker with current lease's `attempt_id` can write events
   - Other Worker gets `ErrStaleAttempt` → aborts
2. **Tool Ledger Acquire**: Same `idempotency_key` → only one Worker can Commit
   - Worker A: Acquire → AllowExecute → execute tool → Commit
   - Worker B: Acquire → WaitOtherWorker or ReturnRecordedResult

**Guarantees**:

- ✅ Tool result is committed once for a stable `idempotency_key`
- ⚠️ Tool execution itself still depends on adapter-side idempotency if the external system is called before a durable result is committed
- ✅ Event stream not polluted by stale Worker (attempt_id validation)

**Example**:

```
Time 0s: Worker A claims job-123 (attempt_id=attempt-1, lease expires at 30s)
Time 30s: Lease expires
Time 30.1s: Worker B claims job-123 (attempt_id=attempt-2)
Time 30.2s: Worker A tries to append event → ErrStaleAttempt (attempt-1 != attempt-2)
Time 30.3s: Worker B executes tool → Ledger Acquire → AllowExecute → success
Result: one committed Tool result for this runtime key
```

---

### 5. LLM Model Update (Medium Impact)

**Scenario**: First execution uses gpt-4o-2024-08-06; replay time uses gpt-4o-2024-11-20

**What happens**:

1. **First execution**: LLM called → response recorded to Effect Store (with model metadata)
2. **Replay**: Effect Store has record → inject response → **LLM NOT called**
3. **Trace**: Shows `llm_model: gpt-4o-2024-08-06` (original)
4. **Warning log**: Model version changed (if version tracking configured)

**Guarantees**:

- ✅ Replay result matches first execution (not affected by model update)
- ✅ Audit can trace "which model was used during execution"

**Example**:

```
First exec (2024-08-06): LLM(model=gpt-4o-2024-08-06) → "Approve"
Replay (2024-11-20): Inject "Approve" from Effect Store (model=gpt-4o-2024-08-06)
→ Replay output = "Approve" (even if new model would return "Reject")
→ Deterministic
```

---

### 6. Tool Schema Change (Medium Impact)

**Scenario**: Tool API updated (e.g., `/api/price?sku=123` returns `$10` → `$12`)

**What happens**:

1. **First execution**: Tool called → result `$10` recorded
2. **Replay**: Inject `$10` from Ledger/Effect Store → Tool NOT re-called
3. **Versioning**: tool_invocation_started includes `tool_version`, `request_schema_hash` (see [design/versioning.md](../../design/internal/versioning.md))
4. **Audit**: Can explain "why historical execution returned X, now returns Y" (tool version changed)

**Guarantees**:

- ✅ Replay uses recorded result (not affected by tool schema change)
- ✅ Version tracking for audit (tool_version, schema_hash)

---

### 7. Database Transaction Rollback (Edge Case)

**Scenario**: Tool writes to database, transaction commits, but process crashes before writing command_committed

**What happens**:

1. **Tool executed**: Database transaction committed (external side effect done)
2. **Crash**: Before writing `command_committed` to event stream
3. **Effect Store** (if enabled): Tool result already written (two-phase commit)
4. **Replay**: Effect Store has record → catch-up: append command_committed without re-executing Tool
5. **Without Effect Store**: Activity Log Barrier (tool_invocation_started, no finished) → Block re-execution → Recover from Ledger or fail

**Guarantees**:

- ✅ With Effect Store: Catch-up writes event, Tool NOT re-executed (two-phase commit)
- ✅ Without Effect Store: Block re-execution (Activity Log Barrier), wait for manual recovery

**Prevention**: Always configure Effect Store for production.

---

### 8. Network Partition (Split Brain)

**Scenario**: Worker A loses network to JobStore, Worker B claims same Job

**What happens**:

1. Worker A: Executing, but cannot heartbeat → lease expires
2. Worker B: Claims Job (new `attempt_id`)
3. Worker A: Network recovers, tries to append event → `ErrStaleAttempt` (attempt_id mismatch)
4. Worker A: Aborts execution
5. Worker B: Continues from Checkpoint

**Guarantees**:

- ✅ Only one Worker can progress (attempt_id validation)
- ✅ Tool executed once (Ledger arbitration)

**Configuration**: Ensure JobStore/Event Store accessible to all Workers (shared Postgres, not local file).

---

## Configuration Requirements

### Development Mode (Minimal)

```yaml
jobstore:
  type: memory # In-memory, no persistence

effect_store:
  enabled: false # Optional

invocation_ledger:
  enabled: false # Optional
```

**Guarantees**: Basic execution, no crash recovery, no at-most-once (tools may duplicate on retry)

**Use for**: Local testing, prototyping

---

### Production Mode (Recommended)

```yaml
jobstore:
  type: postgres
  postgres:
    dsn: "postgres://aetheris:aetheris@localhost:5432/aetheris"

effect_store:
  enabled: true # ← Required for at-most-once & LLM replay guard
  type: postgres

invocation_ledger:
  enabled: true # ← Required for Tool at-most-once
  type: postgres

wakeup_queue:
  type: redis # ← Required for multi-worker signal delivery
  redis:
    addr: "redis:6379"
```

**Guarantees**: All guarantees in [design/execution-guarantees.md](../../design/execution-guarantees.md) hold for native Runtime Tools and Aetheris-managed effects. `external_http` internals remain outside the runtime boundary.

**Use for**: Production deployment, multi-worker, crash recovery, audit

---

## Failure Matrix (Formal)

A single **fault × guarantee × behavior × config** matrix is maintained in [design/failure-matrix.md](../../design/internal/failure-matrix.md). Use it for compliance and operations.

## Guarantee Summary Table

| What Could Go Wrong      | What Happens                     | Guarantee              | Required Config         |
| ------------------------ | -------------------------------- | ---------------------- | ----------------------- |
| Worker crash             | Reclaim → another Worker resumes | Job not lost           | Postgres JobStore       |
| Worker crash during Tool | Effect Store catch-up            | Tool NOT re-executed   | Effect Store            |
| Step timeout             | Classified as retryable_failure  | Timeout safe           | step_timeout configured |
| Signal lost              | Retry signal (idempotent)        | At-least-once delivery | (always)                |
| Two Workers same step    | attempt_id validation            | Only one succeeds      | Event Store + Ledger    |
| LLM model update         | Replay injects old output        | Deterministic          | Effect Store            |
| Tool schema change       | Replay injects old result        | Audit traceable        | tool_version tracking   |
| Database rollback        | Catch-up or barrier              | Tool NOT re-executed   | Effect Store            |
| Network partition        | attempt_id mismatch              | Split-brain safe       | Shared JobStore         |

---

## Testing Failure Scenarios

### Test 1: Worker Crash During Tool

```bash
# Terminal 1: Start API
go run ./cmd/api

# Terminal 2: Start Worker
go run ./cmd/worker

# Terminal 3: Create job with long-running tool
curl -X POST http://localhost:8080/api/agents/test-agent/message \
  -d '{"message": "test crash during tool"}'

# Watch Worker logs → when tool starts executing, kill Worker (Ctrl+C)

# Terminal 4: Start new Worker
go run ./cmd/worker

# Observe:
# - New Worker claims job
# - Replay: Tool result injected (NOT re-executed)
# - Job completes
# - Check Ledger: only one invocation record
```

**Expected**: Tool executed once, no duplicate side effect.

---

### Test 2: Signal Lost (Retry)

```bash
# Create job with Wait node (legacy facade example)
POST /api/agents/agent-1/message

# Job enters StatusParked

# Send signal (simulate network failure by killing API mid-request)
POST /api/jobs/job-xxx/signal
→ 500 Internal Server Error (API crashed)

# Restart API
go run ./cmd/api

# Retry signal (idempotent)
POST /api/jobs/job-xxx/signal
→ 200 OK

# Check events
GET /api/jobs/job-xxx/replay
→ Only ONE wait_completed (not duplicate)
```

**Expected**: Signal idempotent, Job resumes once.

---

### Test 3: Two Workers Same Job

```bash
# Configure short lease TTL (for testing)
# worker.yaml: lease_ttl: "5s"

# Start 2 Workers
# Terminal 1: go run ./cmd/worker
# Terminal 2: go run ./cmd/worker

# Create job (legacy facade example)
POST /api/agents/agent-1/message

# Observe logs:
# - Worker A claims (attempt_id=attempt-1)
# - Lease expires (5s)
# - Worker B claims (attempt_id=attempt-2)
# - Worker A tries to append → ErrStaleAttempt → aborts
# - Worker B continues

# Check Tool invocations
GET /api/jobs/job-xxx/trace
→ Only ONE tool_invocation_finished (not duplicate)
```

**Expected**: Tool executed once, only one Worker succeeded.

---

## Verification Mode

After a Job completes (or fails), you can run **offline verification** to get an auditable proof summary. This supports compliance and demonstrates "provable execution correctness."

### How to use

- **CLI**: `aetheris verify <job_id>` — prints execution hash, event chain root hash, ledger proof, and replay proof.
- **API**: `GET /api/jobs/:id/verify` — returns the same four outputs as JSON.

### Output meanings

| Output                           | Meaning                                                                                                                              |
| -------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------ |
| **Execution hash**               | Deterministic digest of the execution path (plan + node_id/result_type sequence).                                                    |
| **Event chain root hash**        | Root hash of the event stream in order; any tampering or reorder changes this value.                                                 |
| **Tool invocation ledger proof** | Confirms every tool_invocation_started has a matching tool_invocation_finished (at-most-once). `ok: true` means no dangling started. |
| **Replay proof result**          | Read-only Replay (BuildFromEvents) succeeded; context is consistent with the event stream.                                           |

### Relationship to Execution Proof Chain

The **Execution Proof Chain** (Ledger, Confirmation Replay) is the _runtime_ mechanism that enforces at-most-once and deterministic replay. **Verification Mode** is a _post-hoc_, read-only check and summary of the same event stream and derived state. It does not change any state; it only computes and returns hashes and proof results for audit or demo.

See [design/verification-mode.md](../../design/internal/verification-mode.md) for the protocol (event chain digest, execution hash formula) and [design/1.0-runtime-semantics.md](../../design/internal/1.0-runtime-semantics.md) for the Execution Proof Chain.

---

## When Guarantees Do NOT Hold

### Scenario 1: Effect Store Not Configured

**Without Effect Store**:

- ❌ at-most-once NOT guaranteed for Tools (may duplicate on crash before commit)
- ❌ LLM replay guard weakened (Adapter layer still checks, but no two-phase commit)

**Use for**: Development only; **DO NOT use in production**

---

### Scenario 2: Ledger Not Configured

**Without InvocationLedger**:

- ❌ Tool at-most-once NOT guaranteed across Workers
- ❌ Two Workers may execute same Tool (if both claim simultaneously)

**Use for**: Single-worker dev; **DO NOT use in multi-worker production**

---

### Scenario 3: WakeupQueue Not Configured (Multi-Worker)

**Without WakeupQueue**:

- ⚠️ Signal delivery has delay (poll interval, default 2s)
- ⚠️ Under high load (1k+ parked jobs), poll inefficient

**Use for**: Single-worker or low-load; multi-worker production should configure Redis/Postgres WakeupQueue

---

### Scenario 4: Violating Step Contract

**If developer breaks [design/step-contract.md](../../design/internal/step-contract.md)**:

- ❌ Step directly calls external API (not via Tool) → Replay will re-execute → duplicate side effect
- ❌ Step reads `time.Now()` or `rand.Int()` → Replay non-deterministic
- ❌ Step modifies global state → Replay behavior unpredictable

**Fix**: Follow Step Contract (external side effects must go through Tools).

---

## Disaster Recovery

### Scenario: Total Data Loss (Postgres Crash)

**If JobStore/Event Store lost**:

- ❌ All Jobs lost (no recovery)
- ❌ Execution history lost (no audit)

**Prevention**:

- **Database backups**: pg_dump or continuous archiving
- **Replicas**: Postgres streaming replication
- **Disaster recovery plan**: RTO/RPO requirements

Aetheris provides runtime guarantees **given persistent storage**; storage layer disaster recovery is infrastructure responsibility.

---

### Scenario: Ledger Corruption

**If InvocationLedger inconsistent with event stream**:

- ⚠️ Tool may be blocked (Ledger says "in progress", event stream says "finished")
- ⚠️ Manual intervention: Query Ledger + Event Store, reconcile

**Prevention**:

- Use transactional stores (Postgres Ledger + Event Store in same DB)
- Regular consistency checks:
  - API: `GET /api/forensics/consistency/:job_id`
  - Offline: `aetheris export <job_id>` + `aetheris verify <evidence.zip>`

---

## SLA & Performance Characteristics

### Latency

| Operation                        | Latency             | Note                                          |
| -------------------------------- | ------------------- | --------------------------------------------- |
| **Job creation (legacy facade)** | ~50ms               | POST /api/agents/:id/message → job_id         |
| **Worker claim**                 | ~20ms               | Poll JobStore, return Pending job             |
| **Step execution**               | Depends on Tool/LLM | Aetheris overhead ~5ms per step               |
| **Signal delivery**              | <100ms              | With WakeupQueue; ~2s without (poll interval) |
| **Checkpoint save**              | ~30ms               | Write to Postgres                             |
| **Replay**                       | ~50ms per 100 steps | Read event stream, inject results             |

### Throughput

| Metric                | Value           | Note                                           |
| --------------------- | --------------- | ---------------------------------------------- |
| **Jobs/sec**          | ~100-500        | Single API, Postgres JobStore                  |
| **Concurrent Jobs**   | ~10k+           | With StatusParked (long-wait jobs don't block) |
| **Workers**           | Scales linearly | Add Workers → higher throughput                |
| **Event stream size** | ~1KB per step   | 100-step job ≈ 100KB events                    |

### Resource Usage

| Component    | Memory               | CPU          | Disk                                |
| ------------ | -------------------- | ------------ | ----------------------------------- |
| **API**      | ~100MB               | ~5%          | Minimal (logs only)                 |
| **Worker**   | ~50MB per job        | ~20% per job | Minimal                             |
| **Postgres** | Depends on job count | ~10%         | ~1MB per job (events + checkpoints) |

**Scaling**: Horizontal (add Workers) + Vertical (Postgres resources)

---

## Production Checklist

Before deploying Aetheris to production:

- [ ] **Effect Store enabled** (at-most-once guarantee)
- [ ] **InvocationLedger enabled** (Tool at-most-once)
- [ ] **JobStore = Postgres** (crash recovery)
- [ ] **WakeupQueue configured** (multi-worker signal delivery)
- [ ] **Postgres backups** (disaster recovery)
- [ ] **Monitoring** (Prometheus metrics, see [observability.md](observability.md))
- [ ] **Step timeout configured** (prevent hung steps)
- [ ] **Retry policy configured** (max retries, backoff)
- [ ] **All Tools follow Step Contract** (no direct external calls, use idempotency key)
- [ ] **Test failure scenarios** (worker crash, step timeout, signal retry)

---

## Distributed Consistency Test Runbook

Use these tests to validate multi-worker and replay consistency guarantees in CI or pre-release checks.

### 1) Ledger + Tool at-most-once stress (single package)

```bash
go test ./internal/agent/runtime/executor -run 'TestStress_MultiWorkerRace|TestStress_ManyJobs|TestStress_CrashAfterToolBeforeCommit|TestLedger_3_DoubleWorker_OnlyOneCommit' -v
```

Expected:

- No duplicate committed tool result for the same idempotency key.
- Recovery path returns recorded result instead of re-executing tool side effects.

### 2) Reclaim and second-worker resume

```bash
go test ./internal/agent/job -run 'TestHA_ReclaimThenSecondWorkerCanClaim|TestScanAndReclaimExpired_WithAndWithoutBlocking' -v
go test ./internal/api/http -run 'TestJobSignal_SuccessAndIdempotentUnblock|TestJobSignal_WrongCorrelationKey|TestJobSignal_MissingCorrelationKey|TestJobSignal_ConcurrentVersionConflictStillIdempotent' -v
```

Expected:

- Expired running jobs are reclaimed to Pending.
- Another worker can claim and continue after reclaim.
- Blocked jobs are not reclaimed until unblocked.
- Signal delivery writes `wait_completed` and duplicate delivery with same `correlation_key` is idempotent (no duplicate unblock event).
- Fault-injection (version conflict / concurrent append) still converges to idempotent signal delivery.

### 3) Step timeout + retry policy semantics

```bash
go test ./internal/agent/runtime/executor -run 'TestRunnerParallelLevel_TimeoutClassifiedAsRetryableFailure|TestToolNodeAdapter_RetryPolicyBackoff|TestToolNodeAdapter_RetryPolicyStopsOnNonRetryable' -v
```

Expected:

- `context.DeadlineExceeded` is mapped to `retryable_failure` with reason `step timeout`.
- Tool retries honor `RetryPolicy.MaxRetries` and `RetryPolicy.Backoff`.
- Non-retryable errors stop immediately.

### 4) Full regression (recommended before release)

```bash
go test ./...
go test -race ./...
```

Expected:

- All runtime, reclaim, replay, and tool-contract tests pass under normal and race mode.

---

## References

- [design/execution-guarantees.md](../../design/execution-guarantees.md) — Formal guarantees table
- [design/scheduler-correctness.md](../../design/internal/scheduler-correctness.md) — Lease, heartbeat, reclaim
- [design/step-contract.md](../../design/internal/step-contract.md) — How to write correct steps
- [design/effect-system.md](../../design/internal/effect-system.md) — Effect Store, replay, two-phase commit
- [design/runtime-contract.md](../../design/internal/runtime-contract.md) — Blocking, epoch, attempt_id validation

---

## 场景深化：At-Most-Once 在真实业务中如何落地

下面三个场景展示 At-Most-Once 保证在真实业务环境下的完整工作流，帮助开发者理解「为什么这样设计」以及「怎么验证它确实工作了」。

---

### 场景 A：支付 Agent 崩溃恢复（金融电商）

**业务背景**：电商退款 Agent 需要调用第三方支付 SDK 发起退款，退款金额最高 ¥100,000。

**崩溃时序**：

```
Worker A 执行 step=refund_execute：
  T=0ms:   Tool.Execute(PaymentSDK.Refund, idempotency_key="job-abc-refund-1") 开始
  T=200ms: 第三方支付 SDK 返回成功 {txn_id: "TXN9001", status: "success"}
  T=201ms: Effect.Put("job-abc-refund-1", {txn_id: "TXN9001"})  ← 持久化结果
  T=202ms: Worker 进程被 OOM Killer 强杀                         ← 崩溃点
  T=203ms: EventStore.Append(command_committed) 未执行

Scheduler 等待 30s 租约过期：
  T=30s:   Reclaim：job 状态重置为 Pending

Worker B 认领：
  T=30.1s: Replay 已完成步骤
  T=30.2s: 到达 step=refund_execute
  T=30.3s: Ledger.Authorize("job-abc-refund-1") → 已存在
  T=30.4s: Effect Store 查询 → 有结果 {txn_id: "TXN9001"}
  T=30.5s: catch-up: Append(command_committed) ← 补写事件
  T=30.6s: 注入结果，继续下一步（发送确认邮件）

最终结果：退款只执行了一次（TXN9001），没有重复扣款。
```

**如何验证**：

```bash
# 查看 Ledger 只有一条 committed 记录
curl http://localhost:8080/api/jobs/job-abc/trace | \
  jq '[.events[] | select(.type=="tool_invocation_finished")] | length'
# 预期输出: 1

# 查看 Effect Store 的 catch-up 记录
curl http://localhost:8080/api/jobs/job-abc/replay | \
  jq '.events[] | select(.type=="command_committed" and .step=="refund_execute")'
# 预期: 只有一条，timestamp 在原始崩溃时间之后（catch-up 写入）
```

---

### 场景 B：供应链询价 Agent 并发调用 30 个供应商（制造业）

**业务背景**：采购 Agent 并发向 30 个供应商发送 RFQ（询价单），每个 API 调用对应一个独立 Step。网络抖动导致部分请求超时，Worker 在重试中途崩溃。

**关键设计**：每个供应商调用使用独立 idempotency_key：

```go
// Tool 实现示例
func (t *SendRFQTool) Execute(ctx context.Context, sess *session.Session, input map[string]any, state interface{}) (any, error) {
    supplierID := input["supplier_id"].(string)
    
    // idempotency_key 由 runtime 从 ctx 提供，绑定到 (jobID, stepID)
    // 相同的 (job, step, supplier) 组合永远只调用一次
    key := effects.StepIdempotencyKeyForExternal(ctx, sess.JobID, sess.StepID)
    
    resp, err := t.rfqClient.SendRFQ(ctx, supplierID, input["rfq"].(RFQPayload), key)
    if err != nil {
        return nil, err
    }
    return resp, nil
}
```

**崩溃恢复后的状态**：

```
崩溃前已完成：供应商 1-18（StepCompleted 在事件流中）
崩溃时进行中：供应商 19（Ledger 有 Authorize，Effect Store 有结果）
崩溃时未开始：供应商 20-30

恢复后：
- 供应商 1-18：从事件流注入结果，API 不重调
- 供应商 19：catch-up 补写，API 不重调
- 供应商 20-30：正常执行，首次调用

结果：30 个供应商各收到且仅收到一份 RFQ。
```

---

### 场景 C：医疗 AI 诊断报告写入 HIS（医疗）

**业务背景**：AI 分析患者检查数据后，生成结构化诊断建议并写入 HIS 系统。HIS API 本身不提供幂等保证，重复写入会创建两条诊断记录。

**两步提交保护**：

```
Step=generate_and_write_diagnosis：
  阶段 1（Effect.Put）：
    LLM 生成诊断建议 → 内容持久化到 Effect Store
    key="job-p001-diagnosis-v1"
    value={diagnosis: "建议复查...", generated_at: "2026-05-07T10:30:00Z"}

  阶段 2（执行写入）：
    使用 Effect Store 中的内容调用 HIS API
    HIS.WriteRecord(patient_id="P001", content=effect_value)
    → 成功

  阶段 3（EventStore.Append）：
    Append(command_committed, step=generate_and_write_diagnosis)

如果崩溃发生在阶段 2 和 3 之间：
  恢复时：Effect Store 查询 → 有内容 → catch-up 补写事件
  HIS API：带同一 idempotency_key 重发（HIS 层通过 key 去重）
  结果：患者记录唯一，诊断内容一致
```

---

### 场景间共同规律

| 场景 | 关键保护机制 | 崩溃后恢复路径 |
|------|------------|-------------|
| 支付退款 | Ledger + Effect Store | catch-up 注入，不重调支付 SDK |
| 供应链 RFQ | 独立 idempotency_key per 供应商 | 已完成的 step 注入，未完成的继续 |
| 医疗 HIS 写入 | 两步提交（先 Effect.Put 再 Append） | Effect Store 有内容，补写事件不重调 |

**核心不变量**：只要 Effect.Put 成功，就算 EventStore.Append 还没执行，恢复路径也能通过 catch-up 保证 Tool 不被重新执行。
