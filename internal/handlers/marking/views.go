package marking

import (
	"net/http"

	"github.com/grapinou/LazyMarking/internal/handlers/tools"
	"github.com/grapinou/LazyMarking/internal/templates/data"
)

func RenderAddPdfFormMarkingPage(w http.ResponseWriter, dataPage data.MarkingPageData) {
	tools.RenderMergeTemplate(w, dataPage,
		data.DefaultDashboarPath,
		data.DefaultDashboardName,
		data.DefaultMarkingPathTemplate,
		data.DefaultMarkingTemplateName.Table)
}

func RenderProgressMarkingPage(w http.ResponseWriter, dataPage data.MarkingPageData) {
	tools.RenderMergeTemplate(w, dataPage,
		data.DefaultDashboarPath,
		data.DefaultDashboardName,
		data.DefaultMarkingPathTemplate,
		data.DefaultMarkingTemplateName.Progress)
}

func RenderSuccessProgressMarkingPage(w http.ResponseWriter, dataPage data.MarkingResultPageData) {
	tools.RenderMergeTemplate(w, dataPage,
		data.DefaultDashboarPath,
		data.DefaultDashboardName,
		data.DefaultMarkingPathTemplate,
		data.DefaultMarkingTemplateName.Success)
}

func RenderMarkingReviewPage(w http.ResponseWriter, dataPage data.MarkingReviewPageData) {
	tools.RenderMergeTemplate(w, dataPage,
		data.DefaultDashboarPath,
		data.DefaultDashboardName,
		data.DefaultMarkingPathTemplate,
		data.DefaultMarkingTemplateName.Review)
}
