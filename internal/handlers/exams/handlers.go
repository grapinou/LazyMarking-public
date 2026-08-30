package exams

import (
	"context"
	"database/sql"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/grapinou/LazyMarking/internal/db"
	"github.com/grapinou/LazyMarking/internal/handlers/tools"
	"github.com/grapinou/LazyMarking/internal/templates/data"
)

var afterExamDeletePrecheck = func(context.Context, *db.Queries, int64, int64) error { return nil }
var afterExamEditPrecheck = func(context.Context, *db.Queries, int64, int64) error { return nil }

func buildExamListItems(exams []db.GetExamsAllInfosRow, routes data.ExamRoutes) []data.ExamListItem {
	items := make([]data.ExamListItem, 0, len(exams))
	for _, exam := range exams {
		item := data.ExamListItem{
			ID:               exam.ID,
			Name:             exam.ExamName,
			QCMName:          exam.QcmName,
			ClassName:        exam.ClassCodeName,
			YearName:         exam.YearName,
			PeriodName:       exam.PeriodName,
			GenerationStatus: examGenerationStatus(exam.GenerationStatus),
		}
		switch item.GenerationStatus {
		case data.ExamGenerationDraft:
			item.EditURL = examURL(routes.EditURL, exam.ID)
			item.DeleteURL = examURL(routes.DeleteURL, exam.ID)
			item.GenerateURL = examURL(routes.GenerateExamPdf, exam.ID)
			item.MiniURL = examURL(routes.GenerateMiniPdf, exam.ID)
		case data.ExamGenerationRunning, data.ExamGenerationSuccess:
			if exam.GenerationID.Valid {
				item.GenerationURL = examGenerationURL(exam.GenerationID.Int64, exam.ID)
			}
		}
		items = append(items, item)
	}
	return items
}

func examGenerationStatus(status sql.NullString) data.ExamGenerationStatus {
	if !status.Valid {
		return data.ExamGenerationDraft
	}
	return data.ExamGenerationStatus(status.String)
}

func examURL(base string, examID int64) string {
	return base + "?exam_id=" + url.QueryEscape(strconv.FormatInt(examID, 10))
}

func examGenerationURL(generationID, examID int64) string {
	return data.DefaultGenerateExamRoutes.ProcessingStudents +
		"?exam_generated_id=" + url.QueryEscape(strconv.FormatInt(generationID, 10)) +
		"&exam_id=" + url.QueryEscape(strconv.FormatInt(examID, 10)) +
		"&generation_started=1"
}

func buildExamFormData(qcms []db.GetAllQCMRow, classes []db.ClassCode, years []db.Year, periods []db.Period, exam db.Exam) data.ExamFormData {
	return data.ExamFormData{
		QCMs:             qcms,
		Classes:          classes,
		Years:            years,
		Periods:          periods,
		Name:             exam.Name,
		SelectedQCMID:    exam.QcmID,
		SelectedClassID:  exam.ClassCodeID,
		SelectedYearID:   exam.YearID,
		SelectedPeriodID: exam.PeriodID,
	}
}

func buildAddExamPageData(qcms []db.GetAllQCMRow, classes []db.ClassCode, years []db.Year, periods []db.Period) data.ExamPageData {
	return data.ExamPageData{
		Routes:     data.DefaultDashboardRoutes,
		ExamRoutes: data.DefaultExamRoutes,
		PageTitle:  "add exam",
		Form:       buildExamFormData(qcms, classes, years, periods, db.Exam{}),
		CancelURL:  data.DefaultDashboardRoutes.ExamURL,
	}
}

func buildEditExamPageData(exam db.Exam, qcms []db.GetAllQCMRow, classes []db.ClassCode, years []db.Year, periods []db.Period) data.ExamPageData {
	return data.ExamPageData{
		Routes:     data.DefaultDashboardRoutes,
		ExamRoutes: data.DefaultExamRoutes,
		PageTitle:  "edit question",
		Exam:       data.ExamContext{ID: exam.ID, Name: exam.Name},
		Form:       buildExamFormData(qcms, classes, years, periods, exam),
		CancelURL:  data.DefaultDashboardRoutes.ExamURL,
	}
}

