package exams

import (
	"log"
	"net/http"
	"net/url"
	"strconv"

	"github.com/grapinou/LazyMarking/internal/db"
	"github.com/grapinou/LazyMarking/internal/handlers/tools"
	"github.com/grapinou/LazyMarking/internal/templates/data"
)

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

	noExam := true
	if len(examsDB) > 0 {
		noExam = false
	}

	var actionsURLParameters []data.ExamActionURLs
	if !noExam {
		for _, exam := range examsDB {
			params := "?exam_id=" + url.QueryEscape(strconv.FormatInt(exam.ID, 10))
			editURL := data.DefaultExamRoutes.EditURL + params
			deleteURL := data.DefaultExamRoutes.DeleteURL + params

			actionsURLParameters = append(actionsURLParameters, data.ExamActionURLs{
				EditURL:   editURL,
				DeleteURL: deleteURL,
			})
		}
	}

	dataPage := data.ExamPageData{
		Routes:     data.DefaultDashboardRoutes,
		ExamRoutes: data.DefaultExamRoutes,
		PageTitle:  "exams",
		ExtraData: map[string]any{
			"UserID": userID,
			"NoExam": noExam,
			"Exams":  examsDB,
			"Action": actionsURLParameters,
		},
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

	dataPage := data.ExamPageData{
		Routes:     data.DefaultDashboardRoutes,
		ExamRoutes: data.DefaultExamRoutes,
		PageTitle:  "add exam",
		ExtraData: map[string]any{
			"QCM":        qcm,
			"ClassCodes": classcodes,
			"Years":      years,
			"Periods":    periods,
		},
	}

	RenderAddFormExamPage(w, dataPage)
}

func AddExamHandler(w http.ResponseWriter, r *http.Request, queries *db.Queries) {
	userID, _, ok := tools.CheckRequest(w, r, http.MethodPost)
	if !ok {
		log.Println("From AddExamHandler -> tools.CheckRequest return not ok")
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

	exam := r.FormValue("exam")

	if err := queries.CreateExam(r.Context(), db.CreateExamParams{
		Name:        exam,
		QcmID:       qcmID,
		ClassCodeID: classCodeID,
		PeriodID:    periodID,
		YearID:      yearID,
		UserID:      userID,
	}); err != nil {
		log.Printf("From AddExamHandler -> DB CreateQuestion error : %v", err)
		errorMessage := url.QueryEscape("Il ne peut pas exister deux fois le même examen ou l'examen ne peut être vide.")
		http.Redirect(w, r, data.ErrorMessageURL+"?errormessage="+errorMessage, http.StatusSeeOther)
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
		log.Printf("From EditFormExamHandler -> GetExamByID DB error: %v", err)
		http.Error(w, "DB error", http.StatusInternalServerError)
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

	dataPage := data.ExamPageData{
		Routes:     data.DefaultDashboardRoutes,
		ExamRoutes: data.DefaultExamRoutes,
		PageTitle:  "edit question",
		ExtraData: map[string]any{
			"Exam":       exam,
			"ExamID":     examIDStr,
			"QCM":        qcm,
			"ClassCodes": classcodes,
			"Years":      years,
			"Periods":    periods,
		},
	}
	RenderEditFormExamPage(w, dataPage)
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

	exam := r.FormValue("exam")

	if err := queries.UpdateExam(r.Context(), db.UpdateExamParams{
		Name:        exam,
		QcmID:       qcmID,
		ClassCodeID: classCodeID,
		PeriodID:    periodID,
		YearID:      yearID,
		ID:          examID,
		UserID:      userID,
	}); err != nil {
		log.Printf("From EditQuestionHandler -> UpdateQuestion DB error: %v", err)
		errorMessage := url.QueryEscape("Il ne peut pas exister deux fois la même question ou la question ne peut être vide.")
		http.Redirect(w, r, data.ErrorMessageURL+"?errormessage="+errorMessage, http.StatusSeeOther)
		return
	}

	http.Redirect(w, r, data.DefaultDashboardRoutes.ExamURL, http.StatusSeeOther)
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
		log.Printf("From DeleteFormQuestionHandler -> GetExamByID DB error: %v", err)
		http.Error(w, "DB error", http.StatusInternalServerError)
		return
	}

	dataPage := data.ExamPageData{
		Routes:     data.DefaultDashboardRoutes,
		ExamRoutes: data.DefaultExamRoutes,
		PageTitle:  "delete exam",
		ExtraData: map[string]any{
			"Exam":   exam.Name,
			"ExamID": examIDStr,
		},
	}

	RenderDeleteFormExamPage(w, dataPage)
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

	if err := queries.DeleteExam(r.Context(), db.DeleteExamParams{
		ID:     examID,
		UserID: userID,
	}); err != nil {
		log.Printf("From DeleteExamHandler -> DeleteExam DB error: %v", err)
		//errorMessage := url.QueryEscape("La question est utilisée par un qcm. Il n'est pas possible de la supprimer.")
		//http.Redirect(w, r, data.ErrorMessageURL+"?errormessage="+errorMessage, http.StatusSeeOther)
		http.Error(w, "Something went wrong !", http.StatusBadRequest)
		return
	}

	http.Redirect(w, r, data.DefaultDashboardRoutes.ExamURL, http.StatusSeeOther)
}
