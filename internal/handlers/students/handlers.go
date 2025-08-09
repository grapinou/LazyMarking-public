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
		errorMessage := url.QueryEscape("Il ne peut pas exister deux fois le même étudiant ou un étudiant vide ne peut exister.")
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
