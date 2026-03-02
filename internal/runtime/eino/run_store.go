package eino

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
)

var (
	// ErrRunNotFound 表示运行实例不存在。
	ErrRunNotFound = fmt.Errorf("run not found")
	// ErrRunConflict 表示运行状态冲突。
	ErrRunConflict = fmt.Errorf("run status conflict")
	// ErrInvalidRunArg 表示请求参数不合法。
	ErrInvalidRunArg = fmt.Errorf("invalid run argument")
	// ErrToolCallNotFound 表示 tool call 不存在或不属于该运行实例。
	ErrToolCallNotFound = fmt.Errorf("tool call not found")
)

// RuntimeRunStore 运行实例存储接口。
type RuntimeRunStore interface {
	CreateRun(ctx context.Context, run *Run) (*Run, error)
	GetRun(ctx context.Context, runID string) (*Run, error)
	ListEvents(ctx context.Context, runID string, cursor int64, limit int) ([]RuntimeEvent, int64, error)
	UpsertToolCall(ctx context.Context, call *ToolCall) (*ToolCall, error)
	PauseRun(ctx context.Context, runID, reason, operator string) (*Run, error)
	ResumeRun(ctx context.Context, runID string, req ResumeRunRequest) (*Run, error)
	InjectHumanDecision(ctx context.Context, runID string, decision HumanDecision) (*RuntimeEvent, error)
}

// NewMemoryRunStore 创建内存版运行实例存储。
func NewMemoryRunStore() RuntimeRunStore {
	return &memoryRunStore{
		runs:       make(map[string]*Run),
		events:     make(map[string][]RuntimeEvent),
		eventSeq:   make(map[string]int64),
		resumeReqs: make(map[string]map[string]struct{}),
		toolCalls:  make(map[string]map[string]*ToolCall),
	}
}

type memoryRunStore struct {
	mu sync.RWMutex

	runs       map[string]*Run
	events     map[string][]RuntimeEvent
	eventSeq   map[string]int64
	resumeReqs map[string]map[string]struct{}
	toolCalls  map[string]map[string]*ToolCall
}

func (s *memoryRunStore) CreateRun(ctx context.Context, run *Run) (*Run, error) {
	_ = ctx
	if run == nil || run.WorkflowID == "" {
		return nil, ErrInvalidRunArg
	}
	now := time.Now()
	copied := *run
	if copied.ID == "" {
		copied.ID = "run-" + uuid.NewString()
	}
	if copied.Status == "" {
		copied.Status = RunStatusPending
	}
	copied.CreatedAt = now
	copied.UpdatedAt = now

	s.mu.Lock()
	defer s.mu.Unlock()

	s.runs[copied.ID] = cloneRun(&copied)
	s.appendEventLocked(copied.ID, EventTypeRunCreated, "system", map[string]interface{}{
		"workflow_id": copied.WorkflowID,
	})
	return cloneRun(&copied), nil
}

func (s *memoryRunStore) GetRun(ctx context.Context, runID string) (*Run, error) {
	_ = ctx
	s.mu.RLock()
	defer s.mu.RUnlock()
	r, ok := s.runs[runID]
	if !ok {
		return nil, ErrRunNotFound
	}
	return cloneRun(r), nil
}

func (s *memoryRunStore) ListEvents(ctx context.Context, runID string, cursor int64, limit int) ([]RuntimeEvent, int64, error) {
	_ = ctx
	if limit <= 0 {
		limit = 200
	}

	s.mu.RLock()
	defer s.mu.RUnlock()
	if _, ok := s.runs[runID]; !ok {
		return nil, 0, ErrRunNotFound
	}
	all := s.events[runID]
	out := make([]RuntimeEvent, 0, limit)
	nextCursor := cursor
	for _, ev := range all {
		if ev.Seq <= cursor {
			continue
		}
		out = append(out, cloneEvent(ev))
		nextCursor = ev.Seq
		if len(out) >= limit {
			break
		}
	}
	return out, nextCursor, nil
}

func (s *memoryRunStore) PauseRun(ctx context.Context, runID, reason, operator string) (*Run, error) {
	_ = ctx
	s.mu.Lock()
	defer s.mu.Unlock()
	r, ok := s.runs[runID]
	if !ok {
		return nil, ErrRunNotFound
	}
	if r.Status != RunStatusRunning && r.Status != RunStatusPending {
		return nil, ErrRunConflict
	}
	r.Status = RunStatusPaused
	r.UpdatedAt = time.Now()
	s.appendEventLocked(runID, EventTypeRunPaused, actorOrSystem(operator), map[string]interface{}{
		"reason": reason,
	})
	return cloneRun(r), nil
}

