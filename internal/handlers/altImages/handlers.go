package altimages

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

	altQuestion, err := queries.GetAltQuestionByID(r.Context(), db.GetAltQuestionByIDParams{
		ID:     altQuestionID,
		UserID: userID,
	})
	if err != nil {
		log.Printf("From TableAltImageHandler -> GetAltQuestionByID DB error: %v", err)
		http.Error(w, "DB Error", http.StatusInternalServerError)
		return
	}

	altImage, err := queries.GetAltImageByAltQuestionID(r.Context(), db.GetAltImageByAltQuestionIDParams{
		AltQuestionID: altQuestionID,
		UserID:        userID,
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
		params := "?question_id=" + url.QueryEscape(questionIDStr) + "&alt_question_id=" + url.QueryEscape(strconv.FormatInt(altQuestionID, 10))
		editURL = data.DefaultAltImageRoutes.EditURL + params
		deleteURL = data.DefaultAltImageRoutes.DeleteURL + params
	}

	addURL := data.DefaultAltImageRoutes.AddURL + "?question_id=" + url.QueryEscape(questionIDStr) +
		"&alt_question_id=" + url.QueryEscape(altQuestionIDStr)
	altQuestionsURL := data.DefaultQuestionRoutes.AltQuestionsURL + "?question_id=" + url.QueryEscape(questionIDStr)

	dataPage := data.AltImagePageData{
		Routes:         data.DefaultDashboardRoutes,
		AltImageRoutes: data.DefaultAltImageRoutes,
		PageTitle:      "alt image",
		ExtraData: map[string]any{
			"UserID":             userID,
			"AltQuestionContent": altQuestion.Content,
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
	_, _, ok := tools.CheckRequest(w, r, http.MethodGet)
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
	if questionIDStr == "" {
		log.Println("From AddFormAltImageHandler : no alt question id parameter")
		http.Error(w, "Something went wrong !", http.StatusBadRequest)
		return
	}

	dataPage := data.AltImagePageData{
		Routes:         data.DefaultDashboardRoutes,
		AltImageRoutes: data.DefaultAltImageRoutes,
		PageTitle:      "add alt image",
		ExtraData: map[string]any{
			"QuestionID":    questionIDStr,
			"AltQuestionID": altQuestionIDStr,
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

	widthStr := r.FormValue("width")
	widthFloat, err := strconv.ParseFloat(widthStr, 64)
	if err != nil || widthFloat <= 0 {
		log.Printf("From AddAltImageHandler -> strconv.ParseFloat : invalid number : error : %v", err)
		errorMessage := url.QueryEscape("Assurez-vous de bien saisir un nombre entier supérieur à zéro.")
		http.Redirect(w, r, data.ErrorMessageURL+"?errormessage="+errorMessage, http.StatusSeeOther)
		return
	}
	resize := int64(math.Round(widthFloat))

	file, header, err := r.FormFile("image")
	if err != nil {
		log.Printf("From AddAltImageHandler -> r.FormFile: %v", err)
		http.Error(w, "Something went wrong !", http.StatusBadRequest)
		return
	}
	defer file.Close()

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
	ok = tools.ImageCircleCheck(config.ImageSavePath, filename, widthFloat)
	if !ok {
		os.Remove(filepath.Join(config.ImageSavePath, filename))
		errorMessage := url.QueryEscape("L'image contient des cercles incompatibles avec la suites du traitement. Changer la taille de l'image ou prenez une image différente.")
		http.Redirect(w, r, data.ErrorMessageURL+"?errormessage="+errorMessage, http.StatusSeeOther)
		return
	}

	if err := queries.CreateAltImage(r.Context(), db.CreateAltImageParams{
		AltQuestionID:    altQuestionID,
		ImageName:        filename,
		ResizePercentage: resize,
		UserID:           userID,
	}); err != nil {

		os.Remove(filepath.Join(config.ImageSavePath, filename))
		log.Printf("From AddAltImageHandler, CreateAltImage : DB error: %v", err)
		errorMessage := url.QueryEscape("Une alt question peut avoir qu'une seule image.")
		http.Redirect(w, r, data.ErrorMessageURL+"?errormessage="+errorMessage, http.StatusSeeOther)
		return
	}

	altImageURL := data.DefaultAltQuestionRoutes.AltImageURL + "?question_id=" + url.QueryEscape(questionIDStr) +
		"&alt_question_id=" + url.QueryEscape(altQuestionIDStr)
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

	altImage, err := queries.GetAltImageByAltQuestionID(r.Context(), db.GetAltImageByAltQuestionIDParams{
		AltQuestionID: altQuestionID,
		UserID:        userID,
	})
	if err != nil {
		log.Printf("From EditFormAltImageHandler -> GetAltImageByAltQuestionID : DB error : %v", err)
		http.Error(w, "Something went wrong !", http.StatusInternalServerError)
		return
	}

	dataPage := data.AltImagePageData{
		Routes:         data.DefaultDashboardRoutes,
		AltImageRoutes: data.DefaultAltImageRoutes,
		PageTitle:      "edit alt image",
		ExtraData: map[string]any{
			"ImageSize":     altImage.ResizePercentage,
			"QuestionID":    questionIDStr,
			"AltQuestionID": altQuestionIDStr,
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
	})
	if err != nil {

		log.Printf("From  EditImageHandler -> GetImageByQuestionID DB error: %v", err)
		http.Error(w, "Something went wrong !", http.StatusInternalServerError)
		return
	}
	ok = tools.ImageCircleCheck(config.ImageSavePath, image.ImageName, widthFloat)
	if !ok {

		errorMessage := url.QueryEscape("Le redimensionnement de l'image n'est pas compatible avec la détection des réponses. Changer de taille.")
		http.Redirect(w, r, data.ErrorMessageURL+"?errormessage="+errorMessage, http.StatusSeeOther)
		return

	}

	resize := int64(math.Round(widthFloat))

	if err := queries.UpdateSizeAltImage(r.Context(), db.UpdateSizeAltImageParams{
		ResizePercentage: resize,
		AltQuestionID:    altQuestionID,
		UserID:           userID,
	}); err != nil {
		log.Printf("From  EditAltImageHandler : UpdateSizeAltImage DB error: %v", err)
		http.Error(w, "Something went wrong !", http.StatusInternalServerError)
		return
	}

	altImageURL := data.DefaultAltQuestionRoutes.AltImageURL + "?question_id=" + url.QueryEscape(questionIDStr) +
		"&alt_question_id=" + url.QueryEscape(altQuestionIDStr)
	http.Redirect(w, r, altImageURL, http.StatusSeeOther)
}

func DeleteFormAltImageHandler(w http.ResponseWriter, r *http.Request, queries *db.Queries) {
	_, _, ok := tools.CheckRequest(w, r, http.MethodGet)
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

	dataPage := data.AltImagePageData{
		Routes:         data.DefaultDashboardRoutes,
		AltImageRoutes: data.DefaultAltImageRoutes,
		PageTitle:      "delete alt image",
		ExtraData: map[string]any{
			"QuestionID":    questionIDStr,
			"AltQuestionID": altQuestionIDStr,
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

	if err := tools.DeleteAltImageFile(userID,
		altQuestionID, w, r, queries); err != nil {
		log.Printf("From DeleteAltImageHandler -> DeleteAltImageFile : %v", err)
		http.Error(w, "Something went wrong", http.StatusInternalServerError)
		return
	}

	if err := queries.DeleteAltImage(r.Context(), db.DeleteAltImageParams{
		AltQuestionID: altQuestionID,
		UserID:        userID,
	}); err != nil {
		log.Printf("From DeleteAltImageHandler -> DeleteAltImage DB error: %v", err)
		http.Error(w, "DB error", http.StatusInternalServerError)
		return
	}

	altImageURL := data.DefaultAltQuestionRoutes.AltImageURL + "?question_id=" + url.QueryEscape(questionIDStr) +
		"&alt_question_id=" + url.QueryEscape(altQuestionIDStr)
	http.Redirect(w, r, altImageURL, http.StatusSeeOther)
}
