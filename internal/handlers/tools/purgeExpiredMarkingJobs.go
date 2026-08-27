package tools

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/grapinou/LazyMarking/internal/db"
)

const markingJobRetention = 7 * 24 * time.Hour

func PurgeExpiredMarkingJobs(ctx context.Context, queries *db.Queries, now time.Time) error {
	cutoff := now.UTC().Add(-markingJobRetention).Format("2006-01-02 15:04:05")
	jobs, err := queries.ListExpiredMarkingJobs(ctx, cutoff)
	if err != nil {
		return fmt.Errorf("list expired marking jobs: %w", err)
	}

	for _, job := range jobs {
		operation := "marking-" + strconv.FormatInt(job.ID, 10)
		if err := RemoveOperationTempDir(job.Username, operation); err != nil {
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
