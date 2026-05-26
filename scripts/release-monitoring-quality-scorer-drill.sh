#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT_DIR"

ARTIFACT_DIR="${MONITORING_QUALITY_ARTIFACT_DIR:-artifacts/release}"
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
report="$ARTIFACT_DIR/monitoring-quality-scorer-drill-$ts.md"
tmp_dir="$(mktemp -d)"
trap 'rm -rf "$tmp_dir"' EXIT

monitoring_log="$tmp_dir/monitoring.log"
overall="PASS"
monitoring_status="PASS"

echo "[monitoring-quality-scorer-drill] running monitoring quality scorer gate..."
if ! "$GO_BIN" test -v ./pkg/monitoring -run 'TestQualityScorer|TestAssessQualityScore' >"$monitoring_log" 2>&1; then
  overall="FAIL"
  monitoring_status="FAIL"
fi

{
  echo "# Monitoring Quality Scorer Release Drill"
  echo
  echo "- Timestamp: $ts"
  echo "- Overall: $overall"
  echo
  echo "## Suites"
  echo
  echo "- pkg/monitoring ($monitoring_status)"
  echo
  echo "## pkg/monitoring output"
  echo
  echo '```text'
  cat "$monitoring_log"
  echo '```'
} >"$report"

echo "[monitoring-quality-scorer-drill] report written: $report"

if [[ "$overall" != "PASS" ]]; then
  echo "[monitoring-quality-scorer-drill] gate failed" >&2
  exit 1
fi

echo "[monitoring-quality-scorer-drill] gate passed"
