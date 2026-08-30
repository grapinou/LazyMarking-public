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
		log.Println("From TableAltAnswersHandler -> tools.CheckRequest return not ok")
		return
	}

	questionIDStr := r.URL.Query().Get("question_id")

	if questionIDStr == "" {
		log.Println("From TableAltAnswersHandler : no question id parameter")
		http.Error(w, "Something went wrong !", http.StatusBadRequest)
		return
	}

	altQuestionIDStr := r.URL.Query().Get("alt_question_id")

	if altQuestionIDStr == "" {
		log.Println("From TableAltAnswersHandler : no alt question id parameter")
		http.Error(w, "Something went wrong !", http.StatusBadRequest)
		return
	}

	altQuestionID, err := strconv.ParseInt(altQuestionIDStr, 10, 64)
	if err != nil {
		log.Printf("From TableAltAnswersHandler -> strconv.ParseInt : invalid alt question ID : error : %v", err)
		http.Error(w, "Something went wrong !", http.StatusBadRequest)
		return
	}
	questionID, err := strconv.ParseInt(questionIDStr, 10, 64)
	if err != nil {
		http.Error(w, "Something went wrong !", http.StatusBadRequest)
		return
	}

	altQuestion, err := queries.GetAltQuestionByParentID(r.Context(), db.GetAltQuestionByParentIDParams{
		ID:         altQuestionID,
		QuestionID: questionID,
		UserID:     userID,
	})
	if err != nil {
		tools.HandleOwnedLookupError(w, err, "TableAltAnswersHandler GetAltQuestionByParentID")
		return
	}
	question, err := queries.GetQuestionByID(r.Context(), db.GetQuestionByIDParams{
		ID:     questionID,
		UserID: userID,
	})
	if err != nil {
		tools.HandleOwnedLookupError(w, err, "TableAltAnswersHandler GetQuestionByID")
		return
	}

	altAnswersDB, err := queries.GetAllAltAnswersByAltQuestionID(r.Context(), db.GetAllAltAnswersByAltQuestionIDParams{
		AltQuestionID: altQuestionID,
		UserID:        userID,
	})
	if err != nil {
		log.Printf("From TableAltAnswersHandler -> GetAllAltAnswersByAltQuestionID DB error: %v", err)
		http.Error(w, "DB error", http.StatusInternalServerError)
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

	addURL := data.VariantURL(data.DefaultAltAnswerRoutes.AddURL, questionID, altQuestionID)

	altQuestionsURL := data.QuestionURL(data.DefaultQuestionRoutes.AltQuestionsURL, questionID)

	dataPage := data.AltAnswerPageData{
		Routes:          data.DefaultDashboardRoutes,
		QuestionContext: data.QuestionContext{ID: question.ID, Content: question.Content},
		VariantContext:  data.VariantContext{ID: altQuestion.ID, Content: altQuestion.Content},
		PageTitle:       "alt answers",
		ExtraData: map[string]any{
			"AltQuestionsURL": altQuestionsURL,
			"NoAltAnswer":     noAltAnswer,
			"Action":          actionsURLParameters,
			"AltAnswers":      altAnswersDB,
			"AddURL":          addURL,
		},
	}

	RenderTableAltAnswerPage(w, dataPage)
}

