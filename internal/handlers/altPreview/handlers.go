package altpreview

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

func AltPreviewAltQuestionHandler(w http.ResponseWriter, r *http.Request, queries *db.Queries) {
	userID, username, ok := tools.CheckRequest(w, r, http.MethodGet)
	if !ok {
		log.Println("From AltPreviewAltQuestionHandler -> tools.CheckRequest return not ok")
		return
	}

	altquestionIDStr := r.URL.Query().Get("alt_question_id")
	if altquestionIDStr == "" {
		log.Println("From  AltPreviewAltQuestionHandler : no alt question id parameter")
		http.Error(w, "Something went wrong !", http.StatusBadRequest)
		return
	}
	altQuestionID, err := strconv.ParseInt(altquestionIDStr, 10, 64)
	if err != nil {
		log.Printf("From  AltPreviewAltQuestionHandler -> strconv.ParseInt : invalid question ID, error : %v", err)
		http.Error(w, "Something went wrong !", http.StatusBadRequest)
		return
	}

	altquestion, err := tools.GetAltQuestionAltAnswer(userID, altQuestionID, queries, r)
	if err != nil {
		log.Println("From AltPreviewAltQuestionHandler -> tools.GetQuestionAnswer : error")
		http.Error(w, "Something went wrong !", http.StatusInternalServerError)
		return
	}

	questions := []config.Question{altquestion}

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
	typstFilePath, ok := tools.TypstWriter(tempDir, username, qcm, config.PreviewQuestion)
	if !ok {
		log.Println("From AltPreviewAltQuestionHandler -> tools.TypstWriter return not ok")
		http.Error(w, "Something went wrong !", http.StatusInternalServerError)
		return
	}

	_, ok = tools.CompileTypst(typstFilePath)
	if !ok {
		log.Println("From AltPreviewAltQuestionHandler -> tools.CompileTypst return not ok")
		http.Error(w, "Something went wrong !", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, data.DefaultAltPreviewAltQuestionRoutes.AltPreviewAltQuestion+"?operation="+url.QueryEscape(operation), http.StatusSeeOther)
}

func AltServePreviewPDFHandler(w http.ResponseWriter, r *http.Request, queries *db.Queries) {
	_, username, ok := tools.CheckRequest(w, r, http.MethodGet)
	if !ok {
		log.Println("From AltServePreviewPDFHandler -> tools.CheckRequest return not ok")
		return
	}

	if username == "" {
		log.Println("From AltServePreviewPDFHandler, no username")
		http.Error(w, "Something went wrong !", http.StatusBadRequest)
		return
	}

	operation := r.URL.Query().Get("operation")
	if operation == "" {
		http.Error(w, "Missing operation parameter", http.StatusBadRequest)
		return
	}
	tools.ServePdf(username, operation, config.PreviewQuestion, w, r)
}
