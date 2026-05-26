# RBAC, Redaction, and Retention Release Drill

- Timestamp: 20260525-163740
- Overall: PASS

## Suites

- pkg/auth (PASS)
- internal/api/http (PASS)
- pkg/proof (PASS)
- internal/runtime/jobstore (PASS)

## pkg/auth output

```text
=== RUN   TestRBAC_AdminHasAllPermissions
--- PASS: TestRBAC_AdminHasAllPermissions (0.00s)
=== RUN   TestRBAC_UserCannotExport
--- PASS: TestRBAC_UserCannotExport (0.00s)
=== RUN   TestRBAC_TenantIsolation
--- PASS: TestRBAC_TenantIsolation (0.00s)
=== RUN   TestRBAC_AuditorCanViewAndExport
--- PASS: TestRBAC_AuditorCanViewAndExport (0.00s)
=== RUN   TestHasPermission
--- PASS: TestHasPermission (0.00s)
=== RUN   TestSimpleRBACChecker
--- PASS: TestSimpleRBACChecker (0.00s)
=== RUN   TestSimpleRBACChecker_AssignRole
--- PASS: TestSimpleRBACChecker_AssignRole (0.00s)
PASS
ok  	github.com/Colin4k1024/Aetheris/v2/pkg/auth	0.482s
```

## internal/api/http output

