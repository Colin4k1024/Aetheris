//! Aetheris config crate.
//!
//! Configuration loading for agents, workers, and runtime settings.
//! Reads YAML files with environment variable override support.

mod agent;
mod error;

pub use agent::{AgentConfig, ToolBinding};
pub use error::ConfigError;

use serde::{Deserialize, Serialize};

/// Top-level Aetheris configuration.
#[derive(Clone, Debug, Serialize, Deserialize)]
pub struct AetherisConfig {
    /// Agent definitions.
    #[serde(default)]
    pub agents: Vec<AgentConfig>,
    /// Runtime settings.
    #[serde(default)]
    pub runtime: RuntimeConfig,
    /// Worker settings.
    #[serde(default)]
    pub worker: WorkerConfig,
}

/// Runtime configuration.
#[derive(Clone, Debug, Serialize, Deserialize, Default)]
pub struct RuntimeConfig {
    /// Maximum concurrent jobs per worker.
    #[serde(default = "default_max_concurrent_jobs")]
    pub max_concurrent_jobs: usize,
    /// Default step timeout in milliseconds.
    #[serde(default = "default_step_timeout_ms")]
    pub step_timeout_ms: u64,
    /// Checkpoint after each step.
    #[serde(default = "default_true")]
    pub checkpoint_after_each_step: bool,
    /// Database connection string.
    #[serde(default)]
    pub database_url: String,
}

/// Worker configuration.
#[derive(Clone, Debug, Serialize, Deserialize, Default)]
pub struct WorkerConfig {
    /// Worker ID (auto-generated if empty).
    #[serde(default)]
    pub worker_id: String,
    /// Poll interval in milliseconds.
    #[serde(default = "default_poll_interval_ms")]
    pub poll_interval_ms: u64,
    /// Lease duration in seconds.
    #[serde(default = "default_lease_duration_secs")]
    pub lease_duration_secs: i64,
}

fn default_max_concurrent_jobs() -> usize {
    4
}
fn default_step_timeout_ms() -> u64 {
    60_000
}
fn default_true() -> bool {
    true
}
fn default_poll_interval_ms() -> u64 {
    1000
}
fn default_lease_duration_secs() -> i64 {
    30
}

impl AetherisConfig {
    /// Load configuration from a YAML string.
    pub fn from_yaml(yaml: &str) -> Result<Self, ConfigError> {
        serde_yaml::from_str(yaml).map_err(|e| ConfigError::Parse(e.to_string()))
    }

    /// Load configuration from a YAML file path.
    pub fn from_file(path: &str) -> Result<Self, ConfigError> {
        let content = std::fs::read_to_string(path)
            .map_err(|e| ConfigError::Io(format!("failed to read {path}: {e}")))?;
        Self::from_yaml(&content)
    }
}
