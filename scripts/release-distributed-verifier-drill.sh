#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT_DIR"

ARTIFACT_DIR="${DISTRIBUTED_VERIFIER_ARTIFACT_DIR:-artifacts/release}"
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
report="$ARTIFACT_DIR/distributed-verifier-drill-$ts.md"
tmp_dir="$(mktemp -d)"
trap 'rm -rf "$tmp_dir"' EXIT

verifier_log="$tmp_dir/verifier.log"
overall="PASS"
verifier_status="PASS"

echo "[distributed-verifier-drill] running distributed verifier prototype gate..."
if ! "$GO_BIN" test -v ./pkg/distributed -run 'TestVerifyAcrossOrgs|TestDistributedVerifier|TestProtocolEventSource|TestMultiOrgVerifyResult' >"$verifier_log" 2>&1; then
  overall="FAIL"
  verifier_status="FAIL"
fi

{
  echo "# Distributed Verifier Release Drill"
  echo
  echo "- Timestamp: $ts"
  echo "- Overall: $overall"
  echo
  echo "## Suites"
  echo
  echo "- pkg/distributed ($verifier_status)"
  echo
  echo "## pkg/distributed output"
  echo
  echo '```text'
  cat "$verifier_log"
  echo '```'
} >"$report"

echo "[distributed-verifier-drill] report written: $report"

if [[ "$overall" != "PASS" ]]; then
  echo "[distributed-verifier-drill] gate failed" >&2
  exit 1
fi

echo "[distributed-verifier-drill] gate passed"
