package periods

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

func TablePeriodsHandler(w http.ResponseWriter, r *http.Request, queries *db.Queries) {
	userID, _, ok := tools.CheckRequest(w, r, http.MethodGet)
	if !ok {
		log.Println("From TablePeriodsHandler -> tools.CheckRequest return not ok")
		return
	}

	periodsDB, err := queries.GetAllPeriods(r.Context(), userID)
	if err != nil {
		log.Printf("From TablePeriodsHandler -> GetAllPeriods DB error: %v", err)
		http.Error(w, "DB error", http.StatusInternalServerError)
		return
	}

	noPeriod := true
	if len(periodsDB) > 0 {
		noPeriod = false
	}

	var actionsURLParameters []data.PeriodActionURLs
	if !noPeriod {
		for _, period := range periodsDB {
			params := "?period_id=" + url.QueryEscape(strconv.FormatInt(period.ID, 10))
			editURL := data.DefaultPeriodRoutes.EditURL + params
			deleteURL := data.DefaultPeriodRoutes.DeleteURL + params

			actionsURLParameters = append(actionsURLParameters, data.PeriodActionURLs{
				EditURL:   editURL,
				DeleteURL: deleteURL,
			})
		}
	}

	dataPage := data.PeriodPageData{
		Routes:       data.DefaultDashboardRoutes,
		PeriodRoutes: data.DefaultPeriodRoutes,
		PageTitle:    "periods",
		ExtraData: map[string]any{
			"NoPeriod": noPeriod,
			"Action":   actionsURLParameters,
			"Periods":  periodsDB,
		},
	}

	RenderTablePeriodPage(w, dataPage)
}

func AddFormPeriodHandler(w http.ResponseWriter, r *http.Request, queries *db.Queries) {
	_, _, ok := tools.CheckRequest(w, r, http.MethodGet)
	if !ok {
		log.Println("From AddFormPeriodHandler -> tools.CheckRequest return not ok")
		return
	}

	dataPage := data.PeriodPageData{
		Routes:       data.DefaultDashboardRoutes,
		PeriodRoutes: data.DefaultPeriodRoutes,
		PageTitle:    "add period",
	}
	RenderAddFormPeriodPage(w, dataPage)
}

func AddPeriodHandler(w http.ResponseWriter, r *http.Request, queries *db.Queries) {
	userID, _, ok := tools.CheckRequest(w, r, http.MethodPost)
	if !ok {
		log.Println("From AddPeriodHandler -> tools.CheckRequest return not ok")
		return
	}

	name := strings.TrimSpace(r.FormValue("period"))

	err := queries.CreatePeriod(r.Context(), db.CreatePeriodParams{
		Name:   name,
		UserID: userID,
	})
	if err != nil {
		log.Printf("From AddPeriodHandler -> CreatePeriod : DB error: %v", err)
		errorMessage := url.QueryEscape("Il ne peut pas exister deux fois le même champ ou le champ ne peut être vide.")
		http.Redirect(w, r, data.ErrorMessageURL+"?errormessage="+errorMessage, http.StatusSeeOther)
		return
	}

	http.Redirect(w, r, data.DefaultExamRoutes.PeriodsURL, http.StatusSeeOther)
}

func EditFormPeriodHandler(w http.ResponseWriter, r *http.Request, queries *db.Queries) {
	userID, _, ok := tools.CheckRequest(w, r, http.MethodGet)
	if !ok {
		log.Println("From EditFormPeriodHandler -> tools.CheckRequest return not ok")
		return
	}

	periodIDStr := r.URL.Query().Get("period_id")
	if periodIDStr == "" {
		log.Println("From EditFormPeriodHandler : no period id parameter")
		http.Error(w, "Something went wrong !", http.StatusBadRequest)
		return
	}

	periodID, err := strconv.ParseInt(periodIDStr, 10, 64)
	if err != nil {
		log.Printf("From EditFormPeriodHandler -> strconv.ParseInt, invalid period ID, error : %v", err)
		http.Error(w, "Something went wrong !", http.StatusBadRequest)
		return
	}

	period, err := queries.GetPeriodNameByID(r.Context(), db.GetPeriodNameByIDParams{
		ID:     periodID,
		UserID: userID,
	})
	if err != nil {
		tools.HandleOwnedLookupError(w, err, "EditFormPeriodHandler GetPeriodNameByID")
		return
	}

	dataPage := data.PeriodPageData{
		Routes:       data.DefaultDashboardRoutes,
		PeriodRoutes: data.DefaultPeriodRoutes,
		PageTitle:    "edit period",
		ExtraData: map[string]any{
			"Period":   period,
			"PeriodID": periodIDStr,
		},
	}
	RenderEditFormPeriodPage(w, dataPage)
}

