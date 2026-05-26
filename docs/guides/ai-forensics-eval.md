# AI Forensics Eval

AI forensics detection is an experimental anomaly detector for runtime evidence. It is promoted only as an eval-gated detector, not as an autonomous security decision maker.

## Boundary

`POST /api/forensics/ai/detect-anomalies` is available only when `api.forensics.experimental=true`.

The detector currently evaluates deterministic runtime signals:

- missing evidence
- inconsistent decision snapshots
- long-running decision steps
- low confidence
- suspicious retry loops
- tampered or hash-invalid reasoning snapshots

## Golden Eval Dataset

The golden dataset lives in `pkg/ai_forensics` and is versioned as `ai-forensics-golden-2026.05`.

It covers:

| Case | Expected |
|---|---|
| clean execution | no anomalies and zero false positives |
| missing evidence | `missing_evidence`, severity `high` |
| suspicious retry loop | `suspicious_retry_loop`, severity `high` |
| tampered reasoning snapshot | `tampered_reasoning`, severity `critical` |

## False-Positive Budget

Default budget:

- clean execution: `0` false positives
- anomaly cases: `0` false positives
- maximum false-positive rate: `10%`

Any detector change that exceeds the budget fails the release drill.

## Severity Mapping

| Anomaly | Severity |
|---|---|
| `missing_evidence` | `high` |
| `suspicious_retry_loop` | `high` or `critical` when retry count is very high |
| `tampered_reasoning` | `critical` |
| `inconsistent` | `medium` |
| `timing` | `medium` / `high` |
| `low_confidence` | `low` / `medium` / `high` based on threshold gap |

## Non-Goals

- Do not use this endpoint to auto-block production workflows without a human or policy layer.
- Do not promote the API to stable until the dataset, false-positive budget, and severity policy have release history.
- Do not treat detector output as legal, compliance, or security certification.
