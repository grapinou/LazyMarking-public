package tools

import (
	"context"
	"fmt"
	"time"

	"github.com/grapinou/LazyMarking/internal/db"
)

func MarkingFailed(userID, jobDBID int64, ctx context.Context, queries *db.Queries) error {
	params := db.FailMarkingJobParams{
		ID:     jobDBID,
		UserID: userID,
	}
	firstErr := queries.FailMarkingJob(ctx, params)
	if firstErr == nil {
		return nil
	}
	if ctx.Err() == nil {
		return fmt.Errorf("update marking job %d status to failed: %w", jobDBID, firstErr)
	}

	fallbackCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if fallbackErr := queries.FailMarkingJob(fallbackCtx, params); fallbackErr != nil {
		return fmt.Errorf(
			"update marking job %d status to failed with canceled context (%v), then with fallback context: %w",
			jobDBID,
			firstErr,
			fallbackErr,
		)
	}
	return nil
}
