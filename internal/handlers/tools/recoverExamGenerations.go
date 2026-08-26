package tools

import (
	"context"
	"fmt"
	"os"
	"strconv"

	"github.com/grapinou/LazyMarking/internal/db"
)

func RecoverRunningExamGenerations(ctx context.Context, queries *db.Queries) error {
	jobs, err := queries.ListRunningExamGenerations(ctx)
	if err != nil {
		return fmt.Errorf("list running exam generations: %w", err)
	}

	for _, job := range jobs {
		operation := "exam-" + strconv.FormatInt(job.ID, 10)
		tempDir, err := operationTempDir(job.Username, operation)
		if err != nil {
			return fmt.Errorf("resolve workspace for exam generation %d: %w", job.ID, err)
		}

		if err := os.RemoveAll(tempDir); err != nil {
			return fmt.Errorf("remove workspace for exam generation %d: %w", job.ID, err)
		}

		if err := queries.DeleteExamGenerated(ctx, db.DeleteExamGeneratedParams{
			ID:     job.ID,
			UserID: job.UserID,
		}); err != nil {
			return fmt.Errorf("delete interrupted exam generation %d: %w", job.ID, err)
		}
	}

	return nil
}
