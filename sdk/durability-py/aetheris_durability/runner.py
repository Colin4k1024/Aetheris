"""Durable runner with automatic checkpointing and crash recovery."""

from __future__ import annotations

import time
import uuid
from typing import Any, Dict, List, Optional

from .store import Store, JobNotFoundError
from .types import (
    Checkpoint,
    Event,
    EventType,
    Job,
    JobState,
    Step,
)


class Runner:
    """Executes durable jobs with automatic checkpointing and crash recovery.

    Usage:

        store = MemoryStore()
        runner = Runner(store)

        job = runner.start("process-order", {"order_id": "123"})
        steps = [
            Step("validate", validate_order),
            Step("charge", charge_payment, max_retries=3),
            Step("ship", create_shipment),
        ]
        result = runner.execute(job.id, steps)

    If the process crashes during "charge", calling execute() again resumes
    from "charge" — "validate" is not re-executed because its result is checkpointed.
    """

    def __init__(self, store: Store) -> None:
        self._store = store

    @property
    def store(self) -> Store:
        """Access the underlying store (for advanced usage)."""
        return self._store

    def start(
        self, name: str, input_data: Optional[Dict[str, Any]] = None
    ) -> Job:
        """Create a new durable job.

        Args:
            name: Human-readable job name.
            input_data: Initial state (will be passed to the first step).

        Returns:
            The created Job.
        """
        job_id = f"job-{name}-{int(time.time() * 1000000)}"

        job = Job(
            id=job_id,
            name=name,
            state=JobState.CREATED,
            version=0,
            input=input_data or {},
            completed_steps={},
        )

        self._store.save_job(job)

        # Record job_created event
        new_ver = self._store.append_event(
            job_id,
            0,
            Event(
                type=EventType.JOB_CREATED,
                payload={"name": name},
            ),
        )
        job.version = new_ver
        self._store.save_job(job)

        return job

    def execute(
        self, job_id: str, steps: List[Step]
    ) -> Dict[str, Any]:
        """Execute steps sequentially with automatic checkpointing.

        On first call: executes all steps from the beginning.
        On resume (after crash): loads checkpoint, skips completed steps.

        Args:
            job_id: The job to execute.
            steps: Ordered list of steps to execute.

        Returns:
            The final state after all steps complete.

        Raises:
            JobNotFoundError: If the job doesn't exist.
            RuntimeError: If a step fails after all retries.
        """
        # Load job metadata
        job = self._store.load_job(job_id)
        if job is None:
            raise JobNotFoundError(f"Job {job_id} not found")

        if job.state == JobState.COMPLETED:
            return job.result or {}

        if job.state == JobState.CANCELLED:
            raise RuntimeError(f"Job {job_id} is cancelled")

        # Sync version with store
        _, store_version = self._store.list_events(job_id)
        job.version = store_version

        # Try to load checkpoint
        state: Dict[str, Any] = {}
        start_idx = 0

        checkpoint = self._store.load_checkpoint(job_id)
        if checkpoint is not None:
            state = dict(checkpoint.state) if checkpoint.state else {}
            # Find step index to resume from
            for i, step in enumerate(steps):
                if step.id == checkpoint.step_id:
                    start_idx = i + 1  # Resume from step AFTER checkpoint
                    break

        if not state:
            state = dict(job.input) if job.input else {}

        # Mark job as running
        job.state = JobState.RUNNING
        job.updated_at = datetime.now(timezone.utc)
        self._store.save_job(job)

        # Execute steps from start_idx
        for i in range(start_idx, len(steps)):
            step = steps[i]

            # Record step_started
            new_ver = self._store.append_event(
                job_id,
                job.version,
                Event(
                    type=EventType.STEP_STARTED,
                    step_id=step.id,
                    payload={"step_name": step.name},
                ),
            )
            job.version = new_ver

            # Execute with retries
            step_result: Dict[str, Any] = {}
            max_retries = max(1, step.max_retries)
            last_error: Optional[Exception] = None

            for attempt in range(max_retries):
                try:
                    step_result = step.fn(dict(state))
                    last_error = None
                    break
                except Exception as e:
                    last_error = e
                    if attempt < max_retries - 1:
                        # Record retry
                        new_ver = self._store.append_event(
                            job_id,
                            job.version,
                            Event(
                                type=EventType.STEP_RETRIED,
                                step_id=step.id,
                                payload={
                                    "attempt": attempt + 1,
                                    "error": str(e),
                                    "max_retries": max_retries,
                                },
                            ),
                        )
                        job.version = new_ver

            if last_error is not None:
                # Step failed after all retries
                new_ver = self._store.append_event(
                    job_id,
                    job.version,
                    Event(
                        type=EventType.STEP_FAILED,
                        step_id=step.id,
                        payload={"error": str(last_error)},
                    ),
                )
                job.version = new_ver
                job.state = JobState.FAILED
                job.error = str(last_error)
                job.updated_at = datetime.now(timezone.utc)
                self._store.save_job(job)
                raise RuntimeError(
                    f"Step {step.id} failed: {last_error}"
                )

            # Merge step result into state
            state.update(step_result)

            # Record step_finished
            new_ver = self._store.append_event(
                job_id,
                job.version,
                Event(
                    type=EventType.STEP_FINISHED,
                    step_id=step.id,
                    payload={"output_keys": list(step_result.keys())},
                ),
            )
            job.version = new_ver

            # Checkpoint
            self._store.save_checkpoint(job_id, step.id, state, job.version)

            new_ver = self._store.append_event(
                job_id,
                job.version,
                Event(
                    type=EventType.CHECKPOINT_SAVED,
                    step_id=step.id,
                ),
            )
            job.version = new_ver

            # Update completed steps
            job.completed_steps[step.id] = state
            job.updated_at = datetime.now(timezone.utc)
            self._store.save_job(job)

        # All steps completed
        job.state = JobState.COMPLETED
        job.result = state
        job.updated_at = datetime.now(timezone.utc)
        self._store.save_job(job)

        self._store.append_event(
            job_id,
            job.version,
            Event(
                type=EventType.JOB_COMPLETED,
                payload={"output_keys": list(state.keys())},
            ),
        )

        return state

    def resume(
        self, job_id: str, steps: List[Step]
    ) -> Dict[str, Any]:
        """Convenience alias for execute() — resumes from checkpoint."""
        return self.execute(job_id, steps)

    def get_job(self, job_id: str) -> Optional[Job]:
        """Get the current state of a job."""
        return self._store.load_job(job_id)

    def get_events(self, job_id: str) -> List[Event]:
        """Get all events for a job (for audit/replay)."""
        events, _ = self._store.list_events(job_id)
        return events


# Need datetime for timezone-aware timestamps
from datetime import datetime, timezone  # noqa: E402
