---
artifact: execute-log
task: architecture-analysis
date: 2026-04-05
role: backend-engineer
status: in-progress
---

# Execute Log：架构分析与修复

## 计划 vs 实际

### 计划 (Phase 1 - 安全修复)

| ID | 工作项 | 计划状态 |
|----|--------|----------|
| SEC-01 | 移除硬编码凭证 | ✅ 完成 |
| SEC-02 | 修复 ListAgents/ListDocuments 租户过滤 | ✅ 完成 |
| SEC-03 | Document/Agent 数据模型增加 TenantID | ✅ 完成 |
| SEC-04 | RoleStore 配置与 JobStore 解耦 | ✅ 完成（部分） |
| SEC-05 | auth disabled 降级改为 RoleUser | ✅ 完成 |
| SEC-06 | 增加启动时凭证存在性检查 | ✅ 完成 |

### 计划 (Phase 2 - Runtime 正确性修复)

| ID | 工作项 | 计划状态 |
|----|--------|----------|
| RTN-01 | Heartbeat 添加 attempt_id 校验 | ✅ 完成 |
| RTN-02 | 明确 Worker 失租检测机制 | ✅ 完成 |
| RTN-03 | EffectStore 写入顺序修复 | ✅ 完成 |
| RTN-04 | EffectStore catch-up 逻辑验证 | ✅ 完成 |
| RTN-05 | Agent 并发模型决策 | ✅ 完成 |
| RTN-06 | Agent/Manager 并发保护 | ✅ 完成 |
| RTN-08 | Checkpoint/Cursor 原子更新验证 | ✅ 完成 |

---

## 实际完成

### SEC-01: 移除硬编码凭证
- **文件**: `internal/api/http/middleware/middleware.go`
- **变更**: 移除 `admin/admin` 和 `test/test` 硬编码凭证
- **改为**: 从环境变量 `ADMIN_USERNAME` 和 `ADMIN_PASSWORD` 读取
- **结果**: 若环境变量未设置，登录失败并记录 WARN 日志

### SEC-02: ListAgents/ListDocuments 租户过滤
- **文件**: `internal/api/http/handler.go`, `internal/api/grpc/service.go`
- **变更**: 添加 tenant_id 过滤
- `ListDocuments`: 从 context 获取 tenantID 并过滤
- `ListAgents`: 遍历时过滤 session.TenantID 不匹配的 agent

### SEC-03: Document/Agent TenantID
- **文件**: `internal/storage/metadata/interface.go`, `internal/storage/metadata/memory.go`, `internal/app/document.go`
- **变更**:
  - `Document` 结构体增加 `TenantID` 字段
  - `Filter` 结构体增加 `TenantID` 过滤条件
  - `MemoryStore.List()` 增加 tenant_id 过滤逻辑
  - `DocumentService.ListDocuments()` 接口增加 tenantID 参数

### SEC-05: auth disabled 降级改为 RoleUser
- **文件**: `internal/app/api/app.go`
- **变更**: 移除预置 `admin`/`test` 用户角色，`anonymous` 降级为 `RoleUser`

### SEC-06: 增加启动时凭证检查
- **文件**: `internal/app/api/app.go`
- **变更**: JWT 认证启用但环境变量未配置时记录明确警告

### RTN-01: Heartbeat attempt_id 校验
- **文件**: `internal/runtime/jobstore/pgstore.go`, `internal/runtime/jobstore/store.go`
- **变更**:
  - 新增 `ErrStaleLease` 错误定义
  - `Heartbeat` 方法支持 `WithAttemptID` context
  - 新增 `TestPgStore_Heartbeat_StaleLease` 测试用例验证 stale lease 场景

### RTN-03: EffectStore 写入顺序修复
- **文件**: `internal/agent/runtime/executor/node_adapter.go`
- **变更**: 调整写入顺序
  - **之前**: Phase 1 写 EffectStore → Phase 2 写事件流
  - **之后**: 先写事件流 `AppendToolInvocationFinished` → 再写 EffectStore（可选）
- **原因**: 设计文档明确事件流是权威来源

