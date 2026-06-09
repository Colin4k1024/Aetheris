import json
import threading
import urllib.request
from http.server import HTTPServer

from aetheris.integrations.langgraph import AetherisLangGraphAdapter, _AdapterHandler


class RecordingGraph:
    def __init__(self, result):
        self.result = result
        self.calls = []

    def invoke(self, payload):
        self.calls.append(payload)
        return self.result


class Message:
    def __init__(self, content):
        self.content = content


def test_langgraph_adapter_uses_messages_state_and_extracts_last_message():
    graph = RecordingGraph({"messages": [Message("draft"), Message("final answer")]})
    adapter = AetherisLangGraphAdapter(graph)

    result = adapter.invoke(
        {
            "message": "research this",
            "metadata": {"job_id": "job-123"},
        }
    )

    assert graph.calls == [{"messages": [{"role": "user", "content": "research this"}]}]
    assert result == {
        "answer": "final answer",
        "final": True,
        "metadata": {"job_id": "job-123", "framework": "langgraph"},
    }


def test_langgraph_adapter_supports_custom_message_factory():
    graph = RecordingGraph({"answer": "ok"})
    adapter = AetherisLangGraphAdapter(
        graph,
        input_key="query",
        output_key="answer",
        message_factory=lambda message, envelope: {
            "text": message,
            "session_id": envelope.get("session_id"),
        },
    )

    result = adapter.invoke({"message": "hello", "session_id": "sess-1"})

    assert graph.calls == [{"query": {"text": "hello", "session_id": "sess-1"}}]
    assert result["answer"] == "ok"


def _start_test_server(adapter):
    handler_cls = type("_TestLangGraphHandler", (_AdapterHandler,), {"adapter": adapter})
    server = HTTPServer(("127.0.0.1", 0), handler_cls)
    thread = threading.Thread(target=server.serve_forever, daemon=True)
    thread.start()
    return server, thread


def test_langgraph_handler_returns_success_json():
    adapter = AetherisLangGraphAdapter(RecordingGraph({"messages": [{"content": "ok"}]}))
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
        assert payload["metadata"]["framework"] == "langgraph"
    finally:
        server.shutdown()
        server.server_close()
        thread.join(timeout=5)
