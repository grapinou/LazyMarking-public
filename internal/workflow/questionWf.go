package workflow

import (
	"github.com/grapinou/LazyMarking/internal/templates/data"
	"github.com/grapinou/LazyMarking/internal/workflow/worktool"
)

func GetQuestionEmptyTableWf(baseURL string) {

	urlTested := data.DefaultDashboardRoutes.QuestionsURL
	worktool.GetTester(baseURL, urlTested, "Pas de questions pour l'instant.")

}
