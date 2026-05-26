# Aetheris Job Lifecycle

This page explains the authoritative lifecycle of an Aetheris Job. It is the operator-facing companion to the formal runtime contract.

## Short Version

```text
Submit
  -> JobCreated
  -> PlanGenerated
  -> Queued/Pending
  -> Worker Claim + Lease
  -> NodeStarted / ToolInvocationStarted / CommandEmitted
  -> Effect Store + CommandCommitted / ToolInvocationFinished
  -> NodeFinished / StepCommitted / Checkpoint
  -> JobCompleted or JobFailed
```

For wait or human-in-the-loop flows:

```text
NodeStarted
  -> JobWaiting / JobParked
  -> WaitCompleted / JobResumed
  -> Worker Claim
  -> Continue from recorded plan and resumption context
```

## Control Plane vs Execution Plane

| Plane | Components | Responsibility |
|---|---|---|
| Control Plane | API, CLI, SDK, auth, config | Accept requests, validate input, create Job metadata, append initial events, expose status/trace/replay |
| Execution Plane | Worker, Scheduler, Runner, NodeAdapter, Tool plane | Claim Jobs, heartbeat leases, execute recorded plans, write checkpoints, manage effects |
| Evidence Plane | Trace, Replay, Verify, Evidence export | Read event history and derived stores for debugging, audit, and proof |

In production Postgres mode, the API does not own execution. Workers claim Jobs through the durable JobStore and are fenced by lease/attempt semantics.

## State Transitions

| State / event | Meaning | Owner |
|---|---|---|
| `job_created` | A durable Job exists | API |
| `plan_generated` | The execution path has been recorded | API or planner path before execution |
| `job_queued` / Pending metadata | Job is eligible for Worker claim | API / Scheduler |
| `job_leased` / Running metadata | A Worker owns the current attempt | Worker / JobStore |
| `node_started` | A recorded node began execution | Runner |
| `tool_invocation_started` | A Runtime Tool side-effect attempt began | ToolNodeAdapter |
| `command_emitted` | A non-deterministic command was requested | NodeAdapter |
| `command_committed` / `tool_invocation_finished` | A result is recorded and replay can inject it | NodeAdapter |
| `step_committed` | Step-level commit barrier completed | Runner |
| `checkpoint_saved` | Resume cursor/state is durable | Runner |
| `job_waiting` / `job_parked` | Execution is intentionally suspended | Runner |
| `wait_completed` / `job_resumed` | External signal has unblocked the Job | API / signal path |
| `job_completed` | Terminal success | Runner |
| `job_failed` | Terminal failure | Runner / Scheduler |

## Recovery Rules

- If a Worker crashes before claiming a Job, another Worker can claim it normally.
- If a Worker crashes after claiming, lease expiry and reclaim make the Job eligible again.
- If a step was already committed in event history, replay injects its result.
- If a Tool started but did not finish, the Activity Log Barrier prevents blind re-execution.
- If an Effect Store record exists without committed events, catch-up appends the missing events and injects the recorded result.
- If a Job is waiting, reclaim must not run it until a matching signal writes `wait_completed`.

## External HTTP Jobs

For an `external_http` agent, the plan contains one `external_agent_call` Runtime Tool node. Aetheris can recover and trace that outer call, but cannot see or recover the external service's internal steps unless they are exposed as Aetheris Runtime Tools.

## Related Docs

- [Guarantee matrix](guarantee-matrix.md)
- [Runtime guarantees](runtime-guarantees.md)
- [External HTTP adapter](../adapters/external-http-agent.md)
- [Execution guarantees](../../design/execution-guarantees.md)
- [Effect Store contract](../../design/internal/effect-store-contract.md)
- [Event taxonomy](../../design/internal/event-taxonomy.md)
