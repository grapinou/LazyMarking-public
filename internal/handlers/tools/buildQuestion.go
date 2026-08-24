package tools

import (
	"errors"
	"net/http"

	"github.com/grapinou/LazyMarking/internal/config"
	"github.com/grapinou/LazyMarking/internal/db"
)

var ErrQuestionWithNoAnswer = errors.New("no question found with answer")

func BuildQuestion(questionID, userID int64, r *http.Request, queries *db.Queries) (config.Question, error) {

	var question config.Question

	// récupération des tags
	tagsDB, err := queries.GetTagsByQuestionID(r.Context(), db.GetTagsByQuestionIDParams{
		QuestionID: questionID,
		UserID:     userID,
	})
	if err != nil {
		return question, err
	}

	// récupération d'une question avec des réponses
	const maxTries = 100
	found := false
	var questionDB db.GetRandomQuestionByQuestionIDRow
	for i := range maxTries {
		_ = i
		questionDB, err = queries.GetRandomQuestionByQuestionID(r.Context(), db.GetRandomQuestionByQuestionIDParams{
			QuestionID: questionID,
			UserID:     userID,
		})
		if err != nil {
			return question, err
		}

		count, err := GetAnswerCount(userID, questionDB, r, queries)
		if err != nil {
			return question, err
		}
		if count > 0 {
			found = true
			break
		}
	}
	if !found {
		return question, ErrQuestionWithNoAnswer
	}

	if questionDB.IsAlt == 0 {
		question, err = GetQuestionAnswer(userID, questionDB.ItemID, queries, r)
		if err != nil {
			return question, err
		}
	} else {
		question, err = GetAltQuestionAltAnswer(userID, questionDB.ItemID, queries, r)
		if err != nil {
			return question, err
		}
	}

	question.Tags = config.Tags{
		MainQuestionID: questionID,
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

	return question, nil

}
