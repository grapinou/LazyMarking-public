package yearlevels

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

func TableYearLevelsHandler(w http.ResponseWriter, r *http.Request, queries *db.Queries) {
	userID, _, ok := tools.CheckRequest(w, r, http.MethodGet)
	if !ok {
		log.Println("From TableYearLevelsHandler -> tools.CheckRequest return not ok")
		return
	}

	yearlevelsDB, err := queries.GetAllYearLevels(r.Context(), userID)
	if err != nil {
		log.Printf("From TableYearLevelsHandler -> GetAllYearLevels DB error: %v", err)
		http.Error(w, "DB error", http.StatusInternalServerError)
		return
	}

	noYearLevel := true
	if len(yearlevelsDB) > 0 {
		noYearLevel = false
	}

	var actionsURLParameters []data.YearYevelActionURLs
	if !noYearLevel {
		for _, yearlevel := range yearlevelsDB {
			params := "?yearlevel_id=" + url.QueryEscape(strconv.FormatInt(yearlevel.ID, 10))
			editURL := data.DefaultYearLevelRoutes.EditURL + params
			deleteURL := data.DefaultYearLevelRoutes.DeleteURL + params

			actionsURLParameters = append(actionsURLParameters, data.YearYevelActionURLs{
				EditURL:   editURL,
				DeleteURL: deleteURL,
			})
		}
	}

	dataPage := data.YearLevelPageData{
		Routes:          data.DefaultDashboardRoutes,
		YearLevelRoutes: data.DefaultYearLevelRoutes,
		PageTitle:       "year levels",
		ExtraData: map[string]any{
			"NoYearLevel": noYearLevel,
			"Action":      actionsURLParameters,
			"YearLevels":  yearlevelsDB,
		},
	}

	RenderTableYearLevelPage(w, dataPage)
}

func AddFormYearLevelHandler(w http.ResponseWriter, r *http.Request, queries *db.Queries) {
	_, _, ok := tools.CheckRequest(w, r, http.MethodGet)
	if !ok {
		log.Println("From AddFormYearLevelHandler -> tools.CheckRequest return not ok")
		return
	}

	dataPage := data.YearLevelPageData{
		Routes:          data.DefaultDashboardRoutes,
		YearLevelRoutes: data.DefaultYearLevelRoutes,
		PageTitle:       "add year level",
	}
	RenderAddFormYearLevelPage(w, dataPage)
}

func AddYearLevelHandler(w http.ResponseWriter, r *http.Request, queries *db.Queries) {
	userID, _, ok := tools.CheckRequest(w, r, http.MethodPost)
	if !ok {
		log.Println("From AddYearLevelHandler -> tools.CheckRequest return not ok")
		return
	}

	name := strings.TrimSpace(r.FormValue("yearlevel"))

	err := queries.CreateYearLevel(r.Context(), db.CreateYearLevelParams{
		Name:   name,
		UserID: userID,
	})
	if err != nil {
		log.Printf("From AddYearLevelHandler -> CreateYearLevel DB error: %v", err)
		errorMessage := url.QueryEscape("Il ne peut pas exister deux fois le même champ ou le champ ne peut être vide.")
		http.Redirect(w, r, data.ErrorMessageURL+"?errormessage="+errorMessage, http.StatusSeeOther)
		return
	}

	http.Redirect(w, r, data.DefaultQuestionRoutes.YearLevelsURL, http.StatusSeeOther)
}

func EditFormYearLevelHandler(w http.ResponseWriter, r *http.Request, queries *db.Queries) {
	userID, _, ok := tools.CheckRequest(w, r, http.MethodGet)
	if !ok {
		log.Println("From EditFormYearLevelHandler -> tools.CheckRequest return not ok")
		return
	}

	yearLevelIDStr := r.URL.Query().Get("yearlevel_id")
	if yearLevelIDStr == "" {
		log.Println("From EditFormYearLevelHandler, no year level id parameter")
		http.Error(w, "Something went wrong !", http.StatusBadRequest)
		return
	}

	yearLevelID, err := strconv.ParseInt(yearLevelIDStr, 10, 64)
	if err != nil {
		log.Printf("From EditFormYearLevelHandler -> strconv.ParseInt, invalid year level ID, error : %v", err)
		http.Error(w, "Something went wrong !", http.StatusBadRequest)
		return
	}

	yearLevel, err := queries.GetYearLevelByID(r.Context(), db.GetYearLevelByIDParams{
		ID:     yearLevelID,
		UserID: userID,
	})
	if err != nil {
		log.Printf("From EditFormYearLevelHandler -> GetYearLevelByID DB error: %v", err)
		http.Error(w, "DB error", http.StatusInternalServerError)
		return
	}

	dataPage := data.YearLevelPageData{
		Routes:          data.DefaultDashboardRoutes,
		YearLevelRoutes: data.DefaultYearLevelRoutes,
		PageTitle:       "edit year level",
		ExtraData: map[string]any{
			"YearLevel":   yearLevel,
			"YearLevelID": yearLevelIDStr,
		},
	}
	RenderEditFormYearLevelPage(w, dataPage)
}

