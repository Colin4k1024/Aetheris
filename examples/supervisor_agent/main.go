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

// Package main 展示如何使用 Supervisor Agent 模式
//
// Supervisor Agent 是一种多智能体架构，其中一个监督智能体负责：
// 1. 理解用户请求
// 2. 将任务分解为子任务
// 3. 将子任务委托给专门的子智能体
// 4. 收集子智能体的结果
// 5. 综合结果并返回给用户
//
// 这种模式类似于公司中的管理层，Supervisor 负责协调 specialist agents
//
// 运行方式：
//
//	go run ./examples/supervisor_agent/main.go
package main

import (
	"fmt"
	"strings"
)

// ============ 子智能体定义 ============

// Agent 子智能体接口
type Agent interface {
	GetName() string
	GetDescription() string
}

// ResearchAgent 负责研究任务
type ResearchAgent struct{}

func NewResearchAgent() *ResearchAgent   { return &ResearchAgent{} }
func (a *ResearchAgent) GetName() string { return "researcher" }
func (a *ResearchAgent) GetDescription() string {
	return "Research agent - specializes in finding and analyzing information"
}

// WriterAgent 负责写作任务
type WriterAgent struct{}

func NewWriterAgent() *WriterAgent     { return &WriterAgent{} }
func (a *WriterAgent) GetName() string { return "writer" }
func (a *WriterAgent) GetDescription() string {
	return "Writer agent - specializes in creating content and summaries"
}

// CodeAgent 负责代码任务
type CodeAgent struct{}

func NewCodeAgent() *CodeAgent       { return &CodeAgent{} }
func (a *CodeAgent) GetName() string { return "coder" }
func (a *CodeAgent) GetDescription() string {
	return "Code agent - specializes in writing and reviewing code"
}

// ============ 主程序 ============

