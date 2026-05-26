#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT_DIR"

ARTIFACT_DIR="${RBAC_RED_RET_ARTIFACT_DIR:-artifacts/release}"
GOCACHE="${GOCACHE:-/tmp/corag-gocache}"
GO_BIN="${GO_BIN:-}"

if [[ -z "$GO_BIN" ]]; then
  if command -v go >/dev/null 2>&1; then
    GO_BIN="$(command -v go)"
  elif [[ -x /usr/local/go/bin/go ]]; then
    GO_BIN="/usr/local/go/bin/go"
  else
    echo "error: go binary not found; set GO_BIN=/path/to/go" >&2
    exit 1
  fi
fi

mkdir -p "$ARTIFACT_DIR"
mkdir -p "$GOCACHE"
export GOCACHE

ts="$(date +%Y%m%d-%H%M%S)"
report="$ARTIFACT_DIR/rbac-redaction-retention-drill-$ts.md"
tmp_dir="$(mktemp -d)"
trap 'rm -rf "$tmp_dir"' EXIT

auth_log="$tmp_dir/auth.log"
http_log="$tmp_dir/http.log"
proof_log="$tmp_dir/proof.log"
retention_log="$tmp_dir/retention.log"

overall="PASS"
auth_status="PASS"
http_status="PASS"
proof_status="PASS"
retention_status="PASS"

run_suite() {
  local name="$1"
  local logfile="$2"
  shift 2
  echo "[rbac-redaction-retention-drill] running $name..."
  if ! "$@" >"$logfile" 2>&1; then
    overall="FAIL"
    return 1
  fi
  return 0
}

if ! run_suite "auth RBAC matrix" "$auth_log" \
  "$GO_BIN" test -v ./pkg/auth -run 'TestRBAC_(AdminHasAllPermissions|UserCannotExport|TenantIsolation|AuditorCanViewAndExport)|TestHasPermission|TestSimpleRBACChecker'; then
  auth_status="FAIL"
fi

if ! run_suite "HTTP tenant/RBAC matrix" "$http_log" \
  "$GO_BIN" test -v ./internal/api/http -run 'TestGetJob_TenantIsolation|TestGetJob_DefaultTenantFallback|TestGetJobEvents_TenantIsolation|TestGetJobReplay_TenantIsolation|TestGetJobTrace_TenantIsolation|TestJobStop_RBACAndTenantMatrix|TestTenantIsolation_ForensicsQuery'; then
  http_status="FAIL"
fi

if ! run_suite "redacted evidence export" "$proof_log" \
  "$GO_BIN" test -v ./pkg/proof -run 'TestEndToEnd_RedactedExportRemovesPIIAndVerifies'; then
  proof_status="FAIL"
fi

if ! run_suite "retention replay invariant" "$retention_log" \
  "$GO_BIN" test -v ./internal/runtime/jobstore -run 'TestGC_(ArchiveAndDelete|DeleteOnly|RetentionDoesNotMutateEventHistoryForReplay)'; then
  retention_status="FAIL"
fi

{
  echo "# RBAC, Redaction, and Retention Release Drill"
  echo
  echo "- Timestamp: $ts"
  echo "- Overall: $overall"
  echo
  echo "## Suites"
  echo
  echo "- pkg/auth ($auth_status)"
  echo "- internal/api/http ($http_status)"
  echo "- pkg/proof ($proof_status)"
  echo "- internal/runtime/jobstore ($retention_status)"
  echo
  echo "## pkg/auth output"
  echo
  echo '```text'
  cat "$auth_log"
  echo '```'
  echo
  echo "## internal/api/http output"
  echo
  echo '```text'
  cat "$http_log"
  echo '```'
  echo
  echo "## pkg/proof output"
  echo
  echo '```text'
  cat "$proof_log"
  echo '```'
  echo
  echo "## internal/runtime/jobstore output"
  echo
  echo '```text'
  cat "$retention_log"
  echo '```'
} >"$report"

echo "[rbac-redaction-retention-drill] report written: $report"

if [[ "$overall" != "PASS" ]]; then
  echo "[rbac-redaction-retention-drill] gate failed" >&2
  exit 1
fi

echo "[rbac-redaction-retention-drill] gate passed"
