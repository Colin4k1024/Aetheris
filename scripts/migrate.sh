#!/usr/bin/env bash
# migrate.sh — Apply PostgreSQL schema migrations for Aetheris JobStore.
# Uses internal/runtime/jobstore/schema.sql by default.
#
# Usage:
#   ./scripts/migrate.sh                     # uses DATABASE_URL env or defaults
#   ./scripts/migrate.sh "host=localhost dbname=aetheris"
#   DATABASE_URL=postgres://user:pass@host:5432/db ./scripts/migrate.sh
set -euo pipefail

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

info()  { echo -e "${GREEN}[migrate]${NC} $*"; }
warn()  { echo -e "${YELLOW}[migrate]${NC} $*"; }
fatal() { echo -e "${RED}[migrate]${NC} $*"; exit 1; }

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
SCHEMA_FILE="${PROJECT_ROOT}/internal/runtime/jobstore/schema.sql"

# --- Determine connection DSN ---
DSN="${1:-${DATABASE_URL:-}}"

if [ -z "${DSN}" ]; then
    # Fallback defaults for local development
    PG_HOST="${PGHOST:-localhost}"
    PG_PORT="${PGPORT:-5432}"
    PG_USER="${PGUSER:-postgres}"
    PG_DB="${PGDATABASE:-aetheris}"
    DSN="host=${PG_HOST} port=${PG_PORT} user=${PG_USER} dbname=${PG_DB} sslmode=disable"
    info "Using default DSN: ${DSN}"
fi

# --- Check psql is available ---
if ! command -v psql >/dev/null 2>&1; then
    fatal "psql not found. Install PostgreSQL client tools or run schema.sql manually."
fi

# --- Check schema file exists ---
if [ ! -f "${SCHEMA_FILE}" ]; then
    fatal "Schema file not found: ${SCHEMA_FILE}"
fi

info "Schema file: ${SCHEMA_FILE}"
info "Applying migrations..."

# --- Apply schema ---
if psql "${DSN}" -v ON_ERROR_STOP=1 -f "${SCHEMA_FILE}"; then
    info "Migrations applied successfully."
else
    fatal "Migration failed. Check your DATABASE_URL / PostgreSQL connection."
fi

# --- Verify tables exist ---
info "Verifying tables..."
TABLES=$(psql "${DSN}" -t -c "SELECT tablename FROM pg_tables WHERE tablename IN ('job_events', 'job_claims', 'jobs') ORDER BY tablename;")
TABLE_COUNT=$(echo "${TABLES}" | grep -c -e 'job_events' -e 'job_claims' -e 'jobs' || true)
if [ "${TABLE_COUNT}" -ge 3 ]; then
    info "All expected tables present: job_events, job_claims, jobs"
else
    warn "Some tables may be missing. Check output above."
fi

info "Migration complete."
