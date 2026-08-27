package qcmquestions

import (
	"database/sql"
	"log"
	"net/http"
	"net/url"
	"slices"
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
		tools.HandleOwnedLookupError(w, err, "TableQCMQuestionsHandler GetQCMNameByID")
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
			deleteURL := data.DefaultQCMQuestionRoutes.DeleteURL + params

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
	if _, err := queries.GetQCMNameByID(r.Context(), db.GetQCMNameByIDParams{ID: qcmID, UserID: userID}); err != nil {
		tools.HandleOwnedLookupError(w, err, "AddFormQCMQuestionHandler GetQCMNameByID")
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

	allQuestions, err := queries.GetFilteredQuestions(r.Context(), db.GetFilteredQuestionsParams{
		UserID:       userID,
		SubjectID:    subjectID,
		ThemeID:      themeID,
		YearLevelID:  yearLevelID,
		SkillID:      skillID,
		DifficultyID: difficultyID,
		PointID:      pointID,
	})
	if err != nil {
		log.Printf("From AddFormQCMQuestionHandler -> GetFilteredQuestions DB error: %v", err)
		http.Error(w, "DB error", http.StatusInternalServerError)
		return
	}

	questionsIDsInQCM, err := queries.GetQCMQuestionsIDs(r.Context(), db.GetQCMQuestionsIDsParams{
		UserID: userID,
		QcmID:  qcmID,
	})
	if err != nil {
		log.Printf("From AddFormQCMQuestionHandler -> GetQCMQuestionIDs DB error: %v", err)
		http.Error(w, "DB error", http.StatusInternalServerError)
		return
	}

	var questions []db.GetFilteredQuestionsRow
	for _, question := range allQuestions {
		if !slices.Contains(questionsIDsInQCM, question.ID) {
			questions = append(questions, question)
		}

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

func AddQCMQuestionHandler(w http.ResponseWriter, r *http.Request, queries *db.Queries, conn *sql.DB) {
	userID, _, ok := tools.CheckRequest(w, r, http.MethodPost)
	if !ok {
		log.Println("From AddQCMQuestionHandler -> tools.CheckRequest return not ok")
		return
	}

	if err := r.ParseForm(); err != nil {
		log.Printf("From AddQCMQuestionHandler -> r.ParseForm : error : %v", err)
		http.Error(w, "Something went wrong !", http.StatusBadRequest)
		return
	}

	qcmIDStr := r.FormValue("qcm_id")
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

	questionsIDsStr := r.Form["question_ids"]

	var questionsIDs []int64
	for _, questionIDStr := range questionsIDsStr {
		questionID, err := strconv.ParseInt(questionIDStr, 10, 64)
		if err != nil {
			log.Printf("From TableQCMQuestionHandler -> strconv.ParseInt, invalid question ID, error : %v", err)
			http.Error(w, "Something went wrong !", http.StatusBadRequest)
			return
		}
		questionsIDs = append(questionsIDs, questionID)
	}

	tx, err := conn.BeginTx(r.Context(), nil)
	if err != nil {
		log.Printf("From TableQCMQuestionHandler -> conn.BeginTx : Failed to begin transaction: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	defer tx.Rollback()       // rollback automatique en cas d'erreur
	qtx := queries.WithTx(tx) //

	for _, questionID := range questionsIDs {
		rows, err := qtx.CreateQCMQuestion(r.Context(), db.CreateQCMQuestionParams{
			QcmID:      qcmID,
			QuestionID: questionID,
			UserID:     userID,
		})
		if err != nil {
			log.Printf("From TableQCMQuestionHandler -> CreateQCMQuestion : DB error: %v", err)
			errorMessage := url.QueryEscape("Il ne peut pas exister deux fois la même question dans un qcm.")
			http.Redirect(w, r, data.ErrorMessageURL+"?errormessage="+errorMessage, http.StatusSeeOther)
			return
		}
		if !tools.HandleOwnedMutationRows(w, rows, "CreateQCMQuestion") {
			return
		}
	}
	if err := tx.Commit(); err != nil {
		log.Printf("From TableQCMQuestionHandler -> Transaction commit error: %v", err)
		http.Error(w, "Something went wrong !", http.StatusInternalServerError)
		return
	}
	tableURL := data.DefaultQCMRoutes.AddQuestionURL + "?qcm_id=" + url.QueryEscape(strconv.FormatInt(qcmID, 10))
	http.Redirect(w, r, tableURL, http.StatusSeeOther)
}

func DeleteFormQCMQuestionHandler(w http.ResponseWriter, r *http.Request, queries *db.Queries) {
	userID, _, ok := tools.CheckRequest(w, r, http.MethodGet)
	if !ok {
		log.Println("From DeleteFormQCMQuestionHandler -> tools.CheckRequest return not ok")
		return
	}

	qcmIDStr := r.URL.Query().Get("qcm_id")
	if qcmIDStr == "" {
		log.Println("From DeleteFormQCMQuestionHandler : no qcm id parameter")
		http.Error(w, "Something went wrong !", http.StatusBadRequest)
		return
	}
	qcmID, err := strconv.ParseInt(qcmIDStr, 10, 64)
	if err != nil {
		log.Printf("From DeleteFormQCMQuestionHandler -> strconv.ParseInt, invalid qcm ID: %v", err)
		http.Error(w, "Something went wrong !", http.StatusBadRequest)
		return
	}
	if _, err := queries.GetQCMNameByID(r.Context(), db.GetQCMNameByIDParams{ID: qcmID, UserID: userID}); err != nil {
		tools.HandleOwnedLookupError(w, err, "DeleteFormQCMQuestionHandler GetQCMNameByID")
		return
	}

	qcmQuestionIDstr := r.URL.Query().Get("qcm_question_id")
	if qcmQuestionIDstr == "" {
		log.Println("From DeleteFormQCMQuestionHandler : no qcm question id parameter")
		http.Error(w, "Something went wrong !", http.StatusBadRequest)
		return
	}

	qcmQuestionID, err := strconv.ParseInt(qcmQuestionIDstr, 10, 64)
	if err != nil {
		log.Printf("From DeleteFormQCMQuestionHandler -> strconv.ParseInt, invalid qcm ID, error : %v", err)
		http.Error(w, "Something went wrong !", http.StatusBadRequest)
		return
	}

	questionContent, err := queries.GetQuestionContentByQCMQuestionID(r.Context(), db.GetQuestionContentByQCMQuestionIDParams{
		UserID:        userID,
		QcmQuestionID: qcmQuestionID,
		QcmID:         qcmID,
	})
	if err != nil {
		tools.HandleOwnedLookupError(w, err, "DeleteFormQCMQuestionHandler GetQuestionContentByQCMQuestionID")
		return
	}

	dataPage := data.QCMQuestionPageData{
		Routes:            data.DefaultDashboardRoutes,
		QCMQuestionRoutes: data.DefaultQCMQuestionRoutes,
		PageTitle:         "delete qcm question",
		ExtraData: map[string]any{
			"QCMID":           qcmIDStr,
			"QCMQuestionID":   qcmQuestionIDstr,
			"QuestionContent": questionContent,
		},
	}

	RenderDeleteFormQCMQuestionPage(w, dataPage)
}

func DeleteQCMQuestionHandler(w http.ResponseWriter, r *http.Request, queries *db.Queries) {
	userID, _, ok := tools.CheckRequest(w, r, http.MethodPost)
	if !ok {
		log.Println("From DeleteQCMQuestionHandler -> tools.CheckRequest return not ok")
		return
	}

	qcmIDStr := r.FormValue("qcm_id")
	if qcmIDStr == "" {
		log.Println("From DeleteQCMQuestionHandler : no qcm id parameter")
		http.Error(w, "Something went wrong !", http.StatusBadRequest)
		return
	}
	qcmID, err := strconv.ParseInt(qcmIDStr, 10, 64)
	if err != nil {
		log.Printf("From DeleteQCMQuestionHandler -> strconv.ParseInt, invalid qcm ID: %v", err)
		http.Error(w, "Something went wrong !", http.StatusBadRequest)
		return
	}

	qcmQuestionIDStr := r.FormValue("qcm_question_id")
	if qcmQuestionIDStr == "" {
		log.Println("From DeleteQCMQuestionHandler : no qcm question id parameter")
		http.Error(w, "Something went wrong !", http.StatusBadRequest)
		return
	}

	qcmQuestionID, err := strconv.ParseInt(qcmQuestionIDStr, 10, 64)
	if err != nil {
		log.Printf("From DeleteQCMQuestionHandler -> strconv.ParseInt, invalid qcm question ID, error : %v", err)
		http.Error(w, "Something went wrong !", http.StatusBadRequest)
		return
	}

	rows, err := queries.DeleteQCMQuestion(r.Context(), db.DeleteQCMQuestionParams{
		ID:     qcmQuestionID,
		QcmID:  qcmID,
		UserID: userID,
	})
	if err != nil {
		log.Printf("From DeleteQCMQuestionHandler : DeleteQCMQuestion DB error: %v", err)
		http.Error(w, "Something went wrong !", http.StatusInternalServerError)
		return
	}
	if !tools.HandleOwnedMutationRows(w, rows, "DeleteQCMQuestion") {
		return
	}

	tableURL := data.DefaultQCMRoutes.AddQuestionURL + "?qcm_id=" + url.QueryEscape(qcmIDStr)
	http.Redirect(w, r, tableURL, http.StatusSeeOther)
}
