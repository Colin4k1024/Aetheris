---
artifact: arch-design
task: architecture-analysis
date: 2026-04-05
role: architect
status: ready-for-execute
---

# 架构设计：安全与 Runtime 修复

## 1. 系统边界

### 1.1 修改范围

**受影响组件：**
- `internal/api/http/middleware/middleware.go` — 认证、租户注入
- `internal/api/http/handler.go` — API 端点租户过滤
- `internal/runtime/jobstore/pgstore.go` — Heartbeat 校验
- `internal/agent/runtime/node_adapter.go` — EffectStore 写入顺序
- `internal/agent/runtime/agent.go` — Agent 并发保护
- `internal/agent/runtime/manager.go` — Manager 并发保护
- `internal/app/api/app.go` — 凭证检查、RoleStore 配置
- `internal/storage/metadata/` — Document TenantID

### 1.2 边界内外

**边界内（本次修改）：**
- API 认证层
- API 租户隔离
- Runtime 并发安全
- EffectStore 写入顺序

**边界外（不在本次范围）：**
- JobStore 核心逻辑
- Planner/Executor 核心逻辑
- Pipeline 组件
- Frontend/Dashboard

---

## 2. 组件拆分

### 2.1 认证组件

```
Auth Layer (middleware)
├── JWTValidator     — JWT 解析和验证
├── TenantExtractor   — 从 JWT/X-Header 提取 tenant_id
├── RBACChecker      — 权限校验（查 RoleStore）
├── AuditLogger      — 审计日志记录
└── CredentialValidator — 凭证校验（环境变量）
```

**修改点：**
- `CredentialValidator`: 移除硬编码，改为 `os.Getenv("ADMIN_PASSWORD")` 或返回错误
- `AuditLogger`: 增加 `ClientIP`, `UserAgent`, `RequestID` 字段

### 2.2 Runtime 并发组件

```
Runtime Core
├── Agent           — Agent 执行上下文（需并发保护）
├── Manager         — Agent 生命周期管理（需并发保护）
├── Scheduler       — Job 调度（需 attempt_id 校验）
├── Runner          — Step 执行（需 Checkpoint 原子更新）
└── EffectStore     — 副作用记录（需写入顺序修复）
```

**修改点：**
- `Agent`: 增加 `sync.Mutex` 保护 Session/Memory/Planner/Tools
- `Manager.Get()`: 返回副本或引入 Take/Release 语义
- `Scheduler.Heartbeat()`: SQL 增加 `attempt_id` 校验
- `EffectStore.Write()`: 改为事件流优先

---

## 3. 关键数据流

### 3.1 认证数据流

```
Request
  → JWTValidator (验证签名和过期)
  → TenantExtractor (提取 tenant_id)
  → RBACChecker (查 RoleStore 校验权限)
  → AuditLogger (记录操作)
  → Handler (执行业务逻辑)
```

### 3.2 EffectStore 写入顺序（修复后）

```
1. AppendToolInvocationStarted (声明开始)
2. Execute Tool
3. AppendCommandCommitted (提交结果到事件流) ← 权威来源
4. PutEffect (可选，仅用于审计加速) ← 异步
```

**设计原则**: 事件流是权威来源，EffectStore 是可选加速层。崩溃后从事件流恢复。

### 3.3 Heartbeat 校验流程（修复后）

```
Worker.Heartbeat(jobID, workerID, attemptID)
  → UPDATE job_claims
    SET expires_at = $1
    WHERE job_id = $2
      AND worker_id = $3
      AND attempt_id = $4
      AND RowsAffected == 1
  → if RowsAffected == 0:
      return ErrStaleLease
```

---

## 4. 接口约定

### 4.1 认证接口

| 接口 | 说明 |
|------|------|
| `GET /api/health` | 无需认证 |
| 其他所有 `/api/*` | 需要有效 JWT |
| `POST /api/auth/login` | 接受环境变量凭证，返回 JWT |

### 4.2 租户隔离接口

| 接口 | 租户过滤 |
|------|----------|
| `GET /api/agents` | 按 tenant_id 过滤 |
| `GET /api/documents` | 按 tenant_id 过滤 |
| `GET /api/jobs` | 按 tenant_id 过滤 |

---

## 5. 技术选型

| 决策 | 选型 | 原因 |
|------|------|------|
| 凭证存储 | 环境变量 | 避免硬编码，支持 Docker/K8s Secret |
| RoleStore | PostgreSQL (解耦) | 独立配置，多实例共享 |
| 并发保护 | sync.Mutex | 简单有效，避免死锁 |
| EffectStore 顺序 | 事件流优先 | 与 design doc 一致，更安全 |

---

## 6. 风险与约束

| 风险 | 影响 | 缓解 |
|------|------|------|
| Schema 迁移 | Document 增加 TenantID 字段 | 提供 nullable 字段，逐步迁移 |
| 向后兼容 | EffectStore 顺序修改 | 测试覆盖确保 backward compatible |
| 并发锁粒度 | 粗粒度锁影响性能 | 先用粗粒度，后续按需优化 |

---

## 7. 未决项

1. **Agent 并发模型**: Agent 是否允许多 goroutine 并发调用 `Run()`？
2. **Worker 失租检测**: Worker 检测到自己失租后如何反应？
3. **已有数据迁移**: 现有 Document/Agent 的 TenantID 如何回填？
