//! Aetheris executor crate.
//!
//! Core execution engine: runs DAG nodes with step-level checkpointing,
//! deterministic replay, compensation for side effects, and crash recovery.

mod checkpoint;
mod compensation;
mod error;
mod replay;
mod runner;
mod step;
mod types;

pub use checkpoint::CheckpointManager;
pub use compensation::{CompensationAction, CompensationRegistry};
pub use error::ExecutorError;
pub use replay::ReplayEngine;
pub use runner::{Runner, RunnerConfig, RunnerEvent};
pub use step::{StepExecutor, StepOutcome};
pub use types::*;
