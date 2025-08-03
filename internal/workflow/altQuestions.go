package workflow

import (
	"github.com/grapinou/LazyMarking/internal/templates/data"
	"github.com/grapinou/LazyMarking/internal/workflow/worktool"
)

func AltQuestionWf(baseURL string) {
	altQuestions := []worktool.AltQuestionStructWf{
		{QuestionID: "1", Content: "Quelle est la couleur de l'eau ?"},
		{QuestionID: "1", Content: "Quelle est la couleur de la neige ?"},
		{QuestionID: "1", Content: "Quelle est la couleur de l'herbe ?"},
		{QuestionID: "2", Content: "Quelle est la masse de la tour Eiffel ?"},
		{QuestionID: "2", Content: "Quelle est la masse de la Terre ?"},
		{QuestionID: "2", Content: "Quelle est la masse de la tour Burj Khalifa ?"},
		{QuestionID: "3", Content: "Quelle est la surface d'un cercle ?"},
		{QuestionID: "3", Content: "Quelle est la surface d'un triangle ?"},
		{QuestionID: "3", Content: "Quelle est la surface d'un rectangle ?"},
	}

	worktool.AltQuestionFiller(baseURL,
		data.DefaultQuestionRoutes.AltQuestionsURL,
		data.DefaultAltQuestionRoutes.AddURL,
		"Pas de questions alternatives pour l'instant",
		"Ajouter une question alternative",
		3,
		altQuestions,
	)
}
