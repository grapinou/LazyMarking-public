package preview

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

func PreviewQuestionHandler(w http.ResponseWriter, r *http.Request, queries *db.Queries) {
	userID, username, ok := tools.CheckRequest(w, r, http.MethodGet)
	if !ok {
		log.Println("From PreviewQuestionHandler -> tools.CheckRequest return not ok")
		return
	}

	questionIDStr := r.URL.Query().Get("question_id")
	if questionIDStr == "" {
		log.Println("From PreviewQuestionHandler : no question id parameter")
		http.Error(w, "Something went wrong !", http.StatusBadRequest)
		return
	}
	questionID, err := strconv.ParseInt(questionIDStr, 10, 64)
	if err != nil {
		log.Printf("From PreviewQuestionHandler -> strconv.ParseInt : invalid question ID, error : %v", err)
		http.Error(w, "Something went wrong !", http.StatusBadRequest)
		return
	}
	if _, err := queries.GetQuestionByID(r.Context(), db.GetQuestionByIDParams{ID: questionID, UserID: userID}); err != nil {
		tools.HandleOwnedLookupError(w, err, "PreviewQuestionHandler GetQuestionByID")
		return
	}

	question, err := tools.GetQuestionAnswer(userID, questionID, queries, r)
	if err != nil {
		log.Println("From PreviewQuestionHandler -> tools.GetQuestionAnswer : error")
		http.Error(w, "Something went wrong !", http.StatusInternalServerError)
		return
	}

	questions := []config.Question{question}

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
		log.Printf("From PreviewQuestionHandler -> purge stale preview workspaces: %v", err)
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
			log.Printf("From PreviewQuestionHandler -> cleanup failed preview workspace: %v", err)
		}
	}()
	typstFilePath, ok := tools.TypstWriter(tempDir, username, qcm, config.PreviewQuestion)
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
	http.Redirect(w, r, data.DefaultPreviewQuestionRoutes.PreviewQuestion+"?operation="+url.QueryEscape(operation), http.StatusSeeOther)
}

func ServePreviewPDFHandler(w http.ResponseWriter, r *http.Request, queries *db.Queries) {
	_, username, ok := tools.CheckRequest(w, r, http.MethodGet)
	if !ok {
		log.Println("From ServePreviewPDFHandler -> tools.CheckRequest return not ok")
		return
	}

	if username == "" {
		log.Println("From ServePreviewPDFHandler, no username")
		http.Error(w, "Something went wrong !", http.StatusBadRequest)
		return
	}

	// faire une fonction dans tool.
	operation := r.URL.Query().Get("operation")
	if operation == "" {
		http.Error(w, "Missing operation parameter", http.StatusBadRequest)
		return
	}
	tools.ServePdf(username, operation, config.PreviewQuestion, w, r)
}
