package students

import (
	"log"
	"net/http"
	"net/url"
	"strconv"

	"github.com/grapinou/LazyMarking/internal/db"
	"github.com/grapinou/LazyMarking/internal/handlers/tools"
	"github.com/grapinou/LazyMarking/internal/templates/data"
)

func TableStudentsHandler(w http.ResponseWriter, r *http.Request, queries *db.Queries) {
	userID, _, ok := tools.CheckRequest(w, r, http.MethodGet)
	if !ok {
		log.Println("From TableStudentsHandler -> tools.CheckRequest return not ok")
		return
	}

	studentsDB, err := queries.GetAllStudents(r.Context(), userID)
	if err != nil {
		log.Printf("From TableStudentsHandler -> GetAllStudents DB error: %v", err)
		http.Error(w, "DB error", http.StatusInternalServerError)
		return
	}

	noStudent := true
	if len(studentsDB) > 0 {
		noStudent = false
	}

	var actionsURLParameters []data.StudentActionURLs
	if !noStudent {
		for _, student := range studentsDB {
			params := "?student_id=" + url.QueryEscape(strconv.FormatInt(student.ID, 10))
			editURL := data.DefaultStudentRoutes.EditURL + params
			deleteURL := data.DefaultStudentRoutes.DeleteURL + params

			actionsURLParameters = append(actionsURLParameters, data.StudentActionURLs{
				EditURL:   editURL,
				DeleteURL: deleteURL,
			})
		}
	}

	dataPage := data.StudentPageData{
		Routes:        data.DefaultDashboardRoutes,
		StudentRoutes: data.DefaultStudentRoutes,
		PageTitle:     "students",
		ExtraData: map[string]any{
			"NoStudent": noStudent,
			"Action":    actionsURLParameters,
			"Students":  studentsDB,
		},
	}

	RenderTableStudentPage(w, dataPage)
}
