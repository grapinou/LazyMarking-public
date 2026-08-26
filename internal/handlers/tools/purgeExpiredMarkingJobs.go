package tools

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/grapinou/LazyMarking/internal/db"
)

const markingJobRetention = 7 * 24 * time.Hour

func PurgeExpiredMarkingJobs(ctx context.Context, queries *db.Queries, now time.Time) error {
	cutoff := sql.NullTime{
		Time:  now.UTC().Add(-markingJobRetention),
		Valid: true,
	}
	jobs, err := queries.ListExpiredMarkingJobs(ctx, cutoff)
	if err != nil {
		return fmt.Errorf("list expired marking jobs: %w", err)
	}

	for _, job := range jobs {
		operation := "marking-" + strconv.FormatInt(job.ID, 10)
		tempDir, err := operationTempDir(job.Username, operation)
		if err != nil {
			return fmt.Errorf("resolve workspace for expired marking job %d: %w", job.ID, err)
		}
		if err := os.RemoveAll(tempDir); err != nil {
			return fmt.Errorf("remove workspace for expired marking job %d: %w", job.ID, err)
		}
		if err := queries.DeleteMarkingJob(ctx, db.DeleteMarkingJobParams{
			ID:     job.ID,
			UserID: job.UserID,
		}); err != nil {
			return fmt.Errorf("delete expired marking job %d: %w", job.ID, err)
		}
	}

	return nil
}
