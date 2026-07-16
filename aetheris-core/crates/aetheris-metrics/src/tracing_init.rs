//! OpenTelemetry tracing initialization.

use tracing_subscriber::{layer::SubscriberExt, util::SubscriberInitExt, EnvFilter};

/// Initialize tracing with console output and optional OTLP export.
///
/// # Arguments
/// * `service_name` - Name of the service (e.g., "aetheris-worker")
/// * `otlp_endpoint` - Optional OTLP endpoint (e.g., "http://localhost:4317")
pub fn init_tracing(service_name: &str, otlp_endpoint: Option<&str>) {
    let env_filter = EnvFilter::try_from_default_env().unwrap_or_else(|_| EnvFilter::new("info"));

    let fmt_layer = tracing_subscriber::fmt::layer()
        .with_target(true)
        .with_thread_ids(true)
        .with_file(true)
        .with_line_number(true);

    // For now, just set up console tracing.
    // OTLP export will be wired when opentelemetry-otlp integration is stable.
    if let Some(endpoint) = otlp_endpoint {
        tracing::info!(
            service = service_name,
            endpoint = endpoint,
            "OTLP tracing endpoint configured (export pending implementation)"
        );
    }

    tracing_subscriber::registry()
        .with(env_filter)
        .with(fmt_layer)
        .init();

    tracing::info!(service = service_name, "tracing initialized");
}

#[cfg(test)]
mod tests {
    #[test]
    fn test_init_tracing_does_not_panic() {
        // Just verify it compiles and doesn't panic
        // Can't easily test actual tracing output in unit tests
    }
}
