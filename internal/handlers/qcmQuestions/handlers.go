package qcmquestions

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"slices"
	"strconv"

	"github.com/grapinou/LazyMarking/internal/db"
	"github.com/grapinou/LazyMarking/internal/handlers/tools"
	"github.com/grapinou/LazyMarking/internal/questionfamilies"
	"github.com/grapinou/LazyMarking/internal/templates/data"
)

var (
	renderTableQCMQuestionPage      = RenderTableQCMQuestionPage
	renderAddFormQCMQuestionPage    = RenderAddFormQCMQuestionPage
	renderDeleteFormQCMQuestionPage = RenderDeleteFormQCMQuestionPage
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

	questions := make([]data.QCMQuestionItem, 0, len(rows))
	for index, row := range rows {
		params := "?qcm_id=" + url.QueryEscape(strconv.FormatInt(qcmID, 10)) + "&qcm_question_id=" + url.QueryEscape(strconv.FormatInt(row.QcmQuestionID, 10))
		questions = append(questions, data.QCMQuestionItem{
			QCMQuestionID: row.QcmQuestionID,
			Position:      row.Position,
			Content:       row.QuestionContent,
			IsFirst:       index == 0,
			IsLast:        index == len(rows)-1,
			MoveUpURL:     data.DefaultQCMQuestionRoutes.MoveUpURL,
			MoveDownURL:   data.DefaultQCMQuestionRoutes.MoveDownURL,
			DeleteURL:     data.DefaultQCMQuestionRoutes.DeleteURL + params,
		})
	}

	dataPage := data.QCMQuestionPageData{
		Routes:              data.DefaultDashboardRoutes,
		QCMQuestionRoutes:   data.DefaultQCMQuestionRoutes,
		QCMContext:          data.QCMContext{ID: qcmID, Name: qcmName},
		QCMQuestions:        questions,
		AddQuestionsURL:     data.QCMURL(data.DefaultQCMQuestionRoutes.AddURL, qcmID),
		PreviewURL:          data.QCMURL(data.DefaultQCMRoutes.PreviewURL, qcmID),
		PreviewLandscapeURL: data.QCMURL(data.DefaultQCMRoutes.PreviewLandscapeURL, qcmID),
		PageTitle:           "Questions du QCM",
	}

	renderTableQCMQuestionPage(w, dataPage)
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
	qcmName, err := queries.GetQCMNameByID(r.Context(), db.GetQCMNameByIDParams{ID: qcmID, UserID: userID})
	if err != nil {
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

	families, err := loadQCMQuestionFamilies(r.Context(), queries, qcmID, db.GetFilteredQuestionsParams{
		UserID:       userID,
		SubjectID:    subjectID,
		ThemeID:      themeID,
		YearLevelID:  yearLevelID,
		SkillID:      skillID,
		DifficultyID: difficultyID,
		PointID:      pointID,
	})
	if err != nil {
		log.Printf("From AddFormQCMQuestionHandler -> loadQCMQuestionFamilies DB error: %v", err)
		http.Error(w, "DB error", http.StatusInternalServerError)
		return
	}

	addURL := data.QCMURL(data.DefaultQCMQuestionRoutes.AddURL, qcmID)
	tableURL := data.QCMURL(data.DefaultQCMRoutes.AddQuestionURL, qcmID)
	dataPage := data.QCMQuestionPageData{
		Routes:            data.DefaultDashboardRoutes,
		QCMQuestionRoutes: data.DefaultQCMQuestionRoutes,
		QCMContext:        data.QCMContext{ID: qcmID, Name: qcmName},
		PageTitle:         "add qcm question",
		ExtraData: map[string]any{
			"AddURL":               addURL,
			"TableURL":             tableURL,
			"Subjects":             subjects,
			"Themes":               themes,
			"YearLevels":           yearLevels,
			"Skills":               skills,
			"Difficulties":         difficulties,
			"Points":               points,
			"QuestionFamilies":     families,
			"SelectedSubjectID":    subjectIDSelected,
			"SelectedThemeID":      themeIDSelected,
			"SelectedYearLevelID":  yearLevelIDSelected,
			"SelectedSkillID":      skillIDSelected,
			"SelectedDifficultyID": difficultyIDSelected,
			"SelectedPointID":      pointIDSelected,
		},
	}

	renderAddFormQCMQuestionPage(w, dataPage)
}

func loadQCMQuestionFamilies(ctx context.Context, queries *db.Queries, qcmID int64, filters db.GetFilteredQuestionsParams) ([]questionfamilies.QuestionFamily, error) {
	rows, err := queries.GetFilteredQuestions(ctx, filters)
	if err != nil {
		return nil, err
	}
	questionsIDsInQCM, err := queries.GetQCMQuestionsIDs(ctx, db.GetQCMQuestionsIDsParams{UserID: filters.UserID, QcmID: qcmID})
	if err != nil {
		return nil, err
	}
	candidates := make([]db.GetFilteredQuestionsRow, 0, len(rows))
	for _, row := range rows {
		if !slices.Contains(questionsIDsInQCM, row.ID) {
			candidates = append(candidates, row)
		}
	}
	return buildQCMQuestionFamilies(ctx, queries, filters.UserID, candidates)
}

func buildQCMQuestionFamilies(ctx context.Context, queries *db.Queries, userID int64, rows []db.GetFilteredQuestionsRow) ([]questionfamilies.QuestionFamily, error) {
	alternativesDB, err := queries.GetAllOwnedAltQuestions(ctx, userID)
	if err != nil {
		return nil, err
	}
	questions := make([]questionfamilies.Question, 0, len(rows))
	for _, row := range rows {
		questions = append(questions, questionfamilies.Question{
			ID: row.ID, Content: row.Content, SubjectName: row.SubjectName,
			ThemeName: row.ThemeName, YearLevelName: row.YearLevelName,
			SkillName: row.SkillName, DifficultyName: row.DifficultyName,
			PointValue: row.PointValue, Selectable: true,
		})
	}
	variants := make([]questionfamilies.Variant, 0, len(alternativesDB))
	for _, alternative := range alternativesDB {
		variants = append(variants, questionfamilies.Variant{
			ID: alternative.ID, QuestionID: alternative.QuestionID, Content: alternative.Content,
		})
	}
	return questionfamilies.Build(questions, variants), nil
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
	slices.Sort(questionsIDs)

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
	tableURL := data.QCMURL(data.DefaultQCMRoutes.AddQuestionURL, qcmID)
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
	qcmName, err := queries.GetQCMNameByID(r.Context(), db.GetQCMNameByIDParams{ID: qcmID, UserID: userID})
	if err != nil {
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
		QCMContext:        data.QCMContext{ID: qcmID, Name: qcmName},
		PageTitle:         "delete qcm question",
		ExtraData: map[string]any{
			"QCMQuestionID":   qcmQuestionIDstr,
			"QuestionContent": questionContent,
		},
	}

	renderDeleteFormQCMQuestionPage(w, dataPage)
}

func DeleteQCMQuestionHandler(w http.ResponseWriter, r *http.Request, queries *db.Queries, conn *sql.DB) {
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

	tx, err := conn.BeginTx(r.Context(), nil)
	if err != nil {
		log.Printf("From DeleteQCMQuestionHandler -> conn.BeginTx: %v", err)
		http.Error(w, "Something went wrong !", http.StatusInternalServerError)
		return
	}
	defer tx.Rollback()
	qtx := queries.WithTx(tx)

	position, err := qtx.GetQCMQuestionPosition(r.Context(), db.GetQCMQuestionPositionParams{
		ID:     qcmQuestionID,
		QcmID:  qcmID,
		UserID: userID,
	})
	if err != nil {
		tools.HandleOwnedLookupError(w, err, "DeleteQCMQuestionHandler GetQCMQuestionPosition")
		return
	}

	rows, err := qtx.DeleteQCMQuestion(r.Context(), db.DeleteQCMQuestionParams{
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

	shiftedCount := position.MaxPosition - position.Position
	if shiftedCount > 0 {
		moved, err := qtx.MoveQCMQuestionPositionsToTemporaryRange(r.Context(), db.MoveQCMQuestionPositionsToTemporaryRangeParams{
			MaxPosition:     position.MaxPosition,
			QcmID:           qcmID,
			UserID:          userID,
			DeletedPosition: position.Position,
		})
		if err != nil {
			log.Printf("From DeleteQCMQuestionHandler -> move positions to temporary range: %v", err)
			http.Error(w, "Something went wrong !", http.StatusInternalServerError)
			return
		}
		if moved != shiftedCount {
			log.Printf("From DeleteQCMQuestionHandler -> moved %d positions, want %d", moved, shiftedCount)
			http.Error(w, "Something went wrong !", http.StatusInternalServerError)
			return
		}

		compacted, err := qtx.CompactQCMQuestionPositions(r.Context(), db.CompactQCMQuestionPositionsParams{
			MaxPosition: position.MaxPosition,
			QcmID:       qcmID,
			UserID:      userID,
		})
		if err != nil {
			log.Printf("From DeleteQCMQuestionHandler -> compact positions: %v", err)
			http.Error(w, "Something went wrong !", http.StatusInternalServerError)
			return
		}
		if compacted != shiftedCount {
			log.Printf("From DeleteQCMQuestionHandler -> compacted %d positions, want %d", compacted, shiftedCount)
			http.Error(w, "Something went wrong !", http.StatusInternalServerError)
			return
		}
	}

	if err := tx.Commit(); err != nil {
		log.Printf("From DeleteQCMQuestionHandler -> transaction commit: %v", err)
		http.Error(w, "Something went wrong !", http.StatusInternalServerError)
		return
	}

	tableURL := data.QCMURL(data.DefaultQCMRoutes.AddQuestionURL, qcmID)
	http.Redirect(w, r, tableURL, http.StatusSeeOther)
}

type qcmQuestionMoveDirection int64

const (
	moveQCMQuestionUp   qcmQuestionMoveDirection = -1
	moveQCMQuestionDown qcmQuestionMoveDirection = 1
)

func MoveQCMQuestionUpHandler(w http.ResponseWriter, r *http.Request, queries *db.Queries, conn *sql.DB) {
	moveQCMQuestionHandler(w, r, queries, conn, moveQCMQuestionUp)
}

func MoveQCMQuestionDownHandler(w http.ResponseWriter, r *http.Request, queries *db.Queries, conn *sql.DB) {
	moveQCMQuestionHandler(w, r, queries, conn, moveQCMQuestionDown)
}

func moveQCMQuestionHandler(w http.ResponseWriter, r *http.Request, queries *db.Queries, conn *sql.DB, direction qcmQuestionMoveDirection) {
	userID, _, ok := tools.CheckRequest(w, r, http.MethodPost)
	if !ok {
		log.Println("From moveQCMQuestionHandler -> tools.CheckRequest return not ok")
		return
	}

	qcmID, err := parseQCMQuestionMoveID(r.FormValue("qcm_id"))
	if err != nil {
		log.Printf("From moveQCMQuestionHandler -> invalid qcm ID: %v", err)
		http.Error(w, "Something went wrong !", http.StatusBadRequest)
		return
	}
	qcmQuestionID, err := parseQCMQuestionMoveID(r.FormValue("qcm_question_id"))
	if err != nil {
		log.Printf("From moveQCMQuestionHandler -> invalid qcm question ID: %v", err)
		http.Error(w, "Something went wrong !", http.StatusBadRequest)
		return
	}

	_, err = moveQCMQuestion(r.Context(), queries, conn, userID, qcmID, qcmQuestionID, direction)
	if err != nil {
		tools.HandleOwnedLookupError(w, err, "moveQCMQuestionHandler")
		return
	}
	http.Redirect(w, r, data.QCMURL(data.DefaultQCMRoutes.AddQuestionURL, qcmID), http.StatusSeeOther)
}

func parseQCMQuestionMoveID(value string) (int64, error) {
	if value == "" {
		return 0, strconv.ErrSyntax
	}
	return strconv.ParseInt(value, 10, 64)
}

func moveQCMQuestion(ctx context.Context, queries *db.Queries, conn *sql.DB, userID, qcmID, qcmQuestionID int64, direction qcmQuestionMoveDirection) (bool, error) {
	tx, err := conn.BeginTx(ctx, nil)
	if err != nil {
		return false, err
	}
	defer tx.Rollback()
	qtx := queries.WithTx(tx)

	current, err := qtx.GetQCMQuestionPosition(ctx, db.GetQCMQuestionPositionParams{
		ID: qcmQuestionID, QcmID: qcmID, UserID: userID,
	})
	if err != nil {
		return false, err
	}
	targetPosition := current.Position + int64(direction)
	if targetPosition < 1 || targetPosition > current.MaxPosition {
		return false, nil
	}

	adjacent, err := qtx.GetQCMQuestionByPosition(ctx, db.GetQCMQuestionByPositionParams{
		QcmID: qcmID, UserID: userID, Position: targetPosition,
	})
	if err != nil {
		return false, err
	}
	temporaryPosition := current.MaxPosition + 1
	steps := []db.MoveQCMQuestionToPositionParams{
		{Position: temporaryPosition, ID: qcmQuestionID, QcmID: qcmID, UserID: userID},
		{Position: current.Position, ID: adjacent.ID, QcmID: qcmID, UserID: userID},
		{Position: targetPosition, ID: qcmQuestionID, QcmID: qcmID, UserID: userID},
	}
	for _, step := range steps {
		rows, err := qtx.MoveQCMQuestionToPosition(ctx, step)
		if err != nil {
			return false, err
		}
		if rows != 1 {
			return false, fmt.Errorf("move QCM question step affected %d rows", rows)
		}
	}
	if err := tx.Commit(); err != nil {
		return false, err
	}
	return true, nil
}
