package http

import (
	"context"
	"strings"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/protocol/consts"
)

func (h *Handler) RuntimeToolInvoke(ctx context.Context, c *app.RequestContext) {
	if h.runtimeBridge == nil {
		c.JSON(consts.StatusServiceUnavailable, map[string]string{"error": "runtime bridge not configured"})
		return
	}
	jobID := strings.TrimSpace(c.Param("id"))
	toolName := strings.TrimSpace(c.Param("name"))
	var req RuntimeBridgeToolRequest
	if err := c.BindJSON(&req); err != nil {
		c.JSON(consts.StatusBadRequest, map[string]string{"error": "请求参数错误"})
		return
	}
	req.JobID = jobID
	req.ToolName = toolName
	if strings.TrimSpace(req.NodeID) == "" {
		c.JSON(consts.StatusBadRequest, map[string]string{"error": "node_id is required"})
		return
	}
	result, err := h.runtimeBridge.InvokeTool(ctx, req)
	if err != nil {
		c.JSON(consts.StatusBadGateway, map[string]string{"error": err.Error()})
		return
	}
	c.JSON(consts.StatusOK, result)
}

func (h *Handler) RuntimeLLMInvoke(ctx context.Context, c *app.RequestContext) {
	if h.runtimeBridge == nil {
		c.JSON(consts.StatusServiceUnavailable, map[string]string{"error": "runtime bridge not configured"})
		return
	}
	jobID := strings.TrimSpace(c.Param("id"))
	var req RuntimeBridgeLLMRequest
	if err := c.BindJSON(&req); err != nil {
		c.JSON(consts.StatusBadRequest, map[string]string{"error": "请求参数错误"})
		return
	}
	req.JobID = jobID
	if strings.TrimSpace(req.NodeID) == "" {
		c.JSON(consts.StatusBadRequest, map[string]string{"error": "node_id is required"})
		return
	}
	result, err := h.runtimeBridge.InvokeLLM(ctx, req)
	if err != nil {
		c.JSON(consts.StatusBadGateway, map[string]string{"error": err.Error()})
		return
	}
	c.JSON(consts.StatusOK, result)
}
