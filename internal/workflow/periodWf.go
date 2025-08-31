package workflow

import (
	"github.com/grapinou/LazyMarking/internal/templates/data"
	"github.com/grapinou/LazyMarking/internal/workflow/worktool"
)

func PeriodWf(baseURL string) {
	worktool.QcmFiller(
		"Period",
		baseURL,
		data.DefaultExamRoutes.PeriodsURL,
		data.DefaultPeriodRoutes.AddURL,
		"period",
		"1er trimestre",
	)
}
