package qcm

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

func TableQCMHandler(w http.ResponseWriter, r *http.Request, queries *db.Queries) {
	userID, _, ok := tools.CheckRequest(w, r, http.MethodGet)
	if !ok {
		log.Println("From TableQCMHandler -> tools.CheckRequest return not ok")
		return
	}

	rows, err := queries.GetAllQCM(r.Context(), userID)
	if err != nil {
		log.Printf("From TableQCMHandler -> GetAllQCM DB error: %v", err)
		http.Error(w, "DB error", http.StatusInternalServerError)
		return
	}

	noRow := true
	if len(rows) > 0 {
		noRow = false
	}

	var actionsURLParameters []data.QCMActionURLs
	if !noRow {
		for _, row := range rows {
			params := "?qcm_id=" + url.QueryEscape(strconv.FormatInt(row.ID, 10))
			editURL := data.DefaultQCMRoutes.EditURL + params
			deleteURL := data.DefaultQCMRoutes.DeleteURL + params
			addQuestionURL := data.DefaultQCMRoutes.AddQuestionURL + params
			previewQCMURL := data.DefaultQCMRoutes.PreviewURL + params
			previewQCMLandscapeURL := data.DefaultQCMRoutes.PreviewLandscapeURL + params

			actionsURLParameters = append(actionsURLParameters, data.QCMActionURLs{
				EditURL:             editURL,
				DeleteURL:           deleteURL,
				AddQuestionURL:      addQuestionURL,
				PreviewURL:          previewQCMURL,
				PreviewLandscapeURL: previewQCMLandscapeURL,
			})
		}
	}

	dataPage := data.QCMPageData{
		Routes:    data.DefaultDashboardRoutes,
		QCMRoutes: data.DefaultQCMRoutes,
		PageTitle: "qcm",
		ExtraData: map[string]any{
			"NoQCM":  noRow,
			"Action": actionsURLParameters,
			"QCM":    rows,
		},
	}

	RenderTableQCMPage(w, dataPage)
}

func AddFormQCMHandler(w http.ResponseWriter, r *http.Request, queries *db.Queries) {
	_, _, ok := tools.CheckRequest(w, r, http.MethodGet)
	if !ok {
		log.Println("From AddFormQCMHandler -> tools.CheckRequest return not ok")
		return
	}

	dataPage := data.QCMPageData{
		Routes:    data.DefaultDashboardRoutes,
		QCMRoutes: data.DefaultQCMRoutes,
		PageTitle: "add qcm",
	}
	RenderAddFormQCMPage(w, dataPage)
}

func AddQCMHandler(w http.ResponseWriter, r *http.Request, queries *db.Queries) {
	userID, _, ok := tools.CheckRequest(w, r, http.MethodPost)
	if !ok {
		log.Println("From AddQCMHandler -> tools.CheckRequest return not ok")
		return
	}

	name := strings.TrimSpace(r.FormValue("qcm"))

	if err := queries.CreateQCM(r.Context(), db.CreateQCMParams{
		Name:   name,
		UserID: userID,
	}); err != nil {
		log.Printf("From AddQCMHandler -> CreateQCM : DB error: %v", err)
		errorMessage := url.QueryEscape("Il ne peut pas exister deux fois le même nom pour un qcm ou un qcm ne peut pas avoir un nom vide.")
		http.Redirect(w, r, data.ErrorMessageURL+"?errormessage="+errorMessage, http.StatusSeeOther)
		return
	}

	http.Redirect(w, r, data.DefaultDashboardRoutes.QcmURL, http.StatusSeeOther)
}

func EditFormQCMHandler(w http.ResponseWriter, r *http.Request, queries *db.Queries) {
	userID, _, ok := tools.CheckRequest(w, r, http.MethodGet)
	if !ok {
		log.Println("From EditFormQCMHandler -> tools.CheckRequest return not ok")
		return
	}

	qcmIDStr := r.URL.Query().Get("qcm_id")
	if qcmIDStr == "" {
		log.Println("From EditFormQCMHandler : no qcm id parameter")
		http.Error(w, "Something went wrong !", http.StatusBadRequest)
		return
	}

	qcmID, err := strconv.ParseInt(qcmIDStr, 10, 64)
	if err != nil {
		log.Printf("From EditFormQCMHandler -> strconv.ParseInt, invalid qcm ID, error : %v", err)
		http.Error(w, "Something went wrong !", http.StatusBadRequest)
		return
	}

	qcm, err := queries.GetQCMNameByID(r.Context(), db.GetQCMNameByIDParams{
		ID:     qcmID,
		UserID: userID,
	})
	if err != nil {
		log.Printf("From EditFormQCMHandler -> GetQCMNameByID DB error: %v", err)
		http.Error(w, "DB error", http.StatusInternalServerError)
		return
	}

	dataPage := data.QCMPageData{
		Routes:    data.DefaultDashboardRoutes,
		QCMRoutes: data.DefaultQCMRoutes,
		PageTitle: "edit qcm",
		ExtraData: map[string]any{
			"QCM":   qcm,
			"QCMID": qcmIDStr,
		},
	}
	RenderEditFormQCMPage(w, dataPage)
}

