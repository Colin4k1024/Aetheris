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

package eino

import (
	"context"
	"fmt"
	"time"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/components/tool/utils"
	"github.com/cloudwego/eino/compose"
	"go.opentelemetry.io/otel/attribute"

	"rag-platform/pkg/tracing"
)

// WorkflowConfig 工作流配置
type WorkflowConfig struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

// NodeConfig 节点配置
type NodeConfig struct {
	Name        string `json:"name"`
	Type        string `json:"type"`
	Description string `json:"description"`
}

// CreateWorkflow 创建工作流
func CreateWorkflow(name, description string) *Workflow {
	return &Workflow{
		name:        name,
		description: description,
		graph:       compose.NewGraph[*Input, *Output](),
	}
}

// Workflow 工作流
type Workflow struct {
	name        string
	description string
	graph       *compose.Graph[*Input, *Output]
}

// Input 输入
type Input struct {
	Query string `json:"query"`
}

// Output 输出
type Output struct {
	Result string `json:"result"`
}

// AddNode 添加节点
func (w *Workflow) AddNode(name, nodeType string, config *NodeConfig) error {
	switch nodeType {
	case "validate":
		if err := w.graph.AddLambdaNode(name, compose.InvokableLambda(func(ctx context.Context, input *Input) (*Output, error) {
			// 添加 DAG node tracing
			nodeSpan, _ := tracing.StartDAGNodeSpan(ctx, name, nodeType)
			defer nodeSpan.End(nil)
			nodeSpan.SetInputSize(len(input.Query))

			if input.Query == "" {
				return nil, fmt.Errorf("query is required")
			}
			result := input.Query
			nodeSpan.SetOutputSize(len(result))
			return &Output{Result: result}, nil
		})); err != nil {
			return err
		}
	case "generate":
		return fmt.Errorf("generate 节点requires chatModel 实例")
	case "format":
		if err := w.graph.AddLambdaNode(name, compose.InvokableLambda(func(ctx context.Context, input *Output) (*Output, error) {
			// 添加 DAG node tracing
			nodeSpan, _ := tracing.StartDAGNodeSpan(ctx, name, nodeType)
			defer nodeSpan.End(nil)
			nodeSpan.SetInputSize(len(input.Result))

			result := fmt.Sprintf("格式化结果: %s", input.Result)
			nodeSpan.SetOutputSize(len(result))
			return &Output{Result: result}, nil
		})); err != nil {
			return err
		}
	default:
		return fmt.Errorf("unsupported input type节点类型: %s", nodeType)
	}

	return nil
}

// AddEdge 添加边
func (w *Workflow) AddEdge(from, to string) error {
	return w.graph.AddEdge(from, to)
}

// Compile 编译工作流
func (w *Workflow) Compile(ctx context.Context) (compose.Runnable[*Input, *Output], error) {
	ctx, compileSpan := tracing.StartCompileSpan(ctx)
	defer compileSpan.End(nil)

	start := time.Now()
	runnable, err := w.graph.Compile(ctx)
	compileDuration := time.Since(start)

	if err != nil {
		return nil, fmt.Errorf("compile workflow failed: %w", err)
	}

	// 记录编译耗时
	_ = compileDuration
	return runnable, nil
}

// Execute 执行工作流
func (w *Workflow) Execute(ctx context.Context, input *Input) (*Output, error) {
	ctx, invokeSpan := tracing.StartInvokeSpan(ctx)
	defer invokeSpan.End(nil)

	start := time.Now()
	runnable, err := w.Compile(ctx)
	if err != nil {
		return nil, fmt.Errorf("compile workflow failed: %w", err)
	}

	output, err := runnable.Invoke(ctx, input)
	duration := time.Since(start)

	if err != nil {
		return nil, fmt.Errorf("invoke workflow failed: %w", err)
	}

	// 记录输出大小
	if output != nil {
		invokeSpan.SetAttributes(attribute.Int("workflow.output_size", len(output.Result)))
	}

	_ = duration
	return output, nil
}

// CreateIngestWorkflow 创建入库工作流
func CreateIngestWorkflow(ctx context.Context) (*Workflow, error) {
	workflow := CreateWorkflow("ingest_workflow", "文档入库工作流")

	// 添加节点
	if err := workflow.AddNode("validate", "validate", &NodeConfig{
		Name:        "validate",
		Type:        "validate",
		Description: "验证输入",
	}); err != nil {
		return nil, err
	}

	if err := workflow.AddNode("format", "format", &NodeConfig{
		Name:        "format",
		Type:        "format",
		Description: "格式化输出",
	}); err != nil {
		return nil, err
	}

	// 添加边
	if err := workflow.AddEdge(compose.START, "validate"); err != nil {
		return nil, err
	}

	if err := workflow.AddEdge("validate", "format"); err != nil {
		return nil, err
	}

	if err := workflow.AddEdge("format", compose.END); err != nil {
		return nil, err
	}

	return workflow, nil
}

// CreateQueryWorkflow 创建查询工作流
func CreateQueryWorkflow(ctx context.Context) (*Workflow, error) {
	workflow := CreateWorkflow("query_workflow", "查询工作流")

	// 添加节点
	if err := workflow.AddNode("validate", "validate", &NodeConfig{
		Name:        "validate",
		Type:        "validate",
		Description: "验证输入",
	}); err != nil {
		return nil, err
	}

	if err := workflow.AddNode("format", "format", &NodeConfig{
		Name:        "format",
		Type:        "format",
		Description: "格式化输出",
	}); err != nil {
		return nil, err
	}

	// 添加边
	if err := workflow.AddEdge(compose.START, "validate"); err != nil {
		return nil, err
	}

	if err := workflow.AddEdge("validate", "format"); err != nil {
		return nil, err
	}

	if err := workflow.AddEdge("format", compose.END); err != nil {
		return nil, err
	}

	return workflow, nil
}

// CreateToolFromWorkflow 从工作流创建工具
func CreateToolFromWorkflow(workflow *Workflow, toolName, toolDescription string) (tool.BaseTool, error) {
	runnable, err := workflow.Compile(context.Background())
	if err != nil {
		return nil, fmt.Errorf("compile workflow failed: %w", err)
	}

	return utils.InferTool(toolName, toolDescription, func(ctx context.Context, input string) (string, error) {
		output, err := runnable.Invoke(ctx, &Input{Query: input})
		if err != nil {
			return "", err
		}
		return output.Result, nil
	})
}
