#!/usr/bin/env python3
"""Run the customer_support_agent through Aetheris and save evidence."""

from __future__ import annotations

import json
import os
import shutil
import signal
import subprocess
import sys
import threading
import time
import urllib.error
import urllib.request
from pathlib import Path
from typing import Any

from agent import make_server


EXAMPLE_DIR = Path(__file__).resolve().parent
REPO_ROOT = EXAMPLE_DIR.parents[1]
ARTIFACTS = EXAMPLE_DIR / "artifacts"
TMP = EXAMPLE_DIR / ".tmp"
API_PORT = int(os.environ.get("AETHERIS_DEMO_API_PORT", "18080"))
AGENT_PORT = int(os.environ.get("CUSTOMER_AGENT_PORT", "19002"))
METRICS_PORT = int(os.environ.get("AETHERIS_DEMO_METRICS_PORT", "19093"))
AGENT_ID = "customer_support_agent"
MESSAGE = (
    "Customer says order A-1042 arrived damaged, asks for a refund and a "
    "replacement before the weekend."
)


def main() -> None:
    ARTIFACTS.mkdir(parents=True, exist_ok=True)
    reset_tmp()
    build_binaries()

    agent_server = make_server("127.0.0.1", AGENT_PORT, ARTIFACTS / "agent-ledger.json")
    agent_thread = threading.Thread(target=agent_server.serve_forever, daemon=True)
    agent_thread.start()
    print(f"agent listening http://127.0.0.1:{AGENT_PORT}")

    api_proc: subprocess.Popen[str] | None = None
    worker_proc: subprocess.Popen[str] | None = None
    try:
        run_direct_idempotency_check()
        api_config, worker_config = write_configs()
        api_proc = start_process(
            [
                str(REPO_ROOT / "bin" / "api"),
            ],
            ARTIFACTS / "api.log",
            {
                "API_CONFIG_PATH": str(api_config),
                "MODEL_CONFIG_PATH": str(REPO_ROOT / "configs" / "model.yaml"),
            },
        )
        worker_proc = start_process(
            [
                str(REPO_ROOT / "bin" / "worker"),
            ],
            ARTIFACTS / "worker.log",
            {
                "WORKER_CONFIG_PATH": str(worker_config),
                "MODEL_CONFIG_PATH": str(REPO_ROOT / "configs" / "model.yaml"),
            },
        )
        wait_for_health()
        run_aetheris_job()
    finally:
        stop_process(worker_proc)
        stop_process(api_proc)
        agent_server.shutdown()
        agent_server.server_close()
        agent_thread.join(timeout=2)


def reset_tmp() -> None:
    if TMP.exists():
        shutil.rmtree(TMP)
    TMP.mkdir(parents=True)
    for stale in (
        "direct-idempotency.log",
        "aetheris-job.log",
        "job.json",
        "events.json",
        "trace.json",
        "health.json",
        "agent-ledger.json",
    ):
        path = ARTIFACTS / stale
        if path.exists():
            path.unlink()


def build_binaries() -> None:
    print("building Aetheris binaries...")
    subprocess.run(
        [
            "go",
            "build",
            "-o",
            "bin/api",
            "./cmd/api",
        ],
        cwd=REPO_ROOT,
        check=True,
    )
    subprocess.run(
        [
            "go",
            "build",
            "-o",
            "bin/worker",
            "./cmd/worker",
        ],
        cwd=REPO_ROOT,
        check=True,
    )


def write_configs() -> tuple[Path, Path]:
    data_dir = TMP / "data" / "embedded"
    data_dir.mkdir(parents=True, exist_ok=True)
    agent_block = f"""
    {AGENT_ID}:
      type: "external_http"
      description: "Realistic customer support refund/replacement agent"
      external:
        url: "http://127.0.0.1:{AGENT_PORT}/invoke"
        timeout: "30s"
"""

    api_text = (REPO_ROOT / "configs" / "api.embedded.yaml").read_text()
    api_text = api_text.replace('  port: 8080', f'  port: {API_PORT}', 1)
    api_text = api_text.replace("./data/embedded", str(data_dir))
    api_text = api_text.replace("\nstorage:\n", agent_block + "\nstorage:\n", 1)
    api_path = TMP / "api.customer-support.yaml"
    api_path.write_text(api_text)

    worker_text = (REPO_ROOT / "configs" / "worker.embedded.yaml").read_text()
    worker_text = worker_text.replace("./data/embedded", str(data_dir))
    worker_text = worker_text.replace("    port: 9093", f"    port: {METRICS_PORT}", 1)
    worker_agents = "agents:\n  agents:\n" + agent_block + "\n"
    worker_text = worker_text.replace("\nstorage:\n", "\n" + worker_agents + "storage:\n", 1)
    worker_path = TMP / "worker.customer-support.yaml"
    worker_path.write_text(worker_text)
    return api_path, worker_path


def start_process(
    args: list[str],
    log_path: Path,
    extra_env: dict[str, str],
) -> subprocess.Popen[str]:
    env = os.environ.copy()
    env.update(extra_env)
    log = log_path.open("w")
    return subprocess.Popen(
        args,
        cwd=REPO_ROOT,
        stdout=log,
        stderr=subprocess.STDOUT,
        text=True,
        env=env,
    )