func buildDeleteExamPageData(exam db.Exam) data.ExamPageData {
	return data.ExamPageData{
		Routes:     data.DefaultDashboardRoutes,
		ExamRoutes: data.DefaultExamRoutes,
		PageTitle:  "delete exam",
		Exam:       data.ExamContext{ID: exam.ID, Name: exam.Name},
		CancelURL:  data.DefaultDashboardRoutes.ExamURL,
	}
}

func TableExamsHandler(w http.ResponseWriter, r *http.Request, queries *db.Queries) {
	userID, _, ok := tools.CheckRequest(w, r, http.MethodGet)
	if !ok {
		log.Println("From TableExamsHandler -> tools.CheckRequest return not ok")
		return
	}

	examsDB, err := queries.GetExamsAllInfos(r.Context(), userID)
	if err != nil {
		log.Printf("From TableExamsHandler -> GetExamsAllInfos DB error: %v", err)
		http.Error(w, "DB Error", http.StatusInternalServerError)
		return
	}

	dataPage := data.ExamPageData{
		Routes:     data.DefaultDashboardRoutes,
		ExamRoutes: data.DefaultExamRoutes,
		PageTitle:  "exams",
		Items:      buildExamListItems(examsDB, data.DefaultExamRoutes),
	}

	RenderTableExamPage(w, dataPage)
}

func AddFormExamHandler(w http.ResponseWriter, r *http.Request, queries *db.Queries) {
	userID, _, ok := tools.CheckRequest(w, r, http.MethodGet)
	if !ok {
		log.Println("From AddFormexamHandler -> tools.CheckRequest return not ok")
		return
	}

	qcm, err := queries.GetAllQCM(r.Context(), userID)
	if err != nil {
		log.Printf("From AddFormexamHandler -> GetAllQCM DB error: %v", err)
		http.Error(w, "DB Error", http.StatusInternalServerError)
		return
	}

	classcodes, err := queries.GetAllClassCodes(r.Context(), userID)
	if err != nil {
		log.Printf("From AddFormexamHandler -> GetAllClassCodes DB error: %v", err)
		http.Error(w, "DB Error", http.StatusInternalServerError)
		return
	}

	years, err := queries.GetAllYears(r.Context(), userID)
	if err != nil {
		log.Printf("From AddFormexamHandler -> GetAllYears DB error: %v", err)
		http.Error(w, "DB Error", http.StatusInternalServerError)
		return
	}

	periods, err := queries.GetAllPeriods(r.Context(), userID)
	if err != nil {
		log.Printf("From AddFormexamHandler -> GetAllPeriods DB error: %v", err)
		http.Error(w, "DB Error", http.StatusInternalServerError)
		return
	}

	RenderAddFormExamPage(w, buildAddExamPageData(qcm, classcodes, years, periods))
}

