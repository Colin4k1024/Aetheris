#!/usr/bin/env python3
"""
Crash Recovery Demo — Aetheris

A self-contained visual demo that simulates Aetheris's crash-recovery
capability.  No external server required — it runs entirely in-process
to produce a convincing terminal recording.

Usage:
    python demo.py              # full-color output
    python demo.py --no-color   # plain text (for pipe / CI)

The demo processes 25 "records".  At step 16 the process "crashes".
Aetheris's durable checkpoint lets it resume at step 16 instead of
restarting from zero.  Total wall-clock time: ~20-25 seconds.
"""

from __future__ import annotations

import argparse
import sys
import time

# ---------------------------------------------------------------------------
# ANSI helpers
# ---------------------------------------------------------------------------

_COLOR_ENABLED = True


def _c(code: str, text: str) -> str:
    """Wrap *text* in ANSI escape sequences when colour is on."""
    if not _COLOR_ENABLED:
        return text
    return f"\033[{code}m{text}\033[0m"


def bold(t: str) -> str:
    return _c("1", t)


def dim(t: str) -> str:
    return _c("2", t)


def red(t: str) -> str:
    return _c("1;31", t)


def green(t: str) -> str:
    return _c("1;32", t)


def yellow(t: str) -> str:
    return _c("1;33", t)


def cyan(t: str) -> str:
    return _c("1;36", t)


def blue(t: str) -> str:
    return _c("1;34", t)


def magenta(t: str) -> str:
    return _c("1;35", t)


# ---------------------------------------------------------------------------
# Progress bar
# ---------------------------------------------------------------------------

def progress_bar(current: int, total: int, width: int = 30) -> str:
    filled = int(width * current / total)
    bar = green("=") * filled + dim("-") * (width - filled)
    pct = f"{current * 100 // total:3d}%"
    return f"[{bar}] {pct}"


# ---------------------------------------------------------------------------
# Checkpoint store (simulates durable storage)
# ---------------------------------------------------------------------------

class CheckpointStore:
    """Minimal checkpoint store backed by a plain dict."""

    def __init__(self) -> None:
        self._data: dict[str, int] = {}

    def save(self, job_id: str, step: int) -> None:
        self._data[job_id] = step

    def load(self, job_id: str) -> int:
        return self._data.get(job_id, 0)

    def has_checkpoint(self, job_id: str) -> bool:
        return job_id in self._data


# ---------------------------------------------------------------------------
# Simulated crash
# ---------------------------------------------------------------------------

class SimulatedCrash(Exception):
    """Raised to simulate a process crash at a specific step."""


# ---------------------------------------------------------------------------
# Processing logic
# ---------------------------------------------------------------------------

TOTAL_STEPS = 25
CRASH_AT_STEP = 16
STEP_DELAY = 0.6          # seconds per step (first run)
RECOVERY_STEP_DELAY = 0.4  # seconds per step (recovery run — slightly faster)
JOB_ID = "batch-job-20260704"


def process_records(
    checkpoint: CheckpointStore,
    *,
    crash: bool = True,
    start_step: int = 0,
) -> int:
    """Process records from *start_step*, optionally crashing at CRASH_AT_STEP.

    Returns the last completed step.
    """
    for step in range(start_step + 1, TOTAL_STEPS + 1):
        # Simulate work
        delay = STEP_DELAY if crash else RECOVERY_STEP_DELAY
        time.sleep(delay)

        # Progress line
        tag = dim("(recovery)") if not crash and step <= start_step + 1 else ""
        label = f"Processing step {step}/{TOTAL_STEPS}"
        bar = progress_bar(step, TOTAL_STEPS)
        sys.stdout.write(f"\r  {cyan(label)}  {bar}  {tag}")
        sys.stdout.flush()

        # Persist checkpoint after each step
        checkpoint.save(JOB_ID, step)

        # Simulate crash at the designated step
        if crash and step == CRASH_AT_STEP:
            print()
            raise SimulatedCrash(f"killed at step {step}")

    print()  # newline after progress bar
    return TOTAL_STEPS


# ---------------------------------------------------------------------------
# Demo orchestration
# ---------------------------------------------------------------------------

