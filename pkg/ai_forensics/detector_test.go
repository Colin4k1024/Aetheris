// Copyright 2026 fanjia1024

package ai_forensics

import (
	"context"
	"testing"
	"time"
)

type fakeSignalSource struct {
	data map[string][]DecisionSignal
}

func (f *fakeSignalSource) ListDecisionSignals(ctx context.Context, jobID string) ([]DecisionSignal, error) {
	return append([]DecisionSignal(nil), f.data[jobID]...), nil
}

// TestAnomalyDetector 测试异常检测器
func TestAnomalyDetector(t *testing.T) {
	detector := NewAnomalyDetector(0.8).
		WithSignalSource(&fakeSignalSource{
			data: map[string][]DecisionSignal{
				"job_123": {
					{
						StepID:        "step_1",
						EvidenceCount: 0,
						Consistent:    false,
						Duration:      31 * time.Minute,
						Confidence:    0.2,
					},
				},
			},
		}).
		WithMaxStepDuration(10 * time.Minute)

	anomalies, err := detector.DetectAnomalies(context.Background(), "job_123")
	if err != nil {
		t.Fatalf("detect anomalies failed: %v", err)
	}

	if anomalies == nil {
		t.Fatal("anomalies should not be nil")
	}
	if len(anomalies) < 4 {
		t.Fatalf("expected multiple anomalies, got %d", len(anomalies))
	}
}

func TestAnomalyDetector_RetryLoopAndTamperedReasoning(t *testing.T) {
	detector := NewAnomalyDetector(0.8).WithSignalSource(&fakeSignalSource{
		data: map[string][]DecisionSignal{
			"job_eval": {
				{StepID: "retry.step", EvidenceCount: 1, Consistent: true, Confidence: 0.95, RetryCount: 4},
				{StepID: "tampered.step", EvidenceCount: 1, Consistent: true, Confidence: 0.95, TamperedReasoning: true},
			},
		},
	})

	anomalies, err := detector.DetectAnomalies(context.Background(), "job_eval")
	if err != nil {
		t.Fatalf("detect anomalies failed: %v", err)
	}

	seen := map[AnomalyType]string{}
	for _, anomaly := range anomalies {
		seen[anomaly.Type] = anomaly.Severity
	}
	if seen[AnomalySuspiciousRetryLoop] != "high" {
		t.Fatalf("retry loop severity = %q, want high", seen[AnomalySuspiciousRetryLoop])
	}
	if seen[AnomalyTamperedReasoning] != "critical" {
		t.Fatalf("tampered reasoning severity = %q, want critical", seen[AnomalyTamperedReasoning])
	}
}

func TestGoldenEvalCases_PassWithinFalsePositiveBudget(t *testing.T) {
	report, err := EvaluateGoldenCases(context.Background(), 0.8, 10*time.Minute, nil)
	if err != nil {
		t.Fatalf("evaluate golden cases: %v", err)
	}
	if !report.Passed {
		t.Fatalf("golden eval should pass: %+v", report)
	}
	if report.FalsePositiveCount != 0 {
		t.Fatalf("false positives = %d, want 0", report.FalsePositiveCount)
	}
	if report.FalseNegativeCount != 0 {
		t.Fatalf("false negatives = %d, want 0", report.FalseNegativeCount)
	}
	if report.SeverityMismatchCnt != 0 {
		t.Fatalf("severity mismatches = %d, want 0", report.SeverityMismatchCnt)
	}
}

// TestPatternMatcher 测试模式匹配
func TestPatternMatcher(t *testing.T) {
	matcher := NewPatternMatcher().WithSignalSource(&fakeSignalSource{
		data: map[string][]DecisionSignal{
			"job_1": {
				{StepID: "s1", Consistent: false, Duration: 20 * time.Minute, Failed: true, BypassedApproval: true},
				{StepID: "s2", Consistent: true, Failed: true},
			},
			"job_2": {
				{StepID: "s1", Consistent: false, Duration: 25 * time.Minute},
			},
		},
	})

	patterns, err := matcher.FindSuspiciousPatterns(context.Background(), []string{"job_1", "job_2"})
	if err != nil {
		t.Fatalf("find patterns failed: %v", err)
	}

	if patterns == nil {
		t.Fatal("patterns should not be nil")
	}
	if len(patterns) == 0 {
		t.Fatal("expected at least one suspicious pattern")
	}
}
