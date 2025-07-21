package questions

import (
	"log"
	"net/http"
	"net/url"
	"strconv"

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

	subjects, err := queries.GetAllSubjects(r.Context(), userID)
	if err != nil {
		log.Printf("From AddFormQuestionsHandler : GetAllSubject DB error: %v", err)
		http.Error(w, "Database error", http.StatusInternalServerError)
	}

	themes, err := queries.GetAllThemes(r.Context(), userID)
	if err != nil {
		log.Printf("From AddFormQuestionsHandler : GetAllTheme DB error: %v", err)
		http.Error(w, "Database error", http.StatusInternalServerError)
	}

	yearLevels, err := queries.GetAllYearLevels(r.Context(), userID)
	if err != nil {
		log.Printf("From AddFormQuestionsHandler : GetAllYearLevels DB error: %v", err)
		http.Error(w, "Database error", http.StatusInternalServerError)
	}

	skills, err := queries.GetAllSkills(r.Context(), userID)
	if err != nil {
		log.Printf("From AddFormQuestionsHandler : GetAllSkills DB error: %v", err)
		http.Error(w, "Database error", http.StatusInternalServerError)
	}
	difficulties, err := queries.GetAllDifficulties(r.Context(), userID)
	if err != nil {
		log.Printf("From AddFormQuestionsHandler : GetAllDifficulties DB error: %v", err)
		http.Error(w, "Database error", http.StatusInternalServerError)
	}
	points, err := queries.GetAllPoints(r.Context(), userID)
	if err != nil {
		log.Printf("From AddFormQuestionsHandler : GetAllPoints DB error: %v", err)
		http.Error(w, "Database error", http.StatusInternalServerError)
	}

	if len(subjects) == 0 || len(themes) == 0 || len(yearLevels) == 0 || len(skills) == 0 || len(difficulties) == 0 || len(points) == 0 {
		errorMessage := url.QueryEscape("Les champs des caractéristiques d'une question ne peuvent pas être vide.")
		http.Redirect(w, r, data.ErrorMessageURL+"?errormessage="+errorMessage, http.StatusSeeOther)
	}

	dataPage := data.QuestionPageData{
		Routes:         data.DefaultDashboardRoutes,
		QuestionRoutes: data.DefaultQuestionRoutes,
		PageTitle:      "add question",
		ExtraData: map[string]any{
			"Subjects":     subjects,
			"Themes":       themes,
			"YearLevels":   yearLevels,
			"Skills":       skills,
			"Difficulties": difficulties,
			"Points":       points,
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
