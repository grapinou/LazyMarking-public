package tools

import (
	"context"
	"database/sql"
	"log"
	"strconv"

	"github.com/grapinou/LazyMarking/internal/config"
	"github.com/grapinou/LazyMarking/internal/db"
)

func GetQuestionAnswerCtx(userID, questionID int64, queries *db.Queries, ctx context.Context) (config.Question, error) {
	var question config.Question
	var answer config.Answer

	questionDB, err := queries.GetQuestionByID(ctx, db.GetQuestionByIDParams{
		ID:     questionID,
		UserID: userID,
	})
	if err != nil {
		log.Printf("From GetQuestionAnswer -> GetQuestionByID DB error : %v", err)
		return question, err
	}

	question.Content = questionDB.Content

	imageDB, err := queries.GetImageByQuestionID(ctx, db.GetImageByQuestionIDParams{
		QuestionID: questionID,
		UserID:     userID,
	})
	if err == sql.ErrNoRows {
		question.Image.Name = ""
		question.Image.Width = ""
	} else if err != nil {
		log.Printf("From GetQuestionAnswer -> GetImageByQuestionID DB error: %v", err)
		return question, err
	} else {
		question.Image.Name = imageDB.ImageName
		question.Image.Width = strconv.FormatInt(imageDB.ResizePercentage, 10)
	}

	answersDB, err := queries.GetAllAnswersByQuestionID(ctx, db.GetAllAnswersByQuestionIDParams{
		QuestionID: questionID,
		UserID:     userID,
	})
	if err != nil {
		log.Printf("From GetQuestionAnswer -> GetAllAnswersByQuestionID DB error : %v", err)
		return question, err
	}
	ShuffleSlice(answersDB)
	for _, answerDB := range answersDB {
		/*
			if (i+1)%2 == 1 {
				answer.Symbol = "\\u{25B3}"
			} else {
				answer.Symbol = "\\u{25BD}"
			}
		*/
		answer.Symbol = "\\u{25CB}"
		answer.Content = answerDB.Content
		answer.State = answerDB.State
		question.Answers = append(question.Answers, answer)
	}

	return question, nil
}
