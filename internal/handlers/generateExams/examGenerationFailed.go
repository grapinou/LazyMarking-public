package generateexams

import (
	"context"
	"fmt"
	"time"

	"github.com/grapinou/LazyMarking/internal/db"
)

func failExamGeneration(userID, examGeneratedID int64, ctx context.Context, queries *db.Queries) error {
	params := db.FailExamGenerationParams{
		ID:     examGeneratedID,
		UserID: userID,
	}

	rows, firstErr := queries.FailExamGeneration(ctx, params)
	if firstErr == nil && rows == 1 {
		return nil
	}
	if firstErr == nil && rows > 1 {
		return fmt.Errorf("update exam generation %d status to failed: affected %d rows", examGeneratedID, rows)
	}
	if firstErr == nil {
		return interpretFailedExamGeneration(ctx, userID, examGeneratedID, rows, queries)
	}
	if ctx.Err() == nil {
		return fmt.Errorf("update exam generation %d status to failed: %w", examGeneratedID, firstErr)
	}

	fallbackCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	status, statusErr := getExamGenerationStatusForInterpretation(fallbackCtx, userID, examGeneratedID, queries)
	if statusErr != nil {
		return fmt.Errorf(
			"update exam generation %d status to failed with canceled context (%v), then resolve status: %w",
			examGeneratedID,
			firstErr,
			statusErr,
		)
	}
	if status == "failed" {
		return nil
	}
	if status == "success" {
		return fmt.Errorf("update exam generation %d status to failed: terminal conflict, status is success", examGeneratedID)
	}

	rows, fallbackErr := queries.FailExamGeneration(fallbackCtx, params)
	if fallbackErr != nil {
		return fmt.Errorf(
			"update exam generation %d status to failed with canceled context (%v), then with fallback context: %w",
			examGeneratedID,
			firstErr,
			fallbackErr,
		)
	}
	if rows == 1 {
		return nil
	}
	if rows > 1 {
		return fmt.Errorf("update exam generation %d status to failed with fallback: affected %d rows", examGeneratedID, rows)
	}
	return interpretFailedExamGeneration(fallbackCtx, userID, examGeneratedID, rows, queries)
}

func interpretFailedExamGeneration(ctx context.Context, userID, examGeneratedID, rows int64, queries *db.Queries) error {
	status, err := getExamGenerationStatusForInterpretation(ctx, userID, examGeneratedID, queries)
	if err != nil {
		return fmt.Errorf("update exam generation %d status to failed affected %d rows: %w", examGeneratedID, rows, err)
	}
	if status == "failed" {
		return nil
	}
	if status == "success" {
		return fmt.Errorf("update exam generation %d status to failed: terminal conflict, status is success", examGeneratedID)
	}
	return fmt.Errorf("update exam generation %d status to failed affected %d rows while status is %s", examGeneratedID, rows, status)
}
