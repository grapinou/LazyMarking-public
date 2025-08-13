package qcmquestions

import (
	"log"
	"net/http"
	"net/url"
	"strconv"

	"github.com/grapinou/LazyMarking/internal/db"
	"github.com/grapinou/LazyMarking/internal/handlers/tools"
	"github.com/grapinou/LazyMarking/internal/templates/data"
)

func TableQCMQuestionsHandler(w http.ResponseWriter, r *http.Request, queries *db.Queries) {
	userID, _, ok := tools.CheckRequest(w, r, http.MethodGet)
	if !ok {
		log.Println("From TableQCMQuestionHandler -> tools.CheckRequest return not ok")
		return
	}

	qcmIDStr := r.URL.Query().Get("qcm_id")
	if qcmIDStr == "" {
		log.Println("From TableQCMQuestionHandler : no qcm id parameter")
		http.Error(w, "Something went wrong !", http.StatusBadRequest)
		return
	}

	qcmID, err := strconv.ParseInt(qcmIDStr, 10, 64)
	if err != nil {
		log.Printf("From TableQCMQuestionHandler -> strconv.ParseInt, invalid qcm ID, error : %v", err)
		http.Error(w, "Something went wrong !", http.StatusBadRequest)
		return
	}

	qcmName, err := queries.GetQCMNameByID(r.Context(), db.GetQCMNameByIDParams{
		ID:     qcmID,
		UserID: userID,
	})
	if err != nil {
		log.Printf("From TableQCMQuestionHandler -> GetQCMNameByID DB error: %v", err)
		http.Error(w, "DB error", http.StatusInternalServerError)
		return
	}

	rows, err := queries.GetAllQuestionsByQCMID(r.Context(), db.GetAllQuestionsByQCMIDParams{
		UserID: userID,
		QcmID:  qcmID,
	})
	if err != nil {
		log.Printf("From TableQCMQuestionHandler -> GetAllQuestionByQCMID DB error: %v", err)
		http.Error(w, "DB error", http.StatusInternalServerError)
		return
	}

	noRow := true
	if len(rows) > 0 {
		noRow = false
	}

	var actionsURLParameters []data.QCMQuestionActionURLs
	if !noRow {
		for _, row := range rows {

			params := "?qcm_id=" + url.QueryEscape(strconv.FormatInt(qcmID, 10)) + "&qcm_question_id=" + url.QueryEscape(strconv.FormatInt(row.QcmQuestionID, 10))
			deleteURL := data.DefaultQCMRoutes.DeleteURL + params

			actionsURLParameters = append(actionsURLParameters, data.QCMQuestionActionURLs{
				DeleteURL: deleteURL,
			})
		}
	}

	addURL := data.DefaultQCMQuestionRoutes.AddURL + "?qcm_id=" + url.QueryEscape(strconv.FormatInt(qcmID, 10))
	dataPage := data.QCMQuestionPageData{
		Routes:            data.DefaultDashboardRoutes,
		QCMQuestionRoutes: data.DefaultQCMQuestionRoutes,
		PageTitle:         "qcm",
		ExtraData: map[string]any{
			"AddURL":        addURL,
			"NoQCMQuestion": noRow,
			"Action":        actionsURLParameters,
			"QCMQuestions":  rows,
			"QCMName":       qcmName,
		},
	}

	RenderTableQCMQuestionPage(w, dataPage)
}

