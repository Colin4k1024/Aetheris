package executor

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"rag-platform/internal/agent/planner"
	"rag-platform/internal/agent/replay"
	"rag-platform/internal/agent/runtime"
	"rag-platform/internal/runtime/jobstore"
)

// TestRunForJob_PendingToolInvocation_EffectStoreCatchUp 证明：
// Runner 从事件流恢复出 pending tool invocation 时，若 Effect Store 已有结果，
// 会走 catch-up 补齐 tool_invocation_finished / command_committed / node_finished，且不再次执行 tool。
func TestRunForJob_PendingToolInvocation_EffectStoreCatchUp(t *testing.T) {
	ctx := context.Background()
	jobID := "job-tool-effect-catchup"
	taskID := "tool1"
	toolName := "refund"
	cfg := map[string]any{"amount": 42}
	result := []byte(`{"done":true,"output":"from-effect-store"}`)
	eventStore := jobstore.NewMemoryStore()
	effectStore := NewEffectStoreMem()
	invocationStore := NewToolInvocationStoreMem()
	ledger := NewInvocationLedgerFromStore(invocationStore)
	graph := &planner.TaskGraph{
		Nodes: []planner.TaskNode{{ID: taskID, Type: planner.NodeTool, ToolName: toolName, Config: cfg}},
		Edges: []planner.TaskEdge{},
	}
	graphBytes, err := graph.Marshal()
	if err != nil {
		t.Fatalf("marshal graph: %v", err)
	}
	planPayload, _ := json.Marshal(map[string]any{
		"task_graph": json.RawMessage(graphBytes),
		"goal":       "g1",
	})
	if _, err := eventStore.Append(ctx, jobID, 0, jobstore.JobEvent{JobID: jobID, Type: jobstore.PlanGenerated, Payload: planPayload}); err != nil {
		t.Fatalf("append plan_generated: %v", err)
	}
	stepID := DeterministicStepID(jobID, PlanDecisionID(graphBytes), 0, planner.NodeTool)
	idempotencyKey := IdempotencyKey(jobID, stepID, toolName, cfg)
	argsHash := ArgumentsHash(cfg)
	startedPayload, _ := json.Marshal(map[string]any{
		"node_id":         taskID,
		"invocation_id":   "inv-pending-effect",
		"tool_name":       toolName,
		"idempotency_key": idempotencyKey,
		"arguments_hash":  argsHash,
		"started_at":      FormatStartedAt(time.Now().UTC()),
	})
	if _, err := eventStore.Append(ctx, jobID, 1, jobstore.JobEvent{JobID: jobID, Type: jobstore.ToolInvocationStarted, Payload: startedPayload}); err != nil {
		t.Fatalf("append tool_invocation_started: %v", err)
	}
	if err := invocationStore.SetStarted(ctx, &ToolInvocationRecord{
		InvocationID:   "inv-pending-effect",
		JobID:          jobID,
		StepID:         stepID,
		ToolName:       toolName,
		ArgsHash:       argsHash,
		IdempotencyKey: idempotencyKey,
		Status:         ToolInvocationStatusStarted,
	}); err != nil {
		t.Fatalf("SetStarted: %v", err)
	}
	if err := effectStore.PutEffect(ctx, &EffectRecord{
		JobID:          jobID,
		CommandID:      taskID,
		IdempotencyKey: idempotencyKey,
		Kind:           EffectKindTool,
		Output:         result,
		Metadata:       map[string]any{"tool_name": toolName},
	}); err != nil {
		t.Fatalf("PutEffect: %v", err)
	}

	var callCount int32
	sink := &jobStoreReplaySink{store: eventStore}
	adapter := &ToolNodeAdapter{
		Tools:            &countToolExec{count: &callCount},
		InvocationStore:  invocationStore,
		InvocationLedger: ledger,
		EffectStore:      effectStore,
		ToolEventSink:    sink,
		CommandEventSink: sink,
	}
	compiler := NewCompiler(map[string]NodeAdapter{planner.NodeTool: adapter})
	runner := NewRunner(compiler)
	fakeJobStore := &fakeJobStoreForRunner{}
	runner.SetCheckpointStores(runtime.NewCheckpointStoreMem(), fakeJobStore)
	runner.SetReplayContextBuilder(replay.NewReplayContextBuilder(eventStore))
	runner.SetNodeEventSink(sink)

	err = runner.RunForJob(ctx, &runtime.Agent{ID: "a1"}, &JobForRunner{ID: jobID, AgentID: "a1", Goal: "g1"})
	if err != nil {
		t.Fatalf("RunForJob: %v", err)
	}
	if atomic.LoadInt32(&callCount) != 0 {
		t.Fatalf("expected 0 tool calls during replay catch-up, got %d", callCount)
	}
	recovered, exists := ledger.Recover(ctx, jobID, idempotencyKey)
	if !exists || string(recovered) != string(result) {
		t.Fatalf("expected ledger recovered result %s, got (%s, %v)", string(result), string(recovered), exists)
	}
	events, _, err := eventStore.ListEvents(ctx, jobID)
	if err != nil {
		t.Fatalf("ListEvents: %v", err)
	}
	var sawFinished bool
	var sawCommitted bool
	for _, event := range events {
		switch event.Type {
		case jobstore.ToolInvocationFinished:
			var payload struct {
				IdempotencyKey string          `json:"idempotency_key"`
				Outcome        string          `json:"outcome"`
				Result         json.RawMessage `json:"result"`
			}
			if err := json.Unmarshal(event.Payload, &payload); err != nil {
				t.Fatalf("unmarshal tool_invocation_finished: %v", err)
			}
			if payload.IdempotencyKey == idempotencyKey && payload.Outcome == ToolInvocationOutcomeSuccess && string(payload.Result) == string(result) {
				sawFinished = true
			}
		case jobstore.CommandCommitted:
			var payload struct {
				CommandID string          `json:"command_id"`
				Result    json.RawMessage `json:"result"`
			}
			if err := json.Unmarshal(event.Payload, &payload); err != nil {
				t.Fatalf("unmarshal command_committed: %v", err)
			}
			if payload.CommandID == taskID && string(payload.Result) == string(result) {
				sawCommitted = true
			}
		}
	}
	if !sawFinished {
		t.Fatal("expected replay catch-up to append tool_invocation_finished")
	}
	if !sawCommitted {
		t.Fatal("expected replay catch-up to append command_committed")
	}
	rc, err := replay.NewReplayContextBuilder(eventStore).BuildFromEvents(ctx, jobID)
	if err != nil {
		t.Fatalf("BuildFromEvents: %v", err)
	}
	if _, pending := rc.PendingToolInvocations[idempotencyKey]; pending {
		t.Fatalf("expected pending tool invocation to be cleared after catch-up")
	}
	if got, ok := rc.CompletedToolInvocations[idempotencyKey]; !ok || string(got) != string(result) {
		t.Fatalf("expected completed tool invocation %s, got %s, exists=%v", string(result), string(got), ok)
	}
	if _, ok := rc.CompletedCommandIDs[taskID]; !ok {
		t.Fatalf("expected command %s to be marked committed", taskID)
	}
	_, status := fakeJobStore.getLast()
	const statusCompleted = 2
	if status != statusCompleted {
		t.Fatalf("UpdateStatus = %d, want %d", status, statusCompleted)
	}
}