func EditPeriodHandler(w http.ResponseWriter, r *http.Request, queries *db.Queries) {
	userID, _, ok := tools.CheckRequest(w, r, http.MethodPost)
	if !ok {
		log.Println("From EditPeriodHandler -> tools.CheckRequest return not ok")
		return
	}

	newPeriod := strings.TrimSpace(r.FormValue("new_period"))

	periodIDStr := r.FormValue("period_id")
	if periodIDStr == "" {
		log.Println("From EditPeriodHandler : no period ID")
		http.Error(w, "Something went wrong !", http.StatusBadRequest)
		return
	}
	periodID, err := strconv.ParseInt(periodIDStr, 10, 64)
	if err != nil {
		log.Printf("From EditPeriodHandler -> strconv.ParseInt, invalid period ID, error : %v", err)
		http.Error(w, "Something went wrong !", http.StatusBadRequest)
		return
	}

	rows, err := queries.UpdatePeriod(r.Context(), db.UpdatePeriodParams{
		Name:   newPeriod,
		ID:     periodID,
		UserID: userID,
	})
	if err != nil {
		log.Printf("From EditPeriodHandler : UpdatePeriod DB error: %v", err)
		errorMessage := url.QueryEscape("Il ne peut pas exister deux fois le même champ ou le champ ne peut pas être vide.")
		http.Redirect(w, r, data.ErrorMessageURL+"?errormessage="+errorMessage, http.StatusSeeOther)
		return
	}
	if !tools.HandleOwnedMutationRows(w, rows, "UpdatePeriod") {
		return
	}

	http.Redirect(w, r, data.DefaultExamRoutes.PeriodsURL, http.StatusSeeOther)
}

func DeleteFormPeriodHandler(w http.ResponseWriter, r *http.Request, queries *db.Queries) {
	userID, _, ok := tools.CheckRequest(w, r, http.MethodGet)
	if !ok {
		log.Println("From DeleteFormPeriodHandler -> tools.CheckRequest return not ok")
		return
	}

	periodIDStr := r.URL.Query().Get("period_id")
	if periodIDStr == "" {
		log.Println("From DeleteFormPeriodHandler : no skill id parameter")
		http.Error(w, "Something went wrong !", http.StatusBadRequest)
		return
	}

	periodID, err := strconv.ParseInt(periodIDStr, 10, 64)
	if err != nil {
		log.Printf("From DeleteFormPeriodHandler -> strconv.ParseInt, invalid skill ID, error : %v", err)
		http.Error(w, "Something went wrong !", http.StatusBadRequest)
		return
	}

	period, err := queries.GetPeriodNameByID(r.Context(), db.GetPeriodNameByIDParams{
		ID:     periodID,
		UserID: userID,
	})
	if err != nil {
		tools.HandleOwnedLookupError(w, err, "DeleteFormPeriodHandler GetPeriodNameByID")
		return
	}

	dataPage := data.PeriodPageData{
		Routes:       data.DefaultDashboardRoutes,
		PeriodRoutes: data.DefaultPeriodRoutes,
		PageTitle:    "delete period",
		ExtraData: map[string]any{
			"Period":   period,
			"PeriodID": periodIDStr,
		},
	}

	RenderDeleteFormPeriodPage(w, dataPage)
}

func DeletePeriodHandler(w http.ResponseWriter, r *http.Request, queries *db.Queries) {
	userID, _, ok := tools.CheckRequest(w, r, http.MethodPost)
	if !ok {
		log.Println("From DeletePeriodHandler -> tools.CheckRequest return not ok")
		return
	}

	periodIDStr := r.FormValue("period_id")
	if periodIDStr == "" {
		log.Println("From DeletePeriodHandler : no period id parameter")
		http.Error(w, "Something went wrong !", http.StatusBadRequest)
		return
	}

	periodID, err := strconv.ParseInt(periodIDStr, 10, 64)
	if err != nil {
		log.Printf("From DeletePeriodHandler -> strconv.ParseInt, invalid period ID, error : %v", err)
		http.Error(w, "Something went wrong !", http.StatusBadRequest)
		return
	}

	rows, err := queries.DeletePeriod(r.Context(), db.DeletePeriodParams{
		ID:     periodID,
		UserID: userID,
	})
	if err != nil {
		log.Printf("From DeletePeriodHandler : DeletePeriod DB error: %v", err)
		if tools.IsSQLiteForeignKeyConstraint(err) {
			errorMessage := url.QueryEscape("Cette période est utilisée par une évaluation et ne peut pas être supprimée.")
			http.Redirect(w, r, data.ErrorMessageURL+"?errormessage="+errorMessage, http.StatusSeeOther)
			return
		}
		http.Error(w, "DB error", http.StatusInternalServerError)
		return
	}
	if !tools.HandleOwnedMutationRows(w, rows, "DeletePeriod") {
		return
	}

	http.Redirect(w, r, data.DefaultExamRoutes.PeriodsURL, http.StatusSeeOther)
}
