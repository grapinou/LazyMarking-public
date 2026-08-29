package tools

import (
	"net/http"

	"github.com/grapinou/LazyMarking/internal/config"
	"github.com/grapinou/LazyMarking/internal/db"
)

var buildQuestionForQCM = BuildQuestion
var shuffleQCMQuestionIDs = ShuffleSlice[int64]

func GetQCMQuestionsAnswers(userID, qcmID int64, r *http.Request, queries *db.Queries) ([]config.Question, error) {
	var qcmQuestions []config.Question
	questionsIDs, err := queries.GetQCMQuestionsIDs(r.Context(), db.GetQCMQuestionsIDsParams{
		UserID: userID,
		QcmID:  qcmID,
	})
	if err != nil {
		return qcmQuestions, err
	}
	shuffleQCMQuestionIDs(questionsIDs)

	return buildQCMQuestionsInOrder(questionsIDs, func(questionID int64) (config.Question, error) {
		config.DBSemaphore <- struct{}{}
		defer func() { <-config.DBSemaphore }()
		return buildQuestionForQCM(questionID, userID, r, queries)
	})
}