func AddFormQCMQuestionHandler(w http.ResponseWriter, r *http.Request, queries *db.Queries) {
	userID, _, ok := tools.CheckRequest(w, r, http.MethodGet)
	if !ok {
		log.Println("From AddFormQCMQuestionHandler -> tools.CheckRequest return not ok")
		return
	}

	qcmIDStr := r.URL.Query().Get("qcm_id")
	if qcmIDStr == "" {
		log.Println("From TableQCMQuestionHandler : no qcm id parameter")
		http.Error(w, "Something went wrong !", http.StatusBadRequest)
		return
	}

	qcmID, err := strconv.ParseInt(qcmIDStr, 10, 64)
	if err != nil {
		log.Printf("From TableQCMQuestionHandler -> strconv.ParseInt, invalid qcm ID, error : %v", err)
		http.Error(w, "Something went wrong !", http.StatusBadRequest)
		return
	}

	// Récupération des filtres depuis l'URL

	subjectID, subjectIDSelected, err := tools.GetFieldFiltered("subjectID", r)
	if err != nil {
		log.Printf("From TableQCMQuestionHandler -> tools.GetFieldFiltered, error : %v", err)
		http.Error(w, "Something went wrong !", http.StatusBadRequest)
		return
	}

	themeID, themeIDSelected, err := tools.GetFieldFiltered("themeID", r)
	if err != nil {
		log.Printf("From TableQCMQuestionHandler -> tools.GetFieldFiltered, error : %v", err)
		http.Error(w, "Something went wrong !", http.StatusBadRequest)
		return
	}

	yearLevelID, yearLevelIDSelected, err := tools.GetFieldFiltered("yearLevelID", r)
	if err != nil {
		log.Printf("From TableQCMQuestionHandler -> tools.GetFieldFiltered, error : %v", err)
		http.Error(w, "Something went wrong !", http.StatusBadRequest)
		return
	}

	skillID, skillIDSelected, err := tools.GetFieldFiltered("skillID", r)
	if err != nil {
		log.Printf("From TableQCMQuestionHandler -> tools.GetFieldFiltered, error : %v", err)
		http.Error(w, "Something went wrong !", http.StatusBadRequest)
		return
	}

	difficultyID, difficultyIDSelected, err := tools.GetFieldFiltered("difficultyID", r)
	if err != nil {
		log.Printf("From TableQCMQuestionHandler -> tools.GetFieldFiltered, error : %v", err)
		http.Error(w, "Something went wrong !", http.StatusBadRequest)
		return
	}

	pointID, pointIDSelected, err := tools.GetFieldFiltered("pointID", r)
	if err != nil {
		log.Printf("From TableQCMQuestionHandler -> tools.GetFieldFiltered, error : %v", err)
		http.Error(w, "Something went wrong !", http.StatusBadRequest)
		return
	}

	// getting all field from db

	subjects, err := queries.GetAllSubjects(r.Context(), userID)
	if err != nil {
		log.Printf("From AddFormQCMQuestionHandler -> GetAllSubjects DB error: %v", err)
		http.Error(w, "DB error", http.StatusInternalServerError)
		return
	}

	themes, err := queries.GetAllThemes(r.Context(), userID)
	if err != nil {
		log.Printf("From AddFormQCMQuestionHandler -> GetAllThemes DB error: %v", err)
		http.Error(w, "DB error", http.StatusInternalServerError)
		return
	}

	yearLevels, err := queries.GetAllYearLevels(r.Context(), userID)
	if err != nil {
		log.Printf("From AddFormQCMQuestionHandler -> GetAllYearLevels DB error: %v", err)
		http.Error(w, "DB error", http.StatusInternalServerError)
		return
	}

	skills, err := queries.GetAllSkills(r.Context(), userID)
	if err != nil {
		log.Printf("From AddFormQCMQuestionHandler -> GetAllSkills DB error: %v", err)
		http.Error(w, "DB error", http.StatusInternalServerError)
		return
	}

	difficulties, err := queries.GetAllDifficulties(r.Context(), userID)
	if err != nil {
		log.Printf("From AddFormQCMQuestionHandler -> GetAllDifficulties DB error: %v", err)
		http.Error(w, "DB error", http.StatusInternalServerError)
		return
	}

	points, err := queries.GetAllPoints(r.Context(), userID)
	if err != nil {
		log.Printf("From AddFormQCMQuestionHandler -> GetAllPoints DB error: %v", err)
		http.Error(w, "DB error", http.StatusInternalServerError)
		return
	}

	questions, err := queries.GetFilteredQuestions(r.Context(), db.GetFilteredQuestionsParams{
		UserID:       userID,
		SubjectID:    subjectID,
		ThemeID:      themeID,
		YearLevelID:  yearLevelID,
		SkillID:      skillID,
		DifficultyID: difficultyID,
		PointID:      pointID,
	})
	if err != nil {
		log.Printf("From AddFormQCMQuestionHandler -> GetAllQuestions DB error: %v", err)
		http.Error(w, "DB error", http.StatusInternalServerError)
		return
	}

	addURL := data.DefaultQCMQuestionRoutes.AddURL + "?qcm_id=" + url.QueryEscape(strconv.FormatInt(qcmID, 10))
	tableURL := data.DefaultQCMRoutes.AddQuestionURL + "?qcm_id=" + url.QueryEscape(strconv.FormatInt(qcmID, 10))
	dataPage := data.QCMQuestionPageData{
		Routes:            data.DefaultDashboardRoutes,
		QCMQuestionRoutes: data.DefaultQCMQuestionRoutes,
		PageTitle:         "add qcm question",
		ExtraData: map[string]any{
			"AddURL":               addURL,
			"TableURL":             tableURL,
			"QCMID":                qcmID,
			"Subjects":             subjects,
			"Themes":               themes,
			"YearLevels":           yearLevels,
			"Skills":               skills,
			"Difficulties":         difficulties,
			"Points":               points,
			"Questions":            questions,
			"SelectedSubjectID":    subjectIDSelected,
			"SelectedThemeID":      themeIDSelected,
			"SelectedYearLevelID":  yearLevelIDSelected,
			"SelectedSkillID":      skillIDSelected,
			"SelectedDifficultyID": difficultyIDSelected,
			"SelectedPointID":      pointIDSelected,
		},
	}

	RenderAddFormQCMQuestionPage(w, dataPage)
}

