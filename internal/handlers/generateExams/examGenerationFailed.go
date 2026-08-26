package generateexams

import (
	"context"
	"fmt"
	"time"

	"github.com/grapinou/LazyMarking/internal/db"
)

func failExamGeneration(userID, examGeneratedID int64, ctx context.Context, queries *db.Queries) error {
	params := db.UpdateExamGeneratedParams{
		Status: "failed",
		ID:     examGeneratedID,
		UserID: userID,
	}
	firstErr := queries.UpdateExamGenerated(ctx, params)
	if firstErr == nil {
		return nil
	}
	if ctx.Err() == nil {
		return fmt.Errorf("update exam generation %d status to failed: %w", examGeneratedID, firstErr)
	}

	fallbackCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if fallbackErr := queries.UpdateExamGenerated(fallbackCtx, params); fallbackErr != nil {
		return fmt.Errorf(
			"update exam generation %d status to failed with canceled context (%v), then with fallback context: %w",
			examGeneratedID,
			firstErr,
			fallbackErr,
		)
	}
	return nil
}
