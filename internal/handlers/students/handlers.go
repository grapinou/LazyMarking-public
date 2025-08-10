package students

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

func TableStudentsHandler(w http.ResponseWriter, r *http.Request, queries *db.Queries) {
	userID, _, ok := tools.CheckRequest(w, r, http.MethodGet)
	if !ok {
		log.Println("From TableStudentsHandler -> tools.CheckRequest return not ok")
		return
	}

	studentsDB, err := queries.GetAllStudentsWithClassCodesNames(r.Context(), userID)
	if err != nil {
		log.Printf("From TableStudentsHandler -> GetAllStudentsWithClassCodesNames DB error: %v", err)
		http.Error(w, "DB error", http.StatusInternalServerError)
		return
	}

	noStudent := true
	if len(studentsDB) > 0 {
		noStudent = false
	}

	var actionsURLParameters []data.StudentActionURLs
	if !noStudent {
		for _, student := range studentsDB {
			params := "?student_id=" + url.QueryEscape(strconv.FormatInt(student.StudentID, 10))
			editURL := data.DefaultStudentRoutes.EditURL + params
			deleteURL := data.DefaultStudentRoutes.DeleteURL + params

			actionsURLParameters = append(actionsURLParameters, data.StudentActionURLs{
				EditURL:   editURL,
				DeleteURL: deleteURL,
			})
		}
	}

	dataPage := data.StudentPageData{
		Routes:        data.DefaultDashboardRoutes,
		StudentRoutes: data.DefaultStudentRoutes,
		PageTitle:     "students",
		ExtraData: map[string]any{
			"NoStudent": noStudent,
			"Action":    actionsURLParameters,
			"Students":  studentsDB,
		},
	}

	RenderTableStudentPage(w, dataPage)
}

func AddFormStudentHandler(w http.ResponseWriter, r *http.Request, queries *db.Queries) {
	userID, _, ok := tools.CheckRequest(w, r, http.MethodGet)
	if !ok {
		log.Println("From ddFormStudentHandler -> tools.CheckRequest return not ok")
		return
	}

	allClassCodes, err := queries.GetAllClassCodes(r.Context(), userID)
	if err != nil {
		log.Printf("From AddFormStudentHandler -> GetAllClassCodes error DB : %v", err)
		http.Error(w, "Something went wrong !", http.StatusInternalServerError)
		return
	}

	if len(allClassCodes) == 0 {
		errorMessage := url.QueryEscape("Un élève doit avoir au moins une classe. Créer une classe avant de faire un élève.")
		http.Redirect(w, r, data.ErrorMessageURL+"?errormessage="+errorMessage, http.StatusSeeOther)
		return
	}

	dataPage := data.StudentPageData{
		Routes:        data.DefaultDashboardRoutes,
		StudentRoutes: data.DefaultStudentRoutes,
		PageTitle:     "add student",
		ExtraData: map[string]any{
			"ClassCodes": allClassCodes,
		},
	}

	RenderAddFormStudentPage(w, dataPage)
}

