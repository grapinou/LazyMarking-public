package workflow

import (
	"github.com/grapinou/LazyMarking/internal/templates/data"
	"github.com/grapinou/LazyMarking/internal/workflow/worktool"
)

func YearLevelWf(baseURL string) {
	worktool.QuestionFeature(
		"year levels",
		baseURL,
		data.DefaultQuestionRoutes.YearLevelsURL,
		data.DefaultYearLevelRoutes.AddURL,
		"yearlevel",
		"6ème",
		"Pas de classes pour l'instant",
		"Ajouter classe",
	)

	worktool.QuestionFeature(
		"year levels",
		baseURL,
		data.DefaultQuestionRoutes.YearLevelsURL,
		data.DefaultYearLevelRoutes.AddURL,
		"yearlevel",
		"5ème",
		"Ajouter une classe",
		"Ajouter classe",
	)

	worktool.QuestionFeature(
		"year levels",
		baseURL,
		data.DefaultQuestionRoutes.YearLevelsURL,
		data.DefaultYearLevelRoutes.AddURL,
		"yearlevel",
		"seconde",
		"Ajouter une classe",
		"Ajouter classe",
	)
}