func AddFormAltAnswerHandler(w http.ResponseWriter, r *http.Request, queries *db.Queries) {
	userID, _, ok := tools.CheckRequest(w, r, http.MethodGet)
	if !ok {
		log.Println("From AddFormAltAnswerHandler -> tools.CheckRequest return not ok")
		return
	}

	questionIDStr := r.URL.Query().Get("question_id")
	if questionIDStr == "" {
		log.Println("From AddFormAltAnswerHandler : no question id parameter")
		http.Error(w, "Something went wrong !", http.StatusBadRequest)
		return
	}
	altQuestionIDStr := r.URL.Query().Get("alt_question_id")
	if altQuestionIDStr == "" {
		log.Println("From AddFormAltAnswerHandler : no alt question id parameter")
		http.Error(w, "Something went wrong !", http.StatusBadRequest)
		return
	}
	questionID, err := strconv.ParseInt(questionIDStr, 10, 64)
	if err != nil {
		http.Error(w, "Something went wrong !", http.StatusBadRequest)
		return
	}
	altQuestionID, err := strconv.ParseInt(altQuestionIDStr, 10, 64)
	if err != nil {
		http.Error(w, "Something went wrong !", http.StatusBadRequest)
		return
	}
	altQuestion, err := queries.GetAltQuestionByParentID(r.Context(), db.GetAltQuestionByParentIDParams{ID: altQuestionID, QuestionID: questionID, UserID: userID})
	if err != nil {
		tools.HandleOwnedLookupError(w, err, "AddFormAltAnswerHandler GetAltQuestionByParentID")
		return
	}
	question, err := queries.GetQuestionByID(r.Context(), db.GetQuestionByIDParams{ID: questionID, UserID: userID})
	if err != nil {
		tools.HandleOwnedLookupError(w, err, "AddFormAltAnswerHandler GetQuestionByID")
		return
	}

	altAnswersURL := data.VariantURL(data.DefaultAltQuestionRoutes.AltAnswersURL, questionID, altQuestionID)

	dataPage := data.AltAnswerPageData{
		Routes:          data.DefaultDashboardRoutes,
		AltAnswerRoutes: data.DefaultAltAnswerRoutes,
		QuestionContext: data.QuestionContext{ID: question.ID, Content: question.Content},
		VariantContext:  data.VariantContext{ID: altQuestion.ID, Content: altQuestion.Content},
		PageTitle:       "add alt answer",
		ExtraData: map[string]any{
			"AltAnswersURL": altAnswersURL,
		},
	}
	RenderAddFormAltAnswerPage(w, dataPage)
}

func AddAltAnswerHandler(w http.ResponseWriter, r *http.Request, queries *db.Queries) {
	userID, _, ok := tools.CheckRequest(w, r, http.MethodPost)
	if !ok {
		log.Println("From AddAltAnswerHandler -> tools.CheckRequest return not ok")
		return
	}

	questionIDStr := r.FormValue("question_id")
	if questionIDStr == "" {
		log.Println("From AddAltAnswerHandler : no question id parameter")
		http.Error(w, "Something went wrong !", http.StatusBadRequest)
		return
	}

	altQuestionIDStr := r.FormValue("alt_question_id")
	if altQuestionIDStr == "" {
		log.Println("From AddAltAnswerHandler : no alt question id parameter")
		http.Error(w, "Something went wrong !", http.StatusBadRequest)
		return
	}

	altQuestionID, err := strconv.ParseInt(altQuestionIDStr, 10, 64)
	if err != nil {
		log.Printf("From AddAltAnswerHandler -> strconv.ParseInt : invalid alt question ID, error : %v", err)
		http.Error(w, "Something went wrong !", http.StatusBadRequest)
		return
	}
	questionID, err := strconv.ParseInt(questionIDStr, 10, 64)
	if err != nil {
		http.Error(w, "Something went wrong !", http.StatusBadRequest)
		return
	}

	stateStr := r.FormValue("state")
	if stateStr == "" {
		log.Println("From AddAltAnswerHandler : no state alt question")
		http.Error(w, "Something went wrong !", http.StatusBadRequest)
		return
	}
	state, err := strconv.ParseInt(stateStr, 10, 64)
	if err != nil {
		log.Printf("From AddAltAnswerHandler -> strconv.ParseInt : invalid state, error : %v", err)
		http.Error(w, "Something went wrong !", http.StatusBadRequest)
		return
	}

	content := strings.TrimSpace(r.FormValue("content"))

	rows, err := queries.CreateAltAnswer(r.Context(), db.CreateAltAnswerParams{
		AltQuestionID: altQuestionID,
		QuestionID:    questionID,
		State:         state,
		Content:       content,
		UserID:        userID,
	})
	if err != nil {
		log.Printf("From AddAltAnswerHandler, CreateAltAnswer : DB error: %v", err)
		errorMessage := url.QueryEscape("Il ne peut pas exister deux fois la même réponse ou la réponse ne peut pas être vide")
		http.Redirect(w, r, data.ErrorMessageURL+"?errormessage="+errorMessage, http.StatusSeeOther)
		return
	}
	if !tools.HandleOwnedMutationRows(w, rows, "CreateAltAnswer") {
		return
	}

	altAnswerURL := data.VariantURL(data.DefaultAltQuestionRoutes.AltAnswersURL, questionID, altQuestionID)
	http.Redirect(w, r, altAnswerURL, http.StatusSeeOther)
}

