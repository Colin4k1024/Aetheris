#!/usr/bin/env python3
"""Render GIF proof files from the customer support agent demo artifacts."""

from __future__ import annotations

import json
import re
import textwrap
from pathlib import Path

from PIL import Image, ImageDraw, ImageFont


ROOT = Path(__file__).resolve().parent
ARTIFACTS = ROOT / "artifacts"
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


def clean(text: str) -> list[str]:
    text = text.replace("\r", "\n")
    text = re.sub(r"\x1b\[[0-9;]*[A-Za-z]", "", text)
    return [line.rstrip() for line in text.splitlines() if line.strip()]


def wrap(lines: list[str], width: int = 94) -> list[str]:
    out: list[str] = []
    for line in lines:
        if len(line) <= width:
            out.append(line)
        else:
            out.extend(
                textwrap.wrap(
                    line,
                    width=width,
                    replace_whitespace=False,
                    drop_whitespace=False,
                    subsequent_indent="  ",
                )
            )
    return out


def draw_terminal(lines: list[str], title: str, subtitle: str) -> Image.Image:
    image = Image.new("RGB", (WIDTH, HEIGHT), BG)
    draw = ImageDraw.Draw(image)
    draw.rounded_rectangle((24, 28, WIDTH - 24, HEIGHT - 28), radius=14, fill=PANEL, outline=PANEL_EDGE, width=2)
    draw.rounded_rectangle((24, 28, WIDTH - 24, 70), radius=14, fill=(15, 23, 42))
    draw.rectangle((24, 55, WIDTH - 24, 70), fill=(15, 23, 42))
    for i, color in enumerate((RED, YELLOW, GREEN)):
        x = 50 + i * 24
        draw.ellipse((x, 43, x + 12, 55), fill=color)
    draw.text((128, 41), title, fill=TEXT, font=FONT_BOLD)
    draw.text((WIDTH - 44 - draw.textlength(subtitle, font=FONT_SMALL), 43), subtitle, fill=MUTED, font=FONT_SMALL)

    y = PADDING_Y
    visible = wrap(lines)
    max_lines = (HEIGHT - PADDING_Y - 46) // LINE_HEIGHT
    for line in visible[-max_lines:]:
        color = TEXT
        lower = line.lower()
        if line.startswith("$"):
            color = GREEN
        elif "pass" in lower or "completed" in lower or "status: ok" in lower:
            color = GREEN
        elif "event" in lower or "trace" in lower or "ticket" in lower:
            color = BLUE
        elif "fail" in lower or "error" in lower:
            color = RED
        elif line.startswith("#"):
            color = MUTED
        draw.text((PADDING_X, y), line, fill=color, font=FONT)
        y += LINE_HEIGHT
    return image


def save_gif(path: Path, frames: list[Image.Image], durations: list[int]) -> None:
    frames[0].save(
        path,
        save_all=True,
        append_images=frames[1:],
        duration=durations,
        loop=0,
        optimize=True,
    )


def animate(title: str, subtitle: str, lines: list[str], out: Path, chunk: int = 2) -> None:
    frames: list[Image.Image] = []
    durations: list[int] = []
    current: list[str] = []
    for idx in range(0, len(lines), chunk):
        current.extend(lines[idx : idx + chunk])
        frames.append(draw_terminal(current, title, subtitle))
        durations.append(620)
    frames.append(draw_terminal(lines, title, subtitle))
    durations.append(2200)
    save_gif(out, frames, durations)


def render_unit_tests() -> None:
    lines = ["$ python3 -m unittest discover -s examples/customer_support_agent"]
    lines.extend(clean((ARTIFACTS / "unit-tests.log").read_text()))
    animate(
        "Customer Support Agent - Unit Tests",
        "policy and idempotency",
        lines,
        ARTIFACTS / "customer-support-agent-unit-tests.gif",
    )


def render_direct_idempotency() -> None:
    lines = clean((ARTIFACTS / "direct-idempotency.log").read_text())
    animate(
        "Customer Support Agent - Direct HTTP",
        "dedupe ledger",
        lines,
        ARTIFACTS / "customer-support-agent-direct-idempotency.gif",
    )


def render_aetheris_trace() -> None:
    demo_lines = clean((ARTIFACTS / "aetheris-job.log").read_text())
    events = json.loads((ARTIFACTS / "events.json").read_text())["events"]
    trace = json.loads((ARTIFACTS / "trace.json").read_text())
    compact = demo_lines[:]
    compact.append("")
    compact.append("$ event chain")
    for index, event in enumerate(events, 1):
        event_type = event.get("type", "")
        if event_type in {
            "job_created",
            "plan_generated",
            "tool_invocation_started",
            "tool_invocation_finished",
            "state_checkpointed",
            "job_completed",
        }:
            compact.append(f"{index:02d}. {event_type}")
    durations = trace.get("node_durations") or []
    if durations:
        compact.append(f"trace node duration_ms={durations[0].get('duration_ms')}")
    animate(
        "Customer Support Agent - Aetheris Job",
        "events and trace",
        compact,
        ARTIFACTS / "customer-support-agent-aetheris-trace.gif",
        chunk=2,
    )


def main() -> None:
    ARTIFACTS.mkdir(parents=True, exist_ok=True)
    render_unit_tests()
    render_direct_idempotency()
    render_aetheris_trace()


if __name__ == "__main__":
    main()
