package questions

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

func TableQuestionsHandler(w http.ResponseWriter, r *http.Request, queries *db.Queries) {
	userID, username, ok := tools.CheckRequest(w, r, http.MethodGet)
	if !ok {
		return
	}

	questionsDB, err := queries.GetAllQuestions(r.Context(), userID)
	if err != nil {
		log.Printf("From TableQuestionsHandler : GetAllQuestions DB error: %v", err)
		http.Error(w, "DB Error", http.StatusInternalServerError)
	}

	noQuestion := true
	if len(questionsDB) > 0 {
		noQuestion = false
	}

	var actionsURLParameters []data.QuestionActionURLs
	if !noQuestion {
		for _, question := range questionsDB {
			editURL := data.DefaultQuestionRoutes.EditURL + "?question_id=" + url.QueryEscape(strconv.FormatInt(question.ID, 10))
			deleteURL := data.DefaultQuestionRoutes.DeleteURL + "?question_id=" + url.QueryEscape(strconv.FormatInt(question.ID, 10))

			actionsURLParameters = append(actionsURLParameters, data.QuestionActionURLs{
				EditURL:   editURL,
				DeleteURL: deleteURL,
			})
		}
	}

	dataPage := data.QuestionPageData{
		Routes:         data.DefaultDashboardRoutes,
		QuestionRoutes: data.DefaultQuestionRoutes,
		PageTitle:      "questions",
		ExtraData: map[string]any{
			"UserID":     userID,
			"Username":   username,
			"NoQuestion": noQuestion,
			"Questions":  questionsDB,
			"Action":     actionsURLParameters,
		},
	}

	RenderTableQuestionPage(w, dataPage)
}

func AddFormQuestionsHandler(w http.ResponseWriter, r *http.Request, queries *db.Queries) {
	userID, _, ok := tools.CheckRequest(w, r, http.MethodGet)
	if !ok {
		return
	}

	features, ok := tools.GetAllFeaturesQuestion(r, userID, queries)
	if !ok {
		http.Error(w, "Database error", http.StatusInternalServerError)
	}

	for _, feature := range features {
		featurelen, ok := tools.GetSliceLen(feature)
		if !ok {
			log.Printf("From AddFormQuestionsHandler : []db.type error")
		}
		if featurelen == 0 {
			errorMessage := url.QueryEscape("Les champs des caractéristiques d'une question ne peuvent pas être vide.")
			http.Redirect(w, r, data.ErrorMessageURL+"?errormessage="+errorMessage, http.StatusSeeOther)
		}
	}

	dataPage := data.QuestionPageData{
		Routes:         data.DefaultDashboardRoutes,
		QuestionRoutes: data.DefaultQuestionRoutes,
		PageTitle:      "add question",
		ExtraData: map[string]any{
			"Subjects":     features["subjects"],
			"Themes":       features["themes"],
			"YearLevels":   features["yearLevels"],
			"Skills":       features["skills"],
			"Difficulties": features["difficulties"],
			"Points":       features["points"],
		},
	}

	RenderAddFormQuestion(w, dataPage)
}

