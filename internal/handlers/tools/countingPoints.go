package tools

import (
	"github.com/grapinou/LazyMarking/internal/config"
	"github.com/grapinou/LazyMarking/internal/markingscoring"
)

func CountingPoints(qcm config.QCM, answersState []int) []config.QuestionMark {
	var questionsState []config.QuestionMark

	for _, question := range qcm.Questions {
		expectingState := make([]int, 0, len(question.Answers))
		for _, answer := range question.Answers {
			expectingState = append(expectingState, int(answer.State))
		}
		consumed := min(len(answersState), len(expectingState))
		questionsState = append(questionsState, markingscoring.ScoreQuestion(
			expectingState,
			answersState[:consumed],
			question.Tags.Point.PointValue,
		))
		answersState = answersState[consumed:]
	}
	return questionsState
}
