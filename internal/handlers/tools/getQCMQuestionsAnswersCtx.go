package tools

import (
	"context"

	"github.com/grapinou/LazyMarking/internal/config"
	"github.com/grapinou/LazyMarking/internal/db"
)

var buildQuestionCtxForQCM = BuildQuestionCtx

func GetQCMQuestionsAnswersCtx(userID, qcmID int64, ctx context.Context, queries *db.Queries) ([]config.Question, error) {
	var qcmQuestions []config.Question
	questionsIDs, err := queries.GetQCMQuestionsIDs(ctx, db.GetQCMQuestionsIDsParams{
		UserID: userID,
		QcmID:  qcmID,
	})
	if err != nil {
		return qcmQuestions, err
	}
	shuffleQCMQuestionIDs(questionsIDs)

	return buildQCMQuestionsInOrder(questionsIDs, func(questionID int64) (config.Question, error) {
		select {
		case config.DBSemaphore <- struct{}{}:
		case <-ctx.Done():
			return config.Question{}, ctx.Err()
		}
		defer func() { <-config.DBSemaphore }()
		return buildQuestionCtxForQCM(questionID, userID, ctx, queries)
	})
}
