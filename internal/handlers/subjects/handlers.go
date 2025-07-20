package subjects

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

func TableSubjectsHandler(w http.ResponseWriter, r *http.Request, queries *db.Queries) {
	userID, _, ok := tools.CheckRequest(w, r, http.MethodGet)
	if !ok {
		return
	}
	subjectsDB, err := queries.GetAllSubjects(r.Context(), userID)
	if err != nil {
		log.Printf("From TableSubjectsHandler, GetAllSubjects DB error: %v", err)
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	noSubject := true
	if len(subjectsDB) > 0 {
		noSubject = false
	}

	var actionsURLParameters []data.SubjectActionURLs
	if !noSubject {
		for _, subject := range subjectsDB {
			editURL := data.DefaultSubjectRoutes.EditURL + "?subject_id=" + url.QueryEscape(strconv.FormatInt(subject.ID, 10))
			deleteURL := data.DefaultSubjectRoutes.DeleteURL + "?subject_id=" + url.QueryEscape(strconv.FormatInt(subject.ID, 10))

			actionsURLParameters = append(actionsURLParameters, data.SubjectActionURLs{
				EditURL:   editURL,
				DeleteURL: deleteURL,
			})
		}
	}

	dataPage := data.SubjectPageData{
		Routes:        data.DefaultDashboardRoutes,
		SubjectRoutes: data.DefaultSubjectRoutes,
		PageTitle:     "subjects",
		ExtraData: map[string]any{
			"NoSubject": noSubject,
			"Action":    actionsURLParameters,
			"Subjects":  subjectsDB,
		},
	}

	RenderTableSubjectPage(w, dataPage)
}

func AddFormSubjectHandler(w http.ResponseWriter, r *http.Request, queries *db.Queries) {
	_, _, ok := tools.CheckRequest(w, r, http.MethodGet)
	if !ok {
		return
	}

	dataPage := data.SubjectPageData{
		Routes:        data.DefaultDashboardRoutes,
		SubjectRoutes: data.DefaultSubjectRoutes,
		PageTitle:     "add subject",
	}
	RenderAddFormSubject(w, dataPage)
}

func AddSubjectHandler(w http.ResponseWriter, r *http.Request, queries *db.Queries) {
	userID, _, ok := tools.CheckRequest(w, r, http.MethodPost)
	if !ok {
		return
	}

	name := strings.TrimSpace(r.FormValue("subject"))
	if name == "" {
		log.Printf("From AddSubjectHandler : name field can't be empty")
		errorMessage := url.QueryEscape("Le champ ne peut pas être vide.")
		http.Redirect(w, r, data.ErrorMessageURL+"?errormessage="+errorMessage, http.StatusSeeOther)
		return
	}

	err := queries.CreateSubject(r.Context(), db.CreateSubjectParams{
		Name:   name,
		UserID: userID,
	})
	if err != nil {
		log.Printf("From AddSubjectHandler, CreateSubject DB error: %v", err)
		errorMessage := url.QueryEscape("Il ne peut pas exister deux fois le même champ.")
		http.Redirect(w, r, data.ErrorMessageURL+"?errormessage="+errorMessage, http.StatusSeeOther)
		return
	}

	http.Redirect(w, r, data.DefaultDashboardRoutes.SubjectsURL, http.StatusSeeOther)
}

func EditFormSubjectHandler(w http.ResponseWriter, r *http.Request, queries *db.Queries) {
	userID, _, ok := tools.CheckRequest(w, r, http.MethodGet)
	if !ok {
		return
	}

	subjectIDStr := r.FormValue("subject_id")
	if subjectIDStr == "" {
		http.Error(w, "From EditFormSubjectHandler : no subject id parameter", http.StatusBadRequest)
		return
	}

	subjectID, err := strconv.ParseInt(subjectIDStr, 10, 64)
	if err != nil {
		http.Error(w, "From EditFormSubjectHandler : invalid subject ID", http.StatusBadRequest)
		return
	}

	subject, err := queries.GetSubjectNameByID(r.Context(), db.GetSubjectNameByIDParams{
		ID:     subjectID,
		UserID: userID,
	})
	if err != nil {
		log.Printf("From EditFormSubjectHandler, GetSubjectNameByID DB error: %v", err)
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	dataPage := data.SubjectPageData{
		Routes:        data.DefaultDashboardRoutes,
		SubjectRoutes: data.DefaultSubjectRoutes,
		PageTitle:     "edit subject",
		ExtraData: map[string]any{
			"Subject":   subject,
			"SubjectID": subjectIDStr,
		},
	}
	RenderEditFormSubject(w, dataPage)
}

func EditSubjectHandler(w http.ResponseWriter, r *http.Request, queries *db.Queries) {
	userID, _, ok := tools.CheckRequest(w, r, http.MethodPost)
	if !ok {
		return
	}

	newSubject := strings.TrimSpace(r.FormValue("new_subject"))
	if newSubject == "" {
		log.Printf("From EditSubjectHandler : field can't be empty")
		errorMessage := url.QueryEscape("Le champ ne peut pas être vide.")
		http.Redirect(w, r, data.ErrorMessageURL+"?errormessage="+errorMessage, http.StatusSeeOther)
		return
	}

	subjectIDStr := strings.TrimSpace(r.FormValue("subject_id"))
	if subjectIDStr == "" {
		http.Error(w, "From EditSubjectHandler : subjectID missing", http.StatusInternalServerError)
		return
	}
	subjectID, err := strconv.ParseInt(subjectIDStr, 10, 64)
	if err != nil {
		http.Error(w, "From EditSubjectHandler : invalid subject ID", http.StatusBadRequest)
		return
	}

	if err := queries.UpdateSubject(r.Context(), db.UpdateSubjectParams{
		Name:   newSubject,
		ID:     subjectID,
		UserID: userID,
	}); err != nil {
		log.Printf("From EditSubjectHandler, UpdateSubject DB error: %v", err)
		errorMessage := url.QueryEscape("Il ne peut pas exister deux fois le même champ.")
		http.Redirect(w, r, data.ErrorMessageURL+"?errormessage="+errorMessage, http.StatusSeeOther)
		return
	}

	http.Redirect(w, r, data.DefaultDashboardRoutes.SubjectsURL, http.StatusSeeOther)
}

func DeleteFormSubjectHandler(w http.ResponseWriter, r *http.Request, queries *db.Queries) {
	userID, _, ok := tools.CheckRequest(w, r, http.MethodGet)
	if !ok {
		return
	}

	subjectIDStr := r.FormValue("subject_id")
	if subjectIDStr == "" {
		http.Error(w, "From DeleteFormSubjectHandler : no subject id parameter", http.StatusBadRequest)
		return
	}

	subjectID, err := strconv.ParseInt(subjectIDStr, 10, 64)
	if err != nil {
		http.Error(w, "From DeleteFormSubjectHandler : invalid subject ID", http.StatusBadRequest)
		return
	}

	subject, err := queries.GetSubjectNameByID(r.Context(), db.GetSubjectNameByIDParams{
		ID:     subjectID,
		UserID: userID,
	})
	if err != nil {
		log.Printf("From DeleteFormSubjectHandler : GetSubjectNameByID DB error: %v", err)
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	dataPage := data.SubjectPageData{
		Routes:        data.DefaultDashboardRoutes,
		SubjectRoutes: data.DefaultSubjectRoutes,
		PageTitle:     "delete subject",
		ExtraData: map[string]any{
			"Subject":   subject,
			"SubjectID": subjectIDStr,
		},
	}

	RenderDeleteFormSubject(w, dataPage)
}

func DeleteSubjectHandler(w http.ResponseWriter, r *http.Request, queries *db.Queries) {
	userID, _, ok := tools.CheckRequest(w, r, http.MethodPost)
	if !ok {
		return
	}

	subjectIDStr := r.FormValue("subject_id")
	if subjectIDStr == "" {
		http.Error(w, "From DeleteSubjectHandler : no subject id parameter", http.StatusBadRequest)
		return
	}

	subjectID, err := strconv.ParseInt(subjectIDStr, 10, 64)
	if err != nil {
		http.Error(w, "From DeleteSubjectHandler : invalid subject ID", http.StatusBadRequest)
		return
	}

	if err := queries.DeleteSubject(r.Context(), db.DeleteSubjectParams{
		ID:     subjectID,
		UserID: userID,
	}); err != nil {
		log.Printf("From DeleteSubjectHandler : DeleteSubject DB error: %v", err)
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, data.DefaultDashboardRoutes.SubjectsURL, http.StatusSeeOther)
}
