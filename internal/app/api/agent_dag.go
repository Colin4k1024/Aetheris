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

package api

import (
	"context"
	"fmt"
	"os"
	"strings"

	einomodel "github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"

	"rag-platform/internal/agent/planner"
	"rag-platform/internal/agent/runtime"
	agentexec "rag-platform/internal/agent/runtime/executor"
	"rag-platform/internal/agent/tools"
	"rag-platform/internal/model/llm"
	"rag-platform/internal/runtime/eino"
	runtimesession "rag-platform/internal/runtime/session"
)

// llmGenAdapter 将 llm.Client 适配为 executor.LLMGen
type llmGenAdapter struct {
	client llm.Client
}

func (a *llmGenAdapter) Generate(ctx context.Context, prompt string) (string, error) {
	if a.client == nil {
		return "", fmt.Errorf("LLM not configured")
	}
	return a.client.GenerateWithContext(ctx, prompt, llm.GenerateOptions{MaxTokens: 4096, Temperature: 0.1})
}

// einoToolCallingAdapter 将 eino model.ToolCallingChatModel 适配为 executor.EinoToolCallingChatModel
type einoToolCallingAdapter struct {
	model einomodel.ToolCallingChatModel
}

func (a *einoToolCallingAdapter) Generate(ctx context.Context, input []*schema.Message) (*schema.Message, error) {
	return a.model.Generate(ctx, input)
}

func (a *einoToolCallingAdapter) Stream(ctx context.Context, input []*schema.Message) (*schema.StreamReader[*schema.Message], error) {
	return a.model.Stream(ctx, input)
}

func (a *einoToolCallingAdapter) WithTools(tools []*schema.ToolInfo) (agentexec.EinoToolCallingChatModel, error) {
	m, err := a.model.WithTools(tools)
	if err != nil {
		return nil, err
	}
	return &einoToolCallingAdapter{model: m}, nil
}

// toolExecAdapter 从 ctx 取 agent，将 runtime.Session 转为 runtime/session.Session 后调 agent/tools
type toolExecAdapter struct {
	reg *tools.Registry
}

func (a *toolExecAdapter) Execute(ctx context.Context, toolName string, input map[string]any, state interface{}) (agentexec.ToolResult, error) {
	if a.reg == nil {
		return agentexec.ToolResult{}, fmt.Errorf("Tools not configured")
	}
	agent := agentexec.AgentFromContext(ctx)
	var sess *runtimesession.Session
	if agent != nil && agent.Session != nil {
		sess = agentSessionToRuntime(agent.Session)
	} else {
		sess = runtimesession.New("")
	}
	t, ok := a.reg.Get(toolName)
	if !ok {
		return agentexec.ToolResult{}, fmt.Errorf("未知工具: %s", toolName)
	}
	out, err := t.Execute(ctx, sess, input, state)
	if err != nil {
		return agentexec.ToolResult{Err: err.Error()}, err
	}
	if tr, ok := out.(tools.ToolResult); ok {
		return agentexec.ToolResult{Done: tr.Done, State: tr.State, Output: tr.Output, Err: tr.Err}, nil
	}
	return agentexec.ToolResult{Done: true, Output: fmt.Sprint(out)}, nil
}

// agentSessionToRuntime 将 agent/runtime.Session 转为 runtime/session.Session（拷贝 Messages）
func agentSessionToRuntime(s *runtime.Session) *runtimesession.Session {
	if s == nil {
		return runtimesession.New("")
	}
	sess := runtimesession.New(s.ID)
	sess.UpdatedAt = s.GetUpdatedAt()
	msgs := s.CopyMessages()
	for _, m := range msgs {
		sess.AddMessage(m.Role, m.Content)
	}
	return sess
}

// workflowExecAdapter 调用 eino Engine.ExecuteWorkflow
type workflowExecAdapter struct {
	engine *eino.Engine
}

