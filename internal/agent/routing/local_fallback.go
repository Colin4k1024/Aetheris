package routing

import (
	"context"
	"time"
)

// LocalFallbackAdvisor is a deterministic local fallback for fail_open policy.
// It selects the first candidate and produces a recordable decision.
// RecordOutcome is a no-op — no external feedback channel exists.
type LocalFallbackAdvisor struct{}

// NewLocalFallbackAdvisor creates a LocalFallbackAdvisor.
func NewLocalFallbackAdvisor() *LocalFallbackAdvisor {
	return &LocalFallbackAdvisor{}
}

// Decide selects the first candidate from the request.
// Returns an error if no candidates are provided.
func (a *LocalFallbackAdvisor) Decide(ctx context.Context, req RouteDecisionRequest) (RouteDecision, error) {
	if len(req.Candidates) == 0 {
		return RouteDecision{}, ErrNoCandidates
	}

	selected := req.Candidates[0]

	decision := RouteDecision{
		SelectedID:  selected.ID,
		ReasonCodes: []string{"local_fallback_first_candidate"},
		Timestamp:   time.Now().UTC(),
	}

	decision.DecisionHash = ComputeDecisionHash(decision)
	return decision, nil
}

// RecordOutcome is a no-op for the local fallback advisor.
// Feedback is not collected since this advisor uses a fixed strategy.
func (a *LocalFallbackAdvisor) RecordOutcome(ctx context.Context, outcome RouteOutcome) error {
	return nil
}
