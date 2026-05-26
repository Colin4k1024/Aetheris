// Copyright 2026 fanjia1024

package monitoring

import (
	"context"
	"testing"
)

type fakeDecisionInputSource struct {
	input DecisionInput
	err   error
}

func (f *fakeDecisionInputSource) GetDecisionInput(ctx context.Context, stepID string) (DecisionInput, error) {
	if f.err != nil {
		return DecisionInput{}, f.err
	}
	return f.input, nil
}

// TestQualityScorer 测试质量评分
func TestQualityScorer(t *testing.T) {
	scorer := NewQualityScorer()

	score, err := scorer.ScoreDecision(context.Background(), "step_123")
	if err != nil {
		t.Fatalf("score decision failed: %v", err)
	}

	if score.Overall < 0 || score.Overall > 100 {
		t.Errorf("overall score should be 0-100, got %f", score.Overall)
	}
}

func TestQualityScorer_WithInputSource(t *testing.T) {
	scorer := NewQualityScorer().WithInputSource(&fakeDecisionInputSource{
		input: DecisionInput{
			EvidenceCompleteness: 55,
			EvidenceQuality:      60,
			Confidence:           58,
			HumanOversight:       40,
		},
	})
	score, err := scorer.ScoreDecision(context.Background(), "step_abc")
	if err != nil {
		t.Fatalf("score decision failed: %v", err)
	}
	if score.Overall >= 70 {
		t.Fatalf("expected low overall score, got %f", score.Overall)
	}
	if len(score.Recommendations) == 0 {
		t.Fatal("expected recommendations for low-quality decision")
	}
}

func TestAssessQualityScore_Healthy(t *testing.T) {
	assessment := AssessQualityScore(&QualityScore{
		Overall:              88,
		EvidenceCompleteness: 90,
		EvidenceQuality:      86,
		Confidence:           85,
		HumanOversight:       82,
	})
	if assessment.Alert {
		t.Fatalf("healthy score should not alert: %+v", assessment)
	}
	if assessment.Level != AlertHealthy {
		t.Fatalf("level = %s, want healthy", assessment.Level)
	}
}

func TestAssessQualityScore_Degraded(t *testing.T) {
	score := &QualityScore{
		Overall:              64,
		EvidenceCompleteness: 65,
		EvidenceQuality:      68,
		Confidence:           72,
		HumanOversight:       61,
		Recommendations:      []string{"increase human oversight for high-risk decisions"},
	}
	assessment := AssessQualityScore(score)
	if !assessment.Alert {
		t.Fatal("degraded score should alert")
	}
	if assessment.Level != AlertDegraded {
		t.Fatalf("level = %s, want degraded", assessment.Level)
	}
}

func TestAssessQualityScore_Critical(t *testing.T) {
	assessment := AssessQualityScore(&QualityScore{
		Overall:              45,
		EvidenceCompleteness: 39,
		EvidenceQuality:      60,
		Confidence:           55,
		HumanOversight:       52,
	})
	if !assessment.Alert {
		t.Fatal("critical score should alert")
	}
	if assessment.Level != AlertCritical {
		t.Fatalf("level = %s, want critical", assessment.Level)
	}
}

func TestAssessQualityScore_NoisySignal(t *testing.T) {
	assessment := AssessQualityScore(&QualityScore{
		Overall:              66,
		EvidenceCompleteness: 90,
		EvidenceQuality:      78,
		Confidence:           42,
		HumanOversight:       85,
	})
	if !assessment.Alert {
		t.Fatal("noisy score should alert")
	}
	if assessment.Level != AlertNoisy {
		t.Fatalf("level = %s, want noisy", assessment.Level)
	}
}
