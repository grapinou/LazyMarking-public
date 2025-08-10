package studentclasscode

import (
	"log"
	"net/http"
	"net/url"
	"strconv"

	"github.com/grapinou/LazyMarking/internal/config"
	"github.com/grapinou/LazyMarking/internal/db"
	"github.com/grapinou/LazyMarking/internal/handlers/tools"
	"github.com/grapinou/LazyMarking/internal/templates/data"
)

func TableStudentClassCodesHandler(w http.ResponseWriter, r *http.Request, queries *db.Queries) {
	userID, _, ok := tools.CheckRequest(w, r, http.MethodGet)
	if !ok {
		log.Println("From TableStudentClassCodesHandler -> tools.CheckRequest return not ok")
		return
	}

	studentIDStr := r.URL.Query().Get("student_id")
	if studentIDStr == "" {
		log.Println("From TableStudentClassCodesHandler : no student ID")
		http.Error(w, "Something went wrong !", http.StatusBadRequest)
		return
	}
	studentID, err := strconv.ParseInt(studentIDStr, 10, 64)
	if err != nil {
		log.Printf("From TableStudentClassCodesHandler -> strconv.ParseInt, invalid student ID, error : %v", err)
		http.Error(w, "Something went wrong !", http.StatusBadRequest)
		return
	}

	student, err := queries.GetStudentByID(r.Context(), db.GetStudentByIDParams{
		ID:     studentID,
		UserID: userID,
	})
	if err != nil {
		log.Printf("From TableStudentClassCodesHandler -> GetStudentByID, DB error : %v", err)
		http.Error(w, "Something went wrong !", http.StatusInternalServerError)
		return
	}

	classCodesID, err := queries.GetAllClassCodesByStudentID(r.Context(), db.GetAllClassCodesByStudentIDParams{
		StudentID: studentID,
		UserID:    userID,
	})
	if err != nil {
		log.Printf("From TableStudentClassCodesHandler -> GetAllClassCodesByStudentID, DB error : %v", err)
		http.Error(w, "Something went wrong !", http.StatusInternalServerError)
		return
	}

	var classCodes []config.ClassCode
	var actionsURLParameters []data.StudentClassCodeActionURLs
	for _, classCodeID := range classCodesID {
		classCodeName, err := queries.GetClassCodeNameByID(r.Context(), db.GetClassCodeNameByIDParams{
			ID:     classCodeID,
			UserID: userID,
		})
		if err != nil {
			log.Printf("From TableStudentsHandler -> GetClassCodeNameByID DB error: %v", err)
			http.Error(w, "DB error", http.StatusInternalServerError)
			return
		}
		classCode := config.ClassCode{
			ID:   classCodeID,
			Name: classCodeName,
		}
		classCodes = append(classCodes, classCode)
		params := "?student_id=" + url.QueryEscape(strconv.FormatInt(studentID, 10)) +
			"&class_code_id=" + url.QueryEscape(strconv.FormatInt(classCodeID, 10))

		deleteURL := data.DefaultStudentClassCodeRoutes.DeleteURL + params

		actionsURLParameters = append(actionsURLParameters, data.StudentClassCodeActionURLs{
			DeleteURL: deleteURL,
		})
	}

	AddURL := data.DefaultStudentClassCodeRoutes.AddURL + "?student_id=" + url.QueryEscape(strconv.FormatInt(studentID, 10))

	allowedDelete := false
	if len(classCodes) > 1 {
		allowedDelete = true
	}

	dataPage := data.StudentClassCodePageData{
		Routes:                 data.DefaultDashboardRoutes,
		StudentClassCodeRoutes: data.DefaultStudentClassCodeRoutes,
		PageTitle:              "student-classcodes",
		ExtraData: map[string]any{
			"AddURL":        AddURL,
			"Action":        actionsURLParameters,
			"ClassCodes":    classCodes,
			"Student":       student,
			"AllowedDelete": allowedDelete,
		},
	}

	RenderTableStudentClassCodesPage(w, dataPage)
}

func AddFormStudentClassCodeHandler(w http.ResponseWriter, r *http.Request, queries *db.Queries) {
	userID, _, ok := tools.CheckRequest(w, r, http.MethodGet)
	if !ok {
		log.Println("From AddFormSkillHandler -> tools.CheckRequest return not ok")
		return
	}

	studentIDStr := r.URL.Query().Get("student_id")
	if studentIDStr == "" {
		log.Println("From AddFormStudentClassCodeHandler : no student ID")
		http.Error(w, "Something went wrong !", http.StatusBadRequest)
		return
	}

	studentID, err := strconv.ParseInt(studentIDStr, 10, 64)
	if err != nil {
		log.Printf("From AddFormStudentClassCodeHandler -> strconv.ParseInt, invalid student ID, error : %v", err)
		http.Error(w, "Something went wrong !", http.StatusBadRequest)
		return
	}

	classCodes, err := queries.ListClassCodesNotAssignedToStudent(r.Context(), db.ListClassCodesNotAssignedToStudentParams{
		StudentID: studentID,
		UserID:    userID,
	})
	if err != nil {
		log.Printf("From AddFormStudentClassCodeHandler -> ListClassCodesNotAssignedToStudent DB error: %v", err)
		http.Error(w, "DB error", http.StatusInternalServerError)
		return
	}

	if len(classCodes) == 0 {
		errorMessage := url.QueryEscape("L'élève est déjà dans toutes les classes ! Il faut en créer d'autre !")
		http.Redirect(w, r, data.ErrorMessageURL+"?errormessage="+errorMessage, http.StatusSeeOther)
		return
	}

	dataPage := data.StudentClassCodePageData{
		Routes:                 data.DefaultDashboardRoutes,
		StudentClassCodeRoutes: data.DefaultStudentClassCodeRoutes,
		PageTitle:              "add extra class code",
		ExtraData: map[string]any{
			"StudentID":  studentIDStr,
			"ClassCodes": classCodes,
		},
	}
	RenderAddFormStudentClassCodePage(w, dataPage)
}

