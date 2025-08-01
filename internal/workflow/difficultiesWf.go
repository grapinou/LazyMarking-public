package workflow

import (
	"github.com/grapinou/LazyMarking/internal/templates/data"
	"github.com/grapinou/LazyMarking/internal/workflow/worktool"
)

func DifficultiesWf(baseURL string) {
	worktool.QuestionFeature(
		"difficulties",
		baseURL,
		data.DefaultQuestionRoutes.DifficultiesURL,
		data.DefaultDifficultyRoutes.AddURL,
		"difficulty",
		"Facile",
		"Pas de difficultées pour l'instant",
		"Ajouter difficultée",
	)

	worktool.QuestionFeature(
		"difficulties",
		baseURL,
		data.DefaultQuestionRoutes.DifficultiesURL,
		data.DefaultDifficultyRoutes.AddURL,
		"difficulty",
		"Moyen",
		"Ajouter une difficultée",
		"Ajouter difficultée",
	)

	worktool.QuestionFeature(
		"difficulties",
		baseURL,
		data.DefaultQuestionRoutes.DifficultiesURL,
		data.DefaultDifficultyRoutes.AddURL,
		"difficulty",
		"Difficile",
		"Ajouter une difficultée",
		"Ajouter difficultée",
	)
}
