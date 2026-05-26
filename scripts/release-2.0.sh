#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT_DIR"

STRICT_P0_MODE="${RELEASE_STRICT_P0:-false}"

# In CI/release pipelines, run P0 gates by default unless explicitly disabled.
if [[ "$STRICT_P0_MODE" == "true" ]]; then
  RUN_P0_PERF="1"
  RUN_P0_DRILLS="1"
  RUN_DB_DRILL="1"
  RUN_TENANT_REGRESSION="1"
  RUN_EVIDENCE_SIGNING_DRILL="1"
  RUN_FORENSICS_READ_MODEL_DRILL="1"
  RUN_RBAC_RED_RET_DRILL="1"
  RUN_COMPLIANCE_REPORT_DRILL="1"
  RUN_AI_FORENSICS_EVAL_DRILL="1"
  RUN_DISTRIBUTED_VERIFIER_DRILL="1"
  RUN_MONITORING_QUALITY_DRILL="1"
elif [[ "${CI:-}" == "true" ]]; then
  RUN_P0_PERF="${RUN_P0_PERF:-1}"
  RUN_P0_DRILLS="${RUN_P0_DRILLS:-1}"
  RUN_DB_DRILL="${RUN_DB_DRILL:-1}"
  RUN_TENANT_REGRESSION="${RUN_TENANT_REGRESSION:-1}"
  RUN_EVIDENCE_SIGNING_DRILL="${RUN_EVIDENCE_SIGNING_DRILL:-1}"
  RUN_FORENSICS_READ_MODEL_DRILL="${RUN_FORENSICS_READ_MODEL_DRILL:-1}"
  RUN_RBAC_RED_RET_DRILL="${RUN_RBAC_RED_RET_DRILL:-1}"
  RUN_COMPLIANCE_REPORT_DRILL="${RUN_COMPLIANCE_REPORT_DRILL:-1}"
  RUN_AI_FORENSICS_EVAL_DRILL="${RUN_AI_FORENSICS_EVAL_DRILL:-1}"
  RUN_DISTRIBUTED_VERIFIER_DRILL="${RUN_DISTRIBUTED_VERIFIER_DRILL:-1}"
  RUN_MONITORING_QUALITY_DRILL="${RUN_MONITORING_QUALITY_DRILL:-1}"
else
  RUN_P0_PERF="${RUN_P0_PERF:-0}"
  RUN_P0_DRILLS="${RUN_P0_DRILLS:-0}"
  RUN_DB_DRILL="${RUN_DB_DRILL:-0}"
  RUN_TENANT_REGRESSION="${RUN_TENANT_REGRESSION:-1}"
  RUN_EVIDENCE_SIGNING_DRILL="${RUN_EVIDENCE_SIGNING_DRILL:-1}"
  RUN_FORENSICS_READ_MODEL_DRILL="${RUN_FORENSICS_READ_MODEL_DRILL:-1}"
  RUN_RBAC_RED_RET_DRILL="${RUN_RBAC_RED_RET_DRILL:-1}"
  RUN_COMPLIANCE_REPORT_DRILL="${RUN_COMPLIANCE_REPORT_DRILL:-1}"
  RUN_AI_FORENSICS_EVAL_DRILL="${RUN_AI_FORENSICS_EVAL_DRILL:-1}"
  RUN_DISTRIBUTED_VERIFIER_DRILL="${RUN_DISTRIBUTED_VERIFIER_DRILL:-1}"
  RUN_MONITORING_QUALITY_DRILL="${RUN_MONITORING_QUALITY_DRILL:-1}"
fi

echo "[release-2.0] starting release checks..."
echo "[release-2.0] strict P0 mode: $STRICT_P0_MODE"
echo "[release-2.0] gate flags: RUN_P0_PERF=$RUN_P0_PERF RUN_P0_DRILLS=$RUN_P0_DRILLS RUN_DB_DRILL=$RUN_DB_DRILL RUN_TENANT_REGRESSION=$RUN_TENANT_REGRESSION RUN_EVIDENCE_SIGNING_DRILL=$RUN_EVIDENCE_SIGNING_DRILL RUN_FORENSICS_READ_MODEL_DRILL=$RUN_FORENSICS_READ_MODEL_DRILL RUN_RBAC_RED_RET_DRILL=$RUN_RBAC_RED_RET_DRILL RUN_COMPLIANCE_REPORT_DRILL=$RUN_COMPLIANCE_REPORT_DRILL RUN_AI_FORENSICS_EVAL_DRILL=$RUN_AI_FORENSICS_EVAL_DRILL RUN_DISTRIBUTED_VERIFIER_DRILL=$RUN_DISTRIBUTED_VERIFIER_DRILL RUN_MONITORING_QUALITY_DRILL=$RUN_MONITORING_QUALITY_DRILL"

echo "[release-2.0] gofmt check"
if [ -n "$(gofmt -l .)" ]; then
  echo "[release-2.0] gofmt check failed:" >&2
  gofmt -l . >&2
  exit 1
fi

echo "[release-2.0] go vet"
go vet ./...

echo "[release-2.0] unit and integration tests"
go test -v ./...

echo "[release-2.0] build artifacts"
go build -v ./...

echo "[release-2.0] cli smoke"
./scripts/local-2.0-stack.sh --help >/dev/null || true

if [[ "$RUN_P0_PERF" == "1" ]]; then
  echo "[release-2.0] P0 performance gate"
  ./scripts/release-p0-perf.sh
fi

if [[ "$RUN_P0_DRILLS" == "1" ]]; then
  echo "[release-2.0] P0 failure drill gate"
  RUN_DB_DRILL="$RUN_DB_DRILL" ./scripts/release-p0-drill.sh
fi

if [[ "$RUN_TENANT_REGRESSION" == "1" ]]; then
  echo "[release-2.0] tenant regression gate"
  ./scripts/release-tenant-regression.sh
fi

if [[ "$RUN_EVIDENCE_SIGNING_DRILL" == "1" ]]; then
  echo "[release-2.0] evidence signing drill gate"
  ./scripts/release-evidence-signing-drill.sh
fi

if [[ "$RUN_FORENSICS_READ_MODEL_DRILL" == "1" ]]; then
  echo "[release-2.0] forensics read model drill gate"
  ./scripts/release-forensics-read-model-drill.sh
fi

if [[ "$RUN_RBAC_RED_RET_DRILL" == "1" ]]; then
  echo "[release-2.0] RBAC/redaction/retention drill gate"
  ./scripts/release-rbac-redaction-retention-drill.sh
fi

if [[ "$RUN_COMPLIANCE_REPORT_DRILL" == "1" ]]; then
  echo "[release-2.0] compliance report drill gate"
  ./scripts/release-compliance-report-drill.sh
fi

if [[ "$RUN_AI_FORENSICS_EVAL_DRILL" == "1" ]]; then
  echo "[release-2.0] AI forensics eval drill gate"
  ./scripts/release-ai-forensics-eval-drill.sh
fi

if [[ "$RUN_DISTRIBUTED_VERIFIER_DRILL" == "1" ]]; then
  echo "[release-2.0] distributed verifier drill gate"
  ./scripts/release-distributed-verifier-drill.sh
fi

if [[ "$RUN_MONITORING_QUALITY_DRILL" == "1" ]]; then
  echo "[release-2.0] monitoring quality scorer drill gate"
  ./scripts/release-monitoring-quality-scorer-drill.sh
fi

echo "[release-2.0] completed successfully"
echo "[release-2.0] see docs/release-checklist-2.0.md for manual sign-off items"
