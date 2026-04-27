# Aetheris Current Status & Focus (2026-02)

## 战略定位

**Aetheris = Temporal for Agents**

平衡执行可靠性和审计能力，聚焦可生产运营的分布式 Agent Runtime。
当前 Q1–Q4 的工程收敛与运营化推进，建立在四阶段战略重构（本地事件存储、持久化执行 API、MCP Host、Open Core）之上，见 [../strategy-and-user-stories.md](../strategy-and-user-stories.md)。

---

## 当前实际状态（诚实评估）

### ✅ 已稳定实现（Production Ready）

**Runtime Semantic 1.0+**:

- At-most-once tool execution
- Confirmation replay
- Tool invocation ledger
- Event sourcing + JobStore
- Distributed worker + Scheduler correctness
- Lease fencing + Heartbeat
- Crash recovery

**基础可观测**:

- Trace UI（timeline + execution tree）
- Event stream API
- Reasoning snapshot
- CLI trace/debug/replay

**部署基础**:

- Docker Compose stack
- PostgreSQL schema
- CLI 工具

### ⚠️ 已接入，需要验收/配置化（1-2 周可完成）

**Operational Runtime 核心**:

- Event snapshot/compaction（Worker 已有自动化 loop；需要配置化、保留策略验收）
- Rate limiting（LLM 与 Tool 执行路径已接入；需要配置样例与压测验收）
- Tenant isolation（JobStore/API 已有 tenant 过滤；需要幂等查询与导出路径加固）
- Storage lifecycle/GC（Worker 已有定时任务；需要配置化与生命周期 store 验收）

**基础审计**:

- Evidence zip export（HTTP/CLI 已接入 proof 包；需要真实 handler/store E2E 验收）
- Hash chain（已在 Append 中计算，CLI/HTTP verify 已接入；需要篡改失败用例）
- RBAC基础（权限模型与 middleware 已接入；需要关键路径权限矩阵验收）

### 🔬 已有设计原型，3.0再完善（非当前 focus）

**高级审计**:

- Evidence Graph（types定义完成，builder需完善）
- Forensics Query API（接口定义完成，引擎需实现）
- 脱敏引擎（基础实现完成，策略配置需完善）
- Retention engine（框架完成，真实归档需实现）

**3.0 特性（设计框架已有，实际实现暂缓）**:

- 数字签名（Ed25519 keystore完成，集成暂缓）
- 分布式 Ledger（协议定义完成，同步实现暂缓）
- AI 异常检测（接口完成，模型集成暂缓）
- 质量评分（scorer完成，实时计算暂缓）
- 合规模板（模板定义完成，自动应用暂缓）

---

## Q1 2026 Focus（当前季度，极高优先级）

### 目标：Operational Runtime 就绪

让 Aetheris 真正可以在生产环境稳定运行。

### P0 必做（2-4 周）

1. **Rate Limiter 实际应用**
   - 已接入 ToolNodeAdapter 与 LLM client wrapper
   - 补齐配置验证、指标检查和并发压测

2. **Tenant Isolation 完整**
   - Job 创建已绑定 tenant_id
   - 幂等查询下沉到 store 层 tenant-aware 语义
   - 补齐跨 tenant 访问、导出、事件读取测试

3. **Snapshot 自动化**
   - Worker 定时任务已创建 snapshot
   - 阈值、间隔、批量、保留数量需配置化
   - ReplayContextBuilder 使用 snapshot 的 E2E 需要验收

4. **Storage GC 完整**
   - 定时扫描已接入 Worker
   - Tool invocations 归档/清理
   - TTL、批量、启停需配置化

5. **Evidence Export 集成**
   - JobStore 适配已接入
   - HTTP/CLI 导出与离线 verify 保持兼容
   - 补真实 handler/store End-to-end 测试

6. **Metrics 生产配置**
   - Prometheus `/metrics` 与 Jaeger 配置已接入
   - Compose/Grafana dashboard 需运行验收
   - 告警规则调优

### P1 应做（4-8 周）

7. **Basic RBAC**（简化版）
   - Admin/User 两角色
   - 关键 API 权限控制
   - JWT middleware

