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
		tools.HandleOwnedLookupError(w, err, "TableStudentClassCodesHandler GetStudentByID")
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
	}

	RenderTableStudentClassCodesPage(w, buildStudentClassListPageData(student, classCodes))
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
	student, err := queries.GetStudentByID(r.Context(), db.GetStudentByIDParams{ID: studentID, UserID: userID})
	if err != nil {
		tools.HandleOwnedLookupError(w, err, "AddFormStudentClassCodeHandler GetStudentByID")
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

	RenderAddFormStudentClassCodePage(w, buildStudentClassFormPageData(student, classCodes))
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

	rows, err := queries.CreateStudentWithClassCode(r.Context(), db.CreateStudentWithClassCodeParams{
		StudentID:   studentID,
		ClassCodeID: classCodeID,
		UserID:      userID,
	})
	if err != nil {
		log.Printf("From AddStudentClassCodeHandler -> CreateStudentWithClassCode : DB error: %v", err)
		errorMessage := url.QueryEscape("Il ne peut pas exister deux fois la même classe.")
		http.Redirect(w, r, data.ErrorMessageURL+"?errormessage="+errorMessage, http.StatusSeeOther)
		return
	}
	if !tools.HandleOwnedMutationRows(w, rows, "CreateStudentWithClassCode") {
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
	if _, err := queries.GetStudentByID(r.Context(), db.GetStudentByIDParams{ID: studentID, UserID: userID}); err != nil {
		tools.HandleOwnedLookupError(w, err, "DeleteStudentClassCodeHandler GetStudentByID")
		return
	}
	if _, err := queries.GetClassCodeNameByID(r.Context(), db.GetClassCodeNameByIDParams{ID: classCodeID, UserID: userID}); err != nil {
		tools.HandleOwnedLookupError(w, err, "DeleteStudentClassCodeHandler GetClassCodeNameByID")
		return
	}

	classCodeIDs, err := queries.GetAllClassCodesByStudentID(r.Context(), db.GetAllClassCodesByStudentIDParams{
		StudentID: studentID,
		UserID:    userID,
	})
	if err != nil {
		log.Printf("From DeleteStudentClassCodeHandler : GetAllClassCodesByStudentID DB error: %v", err)
		http.Error(w, "Something went wrong !", http.StatusInternalServerError)
		return
	}
	if !containsClassCodeID(classCodeIDs, classCodeID) {
		tools.HandleOwnedMutationRows(w, 0, "DeleteStudentClassCodeByStudentID")
		return
	}
	if len(classCodeIDs) <= 1 {
		redirectLastStudentClassError(w, r)
		return
	}

	rows, err := queries.DeleteStudentClassCodeByStudentID(r.Context(), db.DeleteStudentClassCodeByStudentIDParams{
		StudentID:   studentID,
		ClassCodeID: classCodeID,
		UserID:      userID,
	})
	if err != nil {
		log.Printf("From DeleteStudentClassCodeHandler : DeleteStudentClassCodeByStudentID DB error: %v", err)
		http.Error(w, "Something went wrong !", http.StatusInternalServerError)
		return
	}
	if rows == 0 {
		// The DELETE itself requires another relation to exist, so this branch also
		// closes the race where concurrent requests both observed two classes.
		remainingClassCodeIDs, lookupErr := queries.GetAllClassCodesByStudentID(r.Context(), db.GetAllClassCodesByStudentIDParams{
			StudentID: studentID,
			UserID:    userID,
		})
		if lookupErr != nil {
			log.Printf("From DeleteStudentClassCodeHandler : classify zero-row delete: %v", lookupErr)
			http.Error(w, "Something went wrong !", http.StatusInternalServerError)
			return
		}
		if containsClassCodeID(remainingClassCodeIDs, classCodeID) {
			redirectLastStudentClassError(w, r)
			return
		}
		tools.HandleOwnedMutationRows(w, rows, "DeleteStudentClassCodeByStudentID")
		return
	}

	tableStudentClassCode := data.DefaultStudentRoutes.StudentClassCodesURL + "?student_id=" + url.QueryEscape(strconv.FormatInt(studentID, 10))
	http.Redirect(w, r, tableStudentClassCode, http.StatusSeeOther)
}

func containsClassCodeID(classCodeIDs []int64, target int64) bool {
	for _, classCodeID := range classCodeIDs {
		if classCodeID == target {
			return true
		}
	}
	return false
}

func redirectLastStudentClassError(w http.ResponseWriter, r *http.Request) {
	errorMessage := url.QueryEscape("Un élève doit toujours appartenir à au moins une classe. La dernière classe ne peut pas être retirée.")
	http.Redirect(w, r, data.ErrorMessageURL+"?errormessage="+errorMessage, http.StatusSeeOther)
}
