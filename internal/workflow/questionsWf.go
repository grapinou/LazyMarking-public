package workflow

import (
	"github.com/grapinou/LazyMarking/internal/templates/data"
	"github.com/grapinou/LazyMarking/internal/workflow/worktool"
)

func QuestionsWf(baseURL string) {
	maxField := 3

	worktool.QuestionFiller(
		baseURL,
		data.DefaultDashboardRoutes.QuestionsURL,
		data.DefaultQuestionRoutes.AddURL,
		"Pas de questions pour l'instant.",
		"Ajouter question",
		"Première question de test : la couleur du soleil ?",
		maxField,
	)

	worktool.QuestionFiller(
		baseURL,
		data.DefaultDashboardRoutes.QuestionsURL,
		data.DefaultQuestionRoutes.AddURL,
		"Créer et paramétrer les questions",
		"Ajouter question",
		"Deuxième question de test : la masse de la lune ?",
		maxField,
	)

	worktool.QuestionFiller(
		baseURL,
		data.DefaultDashboardRoutes.QuestionsURL,
		data.DefaultQuestionRoutes.AddURL,
		"Créer et paramétrer les questions",
		"Ajouter question",
		"Troisième question de test : la surface d'un carré ?",
		maxField,
	)
}
