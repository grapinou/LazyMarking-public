package questions

import (
	"log"
	"net/http"

	"github.com/grapinou/LazyMarking/internal/db"
	"github.com/grapinou/LazyMarking/internal/handlers/login"
	"github.com/grapinou/LazyMarking/internal/templates/data"
)

func QuestionHandler(w http.ResponseWriter, r *http.Request, queries *db.Queries) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userID, username, ok := login.FromContext(r)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	questionsDB, err := queries.GetAllQuestions(r.Context(), userID)
	if err != nil {
		http.Error(w, "Can't get all questions", http.StatusInternalServerError)
	}

	log.Println("questionsDB", questionsDB)
	noQuestion := true
	if len(questionsDB) > 0 {
		noQuestion = false
	}

	data := data.DashboardPageData{
		Routes:    data.DefaultDashboardRoutes,
		PageTitle: "questions",
		ExtraData: map[string]any{
			"UserID":     userID,
			"Username":   username,
			"NoQuestion": noQuestion,
			"Questions":  questionsDB,
		},
	}

	RenderQuestionPage(w, data)
}
