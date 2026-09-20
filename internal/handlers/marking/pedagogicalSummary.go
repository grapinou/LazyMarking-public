package marking

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/grapinou/LazyMarking/internal/config"
	"github.com/grapinou/LazyMarking/internal/db"
	"github.com/grapinou/LazyMarking/internal/handlers/tools"
	"github.com/grapinou/LazyMarking/internal/templates/data"
)

type persistedPedagogicalQuestion struct {
	Index int64  `json:"question_index"`
	State string `json:"state"`
	Half  int64  `json:"score_half_units"`
	Total int64  `json:"total_points"`
}

type pedagogicalQuestionKey struct {
	Family  int64
	Version string
}

type pedagogicalQuestionCounter struct {
	Question config.Question
	Count    int
	Correct  int
	Half     int64
	Total    int64
}

// The input is the very same current-result snapshot used by the HTML table.
// No job selection or answer rescoring takes place here.
func buildMarkingPedagogicalSummary(rows []db.ListCurrentExamResultsForGenerationRow) data.MarkingPedagogicalSummaryView {
	summary := data.MarkingPedagogicalSummaryView{}
	scoresByTotal := make(map[int64][]float64)
	questions := make(map[pedagogicalQuestionKey]*pedagogicalQuestionCounter)
	marks := make([]config.MarkExam, 0, len(rows))
	for _, row := range rows {
		if !hasFinalMarkingScore(row) || row.TotalPoints.Int64 <= 0 {
			summary.ExcludedCopies++
			continue
		}
		summary.IncludedCopies++
		total := row.TotalPoints.Int64
		scoresByTotal[total] = append(scoresByTotal[total], float64(row.ScoreHalfUnits.Int64)/2)
		qcm, questionMarks, ok := pedagogicalCopyDetails(row)
		if !ok {
			// Older data can still have a final grade without usable question
			// details. Keep the grade; make the missing detail coverage explicit.
			continue
		}
		summary.DetailedCopies++
		for index, question := range qcm.Questions {
			key := pedagogicalQuestionKey{Family: question.Tags.MainQuestionID, Version: pedagogicalQuestionVersion(question)}
			counter := questions[key]
			if counter == nil {
				counter = &pedagogicalQuestionCounter{Question: question}
				questions[key] = counter
			}
			mark := questionMarks[index]
			counter.Count++
			counter.Half += int64(mark.Score * 2)
			counter.Total += mark.Total
			if mark.State == config.Correct {
				counter.Correct++
			}
		}
		skills, themeSkills := tools.GetThemeSkill(qcm, questionMarks)
		marks = append(marks, config.MarkExam{Skill: skills, ThemeSkill: themeSkills})
	}
	var totals []int64
	for total := range scoresByTotal {
		totals = append(totals, total)
	}
	sort.Slice(totals, func(i, j int) bool { return totals[i] < totals[j] })
	for _, total := range totals {
		scores := scoresByTotal[total]
		summary.ScoreGroups = append(summary.ScoreGroups, data.MarkingScoreStatisticsView{
			Count: len(scores), Total: total,
			Mean: decimalStatistic(tools.Mean(scores)), Median: decimalStatistic(tools.Median(scores)), StdDev: decimalStatistic(tools.StdDev(scores)),
		})
	}
	keys := make([]pedagogicalQuestionKey, 0, len(questions))
	for key := range questions {
		keys = append(keys, key)
	}
	sort.Slice(keys, func(i, j int) bool {
		if keys[i].Family != keys[j].Family {
			return keys[i].Family < keys[j].Family
		}
		return keys[i].Version < keys[j].Version
	})
	familyNumber, versionNumber := 0, 0
	for i, key := range keys {
		// Without a historical family ID only strictly identical contents are
		// grouped. Never infer equivalence from a position or a skill tag.
		if i == 0 || key.Family == 0 || key.Family != keys[i-1].Family {
			familyNumber++
			versionNumber = 0
		}
		versionNumber++
		counter := questions[key]
		label := fmt.Sprintf("Question %d", familyNumber)
		if versionNumber > 1 || (i+1 < len(keys) && key.Family > 0 && keys[i+1].Family == key.Family) {
			label += fmt.Sprintf(" · version %d", versionNumber)
		}
		label += " — " + questionExcerpt(counter.Question)
		summary.Questions = append(summary.Questions, data.MarkingQuestionStatisticsView{
			Label: label, Count: counter.Count, Correct: counter.Correct,
			Success: decimalStatistic(tools.MarkingSuccessPercentage(float64(counter.Half)/2, counter.Total)),
		})
	}
	skills, themes := tools.AgregateThemeSkill(marks)
	for id, counter := range skills {
		if id > 0 && strings.TrimSpace(counter.Name) != "" && counter.Total > 0 {
			summary.Skills = append(summary.Skills, rateView(counter))
		}
	}
	for key, counter := range themes {
		if !strings.HasPrefix(key, "0-") && !strings.HasSuffix(key, "-0") && counter.Total > 0 {
			summary.ThemeSkills = append(summary.ThemeSkills, rateView(counter))
		}
	}
	sort.Slice(summary.Skills, func(i, j int) bool { return summary.Skills[i].Label < summary.Skills[j].Label })
	sort.Slice(summary.ThemeSkills, func(i, j int) bool { return summary.ThemeSkills[i].Label < summary.ThemeSkills[j].Label })
	return summary
}

