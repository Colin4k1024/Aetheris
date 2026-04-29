"""
Aetheris Python SDK — client implementation

Wraps the Aetheris REST API with a simple, blocking interface.
No third-party dependencies beyond the standard library and
the optional `requests` (or `httpx`) package.
"""

from __future__ import annotations

import time
import uuid
from dataclasses import dataclass, field
from enum import Enum
from typing import Any, Dict, Optional

try:
    import requests as _requests

    _HAS_REQUESTS = True
except ImportError:  # pragma: no cover
    _HAS_REQUESTS = False

try:
    import httpx as _httpx

    _HAS_HTTPX = True
except ImportError:
    _HAS_HTTPX = False


# ─── Exceptions ───────────────────────────────────────────────────────────────


class AetherisError(Exception):
    """Base exception for all Aetheris SDK errors."""


class JobFailedError(AetherisError):
    """Raised when a job terminates in a failed or cancelled state."""

    def __init__(self, job_id: str, status: str, message: str = "") -> None:
        self.job_id = job_id
        self.status = status
        super().__init__(f"Job {job_id} ended with status={status}: {message}")


class TimeoutError(AetherisError):  # noqa: A001
    """Raised when `Job.wait()` exceeds `timeout` seconds."""


# ─── Data models ──────────────────────────────────────────────────────────────


class JobStatus(str, Enum):
    PENDING = "pending"
    RUNNING = "running"
    COMPLETED = "completed"
    FAILED = "failed"
    CANCELLED = "cancelled"
    WAITING = "waiting"  # parked, awaiting human signal

    @classmethod
    def _missing_(cls, value: object) -> "JobStatus":  # pragma: no cover
        return cls.PENDING


_TERMINAL_STATUSES = {JobStatus.COMPLETED, JobStatus.FAILED, JobStatus.CANCELLED}


@dataclass
class Job:
    """Represents a submitted Aetheris job."""

    id: str
    agent_id: str
    status: JobStatus
    goal: str = ""
    output: Optional[Any] = None
    raw: Dict[str, Any] = field(default_factory=dict)

    # Back-reference to the client so callers can do `job.wait()` without
    # keeping the client around explicitly.
    _client: Optional["AetherisClient"] = field(default=None, repr=False, compare=False)

    @property
    def is_terminal(self) -> bool:
        return self.status in _TERMINAL_STATUSES

    @property
    def is_waiting(self) -> bool:
        return self.status == JobStatus.WAITING

    def wait(
        self,
        *,
        timeout: float = 300.0,
        poll_interval: float = 2.0,
    ) -> "Job":
        """Block until the job reaches a terminal state.

        Args:
            timeout: Maximum seconds to wait (default 5 minutes).
            poll_interval: Seconds between status checks.

        Returns:
            The updated Job object.

        Raises:
            TimeoutError: If the job hasn't finished within `timeout` seconds.
            JobFailedError: If the job ends in a failed/cancelled state.
        """
        if self._client is None:
            raise AetherisError("This job has no client attached; cannot poll.")

        deadline = time.monotonic() + timeout
        while True:
            updated = self._client.get_job(self.id)
            self.status = updated.status
            self.output = updated.output
            self.raw = updated.raw

            if self.is_terminal:
                if self.status != JobStatus.COMPLETED:
                    raise JobFailedError(self.id, self.status.value)
                return self

            if time.monotonic() >= deadline:
                raise TimeoutError(
                    f"Job {self.id} did not complete within {timeout}s "
                    f"(last status: {self.status.value})"
                )

            time.sleep(poll_interval)

    def signal(self, payload: Dict[str, Any], *, correlation_key: str = "") -> None:
        """Resume a WAITING (human-in-the-loop) job.

        Args:
            payload: Arbitrary JSON payload delivered to the waiting step.
            correlation_key: Must match the key the agent is waiting on.
        """
        if self._client is None:
            raise AetherisError("This job has no client attached.")
        self._client.signal_job(self.id, payload, correlation_key=correlation_key)


# ─── HTTP transport ───────────────────────────────────────────────────────────


class _Transport:
    """Thin HTTP abstraction that works with `requests` or `httpx`."""

    def __init__(
        self,
        base_url: str,
        *,
        token: Optional[str] = None,
        timeout: float = 30.0,
    ) -> None:
        self._base = base_url.rstrip("/")
        self._timeout = timeout
        self._headers: Dict[str, str] = {"Content-Type": "application/json"}
        if token:
            self._headers["Authorization"] = f"Bearer {token}"

        if _HAS_REQUESTS:
            self._impl = "requests"
        elif _HAS_HTTPX:
            self._impl = "httpx"
        else:
            raise ImportError(
                "Aetheris SDK requires either `requests` or `httpx`. "
                "Install with: pip install requests"
            )

    def _url(self, path: str) -> str:
        return f"{self._base}{path}"

    def get(self, path: str) -> Dict[str, Any]:
        url = self._url(path)
        if self._impl == "requests":
            resp = _requests.get(url, headers=self._headers, timeout=self._timeout)
        else:
            resp = _httpx.get(url, headers=self._headers, timeout=self._timeout)
        resp.raise_for_status()
        return resp.json()

    def post(self, path: str, body: Dict[str, Any]) -> Dict[str, Any]:
        url = self._url(path)
        if self._impl == "requests":
            resp = _requests.post(
                url, json=body, headers=self._headers, timeout=self._timeout
            )
        else:
            resp = _httpx.post(
                url, json=body, headers=self._headers, timeout=self._timeout
            )
        resp.raise_for_status()
        # 202 Accepted may have a body or be empty
        try:
            return resp.json()
        except Exception:
            return {}


