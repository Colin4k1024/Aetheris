"""Tests for the IdempotentTool."""

import pytest

from aetheris_durability import IdempotentTool, MemoryStore


def test_wrap_executes_once():
    store = MemoryStore()
    idem = IdempotentTool(store)

    call_count = 0

    def my_tool(input_data):
        nonlocal call_count
        call_count += 1
        return {"result": "ok"}

    wrapped = idem.wrap_func("test-tool", my_tool)

    # First call
    result1 = wrapped({"data": "test"}, idempotency_key="key-1")
    assert result1["result"] == "ok"

    # Second call with same key — should NOT re-execute
    result2 = wrapped({"data": "test"}, idempotency_key="key-1")
    assert result2["result"] == "ok"

    assert call_count == 1


def test_wrap_different_keys():
    store = MemoryStore()
    idem = IdempotentTool(store)

    call_count = 0

    def my_tool(input_data):
        nonlocal call_count
        call_count += 1
        return {"result": "ok"}

    wrapped = idem.wrap_func("test-tool", my_tool)

    wrapped(None, idempotency_key="key-1")
    wrapped(None, idempotency_key="key-2")
    wrapped(None, idempotency_key="key-3")

    assert call_count == 3


def test_wrap_error_not_cached():
    store = MemoryStore()
    idem = IdempotentTool(store)

    call_count = 0

    def failing_tool(input_data):
        nonlocal call_count
        call_count += 1
        if call_count == 1:
            raise RuntimeError("transient error")
        return {"result": "ok"}

    wrapped = idem.wrap_func("failing-tool", failing_tool)

    # First call fails
    with pytest.raises(RuntimeError, match="transient"):
        wrapped(None, idempotency_key="key-1")

    # Second call with same key — should re-execute
    result = wrapped(None, idempotency_key="key-1")
    assert result["result"] == "ok"
    assert call_count == 2


def test_decorator_form():
    store = MemoryStore()
    idem = IdempotentTool(store)

    call_count = 0

    @idem.wrap("my-tool")
    def my_tool(input_data):
        nonlocal call_count
        call_count += 1
        return {"sent": True}

    result1 = my_tool({"to": "test"}, idempotency_key="order-1")
    result2 = my_tool({"to": "test"}, idempotency_key="order-1")

    assert result1["sent"] is True
    assert result2["sent"] is True
    assert call_count == 1
