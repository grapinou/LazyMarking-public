package workflow

import (
	"github.com/grapinou/LazyMarking/internal/templates/data"
	"github.com/grapinou/LazyMarking/internal/workflow/worktool"
)

func ExamWf(baseURL string) {
	worktool.ExamFiller(
		"Exam",
		baseURL,
		data.DefaultDashboardRoutes.ExamURL,
		data.DefaultExamRoutes.AddURL,
	)
}