func EditYearLevelHandler(w http.ResponseWriter, r *http.Request, queries *db.Queries) {
	userID, _, ok := tools.CheckRequest(w, r, http.MethodPost)
	if !ok {
		log.Println("From EditYearLevelHandler -> tools.CheckRequest return not ok")
		return
	}

	newYearLevel := strings.TrimSpace(r.FormValue("new_yearlevel"))

	yearLevelIDStr := r.FormValue("yearlevel_id")
	if yearLevelIDStr == "" {
		log.Println("From EditYearLevelHandler,  YearLevelID missing")
		http.Error(w, "Something went wrong !", http.StatusBadRequest)
		return
	}
	yearLevelID, err := strconv.ParseInt(yearLevelIDStr, 10, 64)
	if err != nil {
		log.Printf("From EditYearLevelHandler -> strconv.ParseInt, invalid year level ID, error : %v", err)
		http.Error(w, "Something went wrong !", http.StatusBadRequest)
		return
	}

	if err := queries.UpdateYearLevel(r.Context(), db.UpdateYearLevelParams{
		Name:   newYearLevel,
		ID:     yearLevelID,
		UserID: userID,
	}); err != nil {
		log.Printf("From EditYearLevelHandler -> UpdateYearLevel DB error: %v", err)
		errorMessage := url.QueryEscape("Il ne peut pas exister deux fois le même champ ou le champ ne peut être vide.")
		http.Redirect(w, r, data.ErrorMessageURL+"?errormessage="+errorMessage, http.StatusSeeOther)
		return
	}

	http.Redirect(w, r, data.DefaultQuestionRoutes.YearLevelsURL, http.StatusSeeOther)
}

func DeleteFormYearLevelHandler(w http.ResponseWriter, r *http.Request, queries *db.Queries) {
	userID, _, ok := tools.CheckRequest(w, r, http.MethodGet)
	if !ok {
		log.Println("From DeleteFormYearLevelHandler -> tools.CheckRequest return not ok")
		return
	}

	yearLevelIDStr := r.URL.Query().Get("yearlevel_id")
	if yearLevelIDStr == "" {
		log.Println("From DeleteFormYearLevelHandler : No year level id parameter")
		http.Error(w, "Something went wrong !", http.StatusBadRequest)
		return
	}

	yearLevelID, err := strconv.ParseInt(yearLevelIDStr, 10, 64)
	if err != nil {
		log.Printf("From DeleteFormYearLevelHandler -> strconv.ParseInt, invalid year level ID, error : %v", err)
		http.Error(w, "Something went wrong !", http.StatusBadRequest)
		return
	}

	yearLevel, err := queries.GetYearLevelByID(r.Context(), db.GetYearLevelByIDParams{
		ID:     yearLevelID,
		UserID: userID,
	})
	if err != nil {
		log.Printf("From DeleteFormYearLevelHandler -> GetYearLevelByID DB error: %v", err)
		http.Error(w, "DB error", http.StatusInternalServerError)
		return
	}

	dataPage := data.YearLevelPageData{
		Routes:          data.DefaultDashboardRoutes,
		YearLevelRoutes: data.DefaultYearLevelRoutes,
		PageTitle:       "delete year level",
		ExtraData: map[string]any{
			"YearLevel":   yearLevel,
			"YearLevelID": yearLevelIDStr,
		},
	}

	RenderDeleteFormYearLevelPage(w, dataPage)
}

func DeleteYearLevelHandler(w http.ResponseWriter, r *http.Request, queries *db.Queries) {
	userID, _, ok := tools.CheckRequest(w, r, http.MethodPost)
	if !ok {
		log.Println("From DeleteYearLevelHandler -> tools.CheckRequest return not ok")
		return
	}

	yearLevelIDStr := r.FormValue("yearlevel_id")
	if yearLevelIDStr == "" {
		log.Printf("From DeleteYearLevelHandler : No year level id parameter")
		http.Error(w, "Something went wrong !", http.StatusBadRequest)
		return
	}

	yearLevelID, err := strconv.ParseInt(yearLevelIDStr, 10, 64)
	if err != nil {
		log.Printf("From DeleteYearLevelHandler -> strconv.ParseInt, Invalid year level ID, error : %v", err)
		http.Error(w, "From DeleteYearLevelHandler : Invalid year level ID", http.StatusBadRequest)
		return
	}

	if err := queries.DeleteYearLevel(r.Context(), db.DeleteYearLevelParams{
		ID:     yearLevelID,
		UserID: userID,
	}); err != nil {
		log.Printf("From DeleteYearLevelHandler -> DeleteYearLevel DB error: %v", err)
		errorMessage := url.QueryEscape("Ce champ est utilisé par une question. Impossible de le supprimer pour l'instant.")
		http.Redirect(w, r, data.ErrorMessageURL+"?errormessage="+errorMessage, http.StatusSeeOther)
		return
	}

	http.Redirect(w, r, data.DefaultQuestionRoutes.YearLevelsURL, http.StatusSeeOther)
}
