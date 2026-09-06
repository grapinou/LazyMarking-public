package db

import (
	"database/sql"
	"testing"
)

func TestDeriveMarkingReviewStatus(t *testing.T) {
	active := sql.NullFloat64{Float64: 5, Valid: true}
	for _, tc := range []struct {
		name           string
		delta          sql.NullFloat64
		total, pending int64
		want           MarkingReviewStatus
		wantError      bool
	}{
		{name: "legacy", want: MarkingReviewUnavailable},
		{name: "none", delta: active, want: MarkingReviewNoReviewNeeded},
		{name: "pending", delta: active, total: 2, pending: 1, want: MarkingReviewPending},
		{name: "completed", delta: active, total: 2, want: MarkingReviewCompleted},
		{name: "invalid legacy", total: 1, pending: 1, wantError: true},
		{name: "invalid counts", delta: active, total: 1, pending: 2, wantError: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := DeriveMarkingReviewStatus(tc.delta, tc.total, tc.pending)
			if (err != nil) != tc.wantError || got != tc.want {
				t.Fatalf("status=%q err=%v, want status=%q error=%v", got, err, tc.want, tc.wantError)
			}
		})
	}
}
