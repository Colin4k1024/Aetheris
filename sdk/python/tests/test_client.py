"""Tests for AetherisClient — uses `responses` to mock HTTP calls."""

import pytest

try:
    import responses as responses_mock
    import requests  # noqa: F401

    HAS_RESPONSES = True
except ImportError:
    HAS_RESPONSES = False

pytestmark = pytest.mark.skipif(
    not HAS_RESPONSES,
    reason="requires `requests` and `responses` packages",
)

if HAS_RESPONSES:
    import responses as rsps_lib
    from aetheris import AetherisClient, JobStatus, JobFailedError
    from aetheris.client import TimeoutError as AetherisTimeoutError


BASE = "http://localhost:8080"


@rsps_lib.activate if HAS_RESPONSES else (lambda f: f)
def test_run_returns_job():
    rsps_lib.add(
        rsps_lib.POST,
        f"{BASE}/api/agents/my-agent/message",
        json={"job_id": "job-001", "status": "pending"},
        status=202,
    )
    client = AetherisClient(BASE)
    job = client.run("my-agent", "Do something")
    assert job.id == "job-001"
    assert job.agent_id == "my-agent"
    assert job.status == JobStatus.PENDING


@rsps_lib.activate if HAS_RESPONSES else (lambda f: f)
def test_get_job():
    rsps_lib.add(
        rsps_lib.GET,
        f"{BASE}/api/jobs/job-001",
        json={"id": "job-001", "agent_id": "my-agent", "status": "completed", "goal": "Do something"},
        status=200,
    )
    client = AetherisClient(BASE)
    job = client.get_job("job-001")
    assert job.status == JobStatus.COMPLETED
    assert job.is_terminal is True


@rsps_lib.activate if HAS_RESPONSES else (lambda f: f)
def test_wait_polls_until_completed():
    # First call returns running, second returns completed
    rsps_lib.add(
        rsps_lib.POST,
        f"{BASE}/api/agents/my-agent/message",
        json={"job_id": "job-002"},
        status=202,
    )
    rsps_lib.add(
        rsps_lib.GET,
        f"{BASE}/api/jobs/job-002",
        json={"id": "job-002", "agent_id": "my-agent", "status": "running", "goal": "x"},
    )
    rsps_lib.add(
        rsps_lib.GET,
        f"{BASE}/api/jobs/job-002",
        json={"id": "job-002", "agent_id": "my-agent", "status": "completed", "goal": "x", "output": "done"},
    )

    client = AetherisClient(BASE)
    job = client.run("my-agent", "x")
    result = job.wait(poll_interval=0.01)
    assert result.status == JobStatus.COMPLETED
    assert result.output == "done"


@rsps_lib.activate if HAS_RESPONSES else (lambda f: f)
def test_wait_raises_on_failure():
    rsps_lib.add(
        rsps_lib.POST,
        f"{BASE}/api/agents/my-agent/message",
        json={"job_id": "job-003"},
        status=202,
    )
    rsps_lib.add(
        rsps_lib.GET,
        f"{BASE}/api/jobs/job-003",
        json={"id": "job-003", "agent_id": "my-agent", "status": "failed", "goal": "x"},
    )
    client = AetherisClient(BASE)
    job = client.run("my-agent", "x")
    with pytest.raises(JobFailedError) as exc_info:
        job.wait(poll_interval=0.01)
    assert exc_info.value.status == "failed"


@rsps_lib.activate if HAS_RESPONSES else (lambda f: f)
def test_wait_raises_timeout():
    rsps_lib.add(
        rsps_lib.POST,
        f"{BASE}/api/agents/my-agent/message",
        json={"job_id": "job-004"},
        status=202,
    )
    # Always returns running — will time out
    rsps_lib.add(
        rsps_lib.GET,
        f"{BASE}/api/jobs/job-004",
        json={"id": "job-004", "agent_id": "my-agent", "status": "running", "goal": "x"},
        match_querystring=False,
    )

    client = AetherisClient(BASE)
    job = client.run("my-agent", "x")
    with pytest.raises(AetherisTimeoutError):
        job.wait(timeout=0.05, poll_interval=0.01)


@rsps_lib.activate if HAS_RESPONSES else (lambda f: f)
def test_signal_job():
    rsps_lib.add(
        rsps_lib.POST,
        f"{BASE}/api/jobs/job-005/signal",
        json={"ok": True},
        status=200,
    )
    client = AetherisClient(BASE)
    # Should not raise
    client.signal_job("job-005", {"approved": True}, correlation_key="approval-key")


@rsps_lib.activate if HAS_RESPONSES else (lambda f: f)
def test_health_returns_true():
    rsps_lib.add(rsps_lib.GET, f"{BASE}/api/health", json={"status": "ok"}, status=200)
    client = AetherisClient(BASE)
    assert client.health() is True


@rsps_lib.activate if HAS_RESPONSES else (lambda f: f)
def test_health_returns_false_on_error():
    rsps_lib.add(rsps_lib.GET, f"{BASE}/api/health", body=Exception("conn refused"))
    client = AetherisClient(BASE)
    assert client.health() is False


@rsps_lib.activate if HAS_RESPONSES else (lambda f: f)
def test_list_jobs():
    rsps_lib.add(
        rsps_lib.GET,
        f"{BASE}/api/agents/my-agent/jobs",
        json={
            "jobs": [
                {"id": "j1", "agent_id": "my-agent", "status": "completed", "goal": "a"},
                {"id": "j2", "agent_id": "my-agent", "status": "running", "goal": "b"},
            ],
            "total": 2,
        },
    )
    client = AetherisClient(BASE)
    jobs = client.list_jobs("my-agent")
    assert len(jobs) == 2
    assert jobs[0].id == "j1"
    assert jobs[1].status == JobStatus.RUNNING
