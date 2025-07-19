package errorsmessages

import (
	"net/http"

	"github.com/grapinou/LazyMarking/internal/handlers/tools"
	"github.com/grapinou/LazyMarking/internal/templates/data"
)

func RenderErrorQuestionFeaturePage(w http.ResponseWriter, dataPage data.ErrorPageData) {
	tools.RenderMergeTemplate(w, dataPage,
		data.DefaultDashboarPath,
		data.DefaultDashboardName,
		data.ErrorPathTemplate,
		data.ErrorTemplateName,
	)
}
