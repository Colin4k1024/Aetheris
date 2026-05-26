package http

import (
	"bytes"
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/cloudwego/hertz/pkg/app/server"
	"github.com/cloudwego/hertz/pkg/common/ut"

	"github.com/Colin4k1024/Aetheris/v2/internal/agent/job"
	"github.com/Colin4k1024/Aetheris/v2/internal/runtime/jobstore"
)

func buildForensicsTestHandler(t *testing.T) (*Handler, string) {
	t.Helper()

	jobStore := job.NewJobStoreMem()
	eventStore := jobstore.NewMemoryStore()

	h := NewHandler(nil, nil)
	h.SetJobStore(jobStore)
	h.SetJobEventStore(eventStore)

	jobID, err := jobStore.Create(context.Background(), &job.Job{
		AgentID:  "agent-1",
		TenantID: "default",
		Goal:     "forensics test",
	})
	if err != nil {
		t.Fatalf("create job: %v", err)
	}

	_, ver, err := eventStore.ListEvents(context.Background(), jobID)
	if err != nil {
		t.Fatalf("list events: %v", err)
	}

	appendEvent := func(ev jobstore.JobEvent) {
		ev.JobID = jobID
		newVer, err := eventStore.Append(context.Background(), jobID, ver, ev)
		if err != nil {
			t.Fatalf("append %s: %v", ev.Type, err)
		}
		ver = newVer
	}

	appendEvent(jobstore.JobEvent{Type: jobstore.JobCreated})

	startedPayload, _ := json.Marshal(map[string]interface{}{
		"invocation_id":   "inv-1",
		"idempotency_key": "key-1",
		"tool_name":       "stripe.charge",
		"arguments_hash":  "hash-1",
		"started_at":      time.Now().UTC().Format(time.RFC3339),
	})
	appendEvent(jobstore.JobEvent{Type: jobstore.ToolInvocationStarted, Payload: startedPayload})

	finishedPayload, _ := json.Marshal(map[string]interface{}{
		"invocation_id":   "inv-1",
		"idempotency_key": "key-1",
		"tool_name":       "stripe.charge",
		"outcome":         "success",
		"result":          map[string]interface{}{"ok": true},
		"finished_at":     time.Now().UTC().Format(time.RFC3339),
	})
	appendEvent(jobstore.JobEvent{Type: jobstore.ToolInvocationFinished, Payload: finishedPayload})

	reasoningPayload, _ := json.Marshal(map[string]interface{}{
		"step_id":     "step-1",
		"node_id":     "node-1",
		"type":        "tool",
		"label":       "charge card",
		"input_keys":  []string{"payment_request"},
		"output_keys": []string{"payment_result"},
		"evidence": map[string]interface{}{
			"tool_invocation_ids": []string{"inv-1"},
		},
	})
	appendEvent(jobstore.JobEvent{Type: jobstore.ReasoningSnapshot, Payload: reasoningPayload})

	auditPayload, _ := json.Marshal(map[string]interface{}{
		"action":  "export",
		"user_id": "user-1",
	})
	appendEvent(jobstore.JobEvent{Type: jobstore.AccessAudited, Payload: auditPayload})

	return h, jobID
}

func TestForensicsConsistencyCheck_OK(t *testing.T) {
	h, jobID := buildForensicsTestHandler(t)
	s := server.Default(server.WithHostPorts(":0"))
	s.GET("/api/forensics/consistency/:job_id", h.ForensicsConsistencyCheck)

	w := ut.PerformRequest(s.Engine, "GET", "/api/forensics/consistency/"+jobID, nil)
	if got := w.Result().StatusCode(); got != 200 {
		t.Fatalf("status = %d, want 200", got)
	}
	body := w.Result().Body()
	if !bytes.Contains(body, []byte(`"hash_chain_valid":true`)) {
		t.Fatalf("hash chain should be valid: %s", body)
	}
	if !bytes.Contains(body, []byte(`"ledger_consistent":true`)) {
		t.Fatalf("ledger should be consistent: %s", body)
	}
}

