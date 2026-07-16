//! Executor types — StepResult classification and execution metadata.

use serde::{Deserialize, Serialize};

/// Classifies the outcome of a single execution step.
#[derive(Clone, Debug, Serialize, Deserialize, PartialEq, Eq)]
pub enum StepResultType {
    /// Step completed successfully, output is a pure value.
    Success,
    /// Step produced a pure result (no side effects).
    PureResult,
    /// Step committed a side effect (already persisted externally).
    SideEffectCommitted,
    /// Step failed but can be retried.
    RetryableFailure,
    /// Step failed permanently (no retry).
    PermanentFailure,
    /// Step failed and its side effects need compensation.
    CompensatableFailure,
}

/// The outcome of executing a single DAG node.
#[derive(Clone, Debug, Serialize, Deserialize)]
pub struct StepResult {
    pub result_type: StepResultType,
    pub output: Option<Vec<u8>>,
    pub error_message: String,
    pub retry_after_ms: u32,
    pub side_effect_ids: Vec<String>,
}

impl StepResult {
    pub fn success(output: Vec<u8>) -> Self {
        Self {
            result_type: StepResultType::Success,
            output: Some(output),
            error_message: String::new(),
            retry_after_ms: 0,
            side_effect_ids: Vec::new(),
        }
    }

    pub fn pure(output: Vec<u8>) -> Self {
        Self {
            result_type: StepResultType::PureResult,
            output: Some(output),
            error_message: String::new(),
            retry_after_ms: 0,
            side_effect_ids: Vec::new(),
        }
    }

    pub fn side_effect_committed(output: Vec<u8>, effect_ids: Vec<String>) -> Self {
        Self {
            result_type: StepResultType::SideEffectCommitted,
            output: Some(output),
            error_message: String::new(),
            retry_after_ms: 0,
            side_effect_ids: effect_ids,
        }
    }

    pub fn retryable(error: &str, retry_after_ms: u32) -> Self {
        Self {
            result_type: StepResultType::RetryableFailure,
            output: None,
            error_message: error.to_string(),
            retry_after_ms,
            side_effect_ids: Vec::new(),
        }
    }

    pub fn permanent(error: &str) -> Self {
        Self {
            result_type: StepResultType::PermanentFailure,
            output: None,
            error_message: error.to_string(),
            retry_after_ms: 0,
            side_effect_ids: Vec::new(),
        }
    }

    pub fn compensatable(error: &str, effect_ids: Vec<String>) -> Self {
        Self {
            result_type: StepResultType::CompensatableFailure,
            output: None,
            error_message: error.to_string(),
            retry_after_ms: 0,
            side_effect_ids: effect_ids,
        }
    }

    pub fn is_success(&self) -> bool {
        matches!(
            self.result_type,
            StepResultType::Success | StepResultType::PureResult | StepResultType::SideEffectCommitted
        )
    }

    pub fn is_retryable(&self) -> bool {
        self.result_type == StepResultType::RetryableFailure
    }

    pub fn needs_compensation(&self) -> bool {
        self.result_type == StepResultType::CompensatableFailure
    }
}

/// Execution context passed to step executors.
#[derive(Clone, Debug, Serialize, Deserialize)]
pub struct StepContext {
    pub job_id: String,
    pub attempt_id: String,
    pub node_id: String,
    pub step_name: String,
    pub input: Vec<u8>,
    pub previous_outputs: Vec<(String, Vec<u8>)>,
}