func (a *workflowExecAdapter) ExecuteWorkflow(ctx context.Context, name string, params map[string]any) (interface{}, error) {
	if a.engine == nil {
		return nil, fmt.Errorf("Workflow engine not configured")
	}
	pm := make(map[string]interface{}, len(params))
	for k, v := range params {
		pm[k] = v
	}
	return a.engine.ExecuteWorkflow(ctx, name, pm)
}

// NewDAGCompiler 创建 TaskGraph→eino DAG 的编译器（注册 llm/tool/workflow 适配器）；toolEventSink/commandEventSink 可选；invocationStore 可选；effectStore 可选，非 nil 时启用两步提交与强 Replay catch-up；resourceVerifier 可选；attemptValidator 可选，非 nil 时 Ledger Commit 前校验 attempt（Lease fencing）
func NewDAGCompiler(llmClient llm.Client, toolsReg *tools.Registry, engine *eino.Engine, toolEventSink agentexec.ToolEventSink, commandEventSink agentexec.CommandEventSink, invocationStore agentexec.ToolInvocationStore, effectStore agentexec.EffectStore, resourceVerifier agentexec.ResourceVerifier, attemptValidator agentexec.AttemptValidator) *agentexec.Compiler {
	return NewDAGCompilerWithOptions(llmClient, toolsReg, engine, toolEventSink, commandEventSink, invocationStore, effectStore, resourceVerifier, attemptValidator, nil)
}

// NewDAGCompilerWithOptions 创建 DAG 编译器，支持可选的 Tool 限流器。
func NewDAGCompilerWithOptions(llmClient llm.Client, toolsReg *tools.Registry, engine *eino.Engine, toolEventSink agentexec.ToolEventSink, commandEventSink agentexec.CommandEventSink, invocationStore agentexec.ToolInvocationStore, effectStore agentexec.EffectStore, resourceVerifier agentexec.ResourceVerifier, attemptValidator agentexec.AttemptValidator, toolRateLimiter *agentexec.ToolRateLimiter) *agentexec.Compiler {
	toolAdapter := &agentexec.ToolNodeAdapter{
		Tools:              &toolExecAdapter{reg: toolsReg},
		ToolCapabilityFunc: toolsReg.GetCapability,
	}
	if toolRateLimiter != nil {
		toolAdapter.RateLimiter = toolRateLimiter
	}
	if toolEventSink != nil {
		toolAdapter.ToolEventSink = toolEventSink
	}
	if commandEventSink != nil {
		toolAdapter.CommandEventSink = commandEventSink
	}
	if invocationStore != nil {
		toolAdapter.InvocationStore = invocationStore
		if attemptValidator != nil {
			toolAdapter.InvocationLedger = agentexec.NewInvocationLedger(invocationStore, attemptValidator)
		} else {
			toolAdapter.InvocationLedger = agentexec.NewInvocationLedgerFromStore(invocationStore)
		}
	}
	if effectStore != nil {
		toolAdapter.EffectStore = effectStore
	}
	if resourceVerifier != nil {
		toolAdapter.ResourceVerifier = resourceVerifier
	}
	llmAdapter := &agentexec.LLMNodeAdapter{LLM: &llmGenAdapter{client: llmClient}}
	if commandEventSink != nil {
		llmAdapter.CommandEventSink = commandEventSink
	}
	if effectStore != nil {
		llmAdapter.EffectStore = effectStore
	}
	workflowAdapter := &agentexec.WorkflowNodeAdapter{Workflow: &workflowExecAdapter{engine: engine}}
	if commandEventSink != nil {
		workflowAdapter.CommandEventSink = commandEventSink
	}

	// 创建 ToolCallingChatModel 用于 Eino Agent
	var toolCallingLLM agentexec.EinoToolCallingChatModel
	if engine != nil {
		ctx := context.Background()
		tcLLM, err := engine.CreateChatModel(ctx)
		if err != nil {
			// 如果创建失败，使用普通模式
			fmt.Printf("Warning: 创建 ToolCallingChatModel 失败: %v\n", err)
		} else {
			toolCallingLLM = &einoToolCallingAdapter{model: tcLLM}
		}
	}

	// 创建 Eino Agent 适配器（带内置工具和 ToolCalling LLM）
	builtinTools := agentexec.NewBuiltinTools()
	einoReactAdapter := &agentexec.EinoNodeAdapter{
		NodeType:       planner.NodeEinoReact,
		Tools:          builtinTools,
		ToolCallingLLM: toolCallingLLM,
		MaxIterations:  10,
	}
	einoDEERAdapter := &agentexec.EinoNodeAdapter{
		NodeType:       planner.NodeEinoDEER,
		Tools:          builtinTools,
		ToolCallingLLM: toolCallingLLM,
		MaxIterations:  10,
	}
	einoManusAdapter := &agentexec.EinoNodeAdapter{
		NodeType:       planner.NodeEinoManus,
		Tools:          builtinTools,
		ToolCallingLLM: toolCallingLLM,
		MaxIterations:  10,
	}

	adapters := map[string]agentexec.NodeAdapter{
		planner.NodeLLM:       llmAdapter,
		planner.NodeTool:      toolAdapter,
		planner.NodeWorkflow:  workflowAdapter,
		planner.NodeWait:      &agentexec.WaitNodeAdapter{},
		planner.NodeApproval:  &agentexec.ApprovalNodeAdapter{},
		planner.NodeCondition: &agentexec.ConditionNodeAdapter{},
		// Eino Agent 节点类型
		planner.NodeEinoReact: einoReactAdapter,
		planner.NodeEinoDEER:  einoDEERAdapter,
		planner.NodeEinoManus: einoManusAdapter,
	}
	if legacyNodeAdaptersEnabled() {
		adapters[planner.NodeLangChainGo] = &agentexec.LangChainGoNodeAdapter{}
		adapters[planner.NodeLangGraphGo] = &agentexec.LangGraphGoNodeAdapter{}
		adapters[planner.NodeADK] = &agentexec.ADKNodeAdapter{}
		adapters[planner.NodeGenkit] = &agentexec.GenkitNodeAdapter{}
		adapters[planner.NodeProtocolLattice] = &agentexec.ProtocolLatticeNodeAdapter{}
		adapters[planner.NodeLinGoose] = &agentexec.LinGooseNodeAdapter{}
		adapters[planner.NodeAnyi] = &agentexec.AnyiNodeAdapter{}
		adapters[planner.NodeAgentSDK] = &agentexec.AgentSDKNodeAdapter{}
	}
	return agentexec.NewCompiler(adapters)
}

