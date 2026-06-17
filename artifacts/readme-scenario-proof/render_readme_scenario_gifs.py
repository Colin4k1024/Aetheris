#!/usr/bin/env python3
from __future__ import annotations

import json
import re
import textwrap
from pathlib import Path

from PIL import Image, ImageDraw, ImageFont


ROOT = Path(__file__).resolve().parent
WIDTH = 1280
HEIGHT = 720
PADDING_X = 42
PADDING_Y = 74
LINE_HEIGHT = 26
BG = (17, 24, 39)
PANEL = (7, 13, 24)
PANEL_EDGE = (51, 65, 85)
TEXT = (226, 232, 240)
MUTED = (148, 163, 184)
GREEN = (74, 222, 128)
YELLOW = (250, 204, 21)
BLUE = (96, 165, 250)
RED = (248, 113, 113)


def font(size: int) -> ImageFont.FreeTypeFont:
    for candidate in (
        "/System/Library/Fonts/SFNSMono.ttf",
        "/System/Library/Fonts/Menlo.ttc",
        "/Library/Fonts/Menlo.ttc",
        "/System/Library/Fonts/Courier.ttc",
    ):
        try:
            return ImageFont.truetype(candidate, size)
        except Exception:
            continue
    return ImageFont.load_default()


FONT = font(20)
FONT_BOLD = font(21)
FONT_SMALL = font(16)


def clean_terminal_text(text: str) -> list[str]:
    text = text.replace("\r", "\n")
    text = re.sub(r"\x1b\[[0-9;]*[A-Za-z]", "", text)
    lines = []
    for raw in text.splitlines():
        line = raw.rstrip()
        if not line:
            if lines and lines[-1] != "":
                lines.append("")
            continue
        line = (
            line.replace("✓", "PASS")
            .replace("✗", "FAIL")
            .replace("→", "->")
            .replace("—", "-")
        )
        lines.append(line)
    return lines


def wrap_lines(lines: list[str], width: int = 94) -> list[str]:
    wrapped: list[str] = []
    for line in lines:
        if len(line) <= width:
            wrapped.append(line)
            continue
        chunks = textwrap.wrap(
            line,
            width=width,
            replace_whitespace=False,
            drop_whitespace=False,
            subsequent_indent="  ",
        )
        wrapped.extend(chunks or [""])
    return wrapped


def draw_terminal(lines: list[str], title: str, subtitle: str = "") -> Image.Image:
    image = Image.new("RGB", (WIDTH, HEIGHT), BG)
    draw = ImageDraw.Draw(image)

    panel = (24, 28, WIDTH - 24, HEIGHT - 28)
    draw.rounded_rectangle(panel, radius=14, fill=PANEL, outline=PANEL_EDGE, width=2)
    draw.rounded_rectangle((24, 28, WIDTH - 24, 70), radius=14, fill=(15, 23, 42))
    draw.rectangle((24, 55, WIDTH - 24, 70), fill=(15, 23, 42))
    for i, color in enumerate((RED, YELLOW, GREEN)):
        x = 50 + i * 24
        draw.ellipse((x, 43, x + 12, 55), fill=color)
    draw.text((128, 41), title, fill=TEXT, font=FONT_BOLD)
    if subtitle:
        draw.text((WIDTH - 44 - draw.textlength(subtitle, font=FONT_SMALL), 43), subtitle, fill=MUTED, font=FONT_SMALL)

    y = PADDING_Y
    visible = wrap_lines(lines)
    max_lines = (HEIGHT - PADDING_Y - 46) // LINE_HEIGHT
    visible = visible[-max_lines:]
    for line in visible:
        color = TEXT
        if line.startswith("$"):
            color = GREEN
        elif "PASS" in line or "completed" in line or "status=ok" in line:
            color = GREEN
        elif "events=" in line or "trace" in line or "checkpoint" in line:
            color = BLUE
        elif "FAIL" in line or "error" in line.lower():
            color = RED
        elif line.startswith("#") or line.startswith("Source:"):
            color = MUTED
        draw.text((PADDING_X, y), line, fill=color, font=FONT)
        y += LINE_HEIGHT
    return image


def save_gif(path: Path, frames: list[Image.Image], durations: list[int]) -> None:
    path.parent.mkdir(parents=True, exist_ok=True)
    frames[0].save(
        path,
        save_all=True,
        append_images=frames[1:],
        duration=durations,
        loop=0,
        optimize=True,
    )


