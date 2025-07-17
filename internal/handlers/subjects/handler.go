package subjects

import (
	"log"
	"net/http"
	"net/url"
	"strconv"

	"github.com/grapinou/LazyMarking/internal/db"
	"github.com/grapinou/LazyMarking/internal/handlers/login"
	"github.com/grapinou/LazyMarking/internal/templates/data"
)

func SubjectsHandler(w http.ResponseWriter, r *http.Request, queries *db.Queries) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userID, _, ok := login.FromContext(r)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	subjectsDB, err := queries.GetAllSubjects(r.Context(), userID)
	if err != nil {
		http.Error(w, "Can't get all subjects", http.StatusInternalServerError)
	}

	log.Println("subjectsDB", subjectsDB)
	noSubject := true
	if len(subjectsDB) > 0 {
		noSubject = false
	}

	addURL := data.DefaultSubjectRoutes.AddURL + "?user_id=" + url.QueryEscape(strconv.Itoa(int(userID)))
	var actionsURLParameters []data.SubjectActionURLs
	if !noSubject {
		for _, subject := range subjectsDB {
			editURL := data.DefaultSubjectRoutes.EditURL + "?user_id=" + url.QueryEscape(strconv.Itoa(int(userID))) + "&subject_id=" + url.QueryEscape(subject.Name)
			deleteURL := data.DefaultSubjectRoutes.DeleteURL + "?user_id=" + url.QueryEscape(strconv.Itoa(int(userID)))

			actionsURLParameters = append(actionsURLParameters, data.SubjectActionURLs{
				EditURL:   editURL,
				DeleteURL: deleteURL,
			})
		}
	}

	data := data.SubjectPageData{
		Routes:    data.DefaultDashboardRoutes,
		PageTitle: "subjects",
		ExtraData: map[string]any{
			"AddURL":    addURL,
			"NoSubject": noSubject,
			"Action":    actionsURLParameters,
			"Subjects":  subjectsDB,
		},
	}

	RenderSubjectPage(w, data)
}
