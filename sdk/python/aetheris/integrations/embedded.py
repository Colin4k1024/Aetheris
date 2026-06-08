"""
Embedded framework runtime bridge for Aetheris.
"""

from __future__ import annotations

import hashlib
import http.server
import inspect
import json
import threading
import urllib.request
from dataclasses import dataclass, field
from pathlib import Path
from typing import Any, Callable, Dict, List, Optional

SCHEMA_VERSION = "aetheris.framework.v1"


@dataclass
class EmbeddedAgentManifest:
    name: str
    framework: str
    input_node: str = ""
    output_node: str = ""
    nodes: List[Dict[str, Any]] = field(default_factory=list)
    edges: List[Dict[str, str]] = field(default_factory=list)
    _callables: Dict[str, Callable[..., Any]] = field(default_factory=dict, init=False, repr=False)

    def remote_node(self, node_id: str, *, callable: Callable[..., Any], config: Optional[Dict[str, Any]] = None) -> "EmbeddedAgentManifest":
        self._callables[node_id] = callable
        self.nodes.append(
            {
                "id": node_id,
                "kind": "remote_callable",
                "callable": getattr(callable, "__name__", node_id),
                "config": config or {},
            }
        )
        return self

    def runtime_llm(self, node_id: str, *, prompt_key: str = "", model: str = "default", config: Optional[Dict[str, Any]] = None) -> "EmbeddedAgentManifest":
        cfg = dict(config or {})
        if prompt_key:
            cfg["prompt_key"] = prompt_key
        if model:
            cfg["model"] = model
        self.nodes.append({"id": node_id, "kind": "runtime_llm", "config": cfg})
        return self

    def runtime_tool(self, node_id: str, *, tool_name: str, config: Optional[Dict[str, Any]] = None) -> "EmbeddedAgentManifest":
        self.nodes.append(
            {
                "id": node_id,
                "kind": "runtime_tool",
                "tool_name": tool_name,
                "config": config or {},
            }
        )
        return self

    def runtime_workflow(self, node_id: str, *, workflow: str, config: Optional[Dict[str, Any]] = None) -> "EmbeddedAgentManifest":
        self.nodes.append(
            {
                "id": node_id,
                "kind": "runtime_workflow",
                "workflow": workflow,
                "config": config or {},
            }
        )
        return self

    def wait(self, node_id: str, *, config: Optional[Dict[str, Any]] = None) -> "EmbeddedAgentManifest":
        self.nodes.append({"id": node_id, "kind": "wait", "config": config or {}})
        return self

    def approval(self, node_id: str, *, config: Optional[Dict[str, Any]] = None) -> "EmbeddedAgentManifest":
        self.nodes.append({"id": node_id, "kind": "approval", "config": config or {}})
        return self

    def edge(self, from_node: str, to_node: str) -> "EmbeddedAgentManifest":
        self.edges.append({"from": from_node, "to": to_node})
        return self

    def to_dict(self) -> Dict[str, Any]:
        return {
            "schema_version": SCHEMA_VERSION,
            "name": self.name,
            "framework": self.framework,
            "input_node": self.input_node,
            "output_node": self.output_node,
            "nodes": self.nodes,
            "edges": self.edges,
        }

    def to_json(self) -> str:
        return json.dumps(self.to_dict(), sort_keys=True)

    def save(self, path: str) -> None:
        target = Path(path)
        target.parent.mkdir(parents=True, exist_ok=True)
        target.write_text(self.to_json() + "\n", encoding="utf-8")


@dataclass
class AetherisRuntimeContext:
    base_url: str = "http://localhost:8080"
    job_id: str = ""
    session_id: str = ""
    idempotency_key: str = ""
    token: Optional[str] = None

    @classmethod
    def from_envelope(cls, envelope: Dict[str, Any], *, base_url: str = "http://localhost:8080", token: Optional[str] = None) -> "AetherisRuntimeContext":
        metadata = envelope.get("metadata") or {}
        return cls(
            base_url=base_url,
            job_id=envelope.get("job_id") or metadata.get("job_id", ""),
            session_id=envelope.get("session_id", ""),
            idempotency_key=metadata.get("idempotency_key", ""),
            token=token,
        )

    def child_key(self, *parts: str) -> str:
        seed = ":".join([self.idempotency_key, self.job_id, *parts])
        return hashlib.sha256(seed.encode()).hexdigest()

    def post(self, path: str, body: Dict[str, Any]) -> Dict[str, Any]:
        data = json.dumps(body).encode()
        headers = {"Content-Type": "application/json"}
        if self.token:
            headers["Authorization"] = f"Bearer {self.token}"
        request = urllib.request.Request(
            self.base_url.rstrip("/") + path,
            data=data,
            headers=headers,
            method="POST",
        )
        with urllib.request.urlopen(request, timeout=30) as response:
            return json.loads(response.read() or b"{}")


