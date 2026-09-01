package markingscoring

import (
	"slices"

	"github.com/grapinou/LazyMarking/internal/config"
)

// ScoreQuestion is the single pure scoring rule shared by automatic marking
// and human-review recalculation.
func ScoreQuestion(expected, actual []int, totalPoints int64) config.QuestionMark {
	normalized := make([]int, len(expected))
	copy(normalized, actual[:min(len(actual), len(expected))])
	if slices.Equal(expected, normalized) {
		return config.QuestionMark{Score: float64(totalPoints), Total: totalPoints, State: config.Correct}
	}

	correctAnswers := make(map[int]struct{})
	for index, state := range expected {
		if state == 1 {
			correctAnswers[index] = struct{}{}
		}
	}
	selected := make([]int, 0)
	for index, state := range normalized {
		if state == 1 {
			selected = append(selected, index)
		}
	}
	if len(selected) > 0 && len(selected) <= len(correctAnswers) {
		allCorrect := true
		for _, index := range selected {
			if _, ok := correctAnswers[index]; !ok {
				allCorrect = false
				break
			}
		}
		if allCorrect {
			return config.QuestionMark{Score: float64(totalPoints) / 2, Total: totalPoints, State: config.Partial}
		}
	}
	return config.QuestionMark{Score: 0, Total: totalPoints, State: config.Incorrect}
}
