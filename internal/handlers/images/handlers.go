package images

import (
	"database/sql"
	"log"
	"math"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"

	"github.com/grapinou/LazyMarking/internal/config"
	"github.com/grapinou/LazyMarking/internal/db"
	"github.com/grapinou/LazyMarking/internal/handlers/tools"
	"github.com/grapinou/LazyMarking/internal/templates/data"
)

func TableImageHandler(w http.ResponseWriter, r *http.Request, queries *db.Queries) {
	userID, _, ok := tools.CheckRequest(w, r, http.MethodGet)
	if !ok {
		log.Println("From TableImageHandler -> tools.CheckRequest return not ok")
		return
	}

	questionIDStr := r.URL.Query().Get("question_id")

	if questionIDStr == "" {
		log.Println("From TableImageHandler : no question id parameter")
		http.Error(w, "Something went wrong !", http.StatusBadRequest)
		return
	}

	questionID, err := strconv.ParseInt(questionIDStr, 10, 64)
	if err != nil {
		log.Printf("From TableImageHandler -> strconv.ParseInt, invalid question id parameter, error : %v", err)
		http.Error(w, "Something went wrong !", http.StatusBadRequest)
		return
	}

	question, err := queries.GetQuestionByID(r.Context(), db.GetQuestionByIDParams{
		ID:     questionID,
		UserID: userID,
	})
	if err != nil {
		log.Printf("From TableImageHandler -> GetQuestionByID DB error: %v", err)
		http.Error(w, "DB Error", http.StatusInternalServerError)
		return
	}

	image, err := queries.GetImageByQuestionID(r.Context(), db.GetImageByQuestionIDParams{
		QuestionID: questionID,
		UserID:     userID,
	})

	noImage := false
	if err == sql.ErrNoRows {
		noImage = true
	} else if err != nil {
		log.Printf("From TableImageHandler -> GetImageByQuestionID DB error: %v", err)
		http.Error(w, "DB Error", http.StatusInternalServerError)
		return
	}

	var editURL string
	var deleteURL string
	if !noImage {
		params := "?question_id=" + url.QueryEscape(strconv.FormatInt(questionID, 10))
		editURL = data.DefaultImageRoutes.EditURL + params
		deleteURL = data.DefaultImageRoutes.DeleteURL + params
	}

	addURL := data.DefaultImageRoutes.AddURL + "?question_id=" + url.QueryEscape(questionIDStr)
	dataPage := data.ImagePageData{
		Routes:      data.DefaultDashboardRoutes,
		ImageRoutes: data.DefaultImageRoutes,
		PageTitle:   "image",
		ExtraData: map[string]any{
			"UserID":             userID,
			"QuestionContent":    question.Content,
			"NoImage":            noImage,
			"Image":              image,
			"AddURL":             addURL,
			"EditURL":            editURL,
			"DeleteURL":          deleteURL,
			"PublicImageBaseURL": config.PublicImageBaseURL,
		},
	}

	RenderTableImagePage(w, dataPage)
}

func AddFormImageHandler(w http.ResponseWriter, r *http.Request, queries *db.Queries) {
	_, _, ok := tools.CheckRequest(w, r, http.MethodGet)
	if !ok {
		log.Println("From AddFormImageHandler -> tools.CheckRequest return not ok")
		return
	}

	questionIDStr := r.URL.Query().Get("question_id")
	if questionIDStr == "" {
		log.Println("From AddFormImageHandler : no question id parameter")
		http.Error(w, "Something went wrong !", http.StatusBadRequest)
		return
	}

	dataPage := data.ImagePageData{
		Routes:      data.DefaultDashboardRoutes,
		ImageRoutes: data.DefaultImageRoutes,
		PageTitle:   "add image",
		ExtraData: map[string]any{
			"QuestionID": questionIDStr,
		},
	}
	RenderAddFormImagePage(w, dataPage)
}

