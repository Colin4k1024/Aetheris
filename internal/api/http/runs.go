package http

import (
	"context"
	"errors"
	"strconv"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/common/hlog"
	"github.com/cloudwego/hertz/pkg/protocol/consts"

	"rag-platform/internal/runtime/eino"
)

type createRunRequest struct {
	WorkflowID     string                 `json:"workflow_id" binding:"required"`
	Input          map[string]interface{} `json:"input"`
	Budget         eino.BudgetPolicy      `json:"budget"`
	IdempotencyKey string                 `json:"idempotency_key"`
}

type pauseRunRequest struct {
	Reason   string `json:"reason"`
	Operator string `json:"operator"`
}

type resumeRunRequest struct {
	Mode            eino.ResumeMode     `json:"mode" binding:"required"`
	FromToolCallID  string              `json:"from_tool_call_id" binding:"required"`
	Strategy        eino.ResumeStrategy `json:"strategy" binding:"required"`
	Operator        string              `json:"operator"`
	Reason          string              `json:"reason"`
	ResumeRequestID string              `json:"resume_request_id,omitempty"`
}

type humanDecisionRequest struct {
	TargetStepID string                 `json:"target_step_id" binding:"required"`
	Patch        map[string]interface{} `json:"patch"`
	Operator     string                 `json:"operator" binding:"required"`
	Comment      string                 `json:"comment"`
}

type upsertToolCallRequest struct {
	ID             string                 `json:"id" binding:"required"`
	StepID         string                 `json:"step_id"`
	ToolName       string                 `json:"tool_name" binding:"required"`
	Status         string                 `json:"status"`
	RequestPayload map[string]interface{} `json:"request_payload"`
	IdempotencyKey string                 `json:"idempotency_key"`
	SideEffectSafe bool                   `json:"side_effect_safe"`
}

// CreateRun 创建运行实例。
func (h *Handler) CreateRun(ctx context.Context, c *app.RequestContext) {
	if h.runStore == nil {
		c.JSON(consts.StatusServiceUnavailable, map[string]string{"error": "RunStore 未启用"})
		return
	}
	var req createRunRequest
	if err := c.BindJSON(&req); err != nil {
		c.JSON(consts.StatusBadRequest, map[string]string{"error": "请求参数错误"})
		return
	}

	run, err := h.runStore.CreateRun(ctx, &eino.Run{
		WorkflowID:     req.WorkflowID,
		Input:          req.Input,
		Budget:         req.Budget,
		IdempotencyKey: req.IdempotencyKey,
	})
	if err != nil {
		hlog.CtxErrorf(ctx, "create run failed: %v", err)
		status, msg := mapRunStoreError(err)
		c.JSON(status, map[string]string{"error": msg})
		return
	}
	c.JSON(consts.StatusAccepted, run)
}

// GetRun 查询运行详情。
func (h *Handler) GetRun(ctx context.Context, c *app.RequestContext) {
	if h.runStore == nil {
		c.JSON(consts.StatusServiceUnavailable, map[string]string{"error": "RunStore 未启用"})
		return
	}
	runID := c.Param("id")
	run, err := h.runStore.GetRun(ctx, runID)
	if err != nil {
		status, msg := mapRunStoreError(err)
		c.JSON(status, map[string]string{"error": msg})
		return
	}
	c.JSON(consts.StatusOK, run)
}

// GetRunEvents 查询运行事件流。
func (h *Handler) GetRunEvents(ctx context.Context, c *app.RequestContext) {
	if h.runStore == nil {
		c.JSON(consts.StatusServiceUnavailable, map[string]string{"error": "RunStore 未启用"})
		return
	}
	runID := c.Param("id")
	cursor, err := parseInt64Query(c, "cursor", 0)
	if err != nil {
		c.JSON(consts.StatusBadRequest, map[string]string{"error": "cursor 参数错误"})
		return
	}
	limit, err := parseIntQuery(c, "limit", 200)
	if err != nil {
		c.JSON(consts.StatusBadRequest, map[string]string{"error": "limit 参数错误"})
		return
	}

	events, nextCursor, listErr := h.runStore.ListEvents(ctx, runID, cursor, limit)
	if listErr != nil {
		status, msg := mapRunStoreError(listErr)
		c.JSON(status, map[string]string{"error": msg})
		return
	}
	c.JSON(consts.StatusOK, map[string]interface{}{
		"events":      events,
		"next_cursor": nextCursor,
	})
}

// PauseRun 暂停运行。
func (h *Handler) PauseRun(ctx context.Context, c *app.RequestContext) {
	if h.runStore == nil {
		c.JSON(consts.StatusServiceUnavailable, map[string]string{"error": "RunStore 未启用"})
		return
	}
	runID := c.Param("id")
	var req pauseRunRequest
	if len(c.Request.Body()) > 0 {
		if err := c.BindJSON(&req); err != nil {
			c.JSON(consts.StatusBadRequest, map[string]string{"error": "请求参数错误"})
			return
		}
	}
	run, err := h.runStore.PauseRun(ctx, runID, req.Reason, req.Operator)
	if err != nil {
		status, msg := mapRunStoreError(err)
		c.JSON(status, map[string]string{"error": msg})
		return
	}
	c.JSON(consts.StatusOK, run)
}

