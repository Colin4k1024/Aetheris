#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT_DIR"

ARTIFACT_DIR="${EVIDENCE_SIGNING_ARTIFACT_DIR:-artifacts/release}"
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
report="$ARTIFACT_DIR/evidence-signing-drill-$ts.md"
tmp_dir="$(mktemp -d)"
trap 'rm -rf "$tmp_dir"' EXIT

app_api_log="$tmp_dir/app-api.log"
http_log="$tmp_dir/http.log"
cli_log="$tmp_dir/cli.log"
proof_log="$tmp_dir/proof.log"

overall="PASS"
app_api_status="PASS"
http_status="PASS"
cli_status="PASS"
proof_status="PASS"

run_suite() {
  local name="$1"
  local logfile="$2"
  shift 2
  echo "[evidence-signing-drill] running $name..."
  if ! "$@" >"$logfile" 2>&1; then
    overall="FAIL"
    return 1
  fi
  return 0
}

if ! run_suite "app/api signing config" "$app_api_log" \
  "$GO_BIN" test -v ./internal/app/api -run 'TestEvidenceSigningConfigFromConfig'; then
  app_api_status="FAIL"
fi

if ! run_suite "api/http signed export" "$http_log" \
  "$GO_BIN" test -v ./internal/api/http -run 'TestBuildForensicsPackage_(SignedProof|InvalidSigningKey|ProofCompatible)'; then
  http_status="FAIL"
fi

if ! run_suite "cli public-key verification" "$cli_log" \
  "$GO_BIN" test -v ./cmd/cli -run 'TestVerifyEvidenceZip_(SignedWithPublicKey|Success|Tampered)|TestParseEvidenceVerifyArgs_PublicKey'; then
  cli_status="FAIL"
fi

if ! run_suite "proof package verification" "$proof_log" \
  "$GO_BIN" test -v ./pkg/proof -run 'TestEndToEnd_(ExportAndVerify|TamperDetection)|TestVerifyEvidenceZip'; then
  proof_status="FAIL"
fi

{
  echo "# Evidence Signing Release Drill"
  echo
  echo "- Timestamp: $ts"
  echo "- Overall: $overall"
  echo
  echo "## Suites"
  echo
  echo "- internal/app/api ($app_api_status)"
  echo "- internal/api/http ($http_status)"
  echo "- cmd/cli ($cli_status)"
  echo "- pkg/proof ($proof_status)"
  echo
  echo "## internal/app/api output"
  echo
  echo '```text'
  cat "$app_api_log"
  echo '```'
  echo
  echo "## internal/api/http output"
  echo
  echo '```text'
  cat "$http_log"
  echo '```'
  echo
  echo "## cmd/cli output"
  echo
  echo '```text'
  cat "$cli_log"
  echo '```'
  echo
  echo "## pkg/proof output"
  echo
  echo '```text'
  cat "$proof_log"
  echo '```'
} >"$report"

echo "[evidence-signing-drill] report written: $report"

if [[ "$overall" != "PASS" ]]; then
  echo "[evidence-signing-drill] gate failed" >&2
  exit 1
fi

echo "[evidence-signing-drill] gate passed"
