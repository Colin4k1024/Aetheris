# 教程: 长程任务 (Long-Running Tasks)

> 使用 Aetheris 处理小时级甚至天级的 AI Agent 任务

## 概述

本教程教你如何使用 Aetheris 处理长时间运行的 AI Agent 任务，包括:
- 任务暂停和恢复
- 人类审批流程集成
- 崩溃恢复
- 进度追踪

## 目标

完成本教程后，你将:
1. 理解 Aetheris 的持久化执行模型
2. 掌握人类在环 (Human-in-the-Loop) 模式
3. 学会处理长时间运行的任务

---

## 什么是长程任务？

| 任务类型 | 执行时间 | 示例 |
|---------|---------|------|
| 短程任务 | 秒级 | 回答问题、简单计算 |
| 中程任务 | 分钟级 | 代码生成、小规模数据处理 |
| 长程任务 | 小时/天级 | 大规模数据处理、系统审计、定期报告生成 |

### 长程任务的特点

1. **执行时间长**: 可能需要数小时甚至数天
2. **中间状态**: 有多个检查点/阶段
3. **外部依赖**: 可能等待人类审批、外部 API
4. **可靠性要求**: 崩溃后需要恢复
5. **可观测性**: 需要实时追踪进度

---

## Aetheris 核心特性

### 1. 事件溯源

```
┌─────────────────────────────────────────────────────────────┐
│                    事件流 (不可变)                           │
├─────────────────────────────────────────────────────────────┤
│ JobCreated → Step1_Start → Step1_End → Step2_Start → ...  │
│                                                             │
│ 任何时刻都可以:                                            │
│   - 重放整个历史                                           │
│   - 从任意检查点恢复                                       │
│   - 审计每个步骤                                           │
└─────────────────────────────────────────────────────────────┘
```

### 2. Lease Fencing (租约围栏)

防止 Worker 崩溃后的双跑问题:

```
时间线:
Worker A (持有 Lease)                    Worker B
    |                                        |
    |--执行 step_3 (lease_token=5)---------->|
    |                                        |
    |           ⚡ 崩溃!                     |
    |                                        |
    |                         |--获取 Lease (token=6)--|
    |                         |                      |
    |                         |    检查 token!       |
    |                         |    token=5 != 6     |
    |                         |    拒绝执行         |
    |                         |<---------------------|
    |                         |                      |
    |--恢复, 检查 token=5 ---->|  (安全地继续)        |
    |                         |                      |
```

### 3. 人类在环 (Human-in-the-Loop)

```
┌─────────────────────────────────────────────────────────────┐
│                    Agent 执行流程                            │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│  ┌─────────┐    ┌─────────┐    ┌─────────┐    ┌─────────┐  │
│  │ Step 1  │───▶│ Step 2  │───▶│ ⚠️ 暂停  │───▶│ Step 3  │  │
│  │ 开始    │    │ 执行中   │    │ 等待审批 │    │ 完成    │  │
│  └─────────┘    └─────────┘    └─────────┘    └─────────┘  │
│       │                            │                       │
│       │                            │ 人类审批               │
│       │                            ▼                       │
│       │                     ┌─────────────┐                │
│       └────────────────────▶│  恢复执行   │                │
│                             └─────────────┘                │
│                                                             │
└─────────────────────────────────────────────────────────────┘
```

---

## 步骤 1: 创建项目

```bash
mkdir -p long-running-task
cd long-running-task
go mod init long-running-task
```

## 步骤 2: 编写长程任务 Agent

创建 `main.go`:

