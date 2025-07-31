package workflow

import (
	"github.com/grapinou/LazyMarking/internal/templates/data"
	"github.com/grapinou/LazyMarking/internal/workflow/worktool"
)

func SubjectsWf(baseURL string) {
	worktool.QuestionFeature(
		"subjects",
		baseURL,
		data.DefaultQuestionRoutes.SubjectsURL,
		data.DefaultSubjectRoutes.AddURL,
		"subject",
		"Physique",
		"Pas de matières pour l'instant",
		"Ajouter matière",
	)

	worktool.QuestionFeature(
		"subjects",
		baseURL,
		data.DefaultQuestionRoutes.SubjectsURL,
		data.DefaultSubjectRoutes.AddURL,
		"subject",
		"Chimie",
		"Ajouter une matière",
		"Ajouter matière",
	)

	worktool.QuestionFeature(
		"subjects",
		baseURL,
		data.DefaultQuestionRoutes.SubjectsURL,
		data.DefaultSubjectRoutes.AddURL,
		"subject",
		"Cacalogie",
		"Ajouter une matière",
		"Ajouter matière",
	)
}

/*
func SubjectWf(baseURL string) {

	urlTested := data.DefaultQuestionRoutes.SubjectsURL
	worktool.GetTester(baseURL, urlTested, "Ajouter une matière")

	urlTested = data.DefaultSubjectRoutes.AddURL
	worktool.GetTester(baseURL, urlTested, "Ajouter matière")

	fields := map[string]string{
		"subject": "Physique",
	}
	worktool.PostTesterWF(baseURL, urlTested, fields)

}
*/