func AddExamHandler(w http.ResponseWriter, r *http.Request, queries *db.Queries) {
	userID, _, ok := tools.CheckRequest(w, r, http.MethodPost)
	if !ok {
		log.Println("From AddExamHandler -> tools.CheckRequest return not ok")
		return
	}
	exam := strings.TrimSpace(r.FormValue("exam"))
	if exam == "" {
		redirectBlankExamNameError(w, r)
		return
	}

	qcmIDStr := r.FormValue("qcm_id")
	if qcmIDStr == "" {
		log.Println("From AddExamHandler : no qcm id parameter")
		http.Error(w, "Something went wrong !", http.StatusBadRequest)
		return
	}
	qcmID, err := strconv.ParseInt(qcmIDStr, 10, 64)
	if err != nil {
		log.Printf("From AddExamHandler -> strconv.ParseInt, invalid qcm ID, error : %v", err)
		http.Error(w, "Something went wrong !", http.StatusBadRequest)
		return
	}

	classCodeIDStr := r.FormValue("class_code_id")
	if classCodeIDStr == "" {
		log.Println("From AddExamHandler : no class code id parameter")
		http.Error(w, "Something went wrong !", http.StatusBadRequest)
		return
	}
	classCodeID, err := strconv.ParseInt(classCodeIDStr, 10, 64)
	if err != nil {
		log.Printf("From AddExamHandler -> strconv.ParseInt, invalid class code ID, error : %v", err)
		http.Error(w, "Something went wrong !", http.StatusBadRequest)
		return
	}

	periodIDStr := r.FormValue("period_id")
	if periodIDStr == "" {
		log.Println("From AddExamHandler : no period id parameter")
		http.Error(w, "Something went wrong !", http.StatusBadRequest)
		return
	}
	periodID, err := strconv.ParseInt(periodIDStr, 10, 64)
	if err != nil {
		log.Printf("From AddExamHandler -> strconv.ParseInt, invalid period ID, error : %v", err)
		http.Error(w, "Something went wrong !", http.StatusBadRequest)
		return
	}

	yearIDStr := r.FormValue("year_id")
	if yearIDStr == "" {
		log.Println("From AddExamHandler : no year id parameter")
		http.Error(w, "Something went wrong !", http.StatusBadRequest)
		return
	}
	yearID, err := strconv.ParseInt(yearIDStr, 10, 64)
	if err != nil {
		log.Printf("From AddExamHandler -> strconv.ParseInt, invalid year ID, error : %v", err)
		http.Error(w, "Something went wrong !", http.StatusBadRequest)
		return
	}

	rows, err := queries.CreateExam(r.Context(), db.CreateExamParams{
		Name:        exam,
		QcmID:       qcmID,
		ClassCodeID: classCodeID,
		PeriodID:    periodID,
		YearID:      yearID,
		UserID:      userID,
	})
	if err != nil {
		log.Printf("From AddExamHandler -> CreateExam DB error: %v", err)
		if tools.IsSQLiteUniqueConstraint(err) {
			redirectDuplicateExamError(w, r)
			return
		}
		http.Error(w, "DB error", http.StatusInternalServerError)
		return
	}
	if !tools.HandleOwnedMutationRows(w, rows, "CreateExam") {
		return
	}

	http.Redirect(w, r, data.DefaultDashboardRoutes.ExamURL, http.StatusSeeOther)
}

func EditFormExamHandler(w http.ResponseWriter, r *http.Request, queries *db.Queries) {
	userID, _, ok := tools.CheckRequest(w, r, http.MethodGet)
	if !ok {
		log.Println("From EditFormExamHandler -> tools.CheckRequest return not ok")
		return
	}

	examIDStr := r.URL.Query().Get("exam_id")
	if examIDStr == "" {
		log.Println("From EditFormExamHandler : no exam id parameter")
		http.Error(w, "Something went wrong !", http.StatusBadRequest)
		return
	}

	examID, err := strconv.ParseInt(examIDStr, 10, 64)
	if err != nil {
		log.Printf("From EditFormExamHandler -> strconv.ParseInt invalid question id parameter, error : %v", err)
		http.Error(w, "Something went wrong !", http.StatusBadRequest)
		return
	}

	exam, err := queries.GetExamByID(r.Context(), db.GetExamByIDParams{
		ID:     examID,
		UserID: userID,
	})
	if err != nil {
		tools.HandleOwnedLookupError(w, err, "EditFormExamHandler GetExamByID")
		return
	}
	if !allowExamEdit(w, r, queries, examID, userID) {
		return
	}

	qcm, err := queries.GetAllQCM(r.Context(), userID)
	if err != nil {
		log.Printf("From EditFormExamHandler -> GetAllQCM DB error: %v", err)
		http.Error(w, "DB Error", http.StatusInternalServerError)
		return
	}

	classcodes, err := queries.GetAllClassCodes(r.Context(), userID)
	if err != nil {
		log.Printf("From EditFormExamHandler -> GetAllClassCodes DB error: %v", err)
		http.Error(w, "DB Error", http.StatusInternalServerError)
		return
	}

	years, err := queries.GetAllYears(r.Context(), userID)
	if err != nil {
		log.Printf("From EditFormExamHandler -> GetAllYears DB error: %v", err)
		http.Error(w, "DB Error", http.StatusInternalServerError)
		return
	}

	periods, err := queries.GetAllPeriods(r.Context(), userID)
	if err != nil {
		log.Printf("From EditFormExamHandler -> GetAllPeriods DB error: %v", err)
		http.Error(w, "DB Error", http.StatusInternalServerError)
		return
	}

	RenderEditFormExamPage(w, buildEditExamPageData(exam, qcm, classcodes, years, periods))
}