// TestRunForJob_StaleAttemptCommitFailsJob 证明：
// 若工具已经真实执行，但在提交阶段因 stale attempt 被 fencing 拒绝，
// step.Run 必须返回错误，Runner 将 Job 收敛为 Failed，而不是错误地标记成功。
func TestRunForJob_StaleAttemptCommitFailsJob(t *testing.T) {
	ctx := WithJobID(context.Background(), job1)
	var callCount int32
	store := NewToolInvocationStoreMem()
	ledger := NewInvocationLedger(store, staticAttemptValidator{err: jobstore.ErrStaleAttempt})
	adapter := &ToolNodeAdapter{
		Tools:            &countToolExec{count: &callCount},
		InvocationStore:  store,
		InvocationLedger: ledger,
	}
	graph := &planner.TaskGraph{
		Nodes: []planner.TaskNode{{ID: step1, Type: planner.NodeTool, ToolName: tool1, Config: map[string]any{"a": 1}}},
		Edges: []planner.TaskEdge{},
	}
	graphBytes, err := graph.Marshal()
	if err != nil {
		t.Fatalf("marshal graph: %v", err)
	}
	eventStore := jobstore.NewMemoryStore()
	planPayload, _ := json.Marshal(map[string]any{"task_graph": json.RawMessage(graphBytes), "goal": "g1"})
	if _, err := eventStore.Append(ctx, job1, 0, jobstore.JobEvent{JobID: job1, Type: jobstore.PlanGenerated, Payload: planPayload}); err != nil {
		t.Fatalf("append plan_generated: %v", err)
	}
	runner := NewRunner(NewCompiler(map[string]NodeAdapter{planner.NodeTool: adapter}))
	fakeJobStore := &fakeJobStoreForRunner{}
	runner.SetCheckpointStores(runtime.NewCheckpointStoreMem(), fakeJobStore)
	runner.SetReplayContextBuilder(replay.NewReplayContextBuilder(eventStore))

	err = runner.RunForJob(ctx, &runtime.Agent{ID: "a1"}, &JobForRunner{ID: job1, AgentID: "a1", Goal: "g1"})
	if !errors.Is(err, jobstore.ErrStaleAttempt) {
		t.Fatalf("expected ErrStaleAttempt from RunForJob, got %v", err)
	}
	if atomic.LoadInt32(&callCount) != 1 {
		t.Fatalf("expected tool to execute exactly once before commit fencing failed, got %d", callCount)
	}
	if recovered, exists := ledger.Recover(ctx, job1, key1); exists || recovered != nil {
		t.Fatalf("stale-attempt failure must not commit ledger result, got (%s, %v)", string(recovered), exists)
	}
	_, status := fakeJobStore.getLast()
	const statusFailed = 3
	if status != statusFailed {
		t.Fatalf("UpdateStatus = %d, want %d", status, statusFailed)
	}
	events, _, err := eventStore.ListEvents(ctx, job1)
	if err != nil {
		t.Fatalf("ListEvents: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("expected only original plan event in event store, got %d events", len(events))
	}
}

// TestRunForJob_StaleAttemptNodeFinishFailsJob 证明：
// 若 step 已执行成功，但 runner 在写 node_finished / step_committed 时被 stale-attempt fencing 拒绝，
// RunForJob 必须失败并将 Job 收敛为 Failed，而不是继续 checkpoint / cursor 前进。
func TestRunForJob_StaleAttemptNodeFinishFailsJob(t *testing.T) {
	ctx := context.Background()
	jobID := "job-node-finish-stale"
	eventStore := jobstore.NewMemoryStore()
	graph := &planner.TaskGraph{
		Nodes: []planner.TaskNode{{ID: "n1", Type: planner.NodeLLM}},
		Edges: []planner.TaskEdge{},
	}
	graphBytes, err := graph.Marshal()
	if err != nil {
		t.Fatalf("marshal graph: %v", err)
	}
	planPayload, _ := json.Marshal(map[string]any{"task_graph": json.RawMessage(graphBytes), "goal": "g1"})
	if _, err := eventStore.Append(ctx, jobID, 0, jobstore.JobEvent{JobID: jobID, Type: jobstore.PlanGenerated, Payload: planPayload}); err != nil {
		t.Fatalf("append plan_generated: %v", err)
	}
	var callCount int32
	mockLLM := &mockLLMWithCallCount{callCount: &callCount, response: "ok"}
	runner := NewRunner(NewCompiler(map[string]NodeAdapter{planner.NodeLLM: &LLMNodeAdapter{LLM: mockLLM}}))
	fakeJobStore := &fakeJobStoreForRunner{}
	runner.SetCheckpointStores(runtime.NewCheckpointStoreMem(), fakeJobStore)
	runner.SetReplayContextBuilder(replay.NewReplayContextBuilder(eventStore))
	runner.SetNodeEventSink(staleNodeFinishSink{})

	err = runner.RunForJob(ctx, &runtime.Agent{ID: "a1"}, &JobForRunner{ID: jobID, AgentID: "a1", Goal: "g1"})
	if !errors.Is(err, jobstore.ErrStaleAttempt) {
		t.Fatalf("expected ErrStaleAttempt from node finish path, got %v", err)
	}
	if atomic.LoadInt32(&callCount) != 1 {
		t.Fatalf("expected step to execute once before node finish rejection, got %d", callCount)
	}
	_, status := fakeJobStore.getLast()
	const statusFailed = 3
	if status != statusFailed {
		t.Fatalf("UpdateStatus = %d, want %d", status, statusFailed)
	}
	events, _, err := eventStore.ListEvents(ctx, jobID)
	if err != nil {
		t.Fatalf("ListEvents: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("expected only plan_generated to remain, got %d events", len(events))
	}
}

// TestRunForJob_ReplayBackfillNodeFinishStaleAttemptFails 证明：
// 当 Replay 已命中 command_committed，仅需 backfill node_finished / step_committed 时，
// 若该写入被 stale-attempt 拒绝，RunForJob 必须失败，且不得退回真实执行节点。
func TestRunForJob_ReplayBackfillNodeFinishStaleAttemptFails(t *testing.T) {
	ctx := context.Background()
	jobID := "job-replay-backfill-stale"
	eventStore := jobstore.NewMemoryStore()
	graph := &planner.TaskGraph{
		Nodes: []planner.TaskNode{{ID: "n1", Type: planner.NodeLLM}},
		Edges: []planner.TaskEdge{},
	}
	buildReplayableEventStream(t, eventStore, jobID, graph, map[string][]byte{"n1": []byte(`"replayed"`)})
	// 删除最后一条 node_finished，只保留 plan_generated + command_committed，模拟需要 backfill node_finished 的 replay。
	events, _, err := eventStore.ListEvents(ctx, jobID)
	if err != nil {
		t.Fatalf("ListEvents: %v", err)
	}
	trimmed := jobstore.NewMemoryStore()
	for i, event := range events[:2] {
		if _, err := trimmed.Append(ctx, jobID, i, event); err != nil {
			t.Fatalf("Append trimmed event %d: %v", i, err)
		}
	}
	var callCount int32
	mockLLM := &mockLLMWithCallCount{callCount: &callCount, response: "should not execute"}
	runner := NewRunner(NewCompiler(map[string]NodeAdapter{planner.NodeLLM: &LLMNodeAdapter{LLM: mockLLM}}))
	fakeJobStore := &fakeJobStoreForRunner{}
	runner.SetCheckpointStores(runtime.NewCheckpointStoreMem(), fakeJobStore)
	runner.SetReplayContextBuilder(replay.NewReplayContextBuilder(trimmed))
	runner.SetNodeEventSink(staleNodeFinishSink{})

	err = runner.RunForJob(ctx, &runtime.Agent{ID: "a1"}, &JobForRunner{ID: jobID, AgentID: "a1", Goal: "g1"})
	if !errors.Is(err, jobstore.ErrStaleAttempt) {
		t.Fatalf("expected ErrStaleAttempt from replay backfill path, got %v", err)
	}
	if atomic.LoadInt32(&callCount) != 0 {
		t.Fatalf("expected replay injection to avoid real LLM execution, got %d calls", callCount)
	}
	_, status := fakeJobStore.getLast()
	const statusFailed = 3
	if status != statusFailed {
		t.Fatalf("UpdateStatus = %d, want %d", status, statusFailed)
	}
	trimmedEvents, _, err := trimmed.ListEvents(ctx, jobID)
	if err != nil {
		t.Fatalf("ListEvents trimmed: %v", err)
	}
	if len(trimmedEvents) != 2 {
		t.Fatalf("expected trimmed event stream to remain unchanged, got %d events", len(trimmedEvents))
	}
}

// TestRunForJob_CheckpointSaveFailure_RetryReplaysWithoutReexecutingTool 证明：
// 即使第一次运行在 node 已完成、事件已落盘后于 checkpoint save 失败，
// 第二次运行在没有 durable cursor 的情况下也必须从事件流恢复完成状态，禁止再次执行 tool。
func TestRunForJob_CheckpointSaveFailure_RetryReplaysWithoutReexecutingTool(t *testing.T) {
	ctx := context.Background()
	jobID := "job-checkpoint-save-failure"
	toolName := "refund"
	cfg := map[string]any{"amount": 42}
	eventStore := jobstore.NewMemoryStore()
	graph := &planner.TaskGraph{
		Nodes: []planner.TaskNode{{ID: "tool1", Type: planner.NodeTool, ToolName: toolName, Config: cfg}},
		Edges: []planner.TaskEdge{},
	}
	graphBytes, err := graph.Marshal()
	if err != nil {
		t.Fatalf("marshal graph: %v", err)
	}
	planPayload, _ := json.Marshal(map[string]any{"task_graph": json.RawMessage(graphBytes), "goal": "g1"})
	if _, err := eventStore.Append(ctx, jobID, 0, jobstore.JobEvent{JobID: jobID, Type: jobstore.PlanGenerated, Payload: planPayload}); err != nil {
		t.Fatalf("append plan_generated: %v", err)
	}

	var callCount int32
	invocationStore := NewToolInvocationStoreMem()
	ledger := NewInvocationLedgerFromStore(invocationStore)
	sink := &jobStoreReplaySink{store: eventStore}
	adapter := &ToolNodeAdapter{
		Tools:            &countToolExec{count: &callCount},
		InvocationStore:  invocationStore,
		InvocationLedger: ledger,
		ToolEventSink:    sink,
		CommandEventSink: sink,
	}
	runner := NewRunner(NewCompiler(map[string]NodeAdapter{planner.NodeTool: adapter}))
	cpStore := &failSaveOnceCheckpointStore{base: runtime.NewCheckpointStoreMem()}
	jobStore := &cursorTrackingJobStore{}
	runner.SetCheckpointStores(cpStore, jobStore)
	runner.SetReplayContextBuilder(replay.NewReplayContextBuilder(eventStore))
	runner.SetNodeEventSink(sink)

	err = runner.RunForJob(ctx, &runtime.Agent{ID: "a1"}, &JobForRunner{ID: jobID, AgentID: "a1", Goal: "g1"})
	if err == nil || !strings.Contains(err.Error(), "save checkpoint failed") {
		t.Fatalf("expected save checkpoint failure, got %v", err)
	}
	if atomic.LoadInt32(&callCount) != 1 {
		t.Fatalf("expected first run to execute tool exactly once, got %d", callCount)
	}
	if cpStore.saveCalls != 1 {
		t.Fatalf("expected one checkpoint save attempt, got %d", cpStore.saveCalls)
	}
	if jobStore.cursorCalls != 0 {
		t.Fatalf("cursor should not update after checkpoint save failure, got %d", jobStore.cursorCalls)
	}
	_, status := jobStore.getLast()
	const statusFailed = 3
	if status != statusFailed {
		t.Fatalf("UpdateStatus = %d, want %d", status, statusFailed)
	}

	retryRunner := NewRunner(NewCompiler(map[string]NodeAdapter{planner.NodeTool: adapter}))
	retryStore := &cursorTrackingJobStore{}
	retryRunner.SetCheckpointStores(runtime.NewCheckpointStoreMem(), retryStore)
	retryRunner.SetReplayContextBuilder(replay.NewReplayContextBuilder(eventStore))
	retryRunner.SetNodeEventSink(sink)
	err = retryRunner.RunForJob(ctx, &runtime.Agent{ID: "a1"}, &JobForRunner{ID: jobID, AgentID: "a1", Goal: "g1"})
	if err != nil {
		t.Fatalf("retry RunForJob: %v", err)
	}
	if atomic.LoadInt32(&callCount) != 1 {
		t.Fatalf("expected retry to replay from events without re-executing tool, got %d calls", callCount)
	}
	_, retryStatus := retryStore.getLast()
	const statusCompleted = 2
	if retryStatus != statusCompleted {
		t.Fatalf("retry UpdateStatus = %d, want %d", retryStatus, statusCompleted)
	}
	if recovered, exists := ledger.Recover(ctx, jobID, IdempotencyKey(jobID, DeterministicStepID(jobID, PlanDecisionID(graphBytes), 0, planner.NodeTool), toolName, cfg)); !exists || len(recovered) == 0 {
		t.Fatalf("expected committed tool result to be recoverable after retry, got exists=%v result=%s", exists, string(recovered))
	}
}

// TestRunForJob_UpdateCursorFailure_RetryReplaysWithoutReexecutingTool 证明：
// 即使 checkpoint 已保存但 UpdateCursor 失败，下一次运行在旧 cursor 下也必须依赖事件流跳过已完成 tool。
func TestRunForJob_UpdateCursorFailure_RetryReplaysWithoutReexecutingTool(t *testing.T) {
	ctx := context.Background()
	jobID := "job-cursor-update-failure"
	toolName := "refund"
	cfg := map[string]any{"amount": 7}
	eventStore := jobstore.NewMemoryStore()
	graph := &planner.TaskGraph{
		Nodes: []planner.TaskNode{{ID: "tool1", Type: planner.NodeTool, ToolName: toolName, Config: cfg}},
		Edges: []planner.TaskEdge{},
	}
	graphBytes, err := graph.Marshal()
	if err != nil {
		t.Fatalf("marshal graph: %v", err)
	}
	planPayload, _ := json.Marshal(map[string]any{"task_graph": json.RawMessage(graphBytes), "goal": "g1"})
	if _, err := eventStore.Append(ctx, jobID, 0, jobstore.JobEvent{JobID: jobID, Type: jobstore.PlanGenerated, Payload: planPayload}); err != nil {
		t.Fatalf("append plan_generated: %v", err)
	}

	var callCount int32
	invocationStore := NewToolInvocationStoreMem()
	ledger := NewInvocationLedgerFromStore(invocationStore)
	sink := &jobStoreReplaySink{store: eventStore}
	adapter := &ToolNodeAdapter{
		Tools:            &countToolExec{count: &callCount},
		InvocationStore:  invocationStore,
		InvocationLedger: ledger,
		ToolEventSink:    sink,
		CommandEventSink: sink,
	}
	runner := NewRunner(NewCompiler(map[string]NodeAdapter{planner.NodeTool: adapter}))
	jobStore := &cursorTrackingJobStore{failCursorOnce: true}
	cpStore := &recordingCheckpointStore{base: runtime.NewCheckpointStoreMem()}
	runner.SetCheckpointStores(cpStore, jobStore)
	runner.SetReplayContextBuilder(replay.NewReplayContextBuilder(eventStore))
	runner.SetNodeEventSink(sink)

	err = runner.RunForJob(ctx, &runtime.Agent{ID: "a1"}, &JobForRunner{ID: jobID, AgentID: "a1", Goal: "g1"})
	if err == nil || !strings.Contains(err.Error(), "update cursor failed") {
		t.Fatalf("expected update cursor failure, got %v", err)
	}
	if atomic.LoadInt32(&callCount) != 1 {
		t.Fatalf("expected first run to execute tool exactly once, got %d", callCount)
	}
	if cpStore.saveCalls != 1 {
		t.Fatalf("expected checkpoint to be saved once before cursor failure, got %d", cpStore.saveCalls)
	}
	if jobStore.cursorCalls != 1 {
		t.Fatalf("expected one cursor update attempt, got %d", jobStore.cursorCalls)
	}
	_, status := jobStore.getLast()
	const statusFailedCursor = 3
	if status != statusFailedCursor {
		t.Fatalf("UpdateStatus = %d, want %d", status, statusFailedCursor)
	}

	retryRunner := NewRunner(NewCompiler(map[string]NodeAdapter{planner.NodeTool: adapter}))
	retryStore := &cursorTrackingJobStore{}
	retryRunner.SetCheckpointStores(runtime.NewCheckpointStoreMem(), retryStore)
	retryRunner.SetReplayContextBuilder(replay.NewReplayContextBuilder(eventStore))
	retryRunner.SetNodeEventSink(sink)
	err = retryRunner.RunForJob(ctx, &runtime.Agent{ID: "a1"}, &JobForRunner{ID: jobID, AgentID: "a1", Goal: "g1"})
	if err != nil {
		t.Fatalf("retry RunForJob: %v", err)
	}
	if atomic.LoadInt32(&callCount) != 1 {
		t.Fatalf("expected retry to skip tool via replay after cursor failure, got %d calls", callCount)
	}
	_, retryStatus := retryStore.getLast()
	const statusCompletedCursor = 2
	if retryStatus != statusCompletedCursor {
		t.Fatalf("retry UpdateStatus = %d, want %d", retryStatus, statusCompletedCursor)
	}
}

// TestRunForJob_PendingWaitRecovery_DoesNotAppendDuplicateJobWaiting 证明：
// 当事件流中已有未解除的 job_waiting 时，恢复必须直接返回 waiting，
// 而不是再次进入 wait 节点并重复追加第二条 job_waiting 事件。
func TestRunForJob_PendingWaitRecovery_DoesNotAppendDuplicateJobWaiting(t *testing.T) {
	ctx := context.Background()
	jobID := "job-pending-wait-recovery"
	eventStore := jobstore.NewMemoryStore()
	graph := &planner.TaskGraph{
		Nodes: []planner.TaskNode{{ID: "wait1", Type: planner.NodeWait, Config: map[string]any{"wait_kind": "signal", "correlation_key": "corr-1", "reason": "need-approval"}}},
		Edges: []planner.TaskEdge{},
	}
	graphBytes, err := graph.Marshal()
	if err != nil {
		t.Fatalf("marshal graph: %v", err)
	}
	planPayload, _ := json.Marshal(map[string]any{"task_graph": json.RawMessage(graphBytes), "goal": "g1"})
	if _, err := eventStore.Append(ctx, jobID, 0, jobstore.JobEvent{JobID: jobID, Type: jobstore.PlanGenerated, Payload: planPayload}); err != nil {
		t.Fatalf("append plan_generated: %v", err)
	}

	sink := &jobStoreReplaySink{store: eventStore}
	runner := NewRunner(NewCompiler(map[string]NodeAdapter{planner.NodeWait: WaitNodeAdapter{}}))
	jobStore := &cursorTrackingJobStore{}
	runner.SetCheckpointStores(runtime.NewCheckpointStoreMem(), jobStore)
	runner.SetReplayContextBuilder(replay.NewReplayContextBuilder(eventStore))
	runner.SetNodeEventSink(sink)

	err = runner.RunForJob(ctx, &runtime.Agent{ID: "a1"}, &JobForRunner{ID: jobID, AgentID: "a1", Goal: "g1"})
	if !errors.Is(err, ErrJobWaiting) {
		t.Fatalf("expected ErrJobWaiting on first run, got %v", err)
	}
	eventsAfterFirst, _, err := eventStore.ListEvents(ctx, jobID)
	if err != nil {
		t.Fatalf("ListEvents after first run: %v", err)
	}
	if count := countEventsByType(eventsAfterFirst, jobstore.JobWaiting); count != 1 {
		t.Fatalf("expected exactly one job_waiting after first run, got %d", count)
	}

	err = runner.RunForJob(ctx, &runtime.Agent{ID: "a1"}, &JobForRunner{ID: jobID, AgentID: "a1", Goal: "g1"})
	if !errors.Is(err, ErrJobWaiting) {
		t.Fatalf("expected ErrJobWaiting on recovery run, got %v", err)
	}
	eventsAfterRetry, _, err := eventStore.ListEvents(ctx, jobID)
	if err != nil {
		t.Fatalf("ListEvents after retry: %v", err)
	}
	if count := countEventsByType(eventsAfterRetry, jobstore.JobWaiting); count != 1 {
		t.Fatalf("expected recovery to avoid duplicate job_waiting, got %d events", count)
	}
}

type staticAttemptValidator struct {
	err error
}

type recordingCheckpointStore struct {
	base      runtime.CheckpointStore
	saveCalls int
}

func (s *recordingCheckpointStore) Save(ctx context.Context, cp *runtime.Checkpoint) (string, error) {
	s.saveCalls++
	return s.base.Save(ctx, cp)
}

func (s *recordingCheckpointStore) Load(ctx context.Context, id string) (*runtime.Checkpoint, error) {
	return s.base.Load(ctx, id)
}

func (s *recordingCheckpointStore) ListByAgent(ctx context.Context, agentID string) ([]*runtime.Checkpoint, error) {
	return s.base.ListByAgent(ctx, agentID)
}

func (s *recordingCheckpointStore) Cleanup(ctx context.Context, olderThan time.Time) (int, error) {
	return s.base.Cleanup(ctx, olderThan)
}

type failSaveOnceCheckpointStore struct {
	base      runtime.CheckpointStore
	saveCalls int
	failed    bool
}

func (s *failSaveOnceCheckpointStore) Save(ctx context.Context, cp *runtime.Checkpoint) (string, error) {
	s.saveCalls++
	if !s.failed {
		s.failed = true
		return "", fmt.Errorf("checkpoint store unavailable")
	}
	return s.base.Save(ctx, cp)
}

func (s *failSaveOnceCheckpointStore) Load(ctx context.Context, id string) (*runtime.Checkpoint, error) {
	return s.base.Load(ctx, id)
}

func (s *failSaveOnceCheckpointStore) ListByAgent(ctx context.Context, agentID string) ([]*runtime.Checkpoint, error) {
	return s.base.ListByAgent(ctx, agentID)
}

func (s *failSaveOnceCheckpointStore) Cleanup(ctx context.Context, olderThan time.Time) (int, error) {
	return s.base.Cleanup(ctx, olderThan)
}

type cursorTrackingJobStore struct {
	fakeJobStoreForRunner
	failCursorOnce bool
	cursorCalls    int
	lastCursor     string
}

func (s *cursorTrackingJobStore) UpdateCursor(ctx context.Context, jobID string, cursor string) error {
	s.cursorCalls++
	s.lastCursor = cursor
	if s.failCursorOnce {
		s.failCursorOnce = false
		return fmt.Errorf("cursor write rejected")
	}
	return nil
}

func (v staticAttemptValidator) ValidateAttempt(ctx context.Context, jobID string) error {
	return v.err
}

type staleNodeFinishSink struct{}

func (staleNodeFinishSink) AppendNodeStarted(ctx context.Context, jobID string, nodeID string, attempt int, workerID string) error {
	return nil
}

func (staleNodeFinishSink) AppendNodeFinished(ctx context.Context, jobID string, nodeID string, payloadResults []byte, durationMs int64, state string, attempt int, resultType StepResultType, reason string, stepID string, inputHash string) error {
	return jobstore.ErrStaleAttempt
}

func (staleNodeFinishSink) AppendStepCommitted(ctx context.Context, jobID string, nodeID string, stepID string, commandID string, idempotencyKey string) error {
	return nil
}

func (staleNodeFinishSink) AppendStateCheckpointed(ctx context.Context, jobID string, nodeID string, stateBefore, stateAfter []byte, opts *StateCheckpointOpts) error {
	return nil
}

func (staleNodeFinishSink) AppendJobWaiting(ctx context.Context, jobID string, nodeID string, waitKind, reason string, expiresAt time.Time, correlationKey string, resumptionContext []byte) error {
	return nil
}

func (staleNodeFinishSink) AppendReasoningSnapshot(ctx context.Context, jobID string, payload []byte) error {
	return nil
}

func (staleNodeFinishSink) AppendStepCompensated(ctx context.Context, jobID string, nodeID string, stepID string, commandID string, reason string) error {
	return nil
}

func (staleNodeFinishSink) AppendMemoryRead(ctx context.Context, jobID string, nodeID string, stepIndex int, memoryType, keyOrScope, summary string) error {
	return nil
}

func (staleNodeFinishSink) AppendMemoryWrite(ctx context.Context, jobID string, nodeID string, stepIndex int, memoryType, keyOrScope, summary string) error {
	return nil
}

func (staleNodeFinishSink) AppendPlanEvolution(ctx context.Context, jobID string, planVersion int, diffSummary string) error {
	return nil
}

type jobStoreReplaySink struct {
	store jobstore.JobStore
}

func (s *jobStoreReplaySink) append(ctx context.Context, jobID string, eventType jobstore.EventType, payload []byte) error {
	if s.store == nil {
		return nil
	}
	_, ver, err := s.store.ListEvents(ctx, jobID)
	if err != nil {
		return err
	}
	_, err = s.store.Append(ctx, jobID, ver, jobstore.JobEvent{JobID: jobID, Type: eventType, Payload: payload})
	return err
}

func (s *jobStoreReplaySink) AppendNodeStarted(ctx context.Context, jobID string, nodeID string, attempt int, workerID string) error {
	payload, _ := json.Marshal(map[string]any{"node_id": nodeID})
	return s.append(ctx, jobID, jobstore.NodeStarted, payload)
}

func (s *jobStoreReplaySink) AppendNodeFinished(ctx context.Context, jobID string, nodeID string, payloadResults []byte, durationMs int64, state string, attempt int, resultType StepResultType, reason string, stepID string, inputHash string) error {
	if stepID == "" {
		stepID = nodeID
	}
	payload, _ := json.Marshal(map[string]any{
		"node_id":         nodeID,
		"step_id":         stepID,
		"payload_results": json.RawMessage(payloadResults),
		"result_type":     string(resultType),
	})
	return s.append(ctx, jobID, jobstore.NodeFinished, payload)
}

func (s *jobStoreReplaySink) AppendStepCommitted(ctx context.Context, jobID string, nodeID string, stepID string, commandID string, idempotencyKey string) error {
	if stepID == "" {
		stepID = nodeID
	}
	if commandID == "" {
		commandID = stepID
	}
	payload, _ := json.Marshal(map[string]any{"node_id": nodeID, "step_id": stepID, "command_id": commandID, "idempotency_key": idempotencyKey})
	return s.append(ctx, jobID, jobstore.StepCommitted, payload)
}

func (s *jobStoreReplaySink) AppendStateCheckpointed(ctx context.Context, jobID string, nodeID string, stateBefore, stateAfter []byte, opts *StateCheckpointOpts) error {
	return nil
}

func (s *jobStoreReplaySink) AppendJobWaiting(ctx context.Context, jobID string, nodeID string, waitKind, reason string, expiresAt time.Time, correlationKey string, resumptionContext []byte) error {
	b, _ := json.Marshal(map[string]any{
		"node_id":            nodeID,
		"wait_type":          waitKind,
		"correlation_key":    correlationKey,
		"wait_kind":          waitKind,
		"reason":             reason,
		"expires_at":         expiresAt.UTC().Format(time.RFC3339),
		"resumption_context": json.RawMessage(resumptionContext),
	})
	return s.append(ctx, jobID, jobstore.JobWaiting, b)
}

func (s *jobStoreReplaySink) AppendReasoningSnapshot(ctx context.Context, jobID string, payload []byte) error {
	return nil
}

func (s *jobStoreReplaySink) AppendStepCompensated(ctx context.Context, jobID string, nodeID string, stepID string, commandID string, reason string) error {
	return nil
}

func (s *jobStoreReplaySink) AppendMemoryRead(ctx context.Context, jobID string, nodeID string, stepIndex int, memoryType, keyOrScope, summary string) error {
	return nil
}

func (s *jobStoreReplaySink) AppendMemoryWrite(ctx context.Context, jobID string, nodeID string, stepIndex int, memoryType, keyOrScope, summary string) error {
	return nil
}

func (s *jobStoreReplaySink) AppendPlanEvolution(ctx context.Context, jobID string, planVersion int, diffSummary string) error {
	return nil
}

func (s *jobStoreReplaySink) AppendToolCalled(ctx context.Context, jobID string, nodeID string, toolName string, input []byte) error {
	return nil
}

func (s *jobStoreReplaySink) AppendToolReturned(ctx context.Context, jobID string, nodeID string, output []byte) error {
	return nil
}

func (s *jobStoreReplaySink) AppendToolResultSummarized(ctx context.Context, jobID string, nodeID string, toolName string, summary string, errMsg string, idempotent bool) error {
	return nil
}

func (s *jobStoreReplaySink) AppendToolInvocationStarted(ctx context.Context, jobID string, nodeID string, payload *ToolInvocationStartedPayload) error {
	if payload == nil {
		return nil
	}
	b, _ := json.Marshal(map[string]any{
		"node_id":         nodeID,
		"invocation_id":   payload.InvocationID,
		"tool_name":       payload.ToolName,
		"idempotency_key": payload.IdempotencyKey,
		"arguments_hash":  payload.ArgumentsHash,
		"started_at":      payload.StartedAt,
	})
	return s.append(ctx, jobID, jobstore.ToolInvocationStarted, b)
}

func (s *jobStoreReplaySink) AppendToolInvocationFinished(ctx context.Context, jobID string, nodeID string, payload *ToolInvocationFinishedPayload) error {
	if payload == nil {
		return nil
	}
	b, _ := json.Marshal(map[string]any{
		"node_id":         nodeID,
		"invocation_id":   payload.InvocationID,
		"idempotency_key": payload.IdempotencyKey,
		"outcome":         payload.Outcome,
		"result":          json.RawMessage(payload.Result),
		"error":           payload.Error,
		"finished_at":     payload.FinishedAt,
	})
	return s.append(ctx, jobID, jobstore.ToolInvocationFinished, b)
}

func (s *jobStoreReplaySink) AppendCommandEmitted(ctx context.Context, jobID string, nodeID string, commandID string, kind string, input []byte) error {
	return nil
}

func (s *jobStoreReplaySink) AppendCommandCommitted(ctx context.Context, jobID string, nodeID string, commandID string, result []byte, inputHash string) error {
	b, _ := json.Marshal(map[string]any{
		"node_id":    nodeID,
		"command_id": commandID,
		"result":     json.RawMessage(result),
		"input_hash": inputHash,
	})
	return s.append(ctx, jobID, jobstore.CommandCommitted, b)
}

func countEventsByType(events []jobstore.JobEvent, eventType jobstore.EventType) int {
	count := 0
	for _, event := range events {
		if event.Type == eventType {
			count++
		}
	}
	return count
}
