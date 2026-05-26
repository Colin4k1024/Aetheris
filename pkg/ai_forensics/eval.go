// Copyright 2026 fanjia1024
// Golden eval harness for AI forensics detector promotion.

package ai_forensics

import (
	"context"
	"fmt"
	"time"
)

const (
	GoldenEvalVersion        = "ai-forensics-golden-2026.05"
	DefaultFalsePositiveRate = 0.10
)

// GoldenCase is a deterministic detector eval case.
type GoldenCase struct {
	Name                   string
	JobID                  string
	Signals                []DecisionSignal
	Expected               []ExpectedAnomaly
	MaxFalsePositiveCount  int
	MaxFalsePositiveRate   float64
	ExpectedSeverityPolicy map[AnomalyType]string
}

// ExpectedAnomaly defines the expected anomaly type and severity for a step.
type ExpectedAnomaly struct {
	StepID   string
	Type     AnomalyType
	Severity string
}

// EvalReport summarizes the golden eval run.
type EvalReport struct {
	Version             string           `json:"version"`
	Passed              bool             `json:"passed"`
	TotalCases          int              `json:"total_cases"`
	PassedCases         int              `json:"passed_cases"`
	FalsePositiveCount  int              `json:"false_positive_count"`
	FalseNegativeCount  int              `json:"false_negative_count"`
	SeverityMismatchCnt int              `json:"severity_mismatch_count"`
	CaseResults         []EvalCaseResult `json:"case_results"`
}

// EvalCaseResult summarizes one golden case.
type EvalCaseResult struct {
	Name                   string   `json:"name"`
	Passed                 bool     `json:"passed"`
	ExpectedCount          int      `json:"expected_count"`
	ActualCount            int      `json:"actual_count"`
	FalsePositiveCount     int      `json:"false_positive_count"`
	FalseNegativeCount     int      `json:"false_negative_count"`
	SeverityMismatchCount  int      `json:"severity_mismatch_count"`
	FalsePositiveBudget    int      `json:"false_positive_budget"`
	FalsePositiveRateLimit float64  `json:"false_positive_rate_limit"`
	Errors                 []string `json:"errors,omitempty"`
}

// GoldenEvalCases returns the promotion dataset for AI forensics detection.
func GoldenEvalCases() []GoldenCase {
	return []GoldenCase{
		{
			Name:  "clean_execution_has_no_false_positive",
			JobID: "eval_clean",
			Signals: []DecisionSignal{
				{StepID: "clean.step", EvidenceCount: 2, Consistent: true, Duration: 2 * time.Minute, Confidence: 0.96},
			},
			MaxFalsePositiveCount: 0,
			MaxFalsePositiveRate:  0,
		},
		{
			Name:  "missing_evidence_is_high",
			JobID: "eval_missing_evidence",
			Signals: []DecisionSignal{
				{StepID: "missing.step", EvidenceCount: 0, Consistent: true, Duration: time.Minute, Confidence: 0.91},
			},
			Expected: []ExpectedAnomaly{
				{StepID: "missing.step", Type: AnomalyMissingEvidence, Severity: "high"},
			},
			MaxFalsePositiveCount: 0,
			MaxFalsePositiveRate:  DefaultFalsePositiveRate,
		},
		{
			Name:  "suspicious_retry_loop_is_high",
			JobID: "eval_retry_loop",
			Signals: []DecisionSignal{
				{StepID: "retry.step", EvidenceCount: 1, Consistent: true, Duration: time.Minute, Confidence: 0.93, RetryCount: 4},
			},
			Expected: []ExpectedAnomaly{
				{StepID: "retry.step", Type: AnomalySuspiciousRetryLoop, Severity: "high"},
			},
			MaxFalsePositiveCount: 0,
			MaxFalsePositiveRate:  DefaultFalsePositiveRate,
		},
		{
			Name:  "tampered_reasoning_is_critical",
			JobID: "eval_tampered_reasoning",
			Signals: []DecisionSignal{
				{StepID: "tampered.step", EvidenceCount: 1, Consistent: true, Duration: time.Minute, Confidence: 0.94, TamperedReasoning: true},
			},
			Expected: []ExpectedAnomaly{
				{StepID: "tampered.step", Type: AnomalyTamperedReasoning, Severity: "critical"},
			},
			MaxFalsePositiveCount: 0,
			MaxFalsePositiveRate:  DefaultFalsePositiveRate,
		},
	}
}

