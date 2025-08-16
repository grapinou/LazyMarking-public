package tools

import (
	"errors"
	"net/http"

	"github.com/grapinou/LazyMarking/internal/config"
	"github.com/grapinou/LazyMarking/internal/db"
)

var ErrQuestionWithNoAnswer = errors.New("no question found with answer")

func GetQCMQuestionsAnswers(userID, qcmID int64, r *http.Request, queries *db.Queries) ([]config.Question, error) {
	var qcmQuestions []config.Question
	questionsIDs, err := queries.GetQCMQuestionsIDs(r.Context(), db.GetQCMQuestionsIDsParams{
		UserID: userID,
		QcmID:  qcmID,
	})
	if err != nil {
		return qcmQuestions, err
	}
	ShuffleSlice(questionsIDs)

	for _, questionID := range questionsIDs {
		var question config.Question

		// récupération des tags
		tagsDB, err := queries.GetTagsByQuestionID(r.Context(), db.GetTagsByQuestionIDParams{
			QuestionID: questionID,
			UserID:     userID,
		})
		if err != nil {
			return qcmQuestions, err
		}

		// récupération d'une question avec des réponses
		const maxTries = 100
		found := false
		var questionDB db.GetRandomQuestionByQuestionIDRow
		for i := range maxTries {
			_ = i
			questionDB, err = queries.GetRandomQuestionByQuestionID(r.Context(), questionID)
			if err != nil {
				return qcmQuestions, err
			}

			count, err := GetAnswerCount(userID, questionDB, r, queries)
			if err != nil {
				return qcmQuestions, err
			}
			if count > 0 {
				found = true
				break
			}
		}
		if !found {
			return qcmQuestions, ErrQuestionWithNoAnswer
		}

		if questionDB.IsAlt == 0 {
			question, err = GetQuestionAnswer(userID, questionDB.ItemID, queries, r)
			if err != nil {
				return qcmQuestions, err
			}
		} else {
			question, err = GetAltQuestionAltAnswer(userID, questionDB.ItemID, queries, r)
			if err != nil {
				return qcmQuestions, err
			}
		}

		question.Tags = config.Tags{
			Subject: config.Subject{
				ID:   tagsDB.SubjectID,
				Name: tagsDB.SubjectName,
			},
			Theme: config.Theme{
				ID:   tagsDB.ThemeID,
				Name: tagsDB.ThemeName,
			},
			YearLevel: config.YearLevel{
				ID:   tagsDB.YearLevelID,
				Name: tagsDB.YearLevelName,
			},
			Skill: config.Skill{
				ID:   tagsDB.SkillID,
				Name: tagsDB.SkillName,
			},
			Difficulty: config.Difficulty{
				ID:   tagsDB.DifficultyID,
				Name: tagsDB.DifficultyName,
			},
			Point: config.Point{
				ID:         tagsDB.PointID,
				PointValue: tagsDB.PointValue,
			},
		}
		qcmQuestions = append(qcmQuestions, question)

	}

	return qcmQuestions, nil
}
