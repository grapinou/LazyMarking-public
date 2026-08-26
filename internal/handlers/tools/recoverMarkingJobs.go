package tools

import (
	"context"
	"fmt"
	"os"
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
		tempDir, err := operationTempDir(job.Username, operation)
		if err != nil {
			return fmt.Errorf("resolve workspace for marking job %d: %w", job.ID, err)
		}

		if err := os.RemoveAll(tempDir); err != nil {
			return fmt.Errorf("remove workspace for marking job %d: %w", job.ID, err)
		}

		if err := queries.FailMarkingJob(ctx, db.FailMarkingJobParams{
			ID:     job.ID,
			UserID: job.UserID,
		}); err != nil {
			return fmt.Errorf("mark interrupted marking job %d as failed: %w", job.ID, err)
		}
	}

	return nil
}