func AddStudentHandler(w http.ResponseWriter, r *http.Request, queries *db.Queries) {
	userID, _, ok := tools.CheckRequest(w, r, http.MethodPost)
	if !ok {
		log.Println("From AddStudentHandler -> tools.CheckRequest return not ok")
		return
	}

	classCodeIDStr := r.FormValue("class_code_id")
	classCodeID, err := strconv.ParseInt(classCodeIDStr, 10, 64)
	if err != nil {
		log.Printf("From AddStudentHandler -> strconv.ParseInt : can't convert class code id : error : %v", err)
		http.Error(w, "Something went wrong !", http.StatusBadRequest)
		return
	}
	firstName := strings.TrimSpace(r.FormValue("first_name"))
	lastName := strings.TrimSpace(r.FormValue("last_name"))

	if err = queries.CreateStudent(r.Context(), db.CreateStudentParams{
		FirstName: firstName,
		LastName:  lastName,
		UserID:    userID,
	}); err != nil {
		log.Printf("From AddStudentHandler -> DB CreateStudent error : %v", err)
		errorMessage := url.QueryEscape("Il ne peut pas exister deux fois le même étudiant ou un étudiant ne peut pas être sans nom.")
		http.Redirect(w, r, data.ErrorMessageURL+"?errormessage="+errorMessage, http.StatusSeeOther)
		return
	}

	studentID, err := queries.GetStudentIDByNameAndUserID(r.Context(), db.GetStudentIDByNameAndUserIDParams{
		FirstName: firstName,
		LastName:  lastName,
		UserID:    userID,
	})
	if err != nil {
		log.Printf("From AddStudentHandler -> DB GetStudentIDByNameAndUserID error : %v", err)
		http.Error(w, "Something went wrong !", http.StatusInternalServerError)
		return
	}

	if err = queries.CreateStudentWithClassCode(r.Context(), db.CreateStudentWithClassCodeParams{
		StudentID:   studentID,
		ClassCodeID: classCodeID,
		UserID:      userID,
	}); err != nil {
		log.Printf("From AddStudentHandler -> DB CreateStudentWithClassCode error : %v", err)
		http.Error(w, "Something went wrong !", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, data.DefaultDashboardRoutes.StudentURL, http.StatusSeeOther)
}

func EditFormStudentHandler(w http.ResponseWriter, r *http.Request, queries *db.Queries) {
	userID, _, ok := tools.CheckRequest(w, r, http.MethodGet)
	if !ok {
		log.Println("From EditFormStudentHandler -> tools.CheckRequest return not ok")
		return
	}

	studentIDStr := r.URL.Query().Get("student_id")
	if studentIDStr == "" {
		log.Println("From EditFormStudentHandler : no student id parameter")
		http.Error(w, "Something went wrong !", http.StatusBadRequest)
		return
	}

	studentID, err := strconv.ParseInt(studentIDStr, 10, 64)
	if err != nil {
		log.Printf("From EditFormStudentHandler -> strconv.ParseInt, invalid skill ID, error : %v", err)
		http.Error(w, "Something went wrong !", http.StatusBadRequest)
		return
	}

	student, err := queries.GetStudentByID(r.Context(), db.GetStudentByIDParams{
		ID:     studentID,
		UserID: userID,
	})
	if err != nil {
		log.Printf("From EditFormStudentHandler -> GetStudentByID DB error: %v", err)
		http.Error(w, "DB error", http.StatusInternalServerError)
		return
	}

	dataPage := data.StudentPageData{
		Routes:        data.DefaultDashboardRoutes,
		StudentRoutes: data.DefaultStudentRoutes,
		PageTitle:     "edit student",
		ExtraData: map[string]any{
			"FirstName": student.FirstName,
			"LastName":  student.LastName,
			"StudentID": studentIDStr,
		},
	}
	RenderEditFormStudentPage(w, dataPage)
}

func EditStudentHandler(w http.ResponseWriter, r *http.Request, queries *db.Queries) {
	userID, _, ok := tools.CheckRequest(w, r, http.MethodPost)
	if !ok {
		log.Println("From EditStudentHandler -> tools.CheckRequest return not ok")
		return
	}

	newFirstName := strings.TrimSpace(r.FormValue("new_first_name"))
	newLastName := strings.TrimSpace(r.FormValue("new_last_name"))

	studentIDStr := r.FormValue("student_id")
	if studentIDStr == "" {
		log.Println("From EditStudentHandler : no student ID")
		http.Error(w, "Something went wrong !", http.StatusBadRequest)
		return
	}
	studentID, err := strconv.ParseInt(studentIDStr, 10, 64)
	if err != nil {
		log.Printf("From EditStudentHandler -> strconv.ParseInt, invalid student ID, error : %v", err)
		http.Error(w, "Something went wrong !", http.StatusBadRequest)
		return
	}

	if err := queries.UpdateStudent(r.Context(), db.UpdateStudentParams{
		FirstName: newFirstName,
		LastName:  newLastName,
		ID:        studentID,
		UserID:    userID,
	}); err != nil {
		log.Printf("From EditStudentHandler : UpdateStudent DB error: %v", err)
		errorMessage := url.QueryEscape("Il ne peut pas exister deux fois le même étudiant ou un étudiant ne peut pas être sans nom.")
		http.Redirect(w, r, data.ErrorMessageURL+"?errormessage="+errorMessage, http.StatusSeeOther)
		return
	}

	http.Redirect(w, r, data.DefaultDashboardRoutes.StudentURL, http.StatusSeeOther)
}

func DeleteFormStudentHandler(w http.ResponseWriter, r *http.Request, queries *db.Queries) {
	userID, _, ok := tools.CheckRequest(w, r, http.MethodGet)
	if !ok {
		log.Println("From DeleteFormStudentHandler -> tools.CheckRequest return not ok")
		return
	}

	studentIDStr := r.URL.Query().Get("student_id")
	if studentIDStr == "" {
		log.Println("From DeleteFormStudentHandler : no student id parameter")
		http.Error(w, "Something went wrong !", http.StatusBadRequest)
		return
	}

	studentID, err := strconv.ParseInt(studentIDStr, 10, 64)
	if err != nil {
		log.Printf("From DeleteFormStudentHandler -> strconv.ParseInt, invalid student ID, error : %v", err)
		http.Error(w, "Something went wrong !", http.StatusBadRequest)
		return
	}

	student, err := queries.GetStudentByID(r.Context(), db.GetStudentByIDParams{
		ID:     studentID,
		UserID: userID,
	})
	if err != nil {
		log.Printf("From DeleteFormStudentHandler -> GetStudentByID DB error: %v", err)
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	dataPage := data.StudentPageData{
		Routes:        data.DefaultDashboardRoutes,
		StudentRoutes: data.DefaultStudentRoutes,
		PageTitle:     "delete student",
		ExtraData: map[string]any{
			"Student":   student,
			"StudentID": studentIDStr,
		},
	}

	RenderDeleteFormStudentPage(w, dataPage)
}

func DeleteStudentHandler(w http.ResponseWriter, r *http.Request, queries *db.Queries) {
	userID, _, ok := tools.CheckRequest(w, r, http.MethodPost)
	if !ok {
		log.Println("From DeleteStudentHandler -> tools.CheckRequest return not ok")
		return
	}

	studentIDStr := r.FormValue("student_id")
	if studentIDStr == "" {
		log.Println("From DeleteStudentHandler : no student id parameter")
		http.Error(w, "Something went wrong !", http.StatusBadRequest)
		return
	}

	studentID, err := strconv.ParseInt(studentIDStr, 10, 64)
	if err != nil {
		log.Printf("From DeleteStudentHandler -> strconv.ParseInt, invalid student ID, error : %v", err)
		http.Error(w, "Something went wrong !", http.StatusBadRequest)
		return
	}

	if err := queries.DeleteStudent(r.Context(), db.DeleteStudentParams{
		ID:     studentID,
		UserID: userID,
	}); err != nil {
		log.Printf("From DeleteStudentHandler : DeleteStudent DB error: %v", err)
		http.Error(w, "Something went wrong !", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, data.DefaultDashboardRoutes.StudentURL, http.StatusSeeOther)
}