func TestGetJobEvidenceGraph_OK(t *testing.T) {
	h, jobID := buildForensicsTestHandler(t)
	s := server.Default(server.WithHostPorts(":0"))
	s.GET("/api/jobs/:id/evidence-graph", h.GetJobEvidenceGraph)

	w := ut.PerformRequest(s.Engine, "GET", "/api/jobs/"+jobID+"/evidence-graph", nil)
	if got := w.Result().StatusCode(); got != 200 {
		t.Fatalf("status = %d, want 200", got)
	}
	body := w.Result().Body()
	if !bytes.Contains(body, []byte(`"step_id":"step-1"`)) {
		t.Fatalf("evidence graph should contain reasoning node: %s", body)
	}
}

func TestGetJobAuditLog_OK(t *testing.T) {
	h, jobID := buildForensicsTestHandler(t)
	s := server.Default(server.WithHostPorts(":0"))
	s.GET("/api/jobs/:id/audit-log", h.GetJobAuditLog)

	w := ut.PerformRequest(s.Engine, "GET", "/api/jobs/"+jobID+"/audit-log", nil)
	if got := w.Result().StatusCode(); got != 200 {
		t.Fatalf("status = %d, want 200", got)
	}
	body := w.Result().Body()
	if !bytes.Contains(body, []byte(`"count":1`)) {
		t.Fatalf("audit log count should be 1: %s", body)
	}
}

func TestForensicsBatchExport_StatusFlow(t *testing.T) {
	h, jobID := buildForensicsTestHandler(t)
	s := server.Default(server.WithHostPorts(":0"))
	s.POST("/api/forensics/batch-export", h.ForensicsBatchExport)
	s.GET("/api/forensics/export-status/:task_id", h.ForensicsExportStatus)

	body := []byte(`{"job_ids":["` + jobID + `"]}`)
	w := ut.PerformRequest(s.Engine, "POST", "/api/forensics/batch-export", &ut.Body{Body: bytes.NewReader(body), Len: len(body)})
	if got := w.Result().StatusCode(); got != 202 {
		t.Fatalf("status = %d, want 202", got)
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Result().Body(), &resp); err != nil {
		t.Fatalf("unmarshal batch export response: %v", err)
	}
	taskID, _ := resp["task_id"].(string)
	if taskID == "" {
		t.Fatalf("task_id should not be empty")
	}

	deadline := time.Now().Add(2 * time.Second)
	for {
		statusResp := ut.PerformRequest(s.Engine, "GET", "/api/forensics/export-status/"+taskID, nil)
		if got := statusResp.Result().StatusCode(); got != 200 {
			t.Fatalf("status query code = %d, want 200", got)
		}
		if bytes.Contains(statusResp.Result().Body(), []byte(`"status":"completed"`)) {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("batch export task did not complete in time: %s", statusResp.Result().Body())
		}
		time.Sleep(20 * time.Millisecond)
	}
}

