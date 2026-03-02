package http

import (
	"bytes"
	"context"
	"encoding/json"
	"testing"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/app/server"
	"github.com/cloudwego/hertz/pkg/common/ut"
)

func TestRuns_CreateAndGet(t *testing.T) {
	handler := NewHandler(nil, nil)
	h := server.Default(server.WithHostPorts(":0"))
	h.POST("/api/runs", func(ctx context.Context, c *app.RequestContext) {
		handler.CreateRun(ctx, c)
	})
	h.GET("/api/runs/:id", func(ctx context.Context, c *app.RequestContext) {
		handler.GetRun(ctx, c)
	})

	body := []byte(`{"workflow_id":"wf_123","input":{"query":"hello"}}`)
	w := ut.PerformRequest(h.Engine, "POST", "/api/runs", &ut.Body{Body: bytes.NewReader(body), Len: len(body)})
	if got := w.Result().StatusCode(); got != 202 {
		t.Fatalf("create run status=%d, want 202 body=%s", got, w.Result().Body())
	}

	var created map[string]interface{}
	if err := json.Unmarshal(w.Result().Body(), &created); err != nil {
		t.Fatalf("unmarshal create response: %v", err)
	}
	runID, _ := created["id"].(string)
	if runID == "" {
		t.Fatalf("create response missing id: %s", w.Result().Body())
	}

	wGet := ut.PerformRequest(h.Engine, "GET", "/api/runs/"+runID, &ut.Body{Body: bytes.NewReader(nil), Len: 0})
	if got := wGet.Result().StatusCode(); got != 200 {
		t.Fatalf("get run status=%d, want 200 body=%s", got, wGet.Result().Body())
	}
	if !bytes.Contains(wGet.Result().Body(), []byte(`"workflow_id":"wf_123"`)) {
		t.Fatalf("get run body unexpected: %s", wGet.Result().Body())
	}
}

func TestRuns_PauseResumeAndEvents(t *testing.T) {
	handler := NewHandler(nil, nil)
	h := server.Default(server.WithHostPorts(":0"))
	h.POST("/api/runs", func(ctx context.Context, c *app.RequestContext) {
		handler.CreateRun(ctx, c)
	})
	h.POST("/api/runs/:id/pause", func(ctx context.Context, c *app.RequestContext) {
		handler.PauseRun(ctx, c)
	})
	h.POST("/api/runs/:id/resume", func(ctx context.Context, c *app.RequestContext) {
		handler.ResumeRun(ctx, c)
	})
	h.POST("/api/runs/:id/tool-calls", func(ctx context.Context, c *app.RequestContext) {
		handler.UpsertToolCall(ctx, c)
	})
	h.GET("/api/runs/:id/events", func(ctx context.Context, c *app.RequestContext) {
		handler.GetRunEvents(ctx, c)
	})

	createBody := []byte(`{"workflow_id":"wf_001"}`)
	created := ut.PerformRequest(h.Engine, "POST", "/api/runs", &ut.Body{Body: bytes.NewReader(createBody), Len: len(createBody)})
	if got := created.Result().StatusCode(); got != 202 {
		t.Fatalf("create run status=%d, want 202 body=%s", got, created.Result().Body())
	}

	var createdResp map[string]interface{}
	if err := json.Unmarshal(created.Result().Body(), &createdResp); err != nil {
		t.Fatalf("unmarshal create response: %v", err)
	}
	runID, _ := createdResp["id"].(string)

	pauseBody := []byte(`{"reason":"manual_check","operator":"alice"}`)
	paused := ut.PerformRequest(h.Engine, "POST", "/api/runs/"+runID+"/pause", &ut.Body{Body: bytes.NewReader(pauseBody), Len: len(pauseBody)})
	if got := paused.Result().StatusCode(); got != 200 {
		t.Fatalf("pause status=%d, want 200 body=%s", got, paused.Result().Body())
	}

	resumeBodyMissingTool := []byte(`{"mode":"FROM_TOOL_CALL","from_tool_call_id":"tc_missing","strategy":"REUSE_SUCCESSFUL_EFFECTS","operator":"alice"}`)
	resumeMissingTool := ut.PerformRequest(h.Engine, "POST", "/api/runs/"+runID+"/resume", &ut.Body{Body: bytes.NewReader(resumeBodyMissingTool), Len: len(resumeBodyMissingTool)})
	if got := resumeMissingTool.Result().StatusCode(); got != 400 {
		t.Fatalf("resume without tool call status=%d, want 400 body=%s", got, resumeMissingTool.Result().Body())
	}

	toolCallBody := []byte(`{"id":"tc_1","step_id":"st_1","tool_name":"search","status":"SUCCEEDED","request_payload":{"query":"hello"}}`)
	toolCallResp := ut.PerformRequest(h.Engine, "POST", "/api/runs/"+runID+"/tool-calls", &ut.Body{Body: bytes.NewReader(toolCallBody), Len: len(toolCallBody)})
	if got := toolCallResp.Result().StatusCode(); got != 202 {
		t.Fatalf("upsert tool call status=%d, want 202 body=%s", got, toolCallResp.Result().Body())
	}

	resumeBody := []byte(`{"mode":"FROM_TOOL_CALL","from_tool_call_id":"tc_1","strategy":"REUSE_SUCCESSFUL_EFFECTS","operator":"alice"}`)
	resumed := ut.PerformRequest(h.Engine, "POST", "/api/runs/"+runID+"/resume", &ut.Body{Body: bytes.NewReader(resumeBody), Len: len(resumeBody)})
	if got := resumed.Result().StatusCode(); got != 200 {
		t.Fatalf("resume status=%d, want 200 body=%s", got, resumed.Result().Body())
	}

	events := ut.PerformRequest(h.Engine, "GET", "/api/runs/"+runID+"/events?cursor=0&limit=2", &ut.Body{Body: bytes.NewReader(nil), Len: 0})
	if got := events.Result().StatusCode(); got != 200 {
		t.Fatalf("events status=%d, want 200 body=%s", got, events.Result().Body())
	}
	if !bytes.Contains(events.Result().Body(), []byte(`"events"`)) {
		t.Fatalf("events body unexpected: %s", events.Result().Body())
	}

	invalidLimit := ut.PerformRequest(h.Engine, "GET", "/api/runs/"+runID+"/events?limit=0", &ut.Body{Body: bytes.NewReader(nil), Len: 0})
	if got := invalidLimit.Result().StatusCode(); got != 400 {
		t.Fatalf("events invalid limit status=%d, want 400 body=%s", got, invalidLimit.Result().Body())
	}
}

