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
		log.Println("From TableAnswersHandler -> tools.CheckRequest return not ok")
		return
	}

	questionIDStr := r.URL.Query().Get("question_id")

	if questionIDStr == "" {
		log.Println("From TableAnswersHandler : no question id parameter")
		http.Error(w, "Something went wrong !", http.StatusBadRequest)
		return
	}

	questionID, err := strconv.ParseInt(questionIDStr, 10, 64)
	if err != nil {
		log.Printf("From TableAnswersHandler -> strconv.ParseInt, invalid question ID, error : %v", err)
		http.Error(w, "Something went wrong !", http.StatusBadRequest)
		return
	}

	question, err := queries.GetQuestionByID(r.Context(), db.GetQuestionByIDParams{
		ID:     questionID,
		UserID: userID,
	})
	if err != nil {
		tools.HandleOwnedLookupError(w, err, "TableAnswersHandler GetQuestionByID")
		return
	}

	answersDB, err := queries.GetAllAnswersByQuestionID(r.Context(), db.GetAllAnswersByQuestionIDParams{
		QuestionID: questionID,
		UserID:     userID,
	})
	if err != nil {
		log.Printf("From TableAnswersHandler -> GetAllAnswersByQuestionID DB error: %v", err)
		http.Error(w, "DB error", http.StatusInternalServerError)
		return
	}

	noAnswer := true
	if len(answersDB) > 0 {
		noAnswer = false
	}

	var actionsURLParameters []data.AnswerActionURLs
	if !noAnswer {
		for _, answer := range answersDB {
			params := "?question_id=" + url.QueryEscape(questionIDStr) + "&answer_id=" + url.QueryEscape(strconv.FormatInt(answer.ID, 10))
			editURL := data.DefaultAnswerRoutes.EditURL + params
			deleteURL := data.DefaultAnswerRoutes.DeleteURL + params

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
	userID, _, ok := tools.CheckRequest(w, r, http.MethodGet)
	if !ok {
		log.Println("From AddFormAnswerHandler -> tools.CheckRequest return not ok")
		return
	}

	questionIDStr := r.URL.Query().Get("question_id")
	if questionIDStr == "" {
		log.Println("From AddFormAnswerHandler : no question id parameter")
		http.Error(w, "Something went wrong !", http.StatusBadRequest)
		return
	}
	questionID, err := strconv.ParseInt(questionIDStr, 10, 64)
	if err != nil {
		http.Error(w, "Something went wrong !", http.StatusBadRequest)
		return
	}
	question, err := queries.GetQuestionByID(r.Context(), db.GetQuestionByIDParams{ID: questionID, UserID: userID})
	if err != nil {
		tools.HandleOwnedLookupError(w, err, "AddFormAnswerHandler GetQuestionByID")
		return
	}
	answersURL := data.DefaultQuestionRoutes.AnswersURL + "?question_id=" + url.QueryEscape(questionIDStr)

	dataPage := data.AnswerPageData{
		Routes:       data.DefaultDashboardRoutes,
		AnswerRoutes: data.DefaultAnswerRoutes,
		PageTitle:    "add answer",
		ExtraData: map[string]any{
			"QuestionID":      questionIDStr,
			"QuestionContent": question.Content,
			"AnswersURL":      answersURL,
		},
	}
	RenderAddFormAnswerPage(w, dataPage)
}

func AddAnswerHandler(w http.ResponseWriter, r *http.Request, queries *db.Queries) {
	userID, _, ok := tools.CheckRequest(w, r, http.MethodPost)
	if !ok {
		log.Println("From AddAnswerHandler -> tools.CheckRequest return not ok")
		return
	}

	questionIDStr := r.FormValue("question_id")
	if questionIDStr == "" {
		log.Println("From AddAnswerHandler : no question id parameter")
		http.Error(w, "Something went wrong !", http.StatusBadRequest)
		return
	}
	questionID, err := strconv.ParseInt(questionIDStr, 10, 64)
	if err != nil {
		log.Printf("From AddAnswerHandler -> strconv.ParseInt, invalid question ID, error : %v", err)
		http.Error(w, "Something went wrong !", http.StatusBadRequest)
		return
	}

	stateStr := r.FormValue("state")
	content := strings.TrimSpace(r.FormValue("content"))

	state, err := strconv.ParseInt(stateStr, 10, 64)
	if err != nil {
		log.Printf("From AddAnswerHandler -> strconv.ParseInt, invalid state, error : %v", err)
		http.Error(w, "Something went wrong !", http.StatusBadRequest)
		return
	}

	rows, err := queries.CreateAnswer(r.Context(), db.CreateAnswerParams{
		QuestionID: questionID,
		State:      state,
		Content:    content,
		UserID:     userID,
	})
	if err != nil {
		log.Printf("From AddAnswerHandler -> CreateAnswer : DB error: %v", err)
		errorMessage := url.QueryEscape("Il ne peut pas exister deux fois la même réponse ou la réponse ne peut pas être vide.")
		http.Redirect(w, r, data.ErrorMessageURL+"?errormessage="+errorMessage, http.StatusSeeOther)
		return
	}
	if !tools.HandleOwnedMutationRows(w, rows, "CreateAnswer") {
		return
	}

	answerURL := data.DefaultQuestionRoutes.AnswersURL + "?question_id=" + url.QueryEscape(questionIDStr)
	http.Redirect(w, r, answerURL, http.StatusSeeOther)
}

func EditFormAnswerHandler(w http.ResponseWriter, r *http.Request, queries *db.Queries) {
	userID, _, ok := tools.CheckRequest(w, r, http.MethodGet)
	if !ok {
		log.Println("From EditFormAnswerHandler -> tools.CheckRequest return not ok")
		return
	}

	questionIDStr := r.URL.Query().Get("question_id")
	if questionIDStr == "" {
		log.Println("From EditFormAnswerHandler : no question id parameter")
		http.Error(w, "Something went wrong !", http.StatusBadRequest)
		return
	}

	answerIDStr := r.URL.Query().Get("answer_id")
	if answerIDStr == "" {
		log.Println("From EditFormAnswerHandler : no answer id parameter")
		http.Error(w, "Something went wrong !", http.StatusBadRequest)
		return
	}

	answerID, err := strconv.ParseInt(answerIDStr, 10, 64)
	if err != nil {
		log.Printf("From EditFormAnswerHandler -> strconv.ParseInt : invalid answer ID, error : %v", err)
		http.Error(w, "Something went wrong !", http.StatusBadRequest)
		return
	}
	questionID, err := strconv.ParseInt(questionIDStr, 10, 64)
	if err != nil {
		http.Error(w, "Something went wrong !", http.StatusBadRequest)
		return
	}

	answer, err := queries.GetAnswerByID(r.Context(), db.GetAnswerByIDParams{
		ID:         answerID,
		QuestionID: questionID,
		UserID:     userID,
	})
	if err != nil {
		tools.HandleOwnedLookupError(w, err, "EditFormAnswerHandler GetAnswerByID")
		return
	}
	question, err := queries.GetQuestionByID(r.Context(), db.GetQuestionByIDParams{
		ID:     questionID,
		UserID: userID,
	})
	if err != nil {
		tools.HandleOwnedLookupError(w, err, "EditFormAnswerHandler GetQuestionByID")
		return
	}
	answersURL := data.DefaultQuestionRoutes.AnswersURL + "?question_id=" + url.QueryEscape(questionIDStr)

	dataPage := data.AnswerPageData{
		Routes:       data.DefaultDashboardRoutes,
		AnswerRoutes: data.DefaultAnswerRoutes,
		PageTitle:    "edit answer",
		ExtraData: map[string]any{
			"Answer":          answer,
			"AnswerID":        answerIDStr,
			"QuestionID":      questionIDStr,
			"QuestionContent": question.Content,
			"AnswersURL":      answersURL,
		},
	}
	RenderEditFormAnswerPage(w, dataPage)
}

func EditAnswerHandler(w http.ResponseWriter, r *http.Request, queries *db.Queries) {
	userID, _, ok := tools.CheckRequest(w, r, http.MethodPost)
	if !ok {
		log.Println("From EditAnswerHandler -> tools.CheckRequest return not ok")
		return
	}

	questionIDStr := r.FormValue("question_id")
	if questionIDStr == "" {
		log.Println("From EditAnswerHandler : no question id parameter")
		http.Error(w, "Something went wrong !", http.StatusBadRequest)
		return
	}
	questionID, err := strconv.ParseInt(questionIDStr, 10, 64)
	if err != nil {
		http.Error(w, "Something went wrong !", http.StatusBadRequest)
		return
	}

	newContent := strings.TrimSpace(r.FormValue("new_content"))

	answerIDStr := r.FormValue("answer_id")
	if answerIDStr == "" {
		log.Println("From EditAnswerHandler : no answer id parameter")
		http.Error(w, "Something went wrong !", http.StatusBadRequest)
		return
	}
	answerID, err := strconv.ParseInt(answerIDStr, 10, 64)
	if err != nil {
		log.Printf("From EditAnswerHandler -> strconv.ParseInt : invalid answer ID, error : %v", err)
		http.Error(w, "Something went wrong !", http.StatusBadRequest)
		return
	}

	newStateStr := r.FormValue("new_state")

	newState, err := strconv.ParseInt(newStateStr, 10, 64)
	if err != nil {
		log.Printf("From EditAnswerHandler -> strconv.ParseInt : invalid new state, error : %v", err)
		http.Error(w, "Something went wrong !", http.StatusBadRequest)
		return
	}

	rows, err := queries.UpdateAnswer(r.Context(), db.UpdateAnswerParams{
		State:      newState,
		Content:    newContent,
		ID:         answerID,
		QuestionID: questionID,
		UserID:     userID,
	})
	if err != nil {
		log.Printf("From EditAnswerHandler : UpdateAnswer DB error: %v", err)
		errorMessage := url.QueryEscape("Il ne peut pas exister deux fois la même réponse, ou la réponse ne peut pas être vide.")
		http.Redirect(w, r, data.ErrorMessageURL+"?errormessage="+errorMessage, http.StatusSeeOther)
		return
	}
	if !tools.HandleOwnedMutationRows(w, rows, "UpdateAnswer") {
		return
	}

	answerURL := data.DefaultQuestionRoutes.AnswersURL + "?question_id=" + url.QueryEscape(questionIDStr)
	http.Redirect(w, r, answerURL, http.StatusSeeOther)
}

func DeleteFormAnswerHandler(w http.ResponseWriter, r *http.Request, queries *db.Queries) {
	userID, _, ok := tools.CheckRequest(w, r, http.MethodGet)
	if !ok {
		log.Println("From DeleteFormAnswerHandler -> tools.CheckRequest return not ok")
		return
	}

	questionIDStr := r.URL.Query().Get("question_id")
	if questionIDStr == "" {
		log.Println("From DeleteFormAnswerHandler : no question id parameter")
		http.Error(w, "Something went wrong !", http.StatusBadRequest)
		return
	}

	answerIDStr := r.URL.Query().Get("answer_id")
	if answerIDStr == "" {
		log.Println("From DeleteFormAnswerHandler : no answer id parameter")
		http.Error(w, "Something went wrong !", http.StatusBadRequest)
		return
	}

	answerID, err := strconv.ParseInt(answerIDStr, 10, 64)
	if err != nil {
		log.Printf("From DeleteFormAnswerHandler -> strconv.ParseInt, invalid answer ID, error : %v", err)
		http.Error(w, "Something went wrong !", http.StatusBadRequest)
		return
	}
	questionID, err := strconv.ParseInt(questionIDStr, 10, 64)
	if err != nil {
		http.Error(w, "Something went wrong !", http.StatusBadRequest)
		return
	}

	answer, err := queries.GetAnswerByID(r.Context(), db.GetAnswerByIDParams{
		ID:         answerID,
		QuestionID: questionID,
		UserID:     userID,
	})
	if err != nil {
		tools.HandleOwnedLookupError(w, err, "DeleteFormAnswerHandler GetAnswerByID")
		return
	}

	dataPage := data.AnswerPageData{
		Routes:       data.DefaultDashboardRoutes,
		AnswerRoutes: data.DefaultAnswerRoutes,
		PageTitle:    "delete answer",
		ExtraData: map[string]any{
			"Answer":     answer,
			"AnswerID":   answerIDStr,
			"QuestionID": questionIDStr,
		},
	}

	RenderDeleteFormAnswerPage(w, dataPage)
}

func DeleteAnswerHandler(w http.ResponseWriter, r *http.Request, queries *db.Queries) {
	userID, _, ok := tools.CheckRequest(w, r, http.MethodPost)
	if !ok {
		log.Println("From DeleteAnswerHandler -> tools.CheckRequest return not ok")
		return
	}

	questionIDStr := r.FormValue("question_id")
	if questionIDStr == "" {
		http.Error(w, "From DeleteAnswerHandler : no question id parameter", http.StatusBadRequest)
		return
	}

	answerIDStr := r.FormValue("answer_id")
	if answerIDStr == "" {
		http.Error(w, "From DeleteAnswerHandler : no answer id parameter", http.StatusBadRequest)
		return
	}

	answerID, err := strconv.ParseInt(answerIDStr, 10, 64)
	if err != nil {
		log.Printf("From DeleteAnswerHandler -> strconv.ParseInt, invalid answer ID, error : %v", err)
		http.Error(w, "Something went wrong !", http.StatusBadRequest)
		return
	}
	questionID, err := strconv.ParseInt(questionIDStr, 10, 64)
	if err != nil {
		http.Error(w, "Something went wrong !", http.StatusBadRequest)
		return
	}

	rows, err := queries.DeleteAnswer(r.Context(), db.DeleteAnswerParams{
		ID:         answerID,
		QuestionID: questionID,
		UserID:     userID,
	})
	if err != nil {
		log.Printf("From DeleteAnswerHandler -> DeleteSkill DB error: %v", err)
		errorMessage := url.QueryEscape("Ce champ est utilisé par une question. Impossible de le supprimer pour l'instant.")
		http.Redirect(w, r, data.ErrorMessageURL+"?errormessage="+errorMessage, http.StatusSeeOther)
		return
	}
	if !tools.HandleOwnedMutationRows(w, rows, "DeleteAnswer") {
		return
	}

	answerURL := data.DefaultQuestionRoutes.AnswersURL + "?question_id=" + url.QueryEscape(questionIDStr)
	http.Redirect(w, r, answerURL, http.StatusSeeOther)
}
