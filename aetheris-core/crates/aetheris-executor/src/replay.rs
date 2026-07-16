//! Replay engine — deterministic re-execution from event log.

use aetheris_jobstore::{JobEvent, JobStore};

use crate::error::ExecutorError;
use crate::types::{StepResult, StepResultType};

/// Replay engine replays a job's execution from its event log.
///
/// Used for:
/// - Verifying execution correctness (determinism check)
/// - Debugging failed runs
/// - Recovering state after crashes
pub struct ReplayEngine;

impl ReplayEngine {
    /// Replay events for a job and verify consistency.
    ///
    /// Returns the reconstructed state as a list of (node_id, StepResult) pairs.
    pub async fn replay<S: JobStore>(
        store: &S,
        job_id: &str,
    ) -> Result<Vec<ReplayStep>, ExecutorError> {
        let (events, _version) = store
            .list_events(job_id)
            .await
            .map_err(|e| ExecutorError::ReplayError(e.to_string()))?;

        let mut steps = Vec::new();
        let mut current_step: Option<ReplayStep> = None;

        for event in &events {
            match event.event_type.as_str() {
                "step_started" | "StepStarted" => {
                    let node_id = extract_node_id(event);
                    current_step = Some(ReplayStep {
                        node_id,
                        input: event.payload.clone(),
                        result: None,
                        started_at: event.timestamp_ms,
                        completed_at: 0,
                    });
                }
                "step_completed" | "StepCompleted" => {
                    if let Some(mut step) = current_step.take() {
                        step.result = Some(parse_step_result(&event.payload));
                        step.completed_at = event.timestamp_ms;
                        steps.push(step);
                    }
                }
                "step_failed" | "StepFailed" => {
                    if let Some(mut step) = current_step.take() {
                        step.result = Some(StepResult::permanent(
                            &String::from_utf8_lossy(&event.payload),
                        ));
                        step.completed_at = event.timestamp_ms;
                        steps.push(step);
                    }
                }
                _ => {} // Skip other event types
            }
        }

        Ok(steps)
    }

    /// Verify that a replay produces the same results as the original execution.
    pub fn verify_consistency(
        original: &[ReplayStep],
        replayed: &[ReplayStep],
    ) -> Result<(), ExecutorError> {
        if original.len() != replayed.len() {
            return Err(ExecutorError::ReplayError(format!(
                "step count mismatch: original={}, replayed={}",
                original.len(),
                replayed.len()
            )));
        }

        for (orig, repl) in original.iter().zip(replayed.iter()) {
            if orig.node_id != repl.node_id {
                return Err(ExecutorError::ReplayError(format!(
                    "node mismatch: original={}, replayed={}",
                    orig.node_id, repl.node_id
                )));
            }

            match (&orig.result, &repl.result) {
                (Some(o), Some(r)) => {
                    if o.result_type != r.result_type {
                        return Err(ExecutorError::ReplayError(format!(
                            "result type mismatch on {}: original={:?}, replayed={:?}",
                            orig.node_id, o.result_type, r.result_type
                        )));
                    }
                }
                _ => {
                    return Err(ExecutorError::ReplayError(format!(
                        "missing result on node {}",
                        orig.node_id
                    )));
                }
            }
        }

        Ok(())
    }
}

/// A single step reconstructed from replay.
#[derive(Clone, Debug)]
pub struct ReplayStep {
    pub node_id: String,
    pub input: Vec<u8>,
    pub result: Option<StepResult>,
    pub started_at: i64,
    pub completed_at: i64,
}

fn extract_node_id(event: &JobEvent) -> String {
    // Try to parse node_id from JSON payload
    if let Ok(val) = serde_json::from_slice::<serde_json::Value>(&event.payload) {
        if let Some(id) = val.get("node_id").and_then(|v| v.as_str()) {
            return id.to_string();
        }
    }
    event.job_id.clone()
}

fn parse_step_result(payload: &[u8]) -> StepResult {
    if let Ok(val) = serde_json::from_slice::<serde_json::Value>(payload) {
        let result_type = val
            .get("type")
            .and_then(|v| v.as_str())
            .map(|s| match s {
                "SUCCESS" | "success" => StepResultType::Success,
                "PURE_RESULT" | "pure" => StepResultType::PureResult,
                "SIDE_EFFECT_COMMITTED" | "side_effect" => StepResultType::SideEffectCommitted,
                "RETRYABLE_FAILURE" | "retryable" => StepResultType::RetryableFailure,
                "PERMANENT_FAILURE" | "permanent" => StepResultType::PermanentFailure,
                "COMPENSATABLE_FAILURE" | "compensatable" => StepResultType::CompensatableFailure,
                _ => StepResultType::Success,
            })
            .unwrap_or(StepResultType::Success);

        let output = val
            .get("output")
            .and_then(|v| serde_json::to_vec(v).ok());

        let error = val
            .get("error_message")
            .and_then(|v| v.as_str())
            .unwrap_or("")
            .to_string();

        StepResult {
            result_type,
            output,
            error_message: error,
            retry_after_ms: 0,
            side_effect_ids: Vec::new(),
        }
    } else {
        StepResult::success(payload.to_vec())
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_verify_consistency_matching() {
        let steps = vec![ReplayStep {
            node_id: "n1".to_string(),
            input: vec![],
            result: Some(StepResult::success(b"out".to_vec())),
            started_at: 0,
            completed_at: 1,
        }];

        let replayed = steps.clone();
        ReplayEngine::verify_consistency(&steps, &replayed).unwrap();
    }

    #[test]
    fn test_verify_consistency_mismatch() {
        let original = vec![ReplayStep {
            node_id: "n1".to_string(),
            input: vec![],
            result: Some(StepResult::success(b"out".to_vec())),
            started_at: 0,
            completed_at: 1,
        }];

        let replayed = vec![ReplayStep {
            node_id: "n1".to_string(),
            input: vec![],
            result: Some(StepResult::permanent("different")),
            started_at: 0,
            completed_at: 1,
        }];

        let result = ReplayEngine::verify_consistency(&original, &replayed);
        assert!(result.is_err());
    }
}
