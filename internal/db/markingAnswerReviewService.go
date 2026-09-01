package db

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/grapinou/LazyMarking/internal/config"
	"github.com/grapinou/LazyMarking/internal/markingscoring"
)

var (
	ErrMarkingReviewUnavailable = errors.New("marking review unavailable")
	ErrMarkingReviewConflict    = errors.New("marking review conflict")
)

type ApplyMarkingAnswerReviewInput struct {
	UserID                       int64
	MarkingJobID                 int64
	AnswerDetectionID            int64
	ReviewedState                int64
	ExpectedAnswerReviewRevision int64
	ExpectedJobReviewRevision    int64
}

type ApplyMarkingAnswerReviewResult struct {
	NoOp                   bool
	DetectedState          int64
	ReviewedState          int64
	EffectiveState         int64
	AnswerReviewRevision   int64
	JobReviewRevision      int64
	ArtifactsRevision      int64
	QuestionState          string
	QuestionScoreHalfUnits int64
	CopyScoreHalfUnits     int64
}

// ApplyMarkingAnswerReview records one human decision and recomputes the
// authoritative effective question/copy result in a single transaction.
func ApplyMarkingAnswerReview(ctx context.Context, queries *Queries, input ApplyMarkingAnswerReviewInput) (result ApplyMarkingAnswerReviewResult, err error) {
	if input.UserID <= 0 || input.MarkingJobID <= 0 || input.AnswerDetectionID <= 0 ||
		(input.ReviewedState != 0 && input.ReviewedState != 1) ||
		input.ExpectedAnswerReviewRevision < 0 || input.ExpectedJobReviewRevision < 0 {
		return result, ErrMarkingReviewUnavailable
	}
	beginner, ok := queries.db.(beginTxDB)
	if !ok {
		return result, fmt.Errorf("marking review store does not support transactions")
	}
	tx, err := beginner.BeginTx(ctx, nil)
	if err != nil {
		return result, err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()
	txQueries := New(tx)
	target, err := txQueries.GetMarkingAnswerReviewTarget(ctx, GetMarkingAnswerReviewTargetParams{
		MarkingJobID: input.MarkingJobID, UserID: input.UserID, AnswerDetectionID: input.AnswerDetectionID,
	})
	if errors.Is(err, sql.ErrNoRows) {
		return result, ErrMarkingReviewUnavailable
	}
	if err != nil {
		return result, err
	}
	if target.AnswerReviewRevision.Valid {
		if target.AnswerReviewRevision.Int64 != input.ExpectedAnswerReviewRevision {
			return result, ErrMarkingReviewConflict
		}
		if target.ReviewedState.Int64 == input.ReviewedState {
			if target.JobReviewRevision != input.ExpectedJobReviewRevision {
				return result, ErrMarkingReviewConflict
			}
			_ = tx.Rollback()
			return ApplyMarkingAnswerReviewResult{
				NoOp: true, DetectedState: target.DetectedState, ReviewedState: input.ReviewedState,
				EffectiveState: target.EffectiveState, AnswerReviewRevision: target.AnswerReviewRevision.Int64,
				JobReviewRevision: target.JobReviewRevision, ArtifactsRevision: target.ArtifactsRevision,
			}, nil
		}
	} else if input.ExpectedAnswerReviewRevision != 0 {
		return result, ErrMarkingReviewConflict
	}

	effectiveChanged := target.EffectiveState != input.ReviewedState
	answerRevision := int64(1)
	if target.AnswerReviewRevision.Valid {
		rows, updateErr := txQueries.UpdateMarkingAnswerReview(ctx, UpdateMarkingAnswerReviewParams{
			ReviewedState: input.ReviewedState, AnswerDetectionID: input.AnswerDetectionID,
			ReviewerUserID: input.UserID, ExpectedRevision: input.ExpectedAnswerReviewRevision,
		})
		if updateErr != nil {
			return result, updateErr
		}
		if rows != 1 {
			return result, ErrMarkingReviewConflict
		}
		answerRevision = input.ExpectedAnswerReviewRevision + 1
	} else {
		if _, createErr := txQueries.CreateMarkingAnswerReview(ctx, CreateMarkingAnswerReviewParams{
			AnswerDetectionID: input.AnswerDetectionID, ReviewerUserID: input.UserID, ReviewedState: input.ReviewedState,
		}); createErr != nil {
			return result, createErr
		}
	}

	answers, err := txQueries.ListEffectiveQuestionAnswersForReview(ctx, ListEffectiveQuestionAnswersForReviewParams{
		QuestionResultID: target.QuestionResultID, CopyResultID: target.CopyResultID,
		MarkingJobID: input.MarkingJobID, UserID: input.UserID,
	})
	if err != nil {
		return result, err
	}
	var snapshot config.QCM
	if err := json.Unmarshal([]byte(target.SnapshotContent), &snapshot); err != nil {
		return result, fmt.Errorf("decode marking snapshot: %w", err)
	}
	if target.QuestionIndex < 0 || target.QuestionIndex >= int64(len(snapshot.Questions)) {
		return result, fmt.Errorf("question index is outside marking snapshot")
	}
	question := snapshot.Questions[target.QuestionIndex]
	if int64(question.Tags.Point.PointValue) != target.TotalPoints || len(answers) != len(question.Answers) {
		return result, fmt.Errorf("persisted question does not match marking snapshot")
	}
	expected := make([]int, len(question.Answers))
	effective := make([]int, len(answers))
	for index, answer := range answers {
		if answer.AnswerIndex != int64(index) || (answer.EffectiveState != 0 && answer.EffectiveState != 1) {
			return result, fmt.Errorf("effective answer vector is invalid")
		}
		expected[index] = int(question.Answers[index].State)
		effective[index] = int(answer.EffectiveState)
	}
	mark := markingscoring.ScoreQuestion(expected, effective, target.TotalPoints)
	questionState, questionHalfUnits := persistedQuestionMark(mark)
	rows, err := txQueries.UpdateMarkingQuestionResultFromReview(ctx, UpdateMarkingQuestionResultFromReviewParams{
		State: questionState, ScoreHalfUnits: questionHalfUnits, QuestionResultID: target.QuestionResultID,
		CopyResultID: target.CopyResultID, UserID: input.UserID, MarkingJobID: input.MarkingJobID,
	})
	if err != nil {
		return result, err
	}
	if rows != 1 {
		return result, ErrMarkingReviewUnavailable
	}
	rows, err = txQueries.RecalculateMarkingCopyScoreFromQuestions(ctx, RecalculateMarkingCopyScoreFromQuestionsParams{
		CopyResultID: target.CopyResultID, UserID: input.UserID, MarkingJobID: input.MarkingJobID,
	})
	if err != nil {
		return result, err
	}
	if rows != 1 {
		return result, ErrMarkingReviewUnavailable
	}
	changed := int64(0)
	if effectiveChanged {
		changed = 1
	}
	rows, err = txQueries.AdvanceMarkingJobReviewRevision(ctx, AdvanceMarkingJobReviewRevisionParams{
		EffectiveChanged: changed, MarkingJobID: input.MarkingJobID, UserID: input.UserID,
		ExpectedReviewRevision: input.ExpectedJobReviewRevision,
	})
	if err != nil {
		return result, err
	}
	if rows != 1 {
		return result, ErrMarkingReviewConflict
	}
	copyResult, err := txQueries.GetMarkingCopyResult(ctx, GetMarkingCopyResultParams{ID: target.CopyResultID, UserID: input.UserID})
	if err != nil {
		return result, err
	}
	newReviewRevision := target.JobReviewRevision + 1
	newArtifactsRevision := target.ArtifactsRevision
	if !effectiveChanged && target.ArtifactsRevision == target.JobReviewRevision {
		newArtifactsRevision = newReviewRevision
	}
	if err = tx.Commit(); err != nil {
		return result, err
	}
	return ApplyMarkingAnswerReviewResult{
		DetectedState: target.DetectedState, ReviewedState: input.ReviewedState,
		EffectiveState: input.ReviewedState, AnswerReviewRevision: answerRevision,
		JobReviewRevision: newReviewRevision, ArtifactsRevision: newArtifactsRevision,
		QuestionState: questionState, QuestionScoreHalfUnits: questionHalfUnits,
		CopyScoreHalfUnits: copyResult.ScoreHalfUnits.Int64,
	}, nil
}

func persistedQuestionMark(mark config.QuestionMark) (string, int64) {
	switch mark.State {
	case config.Correct:
		return "correct", 2 * mark.Total
	case config.Partial:
		return "partial", mark.Total
	default:
		return "incorrect", 0
	}
}
