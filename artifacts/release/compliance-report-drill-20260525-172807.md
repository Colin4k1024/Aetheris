# Compliance Report Release Drill

- Timestamp: 20260525-172807
- Overall: PASS

## Suites

- pkg/compliance (PASS)
- internal/api/http (PASS)

## pkg/compliance output

```text
=== RUN   TestFrameworkFactory_CreateFramework_SOC2
--- PASS: TestFrameworkFactory_CreateFramework_SOC2 (0.00s)
=== RUN   TestFrameworkFactory_CreateFramework_GDPR
--- PASS: TestFrameworkFactory_CreateFramework_GDPR (0.00s)
=== RUN   TestFrameworkFactory_CreateFramework_HIPAA
--- PASS: TestFrameworkFactory_CreateFramework_HIPAA (0.00s)
=== RUN   TestFrameworkFactory_CreateFramework_Unsupported
--- PASS: TestFrameworkFactory_CreateFramework_Unsupported (0.00s)
=== RUN   TestGenerateReport_DefaultControlsExposeUnsupportedScope
--- PASS: TestGenerateReport_DefaultControlsExposeUnsupportedScope (0.00s)
=== RUN   TestGenerateReport_WithMetrics
--- PASS: TestGenerateReport_WithMetrics (0.00s)
=== RUN   TestGenerateReport_WithSignedEvidenceBinding
--- PASS: TestGenerateReport_WithSignedEvidenceBinding (0.00s)
=== RUN   TestGenerateReport_InvalidTimeRange
--- PASS: TestGenerateReport_InvalidTimeRange (0.00s)
=== RUN   TestGetTemplate
--- PASS: TestGetTemplate (0.00s)
=== RUN   TestListTemplates
--- PASS: TestListTemplates (0.00s)
PASS
ok  	github.com/Colin4k1024/Aetheris/v2/pkg/compliance	0.153s
```

## internal/api/http output

