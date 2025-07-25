package altquestions

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

func TableAltQuestionsHandler(w http.ResponseWriter, r *http.Request, queries *db.Queries) {
	userID, username, ok := tools.CheckRequest(w, r, http.MethodGet)
	if !ok {
		return
	}

	questionIDStr := r.FormValue("question_id")

	if questionIDStr == "" {
		http.Error(w, "From TableAnswersHandler : no question id parameter", http.StatusBadRequest)
		return
	}

	questionID, err := strconv.ParseInt(questionIDStr, 10, 64)
	if err != nil {
		http.Error(w, "From TableAnswersHandler : invalid question ID", http.StatusBadRequest)
		return
	}

	altQuestionsDB, err := queries.GetAllAltQuestions(r.Context(), db.GetAllAltQuestionsParams{
		QuestionID: questionID,
		UserID:     userID,
	})
	if err != nil {
		log.Printf("From TableAltQuestionsHandler : GetAllAltQuestions DB error: %v", err)
		http.Error(w, "DB Error", http.StatusInternalServerError)
		return
	}

	noAltQuestion := true
	if len(altQuestionsDB) > 0 {
		noAltQuestion = false
	}

	var actionsURLParameters []data.AltQuestionActionURLs
	if !noAltQuestion {
		for _, altQuestion := range altQuestionsDB {
			editURL := data.DefaultAltQuestionRoutes.EditURL + "?question_id=" + url.QueryEscape(strconv.FormatInt(questionID, 10)) + "&alt_question_id=" + url.QueryEscape(strconv.FormatInt(altQuestion.ID, 10))
			deleteURL := data.DefaultAltQuestionRoutes.DeleteURL + "?question_id=" + url.QueryEscape(strconv.FormatInt(questionID, 10)) + "&alt_question_id=" + url.QueryEscape(strconv.FormatInt(altQuestion.ID, 10))
			altAnswersURL := data.DefaultAltQuestionRoutes.AltAnswersURL + "?question_id=" + url.QueryEscape(strconv.FormatInt(questionID, 10)) + "&alt_question_id=" + url.QueryEscape(strconv.FormatInt(altQuestion.ID, 10))

			actionsURLParameters = append(actionsURLParameters, data.AltQuestionActionURLs{
				EditURL:       editURL,
				DeleteURL:     deleteURL,
				AltAnswersURL: altAnswersURL,
			})
		}
	}

	addURL := data.DefaultAltQuestionRoutes.AddURL + "?question_id=" + url.QueryEscape(questionIDStr)
	dataPage := data.AltQuestionPageData{
		Routes:            data.DefaultDashboardRoutes,
		AltQuestionRoutes: data.DefaultAltQuestionRoutes,
		PageTitle:         "alt questions",
		ExtraData: map[string]any{
			"UserID":        userID,
			"Username":      username,
			"NoAltQuestion": noAltQuestion,
			"AltQuestions":  altQuestionsDB,
			"Action":        actionsURLParameters,
			"AddURL":        addURL,
		},
	}

	RenderTableAltQuestionPage(w, dataPage)
}

func AddFormAltQuestionHandler(w http.ResponseWriter, r *http.Request, queries *db.Queries) {
	_, _, ok := tools.CheckRequest(w, r, http.MethodGet)
	if !ok {
		return
	}

	questionIDStr := r.FormValue("question_id")
	if questionIDStr == "" {
		http.Error(w, "From AddFormAnswerHandler : no question id parameter", http.StatusBadRequest)
		return
	}

	addURL := data.DefaultAltQuestionRoutes.AddURL + "?question_id=" + url.QueryEscape(questionIDStr)

	dataPage := data.AltQuestionPageData{
		Routes:    data.DefaultDashboardRoutes,
		PageTitle: "add alt question",
		ExtraData: map[string]any{
			"AddURL": addURL,
		},
	}
	RenderAddFormAltQuestionPage(w, dataPage)
}