def animate_lines(title: str, subtitle: str, lines: list[str], out: Path, chunk: int = 2) -> None:
    frames: list[Image.Image] = []
    durations: list[int] = []
    current: list[str] = []
    for idx in range(0, len(lines), chunk):
        current.extend(lines[idx : idx + chunk])
        frames.append(draw_terminal(current, title, subtitle))
        durations.append(580)
    frames.append(draw_terminal(lines, title, subtitle))
    durations.append(2200)
    save_gif(out, frames, durations)


def make_quickstart_gif() -> None:
    health = json.loads((ROOT / "health-demo.json").read_text())
    api_log = clean_terminal_text((ROOT / "api.demo.log").read_text())
    worker_log = clean_terminal_text((ROOT / "worker.demo.log").read_text())

    selected_api = [line for line in api_log if "JobStore" in line or "HTTP server listening" in line or "API" in line][:4]
    selected_worker = [line for line in worker_log if "Worker Agent Job" in line or "worker" in line.lower()][:4]
    lines = [
        "$ make build",
        "built: bin/api bin/worker bin/aetheris",
        "$ API_CONFIG_PATH=artifacts/readme-scenario-proof/tmp/api.demo.yaml bin/api",
        *selected_api,
        "$ WORKER_CONFIG_PATH=artifacts/readme-scenario-proof/tmp/worker.demo.yaml bin/worker",
        *selected_worker,
        "$ curl http://localhost:8080/api/health",
        json.dumps(health, ensure_ascii=False),
        "PASS README quickstart health endpoint is live.",
        "Source: health-demo.json, api.demo.log, worker.demo.log",
    ]
    animate_lines(
        "README Scenario 1 - Embedded Quickstart",
        "real health check",
        lines,
        ROOT / "readme-quickstart-health.gif",
        chunk=2,
    )


def make_batch_gif() -> None:
    raw = (ROOT / "crash_recovery_demo_success.log").read_text()
    lines = clean_terminal_text(raw)
    keep: list[str] = [
        "$ AGENT_PORT=19001 python3 examples/crash_recovery/demo.py",
        "# 19001 used only because local 9001 is occupied on this machine.",
    ]
    for line in lines:
        if not line.strip("="):
            continue
        if "This example shows" in line or "itself still runs" in line:
            continue
        keep.append(line)
    animate_lines(
        "README Scenario 2 - external_http Batch Demo",
        "real durable job",
        keep,
        ROOT / "readme-external-http-batch-demo.gif",
        chunk=2,
    )


def make_trace_gif() -> None:
    events = json.loads((ROOT / "events.json").read_text())["events"]
    trace = json.loads((ROOT / "trace.json").read_text())
    job = json.loads((ROOT / "job.json").read_text())

    important = {
        "job_created",
        "plan_generated",
        "decision_snapshot",
        "node_started",
        "tool_invocation_started",
        "tool_called",
        "tool_invocation_finished",
        "command_committed",
        "node_finished",
        "state_checkpointed",
        "reasoning_snapshot",
        "job_completed",
    }
    lines = [
        "$ curl /api/jobs/<job-id>/events",
        f"Job: {job['id']}",
        f"Agent: {job['agent_id']}",
        f"Status: {job['status']}",
        f"Recorded events: {len(events)}",
        "",
    ]
    for idx, event in enumerate(events, 1):
        typ = event["type"]
        if typ not in important:
            continue
        payload = event.get("payload") or {}
        detail = ""
        if typ == "tool_invocation_finished":
            detail = " outcome=success"
        elif typ == "node_finished":
            detail = f" result={payload.get('result_type', 'ok')}"
        elif typ == "job_completed":
            detail = " processed_records=20"
        elif typ == "state_checkpointed":
            detail = " changed_keys=external_agent_call"
        lines.append(f"{idx:02d}. {typ}{detail}")

    durations = trace.get("node_durations") or []
    if durations:
        duration = durations[0]
        lines.extend(
            [
                "",
                "$ curl /api/jobs/<job-id>/trace",
                f"Trace node: {duration.get('node_id')} duration_ms={duration.get('duration_ms')}",
                "PASS design loop: plan -> tool -> commit -> checkpoint -> trace.",
            ]
        )
    animate_lines(
        "README Scenario 3 - Audit Trail & Trace",
        "events and trace",
        lines,
        ROOT / "readme-event-trace-proof.gif",
        chunk=2,
    )


def main() -> None:
    make_quickstart_gif()
    make_batch_gif()
    make_trace_gif()


if __name__ == "__main__":
    main()
