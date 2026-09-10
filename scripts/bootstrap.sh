#!/usr/bin/env bash
# bootstrap.sh — Environment and dependency setup for Aetheris development.
# Checks Go version, PostgreSQL, Redis, and prepares the Go module cache.
set -euo pipefail

MIN_GO_VERSION="1.26.0"
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

info()  { echo -e "${GREEN}[bootstrap]${NC} $*"; }
warn()  { echo -e "${YELLOW}[bootstrap]${NC} $*"; }
fatal() { echo -e "${RED}[bootstrap]${NC} $*"; exit 1; }

# --- 1. Check Go version ---
check_go() {
    command -v go >/dev/null 2>&1 || fatal "Go is not installed. Install Go ${MIN_GO_VERSION}+ from https://go.dev/dl/"
    local go_version
    go_version=$(go version | awk '{print $3}' | sed 's/go//')
    info "Go version: ${go_version}"
    # Simple version comparison (major.minor only)
    local go_major go_minor min_major min_minor
    go_major=$(echo "${go_version}" | cut -d. -f1)
    go_minor=$(echo "${go_version}" | cut -d. -f2)
    min_major=$(echo "${MIN_GO_VERSION}" | cut -d. -f1)
    min_minor=$(echo "${MIN_GO_VERSION}" | cut -d. -f2)
    if [ "${go_major}" -lt "${min_major}" ] || { [ "${go_major}" -eq "${min_major}" ] && [ "${go_minor}" -lt "${min_minor}" ]; }; then
        fatal "Go ${MIN_GO_VERSION}+ required, found ${go_version}"
    fi
}

# --- 2. Check PostgreSQL ---
check_postgres() {
    if command -v psql >/dev/null 2>&1; then
        info "psql found: $(psql --version)"
    else
        warn "psql not found — PostgreSQL client tools are needed for migrations."
        warn "Install PostgreSQL client or set DATABASE_URL to a remote DSN."
    fi
}

# --- 3. Check Redis ---
check_redis() {
    if command -v redis-cli >/dev/null 2>&1; then
        info "redis-cli found: $(redis-cli --version)"
    else
        warn "redis-cli not found — Redis is needed for cache, RAG, and vector index."
        warn "Install Redis or point REDIS_ADDR to a remote instance."
    fi
}

# --- 4. Download Go dependencies ---
download_deps() {
    info "Downloading Go module dependencies..."
    go mod download
    info "Dependencies downloaded."
}

# --- 5. Build all binaries ---
build_all() {
    info "Building API, Worker, and CLI binaries..."
    go build -o bin/api ./cmd/api
    go build -o bin/worker ./cmd/worker
    go build -o bin/cli ./cmd/cli
    info "Binaries built in bin/ directory."
}

# --- Main ---
main() {
    info "Starting Aetheris bootstrap..."
    check_go
    check_postgres
    check_redis
    download_deps
    build_all
    info "Bootstrap complete."
    echo
    info "Next steps:"
    echo "  1. Run database migrations: ./scripts/migrate.sh"
    echo "  2. Start services: make run"
    echo "  3. Or use Docker: make docker-run"
}

main "$@"
