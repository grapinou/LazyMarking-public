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
		return
	}

	yearlevelsDB, err := queries.GetAllYearLevels(r.Context(), userID)
	if err != nil {
		log.Printf("From TableYearLevelsHandler : GetAllYearLevels DB error: %v", err)
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	log.Println("yearLevelDB", yearlevelsDB)
	noYearLevel := true
	if len(yearlevelsDB) > 0 {
		noYearLevel = false
	}

	var actionsURLParameters []data.YearYevelActionURLs
	if !noYearLevel {
		for _, yearlevel := range yearlevelsDB {
			editURL := data.DefaultYearLevelRoutes.EditURL + "?yearlevel_id=" + url.QueryEscape(strconv.FormatInt(yearlevel.ID, 10))
			deleteURL := data.DefaultYearLevelRoutes.DeleteURL + "?yearlevel_id=" + url.QueryEscape(strconv.FormatInt(yearlevel.ID, 10))

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
		return
	}

	dataPage := data.YearLevelPageData{
		YearLevelRoutes: data.DefaultYearLevelRoutes,
		PageTitle:       "add year level",
	}
	RenderAddFormYearLevel(w, dataPage)
}

func AddYearLevelHandler(w http.ResponseWriter, r *http.Request, queries *db.Queries) {
	userID, _, ok := tools.CheckRequest(w, r, http.MethodPost)
	if !ok {
		return
	}

	name := strings.TrimSpace(r.FormValue("yearlevel"))
	if name == "" {
		log.Printf("From AddYearLevelHandler : field can't be empty")
		errorMessage := url.QueryEscape("Le champ ne peut pas être vide.")
		http.Redirect(w, r, data.ErrorMessageURL+"?errormessage="+errorMessage, http.StatusSeeOther)
		return
	}

	err := queries.CreateYearLevel(r.Context(), db.CreateYearLevelParams{
		Name:   name,
		UserID: userID,
	})
	if err != nil {
		log.Printf("From AddYearLevelHandler : CreateYearLeve DB error: %v", err)
		errorMessage := url.QueryEscape("Il ne peut pas exister deux fois le même champ.")
		http.Redirect(w, r, data.ErrorMessageURL+"?errormessage="+errorMessage, http.StatusSeeOther)
		return
	}

	http.Redirect(w, r, data.DefaultDashboardRoutes.YearLevelsURL, http.StatusSeeOther)
}

func EditFormYearLevelHandler(w http.ResponseWriter, r *http.Request, queries *db.Queries) {
	userID, _, ok := tools.CheckRequest(w, r, http.MethodGet)
	if !ok {
		return
	}

	yearLevelIDStr := r.FormValue("yearlevel_id")
	if yearLevelIDStr == "" {
		http.Error(w, "From EditFormYearLevelHandler : No theme id parameter", http.StatusBadRequest)
		return
	}

	yearLevelID, err := strconv.ParseInt(yearLevelIDStr, 10, 64)
	if err != nil {
		http.Error(w, "From EditFormYearLevelHandler : Invalid year level ID", http.StatusBadRequest)
		return
	}

	yearLevel, err := queries.GetYearLevelByID(r.Context(), db.GetYearLevelByIDParams{
		ID:     yearLevelID,
		UserID: userID,
	})
	if err != nil {
		log.Printf("From EditFormYearLevelHandler : GetYearLevelByID DB error: %v", err)
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	dataPage := data.YearLevelPageData{
		YearLevelRoutes: data.DefaultYearLevelRoutes,
		PageTitle:       "edit year level",
		ExtraData: map[string]any{
			"YearLevel":   yearLevel,
			"YearLevelID": yearLevelIDStr,
		},
	}
	RenderEditFormYearLevel(w, dataPage)
}

func EditYearLevelHandler(w http.ResponseWriter, r *http.Request, queries *db.Queries) {
	userID, _, ok := tools.CheckRequest(w, r, http.MethodPost)
	if !ok {
		return
	}

	newYearLevel := strings.TrimSpace(r.FormValue("new_yearlevel"))
	if newYearLevel == "" {
		log.Printf("From EditYearLevelHandler : name field can't be empty")
		errorMessage := url.QueryEscape("Le champ ne peut pas être vide.")
		http.Redirect(w, r, data.ErrorMessageURL+"?errormessage="+errorMessage, http.StatusSeeOther)
		return
	}

	yearLevelIDStr := strings.TrimSpace(r.FormValue("yearlevel_id"))
	if yearLevelIDStr == "" {
		http.Error(w, "From EditYearLevelHandler : YearLevelID missing", http.StatusInternalServerError)
		return
	}
	yearLevelID, err := strconv.ParseInt(yearLevelIDStr, 10, 64)
	if err != nil {
		http.Error(w, "From EditYearLevelHandler : Invalid year level ID", http.StatusBadRequest)
		return
	}

	if err := queries.UpdateYearLevel(r.Context(), db.UpdateYearLevelParams{
		Name:   newYearLevel,
		ID:     yearLevelID,
		UserID: userID,
	}); err != nil {
		log.Printf("From EditYearLevelHandler : UpdateYearLevel DB error: %v", err)
		errorMessage := url.QueryEscape("Il ne peut pas exister deux fois le même champ.")
		http.Redirect(w, r, data.ErrorMessageURL+"?errormessage="+errorMessage, http.StatusSeeOther)
		return
	}

	http.Redirect(w, r, data.DefaultDashboardRoutes.YearLevelsURL, http.StatusSeeOther)
}

func DeleteFormYearLevelHandler(w http.ResponseWriter, r *http.Request, queries *db.Queries) {
	userID, _, ok := tools.CheckRequest(w, r, http.MethodGet)
	if !ok {
		return
	}

	yearLevelIDStr := r.FormValue("yearlevel_id")
	if yearLevelIDStr == "" {
		http.Error(w, "From DeleteFormYearLevelHandler : No year level id parameter", http.StatusBadRequest)
		return
	}

	yearLevelID, err := strconv.ParseInt(yearLevelIDStr, 10, 64)
	if err != nil {
		http.Error(w, "From DeleteFormYearLevelHandler : Invalid year level ID", http.StatusBadRequest)
		return
	}

	yearLevel, err := queries.GetYearLevelByID(r.Context(), db.GetYearLevelByIDParams{
		ID:     yearLevelID,
		UserID: userID,
	})
	if err != nil {
		log.Printf("From DeleteFormYearLevelHandler : GetYearLevelByID DB error: %v", err)
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	dataPage := data.YearLevelPageData{
		YearLevelRoutes: data.DefaultYearLevelRoutes,
		PageTitle:       "delete year level",
		ExtraData: map[string]any{
			"YearLevel":   yearLevel,
			"YearLevelID": yearLevelIDStr,
		},
	}

	RenderDeleteFormYearLevel(w, dataPage)
}

func DeleteYearLevelHandler(w http.ResponseWriter, r *http.Request, queries *db.Queries) {
	userID, _, ok := tools.CheckRequest(w, r, http.MethodPost)
	if !ok {
		return
	}

	yearLevelIDStr := r.FormValue("yearlevel_id")
	if yearLevelIDStr == "" {
		http.Error(w, "From DeleteYearLevelHandler : No year level id parameter", http.StatusBadRequest)
		return
	}

	yearLevelID, err := strconv.ParseInt(yearLevelIDStr, 10, 64)
	if err != nil {
		http.Error(w, "From DeleteYearLevelHandler : Invalid year level ID", http.StatusBadRequest)
		return
	}

	if err := queries.DeleteYearLevel(r.Context(), db.DeleteYearLevelParams{
		ID:     yearLevelID,
		UserID: userID,
	}); err != nil {
		log.Printf("From DeleteYearLevelHandler : DeleteYearLevel DB error: %v", err)
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, data.DefaultDashboardRoutes.YearLevelsURL, http.StatusSeeOther)
}
