#!/usr/bin/env bash
#
# record-demo.sh — Record the crash-recovery demo as a terminal session.
#
# Two modes:
#   1. VHS tape (.tape) — generates a reproducible GIF (recommended).
#   2. script recording — captures raw terminal output for later conversion.
#
# Usage:
#   ./record-demo.sh tape      # Generate a .tape file for VHS
#   ./record-demo.sh record    # Record with `script` (macOS / Linux)
#   ./record-demo.sh gif       # Generate GIF directly (requires VHS)
#   ./record-demo.sh           # Same as `gif` if VHS is installed, else `tape`

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
DEMO_PY="${SCRIPT_DIR}/demo.py"
TAPE_FILE="${SCRIPT_DIR}/demo.tape"
OUTPUT_DIR="${SCRIPT_DIR}"
GIF_FILE="${OUTPUT_DIR}/crash-recovery-demo.gif"
TYPESCRIPT="${OUTPUT_DIR}/demo.typescript"

# ── helpers ──────────────────────────────────────────────────────────────────

info()  { printf "\033[1;34m▸ %s\033[0m\n" "$*"; }
ok()    { printf "\033[1;32m✔ %s\033[0m\n" "$*"; }
warn()  { printf "\033[1;33m⚠ %s\033[0m\n" "$*"; }
die()   { printf "\033[1;31m✖ %s\033[0m\n" "$*" >&2; exit 1; }

check_python() {
    if ! command -v python3 &>/dev/null; then
        die "python3 not found"
    fi
}

# ── tape mode ────────────────────────────────────────────────────────────────

generate_tape() {
    info "Generating VHS tape file: ${TAPE_FILE}"
    cat > "${TAPE_FILE}" <<'TAPE'
# Aetheris Crash Recovery Demo — VHS tape
# https://github.com/charmbracelet/vhs
#
# Usage:
#   vhs demo.tape
#   # produces crash-recovery-demo.gif in the same directory

Output crash-recovery-demo.gif

Set Width  900
Set Height 600
Set Theme  Charm
Set FontSize 16
Set TypingSpeed 0ms
Set PlaybackSpeed 1.0

# Give the terminal a moment to initialize
Sleep 500ms

Type "python3 demo.py --no-color"
Enter

# Wait for the demo to finish (~25 seconds)
Sleep 28000ms
TAPE
    ok "Tape file written to ${TAPE_FILE}"
    echo ""
    echo "  To generate the GIF:"
    echo ""
    echo "    brew install vhs        # install VHS (macOS)"
    echo "    vhs ${TAPE_FILE}        # produce the GIF"
    echo ""
    echo "  Or upload the .tape file to https://vhs.charm.sh for online rendering."
    echo ""
}

# ── record mode (script) ────────────────────────────────────────────────────

record_session() {
    info "Recording terminal session with \`script\`..."

    if [[ "$(uname)" == "Darwin" ]]; then
        # macOS: script uses -t for timing
        script -q "${TYPESCRIPT}" -t 0 python3 "${DEMO_PY}"
    else
        # Linux: script uses -t for timing (to stderr), -c for command
        script -q -t 0 -c "python3 ${DEMO_PY}" "${TYPESCRIPT}"
    fi

    ok "Typescript saved to ${TYPESCRIPT}"
    echo ""
    echo "  To convert to GIF, install and use \`asciinema\` + \`agg\`:"
    echo ""
    echo "    pip install asciinema"
    echo "    asciinema rec demo.cast --command 'python3 ${DEMO_PY}'"
    echo "    agg demo.cast crash-recovery-demo.gif"
    echo ""
}

# ── gif mode (direct VHS) ───────────────────────────────────────────────────

generate_gif() {
    if ! command -v vhs &>/dev/null; then
        warn "VHS not found. Falling back to tape generation."
        generate_tape
        return
    fi

    generate_tape
    info "Running VHS to produce GIF..."
    (cd "${SCRIPT_DIR}" && vhs demo.tape)
    ok "GIF saved to ${GIF_FILE}"
}

# ── main ─────────────────────────────────────────────────────────────────────

main() {
    check_python

    local mode="${1:-auto}"
    case "${mode}" in
        tape)
            generate_tape
            ;;
        record)
            record_session
            ;;
        gif)
            generate_gif
            ;;
        auto)
            if command -v vhs &>/dev/null; then
                generate_gif
            else
                generate_tape
            fi
            ;;
        *)
            die "Unknown mode: ${mode}. Use: tape | record | gif"
            ;;
    esac
}

main "$@"
