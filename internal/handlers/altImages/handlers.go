package altimages

import (
	"database/sql"
	"log"
	"math"
	"net/http"
	"net/url"
	"path/filepath"
	"strconv"

	"github.com/grapinou/LazyMarking/internal/config"
	"github.com/grapinou/LazyMarking/internal/db"
	"github.com/grapinou/LazyMarking/internal/handlers/tools"
	"github.com/grapinou/LazyMarking/internal/templates/data"
)

var removeStoredImageFile = tools.RemoveStoredImageFile
var checkUploadedImageCircles = tools.ImageCircleCheck

func TableAltImageHandler(w http.ResponseWriter, r *http.Request, queries *db.Queries) {
	userID, _, ok := tools.CheckRequest(w, r, http.MethodGet)
	if !ok {
		log.Println("From TableAltImageHandler -> tools.CheckRequest return not ok")
		return
	}

	questionIDStr := r.URL.Query().Get("question_id")
	if questionIDStr == "" {
		log.Println("From TableAltImageHandler : no question id parameter")
		http.Error(w, "Something went wrong !", http.StatusBadRequest)
		return
	}

	altQuestionIDStr := r.URL.Query().Get("alt_question_id")
	if altQuestionIDStr == "" {
		log.Println("From TableAltImageHandler : no alt question id parameter")
		http.Error(w, "Something went wrong !", http.StatusBadRequest)
		return
	}

	altQuestionID, err := strconv.ParseInt(altQuestionIDStr, 10, 64)
	if err != nil {
		log.Printf("From TableAltImageHandler -> strconv.ParseInt, invalid alt question id parameter, error : %v", err)
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
		tools.HandleOwnedLookupError(w, err, "TableAltImageHandler GetAltQuestionByParentID")
		return
	}
	question, err := queries.GetQuestionByID(r.Context(), db.GetQuestionByIDParams{
		ID:     questionID,
		UserID: userID,
	})
	if err != nil {
		tools.HandleOwnedLookupError(w, err, "TableAltImageHandler GetQuestionByID")
		return
	}

	altImage, err := queries.GetAltImageByAltQuestionID(r.Context(), db.GetAltImageByAltQuestionIDParams{
		AltQuestionID: altQuestionID,
		UserID:        userID,
		QuestionID:    questionID,
	})

	noAltImage := false
	if err == sql.ErrNoRows {
		noAltImage = true
	} else if err != nil {
		log.Printf("From TableAltImageHandler -> GetAltImageByAltQuestionID DB error: %v", err)
		http.Error(w, "DB Error", http.StatusInternalServerError)
		return
	}

	var editURL string
	var deleteURL string
	if !noAltImage {
		editURL = data.VariantURL(data.DefaultAltImageRoutes.EditURL, questionID, altQuestionID)
		deleteURL = data.VariantURL(data.DefaultAltImageRoutes.DeleteURL, questionID, altQuestionID)
	}

	addURL := data.VariantURL(data.DefaultAltImageRoutes.AddURL, questionID, altQuestionID)
	altQuestionsURL := data.QuestionURL(data.DefaultQuestionRoutes.AltQuestionsURL, questionID)

	dataPage := data.AltImagePageData{
		Routes:          data.DefaultDashboardRoutes,
		AltImageRoutes:  data.DefaultAltImageRoutes,
		QuestionContext: data.QuestionContext{ID: question.ID, Content: question.Content},
		VariantContext:  data.VariantContext{ID: altQuestion.ID, Content: altQuestion.Content},
		PageTitle:       "alt image",
		ExtraData: map[string]any{
			"UserID":             userID,
			"NoAltImage":         noAltImage,
			"AltImage":           altImage,
			"AltQuestionURL":     altQuestionsURL,
			"AddURL":             addURL,
			"EditURL":            editURL,
			"DeleteURL":          deleteURL,
			"PublicImageBaseURL": config.PublicImageBaseURL,
		},
	}

	RenderTableAltImagePage(w, dataPage)
}

