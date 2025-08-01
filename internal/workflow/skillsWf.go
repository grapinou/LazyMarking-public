package workflow

import (
	"github.com/grapinou/LazyMarking/internal/templates/data"
	"github.com/grapinou/LazyMarking/internal/workflow/worktool"
)

func SkillsWf(baseURL string) {
	worktool.QuestionFeature(
		"skills",
		baseURL,
		data.DefaultQuestionRoutes.SkillsURL,
		data.DefaultSkillRoutes.AddURL,
		"skill",
		"Savoir",
		"Pas de compétences pour l'instant",
		"Ajouter compétence",
	)

	worktool.QuestionFeature(
		"skills",
		baseURL,
		data.DefaultQuestionRoutes.SkillsURL,
		data.DefaultSkillRoutes.AddURL,
		"skill",
		"Réaliser",
		"Ajouter une compétence",
		"Ajouter compétence",
	)

	worktool.QuestionFeature(
		"skills",
		baseURL,
		data.DefaultQuestionRoutes.SkillsURL,
		data.DefaultSkillRoutes.AddURL,
		"skill",
		"Analyser",
		"Ajouter une compétence",
		"Ajouter compétence",
	)
}