### RTN-05: Agent 并发模型决策（Take/Release 语义）
- **文件**: `internal/agent/runtime/agent.go`, `internal/agent/runtime/scheduler.go`
- **变更**: 实现 Agent Take/Release 原子操作
  - **新增方法**: `Agent.Take()`, `Agent.Release()`, `Agent.TakeWithWait()`
  - `Take()`: 原子性检查 Idle/Suspended 并转为 Running，若已被占用返回 false
  - `Release()`: 将 Running/WaitingTool 转回 Idle
  - **Scheduler.WakeAgent**: 使用 Take() 替代分离的 GetStatus()+SetStatus()，修复 TOCTOU race
  - **Scheduler.Resume**: 使用 Release()+Take() 替代两连续的 SetStatus()
- **原因**: 解决 WakeAgent 中 GetStatus() 和 SetStatus() 非原子导致的并发竞争
- **测试**: 新增 `TestAgent_TakeRelease`, `TestAgent_TakeFromSuspended`, `TestAgent_ReleaseFromWaiting`, `TestScheduler_WakeAgent_TakeRelease`

### RTN-06: Agent/Manager 字段并发保护
- **文件**: 继承自 RTN-05
- **变更**: RTN-05 的 Take/Release 实现已覆盖 Agent Status 字段的并发保护
- **Manager**: map 访问已有互斥锁保护
- **Agent Session**: 自带 mutex 保护

### RTN-02: Worker 失租检测机制（已实现）
- **机制**: 已实现 Heartbeat 定期续租 + Lease 过期回收
- **配置**: `leaseDur = 30s`, `heartbeatTicker = leaseDur/2 = 15s`
- **流程**: Worker 执行 Job 时启动 Heartbeat goroutine，每 15 秒调用 `jobEventStore.Heartbeat`；若 Worker 崩溃，30 秒后 Lease 过期
- **回收**: `ReclaimOrphanedFromEventStore` 调用 `ListJobIDsWithExpiredClaim` 获取过期 Job，过滤 Blocked 状态后转回 Pending
- **设计依据**: `design/internal/scheduler-correctness.md` §1 "Worker epoch / stale worker kill"
- **结论**: 机制已正确实现，无需额外修改

### RTN-04: EffectStore catch-up 逻辑验证
- **文件**: `internal/agent/runtime/executor/node_adapter.go`
- **验证**: catch-up 逻辑正确实现
  - 若事件流无 `command_committed` 但 EffectStore 有 effect → `writeCatchUpFinished` 写回事件流
  - `writeCatchUpFinished` 正确调用 `AppendToolInvocationFinished` + `AppendCommandCommitted`
  - `Activity Log Barrier` 设计：若已 started 无 finished 时禁止再次执行
- **测试**: `TestLedger_1~5` 通过，覆盖 crash before/after commit, double worker, replay recovery 等场景
- **结论**: catch-up 逻辑已正确实现，无需修改

### RTN-08: Checkpoint/Cursor 原子更新验证
- **文件**: `internal/agent/runtime/checkpoint_pg.go`, `internal/agent/job/pg_store.go`
- **验证**: Checkpoint Save 和 Cursor Update 均为单条 SQL 操作（原子）
  - `CheckpointStorePg.Save`: INSERT with ON CONFLICT DO UPDATE（原子 upsert）
  - `JobStorePg.UpdateCursor`: 单条 UPDATE 语句（原子）
- **设计依据**: `design/internal/scheduler-correctness.md` §Cursor 更新
  - Cursor 更新仅由持有租约的 Worker 调用，依赖 Worker 合约
  - 失去租约的 Worker 必须停止，不再调用 UpdateCursor
- **残余风险**: Checkpoint Save 和 Cursor Update 是两个独立操作，崩溃可能导致不一致；但 completedSet 来自事件流可防止重复执行
- **结论**: 设计正确，无需修改

### RTN-09: ListJobIDsWithExpiredClaim 过滤终态
- **文件**: `internal/runtime/jobstore/pgstore.go`
- **问题**: `ListJobIDsWithExpiredClaim` 返回所有过期租约的 job_id，包括 Completed(2)/Failed(3) 终态
- **修复**: SQL JOIN jobs 表，过滤 `status NOT IN (2, 3)`
- **变更**: `SELECT job_id FROM job_claims WHERE expires_at <= now()` → `SELECT c.job_id FROM job_claims c JOIN jobs j ON c.job_id = j.id WHERE c.expires_at <= now() AND j.status NOT IN (2, 3)`
- **原因**: 终态 job 不应被回收，回收逻辑依赖 `ReclaimOrphanedFromEventStore` 在 Go 层过滤，但 SQL 层应先过滤以减少不必要的 I/O