8. **Job Quota**
   - Tenant 级别限制
   - 超额拒绝
   - Quota API

9. **Graceful Shutdown**
   - Worker 优雅退出
   - Job 完成后停止
   - Lease 释放

10. **Documentation**
    - Deployment guide
    - Operations manual
    - Troubleshooting guide

---

## Q2 2026 Focus：分布式执行成熟

### 目标：多 Worker 场景的稳定性和性能

1. Job sharding（按 agent_id/tenant_id）
2. Worker capability routing 完善
3. Lease recovery 优化
4. Backpressure 机制
5. OpenTelemetry 完整集成
6. Performance benchmarks

---

## Q3-Q4 2026 Focus：Evidence 能力产品化

### 目标：基础审计能力完整可用

1. Evidence zip 产品化
2. Offline verify 工具完善
3. Basic forensics query
4. Evidence Graph API 完善
5. Audit log 查询
6. Retention policy 完整

---

## 不做清单（避免分散）

❌ **暂不做**：

- 完整的脱敏策略配置（只保留基础 redact）
- 复杂的 Forensics Query（只做基础过滤）
- Evidence Graph UI 渲染（保留 API）
- 数字签名（3.0）
- 分布式 Ledger（3.0）
- AI 辅助取证（3.0）
- 实时质量评分的复杂计算（3.0）
- 合规模板的自动应用（3.0）

---

## 代码组织原则

### 当前可用（src/）

- pkg/proof/ - 基础导出/验证
- pkg/auth/ - 基础 RBAC
- internal/runtime/jobstore/ - Snapshot / GC 接口与实现
- internal/agent/runtime/executor/ - Tool Rate limiter（已接执行路径）
- internal/model/llm/ - LLM Rate limiter（已接 client wrapper）

### 原型/未来（prototypes/ 或标记 TODO）

- pkg/evidence/ - Evidence Graph builder
- pkg/forensics/ - 复杂查询引擎
- pkg/redaction/ - 完整脱敏策略
- pkg/signature/ - 数字签名
- pkg/distributed/ - 分布式同步
- pkg/ai_forensics/ - AI 检测
- pkg/monitoring/ - 质量评分
- pkg/compliance/ - 合规模板

---

## 成功标准（Q1 结束时）

### 功能完整性

- [ ] 100个并发 jobs 稳定运行
- [ ] 单 job 10000+ 事件不爆炸（snapshot）
- [ ] LLM API 不被打爆（rate limit）
- [ ] 10+ tenants 数据隔离
- [ ] 任意 job 可导出+验证 evidence zip

### 质量保证

- [ ] 核心功能测试覆盖 > 80%
- [ ] 生产环境运行 1 周无崩溃
- [ ] Metrics 和 Alert 覆盖关键路径
- [ ] 文档覆盖部署和运维

### 性能目标

- [ ] Job 吞吐: 10-20 jobs/min per worker
- [ ] P95 延迟: < 5s（简单 job）
- [ ] Snapshot 创建: < 1s（1000 events）
- [ ] Evidence export: < 5s（包含 ledger）

---

## 技术债务（已识别）

1. **集成缺失**: 很多功能有接口无集成
2. **配置分散**: 需要统一配置管理
3. **测试不足**: 集成测试需要补充
4. **文档滞后**: 实际使用文档需要更新

---

## 下一步行动

### 本周

1. 补充核心集成代码
2. 简化过度设计
3. 测试关键路径
4. 更新部署文档

### 本月

1. Q1 P0 任务全部完成
2. 生产环境试运行
3. 性能基准测试
4. 用户文档完善

---

## 诚实的话

**我们现在的问题不是缺功能，而是：**

- 功能太多没集成
- 设计太超前
- 测试覆盖不足

**真正需要的是：**

- 收敛聚焦
- 补充集成
- 验证稳定性

---

**当前 Focus**: Operational Runtime（Q1）  
**下一步**: 分布式执行（Q2）  
**远期**: Evidence 产品化（Q3-Q4）

不急于做完所有功能，先把核心做稳。