func decimalStatistic(value float64) string {
	return strings.ReplaceAll(fmt.Sprintf("%.2f", value), ".", ",")
}

func rateView(counter config.CounterTag) data.MarkingSuccessRateView {
	return data.MarkingSuccessRateView{Label: counter.Name, Success: decimalStatistic(tools.MarkingSuccessPercentage(counter.Score, counter.Total))}
}

func pedagogicalCopyDetails(row db.ListCurrentExamResultsForGenerationRow) (config.QCM, []config.QuestionMark, bool) {
	var snapshot config.QCM
	var persisted []persistedPedagogicalQuestion
	if json.Unmarshal([]byte(row.SnapshotContent), &snapshot) != nil || json.Unmarshal([]byte(row.QuestionResults), &persisted) != nil || len(snapshot.Questions) == 0 || len(snapshot.Questions) != len(persisted) {
		return snapshot, nil, false
	}
	marks := make([]config.QuestionMark, len(snapshot.Questions))
	seen := make([]bool, len(marks))
	var half, total int64
	for _, question := range persisted {
		if question.Index < 0 || question.Index >= int64(len(marks)) || seen[question.Index] || question.Total <= 0 || snapshot.Questions[question.Index].Tags.Point.PointValue != question.Total {
			return snapshot, nil, false
		}
		state := config.Incorrect
		switch {
		case question.State == "correct" && question.Half == 2*question.Total:
			state = config.Correct
		case question.State == "partial" && question.Half == question.Total:
			state = config.Partial
		case question.State == "incorrect" && question.Half == 0:
		default:
			return snapshot, nil, false
		}
		seen[question.Index] = true
		marks[question.Index] = config.QuestionMark{Score: float64(question.Half) / 2, Total: question.Total, State: state}
		half += question.Half
		total += question.Total
	}
	return snapshot, marks, half == row.ScoreHalfUnits.Int64 && total == row.TotalPoints.Int64
}

func pedagogicalQuestionVersion(question config.Question) string {
	// Layout, checkbox symbols and answer order are randomized per pupil and
	// must not create artificial variants. Wording, image, answers and tags do.
	type answer struct {
		Content string
		State   int64
	}
	answers := make([]answer, 0, len(question.Answers))
	for _, item := range question.Answers {
		answers = append(answers, answer{Content: item.Content, State: item.State})
	}
	sort.Slice(answers, func(i, j int) bool {
		if answers[i].Content == answers[j].Content {
			return answers[i].State < answers[j].State
		}
		return answers[i].Content < answers[j].Content
	})
	canonical, _ := json.Marshal(struct {
		Tags    config.Tags
		Content string
		Image   config.Image
		Answers []answer
	}{question.Tags, question.Content, question.Image, answers})
	return fmt.Sprintf("%x", sha256.Sum256(canonical))
}

func questionExcerpt(question config.Question) string {
	text := strings.Join(strings.Fields(question.Content), " ")
	if text == "" {
		if question.Image.Name != "" {
			return "Question illustrée"
		}
		return "Énoncé non renseigné"
	}
	runes := []rune(text)
	if len(runes) > 90 {
		return string(runes[:90]) + "…"
	}
	return text
}
