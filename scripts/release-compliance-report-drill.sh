#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT_DIR"

ARTIFACT_DIR="${COMPLIANCE_REPORT_ARTIFACT_DIR:-artifacts/release}"
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
report="$ARTIFACT_DIR/compliance-report-drill-$ts.md"
tmp_dir="$(mktemp -d)"
trap 'rm -rf "$tmp_dir"' EXIT

compliance_log="$tmp_dir/compliance.log"
http_log="$tmp_dir/http.log"

overall="PASS"
compliance_status="PASS"
http_status="PASS"

run_suite() {
  local name="$1"
  local logfile="$2"
  shift 2
  echo "[compliance-report-drill] running $name..."
  if ! "$@" >"$logfile" 2>&1; then
    overall="FAIL"
    return 1
  fi
  return 0
}

if ! run_suite "compliance package report contracts" "$compliance_log" \
  "$GO_BIN" test -v ./pkg/compliance -run 'TestGenerateReport|TestGetTemplate|TestListTemplates|TestFrameworkFactory_CreateFramework'; then
  compliance_status="FAIL"
fi

if ! run_suite "HTTP compliance report export contract" "$http_log" \
  "$GO_BIN" test -v ./internal/api/http -run 'TestComplianceReport_|TestRouter_ForensicsRoutes'; then
  http_status="FAIL"
fi

{
  echo "# Compliance Report Release Drill"
  echo
  echo "- Timestamp: $ts"
  echo "- Overall: $overall"
  echo
  echo "## Suites"
  echo
  echo "- pkg/compliance ($compliance_status)"
  echo "- internal/api/http ($http_status)"
  echo
  echo "## pkg/compliance output"
  echo
  echo '```text'
  cat "$compliance_log"
  echo '```'
  echo
  echo "## internal/api/http output"
  echo
  echo '```text'
  cat "$http_log"
  echo '```'
} >"$report"

echo "[compliance-report-drill] report written: $report"

if [[ "$overall" != "PASS" ]]; then
  echo "[compliance-report-drill] gate failed" >&2
  exit 1
fi

echo "[compliance-report-drill] gate passed"
