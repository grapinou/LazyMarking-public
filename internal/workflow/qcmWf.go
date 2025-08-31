package workflow

import (
	"github.com/grapinou/LazyMarking/internal/templates/data"
	"github.com/grapinou/LazyMarking/internal/workflow/worktool"
)

func QcmWf(baseURL string) {
	worktool.QcmFiller(
		"QCM",
		baseURL,
		data.DefaultDashboardRoutes.QcmURL,
		data.DefaultQCMRoutes.AddURL,
		"qcm",
		"mvt",
	)
}