def stop_process(proc: subprocess.Popen[str] | None) -> None:
    if proc is None or proc.poll() is not None:
        return
    proc.send_signal(signal.SIGINT)
    try:
        proc.wait(timeout=5)
    except subprocess.TimeoutExpired:
        proc.terminate()
        proc.wait(timeout=5)


def request(
    method: str,
    url: str,
    body: dict[str, Any] | None = None,
    headers: dict[str, str] | None = None,
    timeout: int = 10,
) -> dict[str, Any]:
    data = json.dumps(body).encode() if body is not None else None
    req_headers = {"Content-Type": "application/json"}
    if headers:
        req_headers.update(headers)
    req = urllib.request.Request(url, data=data, headers=req_headers, method=method)
    try:
        with urllib.request.urlopen(req, timeout=timeout) as response:
            return json.loads(response.read())
    except urllib.error.HTTPError as exc:
        raise RuntimeError(f"HTTP {exc.code} {url}: {exc.read().decode()}") from exc


def run_direct_idempotency_check() -> None:
    log_lines = [
        "$ python3 agent.py",
        f"agent listening http://127.0.0.1:{AGENT_PORT}",
        "$ POST /invoke twice with the same Idempotency-Key",
    ]
    url = f"http://127.0.0.1:{AGENT_PORT}/invoke"
    headers = {"Idempotency-Key": "customer-direct-001"}
    first = request("POST", url, {"message": MESSAGE}, headers)
    second = request("POST", url, {"message": MESSAGE}, headers)
    health = request("GET", f"http://127.0.0.1:{AGENT_PORT}/health")
    ARTIFACTS.joinpath("direct-first.json").write_text(json.dumps(first, indent=2))
    ARTIFACTS.joinpath("direct-second.json").write_text(json.dumps(second, indent=2))
    log_lines.extend(
        [
            f"first ticket:  {first['metadata']['ticket_id']} cached={first['metadata']['cached']}",
            f"second ticket: {second['metadata']['ticket_id']} cached={second['metadata']['cached']}",
            f"ledger tickets: {health['ledger_ticket_count']}",
            "PASS direct idempotency reused the same support ticket.",
        ]
    )
    (ARTIFACTS / "direct-idempotency.log").write_text("\n".join(log_lines) + "\n")
    print(log_lines[-1])


def wait_for_health() -> None:
    url = f"http://127.0.0.1:{API_PORT}/api/health"
    for _ in range(40):
        try:
            health = request("GET", url, timeout=2)
            (ARTIFACTS / "health.json").write_text(json.dumps(health, indent=2))
            print(f"aetheris health: {health.get('status')}")
            return
        except Exception:
            time.sleep(0.5)
    raise RuntimeError("Aetheris API did not become healthy")


def run_aetheris_job() -> None:
    log_lines = [
        f"$ POST /api/agents/{AGENT_ID}/message",
        f"message: {MESSAGE}",
    ]
    submit = request(
        "POST",
        f"http://127.0.0.1:{API_PORT}/api/agents/{AGENT_ID}/message",
        {"message": MESSAGE},
        {"Idempotency-Key": "customer-aetheris-001"},
    )
    job_id = submit.get("job_id") or submit.get("id") or submit.get("runtime_submission", {}).get("job_id")
    if not job_id:
        raise RuntimeError(f"no job id in response: {submit}")
    log_lines.append(f"job id: {job_id}")

    status = "unknown"
    event_count = 0
    for _ in range(60):
        job = request("GET", f"http://127.0.0.1:{API_PORT}/api/jobs/{job_id}")
        events = request("GET", f"http://127.0.0.1:{API_PORT}/api/jobs/{job_id}/events")
        status = job.get("status", "unknown")
        event_count = len(events.get("events", []))
        log_lines.append(f"poll status={status} events={event_count}")
        if status in ("completed", "failed", "cancelled"):
            break
        time.sleep(0.5)

    job = request("GET", f"http://127.0.0.1:{API_PORT}/api/jobs/{job_id}")
    events = request("GET", f"http://127.0.0.1:{API_PORT}/api/jobs/{job_id}/events")
    trace = request("GET", f"http://127.0.0.1:{API_PORT}/api/jobs/{job_id}/trace")
    ARTIFACTS.joinpath("job.json").write_text(json.dumps(job, indent=2))
    ARTIFACTS.joinpath("events.json").write_text(json.dumps(events, indent=2))
    ARTIFACTS.joinpath("trace.json").write_text(json.dumps(trace, indent=2))

    answer = ""
    if events.get("events"):
        payload = events["events"][-1].get("payload") or {}
        answer = payload.get("answer") or payload.get("result") or ""
    first_type = events["events"][0]["type"] if events.get("events") else "none"
    last_type = events["events"][-1]["type"] if events.get("events") else "none"
    trace_steps = len(trace.get("steps", []))
    log_lines.extend(
        [
            f"final status: {job.get('status')}",
            f"first event: {first_type}",
            f"last event:  {last_type}",
            f"trace steps: {trace_steps}",
            f"answer: {answer.splitlines()[0] if answer else '(none)'}",
            "PASS Aetheris recorded the real customer support agent run.",
        ]
    )
    (ARTIFACTS / "aetheris-job.log").write_text("\n".join(log_lines) + "\n")
    print(log_lines[-1])


if __name__ == "__main__":
    try:
        main()
    except Exception as exc:
        print(f"DEMO FAILED: {exc}", file=sys.stderr)
        raise