```go
package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"rag-platform/internal/agent"
	"rag-platform/internal/agent/runtime"
	"rag-platform/pkg/config"
)

func main() {
	cfg := config.Load()

	// 创建长程任务 Agent
	task := createLongRunningTask(cfg)

	ctx := context.Background()

	// 提交大规模数据处理任务
	result, err := task.Run(ctx, map[string]any{
		"task_type":      "data_migration",
		"source":         "mysql_legacy",
		"target":         "postgres_new",
		"batch_size":     1000,
		"total_records":  100000,
		"require_approval": true,  // 需要人类审批
	})

	if err != nil {
		log.Fatalf("任务执行失败: %v", err)
	}

	fmt.Printf("\n✅ 任务完成!\n\n%s\n", result)
}

// createLongRunningTask 创建长程任务 Agent
func createLongRunningTask(cfg *config.Config) *agent.Agent {
	return &agent.Agent{
		Name:        "data-migration-agent",
		Description: "大规模数据迁移 Agent",
		
		// 配置任务行为
		Options: &agent.AgentOptions{
			MaxRetries:     3,
			Timeout:        24 * time.Hour,  // 24 小时超时
			CheckpointEvery: 100,  // 每 100 条记录创建检查点
		},

		Steps: []agent.Step{
			{
				Name: "init_task",
				Description: "初始化迁移任务",
				Run: initTaskStep,
			},
			{
				Name: "connect_sources",
				Description: "连接源和目标数据库",
				Run: connectSourcesStep,
			},
			{
				Name: "migrate_batch",
				Description: "分批迁移数据 (可暂停)",
				Run: migrateBatchStep,
				// 这个步骤会被分解执行，支持暂停/恢复
			},
			{
				Name: "verify_data",
				Description: "验证数据完整性",
				Run: verifyDataStep,
			},
			{
				Name: "generate_report",
				Description: "生成迁移报告",
				Run: generateReportStep,
			},
		},
	}
}

// Step 1: 初始化任务
func initTaskStep(ctx context.Context, input any) (any, error) {
	inputMap := input.(map[string]any)
	
	taskID := fmt.Sprintf("migration_%d", time.Now().Unix())
	source := inputMap["source"].(string)
	target := inputMap["target"].(string)
	totalRecords := inputMap["total_records"].(int)

	fmt.Printf("📋 初始化数据迁移任务 #%s\n", taskID)
	fmt.Printf("   源: %s → 目标: %s\n", source, target)
	fmt.Printf("   总记录数: %d\n", totalRecords)

	// 保存任务状态
	state := map[string]any{
		"task_id":        taskID,
		"source":         source,
		"target":         target,
		"total_records":  totalRecords,
		"processed":      0,
		"failed":         0,
		"checkpoints":    []int64{},
		"start_time":     time.Now(),
	}

	return state, nil
}

// Step 2: 连接数据库
func connectSourcesStep(ctx context.Context, input any) (any, error) {
	state := input.(map[string]any)

	fmt.Println("🔌 连接数据库...")
	
	// 模拟连接
	time.Sleep(500 * time.Millisecond)
	
	source := state["source"].(string)
	target := state["target"].(string)
	
	fmt.Printf("   ✓ 已连接源数据库: %s\n", source)
	fmt.Printf("   ✓ 已连接目标数据库: %s\n", target)

	// 检查连接状态
	state["source_connected"] = true
	state["target_connected"] = true

	return state, nil
}

// Step 3: 分批迁移 (核心步骤)
func migrateBatchStep(ctx context.Context, input any) (any, error) {
	state := input.(map[string]any)

	totalRecords := state["total_records"].(int)
	batchSize := state["batch_size"].(int)
	processed := state["processed"].(int)
	
	requireApproval := state["require_approval"].(bool)

	fmt.Printf("🔄 开始迁移数据 (已处理: %d/%d)\n", processed, totalRecords)

	// 计算批次数
	totalBatches := (totalRecords + batchSize - 1) / batchSize
	currentBatch := (processed + batchSize - 1) / batchSize

	// 模拟人类审批 (如果是第一批)
	if requireApproval && processed == 0 {
		fmt.Println("⏸️ 等待人类审批...")
		
		// 模拟暂停，等待审批
		// 在实际应用中，这里会:
		// 1. 暂停任务
		// 2. 发送通知给审批人
		// 3. 等待审批
		// 4. 恢复执行
		
		// 模拟审批通过
		fmt.Println("✅ 审批通过，继续执行")
		state["approved"] = true
	}

	// 模拟分批处理
	for i := currentBatch; i < totalBatches; i++ {
		// 检查是否需要暂停 (检查点)
		if (i+1)*batchSize%10000 == 0 {
			checkpoint := int64((i + 1) * batchSize)
			checkpoints := state["checkpoints"].([]int64)
			checkpoints = append(checkpoints, checkpoint)
			state["checkpoints"] = checkpoints
			
			fmt.Printf("  ✓ 检查点: 已处理 %d 条记录\n", checkpoint)
		}

		// 模拟处理一批数据
		time.Sleep(100 * time.Millisecond)
		
		// 模拟可能的失败
		if i == 5 && false { // 演示用，实际可为 true
			state["failed"] = state["failed"].(int) + 10
		}

		state["processed"] = state["processed"].(int) + batchSize
		
		// 更新进度
		processed = state["processed"].(int)
		progress := float64(processed) / float64(totalRecords) * 100
		fmt.Printf("\r  进度: %.1f%% (%d/%d)", progress, processed, totalRecords)
	}

	fmt.Println() // 换行
	return state, nil
}

// Step 4: 验证数据
func verifyDataStep(ctx context.Context, input any) (any, error) {
	state := input.(map[string]any)

	fmt.Println("🔍 验证数据完整性...")

	processed := state["processed"].(int)
	failed := state["failed"].(int)

	// 模拟验证
	time.Sleep(500 * time.Millisecond)

	verified := processed - failed
	
	state["verified"] = verified
	state["verification_complete"] = true

	fmt.Printf("   ✓ 验证完成: %d/%d 条记录有效\n", verified, processed)

	if failed > 0 {
		fmt.Printf("   ⚠️ 警告: %d 条记录迁移失败\n", failed)
	}

	return state, nil
}

// Step 5: 生成报告
func generateReportStep(ctx context.Context, input any) (any, error) {
	state := input.(map[string]any)

	startTime := state["start_time"].(time.Time)
	endTime := time.Now()
	duration := endTime.Sub(startTime)

	totalRecords := state["total_records"].(int)
	processed := state["processed"].(int)
	failed := state["failed"].(int)
	verified := state["verified"].(int)

	report := fmt.Sprintf("═══════════════════════════════════════════════════════════\n")
	report += fmt.Sprintf("                    数据迁移报告\n")
	report += fmt.Sprintf("═══════════════════════════════════════════════════════════\n\n")
	
	report += fmt.Sprintf("任务 ID:       %s\n", state["task_id"])
	report += fmt.Sprintf("源数据库:     %s\n", state["source"])
	report += fmt.Sprintf("目标数据库:   %s\n", state["target"])
	report += fmt.Sprintf("\n")
	
	report += fmt.Sprintf("开始时间:     %s\n", startTime.Format("2006-01-02 15:04:05"))
	report += fmt.Sprintf("结束时间:     %s\n", endTime.Format("2006-01-02 15:04:05"))
	report += fmt.Sprintf("总耗时:       %s\n", duration.Round(time.Second))
	report += fmt.Sprintf("\n")
	
	report += fmt.Sprintf("─────────────────────────────────────────────────────────────\n")
	report += fmt.Sprintf("迁移统计:\n")
	report += fmt.Sprintf("─────────────────────────────────────────────────────────────\n")
	report += fmt.Sprintf("  总记录数:    %d\n", totalRecords)
	report += fmt.Sprintf("  已处理:      %d\n", processed)
	report += fmt.Sprintf("  失败:        %d\n", failed)
	report += fmt.Sprintf("  已验证:      %d\n", verified)
	report += fmt.Sprintf("\n")

	successRate := float64(processed-failed) / float64(totalRecords) * 100
	report += fmt.Sprintf("  成功率:      %.2f%%\n", successRate)
	report += fmt.Sprintf("\n")

	if len(state["checkpoints"].([]int64)) > 0 {
		report += fmt.Sprintf("检查点:\n")
		for _, cp := range state["checkpoints"].([]int64) {
			report += fmt.Sprintf("  - %d 条记录\n", cp)
		}
	}

	report += fmt.Sprintf("\n═══════════════════════════════════════════════════════════\n")

	return report, nil
}
```

