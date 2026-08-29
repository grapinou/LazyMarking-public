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

var (
	renderTableSubjectPage      = RenderTableSubjectPage
	renderAddFormSubjectPage    = RenderAddFormSubjectPage
	renderEditFormSubjectPage   = RenderEditFormSubjectPage
	renderDeleteFormSubjectPage = RenderDeleteFormSubjectPage
)

func TableSubjectsHandler(w http.ResponseWriter, r *http.Request, queries *db.Queries) {
	userID, _, ok := tools.CheckRequest(w, r, http.MethodGet)
	if !ok {
		log.Println("From TableSubjectsHandler -> tools.CheckRequest return not ok")
		return
	}
	subjectsDB, err := queries.GetAllSubjects(r.Context(), userID)
	if err != nil {
		log.Printf("From TableSubjectsHandler -> GetAllSubjects DB error: %v", err)
		http.Error(w, "DB error", http.StatusInternalServerError)
		return
	}

	items := make([]data.SubjectListItem, 0, len(subjectsDB))
	for _, subject := range subjectsDB {
		items = append(items, data.SubjectListItem{
			ID:        subject.ID,
			Name:      subject.Name,
			EditURL:   data.SubjectURL(data.DefaultSubjectRoutes.EditURL, subject.ID),
			DeleteURL: data.SubjectURL(data.DefaultSubjectRoutes.DeleteURL, subject.ID),
		})
	}

	dataPage := data.SubjectPageData{
		Routes:        data.DefaultDashboardRoutes,
		SubjectRoutes: data.DefaultSubjectRoutes,
		PageTitle:     "Matières",
		SubjectItems:  items,
	}

	renderTableSubjectPage(w, dataPage)
}

func AddFormSubjectHandler(w http.ResponseWriter, r *http.Request, queries *db.Queries) {
	_, _, ok := tools.CheckRequest(w, r, http.MethodGet)
	if !ok {
		log.Println("From AddFormSubjectHandler -> tools.CheckRequest return not ok")
		return
	}

	dataPage := data.SubjectPageData{
		Routes:        data.DefaultDashboardRoutes,
		SubjectRoutes: data.DefaultSubjectRoutes,
		CancelURL:     data.DefaultQuestionRoutes.SubjectsURL,
		PageTitle:     "Ajouter une matière",
	}
	renderAddFormSubjectPage(w, dataPage)
}

func AddSubjectHandler(w http.ResponseWriter, r *http.Request, queries *db.Queries) {
	userID, _, ok := tools.CheckRequest(w, r, http.MethodPost)
	if !ok {
		log.Println("From AddSubjectHandler -> tools.CheckRequest return not ok")
		return
	}

	name := strings.TrimSpace(r.FormValue("subject"))

	err := queries.CreateSubject(r.Context(), db.CreateSubjectParams{
		Name:   name,
		UserID: userID,
	})
	if err != nil {
		log.Printf("From AddSubjectHandler, CreateSubject DB error: %v", err)
		errorMessage := url.QueryEscape("Il ne peut pas exister deux fois le même champ ou le champ ne peut pas être vide.")
		http.Redirect(w, r, data.ErrorMessageURL+"?errormessage="+errorMessage, http.StatusSeeOther)
		return
	}

	http.Redirect(w, r, data.DefaultQuestionRoutes.SubjectsURL, http.StatusSeeOther)
}

func EditFormSubjectHandler(w http.ResponseWriter, r *http.Request, queries *db.Queries) {
	userID, _, ok := tools.CheckRequest(w, r, http.MethodGet)
	if !ok {
		log.Println("From EditFormSubjectHandler -> tools.CheckRequest return not ok")
		return
	}

	subjectIDStr := r.URL.Query().Get("subject_id")
	if subjectIDStr == "" {
		log.Println("From EditFormSubjectHandler no subject ID")
		http.Error(w, "Something went wrong !", http.StatusBadRequest)
		return
	}

	subjectID, err := strconv.ParseInt(subjectIDStr, 10, 64)
	if err != nil {
		log.Printf("From EditFormSubjectHandler -> strconv.ParseInt, invalid subject ID, error : %v", err)
		http.Error(w, "Something went wrong !", http.StatusBadRequest)
		return
	}

	subject, err := queries.GetSubjectNameByID(r.Context(), db.GetSubjectNameByIDParams{
		ID:     subjectID,
		UserID: userID,
	})
	if err != nil {
		tools.HandleOwnedLookupError(w, err, "EditFormSubjectHandler GetSubjectNameByID")
		return
	}

	dataPage := data.SubjectPageData{
		Routes:        data.DefaultDashboardRoutes,
		SubjectRoutes: data.DefaultSubjectRoutes,
		SubjectContext: data.SubjectContext{
			ID:   subjectID,
			Name: subject,
		},
		CancelURL: data.DefaultQuestionRoutes.SubjectsURL,
		PageTitle: "Modifier la matière",
	}
	renderEditFormSubjectPage(w, dataPage)
}