func EditExamHandler(w http.ResponseWriter, r *http.Request, queries *db.Queries) {
	userID, _, ok := tools.CheckRequest(w, r, http.MethodPost)
	if !ok {
		log.Println("From EditExamHandler -> tools.CheckRequest return not ok")
		return
	}

	examIDStr := r.FormValue("exam_id")
	if examIDStr == "" {
		log.Println("From EditExamHandler : no exam id parameter")
		http.Error(w, "Something went wrong !", http.StatusBadRequest)
		return
	}
	examID, err := strconv.ParseInt(examIDStr, 10, 64)
	if err != nil {
		log.Printf("From EditExamHandler -> strconv.ParseInt, invalid exam ID, error : %v", err)
		http.Error(w, "Something went wrong !", http.StatusBadRequest)
		return
	}
	if _, err := queries.GetExamByID(r.Context(), db.GetExamByIDParams{ID: examID, UserID: userID}); err != nil {
		tools.HandleOwnedLookupError(w, err, "EditExamHandler GetExamByID")
		return
	}
	if !allowExamEdit(w, r, queries, examID, userID) {
		return
	}
	exam := strings.TrimSpace(r.FormValue("exam"))
	if exam == "" {
		redirectBlankExamNameError(w, r)
		return
	}

	qcmIDStr := r.FormValue("qcm_id")
	if qcmIDStr == "" {
		log.Println("From EditExamHandler : no qcm id parameter")
		http.Error(w, "Something went wrong !", http.StatusBadRequest)
		return
	}
	qcmID, err := strconv.ParseInt(qcmIDStr, 10, 64)
	if err != nil {
		log.Printf("From EditExamHandler -> strconv.ParseInt, invalid qcm ID, error : %v", err)
		http.Error(w, "Something went wrong !", http.StatusBadRequest)
		return
	}

	classCodeIDStr := r.FormValue("class_code_id")
	if classCodeIDStr == "" {
		log.Println("From EditExamHandler : no class code id parameter")
		http.Error(w, "Something went wrong !", http.StatusBadRequest)
		return
	}
	classCodeID, err := strconv.ParseInt(classCodeIDStr, 10, 64)
	if err != nil {
		log.Printf("From EditExamHandler -> strconv.ParseInt, invalid class code ID, error : %v", err)
		http.Error(w, "Something went wrong !", http.StatusBadRequest)
		return
	}

	periodIDStr := r.FormValue("period_id")
	if periodIDStr == "" {
		log.Println("From EditExamHandler : no period id parameter")
		http.Error(w, "Something went wrong !", http.StatusBadRequest)
		return
	}
	periodID, err := strconv.ParseInt(periodIDStr, 10, 64)
	if err != nil {
		log.Printf("From EditExamHandler -> strconv.ParseInt, invalid period ID, error : %v", err)
		http.Error(w, "Something went wrong !", http.StatusBadRequest)
		return
	}

	yearIDStr := r.FormValue("year_id")
	if yearIDStr == "" {
		log.Println("From EditExamHandler : no year id parameter")
		http.Error(w, "Something went wrong !", http.StatusBadRequest)
		return
	}
	yearID, err := strconv.ParseInt(yearIDStr, 10, 64)
	if err != nil {
		log.Printf("From EditExamHandler -> strconv.ParseInt, invalid year ID, error : %v", err)
		http.Error(w, "Something went wrong !", http.StatusBadRequest)
		return
	}

	if err := afterExamEditPrecheck(r.Context(), queries, examID, userID); err != nil {
		log.Printf("From EditExamHandler -> after precheck error: %v", err)
		http.Error(w, "DB error", http.StatusInternalServerError)
		return
	}

	rows, err := queries.UpdateExam(r.Context(), db.UpdateExamParams{
		Name:        exam,
		QcmID:       qcmID,
		ClassCodeID: classCodeID,
		PeriodID:    periodID,
		YearID:      yearID,
		ID:          examID,
		UserID:      userID,
	})
	if err != nil {
		log.Printf("From EditExamHandler -> UpdateExam DB error: %v", err)
		if tools.IsSQLiteUniqueConstraint(err) {
			redirectDuplicateExamError(w, r)
			return
		}
		http.Error(w, "DB error", http.StatusInternalServerError)
		return
	}
	if rows == 0 {
		hasGeneration, generationErr := queries.ExamHasGeneration(r.Context(), db.ExamHasGenerationParams{ExamID: examID, UserID: userID})
		if generationErr != nil {
			log.Printf("From EditExamHandler -> ExamHasGeneration after zero-row update: %v", generationErr)
			http.Error(w, "DB error", http.StatusInternalServerError)
			return
		}
		if hasGeneration {
			redirectGeneratedExamEditError(w, r)
			return
		}
	}
	if !tools.HandleOwnedMutationRows(w, rows, "UpdateExam") {
		return
	}

	http.Redirect(w, r, data.DefaultDashboardRoutes.ExamURL, http.StatusSeeOther)
}

