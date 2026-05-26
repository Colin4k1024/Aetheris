# Effect Store Contract

This document is the current source of truth for the Effect Store write order and recovery boundary.

## Purpose

The Effect Store records completed non-deterministic effects before the runtime commits the corresponding execution events. It covers the crash window where a Tool or LLM call has returned but the event stream has not yet recorded `command_committed` / `tool_invocation_finished`.

## Success Path

For a Runtime Tool step:

```text
Append tool_invocation_started
Acquire execution permission through Invocation Ledger
Execute Tool
PutEffect(input, output, metadata)
Append tool_invocation_finished
Append command_committed
Commit Invocation Ledger / ToolInvocationStore
Append node_finished / step_committed / checkpoint
```

For an LLM step:

```text
Append command_emitted
PutEffect(phase=started) when configured
Call LLM
PutEffect(prompt, response, model metadata)
Append command_committed
Append node_finished / step_committed / checkpoint
```

The event stream remains the replay authority after commit. The Effect Store is the recovery bridge for completed effects that have not yet reached the event stream.

## Recovery Path

On replay or reclaim, the runtime checks in this order:

1. If event history contains a completed command/tool result, inject that result.
2. If event history has `tool_invocation_started` without a finished event, do not execute again. Recover from Ledger or Effect Store, then catch up the missing finished/committed events.
3. If event history has no committed result but Effect Store has output for the same job and idempotency key, append catch-up events and inject the Effect Store output.
4. If neither event history, Ledger, nor Effect Store has a result, the step may execute only if no pending activity barrier says the side effect may already be in flight.

## Failure Windows

| Crash window | Recovery behavior |
|---|---|
| Before Tool/LLM executes | Step may execute normally after reclaim. |
| After `tool_invocation_started`, before Tool returns | Pending activity barrier prevents blind re-execution; recovery needs Ledger result, Effect Store result, timeout policy, or manual intervention. |
| After Tool/LLM returns, before `PutEffect` succeeds | Pending activity barrier prevents blind re-execution if the started event exists; without downstream idempotency the safest outcome is permanent failure/manual recovery. |
| After `PutEffect`, before commit events | Catch up from Effect Store and do not call Tool/LLM again. |
| After commit events, before Ledger commit | Replay injects from event history; Ledger can be reconciled/caught up. |
| After all commits | Replay injects from event history/Ledger and does not re-execute. |

## Invariants

- Production Runtime Tools require a shared Invocation Ledger and shared Effect Store for the strongest guarantee.
- Effect Store writes must not be treated as optional after a side effect succeeds in production.
- `external_http` only records the outer `external_agent_call`; internal effects inside the black-box service are not covered by this contract.
- Event history is still the long-term replay authority; Effect Store is used to bridge incomplete commit windows and provide LLM/tool effect metadata.
