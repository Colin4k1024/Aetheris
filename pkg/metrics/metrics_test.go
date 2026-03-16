// Copyright 2026 fanjia1024
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package metrics

import (
	"bytes"
	"strings"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
)

func TestWritePrometheus(t *testing.T) {
	// Test that DefaultRegistry is properly initialized
	if DefaultRegistry == nil {
		t.Fatal("DefaultRegistry should not be nil")
	}

	// Write to buffer using DefaultRegistry
	var buf bytes.Buffer
	err := WritePrometheus(&buf)
	if err != nil {
		t.Errorf("WritePrometheus failed: %v", err)
	}

	output := buf.String()
	// Should contain at least some metrics
	if output == "" {
		t.Error("expected non-empty output")
	}
}

func TestJobDurationHistogram(t *testing.T) {
	histogram := prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "test_job_duration_seconds",
			Help:    "Test job duration",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"agent_id"},
	)

	reg := prometheus.NewRegistry()
	reg.MustRegister(histogram)

	histogram.WithLabelValues("agent-1").Observe(1.0)
	histogram.WithLabelValues("agent-2").Observe(2.0)

	// Gather and check we have metrics
	metrics, err := reg.Gather()
	if err != nil {
		t.Fatalf("Gather failed: %v", err)
	}
	if len(metrics) != 1 {
		t.Errorf("expected 1 metric family, got %d", len(metrics))
	}
}

func TestJobTotalCounter(t *testing.T) {
	counter := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "test_job_total",
			Help: "Test job total",
		},
		[]string{"status"},
	)

	reg := prometheus.NewRegistry()
	reg.MustRegister(counter)

	counter.WithLabelValues("completed").Inc()
	counter.WithLabelValues("failed").Inc()

	// Gather and check
	metrics, err := reg.Gather()
	if err != nil {
		t.Fatalf("Gather failed: %v", err)
	}
	if len(metrics) != 1 {
		t.Errorf("expected 1 metric family, got %d", len(metrics))
	}
}

func TestWorkerBusyGauge(t *testing.T) {
	gauge := prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "test_worker_busy",
			Help: "Test worker busy",
		},
		[]string{"worker_id"},
	)

	reg := prometheus.NewRegistry()
	reg.MustRegister(gauge)

	gauge.WithLabelValues("worker-1").Set(5.0)
	gauge.WithLabelValues("worker-2").Set(10.0)

	// Gather and check
	metrics, err := reg.Gather()
	if err != nil {
		t.Fatalf("Gather failed: %v", err)
	}
	if len(metrics) != 1 {
		t.Errorf("expected 1 metric family, got %d", len(metrics))
	}
}

func TestMetricsPrometheusRegistry(t *testing.T) {
	// Test that DefaultRegistry contains registered metric families
	// First add some observations to ensure metrics are visible
	JobDuration.WithLabelValues("test").Observe(1.0)
	JobTotal.WithLabelValues("test").Inc()

	metrics, err := DefaultRegistry.Gather()
	if err != nil {
		t.Fatalf("Gather failed: %v", err)
	}

	// Should have metrics registered (after observations)
	if len(metrics) == 0 {
		t.Error("expected metrics to be registered")
	}

	// Check we have at least some metrics registered
	if len(metrics) < 3 {
		t.Errorf("expected at least 3 metrics, got %d", len(metrics))
	}

	// Log available metric names for debugging
	t.Logf("Found %d metric families", len(metrics))
	for _, m := range metrics {
		t.Logf("  - %s", m.GetName())
	}
}

func TestWritePrometheusWithObservations(t *testing.T) {
	// Create observations on metrics
	JobDuration.WithLabelValues("test-agent").Observe(1.5)
	JobTotal.WithLabelValues("completed").Inc()
	ToolDuration.WithLabelValues("calculator").Observe(0.5)

	// Write to buffer
	var buf bytes.Buffer
	err := WritePrometheus(&buf)
	if err != nil {
		t.Errorf("WritePrometheus failed: %v", err)
	}

	output := buf.String()

	// Check that our observed values appear in output
	if !strings.Contains(output, "test-agent") {
		t.Error("expected test-agent label in output")
	}
	if !strings.Contains(output, "completed") {
		t.Error("expected completed label in output")
	}
	if !strings.Contains(output, "calculator") {
		t.Error("expected calculator label in output")
	}
}
