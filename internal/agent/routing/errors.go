package routing

import "errors"

var (
	// ErrNoCandidates is returned when a RouteDecisionRequest has an empty Candidates list.
	ErrNoCandidates = errors.New("routing: no candidates provided")

	// ErrAdvisorUnavailable is returned when the external advisor cannot be reached.
	ErrAdvisorUnavailable = errors.New("routing: advisor unavailable")

	// ErrHashMismatch is returned when a persisted decision hash fails verification.
	ErrHashMismatch = errors.New("routing: decision hash mismatch")

	// ErrDecisionMissing is returned during replay when no recorded decision exists.
	ErrDecisionMissing = errors.New("routing: recorded decision missing")
)
