package subjects

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

func SubjectsHandler(w http.ResponseWriter, r *http.Request, queries *db.Queries) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userID, _, ok := login.FromContext(r)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	subjectsDB, err := queries.GetAllSubjects(r.Context(), userID)
	if err != nil {
		http.Error(w, "Can't get all subjects", http.StatusInternalServerError)
	}

	log.Println("subjectsDB", subjectsDB)
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

	data := data.SubjectPageData{
		Routes:        data.DefaultDashboardRoutes,
		SubjectRoutes: data.DefaultSubjectRoutes,
		PageTitle:     "subjects",
		ExtraData: map[string]any{
			"NoSubject": noSubject,
			"Action":    actionsURLParameters,
			"Subjects":  subjectsDB,
		},
	}

	RenderSubjectPage(w, data)
}

func AddSubjectsFormHandler(w http.ResponseWriter, r *http.Request, queries *db.Queries) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	_, _, ok := login.FromContext(r)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	data := data.SubjectPageData{
		SubjectRoutes: data.DefaultSubjectRoutes,
		PageTitle:     "add subjects",
	}
	RenderAddSubjectForm(w, data)
}

func AddSubjectsHandler(w http.ResponseWriter, r *http.Request, queries *db.Queries) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userID, _, ok := login.FromContext(r)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	name := strings.TrimSpace(r.FormValue("subject"))
	if name == "" {
		http.Error(w, "Name field can't be empty", http.StatusBadRequest)
		return
	}

	err := queries.CreateSubject(r.Context(), db.CreateSubjectParams{
		Name:   name,
		UserID: userID,
	})
	if err != nil {
		log.Printf("CreateSubject DB error: %v", err)
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, data.DefaultDashboardRoutes.SubjectsURL, http.StatusSeeOther)
}

func EditSubjectsFormHandler(w http.ResponseWriter, r *http.Request, queries *db.Queries) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userID, _, ok := login.FromContext(r)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	subjectIDStr := r.FormValue("subject_id")
	if subjectIDStr == "" {
		http.Error(w, "No subject id parameter", http.StatusBadRequest)
		return
	}

	subjectID, err := strconv.ParseInt(subjectIDStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid subject ID", http.StatusBadRequest)
		return
	}

	subject, err := queries.GetSubjectNameByID(r.Context(), db.GetSubjectNameByIDParams{
		ID:     subjectID,
		UserID: userID,
	})
	if err != nil {
		log.Printf("GetSubjectNameByID DB error: %v", err)
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	data := data.SubjectPageData{
		SubjectRoutes: data.DefaultSubjectRoutes,
		PageTitle:     "edit subjects",
		ExtraData: map[string]any{
			"Subject":   subject,
			"SubjectID": subjectIDStr,
		},
	}
	RenderEditSubjectForm(w, data)
}

func EditSubjectsHandler(w http.ResponseWriter, r *http.Request, queries *db.Queries) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userID, _, ok := login.FromContext(r)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	newSubject := strings.TrimSpace(r.FormValue("new_subject"))
	if newSubject == "" {
		http.Error(w, "Subject field can't be empty", http.StatusBadRequest)
		return
	}

	subjectIDStr := strings.TrimSpace(r.FormValue("subject_id"))
	if subjectIDStr == "" {
		http.Error(w, "SubjectID missing", http.StatusInternalServerError)
		return
	}
	subjectID, err := strconv.ParseInt(subjectIDStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid subject ID", http.StatusBadRequest)
		return
	}

	if err := queries.UpdateSubject(r.Context(), db.UpdateSubjectParams{
		Name:   newSubject,
		ID:     subjectID,
		UserID: userID,
	}); err != nil {
		log.Printf("UpdateSubject DB error: %v", err)
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, data.DefaultDashboardRoutes.SubjectsURL, http.StatusSeeOther)
}

func DeleteFormSubjectsHandler(w http.ResponseWriter, r *http.Request, queries *db.Queries) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userID, _, ok := login.FromContext(r)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	subjectIDStr := r.FormValue("subject_id")
	if subjectIDStr == "" {
		http.Error(w, "No subject id parameter", http.StatusBadRequest)
		return
	}

	subjectID, err := strconv.ParseInt(subjectIDStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid subject ID", http.StatusBadRequest)
		return
	}

	subject, err := queries.GetSubjectNameByID(r.Context(), db.GetSubjectNameByIDParams{
		ID:     subjectID,
		UserID: userID,
	})
	if err != nil {
		log.Printf("GetSubjectNameByID DB error: %v", err)
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	data := data.SubjectPageData{
		SubjectRoutes: data.DefaultSubjectRoutes,
		PageTitle:     "delete subjects",
		ExtraData: map[string]any{
			"Subject":   subject,
			"SubjectID": subjectIDStr,
		},
	}

	RenderDeleteSubjectForm(w, data)
}

func DeleteSubjectsHandler(w http.ResponseWriter, r *http.Request, queries *db.Queries) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userID, _, ok := login.FromContext(r)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	subjectIDStr := r.FormValue("subject_id")
	if subjectIDStr == "" {
		http.Error(w, "No subject id parameter", http.StatusBadRequest)
		return
	}

	subjectID, err := strconv.ParseInt(subjectIDStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid subject ID", http.StatusBadRequest)
		return
	}

	if err := queries.DeleteSubject(r.Context(), db.DeleteSubjectParams{
		ID:     subjectID,
		UserID: userID,
	}); err != nil {
		log.Printf("DeleteSubject DB error: %v", err)
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, data.DefaultDashboardRoutes.SubjectsURL, http.StatusSeeOther)
}
