package workflow

import (
	"github.com/grapinou/LazyMarking/internal/templates/data"
	"github.com/grapinou/LazyMarking/internal/workflow/worktool"
)

func ThemesWf(baseURL string) {
	worktool.QuestionFeature(
		"themes",
		baseURL,
		data.DefaultQuestionRoutes.ThemesURL,
		data.DefaultThemeRoutes.AddURL,
		"theme",
		"Atome",
		"Pas de thèmes pour l'instant",
		"Ajouter thème",
	)

	worktool.QuestionFeature(
		"themes",
		baseURL,
		data.DefaultQuestionRoutes.ThemesURL,
		data.DefaultThemeRoutes.AddURL,
		"theme",
		"Mouvement",
		"Ajouter un thème",
		"Ajouter thème",
	)

	worktool.QuestionFeature(
		"themes",
		baseURL,
		data.DefaultQuestionRoutes.ThemesURL,
		data.DefaultThemeRoutes.AddURL,
		"theme",
		"Mesure",
		"Ajouter un thème",
		"Ajouter thème",
	)
}