func redirectBlankExamNameError(w http.ResponseWriter, r *http.Request) {
	errorMessage := url.QueryEscape("Le nom de l'évaluation ne peut pas être vide.")
	http.Redirect(w, r, data.ErrorMessageURL+"?errormessage="+errorMessage, http.StatusSeeOther)
}

func redirectDuplicateExamError(w http.ResponseWriter, r *http.Request) {
	errorMessage := url.QueryEscape("Cette évaluation existe déjà ou cette combinaison est déjà utilisée.")
	http.Redirect(w, r, data.ErrorMessageURL+"?errormessage="+errorMessage, http.StatusSeeOther)
}

func allowExamEdit(w http.ResponseWriter, r *http.Request, queries *db.Queries, examID, userID int64) bool {
	hasGeneration, err := queries.ExamHasGeneration(r.Context(), db.ExamHasGenerationParams{ExamID: examID, UserID: userID})
	if err != nil {
		log.Printf("ExamHasGeneration before edit: %v", err)
		http.Error(w, "DB error", http.StatusInternalServerError)
		return false
	}
	if hasGeneration {
		redirectGeneratedExamEditError(w, r)
		return false
	}
	return true
}

func redirectGeneratedExamEditError(w http.ResponseWriter, r *http.Request) {
	errorMessage := url.QueryEscape("Cette évaluation a déjà été générée et ne peut plus être modifiée.")
	http.Redirect(w, r, data.ErrorMessageURL+"?errormessage="+errorMessage, http.StatusSeeOther)
}

