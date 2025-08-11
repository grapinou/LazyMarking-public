package students

import (
	"database/sql"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/grapinou/LazyMarking/internal/config"
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

	classFilter := r.URL.Query().Get("class_filter")

	studentsRows, err := queries.GetStudentsWithClasses(r.Context(), db.GetStudentsWithClassesParams{
		UserID:      userID,
		ClassFilter: classFilter,
	})
	if err != nil {
		log.Printf("From TableStudentsHandler -> GetStudentsWithClasses DB error: %v", err)
		http.Error(w, "DB error", http.StatusInternalServerError)
		return
	}

	// Regroupement des données par étudiant
	studentsMap := make(map[int64]*config.Student)
	for _, row := range studentsRows {
		st, exists := studentsMap[row.StudentID]
		if !exists {
			st = &config.Student{
				ID:         row.StudentID,
				FirstName:  row.StudentFirstName,
				LastName:   row.StudentLastName,
				ClassCodes: []config.ClassCode{},
			}
			studentsMap[row.StudentID] = st
		}

		// Ajoute la classe si elle existe (LEFT JOIN → peut être NULL)
		if row.ClassID.Valid && row.ClassName.Valid {
			st.ClassCodes = append(st.ClassCodes, config.ClassCode{
				ID:   row.ClassID.Int64,
				Name: row.ClassName.String,
			})
		}
	}

	// Convertir map → slice
	students := make([]config.Student, 0, len(studentsMap))
	for _, s := range studentsMap {
		students = append(students, *s)
	}

	// Requête pour récupérer toutes les classes
	classCodesRows, err := queries.ListClassCodesByUser(r.Context(), userID)
	if err != nil {
		log.Printf("From TableStudentsHandler -> GetStudentsWithClasses DB error: %v", err)
		http.Error(w, "DB error", http.StatusInternalServerError)
		return
	}

	noStudent := true
	if len(students) > 0 {
		noStudent = false
	}

	noClassCode := true
	if len(classCodesRows) > 0 {
		noClassCode = false
	}

	var actionsURLParameters []data.StudentActionURLs
	if !noStudent {
		for _, student := range students {
			params := "?student_id=" + url.QueryEscape(strconv.FormatInt(student.ID, 10))
			editURL := data.DefaultStudentRoutes.EditURL + params
			deleteURL := data.DefaultStudentRoutes.DeleteURL + params
			studentClassCodes := data.DefaultStudentRoutes.StudentClassCodesURL + params

			actionsURLParameters = append(actionsURLParameters, data.StudentActionURLs{
				EditURL:              editURL,
				DeleteURL:            deleteURL,
				StudentClassCodesURL: studentClassCodes,
			})
		}
	}

	dataPage := data.StudentPageData{
		Routes:        data.DefaultDashboardRoutes,
		StudentRoutes: data.DefaultStudentRoutes,
		PageTitle:     "students",
		ExtraData: map[string]any{
			"NoStudent":          noStudent,
			"Action":             actionsURLParameters,
			"Students":           students,
			"NoClassCode":        noClassCode,
			"ClassCodes":         classCodesRows,
			"CurrentClassFilter": classFilter,
		},
	}

	RenderTableStudentPage(w, dataPage)
}

