//! Aetheris metrics crate.
//!
//! Prometheus metrics and OpenTelemetry tracing for the execution core.

mod counters;
mod tracing_init;

pub use counters::Metrics;
pub use tracing_init::init_tracing;