func AddFormAltImageHandler(w http.ResponseWriter, r *http.Request, queries *db.Queries) {
	userID, _, ok := tools.CheckRequest(w, r, http.MethodGet)
	if !ok {
		log.Println("From AddFormAltImageHandler -> tools.CheckRequest return not ok")
		return
	}

	questionIDStr := r.URL.Query().Get("question_id")
	if questionIDStr == "" {
		log.Println("From AddFormAltImageHandler : no question id parameter")
		http.Error(w, "Something went wrong !", http.StatusBadRequest)
		return
	}
	altQuestionIDStr := r.URL.Query().Get("alt_question_id")
	if altQuestionIDStr == "" {
		log.Println("From AddFormAltImageHandler : no alt question id parameter")
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
		tools.HandleOwnedLookupError(w, err, "AddFormAltImageHandler GetAltQuestionByParentID")
		return
	}
	question, err := queries.GetQuestionByID(r.Context(), db.GetQuestionByIDParams{ID: questionID, UserID: userID})
	if err != nil {
		tools.HandleOwnedLookupError(w, err, "AddFormAltImageHandler GetQuestionByID")
		return
	}

	dataPage := data.AltImagePageData{
		Routes:          data.DefaultDashboardRoutes,
		AltImageRoutes:  data.DefaultAltImageRoutes,
		QuestionContext: data.QuestionContext{ID: question.ID, Content: question.Content},
		VariantContext:  data.VariantContext{ID: altQuestion.ID, Content: altQuestion.Content},
		PageTitle:       "add alt image",
		ExtraData: map[string]any{
			"CancelURL": data.VariantURL(data.DefaultAltQuestionRoutes.AltImageURL, questionID, altQuestionID),
		},
	}
	RenderAddFormAltImagePage(w, dataPage)
}

