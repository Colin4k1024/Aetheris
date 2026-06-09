"""
aetheris.integrations.langchain
~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

Make any LangChain agent or chain durable with Aetheris.

Usage::

    from langchain_openai import ChatOpenAI
    from langchain.agents import AgentExecutor, create_react_agent
    from aetheris.integrations.langchain import serve, AetherisLangChainAdapter

    # Build your agent as usual
    agent_executor = create_react_agent(ChatOpenAI(), tools, prompt)

    # Option 1: Serve it as a durable HTTP endpoint (Aetheris calls this)
    serve(agent_executor, port=9000)

    # Option 2: Use the adapter directly (for testing/embedding)
    adapter = AetherisLangChainAdapter(agent_executor)
    response = adapter.invoke({"message": "What is 2+2?"})

"""

from __future__ import annotations

import http.server
import json
import threading
from typing import Any, Dict, Optional

from .embedded import (
    AetherisChatModel,
    AetherisRuntimeContext,
    AetherisRuntimeTool,
    EmbeddedAgentManifest,
    serve_embedded,
)

__all__ = [
    "AetherisLangChainAdapter",
    "AetherisRuntimeContext",
    "AetherisRuntimeTool",
    "AetherisChatModel",
    "EmbeddedAgentManifest",
    "serve",
    "serve_embedded",
]


class AetherisLangChainAdapter:
    """Wraps a LangChain runnable (Agent, Chain, etc.) to handle Aetheris job envelopes.

    Aetheris calls your agent with a JSON envelope like::

        {
            "message": "user goal or prompt",
            "session_id": "...",
            "metadata": {
                "agent_id": "...",
                "job_id": "...",
                "idempotency_key": "..."
            }
        }

    This adapter unpacks the envelope, calls your LangChain runnable, and
    formats the response back as::

        {
            "answer": "...",
            "final": true,
            "metadata": {}
        }

    Args:
        runnable: Any LangChain Runnable — AgentExecutor, Chain, LCEL pipe, etc.
        input_key: Key used to pass the message to ``runnable.invoke()``.
            Defaults to ``"input"`` (works for most ReAct agents).
            Use ``"messages"`` for ChatModel-based runnables.
        output_key: Key to extract from the runnable's output dict.
            Defaults to ``"output"`` (AgentExecutor) or falls back to str().

    Example::

        from langchain.agents import AgentExecutor
        adapter = AetherisLangChainAdapter(agent_executor)

        # Receives the raw Aetheris job envelope
        result = adapter.invoke({"message": "Summarize this document..."})
        # Returns {"answer": "...", "final": True, "metadata": {}}

    """

    def __init__(
        self,
        runnable: Any,
        *,
        input_key: str = "input",
        output_key: str = "output",
    ) -> None:
        self._runnable = runnable
        self._input_key = input_key
        self._output_key = output_key

    def invoke(self, envelope: Dict[str, Any]) -> Dict[str, Any]:
        """Process one Aetheris job invocation.

        Args:
            envelope: The JSON body Aetheris sends to your agent endpoint.

        Returns:
            Dict with ``answer``, ``final``, and ``metadata`` keys.
        """
        message = envelope.get("message", "")
        metadata = envelope.get("metadata", {})
        result = self._runnable.invoke({self._input_key: message})

        if isinstance(result, dict):
            answer = result.get(self._output_key, str(result))
        else:
            answer = str(result)

        return {
            "answer": answer,
            "final": True,
            "metadata": {
                "job_id": metadata.get("job_id", ""),
                "framework": "langchain",
            },
        }

    def __call__(self, envelope: Dict[str, Any]) -> Dict[str, Any]:
        return self.invoke(envelope)


class _AdapterHandler(http.server.BaseHTTPRequestHandler):
    """HTTP handler that calls the adapter for each POST request."""

    adapter: "AetherisLangChainAdapter"  # set on the class before serving

    def log_message(self, format, *args):  # suppress default request logs
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
                "metadata": {},
            }
            status = 502

        body = json.dumps(result).encode()
        self.send_response(status)
        self.send_header("Content-Type", "application/json")
        self.send_header("Content-Length", str(len(body)))
        self.end_headers()
        self.wfile.write(body)

    def do_GET(self):
        # Health check endpoint
        if self.path in ("/health", "/"):
            body = b'{"status":"ok"}'
            self.send_response(200)
            self.send_header("Content-Type", "application/json")
            self.send_header("Content-Length", str(len(body)))
            self.end_headers()
            self.wfile.write(body)
        else:
            self.send_error(404)


def serve(
    runnable: Any,
    *,
    port: int = 9000,
    host: str = "",
    input_key: str = "input",
    output_key: str = "output",
    block: bool = True,
) -> Optional[threading.Thread]:
    """Serve a LangChain runnable as a durable Aetheris-compatible HTTP agent.

    After calling this, add the agent to your Aetheris config::

        # configs/api.embedded.yaml
        agents:
          agents:
            my_langchain_agent:
              type: "external_http"
              external:
                url: "http://localhost:9000"
                timeout: "120s"

    Then submit jobs via the Aetheris API or Python SDK::

        from aetheris import AetherisClient
        client = AetherisClient()
        job = client.run("my_langchain_agent", "Explain quantum computing")
        print(job.wait().output)

    Args:
        runnable: Any LangChain Runnable — AgentExecutor, Chain, LCEL pipe, etc.
        port: Port to listen on. Default 9000.
        host: Host to bind to. Default ``""`` (all interfaces).
        input_key: Input dict key for the runnable. Default ``"input"``.
        output_key: Output dict key to extract. Default ``"output"``.
        block: If True (default), blocks the calling thread. If False,
            starts a daemon thread and returns it.

    Returns:
        The daemon thread if ``block=False``, otherwise ``None``.

    Example::

        from langchain_openai import ChatOpenAI
        from langchain.agents import create_react_agent, AgentExecutor
        from aetheris.integrations.langchain import serve

        llm = ChatOpenAI(model="gpt-4o-mini")
        agent = create_react_agent(llm, tools=[], prompt=hub.pull("hwchase17/react"))
        executor = AgentExecutor(agent=agent, tools=[])

        print(f"Agent listening on http://localhost:9000")
        serve(executor)  # blocks until Ctrl+C

    """
    adapter = AetherisLangChainAdapter(runnable, input_key=input_key, output_key=output_key)

    # Inject adapter into handler class (thread-safe: one handler class per server)
    handler_cls = type(
        "_BoundHandler",
        (_AdapterHandler,),
        {"adapter": adapter},
    )

    server = http.server.HTTPServer((host, port), handler_cls)

    if block:
        print(f"[aetheris] LangChain agent listening on http://localhost:{port}")
        print("[aetheris] Add to runtime config:")
        print("[aetheris]   agents:")
        print("[aetheris]     agents:")
        print("[aetheris]       my_agent:")
        print("[aetheris]         type: langchain")
        print("[aetheris]         external:")
        print(f"[aetheris]           url: http://localhost:{port}")
        print("[aetheris]           timeout: 120s")
        print()
        try:
            server.serve_forever()
        except KeyboardInterrupt:
            pass
        return None
    else:
        t = threading.Thread(target=server.serve_forever, daemon=True)
        t.start()
        return t
