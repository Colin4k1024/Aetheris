#!/bin/bash
# Aetheris Basic Agent Example — Quick Run
#
# This example demonstrates creating and running an eino agent.
# For production use with crash recovery, see examples/ with Aetheris runtime.
#
# Prerequisites:
#   - Go 1.25+
#   - DASHSCOPE_API_KEY (Qwen, recommended) OR OPENAI_API_KEY

set -e

cd "$(dirname "$0")"

echo "=== Aetheris Basic Agent Example ==="
echo ""

# Check for API keys
if [[ -z "$DASHSCOPE_API_KEY" && -z "$OPENAI_API_KEY" ]]; then
    echo "⚠️  No API key found. Set one of:"
    echo "    export DASHSCOPE_API_KEY=sk-...   # Qwen (recommended, free tier available)"
    echo "    export OPENAI_API_KEY=sk-...      # OpenAI"
    echo ""
    echo "Using mock mode for demonstration..."
fi

# Use Qwen by default if available, else OpenAI
if [[ -n "$DASHSCOPE_API_KEY" ]]; then
    export MODEL_PROVIDER="qwen"
    echo "✅ Using Qwen (DASHSCOPE_API_KEY)"
elif [[ -n "$OPENAI_API_KEY" ]]; then
    export MODEL_PROVIDER="openai"
    echo "✅ Using OpenAI (OPENAI_API_KEY)"
fi

echo ""
echo "Running agent..."
echo ""

# Run the agent
go run . "$@"