func AddAltImageHandler(w http.ResponseWriter, r *http.Request, queries *db.Queries) {
	userID, username, ok := tools.CheckRequest(w, r, http.MethodPost)
	if !ok {
		log.Println("From AddAltImageHandler -> tools.CheckRequest return not ok")
		return
	}

	file, header, imageConfig, err := tools.CheckImageFile(w, r)
	if err != nil {
		log.Printf("From AddAltImageHandler -> CheckImageFile: %v", err)
		http.Error(w, "Something went wrong !", http.StatusBadRequest)
		return
	}
	defer file.Close()
	defer func() {
		if err := r.MultipartForm.RemoveAll(); err != nil {
			log.Printf("From AddAltImageHandler -> cleanup multipart files: %v", err)
		}
	}()

	questionIDStr := r.FormValue("question_id")
	if questionIDStr == "" {
		log.Println("From AddAltImageHandler : no question id parameter")
		http.Error(w, "Something went wrong !", http.StatusBadRequest)
		return
	}

	altQuestionIDStr := r.FormValue("alt_question_id")
	if altQuestionIDStr == "" {
		log.Println("From AddAltImageHandler : no alt question id parameter")
		http.Error(w, "Something went wrong !", http.StatusBadRequest)
		return
	}

	altQuestionID, err := strconv.ParseInt(altQuestionIDStr, 10, 64)
	if err != nil {
		log.Printf("From AddAltImageHandler -> strconv.ParseInt, invalid alt question ID, error : %v", err)
		http.Error(w, "Something went wrong !", http.StatusBadRequest)
		return
	}
	questionID, err := strconv.ParseInt(questionIDStr, 10, 64)
	if err != nil {
		http.Error(w, "Something went wrong !", http.StatusBadRequest)
		return
	}
	if _, err := queries.GetAltQuestionByParentID(r.Context(), db.GetAltQuestionByParentIDParams{
		ID:         altQuestionID,
		QuestionID: questionID,
		UserID:     userID,
	}); err != nil {
		tools.HandleOwnedLookupError(w, err, "AddAltImageHandler GetAltQuestionByParentID")
		return
	}

	widthStr := r.FormValue("width")
	widthFloat, err := strconv.ParseFloat(widthStr, 64)
	if err != nil {
		log.Printf("From AddAltImageHandler -> strconv.ParseFloat : invalid number : error : %v", err)
		errorMessage := url.QueryEscape("Assurez-vous de bien saisir un nombre entier supérieur à zéro.")
		http.Redirect(w, r, data.ErrorMessageURL+"?errormessage="+errorMessage, http.StatusSeeOther)
		return
	}
	if _, _, err := tools.ValidateImageResize(imageConfig.Width, imageConfig.Height, widthFloat); err != nil {
		log.Printf("From AddAltImageHandler -> ValidateImageResize: %v", err)
		errorMessage := url.QueryEscape("Assurez-vous de bien saisir un nombre entier supérieur à zéro.")
		http.Redirect(w, r, data.ErrorMessageURL+"?errormessage="+errorMessage, http.StatusSeeOther)
		return
	}
	resize := int64(math.Round(widthFloat))

	filename, err := tools.SanitizeFilename(userID, username, config.AltQuestion, altQuestionID, header.Filename)
	if err != nil {
		log.Printf("From AddAltImageHandler -> SanitizeFilename: %v", err)
		errorMessage := url.QueryEscape("Assurez-vous de bien utiliser une image avec une extension authorisée.")
		http.Redirect(w, r, data.ErrorMessageURL+"?errormessage="+errorMessage, http.StatusSeeOther)
		return
	}

	err = tools.SaveUploadedFile(file, config.ImageSavePath, filename)
	if err != nil {
		log.Printf("From AddAltImageHandler -> SaveUploadedFile: %v, filename : %s", err, filename)
		http.Error(w, "Something went wrong !", http.StatusInternalServerError)
		return
	}

	// on vérifie que l'image ne contient pas de points équivalent à ceux des réponses
	ok = checkUploadedImageCircles(config.ImageSavePath, filename, widthFloat)
	if !ok {
		if err := tools.RemoveStoredImageFile(filename); err != nil {
			log.Printf("From AddAltImageHandler -> cleanup rejected image: %v", err)
		}
		errorMessage := url.QueryEscape("L'image contient des cercles incompatibles avec la suites du traitement. Changer la taille de l'image ou prenez une image différente.")
		http.Redirect(w, r, data.ErrorMessageURL+"?errormessage="+errorMessage, http.StatusSeeOther)
		return
	}

	rows, err := queries.CreateAltImage(r.Context(), db.CreateAltImageParams{
		AltQuestionID:    altQuestionID,
		ImageName:        filename,
		ResizePercentage: resize,
		UserID:           userID,
		QuestionID:       questionID,
	})
	if err != nil {

		if cleanupErr := tools.RemoveStoredImageFile(filename); cleanupErr != nil {
			log.Printf("From AddAltImageHandler -> cleanup failed DB insert: %v", cleanupErr)
		}
		log.Printf("From AddAltImageHandler, CreateAltImage : DB error: %v", err)
		errorMessage := url.QueryEscape("Une alt question peut avoir qu'une seule image.")
		http.Redirect(w, r, data.ErrorMessageURL+"?errormessage="+errorMessage, http.StatusSeeOther)
		return
	}
	if !tools.HandleOwnedMutationRows(w, rows, "CreateAltImage") {
		if err := tools.RemoveStoredImageFile(filename); err != nil {
			log.Printf("From AddAltImageHandler -> cleanup rejected image: %v", err)
		}
		return
	}

	altImageURL := data.VariantURL(data.DefaultAltQuestionRoutes.AltImageURL, questionID, altQuestionID)
	http.Redirect(w, r, altImageURL, http.StatusSeeOther)
}

