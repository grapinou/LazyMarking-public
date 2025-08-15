package tools

import (
	"fmt"
	"net/http"

	"github.com/grapinou/LazyMarking/internal/db"
)

func GetQCMQuestionAnswer(userID, qcmID int64, r *http.Request, queries *db.Queries) error {

	questionsIDs, err := queries.GetQCMQuestionsIDs(r.Context(), db.GetQCMQuestionsIDsParams{
		UserID: userID,
		QcmID:  qcmID,
	})
	if err != nil {
		return err
	}

	for _, questionID := range questionsIDs {

		tags, err := queries.GetTagsByQuestionID(r.Context(), db.GetTagsByQuestionIDParams{
			QuestionID: questionID,
			UserID:     userID,
		})
		if err != nil {
			return err
		}

		fmt.Println("les tags :")
		fmt.Println(tags)

		questionDB, err := queries.GetRandomQuestionByQuestionID(r.Context(), questionID)
		if err != nil {
			return err
		}
		fmt.Println("la q selectionnées :")
		fmt.Println(questionDB)

	}
	return nil
}