# ─── Main client ──────────────────────────────────────────────────────────────


class AetherisClient:
    """Client for the Aetheris durable agent runtime.

    Args:
        base_url: URL of the Aetheris API server (e.g. "http://localhost:8080").
        token: Optional JWT bearer token (required when auth is enabled).
        timeout: HTTP request timeout in seconds.

    Example::

        client = AetherisClient("http://localhost:8080")
        job = client.run("my-agent", "Summarise the Q3 report")
        result = job.wait(timeout=120)
        print(result.output)

    """

    def __init__(
        self,
        base_url: str = "http://localhost:8080",
        *,
        token: Optional[str] = None,
        timeout: float = 30.0,
    ) -> None:
        self._http = _Transport(base_url, token=token, timeout=timeout)

    # ── Core operations ──────────────────────────────────────────────────────

    def run(
        self,
        agent_id: str,
        message: str,
        *,
        idempotency_key: Optional[str] = None,
    ) -> Job:
        """Submit a message to an agent, creating a durable job.

        Args:
            agent_id: Agent identifier (must exist in agents.yaml).
            message: The goal or prompt to deliver.
            idempotency_key: Optional deduplication key; re-submitting the
                same key returns the existing job without creating a duplicate.

        Returns:
            A :class:`Job` instance (status may be ``pending`` initially).
        """
        key = idempotency_key or str(uuid.uuid4())
        body = {"message": message}
        # Send idempotency key as header by injecting into transport headers
        # temporarily — simpler than modifying the transport interface.
        original_headers = dict(self._http._headers)
        self._http._headers["Idempotency-Key"] = key
        try:
            data = self._http.post(f"/api/agents/{agent_id}/message", body)
        finally:
            self._http._headers = original_headers

        job_id: str = (
            data.get("job_id")
            or data.get("id")
            or data.get("runtime_submission", {}).get("job_id", "")
        )
        if not job_id:
            raise AetherisError(
                f"Server did not return a job_id. Response: {data}"
            )

        return Job(
            id=job_id,
            agent_id=agent_id,
            status=JobStatus.PENDING,
            goal=message,
            raw=data,
            _client=self,
        )

    def get_job(self, job_id: str) -> Job:
        """Fetch the current state of a job by ID."""
        data = self._http.get(f"/api/jobs/{job_id}")
        return Job(
            id=data.get("id", job_id),
            agent_id=data.get("agent_id", ""),
            status=JobStatus(data.get("status", "pending")),
            goal=data.get("goal", ""),
            output=data.get("output"),
            raw=data,
            _client=self,
        )

    def signal_job(
        self,
        job_id: str,
        payload: Dict[str, Any],
        *,
        correlation_key: str = "",
    ) -> None:
        """Send a signal to a WAITING (human-in-the-loop) job.

        Args:
            job_id: ID of the waiting job.
            payload: Arbitrary data to deliver to the parked step.
            correlation_key: Must match the key the agent is waiting on.
        """
        body: Dict[str, Any] = {"payload": payload}
        if correlation_key:
            body["correlation_key"] = correlation_key
        self._http.post(f"/api/jobs/{job_id}/signal", body)

    def list_jobs(
        self,
        agent_id: str,
        *,
        status: Optional[str] = None,
        limit: int = 20,
    ) -> list[Job]:
        """List jobs for a given agent.

        Args:
            agent_id: Agent identifier.
            status: Optional filter (``"running"``, ``"completed"``, etc.).
            limit: Maximum number of jobs to return (max 100).
        """
        params = f"?limit={limit}"
        if status:
            params += f"&status={status}"
        data = self._http.get(f"/api/agents/{agent_id}/jobs{params}")
        jobs_raw = data.get("jobs", [])
        return [
            Job(
                id=j.get("id", ""),
                agent_id=j.get("agent_id", agent_id),
                status=JobStatus(j.get("status", "pending")),
                goal=j.get("goal", ""),
                raw=j,
                _client=self,
            )
            for j in jobs_raw
        ]

    def health(self) -> bool:
        """Return True if the server is healthy."""
        try:
            self._http.get("/api/health")
            return True
        except Exception:
            return False