func EditFormAltImageHandler(w http.ResponseWriter, r *http.Request, queries *db.Queries) {
	userID, _, ok := tools.CheckRequest(w, r, http.MethodGet)
	if !ok {
		log.Println("From EditFormAltImageHandler -> tools.CheckRequest return not ok")
		return
	}

	questionIDStr := r.URL.Query().Get("question_id")
	if questionIDStr == "" {
		log.Println("From EditFormAltImageHandler : no question id parameter")
		http.Error(w, "Something went wrong !", http.StatusBadRequest)
		return
	}

	altQuestionIDStr := r.URL.Query().Get("alt_question_id")
	if altQuestionIDStr == "" {
		log.Println("From EditFormAltImageHandler : no alt question id parameter")
		http.Error(w, "Something went wrong !", http.StatusBadRequest)
		return
	}

	altQuestionID, err := strconv.ParseInt(altQuestionIDStr, 10, 64)
	if err != nil {
		log.Printf("From EditFormAltImageHandler -> strconv.ParseInt, invalid alt question ID, error : %v", err)
		http.Error(w, "Something went wrong !", http.StatusBadRequest)
		return
	}
	questionID, err := strconv.ParseInt(questionIDStr, 10, 64)
	if err != nil {
		http.Error(w, "Something went wrong !", http.StatusBadRequest)
		return
	}
	altQuestion, err := queries.GetAltQuestionByParentID(r.Context(), db.GetAltQuestionByParentIDParams{ID: altQuestionID, QuestionID: questionID, UserID: userID})
	if err != nil {
		tools.HandleOwnedLookupError(w, err, "EditFormAltImageHandler GetAltQuestionByParentID")
		return
	}
	question, err := queries.GetQuestionByID(r.Context(), db.GetQuestionByIDParams{ID: questionID, UserID: userID})
	if err != nil {
		tools.HandleOwnedLookupError(w, err, "EditFormAltImageHandler GetQuestionByID")
		return
	}

	altImage, err := queries.GetAltImageByAltQuestionID(r.Context(), db.GetAltImageByAltQuestionIDParams{
		AltQuestionID: altQuestionID,
		UserID:        userID,
		QuestionID:    questionID,
	})
	if err != nil {
		log.Printf("From EditFormAltImageHandler -> GetAltImageByAltQuestionID : DB error : %v", err)
		http.Error(w, "Something went wrong !", http.StatusInternalServerError)
		return
	}

	dataPage := data.AltImagePageData{
		Routes:          data.DefaultDashboardRoutes,
		AltImageRoutes:  data.DefaultAltImageRoutes,
		QuestionContext: data.QuestionContext{ID: question.ID, Content: question.Content},
		VariantContext:  data.VariantContext{ID: altQuestion.ID, Content: altQuestion.Content},
		PageTitle:       "edit alt image",
		ExtraData: map[string]any{
			"AltImage":           altImage,
			"ImageSize":          altImage.ResizePercentage,
			"PublicImageBaseURL": config.PublicImageBaseURL,
			"CancelURL":          data.VariantURL(data.DefaultAltQuestionRoutes.AltImageURL, questionID, altQuestionID),
		},
	}
	RenderEditFormAltImagePage(w, dataPage)
}