func AddImageHandler(w http.ResponseWriter, r *http.Request, queries *db.Queries) {
	userID, username, ok := tools.CheckRequest(w, r, http.MethodPost)
	if !ok {
		log.Println("From AddImageHandler -> tools.CheckRequest return not ok")
		return
	}

	file, header, imageConfig, err := tools.CheckImageFile(w, r)
	if err != nil {
		log.Printf("From AddImageHandler -> CheckImageFile: %v", err)
		http.Error(w, "Something went wrong !", http.StatusBadRequest)
		return
	}
	defer file.Close()
	defer func() {
		if err := r.MultipartForm.RemoveAll(); err != nil {
			log.Printf("From AddImageHandler -> cleanup multipart files: %v", err)
		}
	}()

	questionIDStr := r.FormValue("question_id")
	if questionIDStr == "" {
		log.Println("From AddImageHandler : no question id parameter")
		http.Error(w, "Something went wrong !", http.StatusBadRequest)
		return
	}
	questionID, err := strconv.ParseInt(questionIDStr, 10, 64)
	if err != nil {
		log.Printf("From AddImageHandler -> strconv.ParseInt, invalid question ID, error : %v", err)
		http.Error(w, "Something went wrong !", http.StatusBadRequest)
		return
	}

	widthStr := r.FormValue("width")
	widthFloat, err := strconv.ParseFloat(widthStr, 64)
	if err != nil {
		log.Printf("From AddImageHandler -> strconv.ParseFloat : invalid number : error : %v", err)
		errorMessage := url.QueryEscape("Assurez-vous de bien saisir un nombre entier supérieur à zéro.")
		http.Redirect(w, r, data.ErrorMessageURL+"?errormessage="+errorMessage, http.StatusSeeOther)
		return
	}
	if _, _, err := tools.ValidateImageResize(imageConfig.Width, imageConfig.Height, widthFloat); err != nil {
		log.Printf("From AddImageHandler -> ValidateImageResize: %v", err)
		errorMessage := url.QueryEscape("Assurez-vous de bien saisir un nombre entier supérieur à zéro.")
		http.Redirect(w, r, data.ErrorMessageURL+"?errormessage="+errorMessage, http.StatusSeeOther)
		return
	}
	resize := int64(math.Round(widthFloat))

	filename, err := tools.SanitizeFilename(userID, username, config.MainQuestion, questionID, header.Filename)
	if err != nil {
		log.Printf("From AddImageHandler -> SanitizeFilename: %v", err)
		errorMessage := url.QueryEscape("Assurez-vous de bien utiliser une image avec une extension authorisée.")
		http.Redirect(w, r, data.ErrorMessageURL+"?errormessage="+errorMessage, http.StatusSeeOther)
		return
	}

	err = tools.SaveUploadedFile(file, config.ImageSavePath, filename)
	if err != nil {
		log.Printf("From AddImageHandler -> SaveUploadedFile : %v, file : %s", err, filename)
		http.Error(w, "Something went wrong !", http.StatusInternalServerError)
		return
	}

	// on vérifie que l'image ne contient pas de points équivalent à ceux des réponses
	ok = tools.ImageCircleCheck(config.ImageSavePath, filename, widthFloat)
	if !ok {
		os.Remove(filepath.Join(config.ImageSavePath, filename))
		errorMessage := url.QueryEscape("L'image contient des cercles incompatibles avec la suites du traitement. Changer la taille de l'image ou prenez une image différente.")
		http.Redirect(w, r, data.ErrorMessageURL+"?errormessage="+errorMessage, http.StatusSeeOther)
		return
	}

	if err := queries.CreateImage(r.Context(), db.CreateImageParams{
		QuestionID:       questionID,
		ImageName:        filename,
		ResizePercentage: resize,
		UserID:           userID,
	}); err != nil {
		os.Remove(filepath.Join(config.ImageSavePath, filename))
		log.Printf("From AddImageHandler, CreateImage : DB error: %v", err)
		errorMessage := url.QueryEscape("Une question peut avoir qu'une seule image.")
		http.Redirect(w, r, data.ErrorMessageURL+"?errormessage="+errorMessage, http.StatusSeeOther)
		return
	}

	imageURL := data.DefaultQuestionRoutes.ImageURL + "?question_id=" + url.QueryEscape(questionIDStr)
	http.Redirect(w, r, imageURL, http.StatusSeeOther)
}

