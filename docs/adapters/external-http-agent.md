# External HTTP Agent Adapter

`external_http` lets an existing HTTP-based agent run behind Aetheris with minimal migration. It is the recommended MVP path for teams that already have Python, JavaScript, or Go agents and want durable submission, event traces, timeouts, retries, and audit visibility before decomposing the agent into Runtime Tools.

This is a **Level 1 migration path** in the [guarantee matrix](../guides/guarantee-matrix.md). Aetheris controls the outer Job and the single `external_agent_call` Runtime Tool. It does not inspect or control the external service's internal steps.

For the path from black-box wrapping to Runtime Tool extraction, see the [external HTTP migration guide](external-http-migration.md).

## Configure

Add the agent under the top-level `agents` field in the active runtime config. For embedded API-only development, that is usually `configs/api.embedded.yaml`; for split API/Worker deployments, load the same agent definition into both the API and Worker configs (or mount a shared config into both) so the API can accept `POST /api/agents/:id/message` and the Worker can execute the job.

```yaml
agents:
  agents:
    customer_support_bot:
      type: external_http
      description: Existing customer support agent
      external:
        url: "http://customer-bot:9000/invoke"
        timeout: "120s"
        token_env: "CUSTOMER_BOT_TOKEN"
```

If `token_env` is set, the environment variable must exist at startup. Aetheris forwards it as `Authorization: Bearer <token>`.

## HTTP Protocol

Aetheris sends:

```json
{
  "message": "用户输入",
  "session_id": "session id",
  "metadata": {
    "agent_id": "customer_support_bot",
    "job_id": "job id",
    "idempotency_key": "stable key"
  }
}
```

The external agent returns:

```json
{
  "answer": "最终回复",
  "final": true,
  "metadata": {}
}
```

Aetheris also forwards `Idempotency-Key`, `X-Aetheris-Job-ID`, and `X-Aetheris-Agent-ID` headers. External agents should treat `Idempotency-Key` as the deduplication key for Level 1 migration.

## Reliability Boundary

Black-box mode gives Aetheris visibility over the outer call to the existing agent. It does not automatically make every API call inside that external agent at-most-once.

For high-risk side effects such as payments, email sends, database writes, or order creation, migrate those operations into Aetheris Runtime Tools. Only the Runtime Tool path participates in the Invocation Ledger and Effect Store at-most-once guarantee.

| Concern | `external_http` behavior |
|---|---|
| Job creation and status | Durable when shared JobStore/Event Store are configured |
| Outer HTTP call | Executed through the `external_agent_call` tool path |
| Idempotency key | Forwarded as `Idempotency-Key` plus metadata |
| Trace and audit | Aetheris records the outer call, result, timeout, and error |
| Internal tool calls inside the external agent | Opaque to Aetheris |
| Internal side effects | Must be deduplicated by the external service or migrated to Runtime Tools |

## Submit

Use the existing facade:

```bash
curl -X POST http://localhost:8080/api/agents/customer_support_bot/message \
  -H 'Content-Type: application/json' \
  -H 'Idempotency-Key: telegram-chat-1-message-1' \
  -d '{"message":"帮我查询订单状态"}'
```

The job plan contains one `external_agent_call` tool node, so existing Job/Event/Trace/Replay APIs continue to work.

---

## SSE-Legacy Protocol (sse_legacy)

Some agent platforms expose a streaming SSE endpoint instead of a synchronous JSON endpoint. `sse_legacy` mode lets Aetheris consume that stream and aggregate the result into a single durable event.

Set `protocol: "sse_legacy"` in the `external` block:

```yaml
agents:
  agents:
    my_streaming_agent:
      type: "external_http"
      description: "An agent that streams tokens via SSE"
      external:
        url: "http://my-agent:8888/api/v1/chat/stream"
        timeout: "120s"
        token_env: "MY_AGENT_API_KEY"
        protocol: "sse_legacy"
        agent_id: "research-agent"   # optional: forwarded as agent_id in the request body
```

### Wire format

Aetheris sends:

```json
{
  "agent_id": "research-agent",
  "session_id": "session id",
  "message": "用户输入"
}
```

The external service streams tokens as Server-Sent Events:

```
data: Hello\n\n
data:  world\n\n
data: [DONE]\n\n
```

Aetheris reads lines prefixed with `data:`, concatenates the token values, and treats `data: [DONE]` as the end-of-stream sentinel. The aggregated text becomes the job answer.

### Protocol comparison

| Field | `json` (default) | `sse_legacy` |
|---|---|---|
| Request body | `{message, session_id, metadata}` | `{agent_id, session_id, message}` |
| Response format | JSON `{answer, final, metadata}` | SSE stream; `data: [DONE]` signals end |
| `agent_id` config field | ignored | forwarded in request body |
| Use case | Standard REST agents | SSE-streaming agents (e.g. superagent-base) |

Both modes share the same at-most-once boundary: the outer `external_agent_call` Tool call is recorded in the Effect Store. Internal side effects inside the external service are opaque to Aetheris.

---

## superagent-base Integration

[superagent-base](https://github.com/Colin4k1024/superagent-base) is an open-source AI Agent platform that exposes all chat routes as SSE streams. It is the reference implementation for `sse_legacy` mode.

### Prerequisites

1. Clone and start superagent-base:

```bash
git clone https://github.com/Colin4k1024/superagent-base.git
cd superagent-base
# follow the README to start the service (default port 8888)
```

2. Set the API key environment variable:

```bash
export SUPERAGENT_BASE_API_KEY=your_api_key_here
```

### Enable the integration

Open `configs/agents.yaml` and uncomment the `superagent_base` block:

```yaml
agents:
  superagent_base:
    type: "external_http"
    description: "superagent-base 开源 AI Agent 平台（SSE 流式接入）"
    external:
      url: "http://localhost:8888/api/v1/chat/stream"
      timeout: "120s"
      token_env: "SUPERAGENT_BASE_API_KEY"
      protocol: "sse_legacy"
      agent_id: "research-agent"    # 可选，指定远端 Agent ID
```

### Submit a job

```bash
curl -X POST http://localhost:8080/api/agents/superagent_base/message \
  -H "Content-Type: application/json" \
  -H "Idempotency-Key: task-sb-001" \
  -d '{"message": "调研 Aetheris 的架构设计"}'
```

The response and all subsequent trace/replay APIs work identically to any other `external_http` agent.

### Guarantee boundary

Since superagent-base is a black-box service, the at-most-once guarantee applies only to the outer `external_agent_call` invocation. Side effects executed inside superagent-base (web search calls, LLM calls, etc.) are not tracked by Aetheris. For full side-effect guarantees, migrate those operations into Aetheris Runtime Tools.

See the [guarantee matrix](../guides/guarantee-matrix.md) for details.
