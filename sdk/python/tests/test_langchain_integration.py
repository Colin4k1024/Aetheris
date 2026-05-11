import json
import threading
import urllib.error
import urllib.request
from http.server import HTTPServer

import pytest

from aetheris.integrations.langchain import AetherisLangChainAdapter, _AdapterHandler


class RecordingRunnable:
    def __init__(self, result):
        self.result = result
        self.calls = []

    def invoke(self, payload):
        self.calls.append(payload)
        return self.result


class FailingRunnable:
    def invoke(self, payload):
        raise RuntimeError("boom")


def test_adapter_invoke_returns_expected_shape():
    runnable = RecordingRunnable({"output": "done"})
    adapter = AetherisLangChainAdapter(runnable)

    result = adapter.invoke(
        {
            "message": "hello",
            "metadata": {"job_id": "job-123"},
        }
    )

    assert runnable.calls == [{"input": "hello"}]
    assert result == {
        "answer": "done",
        "final": True,
        "metadata": {"job_id": "job-123"},
    }


def test_adapter_invoke_stringifies_non_dict_results():
    runnable = RecordingRunnable(["a", "b"])
    adapter = AetherisLangChainAdapter(runnable)

    result = adapter.invoke({"message": "hello"})

    assert result["answer"] == "['a', 'b']"
    assert result["final"] is True


def _start_test_server(adapter):
    handler_cls = type("_TestLangChainHandler", (_AdapterHandler,), {"adapter": adapter})
    server = HTTPServer(("127.0.0.1", 0), handler_cls)
    thread = threading.Thread(target=server.serve_forever, daemon=True)
    thread.start()
    return server, thread


def test_handler_returns_success_json():
    adapter = AetherisLangChainAdapter(RecordingRunnable({"output": "ok"}))
    server, thread = _start_test_server(adapter)
    url = f"http://127.0.0.1:{server.server_port}"

    try:
        request = urllib.request.Request(
            url,
            data=json.dumps({"message": "hello"}).encode(),
            headers={"Content-Type": "application/json"},
            method="POST",
        )
        with urllib.request.urlopen(request, timeout=5) as response:
            payload = json.loads(response.read())

        assert response.status == 200
        assert payload["answer"] == "ok"
        assert payload["final"] is True
    finally:
        server.shutdown()
        server.server_close()
        thread.join(timeout=5)


def test_handler_returns_non_2xx_on_adapter_failure():
    adapter = AetherisLangChainAdapter(FailingRunnable())
    server, thread = _start_test_server(adapter)
    url = f"http://127.0.0.1:{server.server_port}"

    try:
        request = urllib.request.Request(
            url,
            data=json.dumps({"message": "hello"}).encode(),
            headers={"Content-Type": "application/json"},
            method="POST",
        )
        with pytest.raises(urllib.error.HTTPError) as exc_info:
            urllib.request.urlopen(request, timeout=5)

        assert exc_info.value.code == 502
        payload = json.loads(exc_info.value.read())
        assert payload["error"] == "boom"
        assert payload["final"] is False
    finally:
        server.shutdown()
        server.server_close()
        thread.join(timeout=5)