func AddQuestionsHandler(w http.ResponseWriter, r *http.Request, queries *db.Queries) {
	userID, _, ok := tools.CheckRequest(w, r, http.MethodPost)
	if !ok {
		return
	}

	r.ParseForm()
	features := r.Form // type: map[string][]string

	content := features["content"][0]
	if content == "" {
		errorMessage := url.QueryEscape("Les champs des caractéristiques d'une question ne peuvent pas être vide.")
		http.Redirect(w, r, data.ErrorMessageURL+"?errormessage="+errorMessage, http.StatusSeeOther)
		return
	}

	delete(features, "content")
	intIDs := make(map[string]int64, 6)

	for feature, value := range features {
		intID, ok := tools.StrToInt(value[0])
		if !ok {
			http.Error(w, "From AddQuestionsHandler : no feature id parameter", http.StatusBadRequest)
			return
		}
		intIDs[feature] = intID
	}

	err := queries.CreateQuestion(r.Context(), db.CreateQuestionParams{
		SubjectID:    intIDs["subjectID"],
		ThemeID:      intIDs["themeID"],
		YearLevelID:  intIDs["yearLevelID"],
		SkillID:      intIDs["skillID"],
		DifficultyID: intIDs["difficultyID"],
		PointID:      intIDs["pointID"],
		Content:      content,
		UserID:       userID,
	})
	if err != nil {
		log.Printf("From AddQuestionsHandler : DB CreateQuestion error : %v", err)
		http.Error(w, "Database Error", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, data.DefaultDashboardRoutes.QuestionsURL, http.StatusSeeOther)
}

func EditFormQuestionHandler(w http.ResponseWriter, r *http.Request, queries *db.Queries) {
	userID, _, ok := tools.CheckRequest(w, r, http.MethodGet)
	if !ok {
		return
	}

	questionIDStr := r.FormValue("question_id")
	if questionIDStr == "" {
		http.Error(w, "From EditFormQuestionHandler : no question id parameter", http.StatusBadRequest)
		return
	}

	questionID, err := strconv.ParseInt(questionIDStr, 10, 64)
	if err != nil {
		http.Error(w, "From EditFormQuestionHandler : invalid question ID", http.StatusBadRequest)
		return
	}

	question, err := queries.GetQuestionByID(r.Context(), db.GetQuestionByIDParams{
		ID:     questionID,
		UserID: userID,
	})
	if err != nil {
		log.Printf("From EditFormQuestionHandler : GetQuestionByID DB error: %v", err)
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	features, ok := tools.GetAllFeaturesQuestion(r, userID, queries)
	if !ok {
		http.Error(w, "Database error", http.StatusInternalServerError)
	}

	for _, feature := range features {
		featurelen, ok := tools.GetSliceLen(feature)
		if !ok {
			log.Printf("From AddFormQuestionsHandler : []db.type error")
		}
		if featurelen == 0 {
			errorMessage := url.QueryEscape("Les champs des caractéristiques d'une question ne peuvent pas être vide.")
			http.Redirect(w, r, data.ErrorMessageURL+"?errormessage="+errorMessage, http.StatusSeeOther)
		}
	}

	dataPage := data.QuestionPageData{
		Routes:         data.DefaultDashboardRoutes,
		QuestionRoutes: data.DefaultQuestionRoutes,
		PageTitle:      "edit question",
		ExtraData: map[string]any{
			"Question":     question,
			"QuestionID":   questionIDStr,
			"Subjects":     features["subjects"],
			"Themes":       features["themes"],
			"YearLevels":   features["yearLevels"],
			"Skills":       features["skills"],
			"Difficulties": features["difficulties"],
			"Points":       features["points"],
		},
	}
	RenderEditFormQuestion(w, dataPage)
}

func EditQuestionHandler(w http.ResponseWriter, r *http.Request, queries *db.Queries) {
	userID, _, ok := tools.CheckRequest(w, r, http.MethodPost)
	if !ok {
		return
	}

	r.ParseForm()
	features := r.Form // type: map[string][]string

	content := features["content"][0]
	if content == "" {
		errorMessage := url.QueryEscape("Les champs des caractéristiques d'une question ne peuvent pas être vide.")
		http.Redirect(w, r, data.ErrorMessageURL+"?errormessage="+errorMessage, http.StatusSeeOther)
		return
	}

	questionIDStr := strings.TrimSpace(r.FormValue("question_id"))
	if questionIDStr == "" {
		http.Error(w, "From EditQuestionHandler : questionID missing", http.StatusInternalServerError)
		return
	}
	questionID, err := strconv.ParseInt(questionIDStr, 10, 64)
	if err != nil {
		http.Error(w, "From EditQuestionHandler : invalid question ID", http.StatusBadRequest)
		return
	}

	delete(features, "content")

	intIDs := make(map[string]int64, 6)

	for feature, value := range features {
		intID, ok := tools.StrToInt(value[0])
		if !ok {
			http.Error(w, "From EditQuestionsHandler : no feature id parameter", http.StatusBadRequest)
			return
		}
		intIDs[feature] = intID
	}
	log.Println(intIDs)

	if err := queries.UpdateQuestion(r.Context(), db.UpdateQuestionParams{
		SubjectID:    intIDs["subjectID"],
		ThemeID:      intIDs["themeID"],
		YearLevelID:  intIDs["yearLevelID"],
		SkillID:      intIDs["skillID"],
		DifficultyID: intIDs["difficultyID"],
		PointID:      intIDs["pointID"],
		Content:      content,
		ID:           questionID,
		UserID:       userID,
	}); err != nil {
		log.Printf("From EditSkillHandler : UpdateQuestion DB error: %v", err)
		errorMessage := url.QueryEscape("Il ne peut pas exister deux fois la même question.")
		http.Redirect(w, r, data.ErrorMessageURL+"?errormessage="+errorMessage, http.StatusSeeOther)
		return
	}

	http.Redirect(w, r, data.DefaultDashboardRoutes.QuestionsURL, http.StatusSeeOther)
}

func DeleteFormQuestionHandler(w http.ResponseWriter, r *http.Request, queries *db.Queries) {
	userID, _, ok := tools.CheckRequest(w, r, http.MethodGet)
	if !ok {
		return
	}

	questionIDStr := r.FormValue("question_id")
	if questionIDStr == "" {
		http.Error(w, "From DeleteFormQuestionHandler : no question id parameter", http.StatusBadRequest)
		return
	}

	questionID, err := strconv.ParseInt(questionIDStr, 10, 64)
	if err != nil {
		http.Error(w, "From DeleteFormQuestionHandler : invalid question ID", http.StatusBadRequest)
		return
	}

	question, err := queries.GetQuestionByID(r.Context(), db.GetQuestionByIDParams{
		ID:     questionID,
		UserID: userID,
	})
	if err != nil {
		log.Printf("From DeleteFormQuestionHandler : GetQuestionByID DB error: %v", err)
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	dataPage := data.QuestionPageData{
		Routes:         data.DefaultDashboardRoutes,
		QuestionRoutes: data.DefaultQuestionRoutes,
		PageTitle:      "delete question",
		ExtraData: map[string]any{
			"Question":   question,
			"QuestionID": questionIDStr,
		},
	}

	RenderDeleteFormQuestion(w, dataPage)
}

func DeleteQuestionHandler(w http.ResponseWriter, r *http.Request, queries *db.Queries) {
	userID, _, ok := tools.CheckRequest(w, r, http.MethodPost)
	if !ok {
		return
	}

	questionIDStr := r.FormValue("question_id")
	if questionIDStr == "" {
		http.Error(w, "From DeletequestionHandler : no skill id parameter", http.StatusBadRequest)
		return
	}

	questionID, err := strconv.ParseInt(questionIDStr, 10, 64)
	if err != nil {
		http.Error(w, "From DeleteQuestionHandler : invalid skill ID", http.StatusBadRequest)
		return
	}

	if err := queries.DeleteQuestion(r.Context(), db.DeleteQuestionParams{
		ID:     questionID,
		UserID: userID,
	}); err != nil {
		log.Printf("From DeleteQuestionHandler : DeleteQuestion DB error: %v", err)
		http.Error(w, "From DeleteQuestionHandler : Database error", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, data.DefaultDashboardRoutes.QuestionsURL, http.StatusSeeOther)
}
