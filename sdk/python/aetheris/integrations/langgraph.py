"""
aetheris.integrations.langgraph
~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

Expose a compiled LangGraph graph as an Aetheris-compatible HTTP agent.
"""

from __future__ import annotations

import http.server
import json
import threading
from typing import Any, Callable, Dict, Optional

from .embedded import (
    AetherisChatModel,
    AetherisRuntimeContext,
    AetherisRuntimeTool,
    EmbeddedAgentManifest,
    serve_embedded,
)

__all__ = [
    "AetherisLangGraphAdapter",
    "AetherisRuntimeContext",
    "AetherisRuntimeTool",
    "AetherisChatModel",
    "EmbeddedAgentManifest",
    "serve",
    "serve_embedded",
]


def _default_message_factory(message: str, envelope: Dict[str, Any]) -> Any:
    _ = envelope
    return [{"role": "user", "content": message}]


def _content_from_message(value: Any) -> str:
    if value is None:
        return ""
    if isinstance(value, (list, tuple)):
        if not value:
            return ""
        return _content_from_message(value[-1])
    if isinstance(value, dict):
        for key in ("content", "answer", "output", "text"):
            if key in value:
                return _content_from_message(value[key])
        if "messages" in value:
            return _content_from_message(value["messages"])
        return str(value)
    content = getattr(value, "content", None)
    if content is not None:
        return _content_from_message(content)
    return str(value)


class AetherisLangGraphAdapter:
    """Wraps a LangGraph compiled graph behind the Aetheris invoke contract.

    The default payload sent to ``graph.invoke`` is::

        {"messages": [{"role": "user", "content": "<message>"}]}

    This matches the common LangGraph message-state and prebuilt ReAct agent
    shape. Use ``message_factory`` if your graph expects a different state.
    """

    def __init__(
        self,
        graph: Any,
        *,
        input_key: str = "messages",
        output_key: str = "messages",
        message_factory: Optional[Callable[[str, Dict[str, Any]], Any]] = None,
    ) -> None:
        self._graph = graph
        self._input_key = input_key
        self._output_key = output_key
        self._message_factory = message_factory or _default_message_factory

    def invoke(self, envelope: Dict[str, Any]) -> Dict[str, Any]:
        message = envelope.get("message", "")
        metadata = envelope.get("metadata", {})
        graph_input = {
            self._input_key: self._message_factory(message, envelope),
        }
        result = self._graph.invoke(graph_input)

        answer_source = result
        if isinstance(result, dict) and self._output_key in result:
            answer_source = result[self._output_key]

        return {
            "answer": _content_from_message(answer_source),
            "final": True,
            "metadata": {
                "job_id": metadata.get("job_id", ""),
                "framework": "langgraph",
            },
        }

    def __call__(self, envelope: Dict[str, Any]) -> Dict[str, Any]:
        return self.invoke(envelope)


class _AdapterHandler(http.server.BaseHTTPRequestHandler):
    adapter: "AetherisLangGraphAdapter"

    def log_message(self, format, *args):
        pass

    def do_POST(self):
        length = int(self.headers.get("Content-Length", 0))
        try:
            envelope = json.loads(self.rfile.read(length))
        except json.JSONDecodeError:
            self.send_error(400, "Invalid JSON")
            return

        try:
            result = self.adapter.invoke(envelope)
            status = 200
        except Exception as exc:
            result = {
                "error": str(exc),
                "final": False,
                "metadata": {"framework": "langgraph"},
            }
            status = 502

        body = json.dumps(result).encode()
        self.send_response(status)
        self.send_header("Content-Type", "application/json")
        self.send_header("Content-Length", str(len(body)))
        self.end_headers()
        self.wfile.write(body)

    def do_GET(self):
        if self.path in ("/health", "/"):
            body = b'{"status":"ok","framework":"langgraph"}'
            self.send_response(200)
            self.send_header("Content-Type", "application/json")
            self.send_header("Content-Length", str(len(body)))
            self.end_headers()
            self.wfile.write(body)
        else:
            self.send_error(404)


def serve(
    graph: Any,
    *,
    port: int = 9000,
    host: str = "",
    input_key: str = "messages",
    output_key: str = "messages",
    message_factory: Optional[Callable[[str, Dict[str, Any]], Any]] = None,
    block: bool = True,
) -> Optional[threading.Thread]:
    """Serve a LangGraph graph as an Aetheris-compatible HTTP agent."""

    adapter = AetherisLangGraphAdapter(
        graph,
        input_key=input_key,
        output_key=output_key,
        message_factory=message_factory,
    )
    handler_cls = type("_BoundLangGraphHandler", (_AdapterHandler,), {"adapter": adapter})
    server = http.server.HTTPServer((host, port), handler_cls)

    if block:
        print(f"[aetheris] LangGraph agent listening on http://localhost:{port}")
        print("[aetheris] Add to runtime config:")
        print("[aetheris]   agents:")
        print("[aetheris]     agents:")
        print("[aetheris]       my_graph_agent:")
        print("[aetheris]         type: langgraph")
        print("[aetheris]         external:")
        print(f"[aetheris]           url: http://localhost:{port}")
        print("[aetheris]           timeout: 120s")
        print()
        try:
            server.serve_forever()
        except KeyboardInterrupt:
            pass
        return None

    thread = threading.Thread(target=server.serve_forever, daemon=True)
    thread.start()
    return thread
