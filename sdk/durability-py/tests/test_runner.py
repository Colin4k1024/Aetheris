"""Tests for the Runner."""

import pytest

from aetheris_durability import Runner, MemoryStore, Step, JobState


def test_start_creates_job():
    store = MemoryStore()
    runner = Runner(store)

    job = runner.start("test-job", {"key": "value"})
    assert job.id != ""
    assert job.name == "test-job"
    assert job.state == JobState.CREATED


def test_execute_simple():
    store = MemoryStore()
    runner = Runner(store)

    job = runner.start("simple", {"count": 0})

    steps = [
        Step("add", lambda s: {**s, "count": s["count"] + 10}),
        Step("multiply", lambda s: {**s, "count": s["count"] * 2}),
    ]

    result = runner.execute(job.id, steps)
    assert result["count"] == 20

    updated_job = runner.get_job(job.id)
    assert updated_job.state == JobState.COMPLETED


def test_execute_crash_recovery():
    store = MemoryStore()
    runner = Runner(store)

    job = runner.start("crash-test", {"records": 0})

    step1_calls = 0
    step2_calls = 0
    crashed = False

    def step1(state):
        nonlocal step1_calls
        step1_calls += 1
        return {**state, "records": 100}

    def step2(state):
        nonlocal step2_calls, crashed
        step2_calls += 1
        if not crashed:
            crashed = True
            raise RuntimeError("process killed!")
        return {**state, "records": 200}

    def step3(state):
        return {**state, "records": 300}

    steps = [
        Step("batch-1", step1),
        Step("batch-2", step2),
        Step("batch-3", step3),
    ]

    # First attempt — crashes at step 2
    with pytest.raises(RuntimeError, match="process killed"):
        runner.execute(job.id, steps)

    assert step1_calls == 1

    # Resume — step 1 should NOT be re-executed
    result = runner.execute(job.id, steps)
    assert step1_calls == 1  # Not re-executed (checkpoint)
    assert step2_calls == 2  # Crashed + resumed
    assert result["records"] == 300


def test_execute_step_retry():
    store = MemoryStore()
    runner = Runner(store)

    job = runner.start("retry-test")
    attempts = 0

    def flaky_fn(state):
        nonlocal attempts
        attempts += 1
        if attempts < 3:
            raise RuntimeError("temporary failure")
        return {**state, "ok": True}

    steps = [Step("flaky", flaky_fn, max_retries=3)]

    result = runner.execute(job.id, steps)
    assert result["ok"] is True
    assert attempts == 3


def test_execute_all_retries_exhausted():
    store = MemoryStore()
    runner = Runner(store)

    job = runner.start("fail-test")

    steps = [
        Step(
            "always-fail",
            lambda s: (_ for _ in ()).throw(RuntimeError("permanent")),
            max_retries=2,
        ),
    ]

    with pytest.raises(RuntimeError, match="permanent"):
        runner.execute(job.id, steps)

    job = runner.get_job(job.id)
    assert job.state == JobState.FAILED


def test_execute_already_completed():
    store = MemoryStore()
    runner = Runner(store)

    job = runner.start("done-test", {"x": 1})
    steps = [Step("noop", lambda s: s)]

    result1 = runner.execute(job.id, steps)
    result2 = runner.execute(job.id, steps)
    assert result1["x"] == result2["x"]


def test_get_events():
    store = MemoryStore()
    runner = Runner(store)

    job = runner.start("event-test")
    steps = [Step("s1", lambda s: {**s, "done": True})]

    runner.execute(job.id, steps)
    events = runner.get_events(job.id)

    assert len(events) >= 3
    assert events[0].type.value == "job_created"

    types = [e.type.value for e in events]
    assert "job_completed" in types


def test_version_mismatch():
    store = MemoryStore()

    from aetheris_durability.types import Event, EventType

    # Append first event at version 0
    store.append_event("test-job", 0, Event(type=EventType.JOB_CREATED))

    # Try to append at wrong version
    with pytest.raises(Exception):
        store.append_event("test-job", 0, Event(type=EventType.JOB_STARTED))