```text
=== RUN   TestGetJob_TenantIsolation
=== RUN   TestGetJob_TenantIsolation/tenant_matched
2026/05/25 16:37:45.372120 engine.go:702: [Debug] HERTZ: Method=GET    absolutePath=/api/jobs/:id             --> handlerName=github.com/Colin4k1024/Aetheris/v2/internal/api/http.TestGetJob_TenantIsolation.func1.1 (num=2 handlers)
=== RUN   TestGetJob_TenantIsolation/tenant_mismatched
2026/05/25 16:37:45.372815 engine.go:702: [Debug] HERTZ: Method=GET    absolutePath=/api/jobs/:id             --> handlerName=github.com/Colin4k1024/Aetheris/v2/internal/api/http.TestGetJob_TenantIsolation.func2.1 (num=2 handlers)
--- PASS: TestGetJob_TenantIsolation (0.00s)
    --- PASS: TestGetJob_TenantIsolation/tenant_matched (0.00s)
    --- PASS: TestGetJob_TenantIsolation/tenant_mismatched (0.00s)
=== RUN   TestGetJob_DefaultTenantFallback
2026/05/25 16:37:45.372880 engine.go:702: [Debug] HERTZ: Method=GET    absolutePath=/api/jobs/:id             --> handlerName=github.com/Colin4k1024/Aetheris/v2/internal/api/http.TestGetJob_DefaultTenantFallback.func1 (num=2 handlers)
--- PASS: TestGetJob_DefaultTenantFallback (0.00s)
=== RUN   TestJobStop_RBACAndTenantMatrix
=== RUN   TestJobStop_RBACAndTenantMatrix/same_tenant_+_operator_role
2026/05/25 16:37:45.372990 engine.go:702: [Debug] HERTZ: Method=POST   absolutePath=/api/jobs/:id/stop        --> handlerName=github.com/Colin4k1024/Aetheris/v2/internal/api/http.TestJobStop_RBACAndTenantMatrix.func1.2 (num=4 handlers)
=== RUN   TestJobStop_RBACAndTenantMatrix/same_tenant_+_user_role_denied_by_RBAC
2026/05/25 16:37:45.373063 engine.go:702: [Debug] HERTZ: Method=POST   absolutePath=/api/jobs/:id/stop        --> handlerName=github.com/Colin4k1024/Aetheris/v2/internal/api/http.TestJobStop_RBACAndTenantMatrix.func1.2 (num=4 handlers)
=== RUN   TestJobStop_RBACAndTenantMatrix/cross_tenant_+_same_operator_role_denied_by_tenant_isolation
2026/05/25 16:37:45.373115 engine.go:702: [Debug] HERTZ: Method=POST   absolutePath=/api/jobs/:id/stop        --> handlerName=github.com/Colin4k1024/Aetheris/v2/internal/api/http.TestJobStop_RBACAndTenantMatrix.func1.2 (num=4 handlers)
--- PASS: TestJobStop_RBACAndTenantMatrix (0.00s)
    --- PASS: TestJobStop_RBACAndTenantMatrix/same_tenant_+_operator_role (0.00s)
    --- PASS: TestJobStop_RBACAndTenantMatrix/same_tenant_+_user_role_denied_by_RBAC (0.00s)
    --- PASS: TestJobStop_RBACAndTenantMatrix/cross_tenant_+_same_operator_role_denied_by_tenant_isolation (0.00s)
=== RUN   TestGetJobEvents_TenantIsolation
=== RUN   TestGetJobEvents_TenantIsolation/tenant_matched
2026/05/25 16:37:45.373197 engine.go:702: [Debug] HERTZ: Method=GET    absolutePath=/api/jobs/:id/events      --> handlerName=github.com/Colin4k1024/Aetheris/v2/internal/api/http.TestGetJobEvents_TenantIsolation.func1.1 (num=2 handlers)
=== RUN   TestGetJobEvents_TenantIsolation/tenant_mismatched
2026/05/25 16:37:45.373332 engine.go:702: [Debug] HERTZ: Method=GET    absolutePath=/api/jobs/:id/events      --> handlerName=github.com/Colin4k1024/Aetheris/v2/internal/api/http.TestGetJobEvents_TenantIsolation.func2.1 (num=2 handlers)
--- PASS: TestGetJobEvents_TenantIsolation (0.00s)
    --- PASS: TestGetJobEvents_TenantIsolation/tenant_matched (0.00s)
    --- PASS: TestGetJobEvents_TenantIsolation/tenant_mismatched (0.00s)
=== RUN   TestGetJobReplay_TenantIsolation
=== RUN   TestGetJobReplay_TenantIsolation/tenant_matched
2026/05/25 16:37:45.373381 engine.go:702: [Debug] HERTZ: Method=GET    absolutePath=/api/jobs/:id/replay      --> handlerName=github.com/Colin4k1024/Aetheris/v2/internal/api/http.TestGetJobReplay_TenantIsolation.func1.1 (num=2 handlers)
=== RUN   TestGetJobReplay_TenantIsolation/tenant_mismatched
2026/05/25 16:37:45.373463 engine.go:702: [Debug] HERTZ: Method=GET    absolutePath=/api/jobs/:id/replay      --> handlerName=github.com/Colin4k1024/Aetheris/v2/internal/api/http.TestGetJobReplay_TenantIsolation.func2.1 (num=2 handlers)
--- PASS: TestGetJobReplay_TenantIsolation (0.00s)
    --- PASS: TestGetJobReplay_TenantIsolation/tenant_matched (0.00s)
    --- PASS: TestGetJobReplay_TenantIsolation/tenant_mismatched (0.00s)
=== RUN   TestGetJobTrace_TenantIsolation
=== RUN   TestGetJobTrace_TenantIsolation/tenant_matched
2026/05/25 16:37:45.373528 engine.go:702: [Debug] HERTZ: Method=GET    absolutePath=/api/jobs/:id/trace       --> handlerName=github.com/Colin4k1024/Aetheris/v2/internal/api/http.TestGetJobTrace_TenantIsolation.func1.1 (num=2 handlers)
=== RUN   TestGetJobTrace_TenantIsolation/tenant_mismatched
2026/05/25 16:37:45.373800 engine.go:702: [Debug] HERTZ: Method=GET    absolutePath=/api/jobs/:id/trace       --> handlerName=github.com/Colin4k1024/Aetheris/v2/internal/api/http.TestGetJobTrace_TenantIsolation.func2.1 (num=2 handlers)
--- PASS: TestGetJobTrace_TenantIsolation (0.00s)
    --- PASS: TestGetJobTrace_TenantIsolation/tenant_matched (0.00s)
    --- PASS: TestGetJobTrace_TenantIsolation/tenant_mismatched (0.00s)
=== RUN   TestTenantIsolation_ForensicsQuery
2026/05/25 16:37:45.374956 engine.go:702: [Debug] HERTZ: Method=POST   absolutePath=/api/forensics/query      --> handlerName=github.com/Colin4k1024/Aetheris/v2/internal/api/http.TestTenantIsolation_ForensicsQuery.func2 (num=2 handlers)
--- PASS: TestTenantIsolation_ForensicsQuery (0.00s)
PASS
ok  	github.com/Colin4k1024/Aetheris/v2/internal/api/http	0.674s
```

## pkg/proof output

```text
=== RUN   TestEndToEnd_RedactedExportRemovesPIIAndVerifies
--- PASS: TestEndToEnd_RedactedExportRemovesPIIAndVerifies (0.00s)
PASS
ok  	github.com/Colin4k1024/Aetheris/v2/pkg/proof	0.479s
```

## internal/runtime/jobstore output

```text
=== RUN   TestGC_ArchiveAndDelete
--- PASS: TestGC_ArchiveAndDelete (0.00s)
=== RUN   TestGC_DeleteOnly
--- PASS: TestGC_DeleteOnly (0.00s)
=== RUN   TestGC_RetentionDoesNotMutateEventHistoryForReplay
--- PASS: TestGC_RetentionDoesNotMutateEventHistoryForReplay (0.00s)
PASS
ok  	github.com/Colin4k1024/Aetheris/v2/internal/runtime/jobstore	0.743s
```
