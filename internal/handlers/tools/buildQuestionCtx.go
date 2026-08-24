package tools

import (
	"context"

	"github.com/grapinou/LazyMarking/internal/config"
	"github.com/grapinou/LazyMarking/internal/db"
)

// var ErrQuestionWithNoAnswer = errors.New("no question found with answer")

func BuildQuestionCtx(questionID, userID int64, ctx context.Context, queries *db.Queries) (config.Question, error) {
	var question config.Question

	// récupération des tags
	tagsDB, err := queries.GetTagsByQuestionID(ctx, db.GetTagsByQuestionIDParams{
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
		questionDB, err = queries.GetRandomQuestionByQuestionID(ctx, db.GetRandomQuestionByQuestionIDParams{
			QuestionID: questionID,
			UserID:     userID,
		})
		if err != nil {
			return question, err
		}

		count, err := GetAnswerCountCtx(userID, questionDB, ctx, queries)
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
		question, err = GetQuestionAnswerCtx(userID, questionDB.ItemID, queries, ctx)
		if err != nil {
			return question, err
		}
	} else {
		question, err = GetAltQuestionAltAnswerCtx(userID, questionDB.ItemID, queries, ctx)
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
