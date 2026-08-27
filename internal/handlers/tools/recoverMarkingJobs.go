package tools

import (
	"context"
	"fmt"
	"strconv"

	"github.com/grapinou/LazyMarking/internal/db"
)

func RecoverRunningMarkingJobs(ctx context.Context, queries *db.Queries) error {
	jobs, err := queries.ListRunningMarkingJobs(ctx)
	if err != nil {
		return fmt.Errorf("list running marking jobs: %w", err)
	}

	for _, job := range jobs {
		operation := "marking-" + strconv.FormatInt(job.ID, 10)
		if err := RemoveOperationTempDir(job.Username, operation); err != nil {
			return fmt.Errorf("remove workspace for marking job %d: %w", job.ID, err)
		}

		rows, err := queries.FailMarkingJob(ctx, db.FailMarkingJobParams{
			ID:     job.ID,
			UserID: job.UserID,
		})
		if err != nil {
			return fmt.Errorf("mark interrupted marking job %d as failed: %w", job.ID, err)
		}
		if rows != 1 {
			return fmt.Errorf("mark interrupted marking job %d as failed: affected %d rows", job.ID, rows)
		}
	}

	return nil
}
