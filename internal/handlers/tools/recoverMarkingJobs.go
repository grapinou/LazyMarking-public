package tools

import (
	"context"
	"fmt"
	"log"
	"strconv"

	"github.com/grapinou/LazyMarking/internal/db"
)

type MarkingRecoveryResult struct {
	Found              int
	Recovered          int
	CleanupFailures    int
	TransitionFailures int
}

func RecoverRunningMarkingJobs(ctx context.Context, queries *db.Queries) (MarkingRecoveryResult, error) {
	var result MarkingRecoveryResult
	jobs, err := queries.ListRunningMarkingJobs(ctx)
	if err != nil {
		return result, fmt.Errorf("list running marking jobs: %w", err)
	}
	result.Found = len(jobs)

	for _, job := range jobs {
		operation := "marking-" + strconv.FormatInt(job.ID, 10)
		if err := RemoveOperationTempDir(job.Username, operation); err != nil {
			result.CleanupFailures++
			log.Printf("marking recovery: cleanup failed for job %d: %v", job.ID, err)
		}

		rows, err := queries.FailMarkingJob(ctx, db.FailMarkingJobParams{
			ID:     job.ID,
			UserID: job.UserID,
		})
		if err != nil {
			result.TransitionFailures++
			log.Printf("marking recovery: failed transition for job %d: %v", job.ID, err)
			continue
		}
		if rows != 1 {
			result.TransitionFailures++
			log.Printf("marking recovery: failed transition for job %d: affected %d rows", job.ID, rows)
			continue
		}
		result.Recovered++
	}

	return result, nil
}
