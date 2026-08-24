package preview

import (
	"log"
	"net/http"
	"strconv"

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

	tempDir, ok := tools.CreateUserTempDir(username)
	if !ok {
		http.Error(w, "Something went wrong !", http.StatusInternalServerError)
		return
	}
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

	http.Redirect(w, r, data.DefaultPreviewQuestionRoutes.PreviewQuestion, http.StatusSeeOther)
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
	tools.ServePdf(username, config.PreviewQuestion, w)
}
