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
	userID, _, ok := tools.CheckRequest(w, r, http.MethodGet)
	if !ok {
		log.Println("From TableAltQuestionsHandler -> tools.CheckRequest return not ok")
		return
	}

	questionIDStr := r.URL.Query().Get("question_id")

	if questionIDStr == "" {
		log.Println("From TableAltQuestionsHandler : no question id parameter")
		http.Error(w, "Something went wrong !", http.StatusBadRequest)
		return
	}

	questionID, err := strconv.ParseInt(questionIDStr, 10, 64)
	if err != nil {
		log.Printf("From TableAltQuestionsHandler -> strconv.ParseInt, invalid question id parameter, error : %v", err)
		http.Error(w, "Something went wrong !", http.StatusBadRequest)
		return
	}

	question, err := queries.GetQuestionByID(r.Context(), db.GetQuestionByIDParams{
		ID:     questionID,
		UserID: userID,
	})
	if err != nil {
		log.Printf("From TableAltQuestionsHandler -> GetQuestionByID DB error: %v", err)
		http.Error(w, "DB Error", http.StatusInternalServerError)
		return
	}

	altQuestionsDB, err := queries.GetAllAltQuestions(r.Context(), db.GetAllAltQuestionsParams{
		QuestionID: questionID,
		UserID:     userID,
	})
	if err != nil {
		log.Printf("From TableAltQuestionsHandler -> GetAllAltQuestions DB error: %v", err)
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
			params := "?question_id=" + url.QueryEscape(strconv.FormatInt(questionID, 10)) + "&alt_question_id=" + url.QueryEscape(strconv.FormatInt(altQuestion.ID, 10))
			editURL := data.DefaultAltQuestionRoutes.EditURL + params
			deleteURL := data.DefaultAltQuestionRoutes.DeleteURL + params
			altAnswersURL := data.DefaultAltQuestionRoutes.AltAnswersURL + params

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
			"UserID":          userID,
			"QuestionContent": question.Content,
			"NoAltQuestion":   noAltQuestion,
			"AltQuestions":    altQuestionsDB,
			"Action":          actionsURLParameters,
			"AddURL":          addURL,
		},
	}

	RenderTableAltQuestionPage(w, dataPage)
}

func AddFormAltQuestionHandler(w http.ResponseWriter, r *http.Request, queries *db.Queries) {
	_, _, ok := tools.CheckRequest(w, r, http.MethodGet)
	if !ok {
		log.Println("From AddFormAltQuestionHandler -> tools.CheckRequest return not ok")
		return
	}

	questionIDStr := r.URL.Query().Get("question_id")
	if questionIDStr == "" {
		log.Println("From AddFormAltQuestionHandler : no question id parameter")
		http.Error(w, "Something went wrong !", http.StatusBadRequest)
		return
	}

	dataPage := data.AltQuestionPageData{
		Routes:            data.DefaultDashboardRoutes,
		AltQuestionRoutes: data.DefaultAltQuestionRoutes,
		PageTitle:         "add alt question",
		ExtraData: map[string]any{
			"QuestionID": questionIDStr,
		},
	}
	RenderAddFormAltQuestionPage(w, dataPage)
}

