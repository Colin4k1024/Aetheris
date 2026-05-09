package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/app/server"
	"github.com/cloudwego/hertz/pkg/common/ut"

	"github.com/Colin4k1024/Aetheris/v2/internal/agent/job"
	"github.com/Colin4k1024/Aetheris/v2/internal/agent/planner"
	"github.com/Colin4k1024/Aetheris/v2/internal/agent/replay"
	agentruntime "github.com/Colin4k1024/Aetheris/v2/internal/agent/runtime"
	agentexec "github.com/Colin4k1024/Aetheris/v2/internal/agent/runtime/executor"
	agenttools "github.com/Colin4k1024/Aetheris/v2/internal/agent/tools"
	apihttp "github.com/Colin4k1024/Aetheris/v2/internal/api/http"
	"github.com/Colin4k1024/Aetheris/v2/internal/runtime/jobstore"
	"github.com/Colin4k1024/Aetheris/v2/pkg/config"
)

func TestExistingHTTPAgentQuickIntake_EndToEnd(t *testing.T) {
	ctx := context.Background()
	const (
		agentID        = "existing_support_agent"
		userMessage    = "帮我查询订单 1001"
		requestKey     = "customer-msg-1001"
		upstreamAnswer = "订单 1001 已完成发货"
	)
	t.Setenv("EXISTING_AGENT_TOKEN", "token-123")

	var upstream externalHTTPAgentCapture
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		upstream.record(t, r)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"answer": upstreamAnswer,
			"final":  true,
			"metadata": map[string]any{
				"adapter": "legacy-python-agent",
			},
		})
	}))
	defer server.Close()

	cfg := &config.AgentsConfig{Agents: map[string]config.AgentDefConfig{
		agentID: {
			Type:        "external_http",
			Description: "Existing customer support agent",
			External: config.AgentExternalConfig{
				URL:      server.URL,
				Timeout:  "2s",
				TokenEnv: "EXISTING_AGENT_TOKEN",
			},
		},
	}}
	toolsReg := agenttools.NewRegistry()
	toolsReg.Register(NewExternalAgentCallTool(collectExternalAgentConfigs(cfg)))
	manager := agentruntime.NewManager()
	if err := RegisterConfiguredAgents(ctx, manager, planner.NewRulePlanner(), toolsReg, cfg); err != nil {
		t.Fatalf("register configured agents: %v", err)
	}

	jobStore := job.NewJobStoreMem()
	eventStore := jobstore.NewMemoryStore()
	handler := apihttp.NewHandler(nil, nil)
	handler.SetAgentRuntime(manager, nil, nil)
	handler.SetJobStore(jobStore)
	handler.SetJobEventStore(eventStore)
	handler.SetToolsRegistry(toolsReg)
	handler.SetPlanAtJobCreation(PlanGoalForJobFuncWithExternalAgents(manager, planner.NewRulePlanner(), cfg))

	h := serverHertz()
	h.POST("/api/agents/:id/message", func(c context.Context, reqCtx *app.RequestContext) {
		handler.AgentMessage(c, reqCtx)
	})

	jobID := postAgentMessage(t, h, agentID, userMessage, requestKey)
	duplicateJobID := postAgentMessage(t, h, agentID, userMessage, requestKey)
	if duplicateJobID != jobID {
		t.Fatalf("duplicate Idempotency-Key returned job_id %q, want %q", duplicateJobID, jobID)
	}

	assertExternalPlanGenerated(t, eventStore, jobID, userMessage)

	sink := NewNodeEventSink(eventStore)
	compiler := agentexec.NewCompiler(map[string]agentexec.NodeAdapter{
		planner.NodeTool: &agentexec.ToolNodeAdapter{
			Tools:            &toolExecAdapter{reg: toolsReg},
			ToolEventSink:    sink,
			CommandEventSink: sink,
		},
	})
	runner := agentexec.NewRunner(compiler)
	runner.SetCheckpointStores(agentruntime.NewCheckpointStoreMem(), runnerJobStoreAdapter{store: jobStore})
	runner.SetNodeEventSink(sink)
	runner.SetReplayContextBuilder(replay.NewReplayContextBuilder(eventStore))

	agent, _ := manager.Get(ctx, agentID)
	if agent == nil {
		t.Fatalf("configured agent %q was not registered", agentID)
	}
	if err := runner.RunForJob(ctx, agent, &agentexec.JobForRunner{
		ID: jobID, AgentID: agentID, Goal: userMessage, TenantID: "default",
	}); err != nil {
		t.Fatalf("RunForJob: %v", err)
	}

	if calls := upstream.calls(); calls != 1 {
		t.Fatalf("external HTTP agent called %d times, want 1", calls)
	}
	got := upstream.snapshot()
	if got.Auth != "Bearer token-123" {
		t.Fatalf("Authorization header = %q, want bearer token", got.Auth)
	}
	if got.AgentID != agentID {
		t.Fatalf("X-Aetheris-Agent-ID = %q, want %q", got.AgentID, agentID)
	}
	if got.JobID != jobID {
		t.Fatalf("X-Aetheris-Job-ID = %q, want %q", got.JobID, jobID)
	}
	if got.IdempotencyKey == "" {
		t.Fatalf("external call did not receive Idempotency-Key")
	}
	if got.Body.Message != userMessage {
		t.Fatalf("external request message = %q, want %q", got.Body.Message, userMessage)
	}
	if got.Body.Metadata["agent_id"] != agentID || got.Body.Metadata["job_id"] != jobID {
		t.Fatalf("external metadata = %#v, want agent_id and job_id", got.Body.Metadata)
	}
	if got.Body.Metadata["idempotency_key"] != got.IdempotencyKey {
		t.Fatalf("metadata idempotency_key = %v, header = %q", got.Body.Metadata["idempotency_key"], got.IdempotencyKey)
	}

	storedJob, err := jobStore.Get(ctx, jobID)
	if err != nil {
		t.Fatalf("jobStore.Get: %v", err)
	}
	if storedJob == nil || storedJob.Status != job.StatusCompleted {
		t.Fatalf("job status = %#v, want completed", storedJob)
	}
	assertJobCompletedAnswer(t, eventStore, jobID, upstreamAnswer)
}

