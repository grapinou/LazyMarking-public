package qcm

import (
	"errors"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/grapinou/LazyMarking/internal/db"
	"github.com/grapinou/LazyMarking/internal/handlers/tools"
	"github.com/grapinou/LazyMarking/internal/templates/data"
	"github.com/mattn/go-sqlite3"
)

var (
	renderTableQCMPage      = RenderTableQCMPage
	renderAddFormQCMPage    = RenderAddFormQCMPage
	renderEditFormQCMPage   = RenderEditFormQCMPage
	renderDeleteFormQCMPage = RenderDeleteFormQCMPage
	deleteOwnedQCM          = (*db.Queries).DeleteQCM
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

	items := make([]data.QCMListItem, 0, len(rows))
	for _, row := range rows {
		items = append(items, data.QCMListItem{
			ID:                  row.ID,
			Name:                row.Name,
			QuestionCount:       row.QuestionCount,
			CompositionURL:      data.QCMURL(data.DefaultQCMRoutes.AddQuestionURL, row.ID),
			PreviewURL:          data.QCMURL(data.DefaultQCMRoutes.PreviewURL, row.ID),
			PreviewLandscapeURL: data.QCMURL(data.DefaultQCMRoutes.PreviewLandscapeURL, row.ID),
			EditURL:             data.QCMURL(data.DefaultQCMRoutes.EditURL, row.ID),
			DeleteURL:           data.QCMURL(data.DefaultQCMRoutes.DeleteURL, row.ID),
		})
	}

	dataPage := data.QCMPageData{
		Routes:    data.DefaultDashboardRoutes,
		QCMRoutes: data.DefaultQCMRoutes,
		QCMItems:  items,
		PageTitle: "Mes QCM",
	}

	renderTableQCMPage(w, dataPage)
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
		PageTitle: "Créer un QCM",
	}
	renderAddFormQCMPage(w, dataPage)
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
		tools.HandleOwnedLookupError(w, err, "EditFormQCMHandler GetQCMNameByID")
		return
	}

	dataPage := data.QCMPageData{
		Routes:    data.DefaultDashboardRoutes,
		QCMRoutes: data.DefaultQCMRoutes,
		QCMContext: data.QCMContext{
			ID:   qcmID,
			Name: qcm,
		},
		PageTitle: "Modifier le QCM",
	}
	renderEditFormQCMPage(w, dataPage)
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

	rows, err := queries.UpdateQCM(r.Context(), db.UpdateQCMParams{
		Name:   newQCM,
		ID:     qcmID,
		UserID: userID,
	})
	if err != nil {
		log.Printf("From EditQCMHandler : UpdateQCM DB error: %v", err)
		errorMessage := url.QueryEscape("Il ne peut pas exister deux fois le même nom de qcm ou un qcm ne peut pas avoir un nom vide.")
		http.Redirect(w, r, data.ErrorMessageURL+"?errormessage="+errorMessage, http.StatusSeeOther)
		return
	}
	if !tools.HandleOwnedMutationRows(w, rows, "UpdateQCM") {
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
		tools.HandleOwnedLookupError(w, err, "DeleteFormQCMHandler GetQCMNameByID")
		return
	}

	dataPage := data.QCMPageData{
		Routes:    data.DefaultDashboardRoutes,
		QCMRoutes: data.DefaultQCMRoutes,
		QCMContext: data.QCMContext{
			ID:   qcmID,
			Name: qcm,
		},
		PageTitle: "Supprimer le QCM",
	}

	renderDeleteFormQCMPage(w, dataPage)
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

	hasExams, err := queries.QCMHasExams(r.Context(), db.QCMHasExamsParams{
		QcmID:  qcmID,
		UserID: userID,
	})
	if err != nil {
		tools.HandleOwnedLookupError(w, err, "DeleteQCMHandler QCMHasExams")
		return
	}
	if hasExams {
		redirectProtectedQCMDeletion(w, r)
		return
	}

	rows, err := deleteOwnedQCM(queries, r.Context(), db.DeleteQCMParams{
		ID:     qcmID,
		UserID: userID,
	})
	if err != nil {
		if isSQLiteForeignKeyConstraint(err) {
			redirectProtectedQCMDeletion(w, r)
			return
		}
		log.Printf("From DeleteQCMHandler : DeleteQCM DB error: %v", err)
		http.Error(w, "Something went wrong !", http.StatusInternalServerError)
		return
	}
	if !tools.HandleOwnedMutationRows(w, rows, "DeleteQCM") {
		return
	}

	http.Redirect(w, r, data.DefaultDashboardRoutes.QcmURL, http.StatusSeeOther)
}

func redirectProtectedQCMDeletion(w http.ResponseWriter, r *http.Request) {
	message := url.QueryEscape("Ce QCM est utilisé par une évaluation et ne peut pas être supprimé.")
	http.Redirect(w, r, data.ErrorMessageURL+"?errormessage="+message, http.StatusSeeOther)
}

func isSQLiteForeignKeyConstraint(err error) bool {
	var sqliteError sqlite3.Error
	return errors.As(err, &sqliteError) && sqliteError.ExtendedCode == sqlite3.ErrConstraintForeignKey
}