func EditFormImageHandler(w http.ResponseWriter, r *http.Request, queries *db.Queries) {
	userID, _, ok := tools.CheckRequest(w, r, http.MethodGet)
	if !ok {
		log.Println("From EditFormImageHandler -> tools.CheckRequest return not ok")
		return
	}

	questionIDStr := r.URL.Query().Get("question_id")
	if questionIDStr == "" {
		log.Println("From EditFormImageHandler : no question id parameter")
		http.Error(w, "Something went wrong !", http.StatusBadRequest)
		return
	}

	questionID, err := strconv.ParseInt(questionIDStr, 10, 64)
	if err != nil {
		log.Printf("From EditFormImageHandler -> strconv.ParseInt, invalid question ID, error : %v", err)
		http.Error(w, "Something went wrong !", http.StatusBadRequest)
		return
	}

	image, err := queries.GetImageByQuestionID(r.Context(), db.GetImageByQuestionIDParams{
		QuestionID: questionID,
		UserID:     userID,
	})
	if err != nil {
		log.Printf("From EditFormImageHandler -> GetImageByQuestionID : DB error : %v", err)
		http.Error(w, "Something went wrong !", http.StatusInternalServerError)
		return
	}

	dataPage := data.ImagePageData{
		Routes:      data.DefaultDashboardRoutes,
		ImageRoutes: data.DefaultImageRoutes,
		PageTitle:   "edit image",
		ExtraData: map[string]any{
			"ImageSize":  image.ResizePercentage,
			"QuestionID": questionIDStr,
		},
	}
	RenderEditFormImagePage(w, dataPage)
}