func EditAltImageHandler(w http.ResponseWriter, r *http.Request, queries *db.Queries) {
	userID, _, ok := tools.CheckRequest(w, r, http.MethodPost)
	if !ok {
		log.Println("From EditAltImageHandler -> tools.CheckRequest return not ok")
		return
	}

	questionIDStr := r.FormValue("question_id")
	if questionIDStr == "" {
		log.Println("From EditAltImageHandler : no question id parameter")
		http.Error(w, "Something went wrong !", http.StatusBadRequest)
		return
	}

	altQuestionIDStr := r.FormValue("alt_question_id")
	if questionIDStr == "" {
		log.Println("From EditAltImageHandler : no alt question id parameter")
		http.Error(w, "Something went wrong !", http.StatusBadRequest)
		return
	}

	altQuestionID, err := strconv.ParseInt(altQuestionIDStr, 10, 64)
	if err != nil {
		log.Printf("From EditAltImageHandler -> strconv.ParseInt, invalid alt question ID, error : %v", err)
		http.Error(w, "Something went wrong !", http.StatusBadRequest)
		return
	}
	questionID, err := strconv.ParseInt(questionIDStr, 10, 64)
	if err != nil {
		http.Error(w, "Something went wrong !", http.StatusBadRequest)
		return
	}

	widthStr := r.FormValue("width")
	widthFloat, err := strconv.ParseFloat(widthStr, 64)
	if err != nil || widthFloat <= 0 {
		log.Printf("From EditAltImageHandler -> strconv.ParseFloat : invalid number : error : %v", err)
		errorMessage := url.QueryEscape("Assurez-vous de bien saisir un nombre entier supérieur à zéro.")
		http.Redirect(w, r, data.ErrorMessageURL+"?errormessage="+errorMessage, http.StatusSeeOther)
		return
	}

	// vérification que la taille de l'image ne contienne pas des cercles identiques à ceux des questions
	image, err := queries.GetAltImageByAltQuestionID(r.Context(), db.GetAltImageByAltQuestionIDParams{
		AltQuestionID: altQuestionID,
		UserID:        userID,
		QuestionID:    questionID,
	})
	if err != nil {
		tools.HandleOwnedLookupError(w, err, "EditAltImageHandler GetAltImageByAltQuestionID")
		return
	}
	imageConfig, err := tools.ReadImageConfig(filepath.Join(config.ImageSavePath, image.ImageName))
	if err != nil {
		log.Printf("From EditAltImageHandler -> ReadImageConfig: %v", err)
		http.Error(w, "Something went wrong !", http.StatusInternalServerError)
		return
	}
	if _, _, err := tools.ValidateImageResize(imageConfig.Width, imageConfig.Height, widthFloat); err != nil {
		log.Printf("From EditAltImageHandler -> ValidateImageResize: %v", err)
		errorMessage := url.QueryEscape("Assurez-vous de bien saisir un nombre entier supérieur à zéro.")
		http.Redirect(w, r, data.ErrorMessageURL+"?errormessage="+errorMessage, http.StatusSeeOther)
		return
	}
	ok = tools.ImageCircleCheck(config.ImageSavePath, image.ImageName, widthFloat)
	if !ok {

		errorMessage := url.QueryEscape("Le redimensionnement de l'image n'est pas compatible avec la détection des réponses. Changer de taille.")
		http.Redirect(w, r, data.ErrorMessageURL+"?errormessage="+errorMessage, http.StatusSeeOther)
		return

	}

	resize := int64(math.Round(widthFloat))

	rows, err := queries.UpdateSizeAltImage(r.Context(), db.UpdateSizeAltImageParams{
		ResizePercentage: resize,
		AltQuestionID:    altQuestionID,
		UserID:           userID,
		QuestionID:       questionID,
	})
	if err != nil {
		log.Printf("From  EditAltImageHandler : UpdateSizeAltImage DB error: %v", err)
		http.Error(w, "Something went wrong !", http.StatusInternalServerError)
		return
	}
	if !tools.HandleOwnedMutationRows(w, rows, "UpdateSizeAltImage") {
		return
	}

	altImageURL := data.VariantURL(data.DefaultAltQuestionRoutes.AltImageURL, questionID, altQuestionID)
	http.Redirect(w, r, altImageURL, http.StatusSeeOther)
}

func DeleteFormAltImageHandler(w http.ResponseWriter, r *http.Request, queries *db.Queries) {
	userID, _, ok := tools.CheckRequest(w, r, http.MethodGet)
	if !ok {
		log.Println("From DeleteFormAltImageHandler -> tools.CheckRequest return not ok")
		return
	}

	questionIDStr := r.URL.Query().Get("question_id")
	if questionIDStr == "" {
		log.Println("From  DeleteFormAltImageHandler : no question id parameter")
		http.Error(w, "Something went wrong !", http.StatusBadRequest)
		return
	}

	altQuestionIDStr := r.URL.Query().Get("alt_question_id")
	if altQuestionIDStr == "" {
		log.Println("From  DeleteFormAltImageHandler : no alt question id parameter")
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
		tools.HandleOwnedLookupError(w, err, "DeleteFormAltImageHandler GetAltQuestionByParentID")
		return
	}
	question, err := queries.GetQuestionByID(r.Context(), db.GetQuestionByIDParams{ID: questionID, UserID: userID})
	if err != nil {
		tools.HandleOwnedLookupError(w, err, "DeleteFormAltImageHandler GetQuestionByID")
		return
	}
	altImage, err := queries.GetAltImageByAltQuestionID(r.Context(), db.GetAltImageByAltQuestionIDParams{
		AltQuestionID: altQuestionID,
		UserID:        userID,
		QuestionID:    questionID,
	})
	if err != nil {
		tools.HandleOwnedLookupError(w, err, "DeleteFormAltImageHandler GetAltImageByAltQuestionID")
		return
	}

	dataPage := data.AltImagePageData{
		Routes:          data.DefaultDashboardRoutes,
		AltImageRoutes:  data.DefaultAltImageRoutes,
		QuestionContext: data.QuestionContext{ID: question.ID, Content: question.Content},
		VariantContext:  data.VariantContext{ID: altQuestion.ID, Content: altQuestion.Content},
		PageTitle:       "delete alt image",
		ExtraData: map[string]any{
			"AltImage":           altImage,
			"PublicImageBaseURL": config.PublicImageBaseURL,
			"CancelURL":          data.VariantURL(data.DefaultAltQuestionRoutes.AltImageURL, questionID, altQuestionID),
		},
	}

	RenderDeleteFormAltImagePage(w, dataPage)
}

