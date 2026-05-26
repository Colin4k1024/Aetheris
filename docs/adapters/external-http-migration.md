# External HTTP Agent Migration Guide

This guide describes how to move an existing black-box HTTP agent from basic Aetheris wrapping to full Runtime Tool reliability.

## Migration Levels

| Level | Pattern | What changes | Reliability boundary |
|---|---|---|---|
| Level 1 | Black-box `external_http` | Register the existing `/invoke` endpoint in Aetheris config | Aetheris controls the outer Job and HTTP call only |
| Level 2 | Propagated idempotency | External agent forwards `Idempotency-Key` to its own payment/email/write calls | Duplicate risk reduced when downstream systems honor the key |
| Level 3 | Runtime Tool extraction | High-risk side effects move into Aetheris Runtime Tools | Invocation Ledger + Effect Store protect those side effects |

## Level 1: Wrap the Existing Agent

Use this when the team needs durable submission, trace, timeout, retries, and audit visibility without changing the agent internals.

```yaml
agents:
  agents:
    customer_support_bot:
      type: external_http
      external:
        url: "http://customer-bot:9000/invoke"
        timeout: "120s"
        token_env: "CUSTOMER_BOT_TOKEN"
```

Aetheris sends the message, job metadata, and idempotency key. The external service returns the final answer.

Boundary:

- Aetheris records the outer `external_agent_call`.
- Internal LLM calls, tool calls, database writes, emails, and payments remain opaque.

## Level 2: Propagate Idempotency

Use this when internal side effects cannot yet be moved into Runtime Tools.

External service requirements:

- Read `Idempotency-Key` from the Aetheris request.
- Derive stable child keys for internal operations.
- Forward those keys to downstream APIs that support deduplication.
- Store local operation results keyed by the idempotency key.

Example key derivation:

```text
outer_key = request.headers["Idempotency-Key"]
email_key = outer_key + ":email:welcome"
payment_key = outer_key + ":payment:refund"
db_key = outer_key + ":db:case-update"
```

Boundary:

- Aetheris still cannot inspect the internal steps.
- Reliability depends on the external service and downstream APIs honoring those keys.
- Trace shows the outer call, not each internal side effect.

## Level 3: Extract Runtime Tools

Use this for high-risk operations:

- payments and refunds
- email/SMS sends
- order creation
- database writes that must not duplicate
- calls whose result must be replayed exactly

Migration pattern:

1. Keep the existing external agent for reasoning and orchestration.
2. Move one high-risk side effect into an Aetheris Runtime Tool.
3. Let the external agent request that side effect through Aetheris or convert the workflow into native Aetheris plan nodes.
4. Repeat until all high-risk side effects are Runtime Tools.

Boundary:

- Runtime Tools participate in Invocation Ledger and Effect Store.
- Replay injects recorded results instead of calling the side effect again.
- Event history and trace can explain each side effect.

## Decision Rule

| If the operation can safely happen twice | Keep it inside `external_http` for now |
|---|---|
| If duplicate execution is costly but downstream supports idempotency | Level 2 is acceptable short term |
| If duplicate execution is unacceptable | Move it to a Runtime Tool |

## Related Docs

- [External HTTP adapter](external-http-agent.md)
- [Guarantee matrix](../guides/guarantee-matrix.md)
- [Effect Store contract](../../design/internal/effect-store-contract.md)
- [Runtime guarantees](../guides/runtime-guarantees.md)
