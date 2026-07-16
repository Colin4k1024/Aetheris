//! Planner error types.

#[derive(Debug, thiserror::Error)]
pub enum PlannerError {
    #[error("graph has no nodes")]
    EmptyGraph,

    #[error("node not found: {0}")]
    NodeNotFound(String),

    #[error("cycle detected involving node: {0}")]
    CycleDetected(String),

    #[error("missing entry node: {0}")]
    MissingEntryNode(String),

    #[error("invalid edge: from={from}, to={to}")]
    InvalidEdge { from: String, to: String },

    #[error("invalid graph: {0}")]
    InvalidGraph(String),
}
