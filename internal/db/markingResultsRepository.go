package db

import (
	"context"
	"database/sql"
	"fmt"
)

// PersistedMarkingCopyInput is an isolated persistence boundary for a complete,
// already-computed corrected copy. It is not connected to the marking pipeline.
type PersistedMarkingCopyInput struct {
	UserID         int64
	MarkingJobID   int64
	StudentExamID  int64
	ExpectedPages  int64
	DetectedPages  int64
	ScoreHalfUnits int64
	TotalPoints    int64
	Questions      []PersistedQuestionInput
}

type PersistedQuestionInput struct {
	QuestionIndex  int64
	State          string
	ScoreHalfUnits int64
	TotalPoints    int64
	Answers        []PersistedAnswerDetectionInput
}

type PersistedAnswerDetectionInput struct {
	AnswerIndex   int64
	DetectedState int64
	MeanGray      float64
}

// PersistCorrectedMarkingCopy writes one terminal corrected result and all of
// its children atomically. OpenCV and the production marking pipeline do not
// call this helper yet.
func PersistCorrectedMarkingCopy(ctx context.Context, conn *sql.DB, input PersistedMarkingCopyInput) (copyResultID int64, err error) {
	var questionScoreTotal int64
	var questionPointTotal int64
	for _, question := range input.Questions {
		questionScoreTotal += question.ScoreHalfUnits
		questionPointTotal += question.TotalPoints
	}
	if questionScoreTotal != input.ScoreHalfUnits || questionPointTotal != input.TotalPoints {
		return 0, fmt.Errorf("copy totals do not match question totals")
	}

	tx, err := conn.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	queries := New(tx)
	copyResultID, err = queries.CreateMarkingCopyResult(ctx, CreateMarkingCopyResultParams{
		UserID:         input.UserID,
		MarkingJobID:   input.MarkingJobID,
		StudentExamID:  input.StudentExamID,
		Outcome:        "corrected",
		ExpectedPages:  input.ExpectedPages,
		DetectedPages:  input.DetectedPages,
		ScoreHalfUnits: sql.NullInt64{Int64: input.ScoreHalfUnits, Valid: true},
		TotalPoints:    sql.NullInt64{Int64: input.TotalPoints, Valid: true},
	})
	if err != nil {
		return 0, err
	}

	for _, question := range input.Questions {
		questionResultID, createErr := queries.CreateMarkingQuestionResult(ctx, CreateMarkingQuestionResultParams{
			CopyResultID:   copyResultID,
			QuestionIndex:  question.QuestionIndex,
			State:          question.State,
			ScoreHalfUnits: question.ScoreHalfUnits,
			TotalPoints:    question.TotalPoints,
		})
		if createErr != nil {
			err = createErr
			return 0, err
		}
		for _, answer := range question.Answers {
			_, createErr = queries.CreateMarkingAnswerDetection(ctx, CreateMarkingAnswerDetectionParams{
				QuestionResultID: questionResultID,
				AnswerIndex:      answer.AnswerIndex,
				DetectedState:    answer.DetectedState,
				MeanGray:         answer.MeanGray,
			})
			if createErr != nil {
				err = createErr
				return 0, err
			}
		}
	}

	if err = tx.Commit(); err != nil {
		return 0, err
	}
	return copyResultID, nil
}