func DeleteFormExamHandler(w http.ResponseWriter, r *http.Request, queries *db.Queries) {
	userID, _, ok := tools.CheckRequest(w, r, http.MethodGet)
	if !ok {
		log.Println("From DeleteFormQuestionHandler -> tools.CheckRequest return not ok")
		return
	}

	examIDStr := r.URL.Query().Get("exam_id")
	if examIDStr == "" {
		log.Println("From DeleteFormQuestionHandler : no exam id parameter")
		http.Error(w, "Something went wrong !", http.StatusBadRequest)
		return
	}

	examID, err := strconv.ParseInt(examIDStr, 10, 64)
	if err != nil {
		log.Printf("From DeleteFormQuestionHandler -> strconv.ParseInt: invalid exam id parameter, error : %v", err)
		http.Error(w, "Something went wrong !", http.StatusBadRequest)
		return
	}

	exam, err := queries.GetExamByID(r.Context(), db.GetExamByIDParams{
		ID:     examID,
		UserID: userID,
	})
	if err != nil {
		tools.HandleOwnedLookupError(w, err, "DeleteFormExamHandler GetExamByID")
		return
	}

	RenderDeleteFormExamPage(w, buildDeleteExamPageData(exam))
}

func DeleteExamHandler(w http.ResponseWriter, r *http.Request, queries *db.Queries) {
	userID, _, ok := tools.CheckRequest(w, r, http.MethodPost)
	if !ok {
		log.Println("From DeleteExamHandler -> tools.CheckRequest return not ok")
		return
	}

	examIDStr := r.FormValue("exam_id")
	if examIDStr == "" {
		log.Println("From DeleteExamHandler : no exam id parameter")
		http.Error(w, "Something went wrong !", http.StatusBadRequest)
		return
	}

	examID, err := strconv.ParseInt(examIDStr, 10, 64)
	if err != nil {
		log.Printf("From DeleteExamHandler -> strconv.ParseInt, invalid exam id, error : %v", err)
		http.Error(w, "Something went wrong !", http.StatusBadRequest)
		return
	}

	if _, err := queries.GetExamByID(r.Context(), db.GetExamByIDParams{
		ID:     examID,
		UserID: userID,
	}); err != nil {
		tools.HandleOwnedLookupError(w, err, "DeleteExamHandler GetExamByID")
		return
	}

	hasGeneration, err := queries.ExamHasGeneration(r.Context(), db.ExamHasGenerationParams{
		ExamID: examID,
		UserID: userID,
	})
	if err != nil {
		log.Printf("From DeleteExamHandler -> ExamHasGeneration DB error: %v", err)
		http.Error(w, "DB error", http.StatusInternalServerError)
		return
	}
	if hasGeneration {
		redirectGeneratedExamDeletionError(w, r)
		return
	}
	if err := afterExamDeletePrecheck(r.Context(), queries, examID, userID); err != nil {
		log.Printf("From DeleteExamHandler -> after precheck error: %v", err)
		http.Error(w, "DB error", http.StatusInternalServerError)
		return
	}

	rows, err := queries.DeleteExam(r.Context(), db.DeleteExamParams{
		ID:     examID,
		UserID: userID,
	})
	if err != nil {
		log.Printf("From DeleteExamHandler -> DeleteExam DB error: %v", err)
		if tools.IsSQLiteForeignKeyConstraint(err) {
			redirectGeneratedExamDeletionError(w, r)
			return
		}
		http.Error(w, "DB error", http.StatusInternalServerError)
		return
	}
	if !tools.HandleOwnedMutationRows(w, rows, "DeleteExam") {
		return
	}

	http.Redirect(w, r, data.DefaultDashboardRoutes.ExamURL, http.StatusSeeOther)
}

func redirectGeneratedExamDeletionError(w http.ResponseWriter, r *http.Request) {
	errorMessage := url.QueryEscape("Cette évaluation a déjà été générée et ne peut plus être supprimée.")
	http.Redirect(w, r, data.ErrorMessageURL+"?errormessage="+errorMessage, http.StatusSeeOther)
}
