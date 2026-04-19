# 教程: Code Review Agent

> 使用 Aetheris 构建企业级代码审查 Agent

## 概述

本教程教你如何使用 Aetheris 创建一个自动化的代码审查 Agent，具备以下能力:
- 自动分析代码质量问题
- 检查安全漏洞
- 提供修复建议
- 生成审查报告

## 目标

完成本教程后，你将:
1. 理解如何使用 Aetheris 构建 Agent
2. 掌握多步骤 DAG 的设计模式
3. 学会集成 LLM 进行代码分析

---

## 步骤 1: 创建项目结构

```bash
mkdir -p code-review-agent
cd code-review-agent
go mod init code-review-agent
```

## 步骤 2: 编写 Agent 代码

创建 `main.go`:

```go
package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/Colin4k1024/Aetheris/v2/internal/agent"
	"github.com/Colin4k1024/Aetheris/v2/internal/agent/runtime"
	"github.com/Colin4k1024/Aetheris/v2/pkg/config"
)

func main() {
	// 从环境变量或配置文件加载配置
	cfg := config.Load()

	// 创建代码审查 Agent
	reviewer := createCodeReviewAgent(cfg)

	// 提交代码审查任务
	ctx := context.Background()
	
	// 示例: 审查 Go 代码
	code := `
package main

import "fmt"

func main() {
    username := "admin"
    password := "secret123"
    
    // 敏感信息硬编码 - 安全问题!
    fmt.Println("User:", username, "Pass:", password)
}
`

	result, err := reviewer.Run(ctx, map[string]interface{}{
		"language": "go",
		"code":     code,
		"options": map[string]bool{
			"security":   true,
			"performance": true,
			"best_practices": true,
		},
	})

	if err != nil {
		log.Fatalf("代码审查失败: %v", err)
	}

	fmt.Printf("\n✅ 审查完成!\n\n%s\n", result)
}

// createCodeReviewAgent 创建代码审查 Agent
func createCodeReviewAgent(cfg *config.Config) *agent.Agent {
	return &agent.Agent{
		Name: "code-reviewer",
		Description: "自动化代码审查 Agent",
		Steps: []agent.Step{
			{
				Name: "parse_code",
				Description: "解析输入代码",
				Run: parseCodeStep,
			},
			{
				Name: "security_check",
				Description: "安全检查",
				Run: securityCheckStep,
			},
			{
				Name: "quality_check",
				Description: "代码质量检查",
				Run: qualityCheckStep,
			},
			{
				Name: "generate_report",
				Description: "生成审查报告",
				Run: generateReportStep,
			},
		},
	}
}

// Step 1: 解析代码
func parseCodeStep(ctx context.Context, input any) (any, error) {
	// 检查输入
	inputMap, ok := input.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("invalid input type")
	}

	language, _ := inputMap["language"].(string)
	code, _ := inputMap["code"].(string)

	fmt.Printf("📄 解析 %s 代码 (%d 字符)...\n", language, len(code))

	// 基础解析
	lines := 0
	for _, c := range code {
		if c == '\n' {
			lines++
		}
	}

	return map[string]any{
		"language": language,
		"code":     code,
		"lines":    lines,
		"issues":   []map[string]any{},
	}, nil
}

// Step 2: 安全检查
func securityCheckStep(ctx context.Context, input any) (any, error) {
	data := input.(map[string]any)
	code := data["code"].(string)
	issues := data["issues"].([]map[string]any)

	fmt("🔒 执行安全检查...")

	// 简单的安全检查规则
	securityPatterns := []struct {
		pattern   string
		severity  string
		message   string
	}{
		{"password", "high", "发现硬编码密码"},
		{"secret", "high", "发现硬编码密钥"},
		{"api_key", "high", "发现 API 密钥"},
		{"token", "medium", "发现可能的认证令牌"},
	}

	for _, sp := range securityPatterns {
		if containsString(code, sp.pattern) {
			issues = append(issues, map[string]any{
				"type":     "security",
				"severity": sp.severity,
				"message":  sp.message,
				"pattern":  sp.pattern,
			})
		}
	}

	data["issues"] = issues
	return data, nil
}

// Step 3: 代码质量检查
func qualityCheckStep(ctx context.Context, input any) (any, error) {
	data := input.(map[string]any)
	code := data["code"].(string)
	issues := data["issues"].([]map[string]any)

	fmt("📊 执行代码质量检查...")

	// 检查代码长度
	lines := data["lines"].(int)
	if lines > 500 {
		issues = append(issues, map[string]any{
			"type":     "maintainability",
			"severity": "low",
			"message":  fmt.Sprintf("文件过长 (%d 行), 建议拆分为多个模块", lines),
		})
	}

	// 检查是否缺少错误处理
	if !containsString(code, "err != nil") && data["language"] == "go" {
		issues = append(issues, map[string]any{
			"type":     "reliability",
			"severity": "medium",
			"message":  "缺少错误处理",
		})
	}

	data["issues"] = issues
	return data, nil
}

// Step 4: 生成报告
func generateReportStep(ctx context.Context, input any) (any, error) {
	data := input.(map[string]any)
	issues := data["issues"].([]map[string]any)

	report := fmt.Sprintf("📋 代码审查报告\n")
	report += fmt.Sprintf("================\n\n")
	report += fmt.Sprintf("代码行数: %d\n\n", data["lines"].(int))

	if len(issues) == 0 {
		report += "✅ 未发现问题!\n"
	} else {
		report += fmt.Sprintf("发现 %d 个问题:\n\n", len(issues))

		highCount := 0
		mediumCount := 0
		lowCount := 0

		for _, issue := range issues {
			severity := issue["severity"].(string)
			msg := issue["message"].(string)
			issueType := issue["type"].(string)

			emoji := "⚪"
			switch severity {
			case "high":
				emoji = "🔴"
				highCount++
			case "medium":
				emoji = "🟡"
				mediumCount++
			case "low":
				emoji = "🔵"
				lowCount++
			}

			report += fmt.Sprintf("%s [%s] %s (%s)\n", emoji, severity, msg, issueType)
		}

		report += fmt.Sprintf("\n总计: 🔴 %d 高 | 🟡 %d 中 | 🔵 %d 低\n", highCount, mediumCount, lowCount)
	}

	return report, nil
}

// 辅助函数
func containsString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
```