func TestRuns_InjectHumanDecision(t *testing.T) {
	handler := NewHandler(nil, nil)
	h := server.Default(server.WithHostPorts(":0"))
	h.POST("/api/runs", func(ctx context.Context, c *app.RequestContext) {
		handler.CreateRun(ctx, c)
	})
	h.POST("/api/runs/:id/human-decisions", func(ctx context.Context, c *app.RequestContext) {
		handler.InjectHumanDecision(ctx, c)
	})

	createBody := []byte(`{"workflow_id":"wf_human"}`)
	created := ut.PerformRequest(h.Engine, "POST", "/api/runs", &ut.Body{Body: bytes.NewReader(createBody), Len: len(createBody)})
	if got := created.Result().StatusCode(); got != 202 {
		t.Fatalf("create run status=%d, want 202 body=%s", got, created.Result().Body())
	}

	var createdResp map[string]interface{}
	if err := json.Unmarshal(created.Result().Body(), &createdResp); err != nil {
		t.Fatalf("unmarshal create response: %v", err)
	}
	runID, _ := createdResp["id"].(string)

	decisionBody := []byte(`{"target_step_id":"st_1","patch":{"approve":true},"operator":"alice","comment":"ok"}`)
	w := ut.PerformRequest(h.Engine, "POST", "/api/runs/"+runID+"/human-decisions", &ut.Body{Body: bytes.NewReader(decisionBody), Len: len(decisionBody)})
	if got := w.Result().StatusCode(); got != 202 {
		t.Fatalf("inject decision status=%d, want 202 body=%s", got, w.Result().Body())
	}
	if !bytes.Contains(w.Result().Body(), []byte(`"type":"HUMAN_INJECTED"`)) {
		t.Fatalf("inject decision body unexpected: %s", w.Result().Body())
	}
}

func TestRuns_ResumeRejectsToolCallFromOtherRun(t *testing.T) {
	handler := NewHandler(nil, nil)
	h := server.Default(server.WithHostPorts(":0"))
	h.POST("/api/runs", func(ctx context.Context, c *app.RequestContext) {
		handler.CreateRun(ctx, c)
	})
	h.POST("/api/runs/:id/pause", func(ctx context.Context, c *app.RequestContext) {
		handler.PauseRun(ctx, c)
	})
	h.POST("/api/runs/:id/tool-calls", func(ctx context.Context, c *app.RequestContext) {
		handler.UpsertToolCall(ctx, c)
	})
	h.POST("/api/runs/:id/resume", func(ctx context.Context, c *app.RequestContext) {
		handler.ResumeRun(ctx, c)
	})

	createBody := []byte(`{"workflow_id":"wf_isolation"}`)
	createRun := func() string {
		created := ut.PerformRequest(h.Engine, "POST", "/api/runs", &ut.Body{Body: bytes.NewReader(createBody), Len: len(createBody)})
		if got := created.Result().StatusCode(); got != 202 {
			t.Fatalf("create run status=%d, want 202 body=%s", got, created.Result().Body())
		}
		var resp map[string]interface{}
		if err := json.Unmarshal(created.Result().Body(), &resp); err != nil {
			t.Fatalf("unmarshal create response: %v", err)
		}
		id, _ := resp["id"].(string)
		return id
	}

	runA := createRun()
	runB := createRun()

	pauseBody := []byte(`{"reason":"for_resume","operator":"alice"}`)
	pausedB := ut.PerformRequest(h.Engine, "POST", "/api/runs/"+runB+"/pause", &ut.Body{Body: bytes.NewReader(pauseBody), Len: len(pauseBody)})
	if got := pausedB.Result().StatusCode(); got != 200 {
		t.Fatalf("pause runB status=%d, want 200 body=%s", got, pausedB.Result().Body())
	}

	toolCallBody := []byte(`{"id":"tc_cross","step_id":"st_1","tool_name":"search","status":"SUCCEEDED"}`)
	upsertA := ut.PerformRequest(h.Engine, "POST", "/api/runs/"+runA+"/tool-calls", &ut.Body{Body: bytes.NewReader(toolCallBody), Len: len(toolCallBody)})
	if got := upsertA.Result().StatusCode(); got != 202 {
		t.Fatalf("upsert runA tool call status=%d, want 202 body=%s", got, upsertA.Result().Body())
	}

	resumeB := []byte(`{"mode":"FROM_TOOL_CALL","from_tool_call_id":"tc_cross","strategy":"REUSE_SUCCESSFUL_EFFECTS","operator":"alice"}`)
	resumed := ut.PerformRequest(h.Engine, "POST", "/api/runs/"+runB+"/resume", &ut.Body{Body: bytes.NewReader(resumeB), Len: len(resumeB)})
	if got := resumed.Result().StatusCode(); got != 400 {
		t.Fatalf("resume runB with runA tool call status=%d, want 400 body=%s", got, resumed.Result().Body())
	}
}
