package api

import (
	"context"
	"encoding/json"
	"time"

	"github.com/Colin4k1024/Aetheris/v2/internal/agent/planner"
	agentexec "github.com/Colin4k1024/Aetheris/v2/internal/agent/runtime/executor"
	"github.com/Colin4k1024/Aetheris/v2/internal/agent/tools"
	apihttp "github.com/Colin4k1024/Aetheris/v2/internal/api/http"
	"github.com/Colin4k1024/Aetheris/v2/internal/model/llm"
)

type runtimeBridgeInvoker struct {
	toolAdapter *agentexec.ToolNodeAdapter
	llmAdapter  *agentexec.LLMNodeAdapter
	nodeSink    agentexec.NodeEventSink
}

func NewRuntimeBridgeInvoker(llmClient llm.Client, toolsReg *tools.Registry, nodeSink agentexec.NodeToolAndCommandEventSink, invocationStore agentexec.ToolInvocationStore, effectStore agentexec.EffectStore, attemptValidator agentexec.AttemptValidator, toolRateLimiter *agentexec.ToolRateLimiter) apihttp.RuntimeBridgeInvoker {
	toolAdapter := &agentexec.ToolNodeAdapter{
		Tools: &toolExecAdapter{reg: toolsReg},
	}
	if toolsReg != nil {
		toolAdapter.ToolCapabilityFunc = toolsReg.GetCapability
	}
	if nodeSink != nil {
		toolAdapter.ToolEventSink = nodeSink
		toolAdapter.CommandEventSink = nodeSink
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
	if toolRateLimiter != nil {
		toolAdapter.RateLimiter = toolRateLimiter
	}
	llmAdapter := &agentexec.LLMNodeAdapter{LLM: &llmGenAdapter{client: llmClient}}
	if nodeSink != nil {
		llmAdapter.CommandEventSink = nodeSink
	}
	if effectStore != nil {
		llmAdapter.EffectStore = effectStore
	}
	return &runtimeBridgeInvoker{
		toolAdapter: toolAdapter,
		llmAdapter:  llmAdapter,
		nodeSink:    nodeSink,
	}
}

func (b *runtimeBridgeInvoker) InvokeTool(ctx context.Context, req apihttp.RuntimeBridgeToolRequest) (map[string]any, error) {
	task := &planner.TaskNode{ID: req.NodeID, Type: planner.NodeTool, ToolName: req.ToolName, Config: req.Input}
	run, err := b.toolAdapter.ToNodeRunner(task, nil)
	if err != nil {
		return nil, err
	}
	return b.runBridgeNode(ctx, req.JobID, req.NodeID, planner.NodeTool, req.SessionID, req.PriorResults, run, agentexec.StepResultSideEffectCommitted)
}

func (b *runtimeBridgeInvoker) InvokeLLM(ctx context.Context, req apihttp.RuntimeBridgeLLMRequest) (map[string]any, error) {
	cfg := map[string]any{}
	if req.Prompt != "" {
		cfg["goal"] = req.Prompt
	}
	if req.PromptKey != "" {
		cfg["prompt_key"] = req.PromptKey
	}
	task := &planner.TaskNode{ID: req.NodeID, Type: planner.NodeLLM, Config: cfg}
	run, err := b.llmAdapter.ToNodeRunner(task, nil)
	if err != nil {
		return nil, err
	}
	payload, err := b.runBridgeNode(ctx, req.JobID, req.NodeID, planner.NodeLLM, req.SessionID, req.PriorResults, run, agentexec.StepResultPure)
	if err != nil {
		return nil, err
	}
	return payload, nil
}

func (b *runtimeBridgeInvoker) runBridgeNode(ctx context.Context, jobID, nodeID, nodeType, sessionID string, priorResults map[string]any, run agentexec.NodeRunner, successType agentexec.StepResultType) (map[string]any, error) {
	start := time.Now()
	runCtx := agentexec.WithJobID(ctx, jobID)
	runCtx = agentexec.WithExecutionStepID(runCtx, nodeID)
	payload := agentexec.NewAgentDAGPayload("", "", sessionID)
	if priorResults != nil {
		payload.Results = priorResults
	}
	if b.nodeSink != nil && jobID != "" {
		_ = b.nodeSink.AppendNodeStarted(runCtx, jobID, nodeID, 1, "runtime_bridge")
	}
	out, err := run(runCtx, payload)
	if out == nil {
		out = payload
	}
	resultType := successType
	reason := ""
	if err != nil {
		resultType, reason = agentexec.ClassifyError(err)
	}
	nodeResult := map[string]any{}
	if out.Results != nil {
		if v, ok := out.Results[nodeID]; ok {
			nodeResult["result"] = v
		}
	}
	if b.nodeSink != nil && jobID != "" {
		resultBytes, _ := json.Marshal(nodeResult)
		_ = b.nodeSink.AppendNodeFinished(runCtx, jobID, nodeID, resultBytes, time.Since(start).Milliseconds(), "completed", 1, resultType, reason, nodeID, "")
	}
	if err != nil {
		return nil, err
	}
	if len(nodeResult) == 0 {
		nodeResult["result"] = map[string]any{}
	}
	return nodeResult, nil
}
