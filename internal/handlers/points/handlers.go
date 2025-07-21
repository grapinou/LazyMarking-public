package points

import (
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/grapinou/LazyMarking/internal/db"
	"github.com/grapinou/LazyMarking/internal/handlers/tools"
	"github.com/grapinou/LazyMarking/internal/templates/data"
)

func TablePointsHandler(w http.ResponseWriter, r *http.Request, queries *db.Queries) {
	userID, _, ok := tools.CheckRequest(w, r, http.MethodGet)
	if !ok {
		return
	}

	pointsDB, err := queries.GetAllPoints(r.Context(), userID)
	if err != nil {
		log.Printf("From TablePointsHandler : GetAllSkills DB error: %v", err)
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	noPoint := true
	if len(pointsDB) > 0 {
		noPoint = false
	}

	var actionsURLParameters []data.PointActionURLs
	if !noPoint {
		for _, point := range pointsDB {
			editURL := data.DefaultPointRoutes.EditURL + "?point_id=" + url.QueryEscape(strconv.FormatInt(point.ID, 10))
			deleteURL := data.DefaultPointRoutes.DeleteURL + "?point_id=" + url.QueryEscape(strconv.FormatInt(point.ID, 10))

			actionsURLParameters = append(actionsURLParameters, data.PointActionURLs{
				EditURL:   editURL,
				DeleteURL: deleteURL,
			})
		}
	}

	dataPage := data.PointPageData{
		Routes:      data.DefaultDashboardRoutes,
		PointRoutes: data.DefaultPointRoutes,
		PageTitle:   "points",
		ExtraData: map[string]any{
			"NoPoint": noPoint,
			"Action":  actionsURLParameters,
			"Points":  pointsDB,
		},
	}

	RenderTablePointPage(w, dataPage)
}

func AddFormPointHandler(w http.ResponseWriter, r *http.Request, queries *db.Queries) {
	_, _, ok := tools.CheckRequest(w, r, http.MethodGet)
	if !ok {
		return
	}

	seq := make([]int, 0, 100)
	for i := 1; i <= 100; i++ {
		seq = append(seq, i)
	}

	dataPage := data.PointPageData{
		Routes:      data.DefaultDashboardRoutes,
		PointRoutes: data.DefaultPointRoutes,
		PageTitle:   "add point",
		ExtraData: map[string]any{
			"Seq": seq,
		},
	}
	RenderAddFormPoint(w, dataPage)
}

func AddPointHandler(w http.ResponseWriter, r *http.Request, queries *db.Queries) {
	userID, _, ok := tools.CheckRequest(w, r, http.MethodPost)
	if !ok {
		return
	}

	pointValueStr := strings.TrimSpace(r.FormValue("point"))
	if pointValueStr == "" {
		log.Printf("From AddPointHandler : field can't be empty")
		errorMessage := url.QueryEscape("Le champ ne peut pas être vide.")
		http.Redirect(w, r, data.ErrorMessageURL+"?errormessage="+errorMessage, http.StatusSeeOther)
		return
	}

	pointValue, err := strconv.ParseInt(pointValueStr, 10, 64)
	if err != nil {
		http.Error(w, "From AddPointHandler : Invalid point value", http.StatusBadRequest)
		return
	}

	err = queries.CreatePoint(r.Context(), db.CreatePointParams{
		PointValue: pointValue,
		UserID:     userID,
	})
	if err != nil {
		log.Printf("From AddPointHandler : CreatePoint DB error: %v", err)
		errorMessage := url.QueryEscape("Il ne peut pas exister deux fois le même champ.")
		http.Redirect(w, r, data.ErrorMessageURL+"?errormessage="+errorMessage, http.StatusSeeOther)
		return
	}

	http.Redirect(w, r, data.DefaultQuestionRoutes.PointsURL, http.StatusSeeOther)
}

func EditFormPointHandler(w http.ResponseWriter, r *http.Request, queries *db.Queries) {
	_, _, ok := tools.CheckRequest(w, r, http.MethodGet)
	if !ok {
		return
	}

	pointIDStr := r.FormValue("point_id")
	if pointIDStr == "" {
		http.Error(w, "From EditFormPointHandler : No point id parameter", http.StatusBadRequest)
		return
	}

	seq := make([]int, 0, 100)
	for i := 1; i <= 100; i++ {
		seq = append(seq, i)
	}

	dataPage := data.PointPageData{
		Routes:      data.DefaultDashboardRoutes,
		PointRoutes: data.DefaultPointRoutes,
		PageTitle:   "edit point",
		ExtraData: map[string]any{
			"PointID": pointIDStr,
			"Seq":     seq,
		},
	}
	RenderEditFormPoint(w, dataPage)
}

func EditPointHandler(w http.ResponseWriter, r *http.Request, queries *db.Queries) {
	userID, _, ok := tools.CheckRequest(w, r, http.MethodPost)
	if !ok {
		return
	}

	newPoint := strings.TrimSpace(r.FormValue("new_point"))
	if newPoint == "" {
		log.Printf("From EditPointHandler : field can't be empty")
		errorMessage := url.QueryEscape("Le champ ne peut pas être vide.")
		http.Redirect(w, r, data.ErrorMessageURL+"?errormessage="+errorMessage, http.StatusSeeOther)
		return
	}

	pointValue, err := strconv.ParseInt(newPoint, 10, 64)
	if err != nil {
		http.Error(w, "From EditPointHandler : Invalid point value", http.StatusBadRequest)
		return
	}

	pointIDStr := strings.TrimSpace(r.FormValue("point_id"))
	if pointIDStr == "" {
		http.Error(w, "From EditPointHandler : pointID missing", http.StatusInternalServerError)
		return
	}

	pointID, err := strconv.ParseInt(pointIDStr, 10, 64)
	if err != nil {
		http.Error(w, "From EditPointHandler : Invalid point ID", http.StatusBadRequest)
		return
	}

	if err := queries.UpdatePoint(r.Context(), db.UpdatePointParams{
		PointValue: pointValue,
		ID:         pointID,
		UserID:     userID,
	}); err != nil {
		log.Printf("From EditPointHandler : UpdatePoint DB error: %v", err)
		errorMessage := url.QueryEscape("Il ne peut pas exister deux fois le même champ.")
		http.Redirect(w, r, data.ErrorMessageURL+"?errormessage="+errorMessage, http.StatusSeeOther)
		return
	}

	http.Redirect(w, r, data.DefaultQuestionRoutes.PointsURL, http.StatusSeeOther)
}

func DeleteFormPointHandler(w http.ResponseWriter, r *http.Request, queries *db.Queries) {
	userID, _, ok := tools.CheckRequest(w, r, http.MethodGet)
	if !ok {
		return
	}

	pointIDStr := r.FormValue("point_id")
	if pointIDStr == "" {
		http.Error(w, "From DeleteFormPointHandler : No point id parameter", http.StatusBadRequest)
		return
	}

	pointID, err := strconv.ParseInt(pointIDStr, 10, 64)
	if err != nil {
		http.Error(w, "From DeleteFormPointHandler : Invalid point ID", http.StatusBadRequest)
		return
	}

	point, err := queries.GetPointByID(r.Context(), db.GetPointByIDParams{
		ID:     pointID,
		UserID: userID,
	})
	if err != nil {
		log.Printf("From DeleteFormPointHandler : GetPointByID DB error: %v", err)
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	dataPage := data.PointPageData{
		Routes:      data.DefaultDashboardRoutes,
		PointRoutes: data.DefaultPointRoutes,
		PageTitle:   "delete point",
		ExtraData: map[string]any{
			"Point":   point,
			"PointID": pointIDStr,
		},
	}

	RenderDeleteFormPoint(w, dataPage)
}

func DeletePointHandler(w http.ResponseWriter, r *http.Request, queries *db.Queries) {
	userID, _, ok := tools.CheckRequest(w, r, http.MethodPost)
	if !ok {
		return
	}

	pointIDStr := r.FormValue("point_id")
	if pointIDStr == "" {
		http.Error(w, "From DeletePointHandler : No point id parameter", http.StatusBadRequest)
		return
	}

	pointID, err := strconv.ParseInt(pointIDStr, 10, 64)
	if err != nil {
		http.Error(w, "From DeletePointHandler : Invalid point ID", http.StatusBadRequest)
		return
	}

	if err := queries.DeletePoint(r.Context(), db.DeletePointParams{
		ID:     pointID,
		UserID: userID,
	}); err != nil {
		log.Printf("From DeletePointHandler : DeletePoint DB error: %v", err)
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, data.DefaultQuestionRoutes.PointsURL, http.StatusSeeOther)
}
