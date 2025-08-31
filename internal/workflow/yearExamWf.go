package workflow

import (
	"github.com/grapinou/LazyMarking/internal/templates/data"
	"github.com/grapinou/LazyMarking/internal/workflow/worktool"
)

func YearExamWf(baseURL string) {
	worktool.QcmFiller(
		"YearExam",
		baseURL,
		data.DefaultExamRoutes.YearsURL,
		data.DefaultYearRoutes.AddURL,
		"year",
		"2025-2016",
	)
}