## 步骤 3: 配置 LLM

创建 `config.yaml`:

```yaml
llm:
  provider: qwen  # 或 openai, ollama
  model: qwen-plus
  api_key: ${DASHSCOPE_API_KEY}
  # OpenAI 示例:
  # provider: openai
  # model: gpt-4
  # api_key: ${OPENAI_API_KEY}
```

设置环境变量:

```bash
# Qwen (阿里云)
export DASHSCOPE_API_KEY=your_api_key

# 或 OpenAI
export OPENAI_API_KEY=your_api_key
```

## 步骤 4: 运行 Agent

```bash
go run main.go
```

预期输出:

```
📄 解析 go 代码 (186 字符)...
🔒 执行安全检查...
📊 执行代码质量检查...

✅ 审查完成!

📋 代码审查报告
================

代码行数: 12

发现 3 个问题:

🔴 [high] 发现硬编码密码 (security)
🔴 [high] 发现硬编码密钥 (security)
🔴 [high] 发现 API 密钥 (security)

总计: 🔴 3 高 | 🟡 0 中 | 🔵 0 低
```

---

## 进阶: 集成 LLM 分析

使用 Aetheris 的 Eino 集成进行深度 LLM 分析:

```go
import (
    "github.com/Colin4k1024/Aetheris/v2/internal/runtime/eino"
)

// 在 generateReportStep 中添加 LLM 分析
func llmAnalysis(ctx context.Context, code, language string) (string, error) {
    engine := eino.NewEngine(&eino.Config{
        Model:   "qwen-plus",
        Provider: "qwen",
    })

    prompt := fmt.Sprintf(`请审查以下 %s 代码，给出改进建议:
    
%s`, language, code)

    result, err := engine.Invoke(ctx, prompt, nil)
    if err != nil {
        return "", err
    }

    return result, nil
}
```

---

## 扩展: 添加更多检查规则

```go
// 添加更多安全检查
var securityPatterns = []struct {
    pattern  string
    severity string
    message  string
}{
    {"eval(", "high", "使用 eval() 存在代码注入风险"},
    {"exec(", "high", "使用 exec() 存在命令注入风险"},
    {"--no-sanitize", "medium", "禁用 sanitizer 可能导致安全问题"},
    {"TODO", "low", "存在未完成的代码"},
    {"FIXME", "low", "存在需要修复的代码"},
}
```

---

## 完整项目结构

```
code-review-agent/
├── config.yaml      # 配置文件
├── main.go          # 主程序
└── go.mod          # Go 模块
```

---

## 下一步

- [Audit Agent 教程](./audit-agent.md) — 构建合规审计 Agent
- [长程任务教程](./long-running-tasks.md) — 处理小时级任务
- [API 参考](../reference/api.md) — 完整 API 文档
