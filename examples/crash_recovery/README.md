# Crash Recovery Demo

A self-contained terminal demo that shows Aetheris's core value proposition: **crash recovery**.

The script processes 25 records. At step 16 it simulates a process crash. Because Aetheris checkpoints after every step, recovery resumes at step 16 — skipping the 15 already-completed steps — instead of restarting from zero.

## Requirements

- Python 3.8+
- No external dependencies (stdlib only)

## Quick start

```bash
# Run the demo directly
python3 demo.py

# Run without colour (for piping or CI)
python3 demo.py --no-color
```

## What the demo shows

```
Phase 1 — Initial run
  Processing step 16/25  [============================---] 96%

  💥  PROCESS KILLED at step 16
  ⚠  15 records processed before crash.
  ⚠  Without checkpointing, ALL 25 steps would restart.

Phase 2 — Checkpoint inspection
  Durable checkpoint found:  step 16 of 25
  Steps already completed:   15
  Steps remaining:           10

Phase 3 — Crash recovery
  🔄  Resumed at step 16, skipping 15 completed steps
  Processing step 25/25  [==============================]100%

  ✅  Recovery complete
  Total steps                 25
  Crashed at                  step 16
  Resumed from                step 16
  Steps skipped on recovery   15
```

## Recording a GIF

### Option A — VHS (recommended)

[VHS](https://github.com/charmbracelet/vhs) produces reproducible, themeable GIFs from a `.tape` description file.

```bash
# Generate the tape file
./record-demo.sh tape

# Install VHS
brew install vhs          # macOS
# or: go install github.com/charmbracelet/vhs@latest

# Render the GIF
vhs demo.tape             # produces crash-recovery-demo.gif
```

The generated `demo.tape` can also be uploaded to [vhs.charm.sh](https://vhs.charm.sh) for browser-based rendering.

### Option B — asciinema + agg

```bash
# Record a terminal session
pip install asciinema
asciinema rec demo.cast --command 'python3 demo.py --no-color'

# Convert to GIF
agg demo.cast crash-recovery-demo.gif
```

### Option C — script (raw recording)

```bash
./record-demo.sh record
# Produces demo.typescript — can be replayed with `scriptreplay` or converted
```

### One-step (auto-detect)

```bash
./record-demo.sh          # uses VHS if installed, otherwise generates .tape
```

## Key points for the recording

| Setting | Value | Reason |
|---------|-------|--------|
| Terminal width | 80-100 columns | Fits the progress bar comfortably |
| Font size | 14-16px | Readable in a GIF at 600-800px wide |
| Colour | on (default) | ANSI colours work in VHS and most terminals |
| Duration | ~25 seconds | Compact enough for a README GIF |
| Playback speed | 1.0x | Real-time pacing feels natural |
