package tools

import (
	"context"
	"fmt"
	"strconv"

	"github.com/grapinou/LazyMarking/internal/db"
)

func RecoverRunningExamGenerations(ctx context.Context, queries *db.Queries) error {
	jobs, err := queries.ListRunningExamGenerations(ctx)
	if err != nil {
		return fmt.Errorf("list running exam generations: %w", err)
	}

	for _, job := range jobs {
		if err := recoverRunningExamGeneration(ctx, queries, job); err != nil {
			return err
		}
	}

	return nil
}

func recoverRunningExamGeneration(ctx context.Context, queries *db.Queries, job db.ListRunningExamGenerationsRow) error {
	rows, err := queries.DeleteRunningExamGenerated(ctx, db.DeleteRunningExamGeneratedParams{
		ID:     job.ID,
		UserID: job.UserID,
	})
	if err != nil {
		return fmt.Errorf("delete interrupted exam generation %d: %w", job.ID, err)
	}
	if rows == 0 {
		return nil
	}
	if rows > 1 {
		return fmt.Errorf("delete interrupted exam generation %d: affected %d rows", job.ID, rows)
	}

	operation := "exam-" + strconv.FormatInt(job.ID, 10)
	if err := RemoveOperationTempDir(job.Username, operation); err != nil {
		return fmt.Errorf("remove workspace for exam generation %d: %w", job.ID, err)
	}
	return nil
}
