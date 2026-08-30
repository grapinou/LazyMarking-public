package students

import (
	"database/sql"
	"errors"
	"fmt"
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

func TableStudentsHandler(w http.ResponseWriter, r *http.Request, queries *db.Queries) {
	userID, _, ok := tools.CheckRequest(w, r, http.MethodGet)
	if !ok {
		log.Println("From TableStudentsHandler -> tools.CheckRequest return not ok")
		return
	}

	classFilter := r.URL.Query().Get("class_filter")

	studentRows, err := queries.GetStudentsWithClasses(r.Context(), db.GetStudentsWithClassesParams{
		UserID:      userID,
		ClassFilter: classFilter,
	})
	if err != nil {
		log.Printf("From TableStudentsHandler -> GetStudentsWithClasses DB error: %v", err)
		http.Error(w, "DB error", http.StatusInternalServerError)
		return
	}

	// Requête pour récupérer toutes les classes
	classCodesRows, err := queries.ListClassCodesByUser(r.Context(), userID)
	if err != nil {
		log.Printf("From TableStudentsHandler -> ListClassCodesByUser DB error: %v", err)
		http.Error(w, "DB error", http.StatusInternalServerError)
		return
	}

	RenderTableStudentPage(w, buildStudentListPageData(studentRows, classCodesRows, classFilter))
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

	RenderAddFormStudentPage(w, buildStudentFormPageData(allClassCodes, "add student"))
}

func AddStudentHandler(w http.ResponseWriter, r *http.Request, queries *db.Queries, conn *sql.DB) {
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

	tx, err := conn.BeginTx(r.Context(), nil)
	if err != nil {
		log.Printf("From AddStudentHandler -> begin transaction: %v", err)
		http.Error(w, "Something went wrong !", http.StatusInternalServerError)
		return
	}
	defer tx.Rollback()
	qtx := queries.WithTx(tx)

	studentID, err := qtx.CreateStudentAndReturnID(r.Context(), db.CreateStudentAndReturnIDParams{
		FirstName: firstName,
		LastName:  lastName,
		UserID:    userID,
	})
	if err != nil {
		if tools.IsSQLiteUniqueConstraint(err) {
			redirectStudentInputError(w, r, "Cet élève existe déjà.")
			return
		}
		if tools.IsSQLiteCheckConstraint(err) {
			redirectStudentInputError(w, r, "Le prénom et le nom de l’élève doivent être renseignés.")
			return
		}
		log.Printf("From AddStudentHandler -> CreateStudentAndReturnID DB error: %v", err)
		http.Error(w, "Something went wrong !", http.StatusInternalServerError)
		return
	}

	rows, err := qtx.CreateStudentWithClassCode(r.Context(), db.CreateStudentWithClassCodeParams{
		StudentID:   studentID,
		ClassCodeID: classCodeID,
		UserID:      userID,
	})
	if err != nil {
		log.Printf("From AddStudentHandler -> DB CreateStudentWithClassCode error : %v", err)
		http.Error(w, "Something went wrong !", http.StatusInternalServerError)
		return
	}
	if !tools.HandleOwnedMutationRows(w, rows, "CreateStudentWithClassCode") {
		return
	}
	if err := tx.Commit(); err != nil {
		log.Printf("From AddStudentHandler -> commit transaction: %v", err)
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
		tools.HandleOwnedLookupError(w, err, "EditFormStudentHandler GetStudentByID")
		return
	}

	RenderEditFormStudentPage(w, buildStudentContextPageData(student, "edit student"))
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

	rows, err := queries.UpdateStudent(r.Context(), db.UpdateStudentParams{
		FirstName: newFirstName,
		LastName:  newLastName,
		ID:        studentID,
		UserID:    userID,
	})
	if err != nil {
		if tools.IsSQLiteUniqueConstraint(err) {
			redirectStudentInputError(w, r, "Cet élève existe déjà.")
			return
		}
		if tools.IsSQLiteCheckConstraint(err) {
			redirectStudentInputError(w, r, "Le prénom et le nom de l’élève doivent être renseignés.")
			return
		}
		log.Printf("From EditStudentHandler -> UpdateStudent DB error: %v", err)
		http.Error(w, "Something went wrong !", http.StatusInternalServerError)
		return
	}
	if !tools.HandleOwnedMutationRows(w, rows, "UpdateStudent") {
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
		tools.HandleOwnedLookupError(w, err, "DeleteFormStudentHandler GetStudentByID")
		return
	}

	RenderDeleteFormStudentPage(w, buildStudentContextPageData(student, "delete student"))
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

	rows, err := queries.DeleteStudent(r.Context(), db.DeleteStudentParams{
		ID:     studentID,
		UserID: userID,
	})
	if err != nil {
		if isStudentExamForeignKeyConstraint(err) {
			redirectStudentHistoryProtectionError(w, r, "Cet élève ne peut pas être supprimé car il est déjà associé à une évaluation générée.")
			return
		}
		log.Printf("From DeleteStudentHandler : DeleteStudent DB error: %v", err)
		http.Error(w, "Something went wrong !", http.StatusInternalServerError)
		return
	}
	if !tools.HandleOwnedMutationRows(w, rows, "DeleteStudent") {
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

	RenderAddCSVFormStudentPage(w, buildStudentFormPageData(allClassCodes, "add csv"))
}

func AddCSVStudentHandler(w http.ResponseWriter, r *http.Request, queries *db.Queries, conn *sql.DB) {
	userID, _, ok := tools.CheckRequest(w, r, http.MethodPost)
	if !ok {
		log.Println("From AddCSVStudentHandler -> tools.CheckRequest return not ok")
		return
	}

	file, err := tools.CheckCSVFile(w, r, tools.MaxCSVRequestBytes)
	if err != nil {
		log.Printf("From AddCSVStudentHandler -> CheckCSVFile : error : %v", err)
		var maxBytesError *http.MaxBytesError
		if errors.As(err, &maxBytesError) {
			errorMessage := url.QueryEscape("La requête d’import CSV dépasse la taille maximale autorisée de 2 Mio.")
			http.Redirect(w, r, data.ErrorMessageURL+"?errormessage="+errorMessage, http.StatusSeeOther)
			return
		}
		errorMessage := url.QueryEscape("Le formulaire d’import CSV est invalide ou incomplet.")
		http.Redirect(w, r, data.ErrorMessageURL+"?errormessage="+errorMessage, http.StatusSeeOther)
		return
	}
	defer r.MultipartForm.RemoveAll()
	defer func() {
		_ = file.Close()
	}()

	classCodeIDs := r.MultipartForm.Value["class_code_id"]
	if len(classCodeIDs) == 0 {
		log.Println("From AddCSVStudentHandler -> invalid class_code_id multipart field")
		http.Error(w, "Something went wrong !", http.StatusBadRequest)
		return
	}
	classCodeID, err := strconv.ParseInt(classCodeIDs[0], 10, 64)
	if err != nil {
		log.Printf("From AddCSVStudentHandler -> strconv.ParseInt : can't convert class code id : error : %v", err)
		http.Error(w, "Something went wrong !", http.StatusBadRequest)
		return
	}

	records, err := tools.ValidateCSVStructure(file)
	if err != nil {
		log.Printf("From AddCSVStudentHandler -> ValidateCSVStructure : error : %v", err)
		errorMessage := url.QueryEscape("Problème d'intégrité des données. Vérifier que le csv est de type \"Jean\";\"Gabin\"")
		http.Redirect(w, r, data.ErrorMessageURL+"?errormessage="+errorMessage, http.StatusSeeOther)
		return
	}

	tx, err := conn.BeginTx(r.Context(), nil)
	if err != nil {
		log.Printf(" From AddCSVStudentHandler -> conn.BeginTx : Failed to begin transaction: %v", err)
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

		rows, err := qtx.CreateStudentWithClassCode(r.Context(), db.CreateStudentWithClassCodeParams{
			StudentID:   studentID,
			ClassCodeID: classCodeID,
			UserID:      userID,
		})
		if err != nil {
			log.Printf("From AddStudentHandler -> DB CreateStudentWithClassCode error : %v", err)
			http.Error(w, "Something went wrong !", http.StatusInternalServerError)
			return
		}
		if !tools.HandleOwnedMutationRows(w, rows, "CreateStudentWithClassCode") {
			return
		}
	}

	if err := tx.Commit(); err != nil {
		log.Printf("From AddStudentHandler -> Transaction commit error: %v", err)
		http.Error(w, "Something went wrong !", http.StatusInternalServerError)
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
		tools.HandleOwnedLookupError(w, err, "DeleteFormAllStudentsHandler GetClassCodeNameByID")
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

	RenderDeleteFormAllStudentsPage(w, buildStudentClassDeletePageData(classCodeID, classCodeName))
}

func DeleteAllStudentsHandler(w http.ResponseWriter, r *http.Request, queries *db.Queries, conn *sql.DB) {
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
	if _, err := queries.GetClassCodeNameByID(r.Context(), db.GetClassCodeNameByIDParams{ID: classCodeID, UserID: userID}); err != nil {
		tools.HandleOwnedLookupError(w, err, "DeleteAllStudentsHandler GetClassCodeNameByID")
		return
	}
	tx, err := conn.BeginTx(r.Context(), nil)
	if err != nil {
		log.Printf("From DeleteAllStudentsHandler -> begin transaction: %v", err)
		http.Error(w, "Something went wrong !", http.StatusInternalServerError)
		return
	}
	defer tx.Rollback()
	qtx := queries.WithTx(tx)

	// Zero rows is valid: the class may contain no students exclusive to it.
	if _, err := qtx.DeleteStudentsOnlyInOneClass(r.Context(), db.DeleteStudentsOnlyInOneClassParams{
		ClassCodeID: classCodeID,
		UserID:      userID,
	}); err != nil {
		if isStudentExamForeignKeyConstraint(err) {
			redirectStudentHistoryProtectionError(w, r, "Impossible de supprimer les élèves de cette classe car au moins un élève est déjà associé à une évaluation générée.")
			return
		}
		log.Printf("From DeleteAllStudentsHandler -> DeleteStudentsOnlyInOneClass DB error: %v", err)
		http.Error(w, "Something went wrong !", http.StatusInternalServerError)
		return
	}

	// Zero rows is valid: the class may contain no multi-class students.
	if _, err := qtx.DeleteStudentsWithSeveralClass(r.Context(), db.DeleteStudentsWithSeveralClassParams{
		ClassCodeID: classCodeID,
		UserID:      userID,
	}); err != nil {
		log.Printf("From DeleteAllStudentsHandler -> DeleteStudentsWithSeveralClass DB error: %v", err)
		http.Error(w, "Something went wrong !", http.StatusInternalServerError)
		return
	}
	if err := tx.Commit(); err != nil {
		log.Printf("From DeleteAllStudentsHandler -> commit transaction: %v", err)
		http.Error(w, "Something went wrong !", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, data.DefaultDashboardRoutes.StudentURL, http.StatusSeeOther)
}

func redirectStudentHistoryProtectionError(w http.ResponseWriter, r *http.Request, message string) {
	http.Redirect(w, r, data.ErrorMessageURL+"?errormessage="+url.QueryEscape(message), http.StatusSeeOther)
}

func redirectStudentInputError(w http.ResponseWriter, r *http.Request, message string) {
	http.Redirect(w, r, data.ErrorMessageURL+"?errormessage="+url.QueryEscape(message), http.StatusSeeOther)
}

// isStudentExamForeignKeyConstraint is intentionally narrower than the shared
// FK helper: student_exam uses SQLite's default NO ACTION and reports the
// explicit FOREIGNKEY extended code. A generic trigger constraint must remain
// a technical error for these student deletion workflows.
func isStudentExamForeignKeyConstraint(err error) bool {
	var sqliteError sqlite3.Error
	return errors.As(err, &sqliteError) && sqliteError.ExtendedCode == sqlite3.ErrConstraintForeignKey
}