type externalHTTPAgentCapture struct {
	mu    sync.Mutex
	count int
	last  struct {
		Auth           string
		IdempotencyKey string
		JobID          string
		AgentID        string
		Body           externalAgentRequest
	}
}

type runnerJobStoreAdapter struct {
	store *job.JobStoreMem
}

func (a runnerJobStoreAdapter) UpdateCursor(ctx context.Context, jobID string, cursor string) error {
	return a.store.UpdateCursor(ctx, jobID, cursor)
}

func (a runnerJobStoreAdapter) UpdateStatus(ctx context.Context, jobID string, status int) error {
	return a.store.UpdateStatus(ctx, jobID, job.JobStatus(status))
}

func (c *externalHTTPAgentCapture) record(t *testing.T, r *http.Request) {
	t.Helper()
	var body externalAgentRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		t.Fatalf("decode external agent request: %v", err)
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.count++
	c.last.Auth = r.Header.Get("Authorization")
	c.last.IdempotencyKey = r.Header.Get("Idempotency-Key")
	c.last.JobID = r.Header.Get("X-Aetheris-Job-ID")
	c.last.AgentID = r.Header.Get("X-Aetheris-Agent-ID")
	c.last.Body = body
}

func (c *externalHTTPAgentCapture) calls() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.count
}

func (c *externalHTTPAgentCapture) snapshot() struct {
	Auth           string
	IdempotencyKey string
	JobID          string
	AgentID        string
	Body           externalAgentRequest
} {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.last
}

func serverHertz() *server.Hertz {
	return server.Default(server.WithHostPorts(":0"))
}

func postAgentMessage(t *testing.T, h *server.Hertz, agentID, message, idempotencyKey string) string {
	t.Helper()
	body, _ := json.Marshal(map[string]string{"message": message})
	w := ut.PerformRequest(
		h.Engine,
		"POST",
		"/api/agents/"+agentID+"/message",
		&ut.Body{Body: bytes.NewReader(body), Len: len(body)},
		ut.Header{Key: "Idempotency-Key", Value: idempotencyKey},
	)
	resp := w.Result()
	if resp.StatusCode() != http.StatusAccepted {
		t.Fatalf("POST /api/agents/%s/message status got %d, want 202; body=%s", agentID, resp.StatusCode(), resp.Body())
	}
	var payload struct {
		Status            string         `json:"status"`
		AgentID           string         `json:"agent_id"`
		JobID             string         `json:"job_id"`
		RuntimeSubmission map[string]any `json:"runtime_submission"`
	}
	if err := json.Unmarshal(resp.Body(), &payload); err != nil {
		t.Fatalf("decode agent message response: %v", err)
	}
	if payload.Status != "accepted" || payload.AgentID != agentID || payload.JobID == "" {
		t.Fatalf("unexpected agent message response: %#v", payload)
	}
	if payload.RuntimeSubmission["canonical_api"] != "/api/runs" {
		t.Fatalf("runtime_submission = %#v, want canonical_api /api/runs", payload.RuntimeSubmission)
	}
	return payload.JobID
}

func assertExternalPlanGenerated(t *testing.T, eventStore jobstore.JobStore, jobID, goal string) {
	t.Helper()
	events, _, err := eventStore.ListEvents(context.Background(), jobID)
	if err != nil {
		t.Fatalf("ListEvents: %v", err)
	}
	for _, ev := range events {
		if ev.Type != jobstore.PlanGenerated {
			continue
		}
		var payload struct {
			Goal      string          `json:"goal"`
			TaskGraph json.RawMessage `json:"task_graph"`
		}
		if err := json.Unmarshal(ev.Payload, &payload); err != nil {
			t.Fatalf("decode PlanGenerated payload: %v", err)
		}
		if payload.Goal != goal {
			t.Fatalf("PlanGenerated goal = %q, want %q", payload.Goal, goal)
		}
		var graph planner.TaskGraph
		if err := graph.Unmarshal(payload.TaskGraph); err != nil {
			t.Fatalf("decode task graph: %v", err)
		}
		if len(graph.Nodes) != 1 {
			t.Fatalf("plan node count = %d, want 1", len(graph.Nodes))
		}
		node := graph.Nodes[0]
		if node.Type != planner.NodeTool || node.ToolName != ExternalAgentCallToolName {
			t.Fatalf("plan node = %#v, want external_agent_call tool", node)
		}
		return
	}
	t.Fatalf("missing PlanGenerated event for job %s", jobID)
}

func assertJobCompletedAnswer(t *testing.T, eventStore jobstore.JobStore, jobID, answer string) {
	t.Helper()
	events, _, err := eventStore.ListEvents(context.Background(), jobID)
	if err != nil {
		t.Fatalf("ListEvents: %v", err)
	}
	for i := len(events) - 1; i >= 0; i-- {
		if events[i].Type != jobstore.JobCompleted {
			continue
		}
		var payload map[string]any
		if err := json.Unmarshal(events[i].Payload, &payload); err != nil {
			t.Fatalf("decode JobCompleted payload: %v", err)
		}
		if payload["answer"] != answer || payload["result"] != answer {
			t.Fatalf("JobCompleted payload = %#v, want answer/result %q", payload, answer)
		}
		return
	}
	t.Fatalf("missing JobCompleted event for job %s", jobID)
}
