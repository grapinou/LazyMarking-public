package qcmpreview

import (
	"log"
	"net/http"
	"strconv"

	"github.com/grapinou/LazyMarking/internal/db"
	"github.com/grapinou/LazyMarking/internal/handlers/tools"
)

func PreviewQCMHandler(w http.ResponseWriter, r *http.Request, queries *db.Queries) {
	userID, _, ok := tools.CheckRequest(w, r, http.MethodGet)
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

	_, err = tools.GetQCMQuestionsAnswers(userID, qcmID, r, queries)

	/*
	   question, err := tools.GetQuestionAnswer(userID, qcmID, queries, r)

	   	if err != nil {
	   		log.Println("From PreviewQCMHandler -> tools.GetQuestionAnswer : error")
	   		http.Error(w, "Something went wrong !", http.StatusInternalServerError)
	   		return
	   	}

	   questions := []config.Question{question}

	   	qcm := config.QCM{
	   		Student:   "John Doe dit la fritte du nord",
	   		Questions: questions,
	   	}

	   typstFilePath, ok := tools.TypstWriter(username, qcm, config.PreviewQuestion)

	   	if !ok {
	   		log.Println("From PreviewQCMHandler -> tools.TypstWriter return not ok")
	   		http.Error(w, "Something went wrong !", http.StatusInternalServerError)
	   		return
	   	}

	   _, ok = tools.CompileTypst(typstFilePath)

	   	if !ok {
	   		log.Println("From PreviewQCMHandler -> tools.CompileTypst return not ok")
	   		http.Error(w, "Something went wrong !", http.StatusInternalServerError)
	   		return
	   	}

	   http.Redirect(w, r, data.DefaultPreviewQuestionRoutes.PreviewQuestion, http.StatusSeeOther)
	*/
}

/*

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
	tools.ServePdf(username, w)
}
*/
