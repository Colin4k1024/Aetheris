# External HTTP Agent Adapter

`external_http` lets an existing HTTP-based agent run behind Aetheris with minimal migration. It is the recommended MVP path for teams that already have Python, JavaScript, or Go agents and want durable submission, event traces, timeouts, retries, and audit visibility before decomposing the agent into Runtime Tools.

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

## Submit

Use the existing facade:

```bash
curl -X POST http://localhost:8080/api/agents/customer_support_bot/message \
  -H 'Content-Type: application/json' \
  -H 'Idempotency-Key: telegram-chat-1-message-1' \
  -d '{"message":"帮我查询订单状态"}'
```

The job plan contains one `external_agent_call` tool node, so existing Job/Event/Trace/Replay APIs continue to work.
