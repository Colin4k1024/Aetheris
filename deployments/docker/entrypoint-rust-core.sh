#!/bin/bash
# Entrypoint for Aetheris Rust Core containers.
# Schema is initialized externally via docker-compose init scripts or manual psql.
# This script just starts the service.

set -e

echo "=== Aetheris Rust Core ==="
echo "AETHERIS_USE_RUST_CORE=${AETHERIS_USE_RUST_CORE:-false}"
echo "JOBSTORE_DSN=${JOBSTORE_DSN:-not set}"

# Execute the main command
exec "$@"