func AddAltQuestionHandler(w http.ResponseWriter, r *http.Request, queries *db.Queries) {
	userID, _, ok := tools.CheckRequest(w, r, http.MethodPost)
	if !ok {
		return
	}

	questionIDStr := r.FormValue("question_id")
	if questionIDStr == "" {
		http.Error(w, "From AddAltQuestionHandler : no question id parameter", http.StatusBadRequest)
		return
	}
	questionID, err := strconv.ParseInt(questionIDStr, 10, 64)
	if err != nil {
		http.Error(w, "From AddAltQuestionHandler : invalid question ID", http.StatusBadRequest)
		return
	}

	content := strings.TrimSpace(r.FormValue("content"))

	err = queries.CreateAltQuestion(r.Context(), db.CreateAltQuestionParams{
		QuestionID: questionID,
		Content:    content,
		UserID:     userID,
	})
	if err != nil {
		log.Printf("From AddAltQuestionHandler, CreateAltQuestion : DB error: %v", err)
		errorMessage := url.QueryEscape("Il ne peut pas exister deux fois la même réponse ou la question ne peut être vide.")
		http.Redirect(w, r, data.ErrorMessageURL+"?errormessage="+errorMessage, http.StatusSeeOther)
		return
	}

	altQuestionURL := data.DefaultQuestionRoutes.AltQuestionsURL + "?question_id=" + url.QueryEscape(questionIDStr)
	http.Redirect(w, r, altQuestionURL, http.StatusSeeOther)
}

