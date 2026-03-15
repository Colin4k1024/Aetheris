// Copyright 2026 fanjia1024
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package main 展示如何使用 Plan-Execute Agent 模式
//
// Plan-Execute Agent 是一种两阶段的智能体模式：
// 1. Plan 阶段：分析任务，制定执行计划
// 2. Execute 阶段：按照计划逐步执行任务
//
// 这种模式的优势：
// - 先思考再行动，避免盲目执行
// - 可以根据执行结果动态调整计划
// - 适合复杂的多步骤任务
//
// 运行方式：
//
//	go run ./examples/plan_execute_agent/main.go
package main

import (
	"fmt"
	"strings"
)

// ============ Plan 阶段数据结构 ============

// Step 执行步骤
type Step struct {
	ID          string `json:"id"`
	Description string `json:"description"`
	Status      string `json:"status"` // pending, completed, failed
	Result      string `json:"result,omitempty"`
}

// Plan 执行计划
type Plan struct {
	Goal    string `json:"goal"`
	Steps   []Step `json:"steps"`
	Current int    `json:"current"`
}

// ============ 主程序 ============

func main() {
	fmt.Println(strings.Repeat("=", 60))
	fmt.Println("Plan-Execute Agent Example")
	fmt.Println(strings.Repeat("=", 60))
	fmt.Println("")

	fmt.Println("Plan-Execute Agent 模式说明：")
	fmt.Println("")
	fmt.Println("┌─────────────────────────────────────────────────────────────┐")
	fmt.Println("│                     User Query                             │")
	fmt.Println("└─────────────────────────┬───────────────────────────────────┘")
	fmt.Println("                          │")
	fmt.Println("                          ▼")
	fmt.Println("┌─────────────────────────────────────────────────────────────┐")
	fmt.Println("│  ┌─────────────────────────────────────────────────────┐   │")
	fmt.Println("│  │                   PLAN 阶段                          │   │")
	fmt.Println("│  │  1. 分析任务目标                                      │   │")
	fmt.Println("│  │  2. 制定执行计划 (步骤序列)                          │   │")
	fmt.Println("│  │  3. 评估计划可行性                                   │   │")
	fmt.Println("│  └─────────────────────────────────────────────────────┘   │")
	fmt.Println("└─────────────────────────┬───────────────────────────────────┘")
	fmt.Println("                          │")
	fmt.Println("                          ▼")
	fmt.Println("┌─────────────────────────────────────────────────────────────┐")
	fmt.Println("│  ┌─────────────────────────────────────────────────────┐   │")
	fmt.Println("│  │                 EXECUTE 阶段                         │   │")
	fmt.Println("│  │  循环:                                              │   │")
	fmt.Println("│  │   1. 执行当前步骤                                    │   │")
	fmt.Println("│  │   2. 检查结果                                        │   │")
	fmt.Println("│  │   3. 如需调整，更新计划                              │   │")
	fmt.Println("│  │   4. 继续下一步                                       │   │")
	fmt.Println("│  └─────────────────────────────────────────────────────┘   │")
	fmt.Println("└─────────────────────────┬───────────────────────────────────┘")
	fmt.Println("                          │")
	fmt.Println("                          ▼")
	fmt.Println("                   Final Result")

	// ============ 1. 示例任务 ============
	tasks := []struct {
		query string
		goal  string
		steps []string
	}{
		{
			query: "帮我写一个排序算法并测试它",
			goal:  "实现并测试一个排序算法",
			steps: []string{
				"选择排序算法（快速排序）",
				"实现排序算法代码",
				"编写测试用例",
				"运行测试验证正确性",
			},
		},
		{
			query: "调研 AI 发展趋势并写一份报告",
			goal:  "完成 AI 发展趋势调研报告",
			steps: []string{
				"搜索最新 AI 研究进展",
				"分析技术趋势",
				"整理关键信息",
				"撰写报告",
			},
		},
		{
			query: "部署一个 Docker 容器到服务器",
			goal:  "完成容器部署",
			steps: []string{
				"编写 Dockerfile",
				"构建 Docker 镜像",
				"配置服务器环境",
				"部署容器",
				"验证服务运行",
			},
		},
	}

	// ============ 2. 执行任务演示 ============
	fmt.Println("")
	fmt.Println(strings.Repeat("=", 60))
	fmt.Println("任务执行演示:")
	fmt.Println(strings.Repeat("=", 60))

	for i, task := range tasks {
		fmt.Printf("\n========== 任务 %d ==========\n", i+1)
		fmt.Printf("用户请求: %s\n", task.query)

		// PLAN 阶段
		fmt.Println("\n[PLAN 阶段]")
		plan := Plan{
			Goal:  task.goal,
			Steps: make([]Step, len(task.steps)),
		}
		for j, stepDesc := range task.steps {
			plan.Steps[j] = Step{
				ID:          fmt.Sprintf("step_%d", j+1),
				Description: stepDesc,
				Status:      "pending",
			}
		}
		fmt.Printf("目标: %s\n", plan.Goal)
		fmt.Println("执行计划:")
		for j, step := range plan.Steps {
			fmt.Printf("  %d. %s\n", j+1, step.Description)
		}

		// EXECUTE 阶段
		fmt.Println("\n[EXECUTE 阶段]")
		for j := range plan.Steps {
			plan.Steps[j].Status = "completed"
			plan.Steps[j].Result = fmt.Sprintf("步骤 %d 执行完成", j+1)
			plan.Current = j
			fmt.Printf("  正在执行: %s ... ✓ 完成\n", plan.Steps[j].Description)
		}
		plan.Current = len(plan.Steps)

		fmt.Println("\n[完成]")
		fmt.Printf("最终结果: %s\n", "任务已成功完成")
	}

	// ============ 3. 动态调整演示 ============
	fmt.Println("")
	fmt.Println(strings.Repeat("=", 60))
	fmt.Println("动态调整演示:")
	fmt.Println(strings.Repeat("=", 60))
	fmt.Println("")
	fmt.Println("场景: 执行过程中发现原计划不可行，需要调整")
	fmt.Println("")
	fmt.Println("初始计划:")
	fmt.Println("  1. 使用某第三方 API")
	fmt.Println("  2. 处理返回数据")
	fmt.Println("  3. 保存结果")
	fmt.Println("")
	fmt.Println("执行过程:")
	fmt.Println("  1. 尝试使用 API... ✗ 失败 (API 不可用)")
	fmt.Println("  2. 分析原因: API 不可用")
	fmt.Println("  3. 调整计划: 改用备用方案")
	fmt.Println("     - 添加新步骤: 部署本地服务")
	fmt.Println("     - 修改后续步骤")
	fmt.Println("  4. 继续执行调整后的计划")
	fmt.Println("  5. 任务完成 ✓")
	fmt.Println("")
	fmt.Println("优势: 能够动态适应变化，而不是盲目执行原计划")

	// ============ 4. 代码示例 ============
	fmt.Println("")
	fmt.Println(strings.Repeat("=", 60))
	fmt.Println("完整 Plan-Execute Agent 创建代码:")
	fmt.Println(strings.Repeat("=", 60))
	fmt.Println("")
	fmt.Println("import (")
	fmt.Println("    \"context\"")
	fmt.Println("")
	fmt.Println("    \"github.com/cloudwego/eino/adk\"")
	fmt.Println("    \"github.com/cloudwego/eino/adk/agents/planexecute\"")
	fmt.Println("    \"github.com/cloudwego/eino/components/tool\"")
	fmt.Println(")")
	fmt.Println("")
	fmt.Println("// 1. 创建 Planner (负责制定计划)")
	fmt.Println("planner := planexecute.NewPlanner(ctx, &planexecute.PlannerConfig{")
	fmt.Println("    Name:        \"planner\",")
	fmt.Println("    Instruction: \"你是一个任务规划器。根据用户目标，制定详细的执行步骤。\",")
	fmt.Println("    Model:       chatModel,")
	fmt.Println("})")
	fmt.Println("")
	fmt.Println("// 2. 创建 Executor (负责执行计划)")
	fmt.Println("executor := planexecute.NewExecutor(ctx, &planexecute.ExecutorConfig{")
	fmt.Println("    Name:        \"executor\",")
	fmt.Println("    Instruction: \"你是一个任务执行器。按照计划步骤执行任务，返回执行结果。\",")
	fmt.Println("    Model:       chatModel,")
	fmt.Println("    Tools:       []tool.InvokableTool{/* 工具列表 */},")
	fmt.Println("})")
	fmt.Println("")
	fmt.Println("// 3. 创建 Plan-Execute Agent")
	fmt.Println("agent := planexecute.NewPlanExecuteAgent(ctx, &planexecute.PlanExecuteConfig{")
	fmt.Println("    Name:     \"plan_execute_agent\",")
	fmt.Println("    Planner:  planner,")
	fmt.Println("    Executor: executor,")
	fmt.Println("})")
	fmt.Println("")
	fmt.Println("// 4. 运行")
	fmt.Println("runner := adk.NewRunner(ctx, &adk.RunnerConfig{Agent: agent})")
	fmt.Println("result := runner.Query(ctx, \"帮我完成一个复杂任务\")")

	fmt.Println("")
	fmt.Println(strings.Repeat("=", 60))
	fmt.Println("Plan-Execute Agent 模式的优势:")
	fmt.Println(strings.Repeat("=", 60))
	fmt.Println("")
	fmt.Println("✓ 先思考再行动: 先制定计划，确保方向正确")
	fmt.Println("✓ 动态调整: 执行过程中可以根据实际情况调整计划")
	fmt.Println("✓ 可追溯: 每个步骤都有明确的状态和结果")
	fmt.Println("✓ 适合复杂任务: 多步骤任务尤其适合这种模式")
	fmt.Println("✓ 可干预: 用户可以在执行过程中审核和干预计划")
}
