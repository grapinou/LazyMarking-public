package qcmpreview

import (
	"log"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/grapinou/LazyMarking/internal/config"
	"github.com/grapinou/LazyMarking/internal/db"
	"github.com/grapinou/LazyMarking/internal/handlers/tools"
	"github.com/grapinou/LazyMarking/internal/templates/data"
)

var getPreviewQCMQuestions = tools.GetQCMQuestionsAnswersInReferenceOrder

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

	questions, ok := loadPreviewQCMQuestions(w, r, queries, userID, qcmID)
	if !ok {
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

	if err := tools.PurgeExpiredUserEphemeralWorkspaces(username, time.Now()); err != nil {
		log.Printf("From PreviewQCMHandler -> purge stale preview workspaces: %v", err)
	}
	operation := "preview-" + uuid.NewString()
	tempDir, ok := tools.CreateOperationTempDir(username, operation)
	if !ok {
		http.Error(w, "Something went wrong !", http.StatusInternalServerError)
		return
	}
	keepWorkspace := false
	defer func() {
		if keepWorkspace {
			return
		}
		if err := tools.RemoveOperationTempDir(username, operation); err != nil {
			log.Printf("From PreviewQCMHandler -> cleanup failed preview workspace: %v", err)
		}
	}()
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

	keepWorkspace = true
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

	questions, ok := loadPreviewQCMQuestions(w, r, queries, userID, qcmID)
	if !ok {
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

	if err := tools.PurgeExpiredUserEphemeralWorkspaces(username, time.Now()); err != nil {
		log.Printf("From PreviewQCMLandscapeHandler -> purge stale preview workspaces: %v", err)
	}
	operation := "preview-" + uuid.NewString()
	tempDir, ok := tools.CreateOperationTempDir(username, operation)
	if !ok {
		http.Error(w, "Something went wrong !", http.StatusInternalServerError)
		return
	}
	keepWorkspace := false
	defer func() {
		if keepWorkspace {
			return
		}
		if err := tools.RemoveOperationTempDir(username, operation); err != nil {
			log.Printf("From PreviewQCMLandscapeHandler -> cleanup failed preview workspace: %v", err)
		}
	}()
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

	keepWorkspace = true
	http.Redirect(w, r, data.DefaultPreviewQCMRoutes.PreviewLandscapeQCM+"?operation="+url.QueryEscape(operation), http.StatusSeeOther)
}

func loadPreviewQCMQuestions(w http.ResponseWriter, r *http.Request, queries *db.Queries, userID, qcmID int64) ([]config.Question, bool) {
	if _, err := queries.GetQCMNameByID(r.Context(), db.GetQCMNameByIDParams{ID: qcmID, UserID: userID}); err != nil {
		tools.HandleOwnedLookupError(w, err, "loadPreviewQCMQuestions GetQCMNameByID")
		return nil, false
	}

	questions, err := getPreviewQCMQuestions(userID, qcmID, r, queries)
	if err == tools.ErrQuestionWithNoAnswer {
		log.Printf("From loadPreviewQCMQuestions -> BuildQuestion: %v", err)
		redirectPreviewError(w, r, "Il y a une question qui n'a pas de réponse. Il n'est pas possible de construire le qcm")
		return nil, false
	}
	if err != nil {
		log.Printf("From loadPreviewQCMQuestions -> construct questions: %v", err)
		http.Error(w, "Something went wrong !", http.StatusInternalServerError)
		return nil, false
	}
	if len(questions) == 0 {
		redirectPreviewError(w, r, "Ce QCM ne contient aucune question. Ajoutez au moins une question avant de générer un aperçu.")
		return nil, false
	}
	return questions, true
}

func redirectPreviewError(w http.ResponseWriter, r *http.Request, message string) {
	http.Redirect(w, r, data.ErrorMessageURL+"?errormessage="+url.QueryEscape(message), http.StatusSeeOther)
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
