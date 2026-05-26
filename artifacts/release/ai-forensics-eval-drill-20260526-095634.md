# AI Forensics Eval Release Drill

- Timestamp: 20260526-095634
- Overall: PASS

## Suites

- pkg/ai_forensics (PASS)
- internal/api/http (PASS)

## pkg/ai_forensics output

```text
=== RUN   TestAnomalyDetector
--- PASS: TestAnomalyDetector (0.00s)
=== RUN   TestAnomalyDetector_RetryLoopAndTamperedReasoning
--- PASS: TestAnomalyDetector_RetryLoopAndTamperedReasoning (0.00s)
=== RUN   TestGoldenEvalCases_PassWithinFalsePositiveBudget
--- PASS: TestGoldenEvalCases_PassWithinFalsePositiveBudget (0.00s)
=== RUN   TestPatternMatcher
--- PASS: TestPatternMatcher (0.00s)
PASS
ok  	github.com/Colin4k1024/Aetheris/v2/pkg/ai_forensics	0.426s
```

## internal/api/http output

```text
=== RUN   TestAIForensicsDetectAnomalies_EventSignals
2026/05/26 09:56:48.745472 engine.go:702: [Debug] HERTZ: Method=POST   absolutePath=/api/forensics/ai/detect-anomalies --> handlerName=github.com/Colin4k1024/Aetheris/v2/internal/api/http.(*Handler).AIForensicsDetectAnomalies-fm (num=2 handlers)
--- PASS: TestAIForensicsDetectAnomalies_EventSignals (0.00s)
=== RUN   TestRouter_ForensicsRoutesDisabledByDefault
2026/05/26 09:56:48.746613 engine.go:702: [Debug] HERTZ: Method=GET    absolutePath=/                         --> handlerName=github.com/Colin4k1024/Aetheris/v2/internal/api/http.(*Handler).HomePage-fm (num=4 handlers)
2026/05/26 09:56:48.746621 engine.go:702: [Debug] HERTZ: Method=GET    absolutePath=/metrics                  --> handlerName=github.com/Colin4k1024/Aetheris/v2/internal/api/http.(*Handler).SystemMetrics-fm (num=4 handlers)
2026/05/26 09:56:48.746625 engine.go:702: [Debug] HERTZ: Method=GET    absolutePath=/api/health               --> handlerName=github.com/Colin4k1024/Aetheris/v2/internal/api/http.(*Handler).HealthCheck-fm (num=4 handlers)
2026/05/26 09:56:48.746629 engine.go:702: [Debug] HERTZ: Method=POST   absolutePath=/api/documents/upload     --> handlerName=github.com/Colin4k1024/Aetheris/v2/internal/api/http.(*Handler).UploadDocument-fm (num=6 handlers)
2026/05/26 09:56:48.746631 engine.go:702: [Debug] HERTZ: Method=POST   absolutePath=/api/documents/upload/async --> handlerName=github.com/Colin4k1024/Aetheris/v2/internal/api/http.(*Handler).UploadDocumentAsync-fm (num=6 handlers)
2026/05/26 09:56:48.746634 engine.go:702: [Debug] HERTZ: Method=GET    absolutePath=/api/documents/upload/status/:task_id --> handlerName=github.com/Colin4k1024/Aetheris/v2/internal/api/http.(*Handler).UploadStatus-fm (num=6 handlers)
2026/05/26 09:56:48.746636 engine.go:702: [Debug] HERTZ: Method=GET    absolutePath=/api/documents/           --> handlerName=github.com/Colin4k1024/Aetheris/v2/internal/api/http.(*Handler).ListDocuments-fm (num=6 handlers)
2026/05/26 09:56:48.746638 engine.go:702: [Debug] HERTZ: Method=GET    absolutePath=/api/documents/:id        --> handlerName=github.com/Colin4k1024/Aetheris/v2/internal/api/http.(*Handler).GetDocument-fm (num=6 handlers)
2026/05/26 09:56:48.746640 engine.go:702: [Debug] HERTZ: Method=DELETE absolutePath=/api/documents/:id        --> handlerName=github.com/Colin4k1024/Aetheris/v2/internal/api/http.(*Handler).DeleteDocument-fm (num=6 handlers)
2026/05/26 09:56:48.746642 engine.go:702: [Debug] HERTZ: Method=GET    absolutePath=/api/knowledge/collections --> handlerName=github.com/Colin4k1024/Aetheris/v2/internal/api/http.(*Handler).ListCollections-fm (num=6 handlers)
2026/05/26 09:56:48.746644 engine.go:702: [Debug] HERTZ: Method=POST   absolutePath=/api/knowledge/collections --> handlerName=github.com/Colin4k1024/Aetheris/v2/internal/api/http.(*Handler).CreateCollection-fm (num=6 handlers)
2026/05/26 09:56:48.746647 engine.go:702: [Debug] HERTZ: Method=DELETE absolutePath=/api/knowledge/collections/:id --> handlerName=github.com/Colin4k1024/Aetheris/v2/internal/api/http.(*Handler).DeleteCollection-fm (num=6 handlers)
2026/05/26 09:56:48.746649 engine.go:702: [Debug] HERTZ: Method=POST   absolutePath=/api/runs                 --> handlerName=github.com/Colin4k1024/Aetheris/v2/internal/api/http.(*Handler).CreateRun-fm (num=6 handlers)
2026/05/26 09:56:48.746650 engine.go:702: [Debug] HERTZ: Method=POST   absolutePath=/api/runs/                --> handlerName=github.com/Colin4k1024/Aetheris/v2/internal/api/http.(*Handler).CreateRun-fm (num=6 handlers)
2026/05/26 09:56:48.746652 engine.go:702: [Debug] HERTZ: Method=GET    absolutePath=/api/runs/:id             --> handlerName=github.com/Colin4k1024/Aetheris/v2/internal/api/http.(*Handler).GetRun-fm (num=6 handlers)
2026/05/26 09:56:48.746654 engine.go:702: [Debug] HERTZ: Method=GET    absolutePath=/api/runs/:id/events      --> handlerName=github.com/Colin4k1024/Aetheris/v2/internal/api/http.(*Handler).GetRunEvents-fm (num=6 handlers)
2026/05/26 09:56:48.746656 engine.go:702: [Debug] HERTZ: Method=POST   absolutePath=/api/runs/:id/tool-calls  --> handlerName=github.com/Colin4k1024/Aetheris/v2/internal/api/http.(*Handler).UpsertToolCall-fm (num=6 handlers)
2026/05/26 09:56:48.746658 engine.go:702: [Debug] HERTZ: Method=POST   absolutePath=/api/runs/:id/pause       --> handlerName=github.com/Colin4k1024/Aetheris/v2/internal/api/http.(*Handler).PauseRun-fm (num=6 handlers)
2026/05/26 09:56:48.746660 engine.go:702: [Debug] HERTZ: Method=POST   absolutePath=/api/runs/:id/resume      --> handlerName=github.com/Colin4k1024/Aetheris/v2/internal/api/http.(*Handler).ResumeRun-fm (num=6 handlers)
2026/05/26 09:56:48.746662 engine.go:702: [Debug] HERTZ: Method=POST   absolutePath=/api/runs/:id/human-decisions --> handlerName=github.com/Colin4k1024/Aetheris/v2/internal/api/http.(*Handler).InjectHumanDecision-fm (num=6 handlers)
2026/05/26 09:56:48.746664 engine.go:702: [Debug] HERTZ: Method=POST   absolutePath=/api/query/               --> handlerName=github.com/Colin4k1024/Aetheris/v2/internal/api/http.(*Handler).Query-fm (num=7 handlers)
2026/05/26 09:56:48.746666 engine.go:702: [Debug] HERTZ: Method=POST   absolutePath=/api/query/batch          --> handlerName=github.com/Colin4k1024/Aetheris/v2/internal/api/http.(*Handler).BatchQuery-fm (num=7 handlers)
2026/05/26 09:56:48.746668 engine.go:702: [Debug] HERTZ: Method=POST   absolutePath=/api/agent/run            --> handlerName=github.com/Colin4k1024/Aetheris/v2/internal/api/http.(*Handler).AgentRun-fm (num=7 handlers)
2026/05/26 09:56:48.746669 engine.go:702: [Debug] HERTZ: Method=POST   absolutePath=/api/agent/resume         --> handlerName=github.com/Colin4k1024/Aetheris/v2/internal/api/http.(*Handler).AgentResumeCheckpoint-fm (num=7 handlers)
2026/05/26 09:56:48.746674 engine.go:702: [Debug] HERTZ: Method=POST   absolutePath=/api/agent/stream         --> handlerName=github.com/Colin4k1024/Aetheris/v2/internal/api/http.(*Handler).AgentStream-fm (num=7 handlers)
2026/05/26 09:56:48.746676 engine.go:702: [Debug] HERTZ: Method=POST   absolutePath=/api/agents/:id/message   --> handlerName=github.com/Colin4k1024/Aetheris/v2/internal/api/http.(*Handler).AgentMessage-fm (num=7 handlers)
2026/05/26 09:56:48.746678 engine.go:702: [Debug] HERTZ: Method=GET    absolutePath=/api/agents/:id/state     --> handlerName=github.com/Colin4k1024/Aetheris/v2/internal/api/http.(*Handler).AgentState-fm (num=7 handlers)
2026/05/26 09:56:48.746680 engine.go:702: [Debug] HERTZ: Method=POST   absolutePath=/api/agents/:id/resume    --> handlerName=github.com/Colin4k1024/Aetheris/v2/internal/api/http.(*Handler).AgentResume-fm (num=7 handlers)
2026/05/26 09:56:48.746682 engine.go:702: [Debug] HERTZ: Method=POST   absolutePath=/api/agents/:id/stop      --> handlerName=github.com/Colin4k1024/Aetheris/v2/internal/api/http.(*Handler).AgentStop-fm (num=7 handlers)
2026/05/26 09:56:48.746684 engine.go:702: [Debug] HERTZ: Method=GET    absolutePath=/api/agents/:id/jobs/:job_id --> handlerName=github.com/Colin4k1024/Aetheris/v2/internal/api/http.(*Handler).GetAgentJob-fm (num=7 handlers)
2026/05/26 09:56:48.746686 engine.go:702: [Debug] HERTZ: Method=GET    absolutePath=/api/agents/:id/jobs      --> handlerName=github.com/Colin4k1024/Aetheris/v2/internal/api/http.(*Handler).ListAgentJobs-fm (num=7 handlers)
2026/05/26 09:56:48.746688 engine.go:702: [Debug] HERTZ: Method=GET    absolutePath=/api/jobs/:id             --> handlerName=github.com/Colin4k1024/Aetheris/v2/internal/api/http.(*Handler).GetJob-fm (num=6 handlers)
2026/05/26 09:56:48.746694 engine.go:702: [Debug] HERTZ: Method=POST   absolutePath=/api/jobs/:id/stop        --> handlerName=github.com/Colin4k1024/Aetheris/v2/internal/api/http.(*Handler).JobStop-fm (num=6 handlers)
2026/05/26 09:56:48.746698 engine.go:702: [Debug] HERTZ: Method=POST   absolutePath=/api/jobs/:id/pause       --> handlerName=github.com/Colin4k1024/Aetheris/v2/internal/api/http.(*Handler).JobPause-fm (num=6 handlers)
2026/05/26 09:56:48.746700 engine.go:702: [Debug] HERTZ: Method=POST   absolutePath=/api/jobs/:id/resume      --> handlerName=github.com/Colin4k1024/Aetheris/v2/internal/api/http.(*Handler).JobResume-fm (num=6 handlers)
2026/05/26 09:56:48.746702 engine.go:702: [Debug] HERTZ: Method=POST   absolutePath=/api/jobs/:id/signal      --> handlerName=github.com/Colin4k1024/Aetheris/v2/internal/api/http.(*Handler).JobSignal-fm (num=6 handlers)
2026/05/26 09:56:48.746704 engine.go:702: [Debug] HERTZ: Method=POST   absolutePath=/api/jobs/:id/message     --> handlerName=github.com/Colin4k1024/Aetheris/v2/internal/api/http.(*Handler).JobMessage-fm (num=6 handlers)
2026/05/26 09:56:48.746706 engine.go:702: [Debug] HERTZ: Method=GET    absolutePath=/api/jobs/:id/events      --> handlerName=github.com/Colin4k1024/Aetheris/v2/internal/api/http.(*Handler).GetJobEvents-fm (num=6 handlers)
2026/05/26 09:56:48.746707 engine.go:702: [Debug] HERTZ: Method=GET    absolutePath=/api/jobs/:id/replay      --> handlerName=github.com/Colin4k1024/Aetheris/v2/internal/api/http.(*Handler).GetJobReplay-fm (num=6 handlers)
2026/05/26 09:56:48.746711 engine.go:702: [Debug] HERTZ: Method=GET    absolutePath=/api/jobs/:id/verify      --> handlerName=github.com/Colin4k1024/Aetheris/v2/internal/api/http.(*Handler).GetJobVerify-fm (num=6 handlers)
2026/05/26 09:56:48.746714 engine.go:702: [Debug] HERTZ: Method=GET    absolutePath=/api/jobs/:id/trace       --> handlerName=github.com/Colin4k1024/Aetheris/v2/internal/api/http.(*Handler).GetJobTrace-fm (num=6 handlers)
2026/05/26 09:56:48.746716 engine.go:702: [Debug] HERTZ: Method=GET    absolutePath=/api/jobs/:id/trace/cognition --> handlerName=github.com/Colin4k1024/Aetheris/v2/internal/api/http.(*Handler).GetJobCognitionTrace-fm (num=6 handlers)
2026/05/26 09:56:48.746718 engine.go:702: [Debug] HERTZ: Method=GET    absolutePath=/api/jobs/:id/nodes/:node_id --> handlerName=github.com/Colin4k1024/Aetheris/v2/internal/api/http.(*Handler).GetJobNode-fm (num=6 handlers)
2026/05/26 09:56:48.746721 engine.go:702: [Debug] HERTZ: Method=GET    absolutePath=/api/jobs/:id/trace/page  --> handlerName=github.com/Colin4k1024/Aetheris/v2/internal/api/http.(*Handler).GetJobTracePage-fm (num=6 handlers)
2026/05/26 09:56:48.746723 engine.go:702: [Debug] HERTZ: Method=POST   absolutePath=/api/jobs/:id/export      --> handlerName=github.com/Colin4k1024/Aetheris/v2/internal/api/http.(*Handler).ExportJobForensics-fm (num=6 handlers)
2026/05/26 09:56:48.746725 engine.go:702: [Debug] HERTZ: Method=POST   absolutePath=/api/approvals            --> handlerName=github.com/Colin4k1024/Aetheris/v2/internal/api/http.(*Handler).CreateApproval-fm (num=6 handlers)
2026/05/26 09:56:48.746726 engine.go:702: [Debug] HERTZ: Method=GET    absolutePath=/api/approvals            --> handlerName=github.com/Colin4k1024/Aetheris/v2/internal/api/http.(*Handler).ListApprovals-fm (num=6 handlers)
2026/05/26 09:56:48.746728 engine.go:702: [Debug] HERTZ: Method=GET    absolutePath=/api/approvals/:id        --> handlerName=github.com/Colin4k1024/Aetheris/v2/internal/api/http.(*Handler).GetApproval-fm (num=6 handlers)
2026/05/26 09:56:48.746743 engine.go:702: [Debug] HERTZ: Method=POST   absolutePath=/api/approvals/:id/approve --> handlerName=github.com/Colin4k1024/Aetheris/v2/internal/api/http.(*Handler).ApproveApproval-fm (num=6 handlers)
2026/05/26 09:56:48.746747 engine.go:702: [Debug] HERTZ: Method=POST   absolutePath=/api/approvals/:id/reject --> handlerName=github.com/Colin4k1024/Aetheris/v2/internal/api/http.(*Handler).RejectApproval-fm (num=6 handlers)
2026/05/26 09:56:48.746749 engine.go:702: [Debug] HERTZ: Method=GET    absolutePath=/api/tools/               --> handlerName=github.com/Colin4k1024/Aetheris/v2/internal/api/http.(*Handler).ListTools-fm (num=6 handlers)
2026/05/26 09:56:48.746751 engine.go:702: [Debug] HERTZ: Method=GET    absolutePath=/api/tools/:name          --> handlerName=github.com/Colin4k1024/Aetheris/v2/internal/api/http.(*Handler).GetTool-fm (num=6 handlers)
2026/05/26 09:56:48.746753 engine.go:702: [Debug] HERTZ: Method=GET    absolutePath=/api/system/status        --> handlerName=github.com/Colin4k1024/Aetheris/v2/internal/api/http.(*Handler).SystemStatus-fm (num=6 handlers)
2026/05/26 09:56:48.746755 engine.go:702: [Debug] HERTZ: Method=GET    absolutePath=/api/system/metrics       --> handlerName=github.com/Colin4k1024/Aetheris/v2/internal/api/http.(*Handler).SystemMetrics-fm (num=6 handlers)
2026/05/26 09:56:48.746757 engine.go:702: [Debug] HERTZ: Method=GET    absolutePath=/api/system/workers       --> handlerName=github.com/Colin4k1024/Aetheris/v2/internal/api/http.(*Handler).SystemWorkers-fm (num=6 handlers)
2026/05/26 09:56:48.746759 engine.go:702: [Debug] HERTZ: Method=GET    absolutePath=/api/observability/summary --> handlerName=github.com/Colin4k1024/Aetheris/v2/internal/api/http.(*Handler).GetObservabilitySummary-fm (num=6 handlers)
2026/05/26 09:56:48.746760 engine.go:702: [Debug] HERTZ: Method=GET    absolutePath=/api/observability/stuck  --> handlerName=github.com/Colin4k1024/Aetheris/v2/internal/api/http.(*Handler).GetObservabilityStuck-fm (num=6 handlers)
2026/05/26 09:56:48.746763 engine.go:702: [Debug] HERTZ: Method=GET    absolutePath=/api/trace/overview/page  --> handlerName=github.com/Colin4k1024/Aetheris/v2/internal/api/http.(*Handler).GetTraceOverviewPage-fm (num=6 handlers)
2026/05/26 09:56:48.746765 engine.go:702: [Debug] HERTZ: Method=GET    absolutePath=/api/rbac/role            --> handlerName=github.com/Colin4k1024/Aetheris/v2/internal/api/http.(*Handler).GetUserRole-fm (num=6 handlers)
2026/05/26 09:56:48.746767 engine.go:702: [Debug] HERTZ: Method=POST   absolutePath=/api/rbac/role            --> handlerName=github.com/Colin4k1024/Aetheris/v2/internal/api/http.(*Handler).AssignRole-fm (num=6 handlers)
2026/05/26 09:56:48.746769 engine.go:702: [Debug] HERTZ: Method=POST   absolutePath=/api/rbac/check           --> handlerName=github.com/Colin4k1024/Aetheris/v2/internal/api/http.(*Handler).CheckPermission-fm (num=6 handlers)
2026/05/26 09:56:48.746818 middleware.go:413: [Info] POST /api/forensics/query 0.0.0.0 404 13.5µs
2026/05/26 09:56:48.746843 middleware.go:413: [Info] GET /api/jobs/job_x/evidence-graph 0.0.0.0 404 1.209µs
2026/05/26 09:56:48.746850 middleware.go:413: [Info] GET /api/jobs/job_x/audit-log 0.0.0.0 404 916ns
2026/05/26 09:56:48.746856 middleware.go:413: [Info] POST /api/forensics/ai/detect-anomalies 0.0.0.0 404 709ns
2026/05/26 09:56:48.746861 middleware.go:413: [Info] GET /api/compliance/templates 0.0.0.0 404 584ns
--- PASS: TestRouter_ForensicsRoutesDisabledByDefault (0.00s)
=== RUN   TestRouter_ForensicsRoutesEnabled
2026/05/26 09:56:48.746884 engine.go:702: [Debug] HERTZ: Method=GET    absolutePath=/                         --> handlerName=github.com/Colin4k1024/Aetheris/v2/internal/api/http.(*Handler).HomePage-fm (num=4 handlers)
2026/05/26 09:56:48.746887 engine.go:702: [Debug] HERTZ: Method=GET    absolutePath=/metrics                  --> handlerName=github.com/Colin4k1024/Aetheris/v2/internal/api/http.(*Handler).SystemMetrics-fm (num=4 handlers)
2026/05/26 09:56:48.746889 engine.go:702: [Debug] HERTZ: Method=GET    absolutePath=/api/health               --> handlerName=github.com/Colin4k1024/Aetheris/v2/internal/api/http.(*Handler).HealthCheck-fm (num=4 handlers)
2026/05/26 09:56:48.746891 engine.go:702: [Debug] HERTZ: Method=POST   absolutePath=/api/documents/upload     --> handlerName=github.com/Colin4k1024/Aetheris/v2/internal/api/http.(*Handler).UploadDocument-fm (num=6 handlers)
2026/05/26 09:56:48.746893 engine.go:702: [Debug] HERTZ: Method=POST   absolutePath=/api/documents/upload/async --> handlerName=github.com/Colin4k1024/Aetheris/v2/internal/api/http.(*Handler).UploadDocumentAsync-fm (num=6 handlers)
2026/05/26 09:56:48.746895 engine.go:702: [Debug] HERTZ: Method=GET    absolutePath=/api/documents/upload/status/:task_id --> handlerName=github.com/Colin4k1024/Aetheris/v2/internal/api/http.(*Handler).UploadStatus-fm (num=6 handlers)
2026/05/26 09:56:48.746899 engine.go:702: [Debug] HERTZ: Method=GET    absolutePath=/api/documents/           --> handlerName=github.com/Colin4k1024/Aetheris/v2/internal/api/http.(*Handler).ListDocuments-fm (num=6 handlers)
2026/05/26 09:56:48.746900 engine.go:702: [Debug] HERTZ: Method=GET    absolutePath=/api/documents/:id        --> handlerName=github.com/Colin4k1024/Aetheris/v2/internal/api/http.(*Handler).GetDocument-fm (num=6 handlers)
2026/05/26 09:56:48.746902 engine.go:702: [Debug] HERTZ: Method=DELETE absolutePath=/api/documents/:id        --> handlerName=github.com/Colin4k1024/Aetheris/v2/internal/api/http.(*Handler).DeleteDocument-fm (num=6 handlers)
2026/05/26 09:56:48.746904 engine.go:702: [Debug] HERTZ: Method=GET    absolutePath=/api/knowledge/collections --> handlerName=github.com/Colin4k1024/Aetheris/v2/internal/api/http.(*Handler).ListCollections-fm (num=6 handlers)
2026/05/26 09:56:48.746906 engine.go:702: [Debug] HERTZ: Method=POST   absolutePath=/api/knowledge/collections --> handlerName=github.com/Colin4k1024/Aetheris/v2/internal/api/http.(*Handler).CreateCollection-fm (num=6 handlers)
2026/05/26 09:56:48.746907 engine.go:702: [Debug] HERTZ: Method=DELETE absolutePath=/api/knowledge/collections/:id --> handlerName=github.com/Colin4k1024/Aetheris/v2/internal/api/http.(*Handler).DeleteCollection-fm (num=6 handlers)
2026/05/26 09:56:48.746909 engine.go:702: [Debug] HERTZ: Method=POST   absolutePath=/api/runs                 --> handlerName=github.com/Colin4k1024/Aetheris/v2/internal/api/http.(*Handler).CreateRun-fm (num=6 handlers)
2026/05/26 09:56:48.746911 engine.go:702: [Debug] HERTZ: Method=POST   absolutePath=/api/runs/                --> handlerName=github.com/Colin4k1024/Aetheris/v2/internal/api/http.(*Handler).CreateRun-fm (num=6 handlers)
2026/05/26 09:56:48.746925 engine.go:702: [Debug] HERTZ: Method=GET    absolutePath=/api/runs/:id             --> handlerName=github.com/Colin4k1024/Aetheris/v2/internal/api/http.(*Handler).GetRun-fm (num=6 handlers)
2026/05/26 09:56:48.746930 engine.go:702: [Debug] HERTZ: Method=GET    absolutePath=/api/runs/:id/events      --> handlerName=github.com/Colin4k1024/Aetheris/v2/internal/api/http.(*Handler).GetRunEvents-fm (num=6 handlers)
2026/05/26 09:56:48.746933 engine.go:702: [Debug] HERTZ: Method=POST   absolutePath=/api/runs/:id/tool-calls  --> handlerName=github.com/Colin4k1024/Aetheris/v2/internal/api/http.(*Handler).UpsertToolCall-fm (num=6 handlers)
2026/05/26 09:56:48.746935 engine.go:702: [Debug] HERTZ: Method=POST   absolutePath=/api/runs/:id/pause       --> handlerName=github.com/Colin4k1024/Aetheris/v2/internal/api/http.(*Handler).PauseRun-fm (num=6 handlers)
2026/05/26 09:56:48.746939 engine.go:702: [Debug] HERTZ: Method=POST   absolutePath=/api/runs/:id/resume      --> handlerName=github.com/Colin4k1024/Aetheris/v2/internal/api/http.(*Handler).ResumeRun-fm (num=6 handlers)
2026/05/26 09:56:48.746942 engine.go:702: [Debug] HERTZ: Method=POST   absolutePath=/api/runs/:id/human-decisions --> handlerName=github.com/Colin4k1024/Aetheris/v2/internal/api/http.(*Handler).InjectHumanDecision-fm (num=6 handlers)
2026/05/26 09:56:48.746946 engine.go:702: [Debug] HERTZ: Method=POST   absolutePath=/api/query/               --> handlerName=github.com/Colin4k1024/Aetheris/v2/internal/api/http.(*Handler).Query-fm (num=7 handlers)
2026/05/26 09:56:48.746948 engine.go:702: [Debug] HERTZ: Method=POST   absolutePath=/api/query/batch          --> handlerName=github.com/Colin4k1024/Aetheris/v2/internal/api/http.(*Handler).BatchQuery-fm (num=7 handlers)
2026/05/26 09:56:48.746951 engine.go:702: [Debug] HERTZ: Method=POST   absolutePath=/api/agent/run            --> handlerName=github.com/Colin4k1024/Aetheris/v2/internal/api/http.(*Handler).AgentRun-fm (num=7 handlers)
2026/05/26 09:56:48.746955 engine.go:702: [Debug] HERTZ: Method=POST   absolutePath=/api/agent/resume         --> handlerName=github.com/Colin4k1024/Aetheris/v2/internal/api/http.(*Handler).AgentResumeCheckpoint-fm (num=7 handlers)
2026/05/26 09:56:48.746957 engine.go:702: [Debug] HERTZ: Method=POST   absolutePath=/api/agent/stream         --> handlerName=github.com/Colin4k1024/Aetheris/v2/internal/api/http.(*Handler).AgentStream-fm (num=7 handlers)
2026/05/26 09:56:48.746960 engine.go:702: [Debug] HERTZ: Method=POST   absolutePath=/api/agents/:id/message   --> handlerName=github.com/Colin4k1024/Aetheris/v2/internal/api/http.(*Handler).AgentMessage-fm (num=7 handlers)
2026/05/26 09:56:48.746963 engine.go:702: [Debug] HERTZ: Method=GET    absolutePath=/api/agents/:id/state     --> handlerName=github.com/Colin4k1024/Aetheris/v2/internal/api/http.(*Handler).AgentState-fm (num=7 handlers)
2026/05/26 09:56:48.746965 engine.go:702: [Debug] HERTZ: Method=POST   absolutePath=/api/agents/:id/resume    --> handlerName=github.com/Colin4k1024/Aetheris/v2/internal/api/http.(*Handler).AgentResume-fm (num=7 handlers)
2026/05/26 09:56:48.746966 engine.go:702: [Debug] HERTZ: Method=POST   absolutePath=/api/agents/:id/stop      --> handlerName=github.com/Colin4k1024/Aetheris/v2/internal/api/http.(*Handler).AgentStop-fm (num=7 handlers)
2026/05/26 09:56:48.746969 engine.go:702: [Debug] HERTZ: Method=GET    absolutePath=/api/agents/:id/jobs/:job_id --> handlerName=github.com/Colin4k1024/Aetheris/v2/internal/api/http.(*Handler).GetAgentJob-fm (num=7 handlers)
2026/05/26 09:56:48.746971 engine.go:702: [Debug] HERTZ: Method=GET    absolutePath=/api/agents/:id/jobs      --> handlerName=github.com/Colin4k1024/Aetheris/v2/internal/api/http.(*Handler).ListAgentJobs-fm (num=7 handlers)
2026/05/26 09:56:48.746973 engine.go:702: [Debug] HERTZ: Method=GET    absolutePath=/api/jobs/:id             --> handlerName=github.com/Colin4k1024/Aetheris/v2/internal/api/http.(*Handler).GetJob-fm (num=6 handlers)
2026/05/26 09:56:48.746975 engine.go:702: [Debug] HERTZ: Method=POST   absolutePath=/api/jobs/:id/stop        --> handlerName=github.com/Colin4k1024/Aetheris/v2/internal/api/http.(*Handler).JobStop-fm (num=6 handlers)
2026/05/26 09:56:48.746977 engine.go:702: [Debug] HERTZ: Method=POST   absolutePath=/api/jobs/:id/pause       --> handlerName=github.com/Colin4k1024/Aetheris/v2/internal/api/http.(*Handler).JobPause-fm (num=6 handlers)
2026/05/26 09:56:48.746979 engine.go:702: [Debug] HERTZ: Method=POST   absolutePath=/api/jobs/:id/resume      --> handlerName=github.com/Colin4k1024/Aetheris/v2/internal/api/http.(*Handler).JobResume-fm (num=6 handlers)
2026/05/26 09:56:48.746981 engine.go:702: [Debug] HERTZ: Method=POST   absolutePath=/api/jobs/:id/signal      --> handlerName=github.com/Colin4k1024/Aetheris/v2/internal/api/http.(*Handler).JobSignal-fm (num=6 handlers)
2026/05/26 09:56:48.746983 engine.go:702: [Debug] HERTZ: Method=POST   absolutePath=/api/jobs/:id/message     --> handlerName=github.com/Colin4k1024/Aetheris/v2/internal/api/http.(*Handler).JobMessage-fm (num=6 handlers)
2026/05/26 09:56:48.746985 engine.go:702: [Debug] HERTZ: Method=GET    absolutePath=/api/jobs/:id/events      --> handlerName=github.com/Colin4k1024/Aetheris/v2/internal/api/http.(*Handler).GetJobEvents-fm (num=6 handlers)
2026/05/26 09:56:48.746986 engine.go:702: [Debug] HERTZ: Method=GET    absolutePath=/api/jobs/:id/replay      --> handlerName=github.com/Colin4k1024/Aetheris/v2/internal/api/http.(*Handler).GetJobReplay-fm (num=6 handlers)
2026/05/26 09:56:48.746988 engine.go:702: [Debug] HERTZ: Method=GET    absolutePath=/api/jobs/:id/verify      --> handlerName=github.com/Colin4k1024/Aetheris/v2/internal/api/http.(*Handler).GetJobVerify-fm (num=6 handlers)
2026/05/26 09:56:48.746990 engine.go:702: [Debug] HERTZ: Method=GET    absolutePath=/api/jobs/:id/trace       --> handlerName=github.com/Colin4k1024/Aetheris/v2/internal/api/http.(*Handler).GetJobTrace-fm (num=6 handlers)
2026/05/26 09:56:48.746992 engine.go:702: [Debug] HERTZ: Method=GET    absolutePath=/api/jobs/:id/trace/cognition --> handlerName=github.com/Colin4k1024/Aetheris/v2/internal/api/http.(*Handler).GetJobCognitionTrace-fm (num=6 handlers)
2026/05/26 09:56:48.746994 engine.go:702: [Debug] HERTZ: Method=GET    absolutePath=/api/jobs/:id/nodes/:node_id --> handlerName=github.com/Colin4k1024/Aetheris/v2/internal/api/http.(*Handler).GetJobNode-fm (num=6 handlers)
2026/05/26 09:56:48.746997 engine.go:702: [Debug] HERTZ: Method=GET    absolutePath=/api/jobs/:id/trace/page  --> handlerName=github.com/Colin4k1024/Aetheris/v2/internal/api/http.(*Handler).GetJobTracePage-fm (num=6 handlers)
2026/05/26 09:56:48.746999 engine.go:702: [Debug] HERTZ: Method=POST   absolutePath=/api/jobs/:id/export      --> handlerName=github.com/Colin4k1024/Aetheris/v2/internal/api/http.(*Handler).ExportJobForensics-fm (num=6 handlers)
2026/05/26 09:56:48.747000 engine.go:702: [Debug] HERTZ: Method=GET    absolutePath=/api/jobs/:id/evidence-graph --> handlerName=github.com/Colin4k1024/Aetheris/v2/internal/api/http.(*Handler).GetJobEvidenceGraph-fm (num=6 handlers)
2026/05/26 09:56:48.747003 engine.go:702: [Debug] HERTZ: Method=GET    absolutePath=/api/jobs/:id/audit-log   --> handlerName=github.com/Colin4k1024/Aetheris/v2/internal/api/http.(*Handler).GetJobAuditLog-fm (num=6 handlers)
2026/05/26 09:56:48.747004 engine.go:702: [Debug] HERTZ: Method=POST   absolutePath=/api/approvals            --> handlerName=github.com/Colin4k1024/Aetheris/v2/internal/api/http.(*Handler).CreateApproval-fm (num=6 handlers)
2026/05/26 09:56:48.747006 engine.go:702: [Debug] HERTZ: Method=GET    absolutePath=/api/approvals            --> handlerName=github.com/Colin4k1024/Aetheris/v2/internal/api/http.(*Handler).ListApprovals-fm (num=6 handlers)
2026/05/26 09:56:48.747008 engine.go:702: [Debug] HERTZ: Method=GET    absolutePath=/api/approvals/:id        --> handlerName=github.com/Colin4k1024/Aetheris/v2/internal/api/http.(*Handler).GetApproval-fm (num=6 handlers)
2026/05/26 09:56:48.747010 engine.go:702: [Debug] HERTZ: Method=POST   absolutePath=/api/approvals/:id/approve --> handlerName=github.com/Colin4k1024/Aetheris/v2/internal/api/http.(*Handler).ApproveApproval-fm (num=6 handlers)
2026/05/26 09:56:48.747012 engine.go:702: [Debug] HERTZ: Method=POST   absolutePath=/api/approvals/:id/reject --> handlerName=github.com/Colin4k1024/Aetheris/v2/internal/api/http.(*Handler).RejectApproval-fm (num=6 handlers)
2026/05/26 09:56:48.747015 engine.go:702: [Debug] HERTZ: Method=POST   absolutePath=/api/forensics/query      --> handlerName=github.com/Colin4k1024/Aetheris/v2/internal/api/http.(*Handler).ForensicsQuery-fm (num=6 handlers)
2026/05/26 09:56:48.747017 engine.go:702: [Debug] HERTZ: Method=POST   absolutePath=/api/forensics/batch-export --> handlerName=github.com/Colin4k1024/Aetheris/v2/internal/api/http.(*Handler).ForensicsBatchExport-fm (num=6 handlers)
2026/05/26 09:56:48.747019 engine.go:702: [Debug] HERTZ: Method=GET    absolutePath=/api/forensics/export-status/:task_id --> handlerName=github.com/Colin4k1024/Aetheris/v2/internal/api/http.(*Handler).ForensicsExportStatus-fm (num=6 handlers)
2026/05/26 09:56:48.747022 engine.go:702: [Debug] HERTZ: Method=GET    absolutePath=/api/forensics/consistency/:job_id --> handlerName=github.com/Colin4k1024/Aetheris/v2/internal/api/http.(*Handler).ForensicsConsistencyCheck-fm (num=6 handlers)
2026/05/26 09:56:48.747026 engine.go:702: [Debug] HERTZ: Method=POST   absolutePath=/api/forensics/ai/detect-anomalies --> handlerName=github.com/Colin4k1024/Aetheris/v2/internal/api/http.(*Handler).AIForensicsDetectAnomalies-fm (num=6 handlers)
2026/05/26 09:56:48.747028 engine.go:702: [Debug] HERTZ: Method=GET    absolutePath=/api/compliance/templates --> handlerName=github.com/Colin4k1024/Aetheris/v2/internal/api/http.(*Handler).ComplianceTemplates-fm (num=6 handlers)
2026/05/26 09:56:48.747029 engine.go:702: [Debug] HERTZ: Method=POST   absolutePath=/api/compliance/apply     --> handlerName=github.com/Colin4k1024/Aetheris/v2/internal/api/http.(*Handler).ComplianceApply-fm (num=6 handlers)
2026/05/26 09:56:48.747031 engine.go:702: [Debug] HERTZ: Method=POST   absolutePath=/api/compliance/report    --> handlerName=github.com/Colin4k1024/Aetheris/v2/internal/api/http.(*Handler).ComplianceReport-fm (num=6 handlers)
2026/05/26 09:56:48.747033 engine.go:702: [Debug] HERTZ: Method=GET    absolutePath=/api/tools/               --> handlerName=github.com/Colin4k1024/Aetheris/v2/internal/api/http.(*Handler).ListTools-fm (num=6 handlers)
2026/05/26 09:56:48.747038 engine.go:702: [Debug] HERTZ: Method=GET    absolutePath=/api/tools/:name          --> handlerName=github.com/Colin4k1024/Aetheris/v2/internal/api/http.(*Handler).GetTool-fm (num=6 handlers)
2026/05/26 09:56:48.747041 engine.go:702: [Debug] HERTZ: Method=GET    absolutePath=/api/system/status        --> handlerName=github.com/Colin4k1024/Aetheris/v2/internal/api/http.(*Handler).SystemStatus-fm (num=6 handlers)
2026/05/26 09:56:48.747042 engine.go:702: [Debug] HERTZ: Method=GET    absolutePath=/api/system/metrics       --> handlerName=github.com/Colin4k1024/Aetheris/v2/internal/api/http.(*Handler).SystemMetrics-fm (num=6 handlers)
2026/05/26 09:56:48.747044 engine.go:702: [Debug] HERTZ: Method=GET    absolutePath=/api/system/workers       --> handlerName=github.com/Colin4k1024/Aetheris/v2/internal/api/http.(*Handler).SystemWorkers-fm (num=6 handlers)
2026/05/26 09:56:48.747046 engine.go:702: [Debug] HERTZ: Method=GET    absolutePath=/api/observability/summary --> handlerName=github.com/Colin4k1024/Aetheris/v2/internal/api/http.(*Handler).GetObservabilitySummary-fm (num=6 handlers)
2026/05/26 09:56:48.747048 engine.go:702: [Debug] HERTZ: Method=GET    absolutePath=/api/observability/stuck  --> handlerName=github.com/Colin4k1024/Aetheris/v2/internal/api/http.(*Handler).GetObservabilityStuck-fm (num=6 handlers)
2026/05/26 09:56:48.747050 engine.go:702: [Debug] HERTZ: Method=GET    absolutePath=/api/trace/overview/page  --> handlerName=github.com/Colin4k1024/Aetheris/v2/internal/api/http.(*Handler).GetTraceOverviewPage-fm (num=6 handlers)
2026/05/26 09:56:48.747052 engine.go:702: [Debug] HERTZ: Method=GET    absolutePath=/api/rbac/role            --> handlerName=github.com/Colin4k1024/Aetheris/v2/internal/api/http.(*Handler).GetUserRole-fm (num=6 handlers)
2026/05/26 09:56:48.747054 engine.go:702: [Debug] HERTZ: Method=POST   absolutePath=/api/rbac/role            --> handlerName=github.com/Colin4k1024/Aetheris/v2/internal/api/http.(*Handler).AssignRole-fm (num=6 handlers)
2026/05/26 09:56:48.747056 engine.go:702: [Debug] HERTZ: Method=POST   absolutePath=/api/rbac/check           --> handlerName=github.com/Colin4k1024/Aetheris/v2/internal/api/http.(*Handler).CheckPermission-fm (num=6 handlers)
2026/05/26 09:56:48.747225 middleware.go:413: [Info] POST /api/forensics/query 0.0.0.0 503 162.458µs
2026/05/26 09:56:48.747238 middleware.go:413: [Info] GET /api/jobs/job_x/evidence-graph 0.0.0.0 503 3.167µs
2026/05/26 09:56:48.747282 middleware.go:413: [Info] GET /api/compliance/templates 0.0.0.0 200 37.167µs
--- PASS: TestRouter_ForensicsRoutesEnabled (0.00s)
PASS
ok  	github.com/Colin4k1024/Aetheris/v2/internal/api/http	0.649s
```
