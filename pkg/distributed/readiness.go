// Copyright 2026 fanjia1024
// Promotion readiness gate for the distributed verifier prototype.

package distributed

// PromotionEvidence is the minimum evidence required before distributed
// verification can be promoted beyond prototype.
type PromotionEvidence struct {
	SingleNodeSaturationObserved bool
	LeaseFailureModesCovered     bool
	RecoveryFailureModesCovered  bool
	RootHashDrillCovered         bool
}

// ReadinessAssessment explains whether the distributed verifier should be
// promoted. It intentionally keeps the default conservative.
type ReadinessAssessment struct {
	Ready    bool     `json:"ready"`
	Decision string   `json:"decision"`
	Missing  []string `json:"missing,omitempty"`
}

// AssessPromotionReadiness evaluates the distributed verifier promotion gate.
func AssessPromotionReadiness(e PromotionEvidence) ReadinessAssessment {
	assessment := ReadinessAssessment{
		Ready:    true,
		Decision: "distributed verifier has enough operational evidence to leave prototype",
	}

	if !e.SingleNodeSaturationObserved {
		assessment.Missing = append(assessment.Missing, "single-node saturation or verifier bottleneck evidence")
	}
	if !e.LeaseFailureModesCovered {
		assessment.Missing = append(assessment.Missing, "lease failure-mode tests")
	}
	if !e.RecoveryFailureModesCovered {
		assessment.Missing = append(assessment.Missing, "recovery failure-mode tests")
	}
	if !e.RootHashDrillCovered {
		assessment.Missing = append(assessment.Missing, "multi-org root hash release drill")
	}

	if len(assessment.Missing) > 0 {
		assessment.Ready = false
		assessment.Decision = "keep distributed verifier as prototype"
	}
	return assessment
}