func main() {
	fmt.Println(strings.Repeat("=", 60))
	fmt.Println("Supervisor Agent Example")
	fmt.Println(strings.Repeat("=", 60))
	fmt.Println(`
Supervisor Agent 模式说明：

┌─────────────────────────────────────────────────────────────┐
│                        User Query                          │
└─────────────────────────┬───────────────────────────────────┘
                          ▼
┌─────────────────────────────────────────────────────────────┐
│                   Supervisor Agent                          │
│  ┌─────────────────────────────────────────────────────┐   │
│  │ 1. 分析请求                                          │   │
│  │ 2. 决定使用哪个子智能体                               │   │
│  │ 3. 委托任务                                          │   │
│  │ 4. 收集结果                                          │   │
│  │ 5. 综合响应                                          │   │
│  └─────────────────────────────────────────────────────┘   │
└──────────┬──────────────────┬──────────────────┬───────────┘
           │                  │                  │
    ┌──────▼──────┐    ┌──────▼──────┐    ┌──────▼──────┐
    │  Researcher │    │   Writer    │    │    Coder    │
    │   Agent     │    │   Agent     │    │   Agent     │
    └─────────────┘    └─────────────┘    └─────────────┘
`)

	// ============ 1. 创建子智能体 ============
	agents := []Agent{
		NewResearchAgent(),
		NewWriterAgent(),
		NewCodeAgent(),
	}

	fmt.Println("注册的子智能体：")
	for _, agent := range agents {
		fmt.Printf("  - %s: %s\n", agent.GetName(), agent.GetDescription())
	}

	// ============ 2. Supervisor Agent 配置 ============
	supervisorInstruction := `You are a Supervisor Agent that coordinates a team of specialized agents.

Available agents:
- researcher: Handles information gathering and research tasks
- writer: Handles content creation and summarization
- coder: Handles programming and code-related tasks

When user asks a question:
1. Analyze what type of task it is
2. Delegate to the appropriate agent(s)
3. If multiple agents are needed, delegate to each one and combine results
4. Provide a final comprehensive answer to the user`

	fmt.Println("\n" + strings.Repeat("=", 60))
	fmt.Println("Supervisor Agent 配置:")
	fmt.Println(strings.Repeat("=", 60))
	fmt.Printf("子智能体数量: %d\n", len(agents))
	fmt.Printf("Instruction: %s...\n", supervisorInstruction[:80])

	// ============ 3. 任务演示 ============
	tasks := []struct {
		query       string
		description string
	}{
		{
			"Research the latest developments in quantum computing and write a summary",
			"需要研究和写作，先委托 researcher，再委托 writer",
		},
		{
			"How do I implement a REST API in Go?",
			"代码任务，直接委托 coder",
		},
		{
			"Find information about climate change and create a presentation",
			"需要研究和写作，先委托 researcher，再委托 writer",
		},
	}

	fmt.Println("\n" + strings.Repeat("=", 60))
	fmt.Println("任务演示:")
	fmt.Println(strings.Repeat("=", 60))

	for i, task := range tasks {
		fmt.Printf("\n任务 %d: %s\n", i+1, task.query)
		fmt.Printf("分析: %s\n", task.description)
		fmt.Println("执行流程:")

		// 模拟 Supervisor 决策
		var targetAgents []string
		if strings.Contains(task.query, "research") || strings.Contains(task.query, "find") ||
			strings.Contains(task.query, "information") {
			targetAgents = append(targetAgents, "researcher")
		}
		if strings.Contains(task.query, "write") || strings.Contains(task.query, "summary") ||
			strings.Contains(task.query, "presentation") {
			targetAgents = append(targetAgents, "writer")
		}
		if strings.Contains(task.query, "implement") || strings.Contains(task.query, "code") ||
			strings.Contains(task.query, "API") || strings.Contains(task.query, "How do") {
			targetAgents = append(targetAgents, "coder")
		}

		fmt.Printf("  1. 分析任务类型...\n")
		fmt.Printf("  2. 选择子智能体: %s\n", strings.Join(targetAgents, " + "))
		fmt.Printf("  3. 依次委托任务...\n")
		for _, agent := range targetAgents {
			fmt.Printf("     → %s 执行任务\n", agent)
		}
		fmt.Printf("  4. 收集所有结果\n")
		fmt.Printf("  5. 综合响应并返回\n")
	}

	// ============ 4. 完整代码示例 ============
	fmt.Println("\n" + strings.Repeat("=", 60))
	fmt.Println("完整 Supervisor Agent 创建代码:")
	fmt.Println(strings.Repeat("=", 60))
	fmt.Println(`
import (
	"context"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/adk/agents"
	"github.com/cloudwego/eino/adk/agents/toolcall"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/components/tool/utils"
)

// 1. 创建子智能体
researcher := toolcall.NewChatModelAgent(ctx, &toolcall.ChatModelAgentConfig{
	Name:        "researcher",
	Instruction: "You are a research agent...",
	Model:       model,
})

writer := toolcall.NewChatModelAgent(ctx, &toolcall.ChatModelAgentConfig{
	Name:        "writer",
	Instruction: "You are a writer agent...",
	Model:       model,
})

coder := toolcall.NewChatModelAgent(ctx, &toolcall.ChatModelAgentConfig{
	Name:        "coder",
	Instruction: "You are a code agent...",
	Model:       model,
})

// 2. 创建委托工具
delegateTool, _ := utils.InferTool("delegate", "将任务委托给子智能体",
	func(ctx context.Context, input string) (string, error) {
		// 实现委托逻辑
		return "result", nil
	})

// 3. 创建 Supervisor Agent
supervisor := toolcall.NewSupervisor(ctx, &toolcall.SupervisorConfig{
	Name:        "supervisor",
	Instruction: "你是一个监督智能体，协调多个子智能体完成复杂任务。",
	Model:       chatModel,
	SubAgents:   []agents.Agent{researcher, writer, coder},
	Tools:       []tool.InvokableTool{delegateTool},
})

// 4. 运行
runner := adk.NewRunner(ctx, &adk.RunnerConfig{Agent: supervisor})
result := runner.Query(ctx, "your question")
`)

	fmt.Println("\n" + strings.Repeat("=", 60))
	fmt.Println("Supervisor Agent 模式的优势:")
	fmt.Println(strings.Repeat("=", 60))
	fmt.Println(`
✓ 任务分解：将复杂任务分解为可管理的子任务
✓ 专业分工：每个子智能体专注于特定领域
✓ 可扩展性：轻松添加新的子智能体
✓ 清晰架构：Supervisor 作为单一协调点
✓ 错误隔离：子智能体的错误不会影响整体
`)
}
