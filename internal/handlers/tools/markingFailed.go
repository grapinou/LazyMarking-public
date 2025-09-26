package tools

import (
	"context"

	"github.com/grapinou/LazyMarking/internal/db"
)

func MarkingFailed(userID, jobDBID int64, ctx context.Context, queries *db.Queries) {
	queries.UpdateMarkingJobStatus(ctx, db.UpdateMarkingJobStatusParams{
		Status: "failed",
		ID:     jobDBID,
		UserID: userID,
	})
}
