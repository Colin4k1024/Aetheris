"""Store interface and in-memory implementation for the durability SDK."""

from __future__ import annotations

import json
import threading
from abc import ABC, abstractmethod
from datetime import datetime, timezone
from typing import Dict, List, Optional, Tuple

from .types import Checkpoint, Event, EventType, Job, JobState


class VersionMismatchError(Exception):
    """Raised when append expected version doesn't match current version."""


class JobNotFoundError(Exception):
    """Raised when a job doesn't exist."""


class Store(ABC):
    """Persistence interface for the durable execution engine.

    Implementations store the event stream and job metadata.
    The in-memory implementation is provided; postgres implementation
    is in aetheris_durability.postgres.
    """

    @abstractmethod
    def append_event(
        self, job_id: str, expected_version: int, event: Event
    ) -> int:
        """Add an event to the job's event stream.

        Returns the new version after successful append.
        Raises VersionMismatchError if expected_version != current version.
        """
        ...

    @abstractmethod
    def list_events(self, job_id: str) -> Tuple[List[Event], int]:
        """Return all events for a job, ordered by version.

        Returns (events, current_version).
        """
        ...

    @abstractmethod
    def save_job(self, job: Job) -> None:
        """Persist job metadata."""
        ...

    @abstractmethod
    def load_job(self, job_id: str) -> Optional[Job]:
        """Retrieve job metadata. Returns None if not found."""
        ...

    @abstractmethod
    def save_checkpoint(
        self, job_id: str, step_id: str, state: Dict[str, Any], version: int
    ) -> None:
        """Persist a checkpoint (serialized state snapshot)."""
        ...

    @abstractmethod
    def load_checkpoint(self, job_id: str) -> Optional[Checkpoint]:
        """Load the latest checkpoint for a job. Returns None if no checkpoint."""
        ...


class MemoryStore(Store):
    """Thread-safe in-memory implementation of Store.

    Perfect for development, testing, and single-process deployments.
    For production with multiple workers, use the postgres implementation.
    """

    def __init__(self) -> None:
        self._lock = threading.RLock()
        self._jobs: Dict[str, Job] = {}
        self._events: Dict[str, List[Event]] = {}
        self._checkpoints: Dict[str, Checkpoint] = {}

    def append_event(
        self, job_id: str, expected_version: int, event: Event
    ) -> int:
        with self._lock:
            current = self._events.get(job_id, [])
            if len(current) != expected_version:
                raise VersionMismatchError(
                    f"Expected version {expected_version}, got {len(current)}"
                )

            event.job_id = job_id
            event.version = len(current) + 1
            if not event.id:
                event.id = f"{job_id}-{event.version}"
            if not event.created_at:
                event.created_at = datetime.now(timezone.utc)

            self._events.setdefault(job_id, []).append(event)
            return event.version

    def list_events(self, job_id: str) -> Tuple[List[Event], int]:
        with self._lock:
            events = list(self._events.get(job_id, []))
            return events, len(events)

    def save_job(self, job: Job) -> None:
        with self._lock:
            self._jobs[job.id] = job

    def load_job(self, job_id: str) -> Optional[Job]:
        with self._lock:
            job = self._jobs.get(job_id)
            if job is None:
                return None
            # Return a copy
            return Job(
                id=job.id,
                name=job.name,
                state=job.state,
                version=job.version,
                input=dict(job.input) if job.input else None,
                result=dict(job.result) if job.result else None,
                error=job.error,
                created_at=job.created_at,
                updated_at=job.updated_at,
                completed_steps=dict(job.completed_steps),
            )

    def save_checkpoint(
        self, job_id: str, step_id: str, state: Dict[str, Any], version: int
    ) -> None:
        with self._lock:
            self._checkpoints[job_id] = Checkpoint(
                job_id=job_id,
                step_id=step_id,
                state=dict(state) if state else None,
                version=version,
                created_at=datetime.now(timezone.utc),
            )

    def load_checkpoint(self, job_id: str) -> Optional[Checkpoint]:
        with self._lock:
            cp = self._checkpoints.get(job_id)
            if cp is None:
                return None
            return Checkpoint(
                job_id=cp.job_id,
                step_id=cp.step_id,
                state=dict(cp.state) if cp.state else None,
                version=cp.version,
                created_at=cp.created_at,
            )
