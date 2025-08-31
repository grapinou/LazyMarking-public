package workflow

import (
	"github.com/grapinou/LazyMarking/internal/templates/data"
	"github.com/grapinou/LazyMarking/internal/workflow/worktool"
)

func StudentsWf(baseURL string) {
	maxClassCode := 1

	worktool.StudentFiller(baseURL,
		data.DefaultDashboardRoutes.StudentURL,
		data.DefaultStudentRoutes.AddURL,
		maxClassCode)
}
