package answers

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

func TableAnswersHandler(w http.ResponseWriter, r *http.Request, queries *db.Queries) {
	userID, _, ok := tools.CheckRequest(w, r, http.MethodGet)
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

	question, err := queries.GetQuestionByID(r.Context(), db.GetQuestionByIDParams{
		ID:     questionID,
		UserID: userID,
	})
	if err != nil {
		log.Printf("From TableAnswersHandler, GetQuestionByID DB error: %v", err)
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	answersDB, err := queries.GetAllAnswersByQuestionID(r.Context(), db.GetAllAnswersByQuestionIDParams{
		QuestionID: questionID,
		UserID:     userID,
	})
	if err != nil {
		log.Printf("From TableAnswersHandler, GetAllSkills DB error: %v", err)
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	noAnswer := true
	if len(answersDB) > 0 {
		noAnswer = false
	}

	var actionsURLParameters []data.AnswerActionURLs
	if !noAnswer {
		for _, answer := range answersDB {
			editURL := data.DefaultAnswerRoutes.EditURL + "?question_id=" + url.QueryEscape(questionIDStr) + "&answer_id=" + url.QueryEscape(strconv.FormatInt(answer.ID, 10))
			deleteURL := data.DefaultAnswerRoutes.DeleteURL + "?question_id=" + url.QueryEscape(questionIDStr) + "&answer_id=" + url.QueryEscape(strconv.FormatInt(answer.ID, 10))

			actionsURLParameters = append(actionsURLParameters, data.AnswerActionURLs{
				EditURL:   editURL,
				DeleteURL: deleteURL,
			})
		}
	}

	addURL := data.DefaultAnswerRoutes.AddURL + "?question_id=" + url.QueryEscape(questionIDStr)

	dataPage := data.AnswerPageData{
		Routes:    data.DefaultDashboardRoutes,
		PageTitle: "answers",
		ExtraData: map[string]any{
			"NoAnswer": noAnswer,
			"Action":   actionsURLParameters,
			"Answers":  answersDB,
			"Question": question.Content,
			"AddURL":   addURL,
		},
	}

	RenderTableAnswerPage(w, dataPage)
}

func AddFormAnswerHandler(w http.ResponseWriter, r *http.Request, queries *db.Queries) {
	_, _, ok := tools.CheckRequest(w, r, http.MethodGet)
	if !ok {
		return
	}

	questionIDStr := r.FormValue("question_id")
	if questionIDStr == "" {
		http.Error(w, "From AddFormAnswerHandler : no question id parameter", http.StatusBadRequest)
		return
	}

	addURL := data.DefaultAnswerRoutes.AddURL + "?question_id=" + url.QueryEscape(questionIDStr)

	dataPage := data.AnswerPageData{
		Routes:    data.DefaultDashboardRoutes,
		PageTitle: "add answer",
		ExtraData: map[string]any{
			"AddURL": addURL,
		},
	}
	RenderAddFormAnswer(w, dataPage)
}

func AddAnswerHandler(w http.ResponseWriter, r *http.Request, queries *db.Queries) {
	userID, _, ok := tools.CheckRequest(w, r, http.MethodPost)
	if !ok {
		return
	}

	questionIDStr := r.FormValue("question_id")
	if questionIDStr == "" {
		http.Error(w, "From AddAnswerHandler : no question id parameter", http.StatusBadRequest)
		return
	}
	questionID, err := strconv.ParseInt(questionIDStr, 10, 64)
	if err != nil {
		http.Error(w, "From TableAnswersHandler : invalid question ID", http.StatusBadRequest)
		return
	}

	state := strings.TrimSpace(r.FormValue("state"))
	content := r.FormValue("content")

	if state == "" || content == "" {
		log.Printf("From AddAnswerHandler :state or content field can't be empty")
		errorMessage := url.QueryEscape("La réponse ne peut pas être vide.")
		http.Redirect(w, r, data.ErrorMessageURL+"?errormessage="+errorMessage, http.StatusSeeOther)
		return
	}

	booleen := 0
	if state == "true" {
		booleen = 1
	}

	err = queries.CreateAnswer(r.Context(), db.CreateAnswerParams{
		QuestionID: questionID,
		State:      int64(booleen),
		Content:    content,
		UserID:     userID,
	})
	if err != nil {
		log.Printf("From AddAnswerHandler, CreateAnswer : DB error: %v", err)
		errorMessage := url.QueryEscape("Il ne peut pas exister deux fois la même réponse.")
		http.Redirect(w, r, data.ErrorMessageURL+"?errormessage="+errorMessage, http.StatusSeeOther)
		return
	}

	answerURL := data.DefaultQuestionRoutes.AnswersURL + "?question_id=" + url.QueryEscape(questionIDStr)
	http.Redirect(w, r, answerURL, http.StatusSeeOther)
}

func EditFormAnswerHandler(w http.ResponseWriter, r *http.Request, queries *db.Queries) {
	userID, _, ok := tools.CheckRequest(w, r, http.MethodGet)
	if !ok {
		return
	}

	questionIDStr := r.FormValue("question_id")
	if questionIDStr == "" {
		http.Error(w, "From AddAnswerHandler : no question id parameter", http.StatusBadRequest)
		return
	}

	answerIDStr := r.FormValue("answer_id")
	if answerIDStr == "" {
		http.Error(w, "From EditFormAnswerHandler : no answer id parameter", http.StatusBadRequest)
		return
	}

	answerID, err := strconv.ParseInt(answerIDStr, 10, 64)
	if err != nil {
		http.Error(w, "From EditFormAnswerHandler : invalid answer ID", http.StatusBadRequest)
		return
	}

	answer, err := queries.GetAnswerByID(r.Context(), db.GetAnswerByIDParams{
		ID:     answerID,
		UserID: userID,
	})
	if err != nil {
		log.Printf("From EditFormAnswerHandler : GetAnswerByID DB error: %v", err)
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	editURL := data.DefaultAnswerRoutes.EditURL + "?question_id=" + url.QueryEscape(questionIDStr)
	dataPage := data.AnswerPageData{
		Routes:       data.DefaultDashboardRoutes,
		AnswerRoutes: data.DefaultAnswerRoutes,
		PageTitle:    "edit answer",
		ExtraData: map[string]any{
			"Answer":   answer,
			"AnswerID": answerIDStr,
			"EditURL":  editURL,
		},
	}
	RenderEditFormAnswer(w, dataPage)
}

func EditAnswerHandler(w http.ResponseWriter, r *http.Request, queries *db.Queries) {
	userID, _, ok := tools.CheckRequest(w, r, http.MethodPost)
	if !ok {
		return
	}

	questionIDStr := r.FormValue("question_id")
	if questionIDStr == "" {
		http.Error(w, "From AddAnswerHandler : no question id parameter", http.StatusBadRequest)
		return
	}

	newContent := r.FormValue("new_content")
	if newContent == "" {
		log.Printf("From EditAnswerHandler : field can't be empty")
		errorMessage := url.QueryEscape("La réponse ne peut pas être vide.")
		http.Redirect(w, r, data.ErrorMessageURL+"?errormessage="+errorMessage, http.StatusSeeOther)
		return
	}

	answerIDStr := strings.TrimSpace(r.FormValue("answer_id"))
	if answerIDStr == "" {
		http.Error(w, "From EditAnswerHandler : answerID missing", http.StatusInternalServerError)
		return
	}
	answerID, err := strconv.ParseInt(answerIDStr, 10, 64)
	if err != nil {
		http.Error(w, "From EditAnswerHandler : invalid answer ID", http.StatusBadRequest)
		return
	}

	newStateStr := r.FormValue("new_state")
	if newStateStr == "" {
		http.Error(w, "From EditAnswerHandler : new_state missing", http.StatusInternalServerError)
		return
	}
	newState, err := strconv.ParseInt(newStateStr, 10, 64)
	if err != nil {
		http.Error(w, "From EditAnswerHandler : invalid new state", http.StatusBadRequest)
		return
	}

	if err := queries.UpdateAnswer(r.Context(), db.UpdateAnswerParams{
		State:   newState,
		Content: newContent,
		ID:      answerID,
		UserID:  userID,
	}); err != nil {
		log.Printf("From EditAnswerHandler : UpdateAnswer DB error: %v", err)
		errorMessage := url.QueryEscape("Il ne peut pas exister deux fois la même réponse")
		http.Redirect(w, r, data.ErrorMessageURL+"?errormessage="+errorMessage, http.StatusSeeOther)
		return
	}

	answerURL := data.DefaultQuestionRoutes.AnswersURL + "?question_id=" + url.QueryEscape(questionIDStr)
	http.Redirect(w, r, answerURL, http.StatusSeeOther)
}

func DeleteFormAnswerHandler(w http.ResponseWriter, r *http.Request, queries *db.Queries) {
	userID, _, ok := tools.CheckRequest(w, r, http.MethodGet)
	if !ok {
		return
	}

	questionIDStr := r.FormValue("question_id")
	if questionIDStr == "" {
		http.Error(w, "From AddAnswerHandler : no question id parameter", http.StatusBadRequest)
		return
	}

	answerIDStr := r.FormValue("answer_id")
	if answerIDStr == "" {
		http.Error(w, "From DeleteFormAnswerHandler : no answer id parameter", http.StatusBadRequest)
		return
	}

	answerID, err := strconv.ParseInt(answerIDStr, 10, 64)
	if err != nil {
		http.Error(w, "From DeleteFormAnswerHandler : invalid answer ID", http.StatusBadRequest)
		return
	}

	answer, err := queries.GetAnswerByID(r.Context(), db.GetAnswerByIDParams{
		ID:     answerID,
		UserID: userID,
	})
	if err != nil {
		log.Printf("From DeleteFormAnswerHandler : GetAnswerByID DB error: %v", err)
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	deleteURL := data.DefaultAnswerRoutes.DeleteURL + "?question_id=" + url.QueryEscape(questionIDStr)
	dataPage := data.AnswerPageData{
		Routes:       data.DefaultDashboardRoutes,
		AnswerRoutes: data.DefaultAnswerRoutes,
		PageTitle:    "delete answer",
		ExtraData: map[string]any{
			"Answer":    answer,
			"AnswerID":  answerIDStr,
			"DeleteURL": deleteURL,
		},
	}

	RenderDeleteFormAnswer(w, dataPage)
}

func DeleteAnswerHandler(w http.ResponseWriter, r *http.Request, queries *db.Queries) {
	userID, _, ok := tools.CheckRequest(w, r, http.MethodPost)
	if !ok {
		return
	}

	questionIDStr := r.FormValue("question_id")
	if questionIDStr == "" {
		http.Error(w, "From AddAnswerHandler : no question id parameter", http.StatusBadRequest)
		return
	}

	answerIDStr := r.FormValue("answer_id")
	if answerIDStr == "" {
		http.Error(w, "From DeleteAnswerHandler : no skill id parameter", http.StatusBadRequest)
		return
	}

	answerID, err := strconv.ParseInt(answerIDStr, 10, 64)
	if err != nil {
		http.Error(w, "From DeleteAnswerHandler : invalid skill ID", http.StatusBadRequest)
		return
	}

	if err := queries.DeleteAnswer(r.Context(), db.DeleteAnswerParams{
		ID:     answerID,
		UserID: userID,
	}); err != nil {
		log.Printf("From DeleteAnswerHandler : DeleteSkill DB error: %v", err)
		errorMessage := url.QueryEscape("Ce champ est utilisé par une question. Impossible de le supprimer pour l'instant.")
		http.Redirect(w, r, data.ErrorMessageURL+"?errormessage="+errorMessage, http.StatusSeeOther)
		return
	}

	answerURL := data.DefaultQuestionRoutes.AnswersURL + "?question_id=" + url.QueryEscape(questionIDStr)
	http.Redirect(w, r, answerURL, http.StatusSeeOther)
}
