//! Prometheus metrics counters for Aetheris core operations.

use prometheus::{Counter, CounterVec, Gauge, HistogramVec, Opts, Registry};

/// Core metrics for the Aetheris execution engine.
pub struct Metrics {
    pub registry: Registry,

    // Job metrics
    pub jobs_created: Counter,
    pub jobs_completed: Counter,
    pub jobs_failed: Counter,

    // Step metrics
    pub steps_executed: CounterVec,
    pub step_duration: HistogramVec,

    // Event store metrics
    pub events_appended: Counter,
    pub claims_made: Counter,
    pub heartbeats: Counter,

    // Worker metrics
    pub active_workers: Gauge,
    pub active_jobs: Gauge,

    // Memory metrics
    pub memory_stores: CounterVec,
    pub memory_searches: CounterVec,

    // Error metrics
    pub errors: CounterVec,
}

impl Metrics {
    pub fn new() -> Self {
        let registry = Registry::new();

        let jobs_created = Counter::with_opts(Opts::new(
            "aetheris_jobs_created_total",
            "Total jobs created",
        ))
        .unwrap();

        let jobs_completed = Counter::with_opts(Opts::new(
            "aetheris_jobs_completed_total",
            "Total jobs completed successfully",
        ))
        .unwrap();

        let jobs_failed = Counter::with_opts(Opts::new(
            "aetheris_jobs_failed_total",
            "Total jobs failed",
        ))
        .unwrap();

        let steps_executed = CounterVec::new(
            Opts::new(
                "aetheris_steps_executed_total",
                "Total steps executed by result type",
            ),
            &["result_type"],
        )
        .unwrap();

        let step_duration = HistogramVec::new(
            prometheus::HistogramOpts::new(
                "aetheris_step_duration_seconds",
                "Step execution duration in seconds",
            )
            .buckets(vec![0.001, 0.005, 0.01, 0.05, 0.1, 0.5, 1.0, 5.0, 10.0, 30.0]),
            &["node_type"],
        )
        .unwrap();

        let events_appended = Counter::with_opts(Opts::new(
            "aetheris_events_appended_total",
            "Total events appended to event store",
        ))
        .unwrap();

        let claims_made = Counter::with_opts(Opts::new(
            "aetheris_claims_made_total",
            "Total job claims made",
        ))
        .unwrap();

        let heartbeats = Counter::with_opts(Opts::new(
            "aetheris_heartbeats_total",
            "Total heartbeats sent",
        ))
        .unwrap();

        let active_workers = Gauge::with_opts(Opts::new(
            "aetheris_active_workers",
            "Number of active workers",
        ))
        .unwrap();

        let active_jobs = Gauge::with_opts(Opts::new(
            "aetheris_active_jobs",
            "Number of currently running jobs",
        ))
        .unwrap();

        let memory_stores = CounterVec::new(
            Opts::new(
                "aetheris_memory_stores_total",
                "Total memory store operations",
            ),
            &["namespace"],
        )
        .unwrap();

        let memory_searches = CounterVec::new(
            Opts::new(
                "aetheris_memory_searches_total",
                "Total memory search operations",
            ),
            &["namespace"],
        )
        .unwrap();

        let errors = CounterVec::new(
            Opts::new("aetheris_errors_total", "Total errors by category"),
            &["category"],
        )
        .unwrap();

        // Register all metrics
        registry.register(Box::new(jobs_created.clone())).unwrap();
        registry.register(Box::new(jobs_completed.clone())).unwrap();
        registry.register(Box::new(jobs_failed.clone())).unwrap();
        registry.register(Box::new(steps_executed.clone())).unwrap();
        registry.register(Box::new(step_duration.clone())).unwrap();
        registry.register(Box::new(events_appended.clone())).unwrap();
        registry.register(Box::new(claims_made.clone())).unwrap();
        registry.register(Box::new(heartbeats.clone())).unwrap();
        registry.register(Box::new(active_workers.clone())).unwrap();
        registry.register(Box::new(active_jobs.clone())).unwrap();
        registry.register(Box::new(memory_stores.clone())).unwrap();
        registry.register(Box::new(memory_searches.clone())).unwrap();
        registry.register(Box::new(errors.clone())).unwrap();

        Self {
            registry,
            jobs_created,
            jobs_completed,
            jobs_failed,
            steps_executed,
            step_duration,
            events_appended,
            claims_made,
            heartbeats,
            active_workers,
            active_jobs,
            memory_stores,
            memory_searches,
            errors,
        }
    }

    /// Record a step execution result.
    pub fn record_step(&self, result_type: &str, node_type: &str, duration_secs: f64) {
        self.steps_executed
            .with_label_values(&[result_type])
            .inc();
        self.step_duration
            .with_label_values(&[node_type])
            .observe(duration_secs);
    }

    /// Export all metrics as Prometheus text format.
    pub fn export(&self) -> String {
        use prometheus::Encoder;
        let encoder = prometheus::TextEncoder::new();
        let metric_families = self.registry.gather();
        let mut buffer = Vec::new();
        encoder.encode(&metric_families, &mut buffer).unwrap();
        String::from_utf8(buffer).unwrap_or_default()
    }
}

impl Default for Metrics {
    fn default() -> Self {
        Self::new()
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_metrics_creation() {
        let metrics = Metrics::new();
        metrics.jobs_created.inc();
        metrics.jobs_completed.inc();
        metrics.jobs_failed.inc();
        metrics.events_appended.inc_by(5.0);

        let output = metrics.export();
        assert!(output.contains("aetheris_jobs_created_total 1"));
        assert!(output.contains("aetheris_jobs_completed_total 1"));
        assert!(output.contains("aetheris_jobs_failed_total 1"));
        assert!(output.contains("aetheris_events_appended_total 5"));
    }

    #[test]
    fn test_step_recording() {
        let metrics = Metrics::new();
        metrics.record_step("success", "tool_call", 0.05);
        metrics.record_step("success", "tool_call", 0.1);
        metrics.record_step("retryable_failure", "llm_call", 1.5);

        let output = metrics.export();
        assert!(output.contains("aetheris_steps_executed_total"));
        assert!(output.contains("aetheris_step_duration_seconds"));
    }
}
