package tools

import (
	"fmt"
	"net/http"

	"github.com/grapinou/LazyMarking/internal/config"
	"github.com/grapinou/LazyMarking/internal/mathcontent"
)

func ValidateQuestionText(w http.ResponseWriter, label, value string) bool {
	if _, err := mathcontent.Parse(value); err != nil {
		http.Error(w, label+" : "+err.Error(), http.StatusUnprocessableEntity)
		return false
	}
	return true
}

func ValidateMathQCM(qcm config.QCM) error {
	for i, question := range qcm.Questions {
		for _, field := range []struct{ label, value string }{
			{"énoncé", question.Content},
			{"consigne", question.Instruction},
		} {
			if _, err := mathcontent.Parse(field.value); err != nil {
				return fmt.Errorf("question %d, %s : %w", i+1, field.label, err)
			}
		}
		for j, answer := range question.Answers {
			if _, err := mathcontent.Parse(answer.Content); err != nil {
				return fmt.Errorf("question %d, réponse %d : %w", i+1, j+1, err)
			}
		}
	}
	return nil
}
