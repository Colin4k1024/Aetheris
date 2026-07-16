//! Aetheris planner crate.
//!
//! DAG compiler: converts a TaskGraph (from protobuf) into an executable
//! ExecutionPlan with topological ordering, parallel groups, and dependency tracking.

mod compiler;
mod error;
mod plan;
mod types;

pub use compiler::DagCompiler;
pub use error::PlannerError;
pub use plan::{ExecutionPlan, PlanNode, PlanNodeStatus};
pub use types::*;

/// Compile a TaskGraph into an ExecutionPlan.
pub fn compile(graph: &TaskGraph) -> Result<ExecutionPlan, PlannerError> {
    DagCompiler::new(graph).compile()
}
