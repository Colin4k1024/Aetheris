#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT_DIR"

ARTIFACT_DIR="${AI_FORENSICS_EVAL_ARTIFACT_DIR:-artifacts/release}"
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
report="$ARTIFACT_DIR/ai-forensics-eval-drill-$ts.md"
tmp_dir="$(mktemp -d)"
trap 'rm -rf "$tmp_dir"' EXIT

detector_log="$tmp_dir/detector.log"
http_log="$tmp_dir/http.log"

overall="PASS"
detector_status="PASS"
http_status="PASS"

run_suite() {
  local name="$1"
  local logfile="$2"
  shift 2
  echo "[ai-forensics-eval-drill] running $name..."
  if ! "$@" >"$logfile" 2>&1; then
    overall="FAIL"
    return 1
  fi
  return 0
}

if ! run_suite "AI forensics golden eval" "$detector_log" \
  "$GO_BIN" test -v ./pkg/ai_forensics -run 'TestAnomalyDetector|TestGoldenEvalCases|TestPatternMatcher'; then
  detector_status="FAIL"
fi

if ! run_suite "HTTP AI forensics event signal extraction" "$http_log" \
  "$GO_BIN" test -v ./internal/api/http -run 'TestAIForensicsDetectAnomalies_EventSignals|TestRouter_ForensicsRoutes'; then
  http_status="FAIL"
fi

{
  echo "# AI Forensics Eval Release Drill"
  echo
  echo "- Timestamp: $ts"
  echo "- Overall: $overall"
  echo
  echo "## Suites"
  echo
  echo "- pkg/ai_forensics ($detector_status)"
  echo "- internal/api/http ($http_status)"
  echo
  echo "## pkg/ai_forensics output"
  echo
  echo '```text'
  cat "$detector_log"
  echo '```'
  echo
  echo "## internal/api/http output"
  echo
  echo '```text'
  cat "$http_log"
  echo '```'
} >"$report"

echo "[ai-forensics-eval-drill] report written: $report"

if [[ "$overall" != "PASS" ]]; then
  echo "[ai-forensics-eval-drill] gate failed" >&2
  exit 1
fi

echo "[ai-forensics-eval-drill] gate passed"