class AetherisRuntimeTool:
    def __init__(self, name: str, *, context: AetherisRuntimeContext, node_id: Optional[str] = None) -> None:
        self.name = name
        self.context = context
        self.node_id = node_id or name

    def invoke(self, input: Optional[Dict[str, Any]] = None) -> Dict[str, Any]:
        return self.context.post(
            f"/api/jobs/{self.context.job_id}/runtime/tools/{self.name}/invoke",
            {
                "node_id": self.node_id,
                "session_id": self.context.session_id,
                "input": input or {},
                "metadata": {
                    "child_idempotency_key": self.context.child_key("tool", self.node_id, self.name),
                },
            },
        )

    def __call__(self, input: Optional[Dict[str, Any]] = None) -> Dict[str, Any]:
        return self.invoke(input)


class AetherisChatModel:
    def __init__(self, *, context: AetherisRuntimeContext, model: str = "default", node_id: str = "runtime_llm") -> None:
        self.context = context
        self.model = model
        self.node_id = node_id

    def invoke(self, prompt: str) -> Dict[str, Any]:
        return self.context.post(
            f"/api/jobs/{self.context.job_id}/runtime/llm/invoke",
            {
                "node_id": self.node_id,
                "session_id": self.context.session_id,
                "prompt": prompt,
                "metadata": {
                    "model": self.model,
                    "child_idempotency_key": self.context.child_key("llm", self.node_id),
                },
            },
        )

    def __call__(self, prompt: str) -> Dict[str, Any]:
        return self.invoke(prompt)


class _EmbeddedHandler(http.server.BaseHTTPRequestHandler):
    manifest: EmbeddedAgentManifest
    base_url: str
    token: Optional[str]

    def log_message(self, format, *args):
        pass

    def _write_json(self, status: int, payload: Dict[str, Any]) -> None:
        body = json.dumps(payload).encode()
        self.send_response(status)
        self.send_header("Content-Type", "application/json")
        self.send_header("Content-Length", str(len(body)))
        self.end_headers()
        self.wfile.write(body)

    def do_GET(self):
        if self.path in ("/health", "/"):
            self._write_json(200, {"status": "ok", "mode": "embedded"})
            return
        if self.path == "/aetheris/manifest":
            self._write_json(200, self.manifest.to_dict())
            return
        self.send_error(404)

    def do_POST(self):
        if not self.path.startswith("/aetheris/nodes/") or not self.path.endswith("/invoke"):
            self.send_error(404)
            return
        node_id = self.path[len("/aetheris/nodes/") : -len("/invoke")].strip("/")
        callable_fn = self.manifest._callables.get(node_id)
        if callable_fn is None:
            self._write_json(404, {"error": f"unknown callable node {node_id}"})
            return
        length = int(self.headers.get("Content-Length", 0))
        try:
            envelope = json.loads(self.rfile.read(length))
        except json.JSONDecodeError:
            self._write_json(400, {"error": "Invalid JSON"})
            return
        context = AetherisRuntimeContext.from_envelope(envelope, base_url=self.base_url, token=self.token)
        try:
            output = _invoke_callable(callable_fn, envelope, context)
            self._write_json(200, {"output": output, "final": True, "metadata": {"node_id": node_id}})
        except Exception as exc:
            self._write_json(502, {"error": str(exc), "final": False, "metadata": {"node_id": node_id}})


def _invoke_callable(callable_fn: Callable[..., Any], envelope: Dict[str, Any], context: AetherisRuntimeContext) -> Any:
    params = inspect.signature(callable_fn).parameters
    if len(params) == 0:
        return callable_fn()
    if len(params) == 1:
        return callable_fn(envelope)
    if len(params) == 2:
        return callable_fn(envelope.get("input", {}), envelope.get("prior_results", {}))
    return callable_fn(envelope.get("input", {}), envelope.get("prior_results", {}), context)


def serve_embedded(
    manifest: EmbeddedAgentManifest,
    *,
    port: int = 9000,
    host: str = "",
    base_url: str = "http://localhost:8080",
    token: Optional[str] = None,
    block: bool = True,
) -> Optional[threading.Thread]:
    handler_cls = type(
        "_BoundEmbeddedHandler",
        (_EmbeddedHandler,),
        {"manifest": manifest, "base_url": base_url, "token": token},
    )
    server = http.server.HTTPServer((host, port), handler_cls)
    if block:
        print(f"[aetheris] Embedded {manifest.framework} agent listening on http://localhost:{port}")
        print("[aetheris] Add to runtime config with external.mode: embedded")
        try:
            server.serve_forever()
        except KeyboardInterrupt:
            pass
        return None
    thread = threading.Thread(target=server.serve_forever, daemon=True)
    thread.server = server  # type: ignore[attr-defined]
    thread.start()
    return thread
