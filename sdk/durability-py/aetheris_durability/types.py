"""Core types for the Aetheris Durability SDK."""

from __future__ import annotations

import json
from dataclasses import dataclass, field
from datetime import datetime, timezone
from enum import Enum
from typing import Any, Callable, Dict, List, Optional


class EventType(str, Enum):
    """Categorizes an immutable event in the job's event stream."""

    JOB_CREATED = "job_created"
    JOB_STARTED = "job_started"
    JOB_COMPLETED = "job_completed"
    JOB_FAILED = "job_failed"
    JOB_CANCELLED = "job_cancelled"

    STEP_STARTED = "step_started"
    STEP_FINISHED = "step_finished"
    STEP_FAILED = "step_failed"
    STEP_RETRIED = "step_retried"

    CHECKPOINT_SAVED = "checkpoint_saved"
    CHECKPOINT_LOADED = "checkpoint_loaded"

    EFFECT_RECORDED = "effect_recorded"


class JobState(str, Enum):
    """Current state of a durable job."""

    CREATED = "created"
    RUNNING = "running"
    WAITING = "waiting"
    COMPLETED = "completed"
    FAILED = "failed"
    CANCELLED = "cancelled"


@dataclass
class Event:
    """An immutable record in the job's event stream."""

    id: str = ""
    job_id: str = ""
    type: EventType = EventType.JOB_CREATED
    step_id: str = ""
    payload: Optional[Dict[str, Any]] = None
    version: int = 0
    created_at: datetime = field(default_factory=lambda: datetime.now(timezone.utc))


@dataclass
class Checkpoint:
    """A point-in-time snapshot of job state."""

    job_id: str = ""
    step_id: str = ""
    state: Optional[Dict[str, Any]] = None
    version: int = 0
    created_at: datetime = field(default_factory=lambda: datetime.now(timezone.utc))


@dataclass
class Job:
    """A durable unit of work."""

    id: str = ""
    name: str = ""
    state: JobState = JobState.CREATED
    version: int = 0
    input: Optional[Dict[str, Any]] = None
    result: Optional[Dict[str, Any]] = None
    error: str = ""
    created_at: datetime = field(default_factory=lambda: datetime.now(timezone.utc))
    updated_at: datetime = field(default_factory=lambda: datetime.now(timezone.utc))
    completed_steps: Dict[str, Any] = field(default_factory=dict)


# StepFunc: receives state dict, returns updated state dict
StepFunc = Callable[[Dict[str, Any]], Dict[str, Any]]


@dataclass
class Step:
    """A single unit of work within a job.

    Args:
        id: Unique step identifier.
        fn: Function that receives state dict and returns updated state dict.
        name: Human-readable name (defaults to id).
        max_retries: Maximum retry attempts on failure (default 1 = no retry).
    """

    id: str
    fn: StepFunc
    name: str = ""
    max_retries: int = 1

    def __post_init__(self) -> None:
        if not self.name:
            self.name = self.id
