package db

import (
	"database/sql"
	"fmt"
)

type MarkingReviewStatus string

const (
	MarkingReviewUnavailable    MarkingReviewStatus = "legacy_unavailable"
	MarkingReviewNoReviewNeeded MarkingReviewStatus = "no_review_needed"
	MarkingReviewPending        MarkingReviewStatus = "pending"
	MarkingReviewCompleted      MarkingReviewStatus = "completed"
)

// DeriveMarkingReviewStatus keeps review lifecycle orthogonal to the marking
// job's running/success/failed lifecycle. A NULL delta is legacy, never zero.
func DeriveMarkingReviewStatus(ambiguityDelta sql.NullFloat64, totalCandidates, pendingCandidates int64) (MarkingReviewStatus, error) {
	if totalCandidates < 0 || pendingCandidates < 0 || pendingCandidates > totalCandidates {
		return "", fmt.Errorf("invalid marking review summary")
	}
	if !ambiguityDelta.Valid {
		if totalCandidates != 0 || pendingCandidates != 0 {
			return "", fmt.Errorf("legacy marking review summary contains candidates")
		}
		return MarkingReviewUnavailable, nil
	}
	if totalCandidates == 0 {
		return MarkingReviewNoReviewNeeded, nil
	}
	if pendingCandidates > 0 {
		return MarkingReviewPending, nil
	}
	return MarkingReviewCompleted, nil
}
