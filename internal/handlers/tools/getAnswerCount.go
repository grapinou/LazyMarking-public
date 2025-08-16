package tools

import (
	"net/http"

	"github.com/grapinou/LazyMarking/internal/db"
)

func GetAnswerCount(userID int64, questionDB db.GetRandomQuestionByQuestionIDRow, r *http.Request, queries *db.Queries) (int64, error) {
	if questionDB.IsAlt == 0 {
		return queries.CountAnswerByQuestionID(r.Context(), db.CountAnswerByQuestionIDParams{
			QuestionID: questionDB.ItemID,
			UserID:     userID,
		})
	}
	return queries.CountAltAnswerByAltQuestionID(r.Context(), db.CountAltAnswerByAltQuestionIDParams{
		AltQuestionID: questionDB.ItemID,
		UserID:        userID,
	})
}
