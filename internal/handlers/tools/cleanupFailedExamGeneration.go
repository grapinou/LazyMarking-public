package tools

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strconv"

	"github.com/grapinou/LazyMarking/internal/db"
)

var removeFailedExamGenerationWorkspace = RemoveOperationTempDir

// CleanupFailedExamGeneration deletes only a failed generation and its DB
// descendants, then removes its workspace. It is safe to call again after the
// generation or workspace has already been removed.
func CleanupFailedExamGeneration(ctx context.Context, queries *db.Queries, generationID, userID int64, username string) error {
	rows, err := queries.DeleteFailedExamGenerated(ctx, db.DeleteFailedExamGeneratedParams{
		ID:     generationID,
		UserID: userID,
	})
	if err != nil {
		return fmt.Errorf("delete failed exam generation %d: %w", generationID, err)
	}
	if rows > 1 {
		return fmt.Errorf("delete failed exam generation %d: affected %d rows", generationID, rows)
	}
	if rows == 0 {
		status, statusErr := queries.GetExamStatus(ctx, db.GetExamStatusParams{ID: generationID, UserID: userID})
		if statusErr != nil && !errors.Is(statusErr, sql.ErrNoRows) {
			return fmt.Errorf("resolve failed exam generation %d after zero-row delete: %w", generationID, statusErr)
		}
		if statusErr == nil {
			return fmt.Errorf("refuse cleanup of exam generation %d with status %s", generationID, status)
		}
	}

	operation := "exam-" + strconv.FormatInt(generationID, 10)
	if err := removeFailedExamGenerationWorkspace(username, operation); err != nil {
		return fmt.Errorf("remove workspace for failed exam generation %d: %w", generationID, err)
	}
	return nil
}
