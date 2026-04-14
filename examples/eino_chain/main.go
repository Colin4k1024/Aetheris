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

// Package main 展示 eino Chain 和 Workflow 组合
//
// 这个示例展示了：
// 1. Chain 组合 - 多个节点顺序执行
// 2. Workflow 组合 - 条件分支和工作流
// 3. 状态在节点间传递
// 4. 与 CoRag 框架的集成
//
// 运行方式：
//
//	go run ./examples/eino_chain/main.go
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/cloudwego/eino/schema"

	eino_examples "github.com/Colin4k1024/Aetheris/v2/internal/agent/runtime/executor/eino_examples"
	"github.com/Colin4k1024/Aetheris/v2/internal/model/llm"
)

// ChainExecutor Chain 执行器
type ChainExecutor struct {
	Name  string
	Nodes []ChainNode
}

// ChainNode 链节点
type ChainNode struct {
	Name    string
	Handler func(ctx context.Context, input any) (any, error)
}

// NewChainExecutor 创建 Chain 执行器
func NewChainExecutor(name string) *ChainExecutor {
	return &ChainExecutor{
		Name:  name,
		Nodes: make([]ChainNode, 0),
	}
}

// AddNode 添加节点
func (c *ChainExecutor) AddNode(name string, handler func(ctx context.Context, input any) (any, error)) *ChainExecutor {
	c.Nodes = append(c.Nodes, ChainNode{
		Name:    name,
		Handler: handler,
	})
	return c
}

// Execute 执行 Chain
func (c *ChainExecutor) Execute(ctx context.Context, input any) (any, error) {
	current := input
	for i, node := range c.Nodes {
		result, err := node.Handler(ctx, current)
		if err != nil {
			return nil, fmt.Errorf("node %s (%d) failed: %w", node.Name, i, err)
		}
		current = result
		log.Printf("Chain [%s] Node [%s] completed", c.Name, node.Name)
	}
	return current, nil
}

// WorkflowNode 工作流节点
type WorkflowNode struct {
	Name    string
	Handler func(ctx context.Context, input any) (any, string, error) // 返回结果、下一节点、错误
}

// WorkflowExecutor 工作流执行器
type WorkflowExecutor struct {
	Name  string
	Nodes map[string]WorkflowNode
	Start string
}

// NewWorkflowExecutor 创建工作流执行器
func NewWorkflowExecutor(name, start string) *WorkflowExecutor {
	return &WorkflowExecutor{
		Name:  name,
		Nodes: make(map[string]WorkflowNode),
		Start: start,
	}
}

// AddNode 添加节点
func (w *WorkflowExecutor) AddNode(name string, handler func(ctx context.Context, input any) (any, string, error)) *WorkflowExecutor {
	w.Nodes[name] = WorkflowNode{
		Name:    name,
		Handler: handler,
	}
	return w
}

// Execute 执行工作流
func (w *WorkflowExecutor) Execute(ctx context.Context, input any) (any, error) {
	current := input
	currentNode := w.Start

	for currentNode != "" {
		node, ok := w.Nodes[currentNode]
		if !ok {
			return nil, fmt.Errorf("node not found: %s", currentNode)
		}

		result, next, err := node.Handler(ctx, current)
		if err != nil {
			return nil, fmt.Errorf("workflow node %s failed: %w", currentNode, err)
		}

		log.Printf("Workflow [%s] Node [%s] -> Next: [%s]", w.Name, currentNode, next)
		current = result
		currentNode = next
	}

	return current, nil
}