## 步骤 3: 配置

创建 `config.yaml`:

```yaml
runtime:
  # 任务执行配置
  max_step_duration: 1h        # 单步最大执行时间
  checkpoint_interval: 100    # 检查点间隔
  idle_timeout: 24h          # 空闲超时
  total_timeout: 72h         # 总超时

jobstore:
  type: postgres             # 使用 PostgreSQL 确保持久化
  dsn: postgres://aetheris:aetheris@localhost:5432/aetheris

worker:
  # Worker 配置
  concurrency: 5             # 并发 Worker 数
  lease_duration: 30s        # Lease 持续时间
  renew_interval: 10s        # Lease 续约间隔
```

## 步骤 4: 运行

```bash
# 确保 PostgreSQL 运行
docker start aetheris-pg

# 应用 schema
psql "postgres://aetheris:aetheris@localhost:5432/aetheris?sslmode=disable" \
  -f ../CoRag/internal/runtime/jobstore/schema.sql

# 启动 API
go run ./cmd/api

# 启动 Worker (另一个终端)
go run ./cmd/worker

# 运行任务
go run main.go
```

预期输出:

```
📋 初始化数据迁移任务 #migration_1710691200
   源: mysql_legacy → 目标: postgres_new
   总记录数: 100000
🔌 连接数据库...
   ✓ 已连接源数据库: mysql_legacy
   ✓ 已连接目标数据库: postgres_new
⏸️ 等待人类审批...
✅ 审批通过，继续执行
🔄 开始迁移数据 (已处理: 0/100000)
  ✓ 检查点: 已处理 10000 条记录
  ✓ 检查点: 已处理 20000 条记录
  ...
  进度: 100.0% (100000/100000)
🔍 验证数据完整性...
   ✓ 验证完成: 100000/100000 条记录有效

✅ 任务完成!

═══════════════════════════════════════════════════════════
                    数据迁移报告
═══════════════════════════════════════════════════════════

任务 ID:       migration_1710691200
源数据库:     mysql_legacy
目标数据库:   postgres_new

开始时间:     2025-03-17 21:30:00
结束时间:     2025-03-17 21:35:30
总耗时:       5分30秒

─────────────────────────────────────────────────────────────
迁移统计:
  总记录数:    100000
  已处理:      100000
  失败:        0
  已验证:      100000
  成功率:      100.00%
```

