package altanswers

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

func TableAltAnswersHandler(w http.ResponseWriter, r *http.Request, queries *db.Queries) {
	userID, _, ok := tools.CheckRequest(w, r, http.MethodGet)
	if !ok {
		return
	}

	questionIDStr := r.URL.Query().Get("question_id")

	if questionIDStr == "" {
		http.Error(w, "From TableAltAnswersHandler : no question id parameter", http.StatusBadRequest)
		return
	}

	altQuestionIDStr := r.URL.Query().Get("alt_question_id")

	if altQuestionIDStr == "" {
		http.Error(w, "From TableAltAnswersHandler : no alt question id parameter", http.StatusBadRequest)
		return
	}

	altQuestionID, err := strconv.ParseInt(altQuestionIDStr, 10, 64)
	if err != nil {
		http.Error(w, "From  TableAltAnswersHandler : invalid alt question ID", http.StatusBadRequest)
		return
	}

	altQuestion, err := queries.GetAltQuestionByID(r.Context(), db.GetAltQuestionByIDParams{
		ID:     altQuestionID,
		UserID: userID,
	})
	if err != nil {
		log.Printf("From TableAltAnswersHandler, GetAltQuestionByID DB error: %v", err)
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	altAnswersDB, err := queries.GetAllAltAnswersByAltQuestionID(r.Context(), db.GetAllAltAnswersByAltQuestionIDParams{
		AltQuestionID: altQuestionID,
		UserID:        userID,
	})
	if err != nil {
		log.Printf("From TableAltAnswersHandler, GetAllAltAnswersByAltQuestionID DB error: %v", err)
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	noAltAnswer := true
	if len(altAnswersDB) > 0 {
		noAltAnswer = false
	}

	var actionsURLParameters []data.AltAnswerActionURLs
	if !noAltAnswer {
		for _, altAnswer := range altAnswersDB {
			params := "?question_id=" + url.QueryEscape(questionIDStr) +
				"&alt_question_id=" + url.QueryEscape(altQuestionIDStr) +
				"&alt_answer_id=" + url.QueryEscape(strconv.FormatInt(altAnswer.ID, 10))
			editURL := data.DefaultAltAnswerRoutes.EditURL + params
			deleteURL := data.DefaultAltAnswerRoutes.DeleteURL + params

			actionsURLParameters = append(actionsURLParameters, data.AltAnswerActionURLs{
				EditURL:   editURL,
				DeleteURL: deleteURL,
			})
		}
	}

	addURL := data.DefaultAltAnswerRoutes.AddURL +
		"?question_id=" + url.QueryEscape(questionIDStr) +
		"&alt_question_id=" + url.QueryEscape(altQuestionIDStr)

	altQuestionsURL := data.DefaultQuestionRoutes.AltQuestionsURL +
		"?question_id=" + url.QueryEscape(questionIDStr)

	dataPage := data.AltAnswerPageData{
		Routes:    data.DefaultDashboardRoutes,
		PageTitle: "alt answers",
		ExtraData: map[string]any{
			"AltQuestionsURL": altQuestionsURL,
			"NoAltAnswer":     noAltAnswer,
			"Action":          actionsURLParameters,
			"AltAnswers":      altAnswersDB,
			"AltQuestion":     altQuestion.Content,
			"AddURL":          addURL,
		},
	}

	RenderTableAltAnswerPage(w, dataPage)
}

func AddFormAltAnswerHandler(w http.ResponseWriter, r *http.Request, queries *db.Queries) {
	_, _, ok := tools.CheckRequest(w, r, http.MethodGet)
	if !ok {
		return
	}

	questionIDStr := r.URL.Query().Get("question_id")
	if questionIDStr == "" {
		http.Error(w, "From AddFormAltAnswerHandler : no question id parameter", http.StatusBadRequest)
		return
	}

	altQuestionIDStr := r.URL.Query().Get("alt_question_id")
	if questionIDStr == "" {
		http.Error(w, "From AddFormAltAnswerHandler : no alt question id parameter", http.StatusBadRequest)
		return
	}

	dataPage := data.AltAnswerPageData{
		Routes:          data.DefaultDashboardRoutes,
		AltAnswerRoutes: data.DefaultAltAnswerRoutes,
		PageTitle:       "add alt answer",
		ExtraData: map[string]any{
			"QuestionID":    questionIDStr,
			"AltQuestionID": altQuestionIDStr,
		},
	}
	RenderAddFormAltAnswerPage(w, dataPage)
}

func AddAltAnswerHandler(w http.ResponseWriter, r *http.Request, queries *db.Queries) {
	userID, _, ok := tools.CheckRequest(w, r, http.MethodPost)
	if !ok {
		return
	}

	questionIDStr := r.FormValue("question_id")
	if questionIDStr == "" {
		http.Error(w, "From AddAltAnswerHandler : no question id parameter", http.StatusBadRequest)
		return
	}

	altQuestionIDStr := r.FormValue("alt_question_id")
	if questionIDStr == "" {
		http.Error(w, "From AddAltAnswerHandler : no alt question id parameter", http.StatusBadRequest)
		return
	}

	altQuestionID, err := strconv.ParseInt(altQuestionIDStr, 10, 64)
	if err != nil {
		http.Error(w, "From AddAltAnswerHandler : invalid alt question ID", http.StatusBadRequest)
		return
	}

	state := strings.TrimSpace(r.FormValue("state"))
	content := strings.TrimSpace(r.FormValue("content"))

	booleen := 0
	if state == "true" {
		booleen = 1
	}

	err = queries.CreateAltAnswer(r.Context(), db.CreateAltAnswerParams{
		AltQuestionID: altQuestionID,
		State:         int64(booleen),
		Content:       content,
		UserID:        userID,
	})
	if err != nil {
		log.Printf("From AddAltAnswerHandler, CreateAltAnswer : DB error: %v", err)
		errorMessage := url.QueryEscape("Il ne peut pas exister deux fois la même réponse ou la réponse ne peut pas être vide")
		http.Redirect(w, r, data.ErrorMessageURL+"?errormessage="+errorMessage, http.StatusSeeOther)
		return
	}

	altAnswerURL := data.DefaultAltQuestionRoutes.AltAnswersURL +
		"?question_id=" + url.QueryEscape(questionIDStr) +
		"&alt_question_id=" + url.QueryEscape(altQuestionIDStr)
	http.Redirect(w, r, altAnswerURL, http.StatusSeeOther)
}

func EditFormAltAnswerHandler(w http.ResponseWriter, r *http.Request, queries *db.Queries) {
	userID, _, ok := tools.CheckRequest(w, r, http.MethodGet)
	if !ok {
		return
	}

	questionIDStr := r.URL.Query().Get("question_id")
	if questionIDStr == "" {
		http.Error(w, "From EditFormAltAnswerHandler : no question id parameter", http.StatusBadRequest)
		return
	}

	altQuestionIDStr := r.URL.Query().Get("alt_question_id")
	if altQuestionIDStr == "" {
		http.Error(w, "From EditFormAltAnswerHandler :no question id parameter", http.StatusBadRequest)
		return
	}

	altAnswerIDStr := r.URL.Query().Get("alt_answer_id")
	if altAnswerIDStr == "" {
		http.Error(w, "From EditFormAltAnswerHandler : no answer id parameter", http.StatusBadRequest)
		return
	}

	altAnswerID, err := strconv.ParseInt(altAnswerIDStr, 10, 64)
	if err != nil {
		http.Error(w, "From EditFormAltAnswerHandler : invalid alt answer ID", http.StatusBadRequest)
		return
	}

	altAnswer, err := queries.GetAltAnswerByID(r.Context(), db.GetAltAnswerByIDParams{
		ID:     altAnswerID,
		UserID: userID,
	})
	if err != nil {
		log.Printf("From EditFormAltAnswerHandler : GetAltAnswerByID DB error: %v", err)
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	dataPage := data.AltAnswerPageData{
		Routes:          data.DefaultDashboardRoutes,
		AltAnswerRoutes: data.DefaultAltAnswerRoutes,
		PageTitle:       "edit alt answer",
		ExtraData: map[string]any{
			"QuestionID":    questionIDStr,
			"AltQuestionID": altQuestionIDStr,
			"AltAnswer":     altAnswer,
			"AltAnswerID":   altAnswerIDStr,
		},
	}
	RenderEditFormAltAnswerPage(w, dataPage)
}

func EditAltAnswerHandler(w http.ResponseWriter, r *http.Request, queries *db.Queries) {
	userID, _, ok := tools.CheckRequest(w, r, http.MethodPost)
	if !ok {
		return
	}

	questionIDStr := r.FormValue("question_id")
	if questionIDStr == "" {
		http.Error(w, "From EditAltAnswerHandler : no question id parameter", http.StatusBadRequest)
		return
	}

	altQuestionIDStr := r.FormValue("alt_question_id")
	if questionIDStr == "" {
		http.Error(w, "From EditAltAnswerHandler : no question id parameter", http.StatusBadRequest)
		return
	}

	newContent := strings.TrimSpace(r.FormValue("new_content"))

	altAnswerIDStr := r.FormValue("alt_answer_id")
	if altAnswerIDStr == "" {
		http.Error(w, "From EditAltAnswerHandler : answerID missing", http.StatusInternalServerError)
		return
	}
	altAnswerID, err := strconv.ParseInt(altAnswerIDStr, 10, 64)
	if err != nil {
		http.Error(w, "From EditAltAnswerHandler : invalid answer ID", http.StatusBadRequest)
		return
	}

	newStateStr := r.FormValue("new_state")
	if newStateStr == "" {
		http.Error(w, "From EditAltAnswerHandler : new_state missing", http.StatusInternalServerError)
		return
	}
	newState, err := strconv.ParseInt(newStateStr, 10, 64)
	if err != nil {
		http.Error(w, "From EditAltAnswerHandler : invalid new state", http.StatusBadRequest)
		return
	}

	if err := queries.UpdateAltAnswer(r.Context(), db.UpdateAltAnswerParams{
		State:   newState,
		Content: newContent,
		ID:      altAnswerID,
		UserID:  userID,
	}); err != nil {
		log.Printf("From EditAltAnswerHandler : UpdateAnswer DB error: %v", err)
		errorMessage := url.QueryEscape("Il ne peut pas exister deux fois la même réponse ou la réponse ne peut être vide")
		http.Redirect(w, r, data.ErrorMessageURL+"?errormessage="+errorMessage, http.StatusSeeOther)
		return
	}

	altAnswerURL := data.DefaultAltQuestionRoutes.AltAnswersURL +
		"?question_id=" + url.QueryEscape(questionIDStr) +
		"&alt_question_id=" + url.QueryEscape(altQuestionIDStr)
	http.Redirect(w, r, altAnswerURL, http.StatusSeeOther)
}

func DeleteFormAltAnswerHandler(w http.ResponseWriter, r *http.Request, queries *db.Queries) {
	userID, _, ok := tools.CheckRequest(w, r, http.MethodGet)
	if !ok {
		return
	}

	questionIDStr := r.URL.Query().Get("question_id")
	if questionIDStr == "" {
		http.Error(w, "From DeleteFormAltAnswerHandler : no question id parameter", http.StatusBadRequest)
		return
	}

	altQuestionIDStr := r.URL.Query().Get("alt_question_id")
	if questionIDStr == "" {
		http.Error(w, "From DeleteFormAltAnswerHandler : no alt question id parameter", http.StatusBadRequest)
		return
	}

	altAnswerIDStr := r.URL.Query().Get("alt_answer_id")
	if altAnswerIDStr == "" {
		http.Error(w, "From DeleteFormAltAnswerHandler : no alt answer id parameter", http.StatusBadRequest)
		return
	}

	altAnswerID, err := strconv.ParseInt(altAnswerIDStr, 10, 64)
	if err != nil {
		http.Error(w, "From DeleteFormAltAnswerHandler : invalid answer ID", http.StatusBadRequest)
		return
	}

	altAnswer, err := queries.GetAltAnswerByID(r.Context(), db.GetAltAnswerByIDParams{
		ID:     altAnswerID,
		UserID: userID,
	})
	if err != nil {
		log.Printf("From DeleteFormAltAnswerHandler : GetAnswerByID DB error: %v", err)
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	dataPage := data.AltAnswerPageData{
		Routes:          data.DefaultDashboardRoutes,
		AltAnswerRoutes: data.DefaultAltAnswerRoutes,
		PageTitle:       "delete alt answer",
		ExtraData: map[string]any{
			"QuestionID":    questionIDStr,
			"AltQuestionID": altQuestionIDStr,
			"AltAnswer":     altAnswer,
			"AltAnswerID":   altAnswerIDStr,
		},
	}

	RenderDeleteFormAltAnswerPage(w, dataPage)
}

func DeleteAltAnswerHandler(w http.ResponseWriter, r *http.Request, queries *db.Queries) {
	userID, _, ok := tools.CheckRequest(w, r, http.MethodPost)
	if !ok {
		return
	}

	questionIDStr := r.FormValue("question_id")
	if questionIDStr == "" {
		http.Error(w, "From DeleteAltAnswerHandler : no question id parameter", http.StatusBadRequest)
		return
	}

	altQuestionIDStr := r.FormValue("alt_question_id")
	if questionIDStr == "" {
		http.Error(w, "From DeleteAltAnswerHandler : no alt question id parameter", http.StatusBadRequest)
		return
	}

	altAnswerIDStr := r.FormValue("alt_answer_id")
	if altAnswerIDStr == "" {
		http.Error(w, "From DeleteAltAnswerHandler : no alt question id parameter", http.StatusBadRequest)
		return
	}

	altAnswerID, err := strconv.ParseInt(altAnswerIDStr, 10, 64)
	if err != nil {
		http.Error(w, "From DeleteAltAnswerHandler : invalid alt question ID", http.StatusBadRequest)
		return
	}

	if err := queries.DeleteAltAnswer(r.Context(), db.DeleteAltAnswerParams{
		ID:     altAnswerID,
		UserID: userID,
	}); err != nil {
		log.Printf("From DeleteAnswerHandler : DeleteSkill DB error: %v", err)
		errorMessage := url.QueryEscape("Ce champ est utilisé par une question. Impossible de le supprimer pour l'instant.")
		http.Redirect(w, r, data.ErrorMessageURL+"?errormessage="+errorMessage, http.StatusSeeOther)
		return
	}

	altAnswerURL := data.DefaultAltQuestionRoutes.AltAnswersURL +
		"?question_id=" + url.QueryEscape(questionIDStr) +
		"&alt_question_id=" + url.QueryEscape(altQuestionIDStr)
	http.Redirect(w, r, altAnswerURL, http.StatusSeeOther)
}