func main() {
	ctx := context.Background()

	// ============ 1. 创建 Ollama LLM ============
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

	chatModel := eino_examples.NewOllamaChatModel(ollamaClient)

	// ============ 2. Chain 示例 ============
	fmt.Println("\n" + strings.Repeat("=", 60))
	fmt.Println("测试 Chain 组合")
	fmt.Println(strings.Repeat("=", 60))

	// 创建 Chain: 输入 -> 清洗 -> 增强 -> 输出
	cleanChain := NewChainExecutor("DataProcessing").
		AddNode("Input", func(ctx context.Context, input any) (any, error) {
			// 第一步: 接收原始输入
			log.Println("Chain: 接收原始输入")
			return map[string]any{
				"raw_text":  "  Hello   World  ",
				"raw_count": 3,
			}, nil
		}).
		AddNode("Clean", func(ctx context.Context, input any) (any, error) {
			// 第二步: 清洗数据
			log.Println("Chain: 清洗数据")
			data := input.(map[string]any)
			text := data["raw_text"].(string)
			// 简单清洗
			cleaned := strings.TrimSpace(text)
			cleaned = strings.Join(strings.Fields(cleaned), " ")
			data["cleaned_text"] = cleaned
			return data, nil
		}).
		AddNode("LLMProcess", func(ctx context.Context, input any) (any, error) {
			// 第三步: LLM 处理
			log.Println("Chain: LLM 处理")
			data := input.(map[string]any)
			cleaned := data["cleaned_text"].(string)

			// 调用 LLM
			result, err := chatModel.Generate(ctx, []*schema.Message{
				{Role: schema.User, Content: "Transform this: " + cleaned},
			})
			if err != nil {
				return nil, err
			}
			data["llm_result"] = result.Content
			return data, nil
		}).
		AddNode("Format", func(ctx context.Context, input any) (any, error) {
			// 第四步: 格式化输出
			log.Println("Chain: 格式化输出")
			data := input.(map[string]any)
			return map[string]any{
				"original": data["raw_text"],
				"cleaned":  data["cleaned_text"],
				"enhanced": data["llm_result"],
				"status":   "completed",
			}, nil
		})

	// 执行 Chain
	chainResult, err := cleanChain.Execute(ctx, nil)
	if err != nil {
		log.Printf("Chain 执行失败: %v", err)
	} else {
		fmt.Printf("Chain 结果: %+v\n", chainResult)
	}

	// ============ 3. Workflow 示例 ============
	fmt.Println("\n" + strings.Repeat("=", 60))
	fmt.Println("测试 Workflow 组合")
	fmt.Println(strings.Repeat("=", 60))

	// 创建工作流: 分类 -> 处理 -> 输出
	routingWorkflow := NewWorkflowExecutor("ContentRouting", "Classify").
		AddNode("Classify", func(ctx context.Context, input any) (any, string, error) {
			// 分类节点
			log.Println("Workflow: 分类")
			content := input.(string)
			var category string

			// 使用简单规则分类
			lower := strings.ToLower(content)
			if strings.Contains(lower, "weather") || strings.Contains(lower, "天气") {
				category = "weather"
			} else if strings.Contains(lower, "news") || strings.Contains(lower, "新闻") {
				category = "news"
			} else if strings.Contains(lower, "help") || strings.Contains(lower, "帮助") {
				category = "help"
			} else {
				category = "general"
			}

			return map[string]any{
				"category": category,
				"content":  content,
			}, category, nil
		}).
		AddNode("weather", func(ctx context.Context, input any) (any, string, error) {
			// 天气处理
			log.Println("Workflow: 处理天气请求")
			data := input.(map[string]any)
			result := fmt.Sprintf("天气查询结果: %s - 晴朗", data["content"])
			return result, "Output", nil
		}).
		AddNode("news", func(ctx context.Context, input any) (any, string, error) {
			// 新闻处理
			log.Println("Workflow: 处理新闻请求")
			data := input.(map[string]any)
			result := fmt.Sprintf("新闻摘要: %s - 今日要闻", data["content"])
			return result, "Output", nil
		}).
		AddNode("help", func(ctx context.Context, input any) (any, string, error) {
			// 帮助处理
			log.Println("Workflow: 处理帮助请求")
			data := input.(map[string]any)
			result := fmt.Sprintf("帮助信息: %s - 我可以帮你", data["content"])
			return result, "Output", nil
		}).
		AddNode("general", func(ctx context.Context, input any) (any, string, error) {
			// 一般处理
			log.Println("Workflow: 处理一般请求")
			data := input.(map[string]any)
			result := fmt.Sprintf("一般回复: %s - 收到", data["content"])
			return result, "Output", nil
		}).
		AddNode("Output", func(ctx context.Context, input any) (any, string, error) {
			// 输出节点
			log.Println("Workflow: 输出结果")
			return input, "", nil
		})

	// 测试不同输入
	testInputs := []string{
		"What's the weather today?",
		"Show me the latest news",
		"Can you help me?",
		"Hello there!",
	}

	for _, input := range testInputs {
		fmt.Printf("\n输入: %s\n", input)
		wfResult, err := routingWorkflow.Execute(ctx, input)
		if err != nil {
			log.Printf("Workflow 执行失败: %v", err)
		} else {
			fmt.Printf("输出: %v\n", wfResult)
		}
	}

	// ============ 4. CoRag 集成 ============
	fmt.Println("\n" + strings.Repeat("=", 60))
	fmt.Println("CoRag 框架集成")
	fmt.Println(strings.Repeat("=", 60))

	// 将 Chain 转换为 NodeRunner
	chainRunner := eino_examples.ToNodeRunner(
		eino_examples.NewReactAgentAdapter(chatModel, nil),
	)

	_ = chainRunner

	fmt.Println("Chain 和 Workflow 已就绪，可以集成到 CoRag 框架")

	fmt.Println("\n" + strings.Repeat("=", 60))
	fmt.Println("所有测试完成!")
	fmt.Println(strings.Repeat("=", 60))
}
