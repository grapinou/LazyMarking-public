package tools

import (
	"net/http"

	"github.com/grapinou/LazyMarking/internal/config"
	"github.com/grapinou/LazyMarking/internal/db"
)

var buildQuestionForQCM = BuildQuestion
var shuffleQCMQuestionIDs = ShuffleSlice[int64]

func GetQCMQuestionsAnswers(userID, qcmID int64, r *http.Request, queries *db.Queries) ([]config.Question, error) {
	questionsIDs, err := getQCMQuestionIDs(userID, qcmID, r, queries)
	if err != nil {
		return nil, err
	}
	shuffleQCMQuestionIDs(questionsIDs)
	return buildQCMQuestionsForRequest(questionsIDs, userID, r, queries)
}

func GetQCMQuestionsAnswersInReferenceOrder(userID, qcmID int64, r *http.Request, queries *db.Queries) ([]config.Question, error) {
	questionsIDs, err := getQCMQuestionIDs(userID, qcmID, r, queries)
	if err != nil {
		return nil, err
	}
	return buildQCMQuestionsForRequest(questionsIDs, userID, r, queries)
}

func getQCMQuestionIDs(userID, qcmID int64, r *http.Request, queries *db.Queries) ([]int64, error) {
	return queries.GetQCMQuestionsIDs(r.Context(), db.GetQCMQuestionsIDsParams{
		UserID: userID,
		QcmID:  qcmID,
	})
}

func buildQCMQuestionsForRequest(questionIDs []int64, userID int64, r *http.Request, queries *db.Queries) ([]config.Question, error) {
	return buildQCMQuestionsInOrder(questionIDs, func(questionID int64) (config.Question, error) {
		config.DBSemaphore <- struct{}{}
		defer func() { <-config.DBSemaphore }()
		return buildQuestionForQCM(questionID, userID, r, queries)
	})
}
