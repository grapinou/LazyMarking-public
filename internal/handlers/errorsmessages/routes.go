package errorsmessages

import (
	"net/http"

	"github.com/grapinou/LazyMarking/internal/handlers/login"
	"github.com/grapinou/LazyMarking/internal/templates/data"
)

func RegisterRoutes(mux *http.ServeMux) {
	mux.Handle("GET "+data.ErrorMessageURL,
		login.CheckAuth(http.HandlerFunc(ErrorQuestionFeatureHandler)),
	)
}
