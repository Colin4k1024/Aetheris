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

// Package main 展示如何在 CoRag 框架中托管 eino ReAct Agent
//
// 这个示例展示了：
// 1. 使用 Ollama 本地 LLM
// 2. 定义和使用工具
// 3. ReAct Agent 的完整执行流程
// 4. 与 CoRag 框架的集成
//
// 运行方式：
//
//	go run ./examples/eino_agent_with_tools/main.go
//
// 环境变量：
//
//	OLLAMA_MODEL=llama3
//	OLLAMA_BASE_URL=http://localhost:11434
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/components/tool/utils"

	"github.com/Colin4k1024/Aetheris/v2/internal/agent/runtime/executor"
	eino_examples "github.com/Colin4k1024/Aetheris/v2/internal/agent/runtime/executor/eino_examples"
	"github.com/Colin4k1024/Aetheris/v2/internal/model/llm"
)

// ToolInput 工具输入参数
type ToolInput struct {
	Operation string `json:"operation"`
	Value1    int    `json:"value1"`
	Value2    int    `json:"value2"`
}

// CalculatorTool 执行数学计算
func CalculatorTool(ctx context.Context, input string) (string, error) {
	var ti ToolInput
	if err := json.Unmarshal([]byte(input), &ti); err != nil {
		return "", fmt.Errorf("invalid input: %w", err)
	}

	var result int
	switch strings.ToLower(ti.Operation) {
	case "add", "加", "+":
		result = ti.Value1 + ti.Value2
	case "subtract", "减", "-":
		result = ti.Value1 - ti.Value2
	case "multiply", "乘", "*":
		result = ti.Value1 * ti.Value2
	case "divide", "除", "/":
		if ti.Value2 == 0 {
			return "", fmt.Errorf("division by zero")
		}
		result = ti.Value1 / ti.Value2
	default:
		return "", fmt.Errorf("unknown operation: %s", ti.Operation)
	}

	return fmt.Sprintf("%d", result), nil
}

// SearchInput 搜索输入
type SearchInput struct {
	Query string `json:"query"`
	Limit int    `json:"limit"`
}

// FakeSearchTool 模拟搜索工具
func FakeSearchTool(ctx context.Context, input string) (string, error) {
	var si SearchInput
	if err := json.Unmarshal([]byte(input), &si); err != nil {
		return "", fmt.Errorf("invalid input: %w", err)
	}

	// 模拟搜索结果
	results := []string{
		fmt.Sprintf("Result 1 for: %s", si.Query),
		fmt.Sprintf("Result 2 for: %s", si.Query),
		fmt.Sprintf("Result 3 for: %s", si.Query),
	}

	if si.Limit > 0 && si.Limit < len(results) {
		results = results[:si.Limit]
	}

	return strings.Join(results, "\n"), nil
}

// WeatherInput 天气查询输入
type WeatherInput struct {
	City string `json:"city"`
}

// FakeWeatherTool 模拟天气查询工具
func FakeWeatherTool(ctx context.Context, input string) (string, error) {
	var wi WeatherInput
	if err := json.Unmarshal([]byte(input), &wi); err != nil {
		return "", fmt.Errorf("invalid input: %w", err)
	}

	// 模拟天气数据
	weather := map[string]string{
		"beijing":   "晴, 25°C, 空气质量良",
		"shanghai":  "多云, 28°C, 空气质量良",
		"guangzhou": "雷阵雨, 32°C, 空气质量中",
		"shenzhen":  "晴, 31°C, 空气质量良",
		"hangzhou":  "晴, 26°C, 空气质量优",
	}

	if w, ok := weather[strings.ToLower(wi.City)]; ok {
		return fmt.Sprintf("%s: %s", wi.City, w), nil
	}

	return fmt.Sprintf("%s: 天气数据未知", wi.City), nil
}