// NewDAGRunner 创建 DAG 执行 Runner
func NewDAGRunner(compiler *agentexec.Compiler) *agentexec.Runner {
	return agentexec.NewRunner(compiler)
}

// RunFuncForScheduler 返回可供 Scheduler.SetRunFunc 使用的回调：从 Manager 取 Agent，取最后一条 user 消息为 goal，调用 Runner.Run
func RunFuncForScheduler(manager *runtime.Manager, runner *agentexec.Runner) func(context.Context, string) {
	return func(ctx context.Context, agentID string) {
		agent, _ := manager.Get(ctx, agentID)
		if agent == nil {
			return
		}
		goal := lastUserMessage(agent.Session)
		if goal == "" {
			goal = "请根据当前上下文回复。"
		}
		_ = runner.Run(ctx, agent, goal)
	}
}

func lastUserMessage(s *runtime.Session) string {
	if s == nil {
		return ""
	}
	msgs := s.CopyMessages()
	for i := len(msgs) - 1; i >= 0; i-- {
		if msgs[i].Role == "user" {
			return msgs[i].Content
		}
	}
	return ""
}

func legacyNodeAdaptersEnabled() bool {
	v := strings.TrimSpace(strings.ToLower(os.Getenv("AETHERIS_ENABLE_LEGACY_NODE_ADAPTERS")))
	return v == "1" || v == "true" || v == "yes" || v == "on"
}
