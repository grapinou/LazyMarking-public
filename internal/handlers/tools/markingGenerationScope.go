package tools

import (
	"context"
	"errors"

	"github.com/grapinou/LazyMarking/internal/db"
)

var ErrQrOutsideMarkingGeneration = errors.New("QR page is outside the selected marking generation")

// ValidateQrCodeForMarkingJob treats missing, cross-user and cross-generation
// student exams identically so a QR ID never becomes an independent authority.
func ValidateQrCodeForMarkingJob(ctx context.Context, queries *db.Queries, markingJobID, userID, studentExamID int64) error {
	_, err := queries.ValidateMarkingJobStudentExam(ctx, db.ValidateMarkingJobStudentExamParams{
		MarkingJobID:  markingJobID,
		UserID:        userID,
		StudentExamID: studentExamID,
	})
	if err != nil {
		return ErrQrOutsideMarkingGeneration
	}
	return nil
}
