package marking

import (
	"errors"
	"log"
	"net/http"
	"net/url"
	"strconv"

	"github.com/grapinou/LazyMarking/internal/db"
	"github.com/grapinou/LazyMarking/internal/handlers/tools"
	"github.com/grapinou/LazyMarking/internal/templates/data"
)

func RegenerateMarkingArtifactsHandler(w http.ResponseWriter, r *http.Request, queries *db.Queries) {
	userID, username, ok := tools.CheckRequest(w, r, http.MethodPost)
	if !ok {
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Formulaire invalide", http.StatusBadRequest)
		return
	}
	jobID, ok := parsePositiveReviewFormInt(r.FormValue("job_id"))
	if !ok {
		http.Error(w, "Formulaire invalide", http.StatusBadRequest)
		return
	}
	resultURL := data.DefaultMarkingRoutes.SuccessURL + "?job_id=" + url.QueryEscape(strconv.FormatInt(jobID, 10))
	_, err := regenerateMarkingArtifacts(r.Context(), queries, userID, username, jobID)
	if errors.Is(err, tools.ErrMarkingArtifactsUnavailable) {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		log.Printf("From RegenerateMarkingArtifactsHandler -> RegenerateMarkingArtifacts: %v", err)
		http.Redirect(w, r, resultURL+"&notice=artifacts_failed", http.StatusSeeOther)
		return
	}
	http.Redirect(w, r, resultURL, http.StatusSeeOther)
}