func main() {
	ctx := context.Background()

	// ============ 1. 创建 Ollama LLM 客户端 ============
	modelName := os.Getenv("OLLAMA_MODEL")
	if modelName == "" {
		modelName = "llama3"
	}

	baseURL := os.Getenv("OLLAMA_BASE_URL")
	if baseURL == "" {
		baseURL = "http://localhost:11434"
	}

	ollamaClient, err := llm.NewOllamaClient(modelName, baseURL)
	if err != nil {
		log.Fatalf("创建 Ollama 客户端失败: %v", err)
	}

	// 列出可用模型
	models, err := ollamaClient.ListModels(ctx)
	if err != nil {
		log.Printf("警告: 获取模型列表失败: %v", err)
	} else {
		fmt.Println("可用模型:")
		for _, m := range models {
			fmt.Printf("  - %s\n", m.Name)
		}
	}

	// ============ 2. 创建 eino ChatModel ============
	chatModel := eino_examples.NewOllamaChatModel(ollamaClient)

	// ============ 3. 创建工具 ============
	// 计算器工具
	calculatorTool, err := utils.InferTool(
		"calculator",
		"执行数学计算，支持加(add)、减(subtract)、乘(multiply)、除(divide)。输入格式: {\"operation\":\"add\",\"value1\":1,\"value2\":2}",
		CalculatorTool,
	)
	if err != nil {
		log.Fatalf("创建计算器工具失败: %v", err)
	}

	// 搜索工具
	searchTool, err := utils.InferTool(
		"search",
		"搜索信息，输入格式: {\"query\":\"关键词\",\"limit\":3}",
		FakeSearchTool,
	)
	if err != nil {
		log.Fatalf("创建搜索工具失败: %v", err)
	}

	// 天气查询工具
	weatherTool, err := utils.InferTool(
		"weather",
		"查询城市天气，输入格式: {\"city\":\"城市名\"}",
		FakeWeatherTool,
	)
	if err != nil {
		log.Fatalf("创建天气工具失败: %v", err)
	}

	// 工具列表
	tools := []tool.InvokableTool{
		calculatorTool,
		searchTool,
		weatherTool,
	}

	// ============ 4. 创建 ReAct Agent ============
	// 将 tools 转换为 []interface{}
	toolsIfaces := make([]interface{}, len(tools))
	for i, t := range tools {
		toolsIfaces[i] = t
	}
	reactAdapter := eino_examples.NewReactAgentAdapter(
		chatModel,
		toolsIfaces,
		eino_examples.WithTemperature(0.7),
	)

	// ============ 5. 执行测试 ============
	fmt.Println("\n" + strings.Repeat("=", 50))
	fmt.Println("测试 1: 数学计算")
	fmt.Println(strings.Repeat("=", 50))

	result1, err := reactAdapter.Invoke(ctx, map[string]any{
		"prompt": "请计算 123 + 456 的结果",
	})
	if err != nil {
		log.Printf("执行失败: %v", err)
	} else {
		fmt.Printf("结果: %v\n", result1["response"])
	}

	fmt.Println("\n" + strings.Repeat("=", 50))
	fmt.Println("测试 2: 天气查询")
	fmt.Println(strings.Repeat("=", 50))

	result2, err := reactAdapter.Invoke(ctx, map[string]any{
		"prompt": "请查询北京今天的天气",
	})
	if err != nil {
		log.Printf("执行失败: %v", err)
	} else {
		fmt.Printf("结果: %v\n", result2["response"])
	}

	fmt.Println("\n" + strings.Repeat("=", 50))
	fmt.Println("测试 3: 复杂任务 - 计算并搜索")
	fmt.Println(strings.Repeat("=", 50))

	result3, err := reactAdapter.Invoke(ctx, map[string]any{
		"prompt": "先计算 100 除以 4，然后搜索这个结果的相关信息",
	})
	if err != nil {
		log.Printf("执行失败: %v", err)
	} else {
		fmt.Printf("结果: %v\n", result3["response"])
	}

	// ============ 6. 转换为 NodeRunner (用于 CoRag 框架) ============
	fmt.Println("\n" + strings.Repeat("=", 50))
	fmt.Println("测试 4: CoRag 框架集成")
	fmt.Println(strings.Repeat("=", 50))

	runner := eino_examples.ToNodeRunner(reactAdapter)

	payload := &executor.AgentDAGPayload{
		Goal:    "计算 50 乘以 2",
		Results: make(map[string]any),
	}

	execResult, err := runner(ctx, payload)
	if err != nil {
		log.Printf("Runner 执行失败: %v", err)
	} else {
		fmt.Printf("CoRag Runner 结果: %v\n", execResult.Results["eino"])
	}

	// ============ 7. 转换为 Planner TaskNode ============
	taskNode := eino_examples.ConvertToPlannerTaskNode(
		reactAdapter,
		"eino_react",
		map[string]any{
			"model":       modelName,
			"temperature": 0.7,
			"tools":       []string{"calculator", "search", "weather"},
		},
	)
	fmt.Printf("\n转换为 TaskNode: ID=%s, Type=%s, Config=%v\n",
		taskNode.ID, taskNode.Type, taskNode.Config)

	fmt.Println("\n" + strings.Repeat("=", 50))
	fmt.Println("所有测试完成!")
	fmt.Println(strings.Repeat("=", 50))
}
