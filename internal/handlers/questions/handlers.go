package questions

import (
	"context"
	"database/sql"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/grapinou/LazyMarking/internal/db"
	"github.com/grapinou/LazyMarking/internal/handlers/tools"
	"github.com/grapinou/LazyMarking/internal/questionfamilies"
	"github.com/grapinou/LazyMarking/internal/templates/data"
)

var removeStoredImageFile = tools.RemoveStoredImageFile

func TableQuestionsHandler(w http.ResponseWriter, r *http.Request, queries *db.Queries) {
	userID, _, ok := tools.CheckRequest(w, r, http.MethodGet)
	if !ok {
		log.Println("From TableQuestionsHandler -> tools.CheckRequest return not ok")
		return
	}

	families, err := loadQuestionFamilies(r.Context(), queries, userID)
	if err != nil {
		log.Printf("From TableQuestionsHandler -> loadQuestionFamilies DB error: %v", err)
		http.Error(w, "DB Error", http.StatusInternalServerError)
		return
	}

	noQuestion := true
	if len(families) > 0 {
		noQuestion = false
	}

	var actionsURLParameters []data.QuestionActionURLs
	if !noQuestion {
		for _, family := range families {
			params := "?question_id=" + url.QueryEscape(strconv.FormatInt(family.Main.ID, 10))
			editURL := data.DefaultQuestionRoutes.EditURL + params
			deleteURL := data.DefaultQuestionRoutes.DeleteURL + params
			answersURL := data.DefaultQuestionRoutes.AnswersURL + params
			altQuestionsURL := data.DefaultQuestionRoutes.AltQuestionsURL + params
			imageURL := data.DefaultQuestionRoutes.ImageURL + params
			previewURL := data.DefaultQuestionRoutes.PreviewURL + params

			actionsURLParameters = append(actionsURLParameters, data.QuestionActionURLs{
				EditURL:         editURL,
				DeleteURL:       deleteURL,
				AnswersURL:      answersURL,
				AltQuestionsURL: altQuestionsURL,
				ImageURL:        imageURL,
				PreviewURL:      previewURL,
			})
		}
	}

	dataPage := data.QuestionPageData{
		Routes:         data.DefaultDashboardRoutes,
		QuestionRoutes: data.DefaultQuestionRoutes,
		PageTitle:      "questions",
		ExtraData: map[string]any{
			"UserID":           userID,
			"NoQuestion":       noQuestion,
			"QuestionFamilies": families,
			"Action":           actionsURLParameters,
		},
	}

	RenderTableQuestionPage(w, dataPage)
}

func loadQuestionFamilies(ctx context.Context, queries *db.Queries, userID int64) ([]questionfamilies.QuestionFamily, error) {
	questionsDB, err := queries.GetAllQuestions(ctx, userID)
	if err != nil {
		return nil, err
	}
	alternativesDB, err := queries.GetAllOwnedAltQuestions(ctx, userID)
	if err != nil {
		return nil, err
	}

	questions := make([]questionfamilies.Question, 0, len(questionsDB))
	for _, question := range questionsDB {
		questions = append(questions, questionfamilies.Question{ID: question.ID, Content: question.Content})
	}
	variants := make([]questionfamilies.Variant, 0, len(alternativesDB))
	for _, alternative := range alternativesDB {
		variants = append(variants, questionfamilies.Variant{
			ID: alternative.ID, QuestionID: alternative.QuestionID, Content: alternative.Content,
		})
	}
	return questionfamilies.Build(questions, variants), nil
}

