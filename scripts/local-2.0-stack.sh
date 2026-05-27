#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
COMPOSE_FILE="${COMPOSE_FILE:-$ROOT_DIR/deployments/compose/docker-compose.yml}"

# Auto-detect CI environment (GitHub Actions sets CI=true).
# When running in CI, overlay docker-compose.ci.yml to add the mock LLM service
# (replacing host.docker.internal:11434 which is unavailable in Linux runners).
CI_COMPOSE_FILE="${ROOT_DIR}/deployments/compose/docker-compose.ci.yml"
CI_COMPOSE_OVERRIDE=""
if [ "${CI:-}" = "true" ] && [ -f "$CI_COMPOSE_FILE" ]; then
  CI_COMPOSE_OVERRIDE="-f $CI_COMPOSE_FILE"
  echo "[stack] CI mode detected — using CI overlay ($CI_COMPOSE_FILE)"
fi

# In CI allow longer wait for the mock LLM service to build and become healthy.
HEALTH_WAIT_SECS="${HEALTH_WAIT_SECS:-${CI:+90}}"
HEALTH_WAIT_SECS="${HEALTH_WAIT_SECS:-30}"

require_cmd() {
  if ! command -v "$1" >/dev/null 2>&1; then
    echo "error: required command not found: $1" >&2
    exit 1
  fi
}

compose_cmd() {
  if command -v docker >/dev/null 2>&1 && docker compose version >/dev/null 2>&1; then
    # shellcheck disable=SC2086
    docker compose -f "$COMPOSE_FILE" $CI_COMPOSE_OVERRIDE "$@"
    return
  fi
  if command -v docker-compose >/dev/null 2>&1; then
    # shellcheck disable=SC2086
    docker-compose -f "$COMPOSE_FILE" $CI_COMPOSE_OVERRIDE "$@"
    return
  fi
  echo "error: neither 'docker compose' nor 'docker-compose' is available" >&2
  exit 1
}

usage() {
  cat <<USAGE
Usage: scripts/local-2.0-stack.sh <command>

Commands:
  start    Build and start local 2.0 stack (postgres + api + worker1 + worker2)
  stop     Stop and remove local 2.0 stack
  status   Show running container status
  logs     Tail logs for all stack services
  health   Call API health endpoint (http://localhost:8080/api/health)
USAGE
}

cmd_start() {
  require_cmd curl
  echo "[stack] starting local 2.0 stack..."
  compose_cmd up -d --build
  echo "[stack] waiting for API health (max ${HEALTH_WAIT_SECS}s)..."
  for i in $(seq 1 "$HEALTH_WAIT_SECS"); do
    if curl -fsS "http://localhost:8080/api/health" >/dev/null 2>&1; then
      echo "[stack] API is healthy"
      compose_cmd ps
      return
    fi
    sleep 1
  done
  echo "[stack] API health check timed out after ${HEALTH_WAIT_SECS}s" >&2
  compose_cmd ps || true
  exit 1
}

cmd_stop() {
  echo "[stack] stopping local 2.0 stack..."
  compose_cmd down
  echo "[stack] stopped"
}

cmd_status() {
  compose_cmd ps
}

cmd_logs() {
  compose_cmd logs -f
}

cmd_health() {
  require_cmd curl
  curl -fsS "http://localhost:8080/api/health"
  echo
}

main() {
  if [[ $# -lt 1 ]]; then
    usage
    exit 1
  fi

  case "$1" in
    start)
      cmd_start
      ;;
    stop)
      cmd_stop
      ;;
    status)
      cmd_status
      ;;
    logs)
      cmd_logs
      ;;
    health)
      cmd_health
      ;;
    -h|--help|help)
      usage
      ;;
    *)
      usage
      exit 1
      ;;
  esac
}

main "$@"