func EditSubjectHandler(w http.ResponseWriter, r *http.Request, queries *db.Queries) {
	userID, _, ok := tools.CheckRequest(w, r, http.MethodPost)
	if !ok {
		log.Println("From EditSubjectHandler -> tools.CheckRequest return not ok")
		return
	}

	newSubject := strings.TrimSpace(r.FormValue("new_subject"))

	subjectIDStr := r.FormValue("subject_id")
	if subjectIDStr == "" {
		log.Println("From EditSubjectHandler :  no subjectID")
		http.Error(w, "Something went wrong !", http.StatusBadRequest)
		return
	}
	subjectID, err := strconv.ParseInt(subjectIDStr, 10, 64)
	if err != nil {
		log.Printf("From EditSubjectHandler -> strconv.ParseInt, invalid subjectId, error : %v", err)
		http.Error(w, "Something went wrong !", http.StatusBadRequest)
		return
	}

	rows, err := queries.UpdateSubject(r.Context(), db.UpdateSubjectParams{
		Name:   newSubject,
		ID:     subjectID,
		UserID: userID,
	})
	if err != nil {
		log.Printf("From EditSubjectHandler -> UpdateSubject DB error: %v", err)
		errorMessage := url.QueryEscape("Il ne peut pas exister deux fois le même champ ou le champ ne peut être vide.")
		http.Redirect(w, r, data.ErrorMessageURL+"?errormessage="+errorMessage, http.StatusSeeOther)
		return
	}
	if !tools.HandleOwnedMutationRows(w, rows, "UpdateSubject") {
		return
	}

	http.Redirect(w, r, data.DefaultQuestionRoutes.SubjectsURL, http.StatusSeeOther)
}

func DeleteFormSubjectHandler(w http.ResponseWriter, r *http.Request, queries *db.Queries) {
	userID, _, ok := tools.CheckRequest(w, r, http.MethodGet)
	if !ok {
		log.Println("From DeleteFormSubjectHandler -> tools.CheckRequest return not ok")
		return
	}

	subjectIDStr := r.URL.Query().Get("subject_id")
	if subjectIDStr == "" {
		log.Println("From DeleteFormSubjectHandler : no subject id parameter")
		http.Error(w, "Something went wrong !", http.StatusBadRequest)
		return
	}

	subjectID, err := strconv.ParseInt(subjectIDStr, 10, 64)
	if err != nil {
		log.Printf("From DeleteFormSubjectHandler -> strconv.ParseInt, invalid subjectID, error : %v", err)
		http.Error(w, "Something went wrong !", http.StatusBadRequest)
		return
	}

	subject, err := queries.GetSubjectNameByID(r.Context(), db.GetSubjectNameByIDParams{
		ID:     subjectID,
		UserID: userID,
	})
	if err != nil {
		tools.HandleOwnedLookupError(w, err, "DeleteFormSubjectHandler GetSubjectNameByID")
		return
	}

	dataPage := data.SubjectPageData{
		Routes:        data.DefaultDashboardRoutes,
		SubjectRoutes: data.DefaultSubjectRoutes,
		SubjectContext: data.SubjectContext{
			ID:   subjectID,
			Name: subject,
		},
		CancelURL: data.DefaultQuestionRoutes.SubjectsURL,
		PageTitle: "Supprimer la matière",
	}

	renderDeleteFormSubjectPage(w, dataPage)
}

func DeleteSubjectHandler(w http.ResponseWriter, r *http.Request, queries *db.Queries) {
	userID, _, ok := tools.CheckRequest(w, r, http.MethodPost)
	if !ok {
		log.Println("From DeleteSubjectHandler -> tools.CheckRequest return not ok")
		return
	}

	subjectIDStr := r.FormValue("subject_id")
	if subjectIDStr == "" {
		log.Println("From DeleteSubjectHandler : no subject id parameter")
		http.Error(w, "Something went wrong !", http.StatusBadRequest)
		return
	}

	subjectID, err := strconv.ParseInt(subjectIDStr, 10, 64)
	if err != nil {
		log.Printf("From DeleteSubjectHandler -> strconv.ParseInt, invalid subject ID, error : %v", err)
		http.Error(w, "Something went wrong !", http.StatusBadRequest)
		return
	}

	rows, err := queries.DeleteSubject(r.Context(), db.DeleteSubjectParams{
		ID:     subjectID,
		UserID: userID,
	})
	if err != nil {
		log.Printf("From DeleteSubjectHandler : DeleteSubject DB error: %v", err)
		if tools.IsSQLiteForeignKeyConstraint(err) {
			errorMessage := url.QueryEscape("Ce champ est utilisé par une question. Impossible de le supprimer pour l'instant.")
			http.Redirect(w, r, data.ErrorMessageURL+"?errormessage="+errorMessage, http.StatusSeeOther)
			return
		}
		http.Error(w, "Something went wrong !", http.StatusInternalServerError)
		return
	}
	if !tools.HandleOwnedMutationRows(w, rows, "DeleteSubject") {
		return
	}

	http.Redirect(w, r, data.DefaultQuestionRoutes.SubjectsURL, http.StatusSeeOther)
}