/*
func AddQCMHandler(w http.ResponseWriter, r *http.Request, queries *db.Queries) {
	userID, _, ok := tools.CheckRequest(w, r, http.MethodPost)
	if !ok {
		log.Println("From AddQCMHandler -> tools.CheckRequest return not ok")
		return
	}

	name := strings.TrimSpace(r.FormValue("qcm"))

	if err := queries.CreateQCM(r.Context(), db.CreateQCMParams{
		Name:   name,
		UserID: userID,
	}); err != nil {
		log.Printf("From AddQCMHandler -> CreateQCM : DB error: %v", err)
		errorMessage := url.QueryEscape("Il ne peut pas exister deux fois le même nom pour un qcm ou un qcm ne peut pas avoir un nom vide.")
		http.Redirect(w, r, data.ErrorMessageURL+"?errormessage="+errorMessage, http.StatusSeeOther)
		return
	}

	http.Redirect(w, r, data.DefaultDashboardRoutes.QcmURL, http.StatusSeeOther)
}

func DeleteFormQCMHandler(w http.ResponseWriter, r *http.Request, queries *db.Queries) {
	userID, _, ok := tools.CheckRequest(w, r, http.MethodGet)
	if !ok {
		log.Println("From DeleteFormQCMHandler -> tools.CheckRequest return not ok")
		return
	}

	qcmIDStr := r.URL.Query().Get("qcm_id")
	if qcmIDStr == "" {
		log.Println("From DeleteFormQCMHandler : no qcm id parameter")
		http.Error(w, "Something went wrong !", http.StatusBadRequest)
		return
	}

	qcmID, err := strconv.ParseInt(qcmIDStr, 10, 64)
	if err != nil {
		log.Printf("From DeleteFormQCMHandler -> strconv.ParseInt, invalid qcm ID, error : %v", err)
		http.Error(w, "Something went wrong !", http.StatusBadRequest)
		return
	}

	qcm, err := queries.GetQCMNameByID(r.Context(), db.GetQCMNameByIDParams{
		ID:     qcmID,
		UserID: userID,
	})
	if err != nil {
		log.Printf("From DeleteFormQCMHandler -> GetQCMNameByID DB error: %v", err)
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	dataPage := data.QCMPageData{
		Routes:    data.DefaultDashboardRoutes,
		QCMRoutes: data.DefaultQCMRoutes,
		PageTitle: "delete qcm",
		ExtraData: map[string]any{
			"QCM":   qcm,
			"QCMID": qcmIDStr,
		},
	}

	RenderDeleteFormQCMPage(w, dataPage)
}

func DeleteQCMHandler(w http.ResponseWriter, r *http.Request, queries *db.Queries) {
	userID, _, ok := tools.CheckRequest(w, r, http.MethodPost)
	if !ok {
		log.Println("From DeleteQCMHandler -> tools.CheckRequest return not ok")
		return
	}

	qcmIDStr := r.FormValue("qcm_id")
	if qcmIDStr == "" {
		log.Println("From DeleteQCMHandler : no qcm id parameter")
		http.Error(w, "Something went wrong !", http.StatusBadRequest)
		return
	}

	qcmID, err := strconv.ParseInt(qcmIDStr, 10, 64)
	if err != nil {
		log.Printf("From DeleteQCMHandler -> strconv.ParseInt, invalid qcm ID, error : %v", err)
		http.Error(w, "Something went wrong !", http.StatusBadRequest)
		return
	}

	if err := queries.DeleteQCM(r.Context(), db.DeleteQCMParams{
		ID:     qcmID,
		UserID: userID,
	}); err != nil {
		log.Printf("From DeleteQCMHandler : DeleteQCM DB error: %v", err)
		http.Error(w, "Something went wrong !", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, data.DefaultDashboardRoutes.QcmURL, http.StatusSeeOther)
}
*/
