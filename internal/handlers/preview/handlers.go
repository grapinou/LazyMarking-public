package preview

import (
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"

	"github.com/grapinou/LazyMarking/internal/config"
	"github.com/grapinou/LazyMarking/internal/db"
	"github.com/grapinou/LazyMarking/internal/handlers/tools"
	"github.com/grapinou/LazyMarking/internal/templates/data"
)

func PreviewQuestionHandler(w http.ResponseWriter, r *http.Request, queries *db.Queries) {
	userID, username, ok := tools.CheckRequest(w, r, http.MethodGet)
	if !ok {
		log.Println("From PreviewHandler -> tools.CheckRequest return not ok")
		return
	}

	questionIDStr := r.URL.Query().Get("question_id")
	if questionIDStr == "" {
		log.Println("From PreviewHandler : no question id parameter")
		http.Error(w, "Something went wrong !", http.StatusBadRequest)
		return
	}
	questionID, err := strconv.ParseInt(questionIDStr, 10, 64)
	if err != nil {
		log.Printf("From PreviewHandler -> strconv.ParseInt : invalid question ID, error : %v", err)
		http.Error(w, "Something went wrong !", http.StatusBadRequest)
		return
	}

	question, err := tools.GetQuestionAnswer(userID, questionID, queries, r)
	if err != nil {
		log.Println("From PreviewHandler -> tools.GetQuestionAnswer : error")
		http.Error(w, "Something went wrong !", http.StatusInternalServerError)
	}

	questions := []config.Question{question}

	qcm := config.QCM{
		Student:   "John Doe dit la fritte du nord",
		Questions: questions,
	}

	typstFilePath, ok := tools.TypstWriter(username, qcm, config.PreviewQuestion)
	if !ok {
		log.Println("From PreviewHandler -> tools.TypstWriter return not ok")
		http.Error(w, "Something went wrong !", http.StatusInternalServerError)
	}

	_, ok = tools.CompileTypst(typstFilePath)
	if !ok {
		log.Println("From PreviewHandler -> tools.CompileTypst return not ok")
		http.Error(w, "Something went wrong !", http.StatusInternalServerError)
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

	pdfPath := filepath.Join("assets", "tmp", username, username+string(config.PreviewQuestion)+".pdf")

	// Open file
	f, err := os.Open(pdfPath)
	if err != nil {
		log.Printf("From ServePreviewPDFHandler -> Open, error : %v", err)
		w.WriteHeader(500)
		return
	}
	defer f.Close()

	// Set header
	w.Header().Set("Content-type", "application/pdf")

	// Stream to response
	if _, err := io.Copy(w, f); err != nil {
		log.Printf("From ServePreviewPDFHandler -> Copy, error : %v", err)
		w.WriteHeader(500)
	}
}
