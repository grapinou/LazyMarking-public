package errorsmessages

import (
	"net/http"

	"github.com/grapinou/LazyMarking/internal/handlers/tools"
	"github.com/grapinou/LazyMarking/internal/templates/data"
)

func ErrorQuestionFeatureHandler(w http.ResponseWriter, r *http.Request) {
	_, _, ok := tools.CheckRequest(w, r, http.MethodGet)
	if !ok {
		return
	}

	errorMessage := r.URL.Query().Get("errormessage")
	if errorMessage == "" {
		errorMessage = "Une erreur inattendue est survenue."
	}

	dataPage := data.ErrorPageData{
		Routes:    data.DefaultDashboardRoutes,
		PageTitle: "error page",
		ExtraData: map[string]any{
			"ErrorMessage": errorMessage,
		},
	}

	RenderErrorQuestionFeaturePage(w, dataPage)
}
