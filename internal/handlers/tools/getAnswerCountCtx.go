package tools

import (
	"context"

	"github.com/grapinou/LazyMarking/internal/db"
)

func GetAnswerCountCtx(userID int64, questionDB db.GetRandomQuestionByQuestionIDRow, ctx context.Context, queries *db.Queries) (int64, error) {
	if questionDB.IsAlt == 0 {
		return queries.CountAnswerByQuestionID(ctx, db.CountAnswerByQuestionIDParams{
			QuestionID: questionDB.ItemID,
			UserID:     userID,
		})
	}
	return queries.CountAltAnswerByAltQuestionID(ctx, db.CountAltAnswerByAltQuestionIDParams{
		AltQuestionID: questionDB.ItemID,
		UserID:        userID,
	})
}