// EvaluateGoldenCases runs the deterministic golden eval suite.
func EvaluateGoldenCases(ctx context.Context, threshold float64, maxStepDuration time.Duration, cases []GoldenCase) (EvalReport, error) {
	if len(cases) == 0 {
		cases = GoldenEvalCases()
	}
	source := &goldenSignalSource{data: map[string][]DecisionSignal{}}
	for _, c := range cases {
		source.data[c.JobID] = append([]DecisionSignal(nil), c.Signals...)
	}

	detector := NewAnomalyDetector(threshold).WithSignalSource(source)
	if maxStepDuration > 0 {
		detector = detector.WithMaxStepDuration(maxStepDuration)
	}

	report := EvalReport{
		Version:     GoldenEvalVersion,
		Passed:      true,
		TotalCases:  len(cases),
		CaseResults: make([]EvalCaseResult, 0, len(cases)),
	}

	for _, c := range cases {
		anomalies, err := detector.DetectAnomalies(ctx, c.JobID)
		if err != nil {
			return report, err
		}
		result := evaluateCase(c, anomalies)
		if result.Passed {
			report.PassedCases++
		} else {
			report.Passed = false
		}
		report.FalsePositiveCount += result.FalsePositiveCount
		report.FalseNegativeCount += result.FalseNegativeCount
		report.SeverityMismatchCnt += result.SeverityMismatchCount
		report.CaseResults = append(report.CaseResults, result)
	}

	return report, nil
}

func evaluateCase(c GoldenCase, anomalies []Anomaly) EvalCaseResult {
	budget := c.MaxFalsePositiveCount
	rateLimit := c.MaxFalsePositiveRate
	if rateLimit < 0 {
		rateLimit = 0
	}

	result := EvalCaseResult{
		Name:                   c.Name,
		ExpectedCount:          len(c.Expected),
		ActualCount:            len(anomalies),
		FalsePositiveBudget:    budget,
		FalsePositiveRateLimit: rateLimit,
	}

	expected := map[string]ExpectedAnomaly{}
	for _, e := range c.Expected {
		expected[anomalyKey(e.StepID, e.Type)] = e
	}
	actual := map[string]Anomaly{}
	for _, a := range anomalies {
		key := anomalyKey(a.StepID, a.Type)
		actual[key] = a
		if e, ok := expected[key]; !ok {
			result.FalsePositiveCount++
			result.Errors = append(result.Errors, fmt.Sprintf("unexpected anomaly %s on %s", a.Type, a.StepID))
		} else if e.Severity != "" && e.Severity != a.Severity {
			result.SeverityMismatchCount++
			result.Errors = append(result.Errors, fmt.Sprintf("severity mismatch for %s on %s: got %s want %s", a.Type, a.StepID, a.Severity, e.Severity))
		}
	}
	for key, e := range expected {
		if _, ok := actual[key]; !ok {
			result.FalseNegativeCount++
			result.Errors = append(result.Errors, fmt.Sprintf("missing anomaly %s on %s", e.Type, e.StepID))
		}
	}

	falsePositiveRate := 0.0
	if len(anomalies) > 0 {
		falsePositiveRate = float64(result.FalsePositiveCount) / float64(len(anomalies))
	}
	result.Passed = result.FalseNegativeCount == 0 &&
		result.SeverityMismatchCount == 0 &&
		result.FalsePositiveCount <= budget &&
		falsePositiveRate <= rateLimit
	return result
}

func anomalyKey(stepID string, typ AnomalyType) string {
	return stepID + "\x00" + string(typ)
}

type goldenSignalSource struct {
	data map[string][]DecisionSignal
}

func (g *goldenSignalSource) ListDecisionSignals(ctx context.Context, jobID string) ([]DecisionSignal, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return append([]DecisionSignal(nil), g.data[jobID]...), nil
}
