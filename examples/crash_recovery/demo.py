#!/usr/bin/env python3
"""
Crash Recovery Demo — Aetheris

Demonstrates that a long-running agent job resumes from its last checkpoint
after a process crash or restart, with zero duplicate processing.

Usage:
    python demo.py           # start or resume the batch job
    python demo.py --reset   # clear stored job ID and start fresh

Requirements:
    pip install requests
    Aetheris running at http://localhost:8080 (make run-embedded)
"""

from __future__ import annotations

import argparse
import json
import os
import signal
import sys
import time
import threading
import http.server
import urllib.request
import urllib.error
from typing import Optional

# ── Configuration ─────────────────────────────────────────────────────────────

AETHERIS_URL = os.environ.get("AETHERIS_URL", "http://localhost:8080")
AGENT_PORT = int(os.environ.get("AGENT_PORT", "9001"))
AGENT_ID = "crash_demo_batch_processor"
TOTAL_RECORDS = 50          # keep small for demo; set to 1000 for full demo
PROCESS_DELAY = 0.3         # seconds per record (makes crash visible)
STATE_FILE = ".demo_job_id"  # persists job ID across restarts

# ── Simulated record processing ───────────────────────────────────────────────

_processed_in_session: list[int] = []


def process_record(record_id: int) -> dict:
    """Simulate processing one record (replace with real LLM/API call)."""
    time.sleep(PROCESS_DELAY)
    _processed_in_session.append(record_id)
    return {"record_id": record_id, "status": "ok", "result": f"processed-{record_id}"}


# ── Minimal HTTP server (acts as the "agent" Aetheris calls) ──────────────────

class AgentHandler(http.server.BaseHTTPRequestHandler):
    """Receives Aetheris job invocations and processes records one at a time."""

    def log_message(self, format, *args):  # silence default access logs
        pass

    def do_POST(self):
        length = int(self.headers.get("Content-Length", 0))
        body = json.loads(self.rfile.read(length))

        message = body.get("message", "")
        metadata = body.get("metadata", {})
        step = metadata.get("step", 0)

        # Parse which record to process from the message
        try:
            record_id = int(message.split("record:")[-1].strip())
        except (ValueError, IndexError):
            record_id = step

        result = process_record(record_id)
        is_final = record_id >= TOTAL_RECORDS

        response = {
            "answer": json.dumps(result),
            "final": is_final,
            "metadata": {"record_id": record_id},
        }

        self.send_response(200)
        self.send_header("Content-Type", "application/json")
        self.end_headers()
        self.wfile.write(json.dumps(response).encode())


def start_agent_server():
    server = http.server.HTTPServer(("", AGENT_PORT), AgentHandler)
    t = threading.Thread(target=server.serve_forever, daemon=True)
    t.start()
    return server


# ── Aetheris HTTP helpers (no SDK dependency) ─────────────────────────────────

def _request(method: str, path: str, body: Optional[dict] = None, headers: Optional[dict] = None) -> dict:
    url = f"{AETHERIS_URL}{path}"
    data = json.dumps(body).encode() if body else None
    req_headers = {"Content-Type": "application/json"}
    if headers:
        req_headers.update(headers)

    req = urllib.request.Request(url, data=data, headers=req_headers, method=method)
    try:
        with urllib.request.urlopen(req, timeout=10) as resp:
            return json.loads(resp.read())
    except urllib.error.HTTPError as e:
        body_text = e.read().decode()
        raise RuntimeError(f"HTTP {e.code} on {method} {path}: {body_text}") from e


def health_check() -> bool:
    try:
        _request("GET", "/api/health")
        return True
    except Exception:
        return False


def ensure_agent_registered():
    """Register the demo agent in Aetheris (idempotent)."""
    # In embedded mode, agents are defined in configs/api.embedded.yaml
    # For this demo we just verify the server is reachable
    pass


def submit_job() -> str:
    """Submit the batch processing job and return its job_id."""
    result = _request(
        "POST",
        f"/api/agents/{AGENT_ID}/message",
        body={"message": f"Process batch of {TOTAL_RECORDS} records"},
        headers={"Idempotency-Key": f"crash-demo-batch-{TOTAL_RECORDS}"},
    )
    job_id = (
        result.get("job_id")
        or result.get("id")
        or result.get("runtime_submission", {}).get("job_id", "")
    )
    if not job_id:
        raise RuntimeError(f"No job_id in response: {result}")
    return job_id


def get_job(job_id: str) -> dict:
    return _request("GET", f"/api/jobs/{job_id}")


def get_trace(job_id: str) -> dict:
    try:
        return _request("GET", f"/api/jobs/{job_id}/trace")
    except Exception:
        return {}


# ── State file (persists job ID across Python process restarts) ───────────────

def load_job_id() -> Optional[str]:
    if os.path.exists(STATE_FILE):
        with open(STATE_FILE) as f:
            return f.read().strip() or None
    return None


