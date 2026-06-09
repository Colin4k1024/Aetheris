import json
import urllib.request

from aetheris.integrations.embedded import (
    AetherisRuntimeContext,
    AetherisRuntimeTool,
    EmbeddedAgentManifest,
    serve_embedded,
)


def test_embedded_manifest_serializes_schema():
    manifest = EmbeddedAgentManifest(
        name="research_agent",
        framework="langchain",
        input_node="load_question",
        output_node="final_answer",
    )
    manifest.remote_node("load_question", callable=lambda request: {"prompt": "hello"})
    manifest.runtime_llm("reason", prompt_key="load_question", model="default")
    manifest.runtime_tool("search", tool_name="knowledge.search")
    manifest.edge("load_question", "reason")

    payload = manifest.to_dict()

    assert payload["schema_version"] == "aetheris.framework.v1"
    assert payload["framework"] == "langchain"
    assert payload["nodes"][0]["kind"] == "remote_callable"
    assert payload["nodes"][1]["config"]["prompt_key"] == "load_question"
    assert payload["nodes"][2]["tool_name"] == "knowledge.search"


def test_embedded_manifest_save_writes_json(tmp_path):
    manifest = EmbeddedAgentManifest(name="research_agent", framework="langchain")
    manifest.runtime_tool("search", tool_name="knowledge.search")
    path = tmp_path / "framework-agents" / "research_agent.manifest.json"

    manifest.save(str(path))

    payload = json.loads(path.read_text())
    assert payload["schema_version"] == "aetheris.framework.v1"
    assert payload["nodes"][0]["id"] == "search"


def test_runtime_context_child_key_is_stable():
    context = AetherisRuntimeContext(job_id="job-1", idempotency_key="root")

    assert context.child_key("tool", "search") == context.child_key("tool", "search")
    assert context.child_key("tool", "search") != context.child_key("tool", "email")


class FakeContext:
    job_id = "job-1"
    session_id = "sess-1"

    def __init__(self):
        self.calls = []

    def child_key(self, *parts):
        return ":".join(parts)

    def post(self, path, body):
        self.calls.append((path, body))
        return {"result": {"ok": True}}


def test_runtime_tool_wrapper_calls_bridge():
    context = FakeContext()
    tool = AetherisRuntimeTool("knowledge.search", context=context, node_id="search")

    result = tool.invoke({"query": "hello"})

    assert result == {"result": {"ok": True}}
    assert context.calls[0][0] == "/api/jobs/job-1/runtime/tools/knowledge.search/invoke"
    assert context.calls[0][1]["node_id"] == "search"
    assert context.calls[0][1]["input"] == {"query": "hello"}


def test_serve_embedded_starts_background_thread():
    def load_question(input, prior_results, context):
        return {
            "prompt": input["topic"],
            "job_id": context.job_id,
        }

    manifest = EmbeddedAgentManifest(name="research_agent", framework="langgraph")
    manifest.remote_node("load_question", callable=load_question)
    thread = serve_embedded(manifest, host="127.0.0.1", port=0, block=False)

    try:
        assert thread is not None
        assert hasattr(thread, "server")
    finally:
        thread.server.shutdown()
        thread.server.server_close()


def test_embedded_handler_round_trip_with_fixed_port():
    def load_question(envelope):
        return {"prompt": envelope["input"]["topic"]}

    manifest = EmbeddedAgentManifest(name="research_agent", framework="langchain")
    manifest.remote_node("load_question", callable=load_question)
    server_thread = serve_embedded(manifest, host="127.0.0.1", port=0, block=False)
    port = server_thread.server.server_port
    try:
        with urllib.request.urlopen(f"http://127.0.0.1:{port}/aetheris/manifest", timeout=5) as response:
            payload = json.loads(response.read())
        assert payload["schema_version"] == "aetheris.framework.v1"

        request = urllib.request.Request(
            f"http://127.0.0.1:{port}/aetheris/nodes/load_question/invoke",
            data=json.dumps({"job_id": "job-1", "input": {"topic": "hello"}}).encode(),
            headers={"Content-Type": "application/json"},
            method="POST",
        )
        with urllib.request.urlopen(request, timeout=5) as response:
            result = json.loads(response.read())
        assert result["output"] == {"prompt": "hello"}
    finally:
        server_thread.server.shutdown()
        server_thread.server.server_close()
