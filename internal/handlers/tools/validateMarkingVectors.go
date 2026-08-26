package tools

import (
	"fmt"

	"github.com/grapinou/LazyMarking/internal/config"
)

func validateMarkingVectors(qcm config.QCM, homoPages []config.HomoPage, answersState []int) error {
	expectedQuestions := len(qcm.Questions)
	expectedAnswers := 0
	for _, question := range qcm.Questions {
		expectedAnswers += len(question.Answers)
	}

	pageQuestions := 0
	pageAnswers := 0
	for _, page := range homoPages {
		pageQuestions += len(page.Content.Questions)
		pageAnswers += len(page.Content.Answers)
	}

	recognizedAnswers := len(answersState)
	if pageQuestions != expectedQuestions || pageAnswers != expectedAnswers || recognizedAnswers != expectedAnswers {
		return fmt.Errorf(
			"marking vector counts mismatch: expected questions=%d, page questions=%d, expected answers=%d, page answers=%d, recognized answers=%d",
			expectedQuestions,
			pageQuestions,
			expectedAnswers,
			pageAnswers,
			recognizedAnswers,
		)
	}

	for index, state := range answersState {
		if state != 0 && state != 1 {
			return fmt.Errorf("recognized answer state at index %d is %d, want 0 or 1", index, state)
		}
	}
	for questionIndex, question := range qcm.Questions {
		for answerIndex, answer := range question.Answers {
			if answer.State != 0 && answer.State != 1 {
				return fmt.Errorf(
					"expected answer state at question %d answer %d is %d, want 0 or 1",
					questionIndex,
					answerIndex,
					answer.State,
				)
			}
		}
	}

	return nil
}