func AddFormStudentHandler(w http.ResponseWriter, r *http.Request, queries *db.Queries) {
	userID, _, ok := tools.CheckRequest(w, r, http.MethodGet)
	if !ok {
		log.Println("From AddFormStudentHandler -> tools.CheckRequest return not ok")
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

	studentID, err := queries.CreateStudentAndReturnID(r.Context(), db.CreateStudentAndReturnIDParams{
		FirstName: firstName,
		LastName:  lastName,
		UserID:    userID,
	})
	if err != nil {
		log.Printf("From AddStudentHandler -> DB CreateStudent error : %v", err)
		errorMessage := url.QueryEscape("Il ne peut pas exister deux fois le même étudiant ou un étudiant ne peut pas être sans nom.")
		http.Redirect(w, r, data.ErrorMessageURL+"?errormessage="+errorMessage, http.StatusSeeOther)
		return
	}

	if err := queries.CreateStudentWithClassCode(r.Context(), db.CreateStudentWithClassCodeParams{
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

func AddCSVFormStudentHandler(w http.ResponseWriter, r *http.Request, queries *db.Queries) {
	userID, _, ok := tools.CheckRequest(w, r, http.MethodGet)
	if !ok {
		log.Println("From AddCSVFormStudentHandler -> tools.CheckRequest return not ok")
		return
	}

	allClassCodes, err := queries.GetAllClassCodes(r.Context(), userID)
	if err != nil {
		log.Printf("From AddCSVFormStudentHandler -> GetAllClassCodes error DB : %v", err)
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
		PageTitle:     "add csv",
		ExtraData: map[string]any{
			"ClassCodes": allClassCodes,
		},
	}

	RenderAddCSVFormStudentPage(w, dataPage)
}

func AddCSVStudentHandler(w http.ResponseWriter, r *http.Request, queries *db.Queries, conn *sql.DB) {
	userID, _, ok := tools.CheckRequest(w, r, http.MethodPost)
	if !ok {
		log.Println("From AddCSVStudentHandler -> tools.CheckRequest return not ok")
		return
	}

	classCodeIDStr := r.FormValue("class_code_id")
	classCodeID, err := strconv.ParseInt(classCodeIDStr, 10, 64)
	if err != nil {
		log.Printf("From AddCSVStudentHandler -> strconv.ParseInt : can't convert class code id : error : %v", err)
		http.Error(w, "Something went wrong !", http.StatusBadRequest)
		return
	}

	file, err := tools.CheckCSVFile(r, 2<<20) // 2 Mo
	if err != nil {
		log.Printf("From AddCSVStudentHandler -> CheckCSVFile : error : %v", err)
		errorMessage := url.QueryEscape("Fichier probablement trop volumineux ou le fichier n'est pas un csv.")
		http.Redirect(w, r, data.ErrorMessageURL+"?errormessage="+errorMessage, http.StatusSeeOther)
		return
	}
	defer func() {
		if closer, ok := file.(io.Closer); ok {
			closer.Close()
		}
	}()

	records, err := tools.ValidateCSVStructure(file)
	if err != nil {
		log.Printf("From AddCSVStudentHandler -> ValidateCSVStructure : error : %v", err)
		errorMessage := url.QueryEscape("Problème d'intégrité des données. Vérifier que le csv est de type \"Jean\";\"Gabin\"")
		http.Redirect(w, r, data.ErrorMessageURL+"?errormessage="+errorMessage, http.StatusSeeOther)
		return
	}

	tx, err := conn.BeginTx(r.Context(), nil)
	if err != nil {
		log.Printf("Failed to begin transaction: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	defer tx.Rollback()       // rollback automatique en cas d'erreur
	qtx := queries.WithTx(tx) //

	for _, record := range records {

		studentID, err := qtx.CreateStudentAndReturnID(r.Context(), db.CreateStudentAndReturnIDParams{
			FirstName: record[0],
			LastName:  record[1],
			UserID:    userID,
		})
		if err != nil {
			log.Printf("From AddCSVStudentHandler -> DB CreateStudentAndReturnID error : %v", err)
			errorMessage := url.QueryEscape(fmt.Sprintf("Il ne peut pas exister deux fois le même étudiant. L'étudiant suivant est en double : %s %s", record[0], record[1]))
			http.Redirect(w, r, data.ErrorMessageURL+"?errormessage="+errorMessage, http.StatusSeeOther)
			return
		}

		if err = qtx.CreateStudentWithClassCode(r.Context(), db.CreateStudentWithClassCodeParams{
			StudentID:   studentID,
			ClassCodeID: classCodeID,
			UserID:      userID,
		}); err != nil {
			log.Printf("From AddStudentHandler -> DB CreateStudentWithClassCode error : %v", err)
			http.Error(w, "Something went wrong !", http.StatusInternalServerError)
			return
		}
	}

	if err := tx.Commit(); err != nil {
		log.Printf("Transaction commit error: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, data.DefaultDashboardRoutes.StudentURL, http.StatusSeeOther)
}

func DeleteFormAllStudentsHandler(w http.ResponseWriter, r *http.Request, queries *db.Queries) {
	userID, _, ok := tools.CheckRequest(w, r, http.MethodGet)
	if !ok {
		log.Println("From DeleteFormAllStudentsHandler -> tools.CheckRequest return not ok")
		return
	}

	classCodeIDStr := r.URL.Query().Get("class_code_id")
	if classCodeIDStr == "" {
		log.Println("From DeleteFormAllStudentsHandler : no class code id parameter")
		errorMessage := url.QueryEscape("Il faut sélectionner une classe. Impossible de toutes les supprimer.")
		http.Redirect(w, r, data.ErrorMessageURL+"?errormessage="+errorMessage, http.StatusSeeOther)
		return

	}

	classCodeID, err := strconv.ParseInt(classCodeIDStr, 10, 64)
	if err != nil {
		log.Printf("From DeleteFormAllStudentsHandler -> strconv.ParseInt, invalid class code ID, error : %v", err)
		http.Error(w, "Something went wrong !", http.StatusBadRequest)
		return
	}

	classCodeName, err := queries.GetClassCodeNameByID(r.Context(), db.GetClassCodeNameByIDParams{
		ID:     classCodeID,
		UserID: userID,
	})
	if err != nil {
		log.Printf("From DeleteFormAllStudentsHandler -> GetClassCodeNameByID, DB error : %v", err)
		http.Error(w, "Something went wrong !", http.StatusInternalServerError)
		return
	}

	nbrStudent, err := queries.CountStudentsInClass(r.Context(), db.CountStudentsInClassParams{
		ClassCodeID: classCodeID,
		UserID:      userID,
	})
	if err != nil {
		log.Printf("From DeleteFormAllStudentsHandler -> CountStudentsInClass, DB error : %v", err)
		http.Error(w, "Something went wrong !", http.StatusInternalServerError)
		return
	}
	if nbrStudent == 0 {
		errorMessage := url.QueryEscape("Aucun n'élève à supprimer dans cette classe.")
		http.Redirect(w, r, data.ErrorMessageURL+"?errormessage="+errorMessage, http.StatusSeeOther)
		return
	}

	dataPage := data.StudentPageData{
		Routes:        data.DefaultDashboardRoutes,
		StudentRoutes: data.DefaultStudentRoutes,
		PageTitle:     "delete all student",
		ExtraData: map[string]any{
			"ClassCodeName": classCodeName,
			"ClassCodeID":   classCodeIDStr,
		},
	}

	RenderDeleteFormAllStudentsPage(w, dataPage)
}

func DeleteAllStudentsHandler(w http.ResponseWriter, r *http.Request, queries *db.Queries) {
	userID, _, ok := tools.CheckRequest(w, r, http.MethodPost)
	if !ok {
		log.Println("From DeleteAllStudentsHandler -> tools.CheckRequest return not ok")
		return
	}

	classCodeIDStr := r.FormValue("class_code_id")
	if classCodeIDStr == "" {
		log.Println("From DeleteAllStudentsHandler : no class code id parameter")
		http.Error(w, "Something went wrong !", http.StatusBadRequest)
		return
	}

	classCodeID, err := strconv.ParseInt(classCodeIDStr, 10, 64)
	if err != nil {
		log.Printf("From DeleteAllStudentsHandler -> strconv.ParseInt, invalid class code ID, error : %v", err)
		http.Error(w, "Something went wrong !", http.StatusBadRequest)
		return
	}

	if err := queries.DeleteStudentsOnlyInOneClass(r.Context(), db.DeleteStudentsOnlyInOneClassParams{
		ClassCodeID: classCodeID,
		UserID:      userID,
	}); err != nil {
		log.Printf("From DeleteStudentHandler -> DeleteStudent DB error: %v", err)
		http.Error(w, "Something went wrong !", http.StatusInternalServerError)
		return
	}

	if err := queries.DeleteStudentsWithSeveralClass(r.Context(), db.DeleteStudentsWithSeveralClassParams{
		ClassCodeID: classCodeID,
		UserID:      userID,
	}); err != nil {
		log.Printf("From DeleteStudentHandler -> DeleteStudentsWithSeveralClass DB error: %v", err)
		http.Error(w, "Something went wrong !", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, data.DefaultDashboardRoutes.StudentURL, http.StatusSeeOther)
}
