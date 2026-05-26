# Monitoring Quality Scorer

The monitoring quality scorer remains an offline SRE report utility. It is not wired into `/api/observability/*` yet.

## Decision

Keep the scorer outside `/api/observability/*` for now.

Reason: current observability APIs expose concrete runtime operations signals: queue backlog and stuck jobs. Decision quality scores are derived analytical signals. Mixing them into the same API without alert semantics would make SRE response ambiguous.

## Alert Semantics

`monitoring.AssessQualityScore` maps scores to SRE alert classes:

| Level | Alert | Meaning |
|---|---|---|
| `healthy` | false | no immediate action |
| `degraded` | true | quality below normal operating threshold or recommendations exist |
| `critical` | true | score or a core dimension is below critical threshold |
| `noisy` | true | evidence coverage is high but model confidence is low; inspect signal quality/calibration |

## Suggested SRE Workflow

1. Use `/api/observability/summary` for queue backlog and stuck jobs.
2. Use the quality scorer as an offline investigation aid for selected high-risk steps.
3. Treat `critical` as an escalation signal.
4. Treat `noisy` as a calibration or evidence-quality investigation, not as proof of bad execution.

## Promotion Requirements

Before wiring into `/api/observability/*`:

- define aggregation windows
- decide whether alerts are per-step, per-job, or per-tenant
- expose Prometheus metrics with stable labels
- document on-call routing and suppression rules
