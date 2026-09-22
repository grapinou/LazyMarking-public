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

type pedagogicalCounter struct {
	Label    string
	Question config.Question
	Count    int
	Correct  int
	Half     int64
	Total    int64
}

type pedagogicalFamilyCounter struct {
	pedagogicalCounter
	Key      string
	Variants map[string]*pedagogicalCounter
}

// The input is the very same current-result snapshot used by the HTML table.
// No job selection or answer rescoring takes place here.
func buildMarkingPedagogicalSummary(rows []db.ListCurrentExamResultsForGenerationRow) data.MarkingPedagogicalSummaryView {
	summary := data.MarkingPedagogicalSummaryView{}
	scoresByTotal := make(map[int64][]float64)
	families := make(map[string]*pedagogicalFamilyCounter)
	themes := make(map[int64]*pedagogicalCounter)
	skills := make(map[int64]*pedagogicalCounter)
	themeSkills := make(map[string]*pedagogicalCounter)
	var overallHalf, overallTotal int64

	for _, row := range rows {
		if !hasFinalMarkingScore(row) || row.TotalPoints.Int64 <= 0 {
			summary.ExcludedCopies++
			continue
		}
		summary.IncludedCopies++
		overallHalf += row.ScoreHalfUnits.Int64
		overallTotal += row.TotalPoints.Int64
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
			mark := questionMarks[index]
			familyKey := pedagogicalFamilyKey(question)
			family := families[familyKey]
			if family == nil {
				family = &pedagogicalFamilyCounter{
					pedagogicalCounter: pedagogicalCounter{Question: question},
					Key:                familyKey, Variants: make(map[string]*pedagogicalCounter),
				}
				families[familyKey] = family
			}
			addPedagogicalResult(&family.pedagogicalCounter, question, mark)

			variantKey := pedagogicalVariantKey(question)
			variant := family.Variants[variantKey]
			if variant == nil {
				variant = &pedagogicalCounter{Question: question}
				family.Variants[variantKey] = variant
			}
			addPedagogicalResult(variant, question, mark)

			addClassifiedCounter(themes, question.Tags.Theme.ID, question.Tags.Theme.Name, mark)
			addClassifiedCounter(skills, question.Tags.Skill.ID, question.Tags.Skill.Name, mark)
			if question.Tags.Theme.ID > 0 && strings.TrimSpace(question.Tags.Theme.Name) != "" &&
				question.Tags.Skill.ID > 0 && strings.TrimSpace(question.Tags.Skill.Name) != "" {
				key := fmt.Sprintf("%020d-%020d", question.Tags.Theme.ID, question.Tags.Skill.ID)
				addNamedCounter(themeSkills, key, question.Tags.Theme.Name+" — "+question.Tags.Skill.Name, mark)
			}
		}
	}

	if overallTotal > 0 {
		summary.HasOverall = true
		summary.Overall = successRateView("Réussite globale", overallHalf, overallTotal)
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

	orderedFamilies := make([]*pedagogicalFamilyCounter, 0, len(families))
	for _, family := range families {
		orderedFamilies = append(orderedFamilies, family)
	}
	sort.Slice(orderedFamilies, func(i, j int) bool {
		return orderedFamilies[i].Key < orderedFamilies[j].Key
	})
	for familyIndex, family := range orderedFamilies {
		variants := make([]*pedagogicalCounter, 0, len(family.Variants))
		for _, variant := range family.Variants {
			variants = append(variants, variant)
		}
		sort.Slice(variants, func(i, j int) bool {
			return pedagogicalVariantKey(variants[i].Question) < pedagogicalVariantKey(variants[j].Question)
		})

		familyLabel := fmt.Sprintf("Question %d", familyIndex+1)
		if len(variants) == 1 {
			familyLabel += " — " + questionExcerpt(variants[0].Question)
		}
		familyView := questionRateView(familyLabel, &family.pedagogicalCounter)
		for variantIndex, variant := range variants {
			label := fmt.Sprintf("Question %d · version %d — %s", familyIndex+1, variantIndex+1, questionExcerpt(variant.Question))
			variantView := questionRateView(label, variant)
			summary.Questions = append(summary.Questions, variantView)
			if len(variants) > 1 {
				familyView.Variants = append(familyView.Variants, variantView)
			}
		}
		summary.QuestionFamilies = append(summary.QuestionFamilies, familyView)
	}

	summary.Themes = orderedRateViews(themes)
	summary.Skills = orderedRateViews(skills)
	summary.ThemeSkills = orderedNamedRateViews(themeSkills)
	return summary
}