func EditQCMHandler(w http.ResponseWriter, r *http.Request, queries *db.Queries) {
	userID, _, ok := tools.CheckRequest(w, r, http.MethodPost)
	if !ok {
		log.Println("From EditQCMHandler -> tools.CheckRequest return not ok")
		return
	}

	newQCM := strings.TrimSpace(r.FormValue("new_qcm"))

	qcmIDStr := r.FormValue("qcm_id")
	if qcmIDStr == "" {
		log.Println("From EditQCMHandler : no qcm ID")
		http.Error(w, "Something went wrong !", http.StatusBadRequest)
		return
	}
	qcmID, err := strconv.ParseInt(qcmIDStr, 10, 64)
	if err != nil {
		log.Printf("From EditQCMHandler -> strconv.ParseInt, invalid qcm ID, error : %v", err)
		http.Error(w, "Something went wrong !", http.StatusBadRequest)
		return
	}

	if err := queries.UpdateQCM(r.Context(), db.UpdateQCMParams{
		Name:   newQCM,
		ID:     qcmID,
		UserID: userID,
	}); err != nil {
		log.Printf("From EditQCMHandler : UpdateQCM DB error: %v", err)
		errorMessage := url.QueryEscape("Il ne peut pas exister deux fois le même nom de qcm ou un qcm ne peut pas avoir un nom vide.")
		http.Redirect(w, r, data.ErrorMessageURL+"?errormessage="+errorMessage, http.StatusSeeOther)
		return
	}

	http.Redirect(w, r, data.DefaultDashboardRoutes.QcmURL, http.StatusSeeOther)
}

func DeleteFormQCMHandler(w http.ResponseWriter, r *http.Request, queries *db.Queries) {
	userID, _, ok := tools.CheckRequest(w, r, http.MethodGet)
	if !ok {
		log.Println("From DeleteFormQCMHandler -> tools.CheckRequest return not ok")
		return
	}

	qcmIDStr := r.URL.Query().Get("qcm_id")
	if qcmIDStr == "" {
		log.Println("From DeleteFormQCMHandler : no qcm id parameter")
		http.Error(w, "Something went wrong !", http.StatusBadRequest)
		return
	}

	qcmID, err := strconv.ParseInt(qcmIDStr, 10, 64)
	if err != nil {
		log.Printf("From DeleteFormQCMHandler -> strconv.ParseInt, invalid qcm ID, error : %v", err)
		http.Error(w, "Something went wrong !", http.StatusBadRequest)
		return
	}

	qcm, err := queries.GetQCMNameByID(r.Context(), db.GetQCMNameByIDParams{
		ID:     qcmID,
		UserID: userID,
	})
	if err != nil {
		log.Printf("From DeleteFormQCMHandler -> GetQCMNameByID DB error: %v", err)
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	dataPage := data.QCMPageData{
		Routes:    data.DefaultDashboardRoutes,
		QCMRoutes: data.DefaultQCMRoutes,
		PageTitle: "delete qcm",
		ExtraData: map[string]any{
			"QCM":   qcm,
			"QCMID": qcmIDStr,
		},
	}

	RenderDeleteFormQCMPage(w, dataPage)
}

func DeleteQCMHandler(w http.ResponseWriter, r *http.Request, queries *db.Queries) {
	userID, _, ok := tools.CheckRequest(w, r, http.MethodPost)
	if !ok {
		log.Println("From DeleteQCMHandler -> tools.CheckRequest return not ok")
		return
	}

	qcmIDStr := r.FormValue("qcm_id")
	if qcmIDStr == "" {
		log.Println("From DeleteQCMHandler : no qcm id parameter")
		http.Error(w, "Something went wrong !", http.StatusBadRequest)
		return
	}

	qcmID, err := strconv.ParseInt(qcmIDStr, 10, 64)
	if err != nil {
		log.Printf("From DeleteQCMHandler -> strconv.ParseInt, invalid qcm ID, error : %v", err)
		http.Error(w, "Something went wrong !", http.StatusBadRequest)
		return
	}

	if err := queries.DeleteQCM(r.Context(), db.DeleteQCMParams{
		ID:     qcmID,
		UserID: userID,
	}); err != nil {
		log.Printf("From DeleteQCMHandler : DeleteQCM DB error: %v", err)
		http.Error(w, "Something went wrong !", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, data.DefaultDashboardRoutes.QcmURL, http.StatusSeeOther)
}
