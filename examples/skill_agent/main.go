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

// Package main 展示如何在 CoRag 框架中使用 eino Skill 能力
//
// 这个示例展示了：
// 1. 定义 Skill 目录结构（包含 SKILL.md）
// 2. 使用 eino skill backend 进行技能管理
// 3. Skill 的渐进式加载机制（发现 -> 激活 -> 执行）
// 4. 多种上下文模式（inline, fork, isolate）
//
// Skill 是包含指令、脚本和资源的文件夹，Agent 可按需发现和使用这些 Skill 来扩展自身能力。
// Skill 的核心是 SKILL.md 文件，包含元数据（至少需要 name 和 description）和指导 Agent 执行特定任务的说明。
//
// 运行方式：
//
//	go run ./examples/skill_agent/main.go
//
// 环境变量：
//
//	DASHSCOPE_API_KEY=your-api-key  # 使用通义千问
//	OPENAI_API_KEY=your-api-key     # 或使用 OpenAI
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"rag-platform/internal/runtime/eino"
)

func main() {
	ctx := context.Background()

	// ============ 1. 创建 Skill Backend ============
	// Skill 目录路径 - 相对于示例目录
	wd, _ := os.Getwd()
	skillsDir := filepath.Join(wd, "skills")

	// 尝试多种路径
	if _, err := os.Stat(skillsDir); os.IsNotExist(err) {
		// 尝试从 examples 目录
		skillsDir = filepath.Join(wd, "..", "skill_agent", "skills")
	}
	if _, err := os.Stat(skillsDir); os.IsNotExist(err) {
		// 尝试从项目根目录
		wd, _ = os.Getwd()
		skillsDir = filepath.Join(wd, "examples", "skill_agent", "skills")
	}

	// 转换为绝对路径
	skillsDir, _ = filepath.Abs(skillsDir)

	fmt.Println("Skills 目录:", skillsDir)

	// 创建 filesystem backend
	fsBackend, err := eino.NewLocalFileBackend(ctx, &eino.LocalFileBackendConfig{
		BaseDir: skillsDir,
	})
	if err != nil {
		log.Fatalf("创建文件系统 backend 失败: %v", err)
	}

	// 创建 skill backend
	skillBackend, err := eino.NewSkillBackendFromFilesystem(ctx, fsBackend, skillsDir)
	if err != nil {
		log.Fatalf("创建 skill backend 失败: %v", err)
	}

	// ============ 2. 列出所有可用的 Skills ============
	skills, err := skillBackend.List(ctx)
	if err != nil {
		log.Fatalf("列出 skills 失败: %v", err)
	}

	fmt.Println("\n" + strings.Repeat("=", 60))
	fmt.Println("可用的 Skills:")
	fmt.Println(strings.Repeat("=", 60))
	for _, skill := range skills {
		fmt.Printf("\n📦 %s\n", skill.Name)
		fmt.Printf("   描述: %s\n", skill.Description)
		fmt.Printf("   上下文模式: %s\n", skill.Context)
		if skill.Agent != "" {
			fmt.Printf("   指定 Agent: %s\n", skill.Agent)
		}
		if skill.Model != "" {
			fmt.Printf("   指定 Model: %s\n", skill.Model)
		}
	}

	// ============ 3. 获取单个 Skill 详情 ============
	fmt.Println("\n" + strings.Repeat("=", 60))
	fmt.Println("获取 pdf_analyzer Skill 详情:")
	fmt.Println(strings.Repeat("=", 60))

	pdfSkill, err := skillBackend.Get(ctx, "pdf_analyzer")
	if err != nil {
		log.Fatalf("获取 pdf_analyzer skill 失败: %v", err)
	}
	fmt.Printf("\n名称: %s\n", pdfSkill.Name)
	fmt.Printf("描述: %s\n", pdfSkill.Description)
	fmt.Printf("上下文模式: %s\n", pdfSkill.Context)
	fmt.Printf("基础目录: %s\n", pdfSkill.BaseDirectory)
	fmt.Printf("\n技能内容:\n%s\n", pdfSkill.Content)

	// ============ 4. 获取 log_analyzer Skill ============
	fmt.Println("\n" + strings.Repeat("=", 60))
	fmt.Println("获取 log_analyzer Skill 详情:")
	fmt.Println(strings.Repeat("=", 60))

	logSkill, err := skillBackend.Get(ctx, "log_analyzer")
	if err != nil {
		log.Fatalf("获取 log_analyzer skill 失败: %v", err)
	}
	fmt.Printf("\n名称: %s\n", logSkill.Name)
	fmt.Printf("描述: %s\n", logSkill.Description)
	fmt.Printf("上下文模式: %s\n", logSkill.Context)
	fmt.Printf("\n技能内容:\n%s\n", logSkill.Content)

	// ============ 5. Skill Agent 使用说明 ============
	fmt.Println("\n" + strings.Repeat("=", 60))
	fmt.Println("Skill Agent 使用指南:")
	fmt.Println(strings.Repeat("=", 60))
	fmt.Print(`
Skill 渐进式加载机制：

1. 发现阶段 (Discovery):
   - Agent 启动时，仅加载每个可用 Skill 的名称和描述
   - 这足以让 Agent 判断何时可能需要使用某个 Skill

2. 激活阶段 (Activation):
   - 当任务匹配某个 Skill 的描述时
   - Agent 将完整的 SKILL.md 内容读入上下文

3. 执行阶段 (Execution):
   - Agent 遵循 Skill 中的指令执行任务
   - 也可以根据需要加载其他文件或执行捆绑的代码

三种上下文模式：

┌─────────────┬────────────────────────────────────────────────────┐
│ 模式        │ 说明                                                │
├─────────────┼────────────────────────────────────────────────────┤
│ inline      │ (默认) Skill 内容直接作为工具结果返回，由当前       │
│ (默认)      │ Agent 继续处理                                       │
├─────────────┼────────────────────────────────────────────────────┤
│ fork        │ 创建新 Agent，复制当前对话历史，独立执行 Skill      │
│             │ 任务后返回结果                                        │
├─────────────┼────────────────────────────────────────────────────┤
│ isolate     │ 创建新 Agent，使用隔离的上下文（仅包含 Skill 内容），│
│             │ 独立执行后返回结果                                    │
└─────────────┴────────────────────────────────────────────────────┘

创建带 Skill 的 Agent (需要配合 eino agent 使用):

	import (
		"github.com/cloudwego/eino/adk/middlewares/skill"
		"github.com/cloudwego/eino/adk"
	)

	// 使用上面创建的 skillBackend
	skillMiddleware, err := skill.NewMiddleware(ctx, &skill.Config{
		Backend:        skillBackend,
		SkillToolName:  ptr.String("skill"),
	})

	agent, err := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:        "DocumentAssistant",
		Instruction: "You are a helpful assistant.",
		Model:       chatModel,
		Handlers:    []adk.ChatModelAgentMiddleware{skillMiddleware},
	})

Skill 目录结构示例：

	skills/
	├── pdf_analyzer/
	│   ├── SKILL.md          # 必需：技能定义
	│   ├── scripts/          # 可选：可执行脚本
	│   │   └── analyze.py
	│   ├── references/      # 可选：参考文档
	│   └── assets/          # 可选：资源文件
	└── log_analyzer/
	    ├── SKILL.md
	    └── scripts/
	        └── parse.py

SKILL.md 格式：

	---
	name: skill_name
	description: 技能描述
	context: fork  # 或 inline, isolate
	agent: agent_name  # 可选
	model: model_name  # 可选
	---
	# 技能指令

	你的技能说明和执行步骤...
`)
}
