package years

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

func TableYearsHandler(w http.ResponseWriter, r *http.Request, queries *db.Queries) {
	userID, _, ok := tools.CheckRequest(w, r, http.MethodGet)
	if !ok {
		log.Println("From TableYearsHandler -> tools.CheckRequest return not ok")
		return
	}

	yearsDB, err := queries.GetAllYears(r.Context(), userID)
	if err != nil {
		log.Printf("From TableYearsHandler -> GetAllYears DB error: %v", err)
		http.Error(w, "DB error", http.StatusInternalServerError)
		return
	}

	noYear := true
	if len(yearsDB) > 0 {
		noYear = false
	}

	var actionsURLParameters []data.YearActionURLs
	if !noYear {
		for _, year := range yearsDB {
			params := "?year_id=" + url.QueryEscape(strconv.FormatInt(year.ID, 10))
			editURL := data.DefaultYearRoutes.EditURL + params
			deleteURL := data.DefaultYearRoutes.DeleteURL + params

			actionsURLParameters = append(actionsURLParameters, data.YearActionURLs{
				EditURL:   editURL,
				DeleteURL: deleteURL,
			})
		}
	}

	dataPage := data.YearPageData{
		Routes:     data.DefaultDashboardRoutes,
		YearRoutes: data.DefaultYearRoutes,
		PageTitle:  "Years",
		ExtraData: map[string]any{
			"NoYear": noYear,
			"Action": actionsURLParameters,
			"Years":  yearsDB,
		},
	}

	RenderTableYearPage(w, dataPage)
}

func AddFormYearHandler(w http.ResponseWriter, r *http.Request, queries *db.Queries) {
	_, _, ok := tools.CheckRequest(w, r, http.MethodGet)
	if !ok {
		log.Println("From AddFormYearHandler -> tools.CheckRequest return not ok")
		return
	}

	dataPage := data.YearPageData{
		Routes:     data.DefaultDashboardRoutes,
		YearRoutes: data.DefaultYearRoutes,
		PageTitle:  "add year",
	}
	RenderAddFormYearPage(w, dataPage)
}

func AddYearHandler(w http.ResponseWriter, r *http.Request, queries *db.Queries) {
	userID, _, ok := tools.CheckRequest(w, r, http.MethodPost)
	if !ok {
		log.Println("From AddYearHandler -> tools.CheckRequest return not ok")
		return
	}

	name := strings.TrimSpace(r.FormValue("year"))

	err := queries.CreateYear(r.Context(), db.CreateYearParams{
		Name:   name,
		UserID: userID,
	})
	if err != nil {
		log.Printf("From AddYearHandler -> CreateYear : DB error: %v", err)
		errorMessage := url.QueryEscape("Il ne peut pas exister deux fois le même champ ou le champ ne peut être vide.")
		http.Redirect(w, r, data.ErrorMessageURL+"?errormessage="+errorMessage, http.StatusSeeOther)
		return
	}

	http.Redirect(w, r, data.DefaultExamRoutes.YearsURL, http.StatusSeeOther)
}

func EditFormYearHandler(w http.ResponseWriter, r *http.Request, queries *db.Queries) {
	userID, _, ok := tools.CheckRequest(w, r, http.MethodGet)
	if !ok {
		log.Println("From EditFormYearHandler -> tools.CheckRequest return not ok")
		return
	}

	yearIDStr := r.URL.Query().Get("year_id")
	if yearIDStr == "" {
		log.Println("From EditFormYearHandler : no year id parameter")
		http.Error(w, "Something went wrong !", http.StatusBadRequest)
		return
	}

	yearID, err := strconv.ParseInt(yearIDStr, 10, 64)
	if err != nil {
		log.Printf("From EditFormYearHandler -> strconv.ParseInt, invalid year ID, error : %v", err)
		http.Error(w, "Something went wrong !", http.StatusBadRequest)
		return
	}

	year, err := queries.GetYearNameByID(r.Context(), db.GetYearNameByIDParams{
		ID:     yearID,
		UserID: userID,
	})
	if err != nil {
		log.Printf("From EditFormYearHandler -> GetYearNameByID DB error: %v", err)
		http.Error(w, "DB error", http.StatusInternalServerError)
		return
	}

	dataPage := data.YearPageData{
		Routes:     data.DefaultDashboardRoutes,
		YearRoutes: data.DefaultYearRoutes,
		PageTitle:  "edit year",
		ExtraData: map[string]any{
			"Year":   year,
			"YearID": yearIDStr,
		},
	}
	RenderEditFormYearPage(w, dataPage)
}

