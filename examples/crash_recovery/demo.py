#!/usr/bin/env python3
"""
External HTTP durability-boundary demo — Aetheris

This example runs a slow batch processor behind the ``external_http`` adapter.
Aetheris tracks the outer job submission, retries, and event trail, but because
the whole batch runs inside one HTTP call it does not checkpoint per record
inside the external service.

Usage:
    python demo.py

Requirements:
    pip install requests
    Aetheris running at http://localhost:8080 (make run-embedded)
"""

from __future__ import annotations

import http.server
import json
import os
import sys
import threading
import time
import urllib.error
import urllib.request
from typing import Optional

AETHERIS_URL = os.environ.get("AETHERIS_URL", "http://localhost:8080")
AGENT_PORT = int(os.environ.get("AGENT_PORT", "9001"))
AGENT_ID = "crash_demo_batch_processor"
TOTAL_RECORDS = 20
PROCESS_DELAY = 0.2

_processed_in_session: list[int] = []


def process_batch(total_records: int) -> dict:
    """Simulate a slow batch processor inside one external_http invocation."""
    for record_id in range(1, total_records + 1):
        time.sleep(PROCESS_DELAY)
        _processed_in_session.append(record_id)
        print(
            f"\r  local agent processed {record_id:3d}/{total_records}",
            end="",
            flush=True,
        )

    print()
    return {
        "processed_records": total_records,
        "last_record_id": total_records,
        "session_records": len(_processed_in_session),
    }


class AgentHandler(http.server.BaseHTTPRequestHandler):
    """Receives one Aetheris invocation and processes the whole batch."""

    def log_message(self, format, *args):
        pass

    def do_POST(self):
        length = int(self.headers.get("Content-Length", 0))
        body = json.loads(self.rfile.read(length) or b"{}")
        message = body.get("message", "")

        print(f"\n→ external_http request received: {message}")
        result = process_batch(TOTAL_RECORDS)
        response = {
            "answer": json.dumps(result),
            "final": True,
            "metadata": result,
        }

        encoded = json.dumps(response).encode()
        self.send_response(200)
        self.send_header("Content-Type", "application/json")
        self.send_header("Content-Length", str(len(encoded)))
        self.end_headers()
        self.wfile.write(encoded)


def start_agent_server():
    server = http.server.HTTPServer(("", AGENT_PORT), AgentHandler)
    thread = threading.Thread(target=server.serve_forever, daemon=True)
    thread.start()
    return server


def _request(method: str, path: str, body: Optional[dict] = None, headers: Optional[dict] = None) -> dict:
    url = f"{AETHERIS_URL}{path}"
    data = json.dumps(body).encode() if body else None
    req_headers = {"Content-Type": "application/json"}
    if headers:
        req_headers.update(headers)

    request = urllib.request.Request(url, data=data, headers=req_headers, method=method)
    try:
        with urllib.request.urlopen(request, timeout=10) as response:
            return json.loads(response.read())
    except urllib.error.HTTPError as exc:
        body_text = exc.read().decode()
        raise RuntimeError(f"HTTP {exc.code} on {method} {path}: {body_text}") from exc


def health_check() -> bool:
    try:
        _request("GET", "/api/health")
        return True
    except Exception:
        return False


def submit_job() -> str:
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


def get_events(job_id: str) -> dict:
    return _request("GET", f"/api/jobs/{job_id}/events")


def run_demo():
    print("\n" + "=" * 60)
    print("  Aetheris External HTTP Durability Demo")
    print("=" * 60)
    print(
        "\n  This example shows the reliability boundary of external_http:\n"
        "  Aetheris tracks the outer job and event trail, while the batch logic\n"
        "  itself still runs inside one external HTTP call."
    )

    if not health_check():
        print(f"\n✗ Cannot reach Aetheris at {AETHERIS_URL}")
        print("  Start it with: make run-embedded  (from repo root)")
        sys.exit(1)
    print(f"\n✓ Aetheris reachable at {AETHERIS_URL}")

    print(f"✓ Starting local external_http demo agent on port {AGENT_PORT}...")
    server = start_agent_server()

    try:
        print(f"\n→ Submitting batch job ({TOTAL_RECORDS} records)...")
        job_id = submit_job()
        print(f"  Job ID: {job_id}")
        print("\n  Monitoring job status while the external agent processes the batch...\n")

        last_status = None
        last_event_count = -1

        while True:
            job = get_job(job_id)
            status = job.get("status", "unknown")
            events = get_events(job_id)
            event_count = len(events.get("events", []))

            if status != last_status or event_count != last_event_count:
                print(
                    f"\r  job status={status:<10} events={event_count:<3} local records={len(_processed_in_session):<3}",
                    end="",
                    flush=True,
                )
                last_status = status
                last_event_count = event_count

            if status in ("completed", "failed", "cancelled"):
                print()
                _print_summary(job_id, job, event_count)
                return

            time.sleep(1.0)
    finally:
        server.shutdown()
        server.server_close()


def _print_summary(job_id: str, job: dict, event_count: int):
    print(f"\n  ✓ Job {job_id}")
    print(f"    Status:        {job.get('status', '?')}")
    print(f"    Output:        {job.get('output', '(none yet)')}")
    print(f"    Event count:   {event_count}")
    print(f"    Local records: {len(_processed_in_session)}")
    print("\n  Inspect the event stream and trace:")
    print(f"    curl {AETHERIS_URL}/api/jobs/{job_id}/events")
    print(f"    curl {AETHERIS_URL}/api/jobs/{job_id}/trace\n")


if __name__ == "__main__":
    run_demo()
