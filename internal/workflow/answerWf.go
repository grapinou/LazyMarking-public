package workflow

import (
	"github.com/grapinou/LazyMarking/internal/templates/data"
	"github.com/grapinou/LazyMarking/internal/workflow/worktool"
)

func AnswerWf(baseURL string) {

	answers := []worktool.AnswerStructWf{
		{QuestionID: "1", State: "O", Content: "bleu"},
		{QuestionID: "1", State: "O", Content: "rose"},
		{QuestionID: "1", State: "1", Content: "jaune"},
		{QuestionID: "1", State: "O", Content: "vert"},
		{QuestionID: "2", State: "O", Content: "100g"},
		{QuestionID: "2", State: "O", Content: "100 kg"},
		{QuestionID: "2", State: "O", Content: "100 000 000 kg"},
		{QuestionID: "2", State: "1", Content: "7,36 × 10^22 kg"},
		{QuestionID: "3", State: "0", Content: "largeur x hauteur"},
		{QuestionID: "3", State: "0", Content: "base x hauteur divisé par deux"},
		{QuestionID: "3", State: "0", Content: "pi x r²"},
		{QuestionID: "3", State: "1", Content: "côté x côté"},
	}

	worktool.AnswerFiller(
		baseURL,
		data.DefaultQuestionRoutes.AnswersURL,
		data.DefaultAnswerRoutes.AddURL,
		"Pas de réponses pour l'instant",
		"Ajouter Réponse",
		4,
		answers,
	)
}
