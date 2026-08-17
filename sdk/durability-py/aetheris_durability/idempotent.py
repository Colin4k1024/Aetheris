"""At-most-once execution for side-effecting operations."""

from __future__ import annotations

from typing import Any, Callable, Dict

from .store import Store
from .types import Event, EventType, Job, JobState


class IdempotentTool:
    """Provides at-most-once execution for side-effecting operations.

    Wrap any function with wrap() to ensure it executes at most once per
    idempotency key, even across retries and crash recovery.

    Usage:

        store = MemoryStore()
        idem = IdempotentTool(store)

        @idem.wrap("send-email")
        def send_email(input_data):
            return email_service.send(input_data["to"], input_data["subject"])

        # Same idempotency key = same result, no re-execution
        result1 = send_email(idempotency_key="order-123-email", input_data={...})
        result2 = send_email(idempotency_key="order-123-email", input_data={...})
    """

    def __init__(self, store: Store) -> None:
        self._store = store

    def wrap(
        self,
        tool_name: str,
    ) -> Callable:
        """Decorator to wrap a function with at-most-once semantics.

        The decorated function gains an `idempotency_key` keyword argument.
        Calling with the same key returns the cached result.

        Args:
            tool_name: Identifies this tool in the event stream.
        """

        def decorator(
            fn: Callable[[Dict[str, Any]], Dict[str, Any]],
        ) -> Callable:
            def wrapper(
                input_data: Dict[str, Any],
                *,
                idempotency_key: str,
            ) -> Dict[str, Any]:
                return self._execute(tool_name, fn, idempotency_key, input_data)

            return wrapper

        return decorator

    def wrap_func(
        self,
        tool_name: str,
        fn: Callable[[Dict[str, Any]], Dict[str, Any]],
    ) -> Callable:
        """Wrap a function directly (non-decorator form).

        Args:
            tool_name: Identifies this tool in the event stream.
            fn: The function to wrap.

        Returns:
            A wrapped function that takes (input_data, idempotency_key=...).
        """

        def wrapper(
            input_data: Dict[str, Any],
            *,
            idempotency_key: str,
        ) -> Dict[str, Any]:
            return self._execute(tool_name, fn, idempotency_key, input_data)

        return wrapper

    def _execute(
        self,
        tool_name: str,
        fn: Callable[[Dict[str, Any]], Dict[str, Any]],
        idempotency_key: str,
        input_data: Dict[str, Any],
    ) -> Dict[str, Any]:
        ledger_job_id = f"ledger:{tool_name}:{idempotency_key}"

        # Check if already committed
        job = self._store.load_job(ledger_job_id)

        if job is not None and job.state == JobState.COMPLETED:
            return job.result or {}

        # If acquired but not committed (previous crash), allow re-execution
        if job is not None and job.state == JobState.RUNNING:
            job.state = JobState.CREATED
            job.error = ""
            self._store.save_job(job)

        # Create or update as "acquired"
        if job is None:
            job = Job(
                id=ledger_job_id,
                name=f"idempotent:{tool_name}",
                state=JobState.RUNNING,
                input=input_data,
                completed_steps={},
            )
        else:
            job.state = JobState.RUNNING
            job.input = input_data

        self._store.save_job(job)

        # Record acquired event
        try:
            new_ver = self._store.append_event(
                ledger_job_id,
                job.version,
                Event(
                    type=EventType.EFFECT_RECORDED,
                    payload={
                        "tool_name": tool_name,
                        "idempotency_key": idempotency_key,
                        "phase": "acquired",
                    },
                ),
            )
            job.version = new_ver
        except Exception:
            pass  # Best effort

        # Execute the actual function
        try:
            result = fn(input_data)
        except Exception as e:
            # Mark as failed so next retry will re-execute
            job.state = JobState.FAILED
            job.error = str(e)
            self._store.save_job(job)
            raise

        # Commit the result
        job.state = JobState.COMPLETED
        job.result = result
        self._store.save_job(job)

        try:
            self._store.append_event(
                ledger_job_id,
                job.version,
                Event(
                    type=EventType.EFFECT_RECORDED,
                    payload={
                        "tool_name": tool_name,
                        "idempotency_key": idempotency_key,
                        "phase": "committed",
                    },
                ),
            )
        except Exception:
            pass  # Best effort

        return result