func EditFormAltQuestionHandler(w http.ResponseWriter, r *http.Request, queries *db.Queries) {
	userID, _, ok := tools.CheckRequest(w, r, http.MethodGet)
	if !ok {
		return
	}

	questionIDStr := r.URL.Query().Get("question_id")
	if questionIDStr == "" {
		http.Error(w, "From EditFormAltQuestionHandler : no alt question id parameter", http.StatusBadRequest)
		return
	}

	altQuestionIDStr := r.URL.Query().Get("alt_question_id")
	if altQuestionIDStr == "" {
		http.Error(w, "From EditFormAltQuestionHandler : no alt question id parameter", http.StatusBadRequest)
		return
	}

	altQuestionID, err := strconv.ParseInt(altQuestionIDStr, 10, 64)
	if err != nil {
		http.Error(w, "From EditFormAltQuestionHandler : invalid alt question ID", http.StatusBadRequest)
		return
	}

	altQuestion, err := queries.GetAltQuestionByID(r.Context(), db.GetAltQuestionByIDParams{
		ID:     altQuestionID,
		UserID: userID,
	})
	if err != nil {
		log.Printf("From EditFormAltQuestionHandler : GetAltQuestionByID DB error: %v", err)
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	// editURL := data.DefaultAnswerRoutes.EditURL + "?question_id=" + url.QueryEscape(questionIDStr)
	dataPage := data.AltQuestionPageData{
		Routes:            data.DefaultDashboardRoutes,
		AltQuestionRoutes: data.DefaultAltQuestionRoutes,
		PageTitle:         "edit alt question",
		ExtraData: map[string]any{
			"AltQuestion":   altQuestion,
			"AltQuestionID": altQuestionIDStr,
			"QuestionID":    questionIDStr,
		},
	}
	RenderEditFormAltQuestionPage(w, dataPage)
}

func EditAltQuestionHandler(w http.ResponseWriter, r *http.Request, queries *db.Queries) {
	userID, _, ok := tools.CheckRequest(w, r, http.MethodPost)
	if !ok {
		return
	}

	questionIDStr := r.FormValue("question_id")
	if questionIDStr == "" {
		http.Error(w, "From EditAltQuestionHandler : no question id parameter", http.StatusBadRequest)
		return
	}

	newContent := r.FormValue("new_content")

	altQuestionIDStr := r.FormValue("alt_question_id")
	if altQuestionIDStr == "" {
		http.Error(w, "From EditAltQuestionHandler : altQuestionID missing", http.StatusInternalServerError)
		return
	}
	altQuestionID, err := strconv.ParseInt(altQuestionIDStr, 10, 64)
	if err != nil {
		http.Error(w, "From  EditAltQuestionHandler : invalid altQuestion ID", http.StatusBadRequest)
		return
	}

	if err := queries.UpdateAltQuestion(r.Context(), db.UpdateAltQuestionParams{
		Content: newContent,
		ID:      altQuestionID,
		UserID:  userID,
	}); err != nil {
		log.Printf("From  EditAltQuestionHandler : UpdateAnswer DB error: %v", err)
		errorMessage := url.QueryEscape("Il ne peut pas exister deux fois la même réponse ou la réponse ne peut être vide")
		http.Redirect(w, r, data.ErrorMessageURL+"?errormessage="+errorMessage, http.StatusSeeOther)
		return
	}

	altQuestionURL := data.DefaultQuestionRoutes.AltQuestionsURL + "?question_id=" + url.QueryEscape(questionIDStr)
	http.Redirect(w, r, altQuestionURL, http.StatusSeeOther)
}

func DeleteFormAltQuestionHandler(w http.ResponseWriter, r *http.Request, queries *db.Queries) {
	userID, _, ok := tools.CheckRequest(w, r, http.MethodGet)
	if !ok {
		return
	}

	questionIDStr := r.URL.Query().Get("question_id")
	if questionIDStr == "" {
		http.Error(w, "From  DeleteFormAltQuestionHandler : no question id parameter", http.StatusBadRequest)
		return
	}

	altQuestionIDStr := r.URL.Query().Get("alt_question_id")
	if altQuestionIDStr == "" {
		http.Error(w, "From DeleteFormAltQuestionHandler : no alt question id parameter", http.StatusBadRequest)
		return
	}

	altQuestionID, err := strconv.ParseInt(altQuestionIDStr, 10, 64)
	if err != nil {
		http.Error(w, "From DeleteFormAnswerHandler : invalid answer ID", http.StatusBadRequest)
		return
	}

	altQuestion, err := queries.GetAltQuestionByID(r.Context(), db.GetAltQuestionByIDParams{
		ID:     altQuestionID,
		UserID: userID,
	})
	if err != nil {
		log.Printf("From DeleteFormAnswerHandler GetAltQuestionByID : DB error: %v", err)
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	dataPage := data.AltQuestionPageData{
		Routes:            data.DefaultDashboardRoutes,
		AltQuestionRoutes: data.DefaultAltQuestionRoutes,
		PageTitle:         "delete alt question",
		ExtraData: map[string]any{
			"AltQuestion":   altQuestion,
			"AltQuestionID": altQuestionIDStr,
			"QuestionID":    questionIDStr,
		},
	}

	RenderDeleteFormAltQuestionPage(w, dataPage)
}

func DeleteAltQuestionHandler(w http.ResponseWriter, r *http.Request, queries *db.Queries) {
	userID, _, ok := tools.CheckRequest(w, r, http.MethodPost)
	if !ok {
		return
	}

	questionIDStr := r.FormValue("question_id")
	if questionIDStr == "" {
		http.Error(w, "From DeleteAltQUestionHandler : no question id parameter", http.StatusBadRequest)
		return
	}

	altQuestionIDStr := r.FormValue("alt_question_id")
	if altQuestionIDStr == "" {
		http.Error(w, "From DeleteAltQUestionHandler : no alt question id parameter", http.StatusBadRequest)
		return
	}

	altQuestionID, err := strconv.ParseInt(altQuestionIDStr, 10, 64)
	if err != nil {
		http.Error(w, "From DeleteAltQUestionHandler : invalid alt question ID", http.StatusBadRequest)
		return
	}

	if err := queries.DeleteAltQuestion(r.Context(), db.DeleteAltQuestionParams{
		ID:     altQuestionID,
		UserID: userID,
	}); err != nil {
		log.Printf("From DeleteAltQUestionHandler : DeleteAltQuestion DB error: %v", err)
		http.Error(w, "Error", http.StatusInternalServerError)
		return
	}

	altQuestionURL := data.DefaultQuestionRoutes.AltQuestionsURL + "?question_id=" + url.QueryEscape(questionIDStr)
	http.Redirect(w, r, altQuestionURL, http.StatusSeeOther)
}