func DeleteAltImageHandler(w http.ResponseWriter, r *http.Request, queries *db.Queries) {
	userID, _, ok := tools.CheckRequest(w, r, http.MethodPost)
	if !ok {
		log.Println("From DeleteAltImageHandler -> tools.CheckRequest return not ok")
		return
	}

	questionIDStr := r.FormValue("question_id")
	if questionIDStr == "" {
		log.Println("From DeleteAltImageHandler : no question id parameter")
		http.Error(w, "Something went wrong !", http.StatusBadRequest)
		return
	}

	altQuestionIDStr := r.FormValue("alt_question_id")
	if altQuestionIDStr == "" {
		log.Println("From DeleteAltImageHandler : no alt question id parameter")
		http.Error(w, "Something went wrong !", http.StatusBadRequest)
		return
	}

	altQuestionID, err := strconv.ParseInt(altQuestionIDStr, 10, 64)
	if err != nil {
		log.Printf("From DeleteAltImageHandler -> strconv.ParseInt : invalid alt question ID, error : %v", err)
		http.Error(w, "Something went wrong !", http.StatusBadRequest)
		return
	}
	questionID, err := strconv.ParseInt(questionIDStr, 10, 64)
	if err != nil {
		http.Error(w, "Something went wrong !", http.StatusBadRequest)
		return
	}

	image, err := queries.GetAltImageByAltQuestionID(r.Context(), db.GetAltImageByAltQuestionIDParams{
		AltQuestionID: altQuestionID,
		UserID:        userID,
		QuestionID:    questionID,
	})
	if err != nil {
		tools.HandleOwnedLookupError(w, err, "DeleteAltImageHandler GetAltImageByAltQuestionID")
		return
	}

	rows, err := queries.DeleteAltImage(r.Context(), db.DeleteAltImageParams{
		AltQuestionID: altQuestionID,
		UserID:        userID,
		QuestionID:    questionID,
	})
	if err != nil {
		log.Printf("From DeleteAltImageHandler -> DeleteAltImage DB error: %v", err)
		http.Error(w, "DB error", http.StatusInternalServerError)
		return
	}
	if rows == 0 {
		log.Printf("From DeleteAltImageHandler -> DeleteAltImage affected no rows for alt question %d and user %d", altQuestionID, userID)
		http.Error(w, "Image not found", http.StatusNotFound)
		return
	}
	if rows > 1 {
		log.Printf("From DeleteAltImageHandler -> DeleteAltImage affected %d rows for alt question %d and user %d", rows, altQuestionID, userID)
		http.Error(w, "DB integrity error", http.StatusInternalServerError)
		return
	}
	if err := removeStoredImageFile(image.ImageName); err != nil {
		log.Printf("From DeleteAltImageHandler -> RemoveStoredImageFile %s: %v", image.ImageName, err)
	}

	altImageURL := data.VariantURL(data.DefaultAltQuestionRoutes.AltImageURL, questionID, altQuestionID)
	http.Redirect(w, r, altImageURL, http.StatusSeeOther)
}