func AddFormQuestionsHandler(w http.ResponseWriter, r *http.Request, queries *db.Queries) {
	userID, _, ok := tools.CheckRequest(w, r, http.MethodGet)
	if !ok {
		log.Println("From AddFormQuestionsHandler -> tools.CheckRequest return not ok")
		return
	}

	features, ok := tools.GetAllFeaturesQuestion(r, userID, queries)
	if !ok {
		log.Println("From AddFormQuestionsHandler -> tools.GetAllFeaturesQuestion return not ok")
		http.Error(w, "Something went wrong !", http.StatusInternalServerError)
		return
	}

	for _, feature := range features {
		featurelen, ok := tools.GetSliceLen(feature)
		if !ok {
			log.Println("From AddFormQuestionsHandler -> tools.GetSliceLen : return not ok, []db.type error")
			http.Error(w, "Something went wrong !", http.StatusInternalServerError)
			return
		}
		if featurelen == 0 {
			errorMessage := url.QueryEscape("Une question doit avoir chaque caractéristique. Vérifiez que chaque caractéristique contient au moins une valeur.")
			http.Redirect(w, r, data.ErrorMessageURL+"?errormessage="+errorMessage, http.StatusSeeOther)
			return
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

	RenderAddFormQuestionPage(w, dataPage)
}

func AddQuestionsHandler(w http.ResponseWriter, r *http.Request, queries *db.Queries) {
	userID, _, ok := tools.CheckRequest(w, r, http.MethodPost)
	if !ok {
		log.Println("From AddQuestionsHandler -> tools.CheckRequest return not ok")
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form", http.StatusBadRequest)
		return
	}
	content := strings.TrimSpace(r.FormValue("content"))
	if content == "" {
		http.Error(w, "Missing question content", http.StatusBadRequest)
		return
	}
	intIDs := make(map[string]int64, 6)
	for _, feature := range []string{"subjectID", "themeID", "yearLevelID", "skillID", "difficultyID", "pointID"} {
		intID, ok := tools.StrToInt(r.FormValue(feature))
		if !ok {
			log.Println("From AddQuestionsHandler -> tools.StrToInt return not ok, no feature id parameter or one missing")
			http.Error(w, "Something went wrong !", http.StatusBadRequest)
			return
		}
		intIDs[feature] = intID
	}

	rows, err := queries.CreateQuestion(r.Context(), db.CreateQuestionParams{
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
		log.Printf("From AddQuestionsHandler -> DB CreateQuestion error : %v", err)
		errorMessage := url.QueryEscape("Il ne peut pas exister deux fois la même question. Ou la question ne peut être vide.")
		http.Redirect(w, r, data.ErrorMessageURL+"?errormessage="+errorMessage, http.StatusSeeOther)
		return
	}
	if !tools.HandleOwnedMutationRows(w, rows, "CreateQuestion") {
		return
	}

	http.Redirect(w, r, data.DefaultDashboardRoutes.QuestionsURL, http.StatusSeeOther)
}

func EditFormQuestionHandler(w http.ResponseWriter, r *http.Request, queries *db.Queries) {
	userID, _, ok := tools.CheckRequest(w, r, http.MethodGet)
	if !ok {
		log.Println("From EditFormQuestionHandler -> tools.CheckRequest return not ok")
		return
	}

	questionIDStr := r.URL.Query().Get("question_id")
	if questionIDStr == "" {
		log.Println("From EditFormQuestionHandler : no question id parameter")
		http.Error(w, "Something went wrong !", http.StatusBadRequest)
		return
	}

	questionID, err := strconv.ParseInt(questionIDStr, 10, 64)
	if err != nil {
		log.Printf("From EditFormQuestionHandler -> strconv.ParseInt invalid question id parameter, error : %v", err)
		http.Error(w, "Something went wrong !", http.StatusBadRequest)
		return
	}

	question, err := queries.GetQuestionByID(r.Context(), db.GetQuestionByIDParams{
		ID:     questionID,
		UserID: userID,
	})
	if err != nil {
		tools.HandleOwnedLookupError(w, err, "EditFormQuestionHandler GetQuestionByID")
		return
	}

	features, ok := tools.GetAllFeaturesQuestion(r, userID, queries)
	if !ok {
		log.Println("From EditFormQuestionHandler -> tools.GetAllFeaturesQuestion : return not ok")
		http.Error(w, "Something went wrong !", http.StatusInternalServerError)
		return
	}

	for _, feature := range features {
		featurelen, ok := tools.GetSliceLen(feature)
		if !ok {
			log.Println("From EditFormQuestionHandler -> tools.GetSliceLen return not ok, []db.type error")
			http.Error(w, "Something went wrong !", http.StatusInternalServerError)
			return
		}
		if featurelen == 0 {
			errorMessage := url.QueryEscape("Les champs des caractéristiques d'une question ne peuvent pas être vide.")
			http.Redirect(w, r, data.ErrorMessageURL+"?errormessage="+errorMessage, http.StatusSeeOther)
			return
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
	RenderEditFormQuestionPage(w, dataPage)
}

func EditQuestionHandler(w http.ResponseWriter, r *http.Request, queries *db.Queries) {
	userID, _, ok := tools.CheckRequest(w, r, http.MethodPost)
	if !ok {
		log.Println("From EditQuestionHandler -> tools.CheckRequest return not ok")
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form", http.StatusBadRequest)
		return
	}
	content := strings.TrimSpace(r.FormValue("content"))
	if content == "" {
		http.Error(w, "Missing question content", http.StatusBadRequest)
		return
	}

	questionIDStr := r.FormValue("question_id")
	if questionIDStr == "" {
		log.Println("From EditQuestionHandler, no question id")
		http.Error(w, "Something went wrong !", http.StatusBadRequest)
		return
	}
	questionID, err := strconv.ParseInt(questionIDStr, 10, 64)
	if err != nil {
		log.Printf("From EditQuestionHandler -> strconv.ParseInt, invalid question id, error : %v", err)
		http.Error(w, "Something went wrong !", http.StatusBadRequest)
		return
	}

	intIDs := make(map[string]int64, 6)
	for _, feature := range []string{"subjectID", "themeID", "yearLevelID", "skillID", "difficultyID", "pointID"} {
		intID, ok := tools.StrToInt(r.FormValue(feature))
		if !ok {
			log.Printf("From EditQuestionHandler -> tools.StrToInt return not ok, some feature missing")
			http.Error(w, "Something went wrong !", http.StatusInternalServerError)
			return
		}
		intIDs[feature] = intID
	}

	rows, err := queries.UpdateQuestion(r.Context(), db.UpdateQuestionParams{
		SubjectID:    intIDs["subjectID"],
		ThemeID:      intIDs["themeID"],
		YearLevelID:  intIDs["yearLevelID"],
		SkillID:      intIDs["skillID"],
		DifficultyID: intIDs["difficultyID"],
		PointID:      intIDs["pointID"],
		Content:      content,
		ID:           questionID,
		UserID:       userID,
	})
	if err != nil {
		log.Printf("From EditQuestionHandler -> UpdateQuestion DB error: %v", err)
		errorMessage := url.QueryEscape("Il ne peut pas exister deux fois la même question ou la question ne peut être vide.")
		http.Redirect(w, r, data.ErrorMessageURL+"?errormessage="+errorMessage, http.StatusSeeOther)
		return
	}
	if !tools.HandleOwnedMutationRows(w, rows, "UpdateQuestion") {
		return
	}

	http.Redirect(w, r, data.DefaultDashboardRoutes.QuestionsURL, http.StatusSeeOther)
}

func DeleteFormQuestionHandler(w http.ResponseWriter, r *http.Request, queries *db.Queries) {
	userID, _, ok := tools.CheckRequest(w, r, http.MethodGet)
	if !ok {
		log.Println("From DeleteFormQuestionHandler -> tools.CheckRequest return not ok")
		return
	}

	questionIDStr := r.URL.Query().Get("question_id")
	if questionIDStr == "" {
		log.Println("From DeleteFormQuestionHandler : no question id parameter")
		http.Error(w, "Something went wrong !", http.StatusBadRequest)
		return
	}

	questionID, err := strconv.ParseInt(questionIDStr, 10, 64)
	if err != nil {
		log.Printf("From DeleteFormQuestionHandler -> strconv.ParseInt: invalid question id parameter, error : %v", err)
		http.Error(w, "Something went wrong !", http.StatusBadRequest)
		return
	}

	question, err := queries.GetQuestionByID(r.Context(), db.GetQuestionByIDParams{
		ID:     questionID,
		UserID: userID,
	})
	if err != nil {
		tools.HandleOwnedLookupError(w, err, "DeleteFormQuestionHandler GetQuestionByID")
		return
	}

	dataPage := data.QuestionPageData{
		Routes:         data.DefaultDashboardRoutes,
		QuestionRoutes: data.DefaultQuestionRoutes,
		PageTitle:      "delete question",
		ExtraData: map[string]any{
			"Question":   question,
			"QuestionID": questionIDStr,
			"CancelURL":  data.DefaultDashboardRoutes.QuestionsURL,
		},
	}

	RenderDeleteFormQuestionPage(w, dataPage)
}

func DeleteQuestionHandler(w http.ResponseWriter, r *http.Request, queries *db.Queries) {
	userID, _, ok := tools.CheckRequest(w, r, http.MethodPost)
	if !ok {
		log.Println("From DeleteQuestionHandler -> tools.CheckRequest return not ok")
		return
	}

	questionIDStr := r.FormValue("question_id")
	if questionIDStr == "" {
		log.Println("From DeleteQuestionHandler : no question id parameter")
		http.Error(w, "Something went wrong !", http.StatusBadRequest)
		return
	}

	questionID, err := strconv.ParseInt(questionIDStr, 10, 64)
	if err != nil {
		log.Printf("From DeleteQuestionHandler -> strconv.ParseInt, invalid question id, error : %v", err)
		http.Error(w, "Something went wrong !", http.StatusBadRequest)
		return
	}
	if _, err := queries.GetQuestionByID(r.Context(), db.GetQuestionByIDParams{
		ID:     questionID,
		UserID: userID,
	}); err != nil {
		tools.HandleOwnedLookupError(w, err, "DeleteQuestionHandler GetQuestionByID")
		return
	}

	altQuestionsIDWithImage, err := queries.GetAltQuestionIDsWithImage(r.Context(), db.GetAltQuestionIDsWithImageParams{
		QuestionID: questionID,
		UserID:     userID,
	})
	if err != nil {
		log.Printf("From DeleteQuestionHandler -> GetAltQuestionIDsWithImage DB error: %v", err)
		http.Error(w, "DB error", http.StatusInternalServerError)
		return
	}

	imageNames := make([]string, 0, len(altQuestionsIDWithImage)+1)
	for _, altQuestionID := range altQuestionsIDWithImage {
		image, err := queries.GetAltImageByAltQuestionID(r.Context(), db.GetAltImageByAltQuestionIDParams{
			AltQuestionID: altQuestionID,
			UserID:        userID,
			QuestionID:    questionID,
		})
		if err != nil {
			log.Printf("From DeleteQuestionHandler -> GetAltImageByAltQuestionID DB error: %v", err)
			http.Error(w, "DB error", http.StatusInternalServerError)
			return
		}
		imageNames = append(imageNames, image.ImageName)
	}

	image, err := queries.GetImageByQuestionID(r.Context(), db.GetImageByQuestionIDParams{
		QuestionID: questionID,
		UserID:     userID,
	})
	if err == nil {
		imageNames = append(imageNames, image.ImageName)
	} else if err != sql.ErrNoRows {
		log.Printf("From DeleteQuestionHandler -> GetImageByQuestionID DB error: %v", err)
		http.Error(w, "DB error", http.StatusInternalServerError)
		return
	}

	rows, err := queries.DeleteQuestion(r.Context(), db.DeleteQuestionParams{
		ID:     questionID,
		UserID: userID,
	})
	if err != nil {
		log.Printf("From DeleteQuestionHandler -> DeleteQuestion DB error: %v", err)
		errorMessage := url.QueryEscape("La question est utilisée par un qcm. Il n'est pas possible de la supprimer.")
		http.Redirect(w, r, data.ErrorMessageURL+"?errormessage="+errorMessage, http.StatusSeeOther)
		return
	}
	if rows == 0 {
		log.Printf("From DeleteQuestionHandler -> DeleteQuestion affected no rows for question %d and user %d", questionID, userID)
		http.Error(w, "Question not found", http.StatusNotFound)
		return
	}
	if rows > 1 {
		log.Printf("From DeleteQuestionHandler -> DeleteQuestion affected %d rows for question %d and user %d", rows, questionID, userID)
		http.Error(w, "DB integrity error", http.StatusInternalServerError)
		return
	}
	for _, imageName := range imageNames {
		if err := removeStoredImageFile(imageName); err != nil {
			log.Printf("From DeleteQuestionHandler -> RemoveStoredImageFile %s: %v", imageName, err)
		}
	}

	http.Redirect(w, r, data.DefaultDashboardRoutes.QuestionsURL, http.StatusSeeOther)
}
