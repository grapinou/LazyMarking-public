package generateexams

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/grapinou/LazyMarking/internal/db"
	"github.com/grapinou/LazyMarking/internal/handlers/tools"
)

func completeExamGeneration(userID, examGeneratedID int64, ctx context.Context, queries *db.Queries) error {
	rows, transitionErr := queries.CompleteExamGeneration(ctx, db.CompleteExamGenerationParams{
		ID:     examGeneratedID,
		UserID: userID,
	})
	if transitionErr == nil && rows == 1 {
		return nil
	}
	if transitionErr == nil && rows > 1 {
		return fmt.Errorf("complete exam generation %d: affected %d rows", examGeneratedID, rows)
	}

	status, statusErr := getExamGenerationStatusForInterpretation(ctx, userID, examGeneratedID, queries)
	if statusErr != nil {
		if transitionErr != nil {
			return fmt.Errorf("complete exam generation %d: transition error: %v; resolve status: %w", examGeneratedID, transitionErr, statusErr)
		}
		return fmt.Errorf("complete exam generation %d affected %d rows; resolve status: %w", examGeneratedID, rows, statusErr)
	}
	if status == "success" {
		return nil
	}
	if status == "failed" {
		return fmt.Errorf("complete exam generation %d: terminal conflict, status is failed", examGeneratedID)
	}
	if transitionErr != nil {
		return fmt.Errorf("complete exam generation %d while status is %s: %w", examGeneratedID, status, transitionErr)
	}
	return fmt.Errorf("complete exam generation %d affected %d rows while status is %s", examGeneratedID, rows, status)
}

func completeExamGenerationWithReferences(userID int64, username string, examGeneratedID int64, ctx context.Context, queries *db.Queries) error {
	if err := tools.ValidateExamGenerationReferences(ctx, queries, userID, username, examGeneratedID); err != nil {
		return fmt.Errorf("validate references for exam generation %d: %w", examGeneratedID, err)
	}
	rows, transitionErr := queries.CompleteExamGenerationWithReferences(ctx, db.CompleteExamGenerationWithReferencesParams{
		ID:     examGeneratedID,
		UserID: userID,
	})
	if transitionErr == nil && rows == 1 {
		return nil
	}
	if transitionErr == nil && rows > 1 {
		return fmt.Errorf("complete exam generation %d with references: affected %d rows", examGeneratedID, rows)
	}
	status, statusErr := getExamGenerationStatusForInterpretation(ctx, userID, examGeneratedID, queries)
	if statusErr != nil {
		if transitionErr != nil {
			return fmt.Errorf("complete exam generation %d with references: transition error: %v; resolve status: %w", examGeneratedID, transitionErr, statusErr)
		}
		return fmt.Errorf("complete exam generation %d with references affected %d rows; resolve status: %w", examGeneratedID, rows, statusErr)
	}
	if status == "failed" || status == "success" {
		return fmt.Errorf("complete exam generation %d with references: terminal conflict, status is %s", examGeneratedID, status)
	}
	if transitionErr != nil {
		return fmt.Errorf("complete exam generation %d with references while status is %s: %w", examGeneratedID, status, transitionErr)
	}
	return fmt.Errorf("complete exam generation %d with references affected %d rows while status is %s", examGeneratedID, rows, status)
}

func getExamGenerationStatusForInterpretation(ctx context.Context, userID, examGeneratedID int64, queries *db.Queries) (string, error) {
	statusCtx := ctx
	cancel := func() {}
	if ctx.Err() != nil {
		statusCtx, cancel = context.WithTimeout(context.Background(), 3*time.Second)
	}
	defer cancel()

	status, err := queries.GetExamStatus(statusCtx, db.GetExamStatusParams{
		ID:     examGeneratedID,
		UserID: userID,
	})
	if errors.Is(err, sql.ErrNoRows) {
		return "", fmt.Errorf("exam generation %d not found", examGeneratedID)
	}
	if err != nil {
		return "", err
	}
	return status, nil
}
