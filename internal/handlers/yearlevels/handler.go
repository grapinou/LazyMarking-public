package yearlevels

import (
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/grapinou/LazyMarking/internal/db"
	"github.com/grapinou/LazyMarking/internal/handlers/login"
	"github.com/grapinou/LazyMarking/internal/templates/data"
)

func YearLevelsHandler(w http.ResponseWriter, r *http.Request, queries *db.Queries) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userID, _, ok := login.FromContext(r)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	yearlevelsDB, err := queries.GetAllYearLevels(r.Context(), userID)
	if err != nil {
		log.Printf("GetAllYearLevels DB error: %v", err)
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	log.Println("yearLevelDB", yearlevelsDB)
	noSubject := true
	if len(yearlevelsDB) > 0 {
		noSubject = false
	}

	var actionsURLParameters []data.YearYevelActionURLs
	if !noSubject {
		for _, yearlevel := range yearlevelsDB {
			editURL := data.DefaultYearLevelRoutes.EditURL + "?yearlevel_id=" + url.QueryEscape(strconv.FormatInt(yearlevel.ID, 10))
			deleteURL := data.DefaultYearLevelRoutes.DeleteURL + "?yearlevel_id=" + url.QueryEscape(strconv.FormatInt(yearlevel.ID, 10))

			actionsURLParameters = append(actionsURLParameters, data.YearYevelActionURLs{
				EditURL:   editURL,
				DeleteURL: deleteURL,
			})
		}
	}

	data := data.YearLevelPageData{
		Routes:          data.DefaultDashboardRoutes,
		YearLevelRoutes: data.DefaultYearLevelRoutes,
		PageTitle:       "year levels",
		ExtraData: map[string]any{
			"NoSubject":  noSubject,
			"Action":     actionsURLParameters,
			"YearLevels": yearlevelsDB,
		},
	}

	RenderYearLevelPage(w, data)
}

func AddYearLevelsFormHandler(w http.ResponseWriter, r *http.Request, queries *db.Queries) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	_, _, ok := login.FromContext(r)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	data := data.YearLevelPageData{
		YearLevelRoutes: data.DefaultYearLevelRoutes,
		PageTitle:       "add year level",
	}
	RenderAddYearLevelForm(w, data)
}

func AddYearLevelsHandler(w http.ResponseWriter, r *http.Request, queries *db.Queries) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userID, _, ok := login.FromContext(r)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	name := strings.TrimSpace(r.FormValue("yearlevel"))
	if name == "" {
		http.Error(w, "Name field can't be empty", http.StatusBadRequest)
		return
	}

	err := queries.CreateYearLevel(r.Context(), db.CreateYearLevelParams{
		Name:   name,
		UserID: userID,
	})
	if err != nil {
		log.Printf("CreateYearLeve DB error: %v", err)
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, data.DefaultDashboardRoutes.YearLevelsURL, http.StatusSeeOther)
}

func EditYearLevelsFormHandler(w http.ResponseWriter, r *http.Request, queries *db.Queries) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userID, _, ok := login.FromContext(r)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	yearLevelIDStr := r.FormValue("yearlevel_id")
	if yearLevelIDStr == "" {
		http.Error(w, "No theme id parameter", http.StatusBadRequest)
		return
	}

	yearLevelID, err := strconv.ParseInt(yearLevelIDStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid year level ID", http.StatusBadRequest)
		return
	}

	yearLevel, err := queries.GetYearLevelByID(r.Context(), db.GetYearLevelByIDParams{
		ID:     yearLevelID,
		UserID: userID,
	})
	if err != nil {
		log.Printf("GetYearLevelByID DB error: %v", err)
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	data := data.YearLevelPageData{
		YearLevelRoutes: data.DefaultYearLevelRoutes,
		PageTitle:       "edit year level",
		ExtraData: map[string]any{
			"YearLevel":   yearLevel,
			"YearLevelID": yearLevelIDStr,
		},
	}
	RenderEditYearLevelForm(w, data)
}

func EditYearLevelsHandler(w http.ResponseWriter, r *http.Request, queries *db.Queries) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userID, _, ok := login.FromContext(r)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	newYearLevel := strings.TrimSpace(r.FormValue("new_yearlevel"))
	if newYearLevel == "" {
		http.Error(w, "Year level field can't be empty", http.StatusBadRequest)
		return
	}

	yearLevelIDStr := strings.TrimSpace(r.FormValue("yearlevel_id"))
	if yearLevelIDStr == "" {
		http.Error(w, "YearLevelID missing", http.StatusInternalServerError)
		return
	}
	yearLevelID, err := strconv.ParseInt(yearLevelIDStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid year level ID", http.StatusBadRequest)
		return
	}

	if err := queries.UpdateYearLevel(r.Context(), db.UpdateYearLevelParams{
		Name:   newYearLevel,
		ID:     yearLevelID,
		UserID: userID,
	}); err != nil {
		log.Printf("UpdateYearLevel DB error: %v", err)
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, data.DefaultDashboardRoutes.YearLevelsURL, http.StatusSeeOther)
}

func DeleteFormYearLevelsHandler(w http.ResponseWriter, r *http.Request, queries *db.Queries) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userID, _, ok := login.FromContext(r)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	yearLevelIDStr := r.FormValue("yearlevel_id")
	if yearLevelIDStr == "" {
		http.Error(w, "No year level id parameter", http.StatusBadRequest)
		return
	}

	yearLevelID, err := strconv.ParseInt(yearLevelIDStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid year level ID", http.StatusBadRequest)
		return
	}

	yearLevel, err := queries.GetYearLevelByID(r.Context(), db.GetYearLevelByIDParams{
		ID:     yearLevelID,
		UserID: userID,
	})
	if err != nil {
		log.Printf("GetYearLevelByID DB error: %v", err)
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	data := data.YearLevelPageData{
		YearLevelRoutes: data.DefaultYearLevelRoutes,
		PageTitle:       "delete year level",
		ExtraData: map[string]any{
			"YearLevel":   yearLevel,
			"YearLevelID": yearLevelIDStr,
		},
	}

	RenderDeleteYearLevelForm(w, data)
}

func DeleteYearLevelsHandler(w http.ResponseWriter, r *http.Request, queries *db.Queries) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userID, _, ok := login.FromContext(r)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	yearLevelIDStr := r.FormValue("yearlevel_id")
	if yearLevelIDStr == "" {
		http.Error(w, "No year level id parameter", http.StatusBadRequest)
		return
	}

	yearLevelID, err := strconv.ParseInt(yearLevelIDStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid year level ID", http.StatusBadRequest)
		return
	}

	if err := queries.DeleteYearLevel(r.Context(), db.DeleteYearLevelParams{
		ID:     yearLevelID,
		UserID: userID,
	}); err != nil {
		log.Printf("DeleteYearLevel DB error: %v", err)
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, data.DefaultDashboardRoutes.YearLevelsURL, http.StatusSeeOther)
}