```text
=== RUN   TestComplianceReport_BindsSignedEvidenceAndExposesUnsupportedControls
2026/05/25 17:28:11.822464 engine.go:702: [Debug] HERTZ: Method=POST   absolutePath=/api/compliance/report    --> handlerName=github.com/Colin4k1024/Aetheris/v2/internal/api/http.(*Handler).ComplianceReport-fm (num=2 handlers)
--- PASS: TestComplianceReport_BindsSignedEvidenceAndExposesUnsupportedControls (0.00s)
=== RUN   TestComplianceReport_RequiresSignedEvidenceVerification
2026/05/25 17:28:11.823862 engine.go:702: [Debug] HERTZ: Method=POST   absolutePath=/api/compliance/report    --> handlerName=github.com/Colin4k1024/Aetheris/v2/internal/api/http.(*Handler).ComplianceReport-fm (num=2 handlers)
--- PASS: TestComplianceReport_RequiresSignedEvidenceVerification (0.00s)
=== RUN   TestRouter_ForensicsRoutesDisabledByDefault
2026/05/25 17:28:11.824067 engine.go:702: [Debug] HERTZ: Method=GET    absolutePath=/                         --> handlerName=github.com/Colin4k1024/Aetheris/v2/internal/api/http.(*Handler).HomePage-fm (num=4 handlers)
2026/05/25 17:28:11.824071 engine.go:702: [Debug] HERTZ: Method=GET    absolutePath=/metrics                  --> handlerName=github.com/Colin4k1024/Aetheris/v2/internal/api/http.(*Handler).SystemMetrics-fm (num=4 handlers)
2026/05/25 17:28:11.824074 engine.go:702: [Debug] HERTZ: Method=GET    absolutePath=/api/health               --> handlerName=github.com/Colin4k1024/Aetheris/v2/internal/api/http.(*Handler).HealthCheck-fm (num=4 handlers)
2026/05/25 17:28:11.824083 engine.go:702: [Debug] HERTZ: Method=POST   absolutePath=/api/documents/upload     --> handlerName=github.com/Colin4k1024/Aetheris/v2/internal/api/http.(*Handler).UploadDocument-fm (num=6 handlers)
2026/05/25 17:28:11.824095 engine.go:702: [Debug] HERTZ: Method=POST   absolutePath=/api/documents/upload/async --> handlerName=github.com/Colin4k1024/Aetheris/v2/internal/api/http.(*Handler).UploadDocumentAsync-fm (num=6 handlers)
2026/05/25 17:28:11.824101 engine.go:702: [Debug] HERTZ: Method=GET    absolutePath=/api/documents/upload/status/:task_id --> handlerName=github.com/Colin4k1024/Aetheris/v2/internal/api/http.(*Handler).UploadStatus-fm (num=6 handlers)
2026/05/25 17:28:11.824104 engine.go:702: [Debug] HERTZ: Method=GET    absolutePath=/api/documents/           --> handlerName=github.com/Colin4k1024/Aetheris/v2/internal/api/http.(*Handler).ListDocuments-fm (num=6 handlers)
2026/05/25 17:28:11.824106 engine.go:702: [Debug] HERTZ: Method=GET    absolutePath=/api/documents/:id        --> handlerName=github.com/Colin4k1024/Aetheris/v2/internal/api/http.(*Handler).GetDocument-fm (num=6 handlers)
2026/05/25 17:28:11.824108 engine.go:702: [Debug] HERTZ: Method=DELETE absolutePath=/api/documents/:id        --> handlerName=github.com/Colin4k1024/Aetheris/v2/internal/api/http.(*Handler).DeleteDocument-fm (num=6 handlers)
2026/05/25 17:28:11.824111 engine.go:702: [Debug] HERTZ: Method=GET    absolutePath=/api/knowledge/collections --> handlerName=github.com/Colin4k1024/Aetheris/v2/internal/api/http.(*Handler).ListCollections-fm (num=6 handlers)
2026/05/25 17:28:11.824112 engine.go:702: [Debug] HERTZ: Method=POST   absolutePath=/api/knowledge/collections --> handlerName=github.com/Colin4k1024/Aetheris/v2/internal/api/http.(*Handler).CreateCollection-fm (num=6 handlers)
2026/05/25 17:28:11.824114 engine.go:702: [Debug] HERTZ: Method=DELETE absolutePath=/api/knowledge/collections/:id --> handlerName=github.com/Colin4k1024/Aetheris/v2/internal/api/http.(*Handler).DeleteCollection-fm (num=6 handlers)
2026/05/25 17:28:11.824116 engine.go:702: [Debug] HERTZ: Method=POST   absolutePath=/api/runs                 --> handlerName=github.com/Colin4k1024/Aetheris/v2/internal/api/http.(*Handler).CreateRun-fm (num=6 handlers)
2026/05/25 17:28:11.824118 engine.go:702: [Debug] HERTZ: Method=POST   absolutePath=/api/runs/                --> handlerName=github.com/Colin4k1024/Aetheris/v2/internal/api/http.(*Handler).CreateRun-fm (num=6 handlers)
2026/05/25 17:28:11.824122 engine.go:702: [Debug] HERTZ: Method=GET    absolutePath=/api/runs/:id             --> handlerName=github.com/Colin4k1024/Aetheris/v2/internal/api/http.(*Handler).GetRun-fm (num=6 handlers)
2026/05/25 17:28:11.824124 engine.go:702: [Debug] HERTZ: Method=GET    absolutePath=/api/runs/:id/events      --> handlerName=github.com/Colin4k1024/Aetheris/v2/internal/api/http.(*Handler).GetRunEvents-fm (num=6 handlers)
2026/05/25 17:28:11.824126 engine.go:702: [Debug] HERTZ: Method=POST   absolutePath=/api/runs/:id/tool-calls  --> handlerName=github.com/Colin4k1024/Aetheris/v2/internal/api/http.(*Handler).UpsertToolCall-fm (num=6 handlers)
2026/05/25 17:28:11.824128 engine.go:702: [Debug] HERTZ: Method=POST   absolutePath=/api/runs/:id/pause       --> handlerName=github.com/Colin4k1024/Aetheris/v2/internal/api/http.(*Handler).PauseRun-fm (num=6 handlers)
2026/05/25 17:28:11.824130 engine.go:702: [Debug] HERTZ: Method=POST   absolutePath=/api/runs/:id/resume      --> handlerName=github.com/Colin4k1024/Aetheris/v2/internal/api/http.(*Handler).ResumeRun-fm (num=6 handlers)
2026/05/25 17:28:11.824132 engine.go:702: [Debug] HERTZ: Method=POST   absolutePath=/api/runs/:id/human-decisions --> handlerName=github.com/Colin4k1024/Aetheris/v2/internal/api/http.(*Handler).InjectHumanDecision-fm (num=6 handlers)
2026/05/25 17:28:11.824134 engine.go:702: [Debug] HERTZ: Method=POST   absolutePath=/api/query/               --> handlerName=github.com/Colin4k1024/Aetheris/v2/internal/api/http.(*Handler).Query-fm (num=7 handlers)
2026/05/25 17:28:11.824136 engine.go:702: [Debug] HERTZ: Method=POST   absolutePath=/api/query/batch          --> handlerName=github.com/Colin4k1024/Aetheris/v2/internal/api/http.(*Handler).BatchQuery-fm (num=7 handlers)
2026/05/25 17:28:11.824140 engine.go:702: [Debug] HERTZ: Method=POST   absolutePath=/api/agent/run            --> handlerName=github.com/Colin4k1024/Aetheris/v2/internal/api/http.(*Handler).AgentRun-fm (num=7 handlers)
2026/05/25 17:28:11.824142 engine.go:702: [Debug] HERTZ: Method=POST   absolutePath=/api/agent/resume         --> handlerName=github.com/Colin4k1024/Aetheris/v2/internal/api/http.(*Handler).AgentResumeCheckpoint-fm (num=7 handlers)
2026/05/25 17:28:11.824144 engine.go:702: [Debug] HERTZ: Method=POST   absolutePath=/api/agent/stream         --> handlerName=github.com/Colin4k1024/Aetheris/v2/internal/api/http.(*Handler).AgentStream-fm (num=7 handlers)
2026/05/25 17:28:11.824146 engine.go:702: [Debug] HERTZ: Method=POST   absolutePath=/api/agents/:id/message   --> handlerName=github.com/Colin4k1024/Aetheris/v2/internal/api/http.(*Handler).AgentMessage-fm (num=7 handlers)
2026/05/25 17:28:11.824148 engine.go:702: [Debug] HERTZ: Method=GET    absolutePath=/api/agents/:id/state     --> handlerName=github.com/Colin4k1024/Aetheris/v2/internal/api/http.(*Handler).AgentState-fm (num=7 handlers)
2026/05/25 17:28:11.824150 engine.go:702: [Debug] HERTZ: Method=POST   absolutePath=/api/agents/:id/resume    --> handlerName=github.com/Colin4k1024/Aetheris/v2/internal/api/http.(*Handler).AgentResume-fm (num=7 handlers)
2026/05/25 17:28:11.824152 engine.go:702: [Debug] HERTZ: Method=POST   absolutePath=/api/agents/:id/stop      --> handlerName=github.com/Colin4k1024/Aetheris/v2/internal/api/http.(*Handler).AgentStop-fm (num=7 handlers)
2026/05/25 17:28:11.824154 engine.go:702: [Debug] HERTZ: Method=GET    absolutePath=/api/agents/:id/jobs/:job_id --> handlerName=github.com/Colin4k1024/Aetheris/v2/internal/api/http.(*Handler).GetAgentJob-fm (num=7 handlers)
2026/05/25 17:28:11.824156 engine.go:702: [Debug] HERTZ: Method=GET    absolutePath=/api/agents/:id/jobs      --> handlerName=github.com/Colin4k1024/Aetheris/v2/internal/api/http.(*Handler).ListAgentJobs-fm (num=7 handlers)
2026/05/25 17:28:11.824158 engine.go:702: [Debug] HERTZ: Method=GET    absolutePath=/api/jobs/:id             --> handlerName=github.com/Colin4k1024/Aetheris/v2/internal/api/http.(*Handler).GetJob-fm (num=6 handlers)
2026/05/25 17:28:11.824160 engine.go:702: [Debug] HERTZ: Method=POST   absolutePath=/api/jobs/:id/stop        --> handlerName=github.com/Colin4k1024/Aetheris/v2/internal/api/http.(*Handler).JobStop-fm (num=6 handlers)
2026/05/25 17:28:11.824162 engine.go:702: [Debug] HERTZ: Method=POST   absolutePath=/api/jobs/:id/pause       --> handlerName=github.com/Colin4k1024/Aetheris/v2/internal/api/http.(*Handler).JobPause-fm (num=6 handlers)
2026/05/25 17:28:11.824164 engine.go:702: [Debug] HERTZ: Method=POST   absolutePath=/api/jobs/:id/resume      --> handlerName=github.com/Colin4k1024/Aetheris/v2/internal/api/http.(*Handler).JobResume-fm (num=6 handlers)
2026/05/25 17:28:11.824166 engine.go:702: [Debug] HERTZ: Method=POST   absolutePath=/api/jobs/:id/signal      --> handlerName=github.com/Colin4k1024/Aetheris/v2/internal/api/http.(*Handler).JobSignal-fm (num=6 handlers)
2026/05/25 17:28:11.824168 engine.go:702: [Debug] HERTZ: Method=POST   absolutePath=/api/jobs/:id/message     --> handlerName=github.com/Colin4k1024/Aetheris/v2/internal/api/http.(*Handler).JobMessage-fm (num=6 handlers)
2026/05/25 17:28:11.824170 engine.go:702: [Debug] HERTZ: Method=GET    absolutePath=/api/jobs/:id/events      --> handlerName=github.com/Colin4k1024/Aetheris/v2/internal/api/http.(*Handler).GetJobEvents-fm (num=6 handlers)
2026/05/25 17:28:11.824172 engine.go:702: [Debug] HERTZ: Method=GET    absolutePath=/api/jobs/:id/replay      --> handlerName=github.com/Colin4k1024/Aetheris/v2/internal/api/http.(*Handler).GetJobReplay-fm (num=6 handlers)
2026/05/25 17:28:11.824174 engine.go:702: [Debug] HERTZ: Method=GET    absolutePath=/api/jobs/:id/verify      --> handlerName=github.com/Colin4k1024/Aetheris/v2/internal/api/http.(*Handler).GetJobVerify-fm (num=6 handlers)
2026/05/25 17:28:11.824175 engine.go:702: [Debug] HERTZ: Method=GET    absolutePath=/api/jobs/:id/trace       --> handlerName=github.com/Colin4k1024/Aetheris/v2/internal/api/http.(*Handler).GetJobTrace-fm (num=6 handlers)
2026/05/25 17:28:11.824177 engine.go:702: [Debug] HERTZ: Method=GET    absolutePath=/api/jobs/:id/trace/cognition --> handlerName=github.com/Colin4k1024/Aetheris/v2/internal/api/http.(*Handler).GetJobCognitionTrace-fm (num=6 handlers)
2026/05/25 17:28:11.824182 engine.go:702: [Debug] HERTZ: Method=GET    absolutePath=/api/jobs/:id/nodes/:node_id --> handlerName=github.com/Colin4k1024/Aetheris/v2/internal/api/http.(*Handler).GetJobNode-fm (num=6 handlers)
2026/05/25 17:28:11.824185 engine.go:702: [Debug] HERTZ: Method=GET    absolutePath=/api/jobs/:id/trace/page  --> handlerName=github.com/Colin4k1024/Aetheris/v2/internal/api/http.(*Handler).GetJobTracePage-fm (num=6 handlers)
2026/05/25 17:28:11.824186 engine.go:702: [Debug] HERTZ: Method=POST   absolutePath=/api/jobs/:id/export      --> handlerName=github.com/Colin4k1024/Aetheris/v2/internal/api/http.(*Handler).ExportJobForensics-fm (num=6 handlers)
2026/05/25 17:28:11.824188 engine.go:702: [Debug] HERTZ: Method=POST   absolutePath=/api/approvals            --> handlerName=github.com/Colin4k1024/Aetheris/v2/internal/api/http.(*Handler).CreateApproval-fm (num=6 handlers)
2026/05/25 17:28:11.824190 engine.go:702: [Debug] HERTZ: Method=GET    absolutePath=/api/approvals            --> handlerName=github.com/Colin4k1024/Aetheris/v2/internal/api/http.(*Handler).ListApprovals-fm (num=6 handlers)
2026/05/25 17:28:11.824192 engine.go:702: [Debug] HERTZ: Method=GET    absolutePath=/api/approvals/:id        --> handlerName=github.com/Colin4k1024/Aetheris/v2/internal/api/http.(*Handler).GetApproval-fm (num=6 handlers)
2026/05/25 17:28:11.824194 engine.go:702: [Debug] HERTZ: Method=POST   absolutePath=/api/approvals/:id/approve --> handlerName=github.com/Colin4k1024/Aetheris/v2/internal/api/http.(*Handler).ApproveApproval-fm (num=6 handlers)
2026/05/25 17:28:11.824196 engine.go:702: [Debug] HERTZ: Method=POST   absolutePath=/api/approvals/:id/reject --> handlerName=github.com/Colin4k1024/Aetheris/v2/internal/api/http.(*Handler).RejectApproval-fm (num=6 handlers)
2026/05/25 17:28:11.824198 engine.go:702: [Debug] HERTZ: Method=GET    absolutePath=/api/tools/               --> handlerName=github.com/Colin4k1024/Aetheris/v2/internal/api/http.(*Handler).ListTools-fm (num=6 handlers)
2026/05/25 17:28:11.824200 engine.go:702: [Debug] HERTZ: Method=GET    absolutePath=/api/tools/:name          --> handlerName=github.com/Colin4k1024/Aetheris/v2/internal/api/http.(*Handler).GetTool-fm (num=6 handlers)
2026/05/25 17:28:11.824202 engine.go:702: [Debug] HERTZ: Method=GET    absolutePath=/api/system/status        --> handlerName=github.com/Colin4k1024/Aetheris/v2/internal/api/http.(*Handler).SystemStatus-fm (num=6 handlers)
2026/05/25 17:28:11.824204 engine.go:702: [Debug] HERTZ: Method=GET    absolutePath=/api/system/metrics       --> handlerName=github.com/Colin4k1024/Aetheris/v2/internal/api/http.(*Handler).SystemMetrics-fm (num=6 handlers)
2026/05/25 17:28:11.824206 engine.go:702: [Debug] HERTZ: Method=GET    absolutePath=/api/system/workers       --> handlerName=github.com/Colin4k1024/Aetheris/v2/internal/api/http.(*Handler).SystemWorkers-fm (num=6 handlers)
2026/05/25 17:28:11.824208 engine.go:702: [Debug] HERTZ: Method=GET    absolutePath=/api/observability/summary --> handlerName=github.com/Colin4k1024/Aetheris/v2/internal/api/http.(*Handler).GetObservabilitySummary-fm (num=6 handlers)
2026/05/25 17:28:11.824210 engine.go:702: [Debug] HERTZ: Method=GET    absolutePath=/api/observability/stuck  --> handlerName=github.com/Colin4k1024/Aetheris/v2/internal/api/http.(*Handler).GetObservabilityStuck-fm (num=6 handlers)
2026/05/25 17:28:11.824212 engine.go:702: [Debug] HERTZ: Method=GET    absolutePath=/api/trace/overview/page  --> handlerName=github.com/Colin4k1024/Aetheris/v2/internal/api/http.(*Handler).GetTraceOverviewPage-fm (num=6 handlers)
2026/05/25 17:28:11.824214 engine.go:702: [Debug] HERTZ: Method=GET    absolutePath=/api/rbac/role            --> handlerName=github.com/Colin4k1024/Aetheris/v2/internal/api/http.(*Handler).GetUserRole-fm (num=6 handlers)
2026/05/25 17:28:11.824216 engine.go:702: [Debug] HERTZ: Method=POST   absolutePath=/api/rbac/role            --> handlerName=github.com/Colin4k1024/Aetheris/v2/internal/api/http.(*Handler).AssignRole-fm (num=6 handlers)
2026/05/25 17:28:11.824218 engine.go:702: [Debug] HERTZ: Method=POST   absolutePath=/api/rbac/check           --> handlerName=github.com/Colin4k1024/Aetheris/v2/internal/api/http.(*Handler).CheckPermission-fm (num=6 handlers)
2026/05/25 17:28:11.824261 middleware.go:413: [Info] POST /api/forensics/query 0.0.0.0 404 19µs
2026/05/25 17:28:11.824360 middleware.go:413: [Info] GET /api/jobs/job_x/evidence-graph 0.0.0.0 404 1.25µs
2026/05/25 17:28:11.824369 middleware.go:413: [Info] GET /api/jobs/job_x/audit-log 0.0.0.0 404 958ns
2026/05/25 17:28:11.824377 middleware.go:413: [Info] POST /api/forensics/ai/detect-anomalies 0.0.0.0 404 1µs
2026/05/25 17:28:11.824383 middleware.go:413: [Info] GET /api/compliance/templates 0.0.0.0 404 833ns
--- PASS: TestRouter_ForensicsRoutesDisabledByDefault (0.00s)
=== RUN   TestRouter_ForensicsRoutesEnabled
2026/05/25 17:28:11.824410 engine.go:702: [Debug] HERTZ: Method=GET    absolutePath=/                         --> handlerName=github.com/Colin4k1024/Aetheris/v2/internal/api/http.(*Handler).HomePage-fm (num=4 handlers)
2026/05/25 17:28:11.824413 engine.go:702: [Debug] HERTZ: Method=GET    absolutePath=/metrics                  --> handlerName=github.com/Colin4k1024/Aetheris/v2/internal/api/http.(*Handler).SystemMetrics-fm (num=4 handlers)
2026/05/25 17:28:11.824416 engine.go:702: [Debug] HERTZ: Method=GET    absolutePath=/api/health               --> handlerName=github.com/Colin4k1024/Aetheris/v2/internal/api/http.(*Handler).HealthCheck-fm (num=4 handlers)
2026/05/25 17:28:11.824418 engine.go:702: [Debug] HERTZ: Method=POST   absolutePath=/api/documents/upload     --> handlerName=github.com/Colin4k1024/Aetheris/v2/internal/api/http.(*Handler).UploadDocument-fm (num=6 handlers)
2026/05/25 17:28:11.824420 engine.go:702: [Debug] HERTZ: Method=POST   absolutePath=/api/documents/upload/async --> handlerName=github.com/Colin4k1024/Aetheris/v2/internal/api/http.(*Handler).UploadDocumentAsync-fm (num=6 handlers)
2026/05/25 17:28:11.824423 engine.go:702: [Debug] HERTZ: Method=GET    absolutePath=/api/documents/upload/status/:task_id --> handlerName=github.com/Colin4k1024/Aetheris/v2/internal/api/http.(*Handler).UploadStatus-fm (num=6 handlers)
2026/05/25 17:28:11.824425 engine.go:702: [Debug] HERTZ: Method=GET    absolutePath=/api/documents/           --> handlerName=github.com/Colin4k1024/Aetheris/v2/internal/api/http.(*Handler).ListDocuments-fm (num=6 handlers)
2026/05/25 17:28:11.824427 engine.go:702: [Debug] HERTZ: Method=GET    absolutePath=/api/documents/:id        --> handlerName=github.com/Colin4k1024/Aetheris/v2/internal/api/http.(*Handler).GetDocument-fm (num=6 handlers)
2026/05/25 17:28:11.824428 engine.go:702: [Debug] HERTZ: Method=DELETE absolutePath=/api/documents/:id        --> handlerName=github.com/Colin4k1024/Aetheris/v2/internal/api/http.(*Handler).DeleteDocument-fm (num=6 handlers)
2026/05/25 17:28:11.824430 engine.go:702: [Debug] HERTZ: Method=GET    absolutePath=/api/knowledge/collections --> handlerName=github.com/Colin4k1024/Aetheris/v2/internal/api/http.(*Handler).ListCollections-fm (num=6 handlers)
2026/05/25 17:28:11.824432 engine.go:702: [Debug] HERTZ: Method=POST   absolutePath=/api/knowledge/collections --> handlerName=github.com/Colin4k1024/Aetheris/v2/internal/api/http.(*Handler).CreateCollection-fm (num=6 handlers)
2026/05/25 17:28:11.824434 engine.go:702: [Debug] HERTZ: Method=DELETE absolutePath=/api/knowledge/collections/:id --> handlerName=github.com/Colin4k1024/Aetheris/v2/internal/api/http.(*Handler).DeleteCollection-fm (num=6 handlers)
2026/05/25 17:28:11.824436 engine.go:702: [Debug] HERTZ: Method=POST   absolutePath=/api/runs                 --> handlerName=github.com/Colin4k1024/Aetheris/v2/internal/api/http.(*Handler).CreateRun-fm (num=6 handlers)
2026/05/25 17:28:11.824438 engine.go:702: [Debug] HERTZ: Method=POST   absolutePath=/api/runs/                --> handlerName=github.com/Colin4k1024/Aetheris/v2/internal/api/http.(*Handler).CreateRun-fm (num=6 handlers)
2026/05/25 17:28:11.824440 engine.go:702: [Debug] HERTZ: Method=GET    absolutePath=/api/runs/:id             --> handlerName=github.com/Colin4k1024/Aetheris/v2/internal/api/http.(*Handler).GetRun-fm (num=6 handlers)
2026/05/25 17:28:11.824442 engine.go:702: [Debug] HERTZ: Method=GET    absolutePath=/api/runs/:id/events      --> handlerName=github.com/Colin4k1024/Aetheris/v2/internal/api/http.(*Handler).GetRunEvents-fm (num=6 handlers)
2026/05/25 17:28:11.824444 engine.go:702: [Debug] HERTZ: Method=POST   absolutePath=/api/runs/:id/tool-calls  --> handlerName=github.com/Colin4k1024/Aetheris/v2/internal/api/http.(*Handler).UpsertToolCall-fm (num=6 handlers)
2026/05/25 17:28:11.824446 engine.go:702: [Debug] HERTZ: Method=POST   absolutePath=/api/runs/:id/pause       --> handlerName=github.com/Colin4k1024/Aetheris/v2/internal/api/http.(*Handler).PauseRun-fm (num=6 handlers)
2026/05/25 17:28:11.824448 engine.go:702: [Debug] HERTZ: Method=POST   absolutePath=/api/runs/:id/resume      --> handlerName=github.com/Colin4k1024/Aetheris/v2/internal/api/http.(*Handler).ResumeRun-fm (num=6 handlers)
2026/05/25 17:28:11.824452 engine.go:702: [Debug] HERTZ: Method=POST   absolutePath=/api/runs/:id/human-decisions --> handlerName=github.com/Colin4k1024/Aetheris/v2/internal/api/http.(*Handler).InjectHumanDecision-fm (num=6 handlers)
2026/05/25 17:28:11.824455 engine.go:702: [Debug] HERTZ: Method=POST   absolutePath=/api/query/               --> handlerName=github.com/Colin4k1024/Aetheris/v2/internal/api/http.(*Handler).Query-fm (num=7 handlers)
2026/05/25 17:28:11.824457 engine.go:702: [Debug] HERTZ: Method=POST   absolutePath=/api/query/batch          --> handlerName=github.com/Colin4k1024/Aetheris/v2/internal/api/http.(*Handler).BatchQuery-fm (num=7 handlers)
2026/05/25 17:28:11.824459 engine.go:702: [Debug] HERTZ: Method=POST   absolutePath=/api/agent/run            --> handlerName=github.com/Colin4k1024/Aetheris/v2/internal/api/http.(*Handler).AgentRun-fm (num=7 handlers)
2026/05/25 17:28:11.824460 engine.go:702: [Debug] HERTZ: Method=POST   absolutePath=/api/agent/resume         --> handlerName=github.com/Colin4k1024/Aetheris/v2/internal/api/http.(*Handler).AgentResumeCheckpoint-fm (num=7 handlers)
2026/05/25 17:28:11.824479 engine.go:702: [Debug] HERTZ: Method=POST   absolutePath=/api/agent/stream         --> handlerName=github.com/Colin4k1024/Aetheris/v2/internal/api/http.(*Handler).AgentStream-fm (num=7 handlers)
2026/05/25 17:28:11.824482 engine.go:702: [Debug] HERTZ: Method=POST   absolutePath=/api/agents/:id/message   --> handlerName=github.com/Colin4k1024/Aetheris/v2/internal/api/http.(*Handler).AgentMessage-fm (num=7 handlers)
2026/05/25 17:28:11.824485 engine.go:702: [Debug] HERTZ: Method=GET    absolutePath=/api/agents/:id/state     --> handlerName=github.com/Colin4k1024/Aetheris/v2/internal/api/http.(*Handler).AgentState-fm (num=7 handlers)
2026/05/25 17:28:11.824488 engine.go:702: [Debug] HERTZ: Method=POST   absolutePath=/api/agents/:id/resume    --> handlerName=github.com/Colin4k1024/Aetheris/v2/internal/api/http.(*Handler).AgentResume-fm (num=7 handlers)
2026/05/25 17:28:11.824490 engine.go:702: [Debug] HERTZ: Method=POST   absolutePath=/api/agents/:id/stop      --> handlerName=github.com/Colin4k1024/Aetheris/v2/internal/api/http.(*Handler).AgentStop-fm (num=7 handlers)
2026/05/25 17:28:11.824493 engine.go:702: [Debug] HERTZ: Method=GET    absolutePath=/api/agents/:id/jobs/:job_id --> handlerName=github.com/Colin4k1024/Aetheris/v2/internal/api/http.(*Handler).GetAgentJob-fm (num=7 handlers)
2026/05/25 17:28:11.824495 engine.go:702: [Debug] HERTZ: Method=GET    absolutePath=/api/agents/:id/jobs      --> handlerName=github.com/Colin4k1024/Aetheris/v2/internal/api/http.(*Handler).ListAgentJobs-fm (num=7 handlers)
2026/05/25 17:28:11.824497 engine.go:702: [Debug] HERTZ: Method=GET    absolutePath=/api/jobs/:id             --> handlerName=github.com/Colin4k1024/Aetheris/v2/internal/api/http.(*Handler).GetJob-fm (num=6 handlers)
2026/05/25 17:28:11.824500 engine.go:702: [Debug] HERTZ: Method=POST   absolutePath=/api/jobs/:id/stop        --> handlerName=github.com/Colin4k1024/Aetheris/v2/internal/api/http.(*Handler).JobStop-fm (num=6 handlers)
2026/05/25 17:28:11.824502 engine.go:702: [Debug] HERTZ: Method=POST   absolutePath=/api/jobs/:id/pause       --> handlerName=github.com/Colin4k1024/Aetheris/v2/internal/api/http.(*Handler).JobPause-fm (num=6 handlers)
2026/05/25 17:28:11.824504 engine.go:702: [Debug] HERTZ: Method=POST   absolutePath=/api/jobs/:id/resume      --> handlerName=github.com/Colin4k1024/Aetheris/v2/internal/api/http.(*Handler).JobResume-fm (num=6 handlers)
2026/05/25 17:28:11.824506 engine.go:702: [Debug] HERTZ: Method=POST   absolutePath=/api/jobs/:id/signal      --> handlerName=github.com/Colin4k1024/Aetheris/v2/internal/api/http.(*Handler).JobSignal-fm (num=6 handlers)
2026/05/25 17:28:11.824509 engine.go:702: [Debug] HERTZ: Method=POST   absolutePath=/api/jobs/:id/message     --> handlerName=github.com/Colin4k1024/Aetheris/v2/internal/api/http.(*Handler).JobMessage-fm (num=6 handlers)
2026/05/25 17:28:11.824511 engine.go:702: [Debug] HERTZ: Method=GET    absolutePath=/api/jobs/:id/events      --> handlerName=github.com/Colin4k1024/Aetheris/v2/internal/api/http.(*Handler).GetJobEvents-fm (num=6 handlers)
2026/05/25 17:28:11.824513 engine.go:702: [Debug] HERTZ: Method=GET    absolutePath=/api/jobs/:id/replay      --> handlerName=github.com/Colin4k1024/Aetheris/v2/internal/api/http.(*Handler).GetJobReplay-fm (num=6 handlers)
2026/05/25 17:28:11.824515 engine.go:702: [Debug] HERTZ: Method=GET    absolutePath=/api/jobs/:id/verify      --> handlerName=github.com/Colin4k1024/Aetheris/v2/internal/api/http.(*Handler).GetJobVerify-fm (num=6 handlers)
2026/05/25 17:28:11.824517 engine.go:702: [Debug] HERTZ: Method=GET    absolutePath=/api/jobs/:id/trace       --> handlerName=github.com/Colin4k1024/Aetheris/v2/internal/api/http.(*Handler).GetJobTrace-fm (num=6 handlers)
2026/05/25 17:28:11.824519 engine.go:702: [Debug] HERTZ: Method=GET    absolutePath=/api/jobs/:id/trace/cognition --> handlerName=github.com/Colin4k1024/Aetheris/v2/internal/api/http.(*Handler).GetJobCognitionTrace-fm (num=6 handlers)
2026/05/25 17:28:11.824521 engine.go:702: [Debug] HERTZ: Method=GET    absolutePath=/api/jobs/:id/nodes/:node_id --> handlerName=github.com/Colin4k1024/Aetheris/v2/internal/api/http.(*Handler).GetJobNode-fm (num=6 handlers)
2026/05/25 17:28:11.824523 engine.go:702: [Debug] HERTZ: Method=GET    absolutePath=/api/jobs/:id/trace/page  --> handlerName=github.com/Colin4k1024/Aetheris/v2/internal/api/http.(*Handler).GetJobTracePage-fm (num=6 handlers)
2026/05/25 17:28:11.824525 engine.go:702: [Debug] HERTZ: Method=POST   absolutePath=/api/jobs/:id/export      --> handlerName=github.com/Colin4k1024/Aetheris/v2/internal/api/http.(*Handler).ExportJobForensics-fm (num=6 handlers)
2026/05/25 17:28:11.824527 engine.go:702: [Debug] HERTZ: Method=GET    absolutePath=/api/jobs/:id/evidence-graph --> handlerName=github.com/Colin4k1024/Aetheris/v2/internal/api/http.(*Handler).GetJobEvidenceGraph-fm (num=6 handlers)
2026/05/25 17:28:11.824529 engine.go:702: [Debug] HERTZ: Method=GET    absolutePath=/api/jobs/:id/audit-log   --> handlerName=github.com/Colin4k1024/Aetheris/v2/internal/api/http.(*Handler).GetJobAuditLog-fm (num=6 handlers)
2026/05/25 17:28:11.824531 engine.go:702: [Debug] HERTZ: Method=POST   absolutePath=/api/approvals            --> handlerName=github.com/Colin4k1024/Aetheris/v2/internal/api/http.(*Handler).CreateApproval-fm (num=6 handlers)
2026/05/25 17:28:11.824533 engine.go:702: [Debug] HERTZ: Method=GET    absolutePath=/api/approvals            --> handlerName=github.com/Colin4k1024/Aetheris/v2/internal/api/http.(*Handler).ListApprovals-fm (num=6 handlers)
2026/05/25 17:28:11.824535 engine.go:702: [Debug] HERTZ: Method=GET    absolutePath=/api/approvals/:id        --> handlerName=github.com/Colin4k1024/Aetheris/v2/internal/api/http.(*Handler).GetApproval-fm (num=6 handlers)
2026/05/25 17:28:11.824537 engine.go:702: [Debug] HERTZ: Method=POST   absolutePath=/api/approvals/:id/approve --> handlerName=github.com/Colin4k1024/Aetheris/v2/internal/api/http.(*Handler).ApproveApproval-fm (num=6 handlers)
2026/05/25 17:28:11.824539 engine.go:702: [Debug] HERTZ: Method=POST   absolutePath=/api/approvals/:id/reject --> handlerName=github.com/Colin4k1024/Aetheris/v2/internal/api/http.(*Handler).RejectApproval-fm (num=6 handlers)
2026/05/25 17:28:11.824542 engine.go:702: [Debug] HERTZ: Method=POST   absolutePath=/api/forensics/query      --> handlerName=github.com/Colin4k1024/Aetheris/v2/internal/api/http.(*Handler).ForensicsQuery-fm (num=6 handlers)
2026/05/25 17:28:11.824543 engine.go:702: [Debug] HERTZ: Method=POST   absolutePath=/api/forensics/batch-export --> handlerName=github.com/Colin4k1024/Aetheris/v2/internal/api/http.(*Handler).ForensicsBatchExport-fm (num=6 handlers)
2026/05/25 17:28:11.824546 engine.go:702: [Debug] HERTZ: Method=GET    absolutePath=/api/forensics/export-status/:task_id --> handlerName=github.com/Colin4k1024/Aetheris/v2/internal/api/http.(*Handler).ForensicsExportStatus-fm (num=6 handlers)
2026/05/25 17:28:11.824548 engine.go:702: [Debug] HERTZ: Method=GET    absolutePath=/api/forensics/consistency/:job_id --> handlerName=github.com/Colin4k1024/Aetheris/v2/internal/api/http.(*Handler).ForensicsConsistencyCheck-fm (num=6 handlers)
2026/05/25 17:28:11.824550 engine.go:702: [Debug] HERTZ: Method=POST   absolutePath=/api/forensics/ai/detect-anomalies --> handlerName=github.com/Colin4k1024/Aetheris/v2/internal/api/http.(*Handler).AIForensicsDetectAnomalies-fm (num=6 handlers)
2026/05/25 17:28:11.824552 engine.go:702: [Debug] HERTZ: Method=GET    absolutePath=/api/compliance/templates --> handlerName=github.com/Colin4k1024/Aetheris/v2/internal/api/http.(*Handler).ComplianceTemplates-fm (num=6 handlers)
2026/05/25 17:28:11.824554 engine.go:702: [Debug] HERTZ: Method=POST   absolutePath=/api/compliance/apply     --> handlerName=github.com/Colin4k1024/Aetheris/v2/internal/api/http.(*Handler).ComplianceApply-fm (num=6 handlers)
2026/05/25 17:28:11.824556 engine.go:702: [Debug] HERTZ: Method=POST   absolutePath=/api/compliance/report    --> handlerName=github.com/Colin4k1024/Aetheris/v2/internal/api/http.(*Handler).ComplianceReport-fm (num=6 handlers)
2026/05/25 17:28:11.824558 engine.go:702: [Debug] HERTZ: Method=GET    absolutePath=/api/tools/               --> handlerName=github.com/Colin4k1024/Aetheris/v2/internal/api/http.(*Handler).ListTools-fm (num=6 handlers)
2026/05/25 17:28:11.824560 engine.go:702: [Debug] HERTZ: Method=GET    absolutePath=/api/tools/:name          --> handlerName=github.com/Colin4k1024/Aetheris/v2/internal/api/http.(*Handler).GetTool-fm (num=6 handlers)
2026/05/25 17:28:11.824562 engine.go:702: [Debug] HERTZ: Method=GET    absolutePath=/api/system/status        --> handlerName=github.com/Colin4k1024/Aetheris/v2/internal/api/http.(*Handler).SystemStatus-fm (num=6 handlers)
2026/05/25 17:28:11.824564 engine.go:702: [Debug] HERTZ: Method=GET    absolutePath=/api/system/metrics       --> handlerName=github.com/Colin4k1024/Aetheris/v2/internal/api/http.(*Handler).SystemMetrics-fm (num=6 handlers)
2026/05/25 17:28:11.824566 engine.go:702: [Debug] HERTZ: Method=GET    absolutePath=/api/system/workers       --> handlerName=github.com/Colin4k1024/Aetheris/v2/internal/api/http.(*Handler).SystemWorkers-fm (num=6 handlers)
2026/05/25 17:28:11.824568 engine.go:702: [Debug] HERTZ: Method=GET    absolutePath=/api/observability/summary --> handlerName=github.com/Colin4k1024/Aetheris/v2/internal/api/http.(*Handler).GetObservabilitySummary-fm (num=6 handlers)
2026/05/25 17:28:11.824570 engine.go:702: [Debug] HERTZ: Method=GET    absolutePath=/api/observability/stuck  --> handlerName=github.com/Colin4k1024/Aetheris/v2/internal/api/http.(*Handler).GetObservabilityStuck-fm (num=6 handlers)
2026/05/25 17:28:11.824571 engine.go:702: [Debug] HERTZ: Method=GET    absolutePath=/api/trace/overview/page  --> handlerName=github.com/Colin4k1024/Aetheris/v2/internal/api/http.(*Handler).GetTraceOverviewPage-fm (num=6 handlers)
2026/05/25 17:28:11.824573 engine.go:702: [Debug] HERTZ: Method=GET    absolutePath=/api/rbac/role            --> handlerName=github.com/Colin4k1024/Aetheris/v2/internal/api/http.(*Handler).GetUserRole-fm (num=6 handlers)
2026/05/25 17:28:11.824575 engine.go:702: [Debug] HERTZ: Method=POST   absolutePath=/api/rbac/role            --> handlerName=github.com/Colin4k1024/Aetheris/v2/internal/api/http.(*Handler).AssignRole-fm (num=6 handlers)
2026/05/25 17:28:11.824577 engine.go:702: [Debug] HERTZ: Method=POST   absolutePath=/api/rbac/check           --> handlerName=github.com/Colin4k1024/Aetheris/v2/internal/api/http.(*Handler).CheckPermission-fm (num=6 handlers)
2026/05/25 17:28:11.824627 middleware.go:413: [Info] POST /api/forensics/query 0.0.0.0 503 45.458µs
2026/05/25 17:28:11.824652 middleware.go:413: [Info] GET /api/jobs/job_x/evidence-graph 0.0.0.0 503 17.042µs
2026/05/25 17:28:11.824728 middleware.go:413: [Info] GET /api/compliance/templates 0.0.0.0 200 69.792µs
--- PASS: TestRouter_ForensicsRoutesEnabled (0.00s)
PASS
ok  	github.com/Colin4k1024/Aetheris/v2/internal/api/http	0.705s
```