func TestForensicsQuery_PaginationLimitCapAndFilters(t *testing.T) {
	ctx := context.Background()
	jobStore := job.NewJobStoreMem()
	eventStore := jobstore.NewMemoryStore()
	h := NewHandler(nil, nil)
	h.SetJobStore(jobStore)
	h.SetJobEventStore(eventStore)

	for i := 0; i < 205; i++ {
		jobID, err := jobStore.Create(ctx, &job.Job{
			AgentID:  "agent-page",
			TenantID: "tenant-page",
			Goal:     "pagination test",
		})
		if err != nil {
			t.Fatalf("create job %d: %v", i, err)
		}
		_, ver, err := eventStore.ListEvents(ctx, jobID)
		if err != nil {
			t.Fatalf("list events for %s: %v", jobID, err)
		}
		payload, _ := json.Marshal(map[string]interface{}{
			"invocation_id":   "inv-page",
			"idempotency_key": "key-page",
			"tool_name":       "stripe.charge",
			"outcome":         "success",
			"finished_at":     time.Now().UTC().Format(time.RFC3339),
		})
		if ver, err = eventStore.Append(ctx, jobID, ver, jobstore.JobEvent{JobID: jobID, Type: jobstore.JobCreated}); err != nil {
			t.Fatalf("append created: %v", err)
		}
		if _, err = eventStore.Append(ctx, jobID, ver, jobstore.JobEvent{JobID: jobID, Type: jobstore.ToolInvocationFinished, Payload: payload}); err != nil {
			t.Fatalf("append tool finished: %v", err)
		}
	}

	s := server.Default(server.WithHostPorts(":0"))
	s.POST("/api/forensics/query", h.ForensicsQuery)

	body := []byte(`{
		"tenant_id":"tenant-page",
		"agent_filter":["agent-page"],
		"tool_filter":["stripe*"],
		"limit":500,
		"offset":0
	}`)
	w := ut.PerformRequest(s.Engine, "POST", "/api/forensics/query", &ut.Body{Body: bytes.NewReader(body), Len: len(body)})
	if got := w.Result().StatusCode(); got != 200 {
		t.Fatalf("status = %d, want 200; body=%s", got, w.Result().Body())
	}

	var resp struct {
		Jobs       []map[string]interface{} `json:"jobs"`
		TotalCount int                      `json:"total_count"`
		Page       int                      `json:"page"`
	}
	if err := json.Unmarshal(w.Result().Body(), &resp); err != nil {
		t.Fatalf("unmarshal query response: %v", err)
	}
	if resp.TotalCount != 205 {
		t.Fatalf("total_count = %d, want 205", resp.TotalCount)
	}
	if len(resp.Jobs) != 200 {
		t.Fatalf("jobs length = %d, want capped page size 200", len(resp.Jobs))
	}
	if resp.Page != 0 {
		t.Fatalf("page = %d, want 0", resp.Page)
	}
}

func TestForensicsQuery_LargeEventStreamEventCount(t *testing.T) {
	ctx := context.Background()
	jobStore := job.NewJobStoreMem()
	eventStore := jobstore.NewMemoryStore()
	h := NewHandler(nil, nil)
	h.SetJobStore(jobStore)
	h.SetJobEventStore(eventStore)

	jobID, err := jobStore.Create(ctx, &job.Job{
		AgentID:  "agent-large-events",
		TenantID: "tenant-large-events",
		Goal:     "large event stream test",
	})
	if err != nil {
		t.Fatalf("create job: %v", err)
	}
	_, ver, err := eventStore.ListEvents(ctx, jobID)
	if err != nil {
		t.Fatalf("list events: %v", err)
	}
	appendEvent := func(ev jobstore.JobEvent) {
		ev.JobID = jobID
		nextVer, err := eventStore.Append(ctx, jobID, ver, ev)
		if err != nil {
			t.Fatalf("append %s: %v", ev.Type, err)
		}
		ver = nextVer
	}
	appendEvent(jobstore.JobEvent{Type: jobstore.JobCreated})
	for i := 0; i < 1000; i++ {
		appendEvent(jobstore.JobEvent{Type: jobstore.CriticalDecisionMade})
	}

	s := server.Default(server.WithHostPorts(":0"))
	s.POST("/api/forensics/query", h.ForensicsQuery)

	body := []byte(`{
		"tenant_id":"tenant-large-events",
		"agent_filter":["agent-large-events"],
		"event_filter":["critical_decision_made"],
		"limit":20,
		"offset":0
	}`)
	w := ut.PerformRequest(s.Engine, "POST", "/api/forensics/query", &ut.Body{Body: bytes.NewReader(body), Len: len(body)})
	if got := w.Result().StatusCode(); got != 200 {
		t.Fatalf("status = %d, want 200; body=%s", got, w.Result().Body())
	}

	var resp struct {
		Jobs []struct {
			JobID      string `json:"job_id"`
			EventCount int    `json:"event_count"`
		} `json:"jobs"`
		TotalCount int `json:"total_count"`
	}
	if err := json.Unmarshal(w.Result().Body(), &resp); err != nil {
		t.Fatalf("unmarshal query response: %v", err)
	}
	if resp.TotalCount != 1 || len(resp.Jobs) != 1 {
		t.Fatalf("expected one matching job, got total=%d jobs=%v", resp.TotalCount, resp.Jobs)
	}
	if resp.Jobs[0].EventCount != 1001 {
		t.Fatalf("event_count = %d, want 1001", resp.Jobs[0].EventCount)
	}
}
