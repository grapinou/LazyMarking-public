package workflow

import (
	"github.com/grapinou/LazyMarking/internal/templates/data"
	"github.com/grapinou/LazyMarking/internal/workflow/worktool"
)

func PointsWf(baseURL string) {
	worktool.QuestionFeature(
		"points",
		baseURL,
		data.DefaultQuestionRoutes.PointsURL,
		data.DefaultPointRoutes.AddURL,
		"point",
		"1",
		"Pas de points pour l'instant",
		"Ajouter point",
	)

	worktool.QuestionFeature(
		"points",
		baseURL,
		data.DefaultQuestionRoutes.PointsURL,
		data.DefaultPointRoutes.AddURL,
		"point",
		"2",
		"Ajouter un nombre de point",
		"Ajouter point",
	)

	worktool.QuestionFeature(
		"points",
		baseURL,
		data.DefaultQuestionRoutes.PointsURL,
		data.DefaultPointRoutes.AddURL,
		"point",
		"3",
		"Ajouter un nombre de point",
		"Ajouter point",
	)
}
