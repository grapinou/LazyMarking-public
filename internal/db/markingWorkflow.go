package db

import "context"

// The job entry point remains available for historical callers. Selection of
// current results has a single implementation, scoped to the generation.
type ListCurrentExamResultsForMarkingJobRow = ListCurrentExamResultsForGenerationRow
type ListCurrentExamResultsForMarkingJobParams struct {
	UserID       int64
	MarkingJobID int64
}

func (q *Queries) ListCurrentExamResultsForMarkingJob(ctx context.Context, arg ListCurrentExamResultsForMarkingJobParams) ([]ListCurrentExamResultsForMarkingJobRow, error) {
	generation, err := q.GetMarkingJobGeneration(ctx, GetMarkingJobGenerationParams{UserID: arg.UserID, MarkingJobID: arg.MarkingJobID})
	if err != nil || !generation.Valid {
		return nil, err
	}
	return q.ListCurrentExamResultsForGeneration(ctx, ListCurrentExamResultsForGenerationParams{UserID: arg.UserID, GenerationID: generation.Int64})
}

type ListRecentMarkingJobsRow = ListMarkingJobHistoryRow

func (q *Queries) ListRecentMarkingJobs(ctx context.Context, userID int64) ([]ListRecentMarkingJobsRow, error) {
	return q.ListMarkingJobHistory(ctx, ListMarkingJobHistoryParams{UserID: userID, GenerationID: int64(0)})
}
