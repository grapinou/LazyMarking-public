package marking

import (
	"sort"

	"github.com/grapinou/LazyMarking/internal/config"
	"github.com/grapinou/LazyMarking/internal/db"
)

// TrainingCandidates consumes P3's current-result coverage and family classification.
// Every observed variant is available in the draft; weak families start selected.
type TrainingCandidate struct {
	Question config.Question
	Weak     bool
}

func TrainingCandidates(rows []db.ListCurrentExamResultsForGenerationRow) []TrainingCandidate {
	summary := buildMarkingPedagogicalSummary(rows)
	familyKeys := make([]string, 0)
	byFamily := map[string]map[string]config.Question{}
	for _, row := range rows {
		if !hasFinalMarkingScore(row) {
			continue
		}
		qcm, _, ok := pedagogicalCopyDetails(row)
		if !ok {
			continue
		}
		for _, question := range qcm.Questions {
			family := pedagogicalFamilyKey(question)
			if byFamily[family] == nil {
				byFamily[family] = map[string]config.Question{}
				familyKeys = append(familyKeys, family)
			}
			key := pedagogicalVariantKey(question)
			if old, exists := byFamily[family][key]; !exists || pedagogicalQuestionVersion(question) < pedagogicalQuestionVersion(old) {
				byFamily[family][key] = question
			}
		}
	}
	sort.Strings(familyKeys)
	result := make([]TrainingCandidate, 0)
	for i, family := range familyKeys {
		weak := i < len(summary.QuestionFamilies) && summary.QuestionFamilies[i].LevelLabel == "À retravailler"
		keys := make([]string, 0, len(byFamily[family]))
		for key := range byFamily[family] {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		for _, key := range keys {
			result = append(result, TrainingCandidate{Question: byFamily[family][key], Weak: weak})
		}
	}
	return result
}