func AddAltQuestionHandler(w http.ResponseWriter, r *http.Request, queries *db.Queries) {
	userID, _, ok := tools.CheckRequest(w, r, http.MethodPost)
	if !ok {
		log.Println("From AddAltQuestionHandler -> tools.CheckRequest return not ok")
		return
	}

	questionIDStr := r.FormValue("question_id")
	if questionIDStr == "" {
		log.Println("From AddAltQuestionHandler : no question id parameter")
		http.Error(w, "Something went wrong !", http.StatusBadRequest)
		return
	}
	questionID, err := strconv.ParseInt(questionIDStr, 10, 64)
	if err != nil {
		log.Printf("From AddAltQuestionHandler -> strconv.ParseInt, invalid question ID, error : %v", err)
		http.Error(w, "Something went wrong !", http.StatusBadRequest)
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
		log.Println("From EditFormAltQuestionHandler -> tools.CheckRequest return not ok")
		return
	}

	questionIDStr := r.URL.Query().Get("question_id")
	if questionIDStr == "" {
		log.Println("From EditFormAltQuestionHandler : no question id parameter")
		http.Error(w, "Something went wrong !", http.StatusBadRequest)
		return
	}

	altQuestionIDStr := r.URL.Query().Get("alt_question_id")
	if altQuestionIDStr == "" {
		log.Println("From EditFormAltQuestionHandler : no alt question id parameter")
		http.Error(w, "Something went wrong !", http.StatusBadRequest)
		return
	}

	altQuestionID, err := strconv.ParseInt(altQuestionIDStr, 10, 64)
	if err != nil {
		log.Printf("From EditFormAltQuestionHandler -> strconv.ParseInt, invalid alt question ID, error : %v", err)
		http.Error(w, "Something went wrong !", http.StatusBadRequest)
		return
	}

	altQuestion, err := queries.GetAltQuestionByID(r.Context(), db.GetAltQuestionByIDParams{
		ID:     altQuestionID,
		UserID: userID,
	})
	if err != nil {
		log.Printf("From EditFormAltQuestionHandler -> GetAltQuestionByID DB error: %v", err)
		http.Error(w, "DB error", http.StatusInternalServerError)
		return
	}

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
		log.Println("From EditAltQuestionHandler -> tools.CheckRequest return not ok")
		return
	}

	questionIDStr := r.FormValue("question_id")
	if questionIDStr == "" {
		log.Println("From EditAltQuestionHandler : no question id parameter")
		http.Error(w, "Something went wrong !", http.StatusBadRequest)
		return
	}

	newContent := strings.TrimSpace(r.FormValue("new_content"))

	altQuestionIDStr := r.FormValue("alt_question_id")
	if altQuestionIDStr == "" {
		log.Println("From EditAltQuestionHandler : no altQuestionID parameter")
		http.Error(w, "Something went wrong !", http.StatusBadRequest)
		return
	}
	altQuestionID, err := strconv.ParseInt(altQuestionIDStr, 10, 64)
	if err != nil {
		log.Printf("From  EditAltQuestionHandler -> strconv.ParseInt, invalid altQuestion ID, error : %v", err)
		http.Error(w, "Something went wrong !", http.StatusBadRequest)
		return
	}

	if err := queries.UpdateAltQuestion(r.Context(), db.UpdateAltQuestionParams{
		Content: newContent,
		ID:      altQuestionID,
		UserID:  userID,
	}); err != nil {
		log.Printf("From  EditAltQuestionHandler : UpdateAltQuestion DB error: %v", err)
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
		log.Println("From DeleteFormAltQuestionHandler -> tools.CheckRequest return not ok")
		return
	}

	questionIDStr := r.URL.Query().Get("question_id")
	if questionIDStr == "" {
		log.Println("From  DeleteFormAltQuestionHandler : no question id parameter")
		http.Error(w, "Something went wrong !", http.StatusBadRequest)
		return
	}

	altQuestionIDStr := r.URL.Query().Get("alt_question_id")
	if altQuestionIDStr == "" {
		log.Println("From DeleteFormAltQuestionHandler : no alt question id parameter")
		http.Error(w, "Something went wrong !", http.StatusBadRequest)
		return
	}

	altQuestionID, err := strconv.ParseInt(altQuestionIDStr, 10, 64)
	if err != nil {
		log.Printf("From DeleteFormAltQuestionHandler -> strconv.ParseInt, invalid answer ID, error : %v", err)
		http.Error(w, "Something went wrong !", http.StatusBadRequest)
		return
	}

	altQuestion, err := queries.GetAltQuestionByID(r.Context(), db.GetAltQuestionByIDParams{
		ID:     altQuestionID,
		UserID: userID,
	})
	if err != nil {
		log.Printf("From DeleteFormAltQuestionHandler -> GetAltQuestionByID : DB error: %v", err)
		http.Error(w, "DB error", http.StatusInternalServerError)
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
		log.Println("From DeleteAltQuestionHandler -> tools.CheckRequest return not ok")
		return
	}

	questionIDStr := r.FormValue("question_id")
	if questionIDStr == "" {
		log.Println("From DeleteAltQuestionHandler : no question id parameter")
		http.Error(w, "Something went wrong !", http.StatusBadRequest)
		return
	}

	altQuestionIDStr := r.FormValue("alt_question_id")
	if altQuestionIDStr == "" {
		log.Println("From DeleteAltQuestionHandler : no alt question id parameter")
		http.Error(w, "Something went wrong !", http.StatusBadRequest)
		return
	}

	altQuestionID, err := strconv.ParseInt(altQuestionIDStr, 10, 64)
	if err != nil {
		log.Printf("From DeleteAltQuestionHandler -> strconv.ParseInt : invalid alt question ID, error : %v", err)
		http.Error(w, "Something went wrong !", http.StatusBadRequest)
		return
	}

	if err := queries.DeleteAltQuestion(r.Context(), db.DeleteAltQuestionParams{
		ID:     altQuestionID,
		UserID: userID,
	}); err != nil {
		log.Printf("From DeleteAltQuestionHandler -> DeleteAltQuestion DB error: %v", err)
		http.Error(w, "DB error", http.StatusInternalServerError)
		return
	}

	altQuestionURL := data.DefaultQuestionRoutes.AltQuestionsURL + "?question_id=" + url.QueryEscape(questionIDStr)
	http.Redirect(w, r, altQuestionURL, http.StatusSeeOther)
}