func EditFormAltAnswerHandler(w http.ResponseWriter, r *http.Request, queries *db.Queries) {
	userID, _, ok := tools.CheckRequest(w, r, http.MethodGet)
	if !ok {
		log.Println("From EditFormAltAnswerHandler -> tools.CheckRequest return not ok")
		return
	}

	questionIDStr := r.URL.Query().Get("question_id")
	if questionIDStr == "" {
		log.Println("From EditFormAltAnswerHandler : no question id parameter")
		http.Error(w, "Something went wrong !", http.StatusBadRequest)
		return
	}

	altQuestionIDStr := r.URL.Query().Get("alt_question_id")
	if altQuestionIDStr == "" {
		log.Println("From EditFormAltAnswerHandler :no alt question id parameter")
		http.Error(w, "Something went wrong !", http.StatusBadRequest)
		return
	}

	altAnswerIDStr := r.URL.Query().Get("alt_answer_id")
	if altAnswerIDStr == "" {
		log.Println("From EditFormAltAnswerHandler : no alt answer id parameter")
		http.Error(w, "Something went wrong !", http.StatusBadRequest)
		return
	}

	altAnswerID, err := strconv.ParseInt(altAnswerIDStr, 10, 64)
	if err != nil {
		log.Printf("From EditFormAltAnswerHandler -> strconv.ParseInt : invalid alt answer ID, error : %v", err)
		http.Error(w, "Something went wrong !", http.StatusBadRequest)
		return
	}
	questionID, err := strconv.ParseInt(questionIDStr, 10, 64)
	if err != nil {
		http.Error(w, "Something went wrong !", http.StatusBadRequest)
		return
	}
	altQuestionID, err := strconv.ParseInt(altQuestionIDStr, 10, 64)
	if err != nil {
		http.Error(w, "Something went wrong !", http.StatusBadRequest)
		return
	}

	altAnswer, err := queries.GetAltAnswerByID(r.Context(), db.GetAltAnswerByIDParams{
		ID:            altAnswerID,
		AltQuestionID: altQuestionID,
		QuestionID:    questionID,
		UserID:        userID,
	})
	if err != nil {
		tools.HandleOwnedLookupError(w, err, "EditFormAltAnswerHandler GetAltAnswerByID")
		return
	}
	altQuestion, err := queries.GetAltQuestionByParentID(r.Context(), db.GetAltQuestionByParentIDParams{
		ID:         altQuestionID,
		QuestionID: questionID,
		UserID:     userID,
	})
	if err != nil {
		tools.HandleOwnedLookupError(w, err, "EditFormAltAnswerHandler GetAltQuestionByParentID")
		return
	}
	question, err := queries.GetQuestionByID(r.Context(), db.GetQuestionByIDParams{ID: questionID, UserID: userID})
	if err != nil {
		tools.HandleOwnedLookupError(w, err, "EditFormAltAnswerHandler GetQuestionByID")
		return
	}

	altAnswersURL := data.VariantURL(data.DefaultAltQuestionRoutes.AltAnswersURL, questionID, altQuestionID)

	dataPage := data.AltAnswerPageData{
		Routes:          data.DefaultDashboardRoutes,
		AltAnswerRoutes: data.DefaultAltAnswerRoutes,
		QuestionContext: data.QuestionContext{ID: question.ID, Content: question.Content},
		VariantContext:  data.VariantContext{ID: altQuestion.ID, Content: altQuestion.Content},
		PageTitle:       "edit alt answer",
		ExtraData: map[string]any{
			"AltAnswer":     altAnswer,
			"AltAnswerID":   altAnswerIDStr,
			"AltAnswersURL": altAnswersURL,
		},
	}
	RenderEditFormAltAnswerPage(w, dataPage)
}