def banner() -> None:
    w = 62
    print()
    print(bold("=" * w))
    print(bold("  Aetheris — Crash Recovery Demo"))
    print(bold("=" * w))
    print()
    print("  Aetheris treats agents as durable virtual processes.")
    print("  When a process crashes, it resumes from the last")
    print("  checkpoint — not from zero.")
    print()
    print(dim(f"  {'Scenario':<14} Process {TOTAL_STEPS} records, crash at #{CRASH_AT_STEP}"))
    print(dim(f"  {'Expected':<14} Recovery resumes at #{CRASH_AT_STEP}, skipping 1-{CRASH_AT_STEP - 1}"))
    print()


def phase_header(phase: str, icon: str) -> None:
    print(bold(f"  {icon}  {phase}"))
    print(dim("  " + "-" * 50))


def run_demo() -> None:
    checkpoint = CheckpointStore()

    # ------------------------------------------------------------------
    # Phase 1: first run — processes until crash
    # ------------------------------------------------------------------
    phase_header("Phase 1 — Initial run", "▶")

    try:
        process_records(checkpoint, crash=True)
    except SimulatedCrash:
        pass  # expected

    # Crash banner
    print()
    crash_msg = f"💥  PROCESS KILLED at step {CRASH_AT_STEP}"
    print(red(f"  {crash_msg}"))
    print(red(f"  {'=' * len(crash_msg)}"))
    print()
    print(yellow(f"  ⚠  {CRASH_AT_STEP - 1} records processed before crash."))
    print(yellow(f"  ⚠  Without checkpointing, ALL {TOTAL_STEPS} steps would restart."))
    print()

    time.sleep(1.0)

    # ------------------------------------------------------------------
    # Phase 2: show checkpoint exists
    # ------------------------------------------------------------------
    phase_header("Phase 2 — Checkpoint inspection", "🔍")

    saved_step = checkpoint.load(JOB_ID)
    print(f"  Durable checkpoint found:  {green(f'step {saved_step}')} of {TOTAL_STEPS}")
    print(f"  Steps already completed:   {green(str(saved_step - 1))}")
    print(f"  Steps remaining:           {yellow(str(TOTAL_STEPS - saved_step + 1))}")
    print()

    time.sleep(0.8)

    # ------------------------------------------------------------------
    # Phase 3: recovery — resumes from checkpoint
    # ------------------------------------------------------------------
    phase_header("Phase 3 — Crash recovery", "🔄")

    resume_msg = f"Resumed at step {saved_step}, skipping {saved_step - 1} completed steps"
    print(green(f"  🔄  {resume_msg}"))
    print()

    process_records(checkpoint, crash=False, start_step=saved_step - 1)

    # ------------------------------------------------------------------
    # Summary
    # ------------------------------------------------------------------
    print()
    print(bold("=" * 62))
    print(bold("  ✅  Recovery complete"))
    print(bold("=" * 62))
    print()
    print(f"  {'Total steps':<28} {TOTAL_STEPS}")
    print(f"  {'Crashed at':<28} step {CRASH_AT_STEP}")
    print(f"  {'Resumed from':<28} step {saved_step}")
    print(f"  {'Steps skipped on recovery':<28} {saved_step - 1}")
    print(f"  {'Checkpoint saves':<28} {TOTAL_STEPS} (one per step)")
    print()
    print(dim("  Without Aetheris: restart from step 1 — waste {w} steps.".format(
        w=CRASH_AT_STEP - 1,
    )))
    print(green("  With    Aetheris: resume  from step {s} — zero wasted work.".format(
        s=saved_step,
    )))
    print()
    print(dim("  Key takeaway: Aetheris persists execution state after every"))
    print(dim("  step.  A crash is an interruption, not a restart."))
    print()


# ---------------------------------------------------------------------------
# CLI
# ---------------------------------------------------------------------------

def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(
        description="Aetheris crash-recovery demo (self-contained, no server needed)",
    )
    parser.add_argument(
        "--no-color",
        action="store_true",
        default=False,
        help="Disable ANSI colour codes (useful for piping or CI)",
    )
    return parser.parse_args()


def main() -> None:
    global _COLOR_ENABLED
    args = parse_args()
    if args.no_color:
        _COLOR_ENABLED = False
    run_demo()


if __name__ == "__main__":
    main()