func (s *memoryRunStore) UpsertToolCall(ctx context.Context, call *ToolCall) (*ToolCall, error) {
	_ = ctx
	if call == nil || call.RunID == "" || call.ID == "" {
		return nil, ErrInvalidRunArg
	}
	now := time.Now()
	copied := *call
	if copied.StartedAt == nil {
		copied.StartedAt = &now
	}
	if copied.Status == "" {
		copied.Status = "PENDING"
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.runs[copied.RunID]; !ok {
		return nil, ErrRunNotFound
	}
	if _, ok := s.toolCalls[copied.RunID]; !ok {
		s.toolCalls[copied.RunID] = make(map[string]*ToolCall)
	}
	prev, hasPrev := s.toolCalls[copied.RunID][copied.ID]
	s.toolCalls[copied.RunID][copied.ID] = cloneToolCall(&copied)
	eventType := EventTypeToolCallStarted
	if isTerminalToolCallStatus(copied.Status) {
		eventType = EventTypeToolCallEnded
	}
	if !hasPrev || prev.Status != copied.Status || prev.ToolName != copied.ToolName {
		s.appendEventLocked(copied.RunID, eventType, "system", map[string]interface{}{
			"tool_call_id": copied.ID,
			"tool_name":    copied.ToolName,
			"status":       copied.Status,
		})
	}
	return cloneToolCall(&copied), nil
}

func (s *memoryRunStore) ResumeRun(ctx context.Context, runID string, req ResumeRunRequest) (*Run, error) {
	_ = ctx
	if req.Mode != ResumeModeFromToolCall || req.FromToolCallID == "" || req.Strategy == "" {
		return nil, ErrInvalidRunArg
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	r, ok := s.runs[runID]
	if !ok {
		return nil, ErrRunNotFound
	}
	if r.Status != RunStatusPaused && r.Status != RunStatusFailed {
		return nil, ErrRunConflict
	}
	if _, ok := s.toolCalls[runID][req.FromToolCallID]; !ok {
		return nil, ErrToolCallNotFound
	}
	if req.ResumeRequestID != "" {
		if _, ok := s.resumeReqs[runID]; !ok {
			s.resumeReqs[runID] = make(map[string]struct{})
		}
		if _, exists := s.resumeReqs[runID][req.ResumeRequestID]; exists {
			return cloneRun(r), nil
		}
		s.resumeReqs[runID][req.ResumeRequestID] = struct{}{}
	}

	r.Status = RunStatusRunning
	r.UpdatedAt = time.Now()
	s.appendEventLocked(runID, EventTypeRunResumed, actorOrSystem(req.Operator), map[string]interface{}{
		"mode":              req.Mode,
		"from_tool_call_id": req.FromToolCallID,
		"strategy":          req.Strategy,
		"reason":            req.Reason,
	})
	return cloneRun(r), nil
}

func (s *memoryRunStore) InjectHumanDecision(ctx context.Context, runID string, decision HumanDecision) (*RuntimeEvent, error) {
	_ = ctx
	if decision.TargetStepID == "" || decision.Operator == "" {
		return nil, ErrInvalidRunArg
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	r, ok := s.runs[runID]
	if !ok {
		return nil, ErrRunNotFound
	}
	r.UpdatedAt = time.Now()
	ev := s.appendEventLocked(runID, EventTypeHumanInjected, decision.Operator, map[string]interface{}{
		"target_step_id": decision.TargetStepID,
		"patch":          cloneMap(decision.Patch),
		"comment":        decision.Comment,
	})
	cloned := cloneEvent(ev)
	return &cloned, nil
}

func (s *memoryRunStore) appendEventLocked(runID string, eventType EventType, actor string, payload map[string]interface{}) RuntimeEvent {
	s.eventSeq[runID]++
	ev := RuntimeEvent{
		ID:         "ev-" + uuid.NewString(),
		RunID:      runID,
		Type:       eventType,
		Seq:        s.eventSeq[runID],
		Actor:      actor,
		Payload:    cloneMap(payload),
		OccurredAt: time.Now(),
	}
	s.events[runID] = append(s.events[runID], ev)
	return ev
}

func actorOrSystem(actor string) string {
	if actor == "" {
		return "system"
	}
	return actor
}

func cloneRun(in *Run) *Run {
	if in == nil {
		return nil
	}
	out := *in
	out.Input = cloneMap(in.Input)
	out.Output = cloneMap(in.Output)
	return &out
}

func cloneToolCall(in *ToolCall) *ToolCall {
	if in == nil {
		return nil
	}
	out := *in
	out.RequestPayload = cloneMap(in.RequestPayload)
	out.ResponsePayload = cloneMap(in.ResponsePayload)
	return &out
}

func cloneMap(in map[string]interface{}) map[string]interface{} {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]interface{}, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

func cloneEvent(in RuntimeEvent) RuntimeEvent {
	in.Payload = cloneMap(in.Payload)
	return in
}

func isTerminalToolCallStatus(status string) bool {
	switch status {
	case "SUCCEEDED", "FAILED", "TIMEOUT", "CANCELED":
		return true
	default:
		return false
	}
}
