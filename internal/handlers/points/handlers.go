package points

import (
	"log"
	"net/http"
	"net/url"
	"strconv"

	"github.com/grapinou/LazyMarking/internal/db"
	"github.com/grapinou/LazyMarking/internal/handlers/tools"
	"github.com/grapinou/LazyMarking/internal/templates/data"
)

func TablePointsHandler(w http.ResponseWriter, r *http.Request, queries *db.Queries) {
	userID, _, ok := tools.CheckRequest(w, r, http.MethodGet)
	if !ok {
		log.Println("From TablePointsHandler -> tools.CheckRequest return not ok")
		return
	}

	pointsDB, err := queries.GetAllPoints(r.Context(), userID)
	if err != nil {
		log.Printf("From TablePointsHandler -> GetAllPoints DB error: %v", err)
		http.Error(w, "DB error", http.StatusInternalServerError)
		return
	}

	noPoint := true
	if len(pointsDB) > 0 {
		noPoint = false
	}

	var actionsURLParameters []data.PointActionURLs
	if !noPoint {
		for _, point := range pointsDB {
			params := "?point_id=" + url.QueryEscape(strconv.FormatInt(point.ID, 10))
			editURL := data.DefaultPointRoutes.EditURL + params
			deleteURL := data.DefaultPointRoutes.DeleteURL + params

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
		log.Println("From AddFormPointHandler -> tools.CheckRequest return not ok")
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
	RenderAddFormPointPage(w, dataPage)
}

func AddPointHandler(w http.ResponseWriter, r *http.Request, queries *db.Queries) {
	userID, _, ok := tools.CheckRequest(w, r, http.MethodPost)
	if !ok {
		log.Println("From AddPointHandler -> tools.CheckRequest return not ok")
		return
	}

	pointValueStr := r.FormValue("point")

	pointValue, err := strconv.ParseInt(pointValueStr, 10, 64)
	if err != nil {
		log.Printf("From AddPointHandler -> strconv.ParseInt, Invalid point value, error : %v", err)
		http.Error(w, "Something went wrong !", http.StatusBadRequest)
		return
	}

	err = queries.CreatePoint(r.Context(), db.CreatePointParams{
		PointValue: pointValue,
		UserID:     userID,
	})
	if err != nil {
		log.Printf("From AddPointHandler -> CreatePoint DB error: %v", err)
		errorMessage := url.QueryEscape("Il ne peut pas exister deux fois le même champ.")
		http.Redirect(w, r, data.ErrorMessageURL+"?errormessage="+errorMessage, http.StatusSeeOther)
		return
	}

	http.Redirect(w, r, data.DefaultQuestionRoutes.PointsURL, http.StatusSeeOther)
}

func EditFormPointHandler(w http.ResponseWriter, r *http.Request, queries *db.Queries) {
	_, _, ok := tools.CheckRequest(w, r, http.MethodGet)
	if !ok {
		log.Println("From EditFormPointHandler -> tools.CheckRequest return not ok")
		return
	}

	pointIDStr := r.URL.Query().Get("point_id")
	if pointIDStr == "" {
		log.Println("From EditFormPointHandler : No point id parameter")
		http.Error(w, "Something went wrong !", http.StatusBadRequest)
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
	RenderEditFormPointPage(w, dataPage)
}

func EditPointHandler(w http.ResponseWriter, r *http.Request, queries *db.Queries) {
	userID, _, ok := tools.CheckRequest(w, r, http.MethodPost)
	if !ok {
		log.Println("From TableDifficultiesHandler -> tools.CheckRequest return not ok")
		return
	}

	newPoint := r.FormValue("new_point")

	pointValue, err := strconv.ParseInt(newPoint, 10, 64)
	if err != nil {
		log.Printf("From EditPointHandler -> strconv.ParseInt, Invalid point value, error : %v", err)
		http.Error(w, "Something went wrong !", http.StatusBadRequest)
		return
	}

	pointIDStr := r.FormValue("point_id")
	if pointIDStr == "" {
		log.Println("From EditPointHandler : pointID missing")
		http.Error(w, "Something went wrong !", http.StatusBadRequest)
		return
	}

	pointID, err := strconv.ParseInt(pointIDStr, 10, 64)
	if err != nil {
		log.Printf("From EditPointHandler -> strconv.ParseInt, Invalid point ID, error : %v", err)
		http.Error(w, "Something went wrong !", http.StatusBadRequest)
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
		log.Println("From DeleteFormPointHandler -> tools.CheckRequest return not ok")
		return
	}

	pointIDStr := r.URL.Query().Get("point_id")
	if pointIDStr == "" {
		http.Error(w, "From DeleteFormPointHandler : No point id parameter", http.StatusBadRequest)
		return
	}

	pointID, err := strconv.ParseInt(pointIDStr, 10, 64)
	if err != nil {
		log.Printf("rom DeleteFormPointHandler -> strconv.ParseInt : Invalid point ID, error : %v", err)
		http.Error(w, "Something went wrong !", http.StatusBadRequest)
		return
	}

	point, err := queries.GetPointByID(r.Context(), db.GetPointByIDParams{
		ID:     pointID,
		UserID: userID,
	})
	if err != nil {
		log.Printf("From DeleteFormPointHandler -> GetPointByID DB error: %v", err)
		http.Error(w, "DB error", http.StatusInternalServerError)
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

	RenderDeleteFormPointPage(w, dataPage)
}

func DeletePointHandler(w http.ResponseWriter, r *http.Request, queries *db.Queries) {
	userID, _, ok := tools.CheckRequest(w, r, http.MethodPost)
	if !ok {
		log.Println("From DeletePointHandler -> tools.CheckRequest return not ok")
		return
	}

	pointIDStr := r.FormValue("point_id")
	if pointIDStr == "" {
		log.Println("From DeletePointHandler : No point id parameter")
		http.Error(w, "Something went wrong !", http.StatusBadRequest)
		return
	}

	pointID, err := strconv.ParseInt(pointIDStr, 10, 64)
	if err != nil {
		log.Printf("From DeletePointHandler -> strconv.ParseInt : Invalid point ID, error : %v", err)
		http.Error(w, "Something went wrong !", http.StatusBadRequest)
		return
	}

	if err := queries.DeletePoint(r.Context(), db.DeletePointParams{
		ID:     pointID,
		UserID: userID,
	}); err != nil {
		log.Printf("From DeletePointHandler -> DeletePoint DB error: %v", err)
		errorMessage := url.QueryEscape("Ce champ est utilisé par une question. Impossible de le supprimer pour l'instant.")
		http.Redirect(w, r, data.ErrorMessageURL+"?errormessage="+errorMessage, http.StatusSeeOther)
		return
	}

	http.Redirect(w, r, data.DefaultQuestionRoutes.PointsURL, http.StatusSeeOther)
}
