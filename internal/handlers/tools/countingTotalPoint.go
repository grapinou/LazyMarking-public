package tools

import "github.com/grapinou/LazyMarking/internal/config"

func CountingTotalPoint(questionsState []config.QuestionMark) (float64, int) {
	var mark float64
	var tot int
	for _, question := range questionsState {
		mark += question.Score
		tot += int(question.Total)
	}

	return mark, tot
}
