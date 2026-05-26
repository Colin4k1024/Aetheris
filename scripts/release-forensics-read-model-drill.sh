#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT_DIR"

ARTIFACT_DIR="${FORENSICS_READ_MODEL_ARTIFACT_DIR:-artifacts/release}"
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
report="$ARTIFACT_DIR/forensics-read-model-drill-$ts.md"
tmp_dir="$(mktemp -d)"
trap 'rm -rf "$tmp_dir"' EXIT

http_log="$tmp_dir/http.log"
status="PASS"

regex='TestTenantIsolation_ForensicsQuery|TestForensicsQuery_PaginationLimitCapAndFilters|TestForensicsQuery_LargeEventStreamEventCount|TestRouter_ForensicsRoutes(DisabledByDefault|Enabled)|TestForensicsBatchExport_StatusFlow'

echo "[forensics-read-model-drill] running internal/api/http suite..."
if ! "$GO_BIN" test -v ./internal/api/http -run "$regex" >"$http_log" 2>&1; then
  status="FAIL"
fi

{
  echo "# Forensics Read Model Release Drill"
  echo
  echo "- Timestamp: $ts"
  echo "- Overall: $status"
  echo "- Regex: \`$regex\`"
  echo
  echo "## internal/api/http output"
  echo
  echo '```text'
  cat "$http_log"
  echo '```'
} >"$report"

echo "[forensics-read-model-drill] report written: $report"

if [[ "$status" != "PASS" ]]; then
  echo "[forensics-read-model-drill] gate failed" >&2
  exit 1
fi

echo "[forensics-read-model-drill] gate passed"