func addPedagogicalResult(counter *pedagogicalCounter, question config.Question, mark config.QuestionMark) {
	if questionExcerpt(question) < questionExcerpt(counter.Question) {
		counter.Question = question
	}
	counter.Count++
	counter.Half += int64(mark.Score * 2)
	counter.Total += mark.Total
	if mark.State == config.Correct {
		counter.Correct++
	}
}

func addClassifiedCounter(counters map[int64]*pedagogicalCounter, id int64, label string, mark config.QuestionMark) {
	if id <= 0 || strings.TrimSpace(label) == "" {
		return
	}
	counter := counters[id]
	if counter == nil {
		counter = &pedagogicalCounter{Label: label}
		counters[id] = counter
	}
	if label < counter.Label {
		counter.Label = label
	}
	counter.Half += int64(mark.Score * 2)
	counter.Total += mark.Total
}

func addNamedCounter(counters map[string]*pedagogicalCounter, key, label string, mark config.QuestionMark) {
	if strings.TrimSpace(label) == "" {
		return
	}
	counter := counters[key]
	if counter == nil {
		counter = &pedagogicalCounter{Label: label}
		counters[key] = counter
	}
	if label < counter.Label {
		counter.Label = label
	}
	counter.Half += int64(mark.Score * 2)
	counter.Total += mark.Total
}

func orderedRateViews(counters map[int64]*pedagogicalCounter) []data.MarkingSuccessRateView {
	rows := make([]data.MarkingSuccessRateView, 0, len(counters))
	for _, counter := range counters {
		if counter.Total > 0 {
			rows = append(rows, successRateView(counter.Label, counter.Half, counter.Total))
		}
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i].Label < rows[j].Label })
	return rows
}

func orderedNamedRateViews(counters map[string]*pedagogicalCounter) []data.MarkingSuccessRateView {
	rows := make([]data.MarkingSuccessRateView, 0, len(counters))
	for _, counter := range counters {
		if counter.Total > 0 {
			rows = append(rows, successRateView(counter.Label, counter.Half, counter.Total))
		}
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i].Label < rows[j].Label })
	return rows
}

func decimalStatistic(value float64) string {
	return strings.ReplaceAll(fmt.Sprintf("%.2f", value), ".", ",")
}

func successRateView(label string, half, total int64) data.MarkingSuccessRateView {
	percentage := tools.MarkingSuccessPercentage(float64(half)/2, total)
	indicator, level, badge := pedagogicalClassification(percentage)
	return data.MarkingSuccessRateView{
		Label: label, Success: decimalStatistic(percentage), SuccessPercent: percentage,
		Indicator: indicator, LevelLabel: level, BadgeClass: badge,
	}
}

func questionRateView(label string, counter *pedagogicalCounter) data.MarkingQuestionStatisticsView {
	rate := successRateView(label, counter.Half, counter.Total)
	return data.MarkingQuestionStatisticsView{
		Label: label, Count: counter.Count, Correct: counter.Correct, Success: rate.Success, SuccessPercent: rate.SuccessPercent,
		Indicator: rate.Indicator, LevelLabel: rate.LevelLabel, BadgeClass: rate.BadgeClass,
	}
}

func pedagogicalClassification(percentage float64) (indicator, label, badge string) {
	switch {
	case percentage < 40:
		return "🔴", "À retravailler", "text-bg-danger"
	case percentage <= 60:
		return "🟡", "Intermédiaire", "text-bg-warning"
	default:
		return "🟢", "Maîtrisé", "text-bg-success"
	}
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

func pedagogicalFamilyKey(question config.Question) string {
	if question.Tags.MainQuestionID > 0 {
		return fmt.Sprintf("question:%020d", question.Tags.MainQuestionID)
	}
	// A missing historical family ID must never cause unrelated questions to be
	// merged merely because they occupied the same position.
	return "legacy:" + pedagogicalQuestionVersion(question)
}

func pedagogicalVariantKey(question config.Question) string {
	if question.Tags.VariantID > 0 && (question.Tags.VariantType == config.MainQuestion || question.Tags.VariantType == config.AltQuestion) {
		return fmt.Sprintf("variant:%s:%020d", question.Tags.VariantType, question.Tags.VariantID)
	}
	// Pre-P3 snapshots have no source variant reference. Their immutable wording,
	// image, answers and tags form a conservative identity: false splitting is
	// acceptable, false merging is not.
	return "snapshot:" + pedagogicalQuestionVersion(question)
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