func EditAltAnswerHandler(w http.ResponseWriter, r *http.Request, queries *db.Queries) {
	userID, _, ok := tools.CheckRequest(w, r, http.MethodPost)
	if !ok {
		log.Println("From EditAltAnswerHandler -> tools.CheckRequest return not ok")
		return
	}

	questionIDStr := r.FormValue("question_id")
	if questionIDStr == "" {
		log.Println("From EditAltAnswerHandler : no question id parameter")
		http.Error(w, "Something went wrong !", http.StatusBadRequest)
		return
	}

	altQuestionIDStr := r.FormValue("alt_question_id")
	if altQuestionIDStr == "" {
		log.Println("From EditAltAnswerHandler : no alt question id parameter")
		http.Error(w, "Somehing went wrong !", http.StatusBadRequest)
		return
	}

	newContent := strings.TrimSpace(r.FormValue("new_content"))

	altAnswerIDStr := r.FormValue("alt_answer_id")
	if altAnswerIDStr == "" {
		log.Println("From EditAltAnswerHandler : answerID missing")
		http.Error(w, "Something went wrong !", http.StatusBadRequest)
		return
	}
	altAnswerID, err := strconv.ParseInt(altAnswerIDStr, 10, 64)
	if err != nil {
		log.Printf("From EditAltAnswerHandler -> strconv.ParseInt : invalid answer ID, error : %v", err)
		http.Error(w, "Something went wrong !", http.StatusBadRequest)
		return
	}
	questionID, err := strconv.ParseInt(questionIDStr, 10, 64)
	if err != nil {
		http.Error(w, "Something went wrong !", http.StatusBadRequest)
		return
	}
	altQuestionID, err := strconv.ParseInt(altQuestionIDStr, 10, 64)
	if err != nil {
		http.Error(w, "Something went wrong !", http.StatusBadRequest)
		return
	}

	newStateStr := r.FormValue("new_state")
	if newStateStr == "" {
		log.Println("From EditAltAnswerHandler : new_state missing")
		http.Error(w, "Something went wrong !", http.StatusBadRequest)
		return
	}
	newState, err := strconv.ParseInt(newStateStr, 10, 64)
	if err != nil {
		log.Printf("From EditAltAnswerHandler -> strconv.ParseInt : invalid new state, error : %v", err)
		http.Error(w, "Something went wrong !", http.StatusBadRequest)
		return
	}

	rows, err := queries.UpdateAltAnswer(r.Context(), db.UpdateAltAnswerParams{
		State:         newState,
		Content:       newContent,
		ID:            altAnswerID,
		AltQuestionID: altQuestionID,
		UserID:        userID,
		QuestionID:    questionID,
	})
	if err != nil {
		log.Printf("From EditAltAnswerHandler : UpdateAnswer DB error: %v", err)
		errorMessage := url.QueryEscape("Il ne peut pas exister deux fois la même réponse ou la réponse ne peut être vide")
		http.Redirect(w, r, data.ErrorMessageURL+"?errormessage="+errorMessage, http.StatusSeeOther)
		return
	}
	if !tools.HandleOwnedMutationRows(w, rows, "UpdateAltAnswer") {
		return
	}

	altAnswerURL := data.VariantURL(data.DefaultAltQuestionRoutes.AltAnswersURL, questionID, altQuestionID)
	http.Redirect(w, r, altAnswerURL, http.StatusSeeOther)
}

