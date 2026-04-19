#!/bin/bash
# MCP Gateway Example — Quick Run
#
# Demonstrates MCP Gateway tools (GitHub, Filesystem, Web Search, Database)
#
# Prerequisites:
#   - Go 1.25+
#   - GITHUB_TOKEN (optional, for GitHub tool)

set -e
cd "$(dirname "$0")"

echo "=== Aetheris MCP Gateway Example ==="
echo ""

# GitHub token (optional)
if [[ -z "$GITHUB_TOKEN" ]]; then
    echo "⚠️  GITHUB_TOKEN not set — GitHub tool will use mock mode"
else
    echo "✅ GITHUB_TOKEN found — using real GitHub API"
fi

echo ""
echo "Running MCP Gateway demo..."
echo ""

go run . "$@"