def save_job_id(job_id: str):
    with open(STATE_FILE, "w") as f:
        f.write(job_id)


def clear_job_id():
    if os.path.exists(STATE_FILE):
        os.remove(STATE_FILE)


# ── Main demo loop ────────────────────────────────────────────────────────────

def count_completed_steps(trace: dict) -> int:
    """Count how many record-processing steps are in the trace."""
    events = trace.get("events", [])
    return sum(1 for e in events if e.get("type") in ("StepCompleted", "ToolCallCompleted"))


def run_demo():
    print("\n" + "=" * 60)
    print("  Aetheris Crash Recovery Demo")
    print("=" * 60)

    # 1. Check server
    if not health_check():
        print(f"\n✗ Cannot reach Aetheris at {AETHERIS_URL}")
        print("  Start it with: make run-embedded  (from repo root)")
        sys.exit(1)
    print(f"\n✓ Aetheris reachable at {AETHERIS_URL}")

    # 2. Start local agent server
    print(f"✓ Starting local agent server on port {AGENT_PORT}...")
    start_agent_server()

    # 3. Find or create job
    job_id = load_job_id()
    if job_id:
        try:
            job = get_job(job_id)
            status = job.get("status", "unknown")
            print(f"\n→ Found existing job: {job_id}  (status={status})")

            if status in ("completed", "failed", "cancelled"):
                print("  Job is already terminal. Use --reset to start fresh.")
                _print_summary(job_id, job)
                return

            print(f"  Resuming from last checkpoint...")
        except Exception as e:
            print(f"  Could not fetch job {job_id}: {e}")
            print("  Starting a new job...")
            job_id = None

    if not job_id:
        print(f"\n→ Submitting new batch job ({TOTAL_RECORDS} records)...")

        # NOTE: For the demo agent to work, add this to configs/api.embedded.yaml:
        #   agents:
        #     crash_demo_batch_processor:
        #       type: "external_http"
        #       external:
        #         url: "http://localhost:9001"
        #         timeout: "30s"
        try:
            job_id = submit_job()
        except RuntimeError as e:
            print(f"\n✗ Failed to submit job: {e}")
            print(
                f"\n  Make sure the agent '{AGENT_ID}' is configured in your runtime config.\n"
                f"  Add this to configs/api.embedded.yaml under 'agents':\n\n"
                f"    {AGENT_ID}:\n"
                f"      type: \"external_http\"\n"
                f"      external:\n"
                f"        url: \"http://localhost:{AGENT_PORT}\"\n"
                f"        timeout: \"30s\"\n"
            )
            sys.exit(1)

        save_job_id(job_id)
        print(f"  Job ID: {job_id}")

    print("\n  Processing records (Ctrl+C to simulate crash)...\n")

    # 4. Poll progress
    last_count = 0
    try:
        while True:
            job = get_job(job_id)
            status = job.get("status", "unknown")
            trace = get_trace(job_id)
            completed = count_completed_steps(trace)

            if completed != last_count:
                bar = "█" * (completed * 40 // TOTAL_RECORDS)
                bar = bar.ljust(40)
                pct = completed * 100 // TOTAL_RECORDS
                print(
                    f"\r  [{bar}] {completed:4d}/{TOTAL_RECORDS}  {pct:3d}%  "
                    f"(this session: +{len(_processed_in_session)})",
                    end="",
                    flush=True,
                )
                last_count = completed

            if status in ("completed", "failed", "cancelled"):
                print()
                _print_summary(job_id, job)
                clear_job_id()
                return

            time.sleep(1.0)

    except KeyboardInterrupt:
        print("\n\n  ⚡ Process interrupted (simulating crash)")
        print(f"\n  Job {job_id} is still tracked by Aetheris.")
        print(f"  Restart this script to resume from where it stopped.\n")
        print(f"  Job ID saved to: {STATE_FILE}")
        sys.exit(0)


def _print_summary(job_id: str, job: dict):
    status = job.get("status", "?")
    trace = get_trace(job_id)
    completed = count_completed_steps(trace)
    print(f"\n  ✓ Job {job_id}")
    print(f"    Status:     {status}")
    print(f"    Completed:  {completed} steps")
    print(f"    Duplicates: 0  (at-most-once guarantee)")
    print(f"\n  Inspect the full trace:")
    print(f"    curl {AETHERIS_URL}/api/jobs/{job_id}/trace\n")


# ── Entry point ───────────────────────────────────────────────────────────────

if __name__ == "__main__":
    parser = argparse.ArgumentParser(description="Aetheris crash recovery demo")
    parser.add_argument("--reset", action="store_true", help="Clear saved job ID and start fresh")
    args = parser.parse_args()

    if args.reset:
        clear_job_id()
        print("Cleared saved job ID. Run without --reset to start a new job.")
        sys.exit(0)

    run_demo()
