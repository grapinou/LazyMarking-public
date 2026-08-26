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
	updateFailed := func(updateCtx context.Context) error {
		rows, err := queries.UpdateExamGenerated(updateCtx, params)
		if err != nil {
			return err
		}
		if rows != 1 {
			return fmt.Errorf("affected %d rows", rows)
		}
		return nil
	}

	firstErr := updateFailed(ctx)
	if firstErr == nil {
		return nil
	}
	if ctx.Err() == nil {
		return fmt.Errorf("update exam generation %d status to failed: %w", examGeneratedID, firstErr)
	}

	fallbackCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if fallbackErr := updateFailed(fallbackCtx); fallbackErr != nil {
		return fmt.Errorf(
			"update exam generation %d status to failed with canceled context (%v), then with fallback context: %w",
			examGeneratedID,
			firstErr,
			fallbackErr,
		)
	}
	return nil
}
