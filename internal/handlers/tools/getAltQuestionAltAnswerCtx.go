package tools

import (
	"context"
	"database/sql"
	"log"
	"strconv"

	"github.com/grapinou/LazyMarking/internal/config"
	"github.com/grapinou/LazyMarking/internal/db"
)

func GetAltQuestionAltAnswerCtx(userID, altQuestionID int64, queries *db.Queries, ctx context.Context) (config.Question, error) {
	var question config.Question
	var answer config.Answer

	altQuestionDB, err := queries.GetAltQuestionByID(ctx, db.GetAltQuestionByIDParams{
		ID:     altQuestionID,
		UserID: userID,
	})
	if err != nil {
		log.Printf("From GetAltQuestionAltAnswer -> GetAltQuestionByID DB error : %v", err)
		return question, err
	}

	question.Content = altQuestionDB.Content

	altImageDB, err := queries.GetAltImageByAltQuestionID(ctx, db.GetAltImageByAltQuestionIDParams{
		AltQuestionID: altQuestionID,
		UserID:        userID,
		QuestionID:    altQuestionDB.QuestionID,
	})
	if err == sql.ErrNoRows {
		question.Image.Name = ""
		question.Image.Width = ""
	} else if err != nil {
		log.Printf("From GetAltQuestionAltAnswer -> GetAltImageByAltQuestionID DB error: %v", err)
		return question, err
	} else {
		question.Image.Name = altImageDB.ImageName
		question.Image.Width = strconv.FormatInt(altImageDB.ResizePercentage, 10)
	}

	altAnswersDB, err := queries.GetAllAltAnswersByAltQuestionID(ctx, db.GetAllAltAnswersByAltQuestionIDParams{
		AltQuestionID: altQuestionID,
		UserID:        userID,
	})
	if err != nil {
		log.Printf("From GetAltQuestionAltAnswer -> GetAllAltAnswersByAltQuestionID DB error : %v", err)
		return question, err
	}
	ShuffleSlice(altAnswersDB)
	for _, altAnswerDB := range altAnswersDB {
		/*
			if (i+1)%2 == 1 {
				answer.Symbol = "\\u{25B3}"
			} else {
				answer.Symbol = "\\u{25BD}"
			}
		*/

		answer.Symbol = "\\u{25CB}"
		answer.Content = altAnswerDB.Content
		answer.State = altAnswerDB.State
		question.Answers = append(question.Answers, answer)
	}

	return question, nil
}
