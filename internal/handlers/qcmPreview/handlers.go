package qcmpreview

import (
	"log"
	"net/http"
	"net/url"
	"strconv"

	"github.com/google/uuid"
	"github.com/grapinou/LazyMarking/internal/config"
	"github.com/grapinou/LazyMarking/internal/db"
	"github.com/grapinou/LazyMarking/internal/handlers/tools"
	"github.com/grapinou/LazyMarking/internal/templates/data"
)

func PreviewQCMHandler(w http.ResponseWriter, r *http.Request, queries *db.Queries) {
	userID, username, ok := tools.CheckRequest(w, r, http.MethodGet)
	if !ok {
		log.Println("From PreviewQCMHandler -> tools.CheckRequest return not ok")
		return
	}

	qcmIDStr := r.URL.Query().Get("qcm_id")
	if qcmIDStr == "" {
		log.Println("From PreviewQCMHandler : no question id parameter")
		http.Error(w, "Something went wrong !", http.StatusBadRequest)
		return
	}
	qcmID, err := strconv.ParseInt(qcmIDStr, 10, 64)
	if err != nil {
		log.Printf("From PreviewQCMHandler -> strconv.ParseInt : invalid qcm ID, error : %v", err)
		http.Error(w, "Something went wrong !", http.StatusBadRequest)
		return
	}

	questions, err := tools.GetQCMQuestionsAnswers(userID, qcmID, r, queries)
	if err == tools.ErrQuestionWithNoAnswer {
		log.Printf("From PreviewQCMHandler -> GetQCMQuestionsAnswers -> BuildQuestion : error : %v", err)
		errorMessage := url.QueryEscape("Il y a une question qui n'a pas de réponse. Il n'est pas possible de construire le qcm")
		http.Redirect(w, r, data.ErrorMessageURL+"?errormessage="+errorMessage, http.StatusSeeOther)
		return
	}
	if err != nil {
		log.Printf("From PreviewQCMHandler -> GetQCMQuestionsAnswers (-> BuildQuestion) : error : %v", err)
		http.Error(w, "Something went wrong !", http.StatusInternalServerError)
		return
	}

	student := config.StudentQCM{
		FirstName: "John Doe",
		LastName:  "dit la fritte du nord",
		ClassCodes: config.ClassCode{
			Name: "666",
		},
	}

	qcm := config.QCM{
		Student:   student,
		Questions: questions,
	}

	operation := "preview-" + uuid.NewString()
	tempDir, ok := tools.CreateOperationTempDir(username, operation)
	if !ok {
		http.Error(w, "Something went wrong !", http.StatusInternalServerError)
		return
	}
	typstFilePath, ok := tools.TypstWriter(tempDir, username, qcm, config.PreviewQCM)
	if !ok {
		log.Println("From PreviewQuestionHandler -> tools.TypstWriter return not ok")
		http.Error(w, "Something went wrong !", http.StatusInternalServerError)
		return
	}

	_, ok = tools.CompileTypst(typstFilePath)
	if !ok {
		log.Println("From PreviewQuestionHandler -> tools.CompileTypst return not ok")
		http.Error(w, "Something went wrong !", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, data.DefaultPreviewQCMRoutes.PreviewQCM+"?operation="+url.QueryEscape(operation), http.StatusSeeOther)
}

func PreviewQCMLandscapeHandler(w http.ResponseWriter, r *http.Request, queries *db.Queries) {
	userID, username, ok := tools.CheckRequest(w, r, http.MethodGet)
	if !ok {
		log.Println("From PreviewQCMLandscapeHandler -> tools.CheckRequest return not ok")
		return
	}

	qcmIDStr := r.URL.Query().Get("qcm_id")
	if qcmIDStr == "" {
		log.Println("From PreviewQCMLandscapeHandler : no question id parameter")
		http.Error(w, "Something went wrong !", http.StatusBadRequest)
		return
	}
	qcmID, err := strconv.ParseInt(qcmIDStr, 10, 64)
	if err != nil {
		log.Printf("From PreviewQCMLandscapeHandler -> strconv.ParseInt : invalid qcm ID, error : %v", err)
		http.Error(w, "Something went wrong !", http.StatusBadRequest)
		return
	}

	questions, err := tools.GetQCMQuestionsAnswers(userID, qcmID, r, queries)
	if err == tools.ErrQuestionWithNoAnswer {
		log.Printf("From PreviewQCMLandscapeHandler -> GetQCMQuestionsAnswers -> BuildQuestion : error : %v", err)
		errorMessage := url.QueryEscape("Il y a une question qui n'a pas de réponse. Il n'est pas possible de construire le qcm")
		http.Redirect(w, r, data.ErrorMessageURL+"?errormessage="+errorMessage, http.StatusSeeOther)
		return
	}
	if err != nil {
		log.Printf("From PreviewQCMLandscapeHandler -> GetQCMQuestionsAnswers (-> BuildQuestion) : error : %v", err)
		http.Error(w, "Something went wrong !", http.StatusInternalServerError)
		return
	}

	student := config.StudentQCM{
		FirstName: "John Doe",
		LastName:  "dit la fritte du nord",
		ClassCodes: config.ClassCode{
			Name: "666",
		},
	}

	qcm := config.QCM{
		Student:   student,
		Questions: questions,
	}

	operation := "preview-" + uuid.NewString()
	tempDir, ok := tools.CreateOperationTempDir(username, operation)
	if !ok {
		http.Error(w, "Something went wrong !", http.StatusInternalServerError)
		return
	}
	typstFilePath, ok := tools.TypstWriterLandscape(tempDir, username, qcm)
	if !ok {
		log.Println("From PreviewQuestionHandler -> tools.TypstWriter return not ok")
		http.Error(w, "Something went wrong !", http.StatusInternalServerError)
		return
	}

	_, ok = tools.CompileTypst(typstFilePath)
	if !ok {
		log.Println("From PreviewQuestionHandler -> tools.CompileTypst return not ok")
		http.Error(w, "Something went wrong !", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, data.DefaultPreviewQCMRoutes.PreviewLandscapeQCM+"?operation="+url.QueryEscape(operation), http.StatusSeeOther)
}

func ServePreviewQCMPDFHandler(w http.ResponseWriter, r *http.Request, queries *db.Queries) {
	_, username, ok := tools.CheckRequest(w, r, http.MethodGet)
	if !ok {
		log.Println("From ServePreviewQCMPDFHandler -> tools.CheckRequest return not ok")
		return
	}

	if username == "" {
		log.Println("From ServePreviewQCMPDFHandler, no username")
		http.Error(w, "Something went wrong !", http.StatusBadRequest)
		return
	}

	operation := r.URL.Query().Get("operation")
	if operation == "" {
		http.Error(w, "Missing operation parameter", http.StatusBadRequest)
		return
	}
	tools.ServePdf(username, operation, config.PreviewQCM, w, r)
}

func ServePreviewQCMLandscapePDFHandler(w http.ResponseWriter, r *http.Request, queries *db.Queries) {
	_, username, ok := tools.CheckRequest(w, r, http.MethodGet)
	if !ok {
		log.Println("From ServePreviewQCMPDFHandler -> tools.CheckRequest return not ok")
		return
	}

	if username == "" {
		log.Println("From ServePreviewQCMPDFHandler, no username")
		http.Error(w, "Something went wrong !", http.StatusBadRequest)
		return
	}

	operation := r.URL.Query().Get("operation")
	if operation == "" {
		http.Error(w, "Missing operation parameter", http.StatusBadRequest)
		return
	}
	tools.ServePdf(username, operation, config.PreviewLandscapeQCM, w, r)
}
