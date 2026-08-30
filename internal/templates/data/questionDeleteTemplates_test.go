package data

import (
	"os"
	"strings"
	"testing"
)

func TestQuestionBankDeleteTemplatesKeepFormContractsAndModernWording(t *testing.T) {
	tests := []struct {
		name       string
		path       string
		action     string
		fields     []string
		cancel     string
		title      string
		buttonText string
	}{
		{"question", "../questions/delete_form_question.html", `.QuestionRoutes.DeleteURL`, []string{`name="question_id"`}, `.ExtraData.CancelURL`, "Supprimer la question", "Supprimer la question"},
		{"answer", "../answers/delete_form_answer.html", `.AnswerRoutes.DeleteURL`, []string{`name="answer_id"`, `name="question_id"`}, `.ExtraData.CancelURL`, "Supprimer la réponse", "Supprimer la réponse"},
		{"variant", "../altquestions/delete_form_alt_question.html", `.AltQuestionRoutes.DeleteURL`, []string{`name="question_id"`, `name="alt_question_id"`}, `.ExtraData.CancelURL`, "Supprimer la variante", "Supprimer la variante"},
		{"variant answer", "../altanswers/delete_form_alt_answer.html", `.AltAnswerRoutes.DeleteURL`, []string{`name="question_id"`, `name="alt_question_id"`, `name="alt_answer_id"`}, `.ExtraData.CancelURL`, "Supprimer la réponse de la variante", "Supprimer la réponse"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			content, err := os.ReadFile(test.path)
			if err != nil {
				t.Fatal(err)
			}
			template := string(content)
			for _, required := range append([]string{test.action, test.cancel, test.title, test.buttonText, ">Annuler<"}, test.fields...) {
				if !strings.Contains(template, required) {
					t.Fatalf("template missing contract %q", required)
				}
			}
			for _, obsolete := range []string{"C'est mon dernier mot", "Es-tu sur de supprimer", "Supprimer ?"} {
				if strings.Contains(template, obsolete) {
					t.Fatalf("template still contains obsolete wording %q", obsolete)
				}
			}
		})
	}
}