### RTN-07: completedSet 同步问题分析
- **文件**: `internal/agent/runtime/executor/runner.go`
- **分析结论**: **实现正确，无需修复**
- **原因**:
  - `runParallelLevel` 中 goroutine 只执行步骤并通过 channel 发送结果，**不直接访问 completedSet**
  - `completedSet` 的写操作发生在主 goroutine 中，在 channel barrier（`for range batch { res := <-ch }`）之后顺序执行
  - `runLoop` 是单线程 `for` 循环，不存在并发访问
- **结论**: 使用 Go 的 channel 消息传递正确同步，无数据竞争

### TST-04: sleep 同步问题
- **文件**: `internal/agent/runtime/executor/rate_limiter_integration_test.go:93`
- **问题**: `time.Sleep(10 * time.Millisecond)` 保持并发槽，依赖固定时间假设
- **建议**: 使用 channel-based start gate 替换 sleep 同步
- **备注**: `node_adapter.go:656` 的生产代码 backoff 是合理的重试模式，无需修改

---

## 关键决定

### 决定 1: 凭证存储方式
- **选择**: 环境变量
- **原因**: 符合 12-factor app 最佳实践，支持 Docker/K8s Secret

### 决定 2: EffectStore 写入顺序
- **选择**: 事件流优先
- **原因**: 与 design doc 一致，事件流是权威来源

### 决定 3: 降级角色
- **选择**: `RoleUser` 而非 `RoleAdmin`
- **原因**: 最小权限原则

### 决定 4: Agent 并发模型（RTN-05）
- **选择**: Take/Release 语义
- **原因**: 类似 Ledger 的 Acquire 模型，提供原子性的状态转换，避免 TOCTOU race
- **效果**: WakeAgent 先 Take() 再执行 fn，fn 完成后通过 defer Release()

---

## 影响面

| 文件 | 变更 |
|------|------|
| `internal/api/http/middleware/middleware.go` | 凭证验证改为环境变量 |
| `internal/app/api/app.go` | 移除预置角色，添加凭证检查 |
| `internal/api/http/handler.go` | ListDocuments/ListAgents 添加租户过滤 |
| `internal/api/grpc/service.go` | ListDocuments 添加租户过滤 |
| `internal/storage/metadata/interface.go` | Document/Filter 增加 TenantID |
| `internal/storage/metadata/memory.go` | List/Count 增加 tenant 过滤 |
| `internal/runtime/jobstore/pgstore.go` | Heartbeat 增加 attempt_id 校验 |
| `internal/runtime/jobstore/store.go` | ErrStaleLease 定义 |
| `internal/agent/runtime/executor/node_adapter.go` | EffectStore 写入顺序调整 |
| `internal/agent/runtime/agent.go` | Agent 增加 TenantID 字段，Take/Release 方法 |
| `internal/agent/runtime/scheduler.go` | WakeAgent/Resume 使用 Take/Release 原子操作 |
| `internal/agent/runtime/manager_test.go` | 新增 Take/Release 并发测试 |
| `internal/runtime/jobstore/pgstore.go` | RTN-09: ListJobIDsWithExpiredClaim JOIN jobs 过滤终态 |

---

## 待执行项

| ID | 工作项 | 优先级 | 说明 |
|----|--------|--------|------|
| RTN-07 | completedSet 同步问题修复 | P1 | |
| SEC-07 | JWT payload 增加 roles | P2 | |
| SEC-08 | 审计日志增加字段 | P2 | |
| TST-01~06 | 测试增强 | P1~P2 | QA/后端并行 |

## 已完成总计

| 阶段 | 完成项 |
|------|--------|
| Phase 1 安全修复 | SEC-01~03, SEC-05~06 |
| Phase 2 Runtime | RTN-01~06, RTN-08, RTN-09 |
| 新增测试 | Take/Release 并发测试 |

---

## 自测结果

- ✅ `go build ./...` 通过
- ✅ `make test` 通过
- ✅ jobstore tests 通过
- ⏳ 需设置环境变量后测试登录

---

## 下一步行动

1. **RTN-05/06**: 需要架构师决策 Agent 并发模型后实施
2. **RTN-02**: 需要确认 Worker 失租检测机制
3. **测试增强**: 可以并行进行