func EditYearHandler(w http.ResponseWriter, r *http.Request, queries *db.Queries) {
	userID, _, ok := tools.CheckRequest(w, r, http.MethodPost)
	if !ok {
		log.Println("From EditYearHandler -> tools.CheckRequest return not ok")
		return
	}

	newYear := strings.TrimSpace(r.FormValue("new_year"))

	yearIDStr := r.FormValue("year_id")
	if yearIDStr == "" {
		log.Println("From EditYearHandler : no year ID")
		http.Error(w, "Something went wrong !", http.StatusBadRequest)
		return
	}
	yearID, err := strconv.ParseInt(yearIDStr, 10, 64)
	if err != nil {
		log.Printf("From EditYearHandler -> strconv.ParseInt, invalid year ID, error : %v", err)
		http.Error(w, "Something went wrong !", http.StatusBadRequest)
		return
	}

	if err := queries.UpdateYear(r.Context(), db.UpdateYearParams{
		Name:   newYear,
		ID:     yearID,
		UserID: userID,
	}); err != nil {
		log.Printf("From EditYearHandler : UpdateYear DB error: %v", err)
		errorMessage := url.QueryEscape("Il ne peut pas exister deux fois le même champ ou le champ ne peut pas être vide.")
		http.Redirect(w, r, data.ErrorMessageURL+"?errormessage="+errorMessage, http.StatusSeeOther)
		return
	}

	http.Redirect(w, r, data.DefaultExamRoutes.YearsURL, http.StatusSeeOther)
}

func DeleteFormYearHandler(w http.ResponseWriter, r *http.Request, queries *db.Queries) {
	userID, _, ok := tools.CheckRequest(w, r, http.MethodGet)
	if !ok {
		log.Println("From DeleteFormYearHandler -> tools.CheckRequest return not ok")
		return
	}

	yearIDStr := r.URL.Query().Get("year_id")
	if yearIDStr == "" {
		log.Println("From DeleteFormYearHandler : no year id parameter")
		http.Error(w, "Something went wrong !", http.StatusBadRequest)
		return
	}

	yearID, err := strconv.ParseInt(yearIDStr, 10, 64)
	if err != nil {
		log.Printf("From DeleteFormYearHandler -> strconv.ParseInt, invalid year ID, error : %v", err)
		http.Error(w, "Something went wrong !", http.StatusBadRequest)
		return
	}

	year, err := queries.GetYearNameByID(r.Context(), db.GetYearNameByIDParams{
		ID:     yearID,
		UserID: userID,
	})
	if err != nil {
		log.Printf("From DeleteFormYearHandler -> GetYearNameByID DB error: %v", err)
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	dataPage := data.YearPageData{
		Routes:     data.DefaultDashboardRoutes,
		YearRoutes: data.DefaultYearRoutes,
		PageTitle:  "delete year",
		ExtraData: map[string]any{
			"Year":   year,
			"YearID": yearIDStr,
		},
	}

	RenderDeleteFormYearPage(w, dataPage)
}

func DeleteYearHandler(w http.ResponseWriter, r *http.Request, queries *db.Queries) {
	userID, _, ok := tools.CheckRequest(w, r, http.MethodPost)
	if !ok {
		log.Println("From DeleteYearHandler -> tools.CheckRequest return not ok")
		return
	}

	yearIDStr := r.FormValue("year_id")
	if yearIDStr == "" {
		log.Println("From DeleteYearHandler : no year id parameter")
		http.Error(w, "Something went wrong !", http.StatusBadRequest)
		return
	}

	yearID, err := strconv.ParseInt(yearIDStr, 10, 64)
	if err != nil {
		log.Printf("From DeleteYearHandler -> strconv.ParseInt, invalid year ID, error : %v", err)
		http.Error(w, "Something went wrong !", http.StatusBadRequest)
		return
	}

	if err := queries.DeleteYear(r.Context(), db.DeleteYearParams{
		ID:     yearID,
		UserID: userID,
	}); err != nil {
		log.Printf("From DeleteYearHandler : DeleteYear DB error: %v", err)
		errorMessage := url.QueryEscape("Ce champ est utilisé par un examen. Impossible de le supprimer pour l'instant.")
		http.Redirect(w, r, data.ErrorMessageURL+"?errormessage="+errorMessage, http.StatusSeeOther)
		return
	}

	http.Redirect(w, r, data.DefaultExamRoutes.YearsURL, http.StatusSeeOther)
}
