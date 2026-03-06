# Human-in-the-Loop：审批流与 Wait/Signal

> 本文介绍如何在 Aetheris 中实现「人工审批后再继续」的流程：Wait 节点、StatusParked、以及通过 Signal API 唤醒任务。

## 场景与需求

在法务审批合同、财务审批付款、HR 审批招聘等业务中，Agent 常需要：

- 生成文档或建议后**等待人工审批**（可能等待数小时甚至数天）
- 等待期间**系统重启或 Worker 崩溃不能丢失状态**
- 审批结果到达后**从断点继续执行**（如发送合同、触发下一步流程）

传统做法要么把状态存数据库自己轮询，要么用独立工作流引擎；Aetheris 将「等待外部事件」作为一等公民，通过 **Wait 节点** 与 **Signal** 原生支持。

## Aetheris 能力概览

| 能力               | 说明                                                              |
| ------------------ | ----------------------------------------------------------------- |
| **Wait 节点**      | 执行到该节点时 Job 进入等待状态，不占用 Scheduler 执行槽          |
| **StatusParked**   | Job 状态为「已暂停」，可安全重启 Worker，不会丢失等待中的 Job     |
| **Signal API**     | 审批完成后调用 `POST /api/jobs/:id/signal` 传入结果，唤醒对应 Job |
| **Event Sourcing** | 等待期间若崩溃，恢复后从事件流重建状态，Wait 与已完成的步骤不丢   |

## 典型流程

```
Plan → 生成合同/建议 → Wait(correlation_key="approval-123") → [人工审批 1～3 天]
                                                                    ↓
                                            Signal(approval-123, approved=true)
                                                                    ↓
                                            发送合同 / 执行后续步骤 → 完成
```

- **correlation_key**：用于关联「哪个 Wait 节点」与「哪一次 Signal」，同一 Job 内可多个 Wait 用不同 key。
- **Signal 负载**：可携带审批结果（如 `approved`、`rejected`、注释），Agent 后续步骤根据 payload 分支。

## API 示例

### 发送 Signal 唤醒 Job

审批通过后，由业务系统或人工触发调用：

```bash
# 唤醒 Job，并传入审批结果
curl -X POST http://localhost:8080/api/jobs/<job_id>/signal \
  -H "Content-Type: application/json" \
  -d '{
    "correlation_key": "approval-123",
    "payload": {"approved": true, "comment": "已阅，同意"}
  }'
```

- `job_id`：当前处于 Wait 状态的 Job ID（可从创建 Agent 消息时返回的 job_id 或业务表记录获取）。
- `correlation_key`：需与 Plan 中 Wait 节点配置的 key 一致，否则不会解除该 Wait。
- `payload`：任意 JSON，会注入到后续步骤的上下文中，供 Agent 使用（如判断 approved/rejected）。

### 查询 Job 状态

在等待期间可随时查询 Job 是否仍在等待、或已被 Signal 唤醒并完成：

```bash
curl http://localhost:8080/api/jobs/<job_id>
# status 可能为 "running" | "completed" | "failed" | "waiting" 等
```

## 实现要点

1. **Planner 与 TaskGraph**：在规划阶段需要支持「生成文档 → Wait → 根据 Signal 继续」的 DAG；Wait 节点需配置 `correlation_key`，便于与 Signal 匹配。
2. **幂等 Signal**：对同一 `correlation_key` 重复发送 Signal，Aetheris 做幂等处理，不会重复推进。
3. **长时间等待**：StatusParked 的 Job 不占 Worker 执行槽，适合审批可能数天的场景；Worker 重启或扩缩容不影响已 Parked 的 Job。

## 小结

- **Wait + Signal** 让人工审批与 Agent 执行无缝衔接，且等待期间状态持久化、可恢复。
- 通过 `POST /api/jobs/:id/signal` 传入 `correlation_key` 与 `payload` 即可唤醒对应 Wait 节点并继续执行。
- 适合合同审批、付款审批、招聘审批等 Human-in-the-Loop 场景。

## 延伸阅读

- [design/core.md](../../design/core.md) §13.1 使用场景（审批流）
- [使用 Aetheris 构建生产级 AI Agent](./02-production-agents.md)