// ResumeRun 从 tool call 恢复运行。
func (h *Handler) ResumeRun(ctx context.Context, c *app.RequestContext) {
	if h.runStore == nil {
		c.JSON(consts.StatusServiceUnavailable, map[string]string{"error": "RunStore 未启用"})
		return
	}
	runID := c.Param("id")
	var req resumeRunRequest
	if err := c.BindJSON(&req); err != nil {
		c.JSON(consts.StatusBadRequest, map[string]string{"error": "请求参数错误"})
		return
	}

	run, err := h.runStore.ResumeRun(ctx, runID, eino.ResumeRunRequest{
		Mode:            req.Mode,
		FromToolCallID:  req.FromToolCallID,
		Strategy:        req.Strategy,
		Operator:        req.Operator,
		Reason:          req.Reason,
		ResumeRequestID: req.ResumeRequestID,
	})
	if err != nil {
		status, msg := mapRunStoreError(err)
		c.JSON(status, map[string]string{"error": msg})
		return
	}
	c.JSON(consts.StatusOK, run)
}

// InjectHumanDecision 注入人工决策。
func (h *Handler) InjectHumanDecision(ctx context.Context, c *app.RequestContext) {
	if h.runStore == nil {
		c.JSON(consts.StatusServiceUnavailable, map[string]string{"error": "RunStore 未启用"})
		return
	}
	runID := c.Param("id")
	var req humanDecisionRequest
	if err := c.BindJSON(&req); err != nil {
		c.JSON(consts.StatusBadRequest, map[string]string{"error": "请求参数错误"})
		return
	}

	ev, err := h.runStore.InjectHumanDecision(ctx, runID, eino.HumanDecision{
		TargetStepID: req.TargetStepID,
		Patch:        req.Patch,
		Operator:     req.Operator,
		Comment:      req.Comment,
	})
	if err != nil {
		status, msg := mapRunStoreError(err)
		c.JSON(status, map[string]string{"error": msg})
		return
	}
	c.JSON(consts.StatusAccepted, ev)
}

// UpsertToolCall 记录或更新运行内的工具调用（供 resume 校验归属）。
func (h *Handler) UpsertToolCall(ctx context.Context, c *app.RequestContext) {
	if h.runStore == nil {
		c.JSON(consts.StatusServiceUnavailable, map[string]string{"error": "RunStore 未启用"})
		return
	}
	runID := c.Param("id")
	var req upsertToolCallRequest
	if err := c.BindJSON(&req); err != nil {
		c.JSON(consts.StatusBadRequest, map[string]string{"error": "请求参数错误"})
		return
	}

	call, err := h.runStore.UpsertToolCall(ctx, &eino.ToolCall{
		ID:             req.ID,
		RunID:          runID,
		StepID:         req.StepID,
		ToolName:       req.ToolName,
		Status:         req.Status,
		RequestPayload: req.RequestPayload,
		IdempotencyKey: req.IdempotencyKey,
		SideEffectSafe: req.SideEffectSafe,
	})
	if err != nil {
		status, msg := mapRunStoreError(err)
		c.JSON(status, map[string]string{"error": msg})
		return
	}
	c.JSON(consts.StatusAccepted, call)
}

func parseInt64Query(c *app.RequestContext, key string, defaultValue int64) (int64, error) {
	v := string(c.Query(key))
	if v == "" {
		return defaultValue, nil
	}
	return strconv.ParseInt(v, 10, 64)
}

func parseIntQuery(c *app.RequestContext, key string, defaultValue int) (int, error) {
	v := string(c.Query(key))
	if v == "" {
		return defaultValue, nil
	}
	parsed, err := strconv.Atoi(v)
	if err != nil {
		return 0, err
	}
	if parsed <= 0 {
		return 0, errors.New("must be > 0")
	}
	return parsed, nil
}

func mapRunStoreError(err error) (int, string) {
	switch {
	case errors.Is(err, eino.ErrRunNotFound):
		return consts.StatusNotFound, "运行实例不存在"
	case errors.Is(err, eino.ErrInvalidRunArg):
		return consts.StatusBadRequest, "请求参数错误"
	case errors.Is(err, eino.ErrRunConflict):
		return consts.StatusConflict, "运行状态冲突"
	case errors.Is(err, eino.ErrToolCallNotFound):
		return consts.StatusBadRequest, "tool_call_id 不存在或不属于当前运行"
	default:
		return consts.StatusInternalServerError, "运行操作失败"
	}
}