func AddStudentClassCodeHandler(w http.ResponseWriter, r *http.Request, queries *db.Queries) {
	userID, _, ok := tools.CheckRequest(w, r, http.MethodPost)
	if !ok {
		log.Println("From AddStudentClassCodeHandler -> tools.CheckRequest return not ok")
		return
	}

	studentIDStr := r.FormValue("student_id")
	if studentIDStr == "" {
		log.Println("From AddStudentClassCodeHandler : no student ID")
		http.Error(w, "Something went wrong !", http.StatusBadRequest)
		return
	}

	studentID, err := strconv.ParseInt(studentIDStr, 10, 64)
	if err != nil {
		log.Printf("From AddStudentClassCodeHandler -> strconv.ParseInt, invalid student ID, error : %v", err)
		http.Error(w, "Something went wrong !", http.StatusBadRequest)
		return
	}

	classCodeIDStr := r.FormValue("class_code_id")
	if classCodeIDStr == "" {
		log.Println("From AddStudentClassCodeHandler : no class code ID")
		http.Error(w, "Something went wrong !", http.StatusBadRequest)
		return
	}

	classCodeID, err := strconv.ParseInt(classCodeIDStr, 10, 64)
	if err != nil {
		log.Printf("From AddStudentClassCodeHandler -> strconv.ParseInt, invalid class code ID, error : %v", err)
		http.Error(w, "Something went wrong !", http.StatusBadRequest)
		return
	}

	if err := queries.CreateStudentWithClassCode(r.Context(), db.CreateStudentWithClassCodeParams{
		StudentID:   studentID,
		ClassCodeID: classCodeID,
		UserID:      userID,
	}); err != nil {
		log.Printf("From AddStudentClassCodeHandler -> CreateStudentWithClassCode : DB error: %v", err)
		errorMessage := url.QueryEscape("Il ne peut pas exister deux fois la même classe.")
		http.Redirect(w, r, data.ErrorMessageURL+"?errormessage="+errorMessage, http.StatusSeeOther)
		return
	}

	tableStudentClassCode := data.DefaultStudentRoutes.StudentClassCodesURL + "?student_id=" + url.QueryEscape(strconv.FormatInt(studentID, 10))
	http.Redirect(w, r, tableStudentClassCode, http.StatusSeeOther)
}

func DeleteStudentClassCodeHandler(w http.ResponseWriter, r *http.Request, queries *db.Queries) {
	userID, _, ok := tools.CheckRequest(w, r, http.MethodGet)
	if !ok {
		log.Println("From DeleteStudentClassCodeHandler -> tools.CheckRequest return not ok")
		return
	}

	studentIDStr := r.URL.Query().Get("student_id")
	if studentIDStr == "" {
		log.Println("From DeleteStudentClassCodeHandler : no student id parameter")
		http.Error(w, "Something went wrong !", http.StatusBadRequest)
		return
	}

	studentID, err := strconv.ParseInt(studentIDStr, 10, 64)
	if err != nil {
		log.Printf("From DeleteStudentClassCodeHandler -> strconv.ParseInt, invalid student ID, error : %v", err)
		http.Error(w, "Something went wrong !", http.StatusBadRequest)
		return
	}

	classCodeIDStr := r.URL.Query().Get("class_code_id")
	if classCodeIDStr == "" {
		log.Println("From DeleteStudentClassCodeHandler : no class code id parameter")
		http.Error(w, "Something went wrong !", http.StatusBadRequest)
		return
	}

	classCodeID, err := strconv.ParseInt(classCodeIDStr, 10, 64)
	if err != nil {
		log.Printf("From DeleteStudentClassCodeHandler -> strconv.ParseInt, invalid class code ID, error : %v", err)
		http.Error(w, "Something went wrong !", http.StatusBadRequest)
		return
	}

	if err := queries.DeleteStudentClassCodeByStudentID(r.Context(), db.DeleteStudentClassCodeByStudentIDParams{
		StudentID:   studentID,
		ClassCodeID: classCodeID,
		UserID:      userID,
	}); err != nil {
		log.Printf("From DeleteStudentClassCodeHandler : DeleteStudentClassCodeByStudentID DB error: %v", err)
		http.Error(w, "Something went wrong !", http.StatusInternalServerError)
		return
	}

	tableStudentClassCode := data.DefaultStudentRoutes.StudentClassCodesURL + "?student_id=" + url.QueryEscape(strconv.FormatInt(studentID, 10))
	http.Redirect(w, r, tableStudentClassCode, http.StatusSeeOther)
}
