package training

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"sort"

	"github.com/grapinou/LazyMarking/internal/config"
	"github.com/grapinou/LazyMarking/internal/db"
	"github.com/grapinou/LazyMarking/internal/handlers/marking"
)

// libraryCandidates adds current bank questions from the same P3 themes as
// optional draft cards. P3 candidates always remain the initial selection.
func libraryCandidates(ctx context.Context, q *db.Queries, conn *sql.DB, userID int64, observed []marking.TrainingCandidate) ([]marking.TrainingCandidate, error) {
	themes := map[int64]bool{}
	seen := map[string]bool{}
	for _, item := range observed {
		tags := item.Question.Tags
		if tags.Theme.ID > 0 {
			themes[tags.Theme.ID] = true
		}
		if tags.VariantID > 0 {
			seen[fmt.Sprintf("%s:%d", tags.VariantType, tags.VariantID)] = true
		}
	}
	themeIDs := make([]int64, 0, len(themes))
	for id := range themes {
		themeIDs = append(themeIDs, id)
	}
	sort.Slice(themeIDs, func(i, j int) bool { return themeIDs[i] < themeIDs[j] })
	familyIDs := make([]int64, 0)
	families := map[int64]bool{}
	for _, themeID := range themeIDs {
		rows, err := conn.QueryContext(ctx, `SELECT id FROM questions WHERE user_id=? AND theme_id=? ORDER BY id LIMIT 40`, userID, themeID)
		if err != nil {
			return nil, err
		}
		for rows.Next() {
			var id int64
			if err := rows.Scan(&id); err != nil {
				rows.Close()
				return nil, err
			}
			if !families[id] {
				families[id] = true
				familyIDs = append(familyIDs, id)
			}
		}
		if err := rows.Err(); err != nil {
			rows.Close()
			return nil, err
		}
		rows.Close()
	}
	result := make([]marking.TrainingCandidate, 0)
	for _, familyID := range familyIDs {
		main, err := q.GetQuestionByID(ctx, db.GetQuestionByIDParams{ID: familyID, UserID: userID})
		if errors.Is(err, sql.ErrNoRows) {
			continue
		}
		if err != nil {
			return nil, err
		}
		answers, err := q.GetAllAnswersByQuestionID(ctx, db.GetAllAnswersByQuestionIDParams{QuestionID: familyID, UserID: userID})
		if err != nil {
			return nil, err
		}
		sort.Slice(answers, func(i, j int) bool { return answers[i].ID < answers[j].ID })
		if _, err := q.GetImageByQuestionID(ctx, db.GetImageByQuestionIDParams{QuestionID: familyID, UserID: userID}); err == nil {
			continue
		} else if !errors.Is(err, sql.ErrNoRows) {
			return nil, err
		}
		question := config.Question{Instruction: main.Instruction, Content: main.Content, Tags: config.Tags{MainQuestionID: familyID, VariantType: config.MainQuestion, VariantID: familyID}}
		for _, a := range answers {
			question.Answers = append(question.Answers, config.Answer{Content: a.Content, State: a.State})
		}
		if !seen[fmt.Sprintf("%s:%d", config.MainQuestion, familyID)] {
			if _, err := fromQuestion(question); err == nil {
				result = append(result, marking.TrainingCandidate{Question: question})
			}
		}
		variants, err := q.GetAllAltQuestions(ctx, db.GetAllAltQuestionsParams{QuestionID: familyID, UserID: userID})
		if err != nil {
			return nil, err
		}
		sort.Slice(variants, func(i, j int) bool { return variants[i].ID < variants[j].ID })
		for _, variant := range variants {
			key := fmt.Sprintf("%s:%d", config.AltQuestion, variant.ID)
			if seen[key] {
				continue
			}
			if _, err := q.GetAltImageByAltQuestionID(ctx, db.GetAltImageByAltQuestionIDParams{AltQuestionID: variant.ID, UserID: userID, QuestionID: familyID}); err == nil {
				continue
			} else if !errors.Is(err, sql.ErrNoRows) {
				return nil, err
			}
			altAnswers, err := q.GetAllAltAnswersByAltQuestionID(ctx, db.GetAllAltAnswersByAltQuestionIDParams{AltQuestionID: variant.ID, UserID: userID})
			if err != nil {
				return nil, err
			}
			alt := config.Question{Instruction: main.Instruction, Content: variant.Content, Tags: config.Tags{MainQuestionID: familyID, VariantType: config.AltQuestion, VariantID: variant.ID}}
			sort.Slice(altAnswers, func(i, j int) bool { return altAnswers[i].ID < altAnswers[j].ID })
			for _, a := range altAnswers {
				alt.Answers = append(alt.Answers, config.Answer{Content: a.Content, State: a.State})
			}
			if _, err := fromQuestion(alt); err == nil {
				result = append(result, marking.TrainingCandidate{Question: alt})
			}
		}
	}
	return result, nil
}
