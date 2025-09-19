package tools

import (
	"slices"

	"github.com/grapinou/LazyMarking/internal/config"
)

func CountingPoints(qcm config.QCM, answersState []int) []config.QuestionMark {
	var questionsState []config.QuestionMark

	for _, question := range qcm.Questions {
		var expectingState []int
		for _, answer := range question.Answers {
			expectingState = append(expectingState, int(answer.State))
		}

		// normalement ok
		toCorrect := answersState[:len(expectingState)]
		answersState = answersState[len(expectingState):]

		if slices.Equal(expectingState, toCorrect) {
			questionsState = append(questionsState, config.QuestionMark{
				Score: float64(question.Tags.Point.PointValue),
				Total: question.Tags.Point.PointValue,
				State: config.Correct,
			})
		} else {

			// sur ce qui est attendu
			setAnswers := make(map[int]struct{})
			var nbrCorrectAnswers int
			for i, s := range expectingState {
				if s == 1 {
					setAnswers[i] = struct{}{}
					nbrCorrectAnswers += 1
				}
			}

			// sur ce qui est répondu
			var answered []int
			for i, s := range toCorrect {
				if s == 1 {
					answered = append(answered, i)
				}
			}

			if len(answered) > 0 && len(answered) <= nbrCorrectAnswers {
				partial := true
				foundCorrect := false

				for _, ans := range answered {
					if _, ok := setAnswers[ans]; ok {
						foundCorrect = true // au moins une bonne réponse
					} else {
						partial = false // une mauvaise réponse -> plus de partiel
					}
				}

				if foundCorrect && partial {
					questionsState = append(questionsState, config.QuestionMark{
						Score: float64(question.Tags.Point.PointValue / 2),
						Total: question.Tags.Point.PointValue,
						State: config.Partial,
					})
				} else {
					questionsState = append(questionsState, config.QuestionMark{
						Score: float64(0),
						Total: question.Tags.Point.PointValue,
						State: config.Incorrect,
					})
				}
			} else {
				questionsState = append(questionsState, config.QuestionMark{
					Score: float64(0),
					Total: question.Tags.Point.PointValue,
					State: config.Incorrect,
				})
			}
		}
	}
	return questionsState
}
