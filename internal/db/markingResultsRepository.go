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
	AnswerIndex     int64
	DetectedState   int64
	MeanGray        float64
	HistoricalState sql.NullInt64
	V2State         sql.NullInt64
	DarkRatio       sql.NullFloat64
	ChromaRatio     sql.NullFloat64
	GrayscaleSignal sql.NullInt64
	ColorSignal     sql.NullInt64
	AutomaticState  sql.NullInt64
	ReviewReason    sql.NullString
}

// PersistCorrectedMarkingCopy writes one terminal corrected result and all of
// its children atomically.
func PersistCorrectedMarkingCopy(ctx context.Context, conn *sql.DB, input PersistedMarkingCopyInput) (copyResultID int64, err error) {
	return PersistCorrectedMarkingCopyWithQueries(ctx, New(conn), input)
}

type beginTxDB interface {
	BeginTx(context.Context, *sql.TxOptions) (*sql.Tx, error)
}

func PersistCorrectedMarkingCopyWithQueries(ctx context.Context, queries *Queries, input PersistedMarkingCopyInput) (copyResultID int64, err error) {
	var questionScoreTotal int64
	var questionPointTotal int64
	for _, question := range input.Questions {
		questionScoreTotal += question.ScoreHalfUnits
		questionPointTotal += question.TotalPoints
	}
	if questionScoreTotal != input.ScoreHalfUnits || questionPointTotal != input.TotalPoints {
		return 0, fmt.Errorf("copy totals do not match question totals")
	}

	beginner, ok := queries.db.(beginTxDB)
	if !ok {
		return 0, fmt.Errorf("marking result store does not support transactions")
	}
	tx, err := beginner.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	txQueries := New(tx)
	copyResultID, err = txQueries.CreateMarkingCopyResult(ctx, CreateMarkingCopyResultParams{
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
		questionResultID, createErr := txQueries.CreateMarkingQuestionResult(ctx, CreateMarkingQuestionResultParams{
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
			if answer.HistoricalState.Valid {
				_, createErr = txQueries.CreateHybridMarkingAnswerDetection(ctx, CreateHybridMarkingAnswerDetectionParams{
					QuestionResultID: questionResultID, AnswerIndex: answer.AnswerIndex,
					DetectedState: answer.DetectedState, MeanGray: answer.MeanGray,
					HistoricalState: answer.HistoricalState, V2State: answer.V2State,
					DarkRatio: answer.DarkRatio, ChromaRatio: answer.ChromaRatio,
					GrayscaleSignal: answer.GrayscaleSignal, ColorSignal: answer.ColorSignal,
					AutomaticState: answer.AutomaticState,
					ReviewReason:   answer.ReviewReason,
				})
			} else {
				_, createErr = txQueries.CreateMarkingAnswerDetection(ctx, CreateMarkingAnswerDetectionParams{
					QuestionResultID: questionResultID, AnswerIndex: answer.AnswerIndex,
					DetectedState: answer.DetectedState, MeanGray: answer.MeanGray,
				})
			}
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

type PersistedTerminalMarkingCopyInput struct {
	UserID        int64
	MarkingJobID  int64
	StudentExamID int64
	Outcome       string
	ExpectedPages int64
	DetectedPages int64
	FailureCode   string
}

func PersistTerminalMarkingCopy(ctx context.Context, queries *Queries, input PersistedTerminalMarkingCopyInput) (int64, error) {
	if input.Outcome != "incomplete" && input.Outcome != "not_seen" && input.Outcome != "error" {
		return 0, fmt.Errorf("invalid non-corrected outcome %q", input.Outcome)
	}
	return queries.CreateMarkingCopyResult(ctx, CreateMarkingCopyResultParams{
		UserID:        input.UserID,
		MarkingJobID:  input.MarkingJobID,
		StudentExamID: input.StudentExamID,
		Outcome:       input.Outcome,
		ExpectedPages: input.ExpectedPages,
		DetectedPages: input.DetectedPages,
		FailureCode: sql.NullString{
			String: input.FailureCode,
			Valid:  input.FailureCode != "",
		},
	})
}