---

## 崩溃恢复演示

```bash
# 1. 启动任务
go run main.go

# 2. 在执行过程中 (比如处理了 50% 时) 手动停止 Worker
# Ctrl+C

# 3. 重新启动 Worker
go run ./cmd/worker

# 4. 观察任务自动恢复
# Aetheris 会:
#   - 读取事件历史
#   - 找到最后一个检查点
#   - 从断点继续执行
```

查看恢复过程:

```bash
./bin/aetheris trace <job_id>
```

输出示例:
```
Job: migration_123456 (data-migration-agent)
Status: RUNNING (was PAUSED)
─────────────────────────────────────
Event History:
  [21:30:00] JobCreated
  [21:30:01] StepStarted: init_task
  [21:30:02] StepCompleted: init_task
  [21:30:02] StepStarted: connect_sources
  [21:30:03] StepCompleted: connect_sources
  [21:30:03] StepStarted: migrate_batch
  [21:30:04] StepCheckpoint: 10000 records
  [21:30:05] StepCheckpoint: 20000 records
  ...
  [21:32:00] ⚡ WORKER CRASH
  [21:32:01] JobPaused (reason: worker_dead)
  [21:35:00] Worker B acquired lease
  [21:35:01] JobResumed (from checkpoint: 50000)
  [21:35:02] StepResumed: migrate_batch (continuing from 50000)
  ...
```

---

## API: 人类审批集成

```bash
# 任务暂停后，可以审批恢复

# 1. 查看暂停的任务
./bin/aetheris jobs list --status=paused

# 2. 获取任务详情
./bin/aetheris jobs show <job_id>

# 3. 批准并恢复
./bin/aetheris jobs resume <job_id> --approve

# 或拒绝
./bin/aetheris jobs cancel <job_id> --reason="数据验证失败"
```

---

## 监控与可观测性

```bash
# 查看任务指标
./bin/aetheris metrics

# 查看任务日志
./bin/aetheris logs <job_id> --follow

# 查看完整事件流
./bin/aetheris trace <job_id>

# 导出证据包
./bin/aetheris export <job_id> --format=evidence
```

---

## 完整项目结构

```
long-running-task/
├── config.yaml      # 配置文件
├── main.go          # 主程序
└── go.mod          # Go 模块
```

---

## 最佳实践

1. **设置合理的检查点**: 每处理 N 条记录创建检查点
2. **实现幂等性**: 重复执行不会产生副作用
3. **记录详细日志**: 便于问题排查
4. **设置超时**: 防止任务无限期挂起
5. **实现重试机制**: 处理临时失败
6. **人类审批**: 关键步骤需要人工确认

---

## 下一步

- [事件溯源概念](../concepts/event-sourcing.md) — 深入理解 Aetheris 设计
- [CLI 参考](../reference/cli.md) — 完整 CLI 文档
- [API 参考](../reference/api.md) — 完整 API 文档