func DeleteFormAltAnswerHandler(w http.ResponseWriter, r *http.Request, queries *db.Queries) {
	userID, _, ok := tools.CheckRequest(w, r, http.MethodGet)
	if !ok {
		log.Println("From DeleteFormAltAnswerHandler -> tools.CheckRequest return not ok")
		return
	}

	questionIDStr := r.URL.Query().Get("question_id")
	if questionIDStr == "" {
		log.Println("From DeleteFormAltAnswerHandler : no question id parameter")
		http.Error(w, "Something went wrong !", http.StatusBadRequest)
		return
	}

	altQuestionIDStr := r.URL.Query().Get("alt_question_id")
	if altQuestionIDStr == "" {
		log.Println("From DeleteFormAltAnswerHandler : no alt question id parameter")
		http.Error(w, "Something went wrong !", http.StatusBadRequest)
		return
	}

	altAnswerIDStr := r.URL.Query().Get("alt_answer_id")
	if altAnswerIDStr == "" {
		log.Println("From DeleteFormAltAnswerHandler : no alt answer id parameter")
		http.Error(w, "Something went wrong !", http.StatusBadRequest)
		return
	}

	altAnswerID, err := strconv.ParseInt(altAnswerIDStr, 10, 64)
	if err != nil {
		log.Printf("From DeleteFormAltAnswerHandler -> strconv.ParseInt : invalid answer ID, error : %v", err)
		http.Error(w, "Something went wrong !", http.StatusBadRequest)
		return
	}
	questionID, err := strconv.ParseInt(questionIDStr, 10, 64)
	if err != nil {
		http.Error(w, "Something went wrong !", http.StatusBadRequest)
		return
	}
	altQuestionID, err := strconv.ParseInt(altQuestionIDStr, 10, 64)
	if err != nil {
		http.Error(w, "Something went wrong !", http.StatusBadRequest)
		return
	}

	altAnswer, err := queries.GetAltAnswerByID(r.Context(), db.GetAltAnswerByIDParams{
		ID:            altAnswerID,
		AltQuestionID: altQuestionID,
		QuestionID:    questionID,
		UserID:        userID,
	})
	if err != nil {
		tools.HandleOwnedLookupError(w, err, "DeleteFormAltAnswerHandler GetAltAnswerByID")
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
			"CancelURL":     data.VariantURL(data.DefaultAltQuestionRoutes.AltAnswersURL, questionID, altQuestionID),
		},
	}

	RenderDeleteFormAltAnswerPage(w, dataPage)
}

func DeleteAltAnswerHandler(w http.ResponseWriter, r *http.Request, queries *db.Queries) {
	userID, _, ok := tools.CheckRequest(w, r, http.MethodPost)
	if !ok {
		log.Println("From DeleteAltAnswerHandler -> tools.CheckRequest return not ok")
		return
	}

	questionIDStr := r.FormValue("question_id")
	if questionIDStr == "" {
		log.Println("From DeleteAltAnswerHandler : no question id parameter")
		http.Error(w, "Something went wrong !", http.StatusBadRequest)
		return
	}

	altQuestionIDStr := r.FormValue("alt_question_id")
	if altQuestionIDStr == "" {
		log.Println("From DeleteAltAnswerHandler : no alt question id parameter")
		http.Error(w, "Something went wrong !", http.StatusBadRequest)
		return
	}

	altAnswerIDStr := r.FormValue("alt_answer_id")
	if altAnswerIDStr == "" {
		log.Println("From DeleteAltAnswerHandler : no alt answer id parameter")
		http.Error(w, "Something went wrong !", http.StatusBadRequest)
		return
	}

	altAnswerID, err := strconv.ParseInt(altAnswerIDStr, 10, 64)
	if err != nil {
		log.Printf("From DeleteAltAnswerHandler -> strconv.ParseInt : invalid alt question ID, error : %v", err)
		http.Error(w, "Something went wrong !", http.StatusBadRequest)
		return
	}
	questionID, err := strconv.ParseInt(questionIDStr, 10, 64)
	if err != nil {
		http.Error(w, "Something went wrong !", http.StatusBadRequest)
		return
	}
	altQuestionID, err := strconv.ParseInt(altQuestionIDStr, 10, 64)
	if err != nil {
		http.Error(w, "Something went wrong !", http.StatusBadRequest)
		return
	}

	rows, err := queries.DeleteAltAnswer(r.Context(), db.DeleteAltAnswerParams{
		ID:            altAnswerID,
		AltQuestionID: altQuestionID,
		QuestionID:    questionID,
		UserID:        userID,
	})
	if err != nil {
		log.Printf("From DeleteAnswerHandler : DeleteSkill DB error: %v", err)
		errorMessage := url.QueryEscape("Ce champ est utilisé par une question. Impossible de le supprimer pour l'instant.")
		http.Redirect(w, r, data.ErrorMessageURL+"?errormessage="+errorMessage, http.StatusSeeOther)
		return
	}
	if !tools.HandleOwnedMutationRows(w, rows, "DeleteAltAnswer") {
		return
	}

	altAnswerURL := data.DefaultAltQuestionRoutes.AltAnswersURL +
		"?question_id=" + url.QueryEscape(questionIDStr) +
		"&alt_question_id=" + url.QueryEscape(altQuestionIDStr)
	http.Redirect(w, r, altAnswerURL, http.StatusSeeOther)
}
