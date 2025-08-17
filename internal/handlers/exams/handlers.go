package exams

import (
	"log"
	"net/http"
	"net/url"
	"strconv"

	"github.com/grapinou/LazyMarking/internal/db"
	"github.com/grapinou/LazyMarking/internal/handlers/tools"
	"github.com/grapinou/LazyMarking/internal/templates/data"
)

func TableExamsHandler(w http.ResponseWriter, r *http.Request, queries *db.Queries) {
	userID, _, ok := tools.CheckRequest(w, r, http.MethodGet)
	if !ok {
		log.Println("From TableExamsHandler -> tools.CheckRequest return not ok")
		return
	}

	examsDB, err := queries.GetExamsAllInfos(r.Context(), userID)
	if err != nil {
		log.Printf("From TableExamsHandler -> GetExamsAllInfos DB error: %v", err)
		http.Error(w, "DB Error", http.StatusInternalServerError)
		return
	}

	noExam := true
	if len(examsDB) > 0 {
		noExam = false
	}

	var actionsURLParameters []data.ExamActionURLs
	if !noExam {
		for _, exam := range examsDB {
			params := "?exam_id=" + url.QueryEscape(strconv.FormatInt(exam.ID, 10))
			editURL := data.DefaultQuestionRoutes.EditURL + params
			deleteURL := data.DefaultQuestionRoutes.DeleteURL + params

			actionsURLParameters = append(actionsURLParameters, data.ExamActionURLs{
				EditURL:   editURL,
				DeleteURL: deleteURL,
			})
		}
	}

	dataPage := data.ExamPageData{
		Routes:     data.DefaultDashboardRoutes,
		ExamRoutes: data.DefaultExamRoutes,
		PageTitle:  "exams",
		ExtraData: map[string]any{
			"UserID": userID,
			"NoExam": noExam,
			"Exams":  examsDB,
			"Action": actionsURLParameters,
		},
	}

	RenderTableExamsPage(w, dataPage)
}