func EditImageHandler(w http.ResponseWriter, r *http.Request, queries *db.Queries) {
	userID, _, ok := tools.CheckRequest(w, r, http.MethodPost)
	if !ok {
		log.Println("From EditImageHandler -> tools.CheckRequest return not ok")
		return
	}

	questionIDStr := r.FormValue("question_id")
	if questionIDStr == "" {
		log.Println("From EditImageHandler : no question id parameter")
		http.Error(w, "Something went wrong !", http.StatusBadRequest)
		return
	}

	questionID, err := strconv.ParseInt(questionIDStr, 10, 64)
	if err != nil {
		log.Printf("From EditImageHandler -> strconv.ParseInt, invalid question ID, error : %v", err)
		http.Error(w, "Something went wrong !", http.StatusBadRequest)
		return
	}

	widthStr := r.FormValue("width")

	widthFloat, err := strconv.ParseFloat(widthStr, 64)
	if err != nil || widthFloat <= 0 {
		log.Printf("From EditImageHandler -> strconv.ParseFloat : invalid number : error : %v", err)
		errorMessage := url.QueryEscape("Assurez-vous de bien saisir un nombre entier supérieur à zéro.")
		http.Redirect(w, r, data.ErrorMessageURL+"?errormessage="+errorMessage, http.StatusSeeOther)
		return
	}

	// vérification que la taille de l'image ne contienne pas des cercles identiques à ceux des questions
	image, err := queries.GetImageByQuestionID(r.Context(), db.GetImageByQuestionIDParams{
		QuestionID: questionID,
		UserID:     userID,
	})
	if err != nil {

		log.Printf("From  EditImageHandler -> GetImageByQuestionID DB error: %v", err)
		http.Error(w, "Something went wrong !", http.StatusInternalServerError)
		return
	}
	imageConfig, err := tools.ReadImageConfig(filepath.Join(config.ImageSavePath, image.ImageName))
	if err != nil {
		log.Printf("From EditImageHandler -> ReadImageConfig: %v", err)
		http.Error(w, "Something went wrong !", http.StatusInternalServerError)
		return
	}
	if _, _, err := tools.ValidateImageResize(imageConfig.Width, imageConfig.Height, widthFloat); err != nil {
		log.Printf("From EditImageHandler -> ValidateImageResize: %v", err)
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

	if err := queries.UpdateSizeImage(r.Context(), db.UpdateSizeImageParams{
		ResizePercentage: resize,
		QuestionID:       questionID,
		UserID:           userID,
	}); err != nil {
		log.Printf("From  EditImageHandler : UpdateSizeImage DB error: %v", err)
		http.Error(w, "Something went wrong !", http.StatusInternalServerError)
		return
	}

	imageURL := data.DefaultQuestionRoutes.ImageURL + "?question_id=" + url.QueryEscape(questionIDStr)
	http.Redirect(w, r, imageURL, http.StatusSeeOther)
}

func DeleteFormImageHandler(w http.ResponseWriter, r *http.Request, queries *db.Queries) {
	_, _, ok := tools.CheckRequest(w, r, http.MethodGet)
	if !ok {
		log.Println("From DeleteFormImageHandler -> tools.CheckRequest return not ok")
		return
	}

	questionIDStr := r.URL.Query().Get("question_id")
	if questionIDStr == "" {
		log.Println("From  DeleteFormImageHandler : no question id parameter")
		http.Error(w, "Something went wrong !", http.StatusBadRequest)
		return
	}

	dataPage := data.ImagePageData{
		Routes:      data.DefaultDashboardRoutes,
		ImageRoutes: data.DefaultImageRoutes,
		PageTitle:   "delete image",
		ExtraData: map[string]any{
			"QuestionID": questionIDStr,
		},
	}

	RenderDeleteFormImagePage(w, dataPage)
}

func DeleteImageHandler(w http.ResponseWriter, r *http.Request, queries *db.Queries) {
	userID, _, ok := tools.CheckRequest(w, r, http.MethodPost)
	if !ok {
		log.Println("From DeleteImageHandler -> tools.CheckRequest return not ok")
		return
	}

	questionIDStr := r.FormValue("question_id")
	if questionIDStr == "" {
		log.Println("From DeleteImageHandler : no question id parameter")
		http.Error(w, "Something went wrong !", http.StatusBadRequest)
		return
	}

	questionID, err := strconv.ParseInt(questionIDStr, 10, 64)
	if err != nil {
		log.Printf("From DeleteImageHandler -> strconv.ParseInt : invalid question ID, error : %v", err)
		http.Error(w, "Something went wrong !", http.StatusBadRequest)
		return
	}

	image, err := queries.GetImageByQuestionID(r.Context(), db.GetImageByQuestionIDParams{
		QuestionID: questionID,
		UserID:     userID,
	})
	if err != nil {
		log.Printf("From DeleteImageHandler -> GetImageByQuestionID DB error: %v", err)
		http.Error(w, "DB error", http.StatusInternalServerError)
		return
	}

	rows, err := queries.DeleteImage(r.Context(), db.DeleteImageParams{
		QuestionID: questionID,
		UserID:     userID,
	})
	if err != nil {
		log.Printf("From DeleteImageHandler -> DeleteImage DB error: %v", err)
		http.Error(w, "DB error", http.StatusInternalServerError)
		return
	}
	if rows == 0 {
		log.Printf("From DeleteImageHandler -> DeleteImage affected no rows for question %d and user %d", questionID, userID)
		http.Error(w, "Image not found", http.StatusNotFound)
		return
	}
	if rows > 1 {
		log.Printf("From DeleteImageHandler -> DeleteImage affected %d rows for question %d and user %d", rows, questionID, userID)
		http.Error(w, "DB integrity error", http.StatusInternalServerError)
		return
	}
	if err := tools.RemoveStoredImageFile(image.ImageName); err != nil {
		log.Printf("From DeleteImageHandler -> RemoveStoredImageFile : %v", err)
	}

	imageURL := data.DefaultQuestionRoutes.ImageURL + "?question_id=" + url.QueryEscape(questionIDStr)
	http.Redirect(w, r, imageURL, http.StatusSeeOther)
}
